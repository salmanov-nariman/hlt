package com.hlt.auth.grpc.client;

import com.google.protobuf.Empty;
import com.hlt.user.grpc.CreateUserProfileRequest;
import com.hlt.user.grpc.UserServiceGrpc;
import io.grpc.stub.StreamObserver;
import org.springframework.stereotype.Component;
import reactor.core.publisher.Mono;

import java.util.UUID;
import java.util.concurrent.TimeUnit;

@Component
public class UserClient {

    private final UserServiceGrpc.UserServiceStub stub;

    public UserClient(UserServiceGrpc.UserServiceStub stub) {
        this.stub = stub;
    }

    public Mono<Void> createProfile(UUID id, String username, String email) {
        return Mono.create(sink -> {
            CreateUserProfileRequest request = CreateUserProfileRequest.newBuilder()
                    .setId(id.toString())
                    .setUsername(username)
                    .setEmail(email)
                    .build();

            stub.withDeadlineAfter(5, TimeUnit.SECONDS)
                    .createUserProfile(request, new StreamObserver<>() {
                        @Override
                        public void onNext(Empty response) {
                        }

                        @Override
                        public void onError(Throwable error) {
                            sink.error(error);
                        }

                        @Override
                        public void onCompleted() {
                            sink.success();
                        }
                    });
        });
    }
}
