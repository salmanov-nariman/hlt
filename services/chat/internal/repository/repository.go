package repository

import (
	"chat/internal/domain"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
    ErrMessageConflict      = errors.New("client_message_id conflict with different text")
    ErrMessageAlreadyExists = errors.New("message already exists")
)

type ChatRepository struct {
	db *pgxpool.Pool
}

func NewChatRepository(db *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) SaveMessage(ctx context.Context, msg *domain.Message) error {
	var existingID, existingText string
	var existingCreatedAt time.Time
	var existingSequence int64

	checkQuery := `
		SELECT id, content, created_at, sequence
		FROM messages
		WHERE chat_id=$1 AND client_message_id=$2 AND sender_id=$3
	`
	err := r.db.QueryRow(ctx, checkQuery, msg.ChatID, msg.ClientMessageID, msg.SenderID).Scan(&existingID, &existingText, &existingCreatedAt, &existingSequence)

	if err == nil {
		if existingText != msg.Text {
			return ErrMessageConflict // 409
		}

		msg.ID = existingID
		msg.CreatedAT = existingCreatedAt
		msg.Sequence = existingSequence
		return ErrMessageAlreadyExists // 200 OK
	}

	query := `
		WITH updated_chat AS (
			UPDATE chats
			SET last_sequence = last_sequence + 1,
                revision = revision + 1,
                updated_at = CURRENT_TIMESTAMP
            WHERE id = $1
            RETURNING last_sequence
        )
		INSERT INTO messages (chat_id, sender_id, client_message_id, content, sequence)
		SELECT $1, $2, $3, $4, updated_chat.last_sequence
		FROM updated_chat
		RETURNING id, created_at, sequence
	`
	
	err = r.db.QueryRow(ctx, query, msg.ChatID, msg.SenderID, msg.ClientMessageID, msg.Text).Scan(&msg.ID, &msg.CreatedAT, &msg.Sequence)
	if err != nil {
		return fmt.Errorf("failed to insert message and update chat: %w", err)
	}

	return nil
}

func (r *ChatRepository) GetUsersInChat(ctx context.Context, chatID string) ([]string, error) {
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

func (r *ChatRepository) CreateChat(ctx context.Context, membersIDs []string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var chatID string

	if err := tx.QueryRow(ctx, `INSERT INTO chats DEFAULT VALUES RETURNING id`).Scan(&chatID); err != nil {
		return "", fmt.Errorf("failed to insert chat: %w", err)
	}

	queryUsers := `INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2)`
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
