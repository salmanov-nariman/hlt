package com.hlt.auth.repository;


import com.hlt.auth.model.entity.Credentials;
import org.springframework.data.repository.reactive.ReactiveCrudRepository;
import org.springframework.stereotype.Repository;
import reactor.core.publisher.Mono;

import java.util.UUID;

@Repository
public interface CredentialsRepository extends ReactiveCrudRepository<Credentials, UUID> {
    Mono<Credentials> findByEmail(String email);
    Mono<Boolean> existsByEmail(String email);
    Mono<Boolean> existsByUsername(String username);
}
