package com.hlt.user.model.dto.request;

public record UpdateUserRequest(
        String firstname,
        String lastname,
        String bio,
        String avatar
) {
    
}
