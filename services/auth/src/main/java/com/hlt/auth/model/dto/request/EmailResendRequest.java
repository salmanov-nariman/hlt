package com.hlt.auth.model.dto.request;

import org.jspecify.annotations.NonNull;

public record EmailResendRequest(
        @NonNull String email
) {
}
