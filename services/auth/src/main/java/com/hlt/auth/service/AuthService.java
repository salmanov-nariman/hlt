package com.hlt.auth.service;

import com.hlt.auth.grpc.client.EmailClient;
import com.hlt.auth.grpc.client.UserClient;
import com.hlt.auth.jwt.JwtService;
import com.hlt.auth.model.data.EmailVerificationResult;
import com.hlt.auth.model.dto.request.*;
import com.hlt.auth.model.dto.response.RegisterResponse;
import com.hlt.auth.model.dto.response.TokenResponse;
import com.hlt.auth.model.entity.AccountStatus;
import com.hlt.auth.model.entity.Credentials;
import com.hlt.auth.model.entity.EmailVerification;
import com.hlt.auth.model.entity.RefreshToken;
import com.hlt.auth.repository.CredentialsRepository;
import com.hlt.auth.repository.EmailVerificationRepository;
import com.hlt.auth.repository.RefreshTokenRepository;
import io.grpc.StatusRuntimeException;
import org.springframework.http.HttpStatus;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.server.ResponseStatusException;
import reactor.core.publisher.Mono;
import reactor.util.function.Tuple2;

import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.UUID;

@Service
public class AuthService {

    private static final String INVALID_CREDENTIALS = "Неверный email или пароль";
    private static final String EMAIL_VERIFICATION_REQUIRED = "Подтвердите почту";
    private static final String INVALID_REFRESH_TOKEN = "Невалидный токен";
    private static final String SESSION_EXPIRED = "Сессия истекла, авторизуйтесь заново";
    private static final String INVALID_AUTHORIZATION_HEADER = "Отсутствует или неверный формат заголовка";
    private static final String INVALID_OR_EXPIRED_ACCESS_TOKEN = "Токен недействителен или просрочен";
    private static final String EMAIL_AND_USERNAME_TAKEN = "Email и Username уже заняты";
    private static final String EMAIL_TAKEN = "Email уже занят";
    private static final String USERNAME_TAKEN = "Username уже занят";
    private static final String PROFILE_SERVICE_UNAVAILABLE = "Сервис профилей временно недоступен";
    private static final String INVALID_PROFILE_DATA = "Некорректные данные профиля";
    private static final String PROFILE_CREATION_FAILED = "Не удалось создать профиль";
    private static final String VERIFICATION_CODE_DELIVERY_FAILED = "Не удалось отправить код. Попробуйте ещё раз";
    private static final String VERIFICATION_EXPIRED = "Срок подтверждения истёк. Зарегистрируйтесь заново";

    private final CredentialsRepository credentialsRepository;
    private final EmailVerificationRepository emailVerificationRepository;
    private final RefreshTokenRepository refreshTokenRepository;
    private final RegistrationPersistenceService registrationPersistenceService;
    private final VerificationCodeService verificationCodeService;
    private final EmailClient emailClient;
    private final UserClient userClient;
    private final PasswordEncoder passwordEncoder;
    private final JwtService jwtService;

    public AuthService(
            CredentialsRepository credentialsRepository,
            EmailVerificationRepository emailVerificationRepository,
            RefreshTokenRepository refreshTokenRepository,
            RegistrationPersistenceService registrationPersistenceService,
            VerificationCodeService verificationCodeService,
            EmailClient emailClient,
            UserClient userClient,
            PasswordEncoder passwordEncoder,
            JwtService jwtService
    ) {
        this.credentialsRepository = credentialsRepository;
        this.emailVerificationRepository = emailVerificationRepository;
        this.refreshTokenRepository = refreshTokenRepository;
        this.registrationPersistenceService = registrationPersistenceService;
        this.verificationCodeService = verificationCodeService;
        this.emailClient = emailClient;
        this.userClient = userClient;
        this.passwordEncoder = passwordEncoder;
        this.jwtService = jwtService;
    }

    public Mono<RegisterResponse> register(RegisterRequest request) {

        return Mono
                .zip(
                    credentialsRepository.existsByEmail(request.email()),
                    credentialsRepository.existsByUsername(request.username())
                )
                .flatMap(tuple -> {
                    if (tuple.getT1() || tuple.getT2()) {
                        return Mono.error(new IllegalArgumentException(getErrorMessage(tuple)));
                    }

                    String verificationCode = verificationCodeService.generateCode();
                    String codeHash = verificationCodeService.hashCode(verificationCode);

                    Credentials credentials = new Credentials(
                            null, request.username(),
                            request.email(),
                            passwordEncoder.encode(request.password()),
                            AccountStatus.PENDING_VERIFICATION,
                            Instant.now()
                    );

                    EmailVerification emailVerification = new EmailVerification(
                            null,
                            null,
                            codeHash,
                            Instant.now().plus(verificationCodeService.getCodeExpiration()),
                            0,
                            null
                    );

                    return registrationPersistenceService
                            .createRegistrationData(credentials, emailVerification)
                            .flatMap(data ->
                                    emailClient.sendVerificationEmail(data.credentials().getEmail(), verificationCode)
                                            .thenReturn(true)
                                            .onErrorReturn(false)
                                            .flatMap(sent -> {
                                                if (!sent) {
                                                    return Mono.just(data.credentials());
                                                }

                                                return registrationPersistenceService.markSentEmail(data.emailVerification())
                                                        .thenReturn(data.credentials());
                                            })
                            );
                })
                .map(saved -> new RegisterResponse(saved.getUsername(), saved.getEmail()));
    }

    public Mono<TokenResponse> login(LoginRequest request) {

        return credentialsRepository.findByEmail(request.email())
                .switchIfEmpty(Mono.error(new IllegalArgumentException(INVALID_CREDENTIALS)))
                .flatMap(credentials -> {
                    if (!passwordEncoder.matches(request.password(), credentials.getPassword())) {
                        return Mono.error(new IllegalArgumentException(INVALID_CREDENTIALS));
                    }

                    if (credentials.getStatus() != AccountStatus.ACTIVE) {
                        return Mono.error(new ResponseStatusException(HttpStatus.FORBIDDEN, EMAIL_VERIFICATION_REQUIRED));
                    }

                    String accessToken = jwtService.generateAccessToken(credentials.getId());
                    String refreshToken = jwtService.generateRefreshToken();

                    RefreshToken refreshTokenEntity = new RefreshToken(
                            null,
                            credentials.getId(),
                            refreshToken,
                            Instant.now().plus(30, ChronoUnit.DAYS),
                            false
                    );

                    return refreshTokenRepository.save(refreshTokenEntity)
                            .map(savedToken -> new TokenResponse(savedToken.getToken(), accessToken));
                });
    }

    public Mono<TokenResponse> refresh(RefreshTokenRequest request) {

        return refreshTokenRepository.findByToken(request.refreshToken())
                .switchIfEmpty(Mono.error(new IllegalArgumentException(INVALID_REFRESH_TOKEN)))
                .flatMap(existingToken -> {
                    if (existingToken.isRevoked() || existingToken.getExpiresAt().isBefore(Instant.now())) {
                        return Mono.error(new IllegalArgumentException(SESSION_EXPIRED));
                    }

                    String newAccessToken = jwtService.generateAccessToken(existingToken.getAccountId());
                    String newRefreshToken = jwtService.generateRefreshToken();

                    existingToken.setToken(newRefreshToken);
                    existingToken.setExpiresAt(Instant.now().plus(30, ChronoUnit.DAYS));

                    return refreshTokenRepository.save(existingToken)
                            .map(savedToken -> new TokenResponse(savedToken.getToken(), newAccessToken));
                });
    }

    public Mono<UUID> validate(String authHeader) {

        return Mono.justOrEmpty(authHeader)
                .filter(header -> header.startsWith("Bearer "))
                .switchIfEmpty(Mono.error(new ResponseStatusException(HttpStatus.UNAUTHORIZED, INVALID_AUTHORIZATION_HEADER)))
                .map(header -> header.substring(7))
                .flatMap(jwtService::validateAndExtractUserId)
                .onErrorMap(e -> new ResponseStatusException(HttpStatus.UNAUTHORIZED, INVALID_OR_EXPIRED_ACCESS_TOKEN));
    }

    private String getErrorMessage(Tuple2<Boolean, Boolean> tuple) {
        if (tuple.getT1() && tuple.getT2()) return EMAIL_AND_USERNAME_TAKEN;
        if (tuple.getT1()) return EMAIL_TAKEN;
        return USERNAME_TAKEN;
    }

    @Transactional
    public Mono<EmailVerificationResult> emailVerify(EmailVerifyRequest request) {
        return credentialsRepository.findByEmailForUpdate(request.email())
                .flatMap(credentials -> {
                    if (credentials.getStatus() == AccountStatus.ACTIVE) {
                        return Mono.just(EmailVerificationResult.ALREADY_VERIFIED);
                    }

                    if (registrationPersistenceService.isRegistrationExpired(credentials.getCreatedAt(), Instant.now())) {
                        return Mono.just(EmailVerificationResult.EXPIRED);
                    }

                    return emailVerificationRepository
                            .findByAccountId(credentials.getId())
                            .flatMap(verification -> {
                                if (!Instant.now().isBefore(verification.getExpiresAt())) {
                                    return Mono.just(EmailVerificationResult.EXPIRED);
                                }

                                if (verification.getFailedAttempts() >= verificationCodeService.getMaxAttempts()) {
                                    return Mono.just(EmailVerificationResult.ATTEMPTS_EXHAUSTED);
                                }

                                if (!verificationCodeService.matches(request.verificationCode(), verification.getCodeHash())) {
                                    int attempts = verification.getFailedAttempts() + 1;

                                    verification.setFailedAttempts(attempts);

                                    EmailVerificationResult result =
                                            attempts >= verificationCodeService.getMaxAttempts()
                                                    ? EmailVerificationResult.ATTEMPTS_EXHAUSTED
                                                    : EmailVerificationResult.INVALID_CODE;

                                    return emailVerificationRepository.save(verification)
                                            .thenReturn(result);
                                }

                                return userClient.createProfile(
                                                credentials.getId(), credentials.getUsername(), credentials.getEmail()
                                        )
                                        .onErrorMap(StatusRuntimeException.class, AuthService::profileCreationError)
                                        .then(Mono.defer(() -> {
                                            credentials.setStatus(AccountStatus.ACTIVE);
                                            return credentialsRepository.save(credentials)
                                                    .then(emailVerificationRepository.deleteByAccountId(credentials.getId()))
                                                    .thenReturn(EmailVerificationResult.VERIFIED);
                                        }));
                            })
                            .switchIfEmpty(Mono.just(EmailVerificationResult.NOT_FOUND));
                })
                .switchIfEmpty(Mono.just(EmailVerificationResult.NOT_FOUND));
    }

    private static ResponseStatusException profileCreationError(StatusRuntimeException error) {
        return switch (error.getStatus().getCode()) {
            case UNAVAILABLE, DEADLINE_EXCEEDED -> new ResponseStatusException(
                    HttpStatus.SERVICE_UNAVAILABLE, PROFILE_SERVICE_UNAVAILABLE, error);
            case INVALID_ARGUMENT -> new ResponseStatusException(
                    HttpStatus.BAD_REQUEST, INVALID_PROFILE_DATA, error);
            default -> new ResponseStatusException(
                    HttpStatus.BAD_GATEWAY, PROFILE_CREATION_FAILED, error);
        };
    }

    public Mono<Void> emailResend(EmailResendRequest request) {
        return Mono.defer(() -> {
            String code = verificationCodeService.generateCode();
            String codeHash = verificationCodeService.hashCode(code);

            return registrationPersistenceService
                    .prepareEmailResend(request.email(), codeHash)
                    .flux()
                    .collectList()
                    .flatMap(results -> {
                        var prepared = results.getFirst();

                        return prepared.map(emailVerification -> emailClient
                                .sendVerificationEmail(request.email(), code)
                                .onErrorMap(error -> new ResponseStatusException(HttpStatus.SERVICE_UNAVAILABLE, VERIFICATION_CODE_DELIVERY_FAILED, error))
                                .then(Mono.defer(() -> registrationPersistenceService
                                        .markSentEmail(emailVerification)))
                                .then()).orElseGet(() -> Mono.error(new ResponseStatusException(HttpStatus.GONE, VERIFICATION_EXPIRED)));

                    });
        });
    }

}
