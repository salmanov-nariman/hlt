ALTER TABLE chat_members DROP COLUMN last_read_sequence;
ALTER TABLE chats DROP COLUMN revision;
ALTER TABLE chats DROP COLUMN last_sequence;
DROP INDEX idx_messages_chat_client_msg;
ALTER TABLE messages DROP COLUMN client_message_id;
ALTER TABLE messages DROP COLUMN sequence;

ALTER TABLE messages RENAME COLUMN chat_id TO duo_id;