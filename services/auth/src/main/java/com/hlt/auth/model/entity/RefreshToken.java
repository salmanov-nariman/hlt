package com.hlt.auth.model.entity;


import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Column;
import org.springframework.data.relational.core.mapping.Table;

import java.time.Instant;
import java.util.UUID;

@Table("refresh_tokens")
public class RefreshToken {

    @Id
    private UUID id;

    @Column("account_id")
    private UUID accountId;

    private String token;

    @Column("expires_at")
    private Instant expiresAt;

    @Column("is_revoked")
    private boolean isRevoked;

    public RefreshToken() {
    }

    public RefreshToken(UUID id, UUID accountId, String token, Instant expiresAt, boolean isRevoked) {
        this.id = id;
        this.accountId = accountId;
        this.token = token;
        this.expiresAt = expiresAt;
        this.isRevoked = isRevoked;
    }

    public UUID getId() {
        return id;
    }

    public void setId(UUID id) {
        this.id = id;
    }

    public UUID getAccountId() {
        return accountId;
    }

    public void setAccountId(UUID accountId) {
        this.accountId = accountId;
    }

    public String getToken() {
        return token;
    }

    public void setToken(String token) {
        this.token = token;
    }

    public Instant getExpiresAt() {
        return expiresAt;
    }

    public void setExpiresAt(Instant expiresAt) {
        this.expiresAt = expiresAt;
    }

    public boolean isRevoked() {
        return isRevoked;
    }

    public void setRevoked(boolean revoked) {
        isRevoked = revoked;
    }

}
