package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ChatRepository interface {
	GetUsersInChat(ctx context.Context, chatID string) ([]string, error)
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
		return nil, fmt.Errorf("error of executing query: %s", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("error of scanning id: %s", err)
		}
		userIDs = append(userIDs, uid)
	}

	return userIDs, nil
}
