package com.hlt.auth.repository;


import com.hlt.auth.model.entity.EmailVerification;
import org.springframework.data.repository.reactive.ReactiveCrudRepository;
import org.springframework.stereotype.Repository;
import reactor.core.publisher.Mono;

import java.util.UUID;

@Repository
public interface EmailVerificationRepository extends ReactiveCrudRepository<EmailVerification, UUID> {
    Mono<EmailVerification> findByAccountId(UUID accountId);
    Mono<Void> deleteByAccountId(UUID accountId);
}
