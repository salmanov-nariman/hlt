package com.hlt.auth.model.data;

import com.hlt.auth.model.entity.Credentials;
import com.hlt.auth.model.entity.EmailVerification;

public record RegistrationData(
        Credentials credentials,
        EmailVerification emailVerification
) {
}
