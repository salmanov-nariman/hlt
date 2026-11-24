package com.hlt.user.controller;


import com.hlt.user.model.dto.request.UpdateUserRequest;
import com.hlt.user.model.dto.response.GetUserResponse;
import com.hlt.user.service.UserService;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Mono;

import java.util.UUID;

@RestController
@RequestMapping("/api/user")
public class UserController {

    private final UserService userService;

    public UserController(UserService userService) {
        this.userService = userService;
    }

    @GetMapping("/me")
    public Mono<GetUserResponse> getMe(@RequestHeader(value = "X-User-Id") UUID userId) {
        return userService.getMe(userId);
    }

    @PatchMapping("/me")
    public Mono<GetUserResponse> updateMe(@RequestHeader(value = "X-User-Id") UUID userId, @RequestBody UpdateUserRequest request) {
        return userService.updateMe(userId, request);
    }

}
