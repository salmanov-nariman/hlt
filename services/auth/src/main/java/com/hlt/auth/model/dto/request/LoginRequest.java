package com.hlt.auth.model.dto.request;

public record LoginRequest(
        String email,
        String password
) {
}
