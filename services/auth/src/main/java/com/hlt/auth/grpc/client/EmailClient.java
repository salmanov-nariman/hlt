package com.hlt.auth.grpc.client;


import com.google.protobuf.Empty;
import com.hlt.email.grpc.EmailServiceGrpc;
import com.hlt.email.grpc.SendVerificationEmailRequest;
import io.grpc.stub.StreamObserver;
import org.springframework.stereotype.Component;
import reactor.core.publisher.Mono;

import java.util.concurrent.TimeUnit;

@Component
public class EmailClient {

    private final EmailServiceGrpc.EmailServiceStub stub;

    public EmailClient(EmailServiceGrpc.EmailServiceStub stub) {
        this.stub = stub;
    }

    public Mono<Void> sendVerificationEmail(String email, String code) {
        return Mono.create(sink -> {
            SendVerificationEmailRequest request =
                    SendVerificationEmailRequest.newBuilder()
                            .setRecipientEmail(email)
                            .setVerificationCode(code)
                            .build();

            stub.withDeadlineAfter(10, TimeUnit.SECONDS)
                    .sendVerificationEmail(request, new StreamObserver<>() {
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