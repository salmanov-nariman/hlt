package com.hlt.auth.service;


import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.stereotype.Service;

import java.security.SecureRandom;
import java.time.Duration;

@Service
public class VerificationCodeService {

    private final SecureRandom secureRandom = new SecureRandom();
    private final PasswordEncoder passwordEncoder;
    private final Duration codeExpiration;
    private final int maxAttempts;


    public VerificationCodeService(PasswordEncoder passwordEncoder, @Value("${verification.code-expiration}") Duration codeExpiration, @Value("${verification.max-attempts}") int maxAttempts) {
        this.passwordEncoder = passwordEncoder;
        this.codeExpiration = codeExpiration;
        this.maxAttempts = maxAttempts;
    }

    public String generateCode() {
        int code = secureRandom.nextInt(1_000_000);
        return String.format("%06d", code);
    }

    public String hashCode(String code) {
        return passwordEncoder.encode(code);
    }

    public boolean matches(String code, String codeHash) {
        return passwordEncoder.matches(code, codeHash);
    }

    public Duration getCodeExpiration() {
        return codeExpiration;
    }

    public int getMaxAttempts() { return maxAttempts; }

}
