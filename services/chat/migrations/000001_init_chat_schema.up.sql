CREATE TABLE chats (
                       id UUID PRIMARY KEY,
                       is_active BOOLEAN DEFAULT TRUE,
                       created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                       updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE chat_members (
                              chat_id UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
                              user_id UUID NOT NULL,
                              joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

                              PRIMARY KEY (chat_id, user_id)
);

CREATE INDEX idx_chat_members_user ON chat_members(user_id);


CREATE TABLE messages (
                          id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                          duo_id UUID NOT NULL,
                          sender_id UUID NOT NULL,
                          content TEXT NOT NULL,
                          is_summary BOOLEAN DEFAULT FALSE,
                          created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_duo_history ON messages(duo_id, created_at DESC);