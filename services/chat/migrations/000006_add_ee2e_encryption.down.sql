TRUNCATE TABLE messages;

ALTER TABLE messages
    DROP COLUMN IF EXISTS ciphertext,
    ADD COLUMN content TEXT NOT NULL;

DROP TABLE IF EXISTS user_public_keys;