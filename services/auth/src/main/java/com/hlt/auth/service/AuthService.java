package com.hlt.auth.service;

import com.hlt.auth.grpc.client.EmailClient;
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
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
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

    private static final Logger log = LoggerFactory.getLogger(AuthService.class);
    private final CredentialsRepository credentialsRepository;
    private final EmailVerificationRepository emailVerificationRepository;
    private final RefreshTokenRepository refreshTokenRepository;
    private final RegistrationPersistenceService registrationPersistenceService;
    private final VerificationCodeService verificationCodeService;
    private final EmailClient emailClient;
    private final PasswordEncoder passwordEncoder;
    private final JwtService jwtService;

    public AuthService(
            CredentialsRepository credentialsRepository,
            EmailVerificationRepository emailVerificationRepository,
            RefreshTokenRepository refreshTokenRepository,
            RegistrationPersistenceService registrationPersistenceService,
            VerificationCodeService verificationCodeService,
            EmailClient emailClient,
            PasswordEncoder passwordEncoder,
            JwtService jwtService
    ) {
        this.credentialsRepository = credentialsRepository;
        this.emailVerificationRepository = emailVerificationRepository;
        this.refreshTokenRepository = refreshTokenRepository;
        this.registrationPersistenceService = registrationPersistenceService;
        this.verificationCodeService = verificationCodeService;
        this.emailClient = emailClient;
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
                .doOnSuccess(saved ->
                    log.info("Отправил данные о новом пользователе в кафку, userId (ЗАГЛУШКА)")
                )
                .map(saved -> new RegisterResponse(saved.getUsername(), saved.getEmail()));
    }

    public Mono<TokenResponse> login(LoginRequest request) {

        return credentialsRepository.findByEmail(request.email())
                .switchIfEmpty(Mono.error(new IllegalArgumentException("Неверный email или пароль")))
                .flatMap(credentials -> {
                    if (!passwordEncoder.matches(request.password(), credentials.getPassword())) {
                        return Mono.error(new IllegalArgumentException("Неверный email или пароль"));
                    }

                    if (credentials.getStatus() != AccountStatus.ACTIVE) {
                        return Mono.error(new ResponseStatusException(HttpStatus.FORBIDDEN, "Подтвердите почту"));
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
                .switchIfEmpty(Mono.error(new IllegalArgumentException("Невалидный токен")))
                .flatMap(existingToken -> {
                    if (existingToken.isRevoked() || existingToken.getExpiresAt().isBefore(Instant.now())) {
                        return Mono.error(new IllegalArgumentException("Сессия истекла, авторизуйтесь заново"));
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
                .switchIfEmpty(Mono.error(new ResponseStatusException(HttpStatus.UNAUTHORIZED, "Отсутствует или неверный формат заголовка")))
                .map(header -> header.substring(7))
                .flatMap(jwtService::validateAndExtractUserId)
                .onErrorMap(e -> new ResponseStatusException(HttpStatus.UNAUTHORIZED, "Токен недействителен или просрочен"));
    }

    private String getErrorMessage(Tuple2<Boolean, Boolean> tuple) {
        if (tuple.getT1() && tuple.getT2()) return "Email и Username уже заняты";
        if (tuple.getT1()) return "Email уже занят";
        return "Username уже занят";
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

                                credentials.setStatus(AccountStatus.ACTIVE);

                                return credentialsRepository.save(credentials)
                                        .then(emailVerificationRepository.deleteByAccountId(credentials.getId()))
                                        .thenReturn(EmailVerificationResult.VERIFIED);
                            })
                            .switchIfEmpty(Mono.just(EmailVerificationResult.NOT_FOUND));
                })
                .switchIfEmpty(Mono.just(EmailVerificationResult.NOT_FOUND));
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
                                .onErrorMap(error -> new ResponseStatusException(HttpStatus.SERVICE_UNAVAILABLE, "Не удалось отправить код. Попробуйте ещё раз", error))
                                .then(Mono.defer(() -> registrationPersistenceService
                                        .markSentEmail(emailVerification)))
                                .then()).orElseGet(() -> Mono.error(new ResponseStatusException(HttpStatus.GONE, "Срок подтверждения истёк. Зарегистрируйтесь заново")));

                    });
        });
    }

}
