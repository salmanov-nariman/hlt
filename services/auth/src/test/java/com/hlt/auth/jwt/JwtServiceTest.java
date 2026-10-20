package com.hlt.auth.jwt;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import reactor.test.StepVerifier;

import java.util.Base64;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;

public class JwtServiceTest {

    private JwtService jwtService;

    @BeforeEach
    void setUp() {
        String secret = Base64.getEncoder()
                .encodeToString(new byte[32]);

        long accessExpiration = 900_000L;

        jwtService = new JwtService(secret, accessExpiration);
    }

    @Test
    @DisplayName("Генерируем access токен")
    public void generateAccessToken() {
        UUID userId = UUID.randomUUID();

        String accessToken = jwtService.generateAccessToken(userId);

        assertNotNull(accessToken);
    }

    @Test
    @DisplayName("Генерируем refresh токен")
    public void generateRefreshToken() {
        String refreshToken = jwtService.generateRefreshToken();

        assertNotNull(refreshToken);
    }

    @Test
    @DisplayName("Проверяем и достаем userId из токена")
    public void validateAndExtractUserId() {
        UUID userId = UUID.randomUUID();
        String token = jwtService.generateAccessToken(userId);

        StepVerifier.create(jwtService.validateAndExtractUserId(token))
                .expectNext(userId)
                .verifyComplete();
    }

    @Test
    @DisplayName("Проверяем невалидный токен")
    public void validateNotValidToken() {
        String token = UUID.randomUUID().toString();

        StepVerifier.create(jwtService.validateAndExtractUserId(token))
                .expectError(IllegalArgumentException.class)
                .verify();
    }

}
