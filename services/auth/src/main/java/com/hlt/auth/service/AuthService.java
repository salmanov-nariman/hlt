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
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;
import reactor.core.publisher.Mono;
import reactor.util.function.Tuple2;

import java.time.Instant;
import java.time.temporal.ChronoUnit;

@Service
public class AuthService {

    private static final Logger log = LoggerFactory.getLogger(AuthService.class);
    private final CredentialsRepository credentialsRepository;
    private final RefreshTokenRepository refreshTokenRepository;
    private final PasswordEncoder passwordEncoder;
    private final JwtService jwtService;

    public AuthService(CredentialsRepository credentialsRepository, RefreshTokenRepository refreshTokenRepository, PasswordEncoder passwordEncoder, JwtService jwtService) {
        this.credentialsRepository = credentialsRepository;
        this.refreshTokenRepository = refreshTokenRepository;
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

                    Credentials credentials = new Credentials(
                            null, request.username(),
                            request.email(),
                            passwordEncoder.encode(request.password()),
                            Instant.now()
                    );

                    return credentialsRepository.save(credentials);
                })
                .doOnSuccess(saved ->
                    log.info("Отправил данные о новом пользователе в кафку, userId: {}", saved.getId())
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

    private String getErrorMessage(Tuple2<Boolean, Boolean> tuple) {
        if (tuple.getT1() && tuple.getT2()) return "Email и Username уже заняты";
        if (tuple.getT1()) return "Email уже занят";
        return "Username уже занят";
    }

}
