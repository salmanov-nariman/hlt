package com.hlt.user.model.dto.response;

public record GetUserResponse(
        String username,
        String email,
        String firstname,
        String lastname,
        String avatar,
        String bio
) {

}
