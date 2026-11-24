package com.hlt.auth.service;

import com.hlt.auth.grpc.client.EmailClient;
import com.hlt.auth.grpc.client.UserClient;
import com.hlt.auth.jwt.JwtService;
import com.hlt.auth.model.data.EmailVerificationResult;
import com.hlt.auth.model.data.RegistrationData;
import com.hlt.auth.model.dto.request.EmailResendRequest;
import com.hlt.auth.model.dto.request.EmailVerifyRequest;
import com.hlt.auth.model.dto.request.LoginRequest;
import com.hlt.auth.model.dto.request.RefreshTokenRequest;
import com.hlt.auth.model.dto.request.RegisterRequest;
import com.hlt.auth.model.dto.response.RegisterResponse;
import com.hlt.auth.model.dto.response.TokenResponse;
import com.hlt.auth.model.entity.AccountStatus;
import com.hlt.auth.model.entity.Credentials;
import com.hlt.auth.model.entity.EmailVerification;
import com.hlt.auth.model.entity.RefreshToken;
import com.hlt.auth.repository.CredentialsRepository;
import com.hlt.auth.repository.EmailVerificationRepository;
import com.hlt.auth.repository.RefreshTokenRepository;
import io.grpc.Status;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.CsvSource;
import org.junit.jupiter.params.provider.NullAndEmptySource;
import org.junit.jupiter.params.provider.ValueSource;
import org.mockito.ArgumentCaptor;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.Mockito;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.http.HttpStatus;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.web.server.ResponseStatusException;
import reactor.core.publisher.Mono;
import reactor.test.StepVerifier;

import java.time.Duration;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.Optional;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;

@ExtendWith(MockitoExtension.class)
public class AuthServiceTest {

    @InjectMocks
    private AuthService authService;

    @Mock
    private CredentialsRepository credentialsRepository;

    @Mock
    private RefreshTokenRepository refreshTokenRepository;

    @Mock
    private PasswordEncoder passwordEncoder;

    @Mock
    private JwtService jwtService;

    @Mock
    private RegistrationPersistenceService registrationPersistenceService;

    @Mock
    private VerificationCodeService verificationCodeService;

    @Mock
    private EmailVerificationRepository emailVerificationRepository;

    @Mock
    private EmailClient emailClient;

    @Mock
    private UserClient userClient;

    @DisplayName("Регистрация возвращает пользователя независимо от результата отправки письма")
    @ParameterizedTest(name = "Письмо отправлено: {0}")
    @ValueSource(booleans = {true, false})
    void shouldRegisterAndReturnUser(boolean sent) {
        RegisterRequest request = new RegisterRequest(
                "test",
                "test@gmail.com",
                "123"
        );

        String passwordHash = "password-hash";

        Credentials savedCredentials = new Credentials(
                UUID.randomUUID(),
                request.username(),
                request.email(),
                passwordHash,
                AccountStatus.PENDING_VERIFICATION,
                Instant.now()
        );

        Mockito.when(credentialsRepository.existsByEmail(request.email()))
                .thenReturn(Mono.just(false));

        Mockito.when(credentialsRepository.existsByUsername(request.username()))
                .thenReturn(Mono.just(false));

        Mockito.when(passwordEncoder.encode(request.password()))
                .thenReturn(passwordHash);

        String code = "012345";
        String codeHash = "code-hash";
        EmailVerification savedVerification = new EmailVerification(
                UUID.randomUUID(), savedCredentials.getId(), codeHash,
                Instant.now().plusSeconds(300), 0, null
        );

        Mockito.when(verificationCodeService.generateCode()).thenReturn(code);
        Mockito.when(verificationCodeService.hashCode(code)).thenReturn(codeHash);
        Mockito.when(verificationCodeService.getCodeExpiration()).thenReturn(Duration.ofMinutes(5));
        Mockito.when(registrationPersistenceService.createRegistrationData(
                Mockito.any(Credentials.class), Mockito.any(EmailVerification.class)
        )).thenReturn(Mono.just(new RegistrationData(savedCredentials, savedVerification)));
        Mockito.when(emailClient.sendVerificationEmail(request.email(), code))
                .thenReturn(sent ? Mono.empty() : Mono.error(new IllegalStateException("Email service unavailable")));

        if (sent) {
            Mockito.when(registrationPersistenceService.markSentEmail(savedVerification))
                    .thenReturn(Mono.just(savedVerification));
        }

        StepVerifier.create(authService.register(request))
                .expectNext(new RegisterResponse(
                        request.username(),
                        request.email()
                ))
                .verifyComplete();

        ArgumentCaptor<Credentials> captor =
                ArgumentCaptor.forClass(Credentials.class);

        ArgumentCaptor<EmailVerification> verificationCaptor =
                ArgumentCaptor.forClass(EmailVerification.class);
        Mockito.verify(registrationPersistenceService)
                .createRegistrationData(captor.capture(), verificationCaptor.capture());

        Credentials credentialsToSave = captor.getValue();

        assertAll(
                () -> assertNull(credentialsToSave.getId()),
                () -> assertEquals(request.username(), credentialsToSave.getUsername()),
                () -> assertEquals(request.email(), credentialsToSave.getEmail()),
                () -> assertEquals(passwordHash, credentialsToSave.getPassword()),
                () -> assertEquals(AccountStatus.PENDING_VERIFICATION, credentialsToSave.getStatus()),
                () -> assertNotNull(credentialsToSave.getCreatedAt()),
                () -> assertEquals(codeHash, verificationCaptor.getValue().getCodeHash())
        );

        Mockito.verify(emailClient).sendVerificationEmail(request.email(), code);
        Mockito.verify(registrationPersistenceService, Mockito.times(sent ? 1 : 0))
                .markSentEmail(savedVerification);
    }

    @DisplayName("Отказ в регистрации при занятых данных")
    @ParameterizedTest(name = "[{index}] email занят: {0}, username занят: {1} -> {2}")
    @CsvSource({
            "true, false, Email уже занят",
            "false, true, Username уже занят",
            "true, true, Email и Username уже заняты"
    })
    void shouldRejectRegistration(
            boolean emailExists,
            boolean usernameExists,
            String expectedMessage
    ) {
        RegisterRequest request = new RegisterRequest(
                "test",
                "test@gmail.com",
                "123"
        );

        Mockito.when(credentialsRepository.existsByEmail(request.email()))
                .thenReturn(Mono.just(emailExists));

        Mockito.when(credentialsRepository.existsByUsername(request.username()))
                .thenReturn(Mono.just(usernameExists));

        StepVerifier.create(authService.register(request))
                .expectErrorMatches(error ->
                        error instanceof IllegalArgumentException
                                && expectedMessage.equals(error.getMessage()))
                .verify();

        Mockito.verifyNoInteractions(registrationPersistenceService, emailClient);
    }

    @DisplayName("Аккаунт активируется только после создания профиля")
    @ParameterizedTest(name = "Профиль создан: {0}")
    @ValueSource(booleans = {true, false})
    void shouldVerifyPendingAccount(boolean profileCreated) {
        Instant now = Instant.now();
        Credentials account = pendingAccount(now.minusSeconds(60));
        EmailVerifyRequest request = new EmailVerifyRequest(account.getEmail(), "012345");
        EmailVerification verification = new EmailVerification(
                UUID.randomUUID(), account.getId(), "code-hash", now.plusSeconds(240), 0, null
        );

        Mockito.when(credentialsRepository.findByEmailForUpdate(request.email()))
                .thenReturn(Mono.just(account));
        Mockito.when(emailVerificationRepository.findByAccountId(account.getId()))
                .thenReturn(Mono.just(verification));
        Mockito.when(verificationCodeService.getMaxAttempts()).thenReturn(5);
        Mockito.when(verificationCodeService.matches(request.verificationCode(), verification.getCodeHash()))
                .thenReturn(true);

        if (!profileCreated) {
            Mockito.when(userClient.createProfile(account.getId(), account.getUsername(), account.getEmail()))
                    .thenReturn(Mono.error(Status.UNAVAILABLE.asRuntimeException()));

            StepVerifier.create(authService.emailVerify(request))
                    .expectErrorMatches(error -> error instanceof ResponseStatusException response
                            && response.getStatusCode() == HttpStatus.SERVICE_UNAVAILABLE)
                    .verify();

            assertEquals(AccountStatus.PENDING_VERIFICATION, account.getStatus());
            Mockito.verify(credentialsRepository, Mockito.never()).save(account);
            Mockito.verify(emailVerificationRepository, Mockito.never()).deleteByAccountId(account.getId());
            return;
        }

        Mockito.when(userClient.createProfile(account.getId(), account.getUsername(), account.getEmail()))
                .thenReturn(Mono.empty());
        Mockito.when(credentialsRepository.save(account)).thenReturn(Mono.just(account));
        Mockito.when(emailVerificationRepository.deleteByAccountId(account.getId())).thenReturn(Mono.empty());

        StepVerifier.create(authService.emailVerify(request))
                .expectNext(EmailVerificationResult.VERIFIED)
                .verifyComplete();

        assertEquals(AccountStatus.ACTIVE, account.getStatus());
        var calls = Mockito.inOrder(userClient, credentialsRepository, emailVerificationRepository);
        calls.verify(userClient).createProfile(account.getId(), account.getUsername(), account.getEmail());
        calls.verify(credentialsRepository).save(account);
        calls.verify(emailVerificationRepository).deleteByAccountId(account.getId());
    }

    @DisplayName("Истёкший срок регистрации блокирует подтверждение")
    @Test
    void shouldRejectVerificationWhenRegistrationExpired() {
        Credentials account = pendingAccount(Instant.now().minusSeconds(16 * 60));
        EmailVerifyRequest request = new EmailVerifyRequest(account.getEmail(), "012345");

        Mockito.when(credentialsRepository.findByEmailForUpdate(request.email()))
                .thenReturn(Mono.just(account));
        Mockito.when(registrationPersistenceService.isRegistrationExpired(
                Mockito.eq(account.getCreatedAt()), Mockito.any(Instant.class)
        )).thenReturn(true);

        StepVerifier.create(authService.emailVerify(request))
                .expectNext(EmailVerificationResult.EXPIRED)
                .verifyComplete();

        assertEquals(AccountStatus.PENDING_VERIFICATION, account.getStatus());
        Mockito.verify(credentialsRepository, Mockito.never()).save(account);
        Mockito.verifyNoInteractions(emailVerificationRepository, verificationCodeService);
    }

    @DisplayName("Просроченный код отклоняется, даже если регистрация ещё действует")
    @Test
    void shouldRejectExpiredVerificationCode() {
        Instant now = Instant.now();
        Credentials account = pendingAccount(now.minusSeconds(6 * 60));
        EmailVerifyRequest request = new EmailVerifyRequest(account.getEmail(), "012345");
        EmailVerification verification = new EmailVerification(
                UUID.randomUUID(), account.getId(), "code-hash", now.minusSeconds(60), 0, null
        );

        Mockito.when(credentialsRepository.findByEmailForUpdate(request.email()))
                .thenReturn(Mono.just(account));
        Mockito.when(emailVerificationRepository.findByAccountId(account.getId()))
                .thenReturn(Mono.just(verification));

        StepVerifier.create(authService.emailVerify(request))
                .expectNext(EmailVerificationResult.EXPIRED)
                .verifyComplete();

        assertEquals(AccountStatus.PENDING_VERIFICATION, account.getStatus());
        Mockito.verify(credentialsRepository, Mockito.never()).save(account);
        Mockito.verifyNoInteractions(verificationCodeService);
    }

    @DisplayName("Повторная отправка сохраняет отметку только после успешной отправки")
    @ParameterizedTest(name = "Письмо отправлено: {0}")
    @ValueSource(booleans = {true, false})
    void shouldHandleVerificationEmailResend(boolean sent) {
        EmailResendRequest request = new EmailResendRequest("test@gmail.com");
        String code = "012345";
        String codeHash = "code-hash";
        EmailVerification verification = new EmailVerification(
                UUID.randomUUID(), UUID.randomUUID(), codeHash, Instant.now().plusSeconds(300), 0, null
        );

        Mockito.when(verificationCodeService.generateCode()).thenReturn(code);
        Mockito.when(verificationCodeService.hashCode(code)).thenReturn(codeHash);
        Mockito.when(registrationPersistenceService.prepareEmailResend(request.email(), codeHash))
                .thenReturn(Mono.just(Optional.of(verification)));
        Mockito.when(emailClient.sendVerificationEmail(request.email(), code))
                .thenReturn(sent ? Mono.empty() : Mono.error(new IllegalStateException("Email service unavailable")));

        if (sent) {
            Mockito.when(registrationPersistenceService.markSentEmail(verification))
                    .thenReturn(Mono.just(verification));
            StepVerifier.create(authService.emailResend(request)).verifyComplete();

            var order = Mockito.inOrder(registrationPersistenceService, emailClient);
            order.verify(registrationPersistenceService).prepareEmailResend(request.email(), codeHash);
            order.verify(emailClient).sendVerificationEmail(request.email(), code);
            order.verify(registrationPersistenceService).markSentEmail(verification);
        } else {
            StepVerifier.create(authService.emailResend(request))
                    .expectErrorMatches(error -> error instanceof ResponseStatusException ex
                            && HttpStatus.SERVICE_UNAVAILABLE.equals(ex.getStatusCode()))
                    .verify();
            Mockito.verify(registrationPersistenceService, Mockito.never()).markSentEmail(verification);
        }
    }

    @DisplayName("Повторная отправка после истечения регистрации возвращает 410 без письма")
    @Test
    void shouldRejectResendWhenRegistrationExpired() {
        EmailResendRequest request = new EmailResendRequest("test@gmail.com");
        String code = "012345";
        String codeHash = "code-hash";

        Mockito.when(verificationCodeService.generateCode()).thenReturn(code);
        Mockito.when(verificationCodeService.hashCode(code)).thenReturn(codeHash);
        Mockito.when(registrationPersistenceService.prepareEmailResend(request.email(), codeHash))
                .thenReturn(Mono.just(Optional.empty()));

        StepVerifier.create(authService.emailResend(request))
                .expectErrorMatches(error -> error instanceof ResponseStatusException ex
                        && HttpStatus.GONE.equals(ex.getStatusCode())
                        && "Срок подтверждения истёк. Зарегистрируйтесь заново".equals(ex.getReason()))
                .verify();

        Mockito.verifyNoInteractions(emailClient);
        Mockito.verify(registrationPersistenceService, Mockito.never())
                .markSentEmail(Mockito.any());
    }

    private Credentials pendingAccount(Instant createdAt) {
        return new Credentials(
                UUID.randomUUID(), "test", "test@gmail.com", "password-hash",
                AccountStatus.PENDING_VERIFICATION, createdAt
        );
    }

    @DisplayName("Успешный логин пользователя")
    @Test
    void shouldLoginUserAndReturnTokens() {
        LoginRequest request = new LoginRequest(
                "test@gmail.com",
                "123"
        );

        String passwordHash = "password-hash";

        Credentials foundUser = new Credentials(
                UUID.randomUUID(),
                "Test",
                request.email(),
                passwordHash,
                AccountStatus.ACTIVE,
                Instant.now()
        );

        Mockito.when(credentialsRepository.findByEmail(request.email()))
                .thenReturn(Mono.just(foundUser));

        Mockito.when(passwordEncoder.matches(request.password(), passwordHash))
                .thenReturn(true);

        Mockito.when(jwtService.generateAccessToken(foundUser.getId()))
                .thenReturn("access-token");

        Mockito.when(jwtService.generateRefreshToken())
                .thenReturn("refresh-token");

        Mockito.when(refreshTokenRepository.save(Mockito.any(RefreshToken.class)))
                .thenAnswer(invocation -> {
                    RefreshToken token = invocation.getArgument(0);
                    return Mono.just(token);
                });

        Instant beforeLogin = Instant.now();

        StepVerifier.create(authService.login(request))
                .expectNext(new TokenResponse(
                        "refresh-token",
                        "access-token"
                ))
                .verifyComplete();
        Instant afterLogin = Instant.now();

        Mockito.verify(passwordEncoder)
                .matches(request.password(), passwordHash);

        Mockito.verify(jwtService)
                .generateAccessToken(foundUser.getId());

        Mockito.verify(jwtService)
                .generateRefreshToken();

        ArgumentCaptor<RefreshToken> captor =
                ArgumentCaptor.forClass(RefreshToken.class);

        Mockito.verify(refreshTokenRepository).save(captor.capture());

        RefreshToken tokenToSave = captor.getValue();

        assertAll(
                () -> assertEquals(foundUser.getId(), tokenToSave.getAccountId()),
                () -> assertEquals("refresh-token", tokenToSave.getToken()),
                () -> assertFalse(tokenToSave.isRevoked()),
                () -> assertNull(tokenToSave.getId()),
                () -> assertFalse(tokenToSave.getExpiresAt().isBefore(beforeLogin.plus(30, ChronoUnit.DAYS))),
                () -> assertFalse(tokenToSave.getExpiresAt().isAfter(afterLogin.plus(30, ChronoUnit.DAYS)))
        );
    }

    @DisplayName("Вход с неизвестным email")
    @Test
    void shouldRejectLoginWhenUserIsNotFound() {
        LoginRequest request = new LoginRequest(
                "test@gmail.com",
                "123"
        );

        Mockito.when(credentialsRepository.findByEmail(request.email()))
                .thenReturn(Mono.empty());

        StepVerifier.create(authService.login(request))
                .expectErrorMatches(error ->
                        error instanceof IllegalArgumentException
                                && "Неверный email или пароль".equals(error.getMessage()))
                .verify();

        Mockito.verifyNoInteractions(passwordEncoder, jwtService, refreshTokenRepository);
    }

    @DisplayName("Вход с неверным паролем")
    @Test
    void shouldRejectLoginWhenPasswordDoesNotMatch() {
        LoginRequest request = new LoginRequest(
                "test@gmail.com",
                "wrong-password"
        );

        String passwordHash = "password-hash";

        Credentials foundUser = new Credentials(
                UUID.randomUUID(),
                "Test",
                request.email(),
                passwordHash,
                AccountStatus.ACTIVE,
                Instant.now()
        );

        Mockito.when(credentialsRepository.findByEmail(request.email()))
                .thenReturn(Mono.just(foundUser));

        Mockito.when(passwordEncoder.matches(request.password(), passwordHash))
                .thenReturn(false);

        StepVerifier.create(authService.login(request))
                .expectErrorMatches(error ->
                        error instanceof IllegalArgumentException
                                && "Неверный email или пароль".equals(error.getMessage()))
                .verify();

        Mockito.verify(passwordEncoder)
                .matches(request.password(), passwordHash);

        Mockito.verifyNoInteractions(jwtService, refreshTokenRepository);
    }

    @DisplayName("Успешное обновление токенов")
    @Test
    void shouldRefreshTokens() {
        RefreshTokenRequest request = new RefreshTokenRequest("old-refresh");

        UUID tokenId = UUID.randomUUID();
        UUID accountId = UUID.randomUUID();

        RefreshToken existingToken = new RefreshToken(
                tokenId,
                accountId,
                request.refreshToken(),
                Instant.now().plus(1, ChronoUnit.DAYS),
                false
        );

        Mockito.when(refreshTokenRepository.findByToken(request.refreshToken()))
                .thenReturn(Mono.just(existingToken));

        Mockito.when(jwtService.generateAccessToken(accountId))
                .thenReturn("new-access");

        Mockito.when(jwtService.generateRefreshToken())
                .thenReturn("new-refresh");

        Mockito.when(refreshTokenRepository.save(Mockito.any(RefreshToken.class)))
                .thenAnswer(invocation -> {
                    RefreshToken token = invocation.getArgument(0);
                    return Mono.just(token);
                });

        Instant beforeRefresh = Instant.now();

        StepVerifier.create(authService.refresh(request))
                .expectNext(new TokenResponse(
                        "new-refresh",
                        "new-access"
                ))
                .verifyComplete();
        Instant afterRefresh = Instant.now();

        ArgumentCaptor<RefreshToken> captor =
                ArgumentCaptor.forClass(RefreshToken.class);

        Mockito.verify(refreshTokenRepository).save(captor.capture());

        RefreshToken tokenToSave = captor.getValue();

        assertAll(
                () -> assertEquals(tokenId, tokenToSave.getId()),
                () -> assertEquals(accountId, tokenToSave.getAccountId()),
                () -> assertEquals("new-refresh", tokenToSave.getToken()),
                () -> assertFalse(tokenToSave.isRevoked()),
                () -> assertFalse(tokenToSave.getExpiresAt().isBefore(beforeRefresh.plus(30, ChronoUnit.DAYS))),
                () -> assertFalse(tokenToSave.getExpiresAt().isAfter(afterRefresh.plus(30, ChronoUnit.DAYS)))
        );

        Mockito.verify(jwtService)
                .generateAccessToken(accountId);
    }

    @DisplayName("Обновление с неизвестным refresh-токеном")
    @Test
    void shouldRejectRefreshWhenTokenIsNotFound() {
        RefreshTokenRequest request = new RefreshTokenRequest("unknown-token");

        Mockito.when(refreshTokenRepository.findByToken(request.refreshToken()))
                .thenReturn(Mono.empty());

        StepVerifier.create(authService.refresh(request))
                .expectErrorMatches(error ->
                        error instanceof IllegalArgumentException
                                && "Невалидный токен".equals(error.getMessage()))
                .verify();

        Mockito.verifyNoInteractions(jwtService);
        Mockito.verify(refreshTokenRepository, Mockito.never())
                .save(Mockito.any(RefreshToken.class));
    }

    @DisplayName("Обновление с отозванным или просроченным refresh-токеном")
    @ParameterizedTest(name = "[{index}] отозван: {0}, дней до истечения: {1}")
    @CsvSource({
            "true, 1",
            "false, -1",
            "true, -1"
    })
    void shouldRejectRefreshWhenTokenIsInactive(
            boolean revoked,
            long daysUntilExpiration
    ) {
        RefreshTokenRequest request = new RefreshTokenRequest("old-refresh");

        Instant expiresAt = Instant.now().plus(daysUntilExpiration, ChronoUnit.DAYS);

        RefreshToken existingToken = new RefreshToken(
                UUID.randomUUID(),
                UUID.randomUUID(),
                request.refreshToken(),
                expiresAt,
                revoked
        );

        Mockito.when(refreshTokenRepository.findByToken(request.refreshToken()))
                .thenReturn(Mono.just(existingToken));

        StepVerifier.create(authService.refresh(request))
                .expectErrorMatches(error ->
                        error instanceof IllegalArgumentException
                                && "Сессия истекла, авторизуйтесь заново".equals(error.getMessage()))
                .verify();

        Mockito.verifyNoInteractions(jwtService);
        Mockito.verify(refreshTokenRepository, Mockito.never())
                .save(Mockito.any(RefreshToken.class));

        assertAll(
                () -> assertEquals(request.refreshToken(), existingToken.getToken()),
                () -> assertEquals(expiresAt, existingToken.getExpiresAt())
        );
    }

    @DisplayName("Проверка действительного access-токена")
    @Test
    void shouldValidateTokenAndReturnUserId() {
        UUID userId = UUID.randomUUID();

        Mockito.when(jwtService.validateAndExtractUserId("access-token"))
                .thenReturn(Mono.just(userId));

        StepVerifier.create(authService.validate("Bearer access-token"))
                .expectNext(userId)
                .verifyComplete();

        Mockito.verify(jwtService)
                .validateAndExtractUserId("access-token");
    }

    @DisplayName("Отсутствующий или некорректный заголовок авторизации")
    @ParameterizedTest(name = "[{index}] заголовок: {0}")
    @NullAndEmptySource
    @ValueSource(strings = {
            "Basic access-token",
            "Bearer",
            "bearer access-token",
            "access-token"
    })
    void shouldRejectInvalidAuthorizationHeader(String header) {
        StepVerifier.create(authService.validate(header))
                .expectErrorSatisfies(error -> {
                    ResponseStatusException exception = assertInstanceOf(ResponseStatusException.class, error);
                    assertEquals(HttpStatus.UNAUTHORIZED, exception.getStatusCode());
                })
                .verify();

        Mockito.verifyNoInteractions(jwtService);
    }

    @DisplayName("Проверка недействительного access-токена")
    @Test
    void shouldRejectTokenWhenJwtValidationFails() {
        Mockito.when(jwtService.validateAndExtractUserId("invalid-token"))
                .thenReturn(Mono.error(new IllegalArgumentException("Невалидный токен")));

        StepVerifier.create(authService.validate("Bearer invalid-token"))
                .expectErrorSatisfies(error -> {
                    ResponseStatusException exception = assertInstanceOf(ResponseStatusException.class, error);
                    assertAll(
                            () -> assertEquals(HttpStatus.UNAUTHORIZED, exception.getStatusCode()),
                            () -> assertEquals("Токен недействителен или просрочен", exception.getReason())
                    );
                })
                .verify();
    }
}
