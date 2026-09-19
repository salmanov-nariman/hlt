package com.hlt.auth.controller;


import com.hlt.auth.model.dto.request.LoginRequest;
import com.hlt.auth.model.dto.request.RefreshTokenRequest;
import com.hlt.auth.model.dto.request.RegisterRequest;
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
    public Mono<ResponseEntity<?>> register(@RequestBody RegisterRequest request) {
        return authService.register(request)
                .map(response -> ResponseEntity.status(HttpStatus.CREATED).body(response));
    }

    @PostMapping("/login")
    public Mono<ResponseEntity<?>> login(@RequestBody LoginRequest request) {
        return authService.login(request)
                .map(response -> ResponseEntity.status(HttpStatus.OK).body(response));
    }

    @PostMapping("/refresh")
    public Mono<ResponseEntity<?>> refresh(@RequestBody RefreshTokenRequest request) {
        return authService.refresh(request)
                .map(response -> ResponseEntity.status(HttpStatus.OK).body(response));
    }

}
