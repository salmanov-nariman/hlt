package com.hlt.auth.controller;


import com.hlt.auth.model.dto.request.*;
import com.hlt.auth.model.dto.response.RegisterResponse;
import com.hlt.auth.model.dto.response.TokenResponse;
import com.hlt.auth.service.AuthService;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Mono;

@RestController
@RequestMapping("/api/auth")
public class AuthController {

    private final AuthService authService;

    public AuthController(AuthService authService) {
        this.authService = authService;
    }

    @PostMapping("/register")
    @ResponseStatus(HttpStatus.CREATED)
    public Mono<RegisterResponse> register(@RequestBody RegisterRequest request) {
        return authService.register(request);
    }

    @PostMapping("/login")
    @ResponseStatus(HttpStatus.OK)
    public Mono<TokenResponse> login(@RequestBody LoginRequest request) {
        return authService.login(request);
    }

    @PostMapping("/refresh")
    @ResponseStatus(HttpStatus.OK)
    public Mono<TokenResponse> refresh(@RequestBody RefreshTokenRequest request) {
        return authService.refresh(request);
    }

    @GetMapping("/validate")
    public Mono<ResponseEntity<Void>> validate(@RequestHeader(value = "Authorization", required = false) String authHeader) {
        return authService.validate(authHeader)
                .map(response -> ResponseEntity.status(HttpStatus.OK)
                        .header("X-User-Id", response.toString())
                        .build());
    }

    @PostMapping("/emailVerify")
    public Mono<ResponseEntity<Void>> emailVerify(@RequestBody EmailVerifyRequest request) {
        return authService.emailVerify(request)
                .map(result -> switch (result) {
                    case VERIFIED, ALREADY_VERIFIED -> ResponseEntity.noContent().build();
                    case INVALID_CODE, NOT_FOUND -> ResponseEntity.badRequest().build();
                    case EXPIRED -> ResponseEntity.status(HttpStatus.GONE).build();
                    case ATTEMPTS_EXHAUSTED ->
                            ResponseEntity.status(HttpStatus.TOO_MANY_REQUESTS).build();
                });
    }

    @PostMapping("/emailResend")
    @ResponseStatus(HttpStatus.NO_CONTENT)
    public Mono<Void> emailResend(@RequestBody EmailResendRequest request) {
        return authService.emailResend(request);
    }

}
