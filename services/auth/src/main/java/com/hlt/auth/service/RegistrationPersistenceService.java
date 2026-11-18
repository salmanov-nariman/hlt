package com.hlt.auth.service;


import com.hlt.auth.model.data.RegistrationData;
import com.hlt.auth.model.entity.AccountStatus;
import com.hlt.auth.model.entity.Credentials;
import com.hlt.auth.model.entity.EmailVerification;
import com.hlt.auth.repository.CredentialsRepository;
import com.hlt.auth.repository.EmailVerificationRepository;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.HttpStatus;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.server.ResponseStatusException;
import reactor.core.publisher.Mono;

import java.time.Duration;
import java.time.Instant;
import java.util.Optional;

@Service
public class RegistrationPersistenceService {

    private final CredentialsRepository credentialsRepository;
    private final EmailVerificationRepository emailVerificationRepository;
    private final VerificationCodeService verificationCodeService;
    private final Duration registrationExpiration;

    public RegistrationPersistenceService(
            CredentialsRepository credentialsRepository,
            EmailVerificationRepository emailVerificationRepository,
            VerificationCodeService verificationCodeService,
            @Value("${verification.registration-expiration}") Duration registrationExpiration
    ) {
        this.credentialsRepository = credentialsRepository;
        this.emailVerificationRepository = emailVerificationRepository;
        this.registrationExpiration = registrationExpiration;
        this.verificationCodeService = verificationCodeService;
    }

    @Transactional
    public Mono<RegistrationData> createRegistrationData(Credentials credentials, EmailVerification emailVerification) {
        return credentialsRepository.save(credentials)
                .flatMap(savedCredentials -> {
                    emailVerification.setAccountId(savedCredentials.getId());

                    return emailVerificationRepository.save(emailVerification)
                            .map(savedVerification ->
                                new RegistrationData(savedCredentials, savedVerification)
                            );
                });
    }

    @Transactional
    public Mono<Optional<EmailVerification>> prepareEmailResend(
            String email, String codeHash
    ) {
        return credentialsRepository.findByEmailForUpdate(email)
                .switchIfEmpty(Mono.error(new ResponseStatusException(HttpStatus.GONE, "Аккаунт не найден. Зарегистрируйтесь заново")))
                .flatMap(account -> {
                    if (account.getStatus() != AccountStatus.PENDING_VERIFICATION) {
                        return Mono.error(new ResponseStatusException(HttpStatus.BAD_REQUEST, "Повторная отправка недоступна"));
                    }

                    Instant now = Instant.now();

                    if (isRegistrationExpired(account.getCreatedAt(), now)) {
                        return credentialsRepository.deleteById(account.getId())
                                .thenReturn(Optional.empty());
                    }

                    return emailVerificationRepository
                            .findByAccountId(account.getId())
                            .switchIfEmpty(Mono.error(new ResponseStatusException(HttpStatus.BAD_REQUEST, "Запись подтверждения не найдена")))
                            .flatMap(verification -> {
                                Instant lastSentAt = verification.getLastSentAt();

                                if (lastSentAt != null && now.isBefore(lastSentAt.plusSeconds(60))) {
                                    return Mono.error(new ResponseStatusException(HttpStatus.TOO_MANY_REQUESTS, "Повторите запрос позже"));
                                }

                                verification.setCodeHash(codeHash);
                                verification.setExpiresAt(now.plus(verificationCodeService.getCodeExpiration()));
                                verification.setFailedAttempts(0);

                                return emailVerificationRepository.save(verification)
                                        .map(Optional::of);
                            });
                });
    }

    public Mono<EmailVerification> markSentEmail(EmailVerification emailVerification) {
        emailVerification.setLastSentAt(Instant.now());
        return emailVerificationRepository.save(emailVerification);
    }

    public boolean isRegistrationExpired(Instant createdAt, Instant now) {
        return !now.isBefore(createdAt.plus(registrationExpiration));
    }

}
