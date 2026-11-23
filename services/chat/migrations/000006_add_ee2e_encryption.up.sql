CREATE TABLE IF NOT EXISTS user_public_keys (
	id         UUID                     PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID                     NOT NULL,
    public_key BYTEA                    NOT NULL CHECK (octet_length(public_key) = 32),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

	UNIQUE (user_id, public_key)
);

CREATE INDEX IF NOT EXISTS idx_user_public_keys_latest
    ON user_public_keys (user_id, updated_at DESC, id DESC);

TRUNCATE TABLE messages;

ALTER TABLE messages
	DROP COLUMN IF EXISTS content,
    ADD  COLUMN cipher_text BYTEA NOT NULL CHECK (octet_length(cipher_text) > 0);
    
