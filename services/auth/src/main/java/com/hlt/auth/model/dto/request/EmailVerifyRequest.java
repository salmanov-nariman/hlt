package com.hlt.auth.model.dto.request;

public record EmailVerifyRequest(
        String email,
        String verificationCode
) {
}
