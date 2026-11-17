package com.hlt.auth.model.data;

public enum EmailVerificationResult {
    VERIFIED,
    ALREADY_VERIFIED,
    INVALID_CODE,
    EXPIRED,
    ATTEMPTS_EXHAUSTED,
    NOT_FOUND
}
