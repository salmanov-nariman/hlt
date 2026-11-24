package com.hlt.auth.grpc.config;


import com.hlt.email.grpc.EmailServiceGrpc;
import com.hlt.user.grpc.UserServiceGrpc;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.grpc.client.GrpcChannelFactory;

@Configuration
public class GrpcClientConfig {

    @Bean
    public EmailServiceGrpc.EmailServiceStub emailServiceStub(
            GrpcChannelFactory channelFactory
    ) {
        return EmailServiceGrpc.newStub(
                channelFactory.createChannel("email")
        );
    }

    @Bean
    public UserServiceGrpc.UserServiceStub userServiceStub(GrpcChannelFactory channelFactory) {
        return UserServiceGrpc.newStub(
                channelFactory.createChannel("user")
        );
    }

}
