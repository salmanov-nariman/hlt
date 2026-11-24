package com.hlt.user.service;


import com.hlt.user.model.dto.request.UpdateUserRequest;
import com.hlt.user.model.dto.response.GetUserResponse;
import com.hlt.user.model.mapper.UserMapper;
import com.hlt.user.repository.UserRepository;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;
import org.springframework.web.server.ResponseStatusException;
import reactor.core.publisher.Mono;

import java.util.UUID;

@Service
public class UserService {

    private static final String PROFILE_NOT_FOUND = "Профиль не найден";

    private final UserRepository userRepository;

    public UserService(UserRepository userRepository) {
        this.userRepository = userRepository;
    }

    public Mono<Void> createProfile(UUID id, String username, String email) {
        return userRepository.createProfile(id, username, email);
    }

    public Mono<GetUserResponse> getMe(UUID userId) {
        return userRepository.findById(userId)
                .map(UserMapper::toGetUserResponse);
    }

    public Mono<GetUserResponse> updateMe(UUID userId, UpdateUserRequest request) {
        return userRepository.patchProfile(userId, request.firstname(), request.lastname(), request.bio(), request.avatar())
                .switchIfEmpty(Mono.error(new ResponseStatusException(HttpStatus.NOT_FOUND, PROFILE_NOT_FOUND)))
                .map(UserMapper::toGetUserResponse);
                

    }

}
