package com.hlt.auth.model.dto.response;

public record TokenResponse(
        String refreshToken,
        String accessToken
) {
}
