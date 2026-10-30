ALTER TABLE messages RENAME COLUMN duo_id TO chat_id;

ALTER TABLE messages 
ADD COLUMN client_message_id UUID NOT NULL DEFAULT gen_random_uuid(),
ADD COLUMN sequence BIGINT NOT NULL DEFAULT 1;

CREATE UNIQUE INDEX idx_messages_chat_client_msg ON messages(chat_id, client_message_id);

ALTER TABLE chats 
ADD COLUMN revision BIGINT NOT NULL DEFAULT 1,
ADD COLUMN last_sequence BIGINT NOT NULL DEFAULT 0;

ALTER TABLE chat_members
ADD COLUMN last_read_sequence BIGINT NOT NULL DEFAULT 0;