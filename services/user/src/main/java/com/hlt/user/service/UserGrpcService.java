package com.hlt.user.service;

import com.google.protobuf.Empty;
import com.hlt.user.grpc.CreateUserProfileRequest;
import com.hlt.user.grpc.UserServiceGrpc;
import io.grpc.Status;
import io.grpc.stub.StreamObserver;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

import java.util.UUID;

@Service
public class UserGrpcService extends UserServiceGrpc.UserServiceImplBase {

    private static final Logger log = LoggerFactory.getLogger(UserGrpcService.class);

    private static final String INVALID_PROFILE_ID = "ID должен быть UUID";
    private static final String REQUIRED_PROFILE_FIELDS_MISSING = "Имя пользователя и email обязательны";
    private static final String PROFILE_SAVE_FAILED = "Не удалось сохранить профиль";
    private static final String PROFILE_SAVE_ERROR_LOG = "Не удалось сохранить профиль {}";

    private final UserService userService;

    public UserGrpcService(UserService userService) {
        this.userService = userService;
    }

    @Override
    public void createUserProfile(CreateUserProfileRequest request, StreamObserver<Empty> responseObserver) {
        UUID id;
        try {
            id = UUID.fromString(request.getId());
        } catch (IllegalArgumentException error) {
            responseObserver.onError(Status.INVALID_ARGUMENT.withDescription(INVALID_PROFILE_ID).asRuntimeException());
            return;
        }

        if (request.getUsername().isBlank() || request.getEmail().isBlank()) {
            responseObserver.onError(Status.INVALID_ARGUMENT
                    .withDescription(REQUIRED_PROFILE_FIELDS_MISSING).asRuntimeException());
            return;
        }

        userService.createProfile(id, request.getUsername(), request.getEmail())
                .thenReturn(Empty.getDefaultInstance())
                .subscribe(responseObserver::onNext, error -> {
                    log.error(PROFILE_SAVE_ERROR_LOG, id, error);
                    responseObserver.onError(Status.INTERNAL
                            .withDescription(PROFILE_SAVE_FAILED).asRuntimeException());
                }, responseObserver::onCompleted);
    }
}
