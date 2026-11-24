CREATE INDEX IF NOT EXISTS idx_chats_pagination
ON chats (updated_at DESC, id DESC);