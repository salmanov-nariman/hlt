package repository

import (
	"context"
	"database/sql"
)

type Message struct {
	ID        string
	DuoID     string
	SenderID  string
	Content   string
	IsSummary bool
}

type ChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) SaveMessage(ctx context.Context, msg Message) error {
	query := `
        INSERT INTO messages (id, duo_id, sender_id, content, is_summary)
        VALUES ($1, $2, $3, $4, $5)
    `
	_, err := r.db.ExecContext(ctx, query, msg.ID, msg.DuoID, msg.SenderID, msg.Content, msg.IsSummary)
	return err
}
