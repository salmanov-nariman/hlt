package com.hlt.auth.model.entity;

import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Column;
import org.springframework.data.relational.core.mapping.Table;

import java.time.Instant;
import java.util.UUID;

@Table("email_verifications")
public class EmailVerification {

    @Id
    private UUID id;

    @Column("account_id")
    private UUID accountId;

    @Column("code_hash")
    private String codeHash;

    @Column("expires_at")
    private Instant expiresAt;

    @Column("failed_attempts")
    private Integer failedAttempts;

    @Column("last_sent_at")
    private Instant lastSentAt;

    public EmailVerification() {
    }

    public EmailVerification(UUID id, UUID accountId, String codeHash, Instant expiresAt, Integer failedAttempts, Instant lastSentAt) {
        this.id = id;
        this.accountId = accountId;
        this.codeHash = codeHash;
        this.expiresAt = expiresAt;
        this.failedAttempts = failedAttempts;
        this.lastSentAt = lastSentAt;
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

    public String getCodeHash() {
        return codeHash;
    }

    public void setCodeHash(String codeHash) {
        this.codeHash = codeHash;
    }

    public Instant getExpiresAt() {
        return expiresAt;
    }

    public void setExpiresAt(Instant expiresAt) {
        this.expiresAt = expiresAt;
    }

    public Integer getFailedAttempts() {
        return failedAttempts;
    }

    public void setFailedAttempts(Integer failedAttempts) {
        this.failedAttempts = failedAttempts;
    }

    public Instant getLastSentAt() {
        return lastSentAt;
    }

    public void setLastSentAt(Instant lastSentAt) {
        this.lastSentAt = lastSentAt;
    }
}