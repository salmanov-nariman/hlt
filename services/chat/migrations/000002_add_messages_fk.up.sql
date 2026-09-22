ALTER TABLE messages
    ADD CONSTRAINT fk_messages_chat
        FOREIGN KEY (duo_id) REFERENCES chats(id) ON DELETE CASCADE;