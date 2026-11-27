package com.hlt.user.repository;


import com.hlt.user.model.entity.Users;
import jakarta.annotation.Nullable;
import org.springframework.data.r2dbc.repository.Modifying;
import org.springframework.data.r2dbc.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.data.repository.reactive.ReactiveCrudRepository;
import org.springframework.stereotype.Repository;
import reactor.core.publisher.Mono;

import java.util.UUID;

@Repository
public interface UserRepository extends ReactiveCrudRepository<Users, UUID> {

    @Modifying
    @Query("""
            INSERT INTO users (id, username, email)
            VALUES (:id, :username, :email)
            ON CONFLICT (id) DO NOTHING
            """)
    Mono<Void> createProfile(
            @Param("id") UUID id,
            @Param("username") String username,
            @Param("email") String email
    );

    @Query("""
        UPDATE users
        SET firstname = CASE WHEN :firstname = '' THEN NULL ELSE COALESCE(:firstname, firstname) END,
            lastname  = CASE WHEN :lastname  = '' THEN NULL ELSE COALESCE(:lastname, lastname) END,
            bio       = CASE WHEN :bio       = '' THEN NULL ELSE COALESCE(:bio, bio) END,
            avatar    = CASE WHEN :avatar    = '' THEN NULL ELSE COALESCE(:avatar, avatar) END
        WHERE id = :userId
        RETURNING *
        """)
    Mono<Users> patchProfile(
            @Param("userId") UUID userId,
            @Param("firstname") @Nullable String firstname,
            @Param("lastname") @Nullable String lastname,
            @Param("bio") @Nullable String bio,
            @Param("avatar") @Nullable String avatar
    );

}
