package com.hlt.auth.service;

import com.hlt.auth.jwt.JwtService;
import com.hlt.auth.model.dto.request.LoginRequest;
import com.hlt.auth.model.dto.request.RefreshTokenRequest;
import com.hlt.auth.model.dto.request.RegisterRequest;
import com.hlt.auth.model.dto.response.RegisterResponse;
import com.hlt.auth.model.dto.response.TokenResponse;
import com.hlt.auth.model.entity.Credentials;
import com.hlt.auth.model.entity.RefreshToken;
import com.hlt.auth.repository.CredentialsRepository;
import com.hlt.auth.repository.RefreshTokenRepository;
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

import java.time.Instant;
import java.time.temporal.ChronoUnit;
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

    @DisplayName("Успешная регистрация пользователя")
    @Test
    void shouldRegisterAndReturnUser() {
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
                Instant.now()
        );

        Mockito.when(credentialsRepository.existsByEmail(request.email()))
                .thenReturn(Mono.just(false));

        Mockito.when(credentialsRepository.existsByUsername(request.username()))
                .thenReturn(Mono.just(false));

        Mockito.when(passwordEncoder.encode(request.password()))
                .thenReturn(passwordHash);

        Mockito.when(credentialsRepository.save(Mockito.any(Credentials.class)))
                .thenReturn(Mono.just(savedCredentials));

        StepVerifier.create(authService.register(request))
                .expectNext(new RegisterResponse(
                        request.username(),
                        request.email()
                ))
                .verifyComplete();

        ArgumentCaptor<Credentials> captor =
                ArgumentCaptor.forClass(Credentials.class);

        Mockito.verify(credentialsRepository).save(captor.capture());

        Credentials credentialsToSave = captor.getValue();

        assertAll(
                () -> assertNull(credentialsToSave.getId()),
                () -> assertEquals(request.username(), credentialsToSave.getUsername()),
                () -> assertEquals(request.email(), credentialsToSave.getEmail()),
                () -> assertEquals(passwordHash, credentialsToSave.getPassword()),
                () -> assertNotNull(credentialsToSave.getCreatedAt())
        );

        Mockito.verify(passwordEncoder).encode(request.password());
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

        Mockito.verify(credentialsRepository, Mockito.never())
                .save(Mockito.any(Credentials.class));
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
