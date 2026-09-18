package com.hlt.auth.model.dto.request;

public record RegisterRequest(
        String username,
        String email,
        String password
) {
}
