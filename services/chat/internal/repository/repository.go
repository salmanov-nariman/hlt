package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository interface {
	GetUsersInChat(ctx context.Context, chatID string) ([]string, error)
	CreateChat(ctx context.Context, membersIDs []string) (string, error)
}

type chatRepo struct {
	db *pgxpool.Pool
}

func NewChatRepository(db *pgxpool.Pool) ChatRepository {
	return &chatRepo{
		db: db,
	}
}

func (r *chatRepo) GetUsersInChat(ctx context.Context, chatID string) ([]string, error) {
	query := "SELECT user_id FROM chat_members WHERE chat_id = $1"

	rows, err := r.db.Query(ctx, query, chatID)
	if err != nil {
		return nil, fmt.Errorf("error of executing query: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("error of scanning id: %w", err)
		}
		userIDs = append(userIDs, uid)
	}

	return userIDs, nil
}

func (r *chatRepo) CreateChat(ctx context.Context, membersIDs []string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var chatID string

	if err := tx.QueryRow(ctx, `INSERT INTO chats DEFAULT VALUES RETURNING id`).Scan(&chatID); err != nil {
		return "", fmt.Errorf("failed to insert chat: %w", err)
	}

	queryUsers := `INSERT INTO chats_members (chat_id, user_id) VALUES ($1, $2)`
	for _, uid := range membersIDs {
		_, err := tx.Exec(ctx, queryUsers, chatID, uid)
		if err != nil {
			return "", fmt.Errorf("failed to add user %s to chat: %w", uid, err)
		}

	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return chatID, nil
}
