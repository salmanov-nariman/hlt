package com.hlt.email.service;


import com.hlt.email.grpc.EmailServiceGrpc;
import com.hlt.email.grpc.SendVerificationEmailRequest;
import com.google.protobuf.Empty;
import io.grpc.Status;
import io.grpc.stub.StreamObserver;
import org.springframework.mail.MailException;
import org.springframework.mail.SimpleMailMessage;
import org.springframework.mail.javamail.JavaMailSender;
import org.springframework.stereotype.Service;

@Service
public class EmailGrpcService extends EmailServiceGrpc.EmailServiceImplBase {

    private final JavaMailSender mailSender;

    public EmailGrpcService(JavaMailSender mailSender) {
        this.mailSender = mailSender;
    }

    @Override
    public void sendVerificationEmail(
            SendVerificationEmailRequest request,
            StreamObserver<Empty> responseObserver) {

        var message = new SimpleMailMessage();
        message.setTo(request.getRecipientEmail());
        message.setSubject("Подтверждение почты в DuoGoal");
        message.setText("Твой код подтверждения: " + request.getVerificationCode());

        try {
            mailSender.send(message);
        } catch (MailException e) {
            responseObserver.onError(Status.UNAVAILABLE
                    .withDescription("Не удалось отправить письмо")
                    .asRuntimeException());
            return;
        }

        responseObserver.onNext(Empty.getDefaultInstance());
        responseObserver.onCompleted();
    }
}
