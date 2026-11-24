TRUNCATE TABLE messages;

UPDATE chats
SET last_sequence = 0;

UPDATE chat_members
SET last_read_sequence = 0;

ALTER TABLE messages
    DROP COLUMN IF EXISTS cipher_text,
    ADD COLUMN content TEXT NOT NULL;

DROP TABLE IF EXISTS user_public_keys;