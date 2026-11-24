package com.hlt.user.model.mapper;

import com.hlt.user.model.dto.response.GetUserResponse;
import com.hlt.user.model.entity.Users;

public class UserMapper {

    public static GetUserResponse toGetUserResponse(Users users) {
        return new GetUserResponse(
                users.getUsername(),
                users.getEmail(),
                users.getFirstname(),
                users.getLastname(),
                users.getAvatar(),
                users.getBio()
        );
    }

}
