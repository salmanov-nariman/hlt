package repository

import (
	"chat/internal/domain"
	"context"
	"fmt"
	"time"
)


func (r *ChatRepository) SaveMessageWithReadUpdate(ctx context.Context, msg *domain.Message) error {
	var existingID, existingContent string
	var existingCreatedAt time.Time
	var existingSequence int64

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	checkQuery := `
		SELECT id, content, created_at, sequence
		FROM messages
		WHERE chat_id=$1 AND client_message_id=$2 AND sender_id=$3
	`
	err = tx.QueryRow(ctx, checkQuery, msg.ChatID, msg.ClientMessageID, msg.SenderID).Scan(&existingID, &existingContent, &existingCreatedAt, &existingSequence)

	if err == nil {
		if existingContent != msg.Content {
			return domain.ErrMessageConflict // 409
		}

		msg.ID = existingID
		msg.CreatedAt = existingCreatedAt
		msg.Sequence = existingSequence
		return domain.ErrMessageAlreadyExists // 200 OK
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
	
	err = tx.QueryRow(ctx, query, msg.ChatID, msg.SenderID, msg.ClientMessageID, msg.Content).Scan(&msg.ID, &msg.CreatedAt, &msg.Sequence)
	if err != nil {
		return fmt.Errorf("failed to insert message and update chat: %w", err)
	}

	err = r.updateLastReadSequence(ctx, tx, msg.ChatID, msg.SenderID, msg.Sequence)
	if err != nil {
		return fmt.Errorf("failed to update last_read_sequence in chat_members: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *ChatRepository) GetMessagesAfter(ctx context.Context, chatID string, afterSequence int64, limit int) ([]domain.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, client_message_id, content, sequence, created_at
		FROM messages
		WHERE chat_id=$1 AND sequence>$2
		ORDER BY sequence ASC
		LIMIT $3
	`

	rows, err := r.db.Query(ctx, query, chatID, afterSequence, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	messages := make([]domain.Message, 0)

	for rows.Next() {
		var msg domain.Message
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID,
			&msg.SenderID,
			&msg.ClientMessageID,
			&msg.Content,
			&msg.Sequence,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message rows: %w", err)
	}

	return messages, nil
}

func (r *ChatRepository) GetChatMessages(ctx context.Context, chatID string, cursor *int64, limit int) ([]domain.Message, error) {
	query := `
		SELECT 
			id, chat_id, sender_id, content, created_at, client_message_id, sequence
		FROM messages
		WHERE chat_id = $1 AND ($2::bigint IS NULL OR sequence < $2)
		ORDER BY sequence DESC
		LIMIT $3;
	`

	rows, err := r.db.Query(ctx, query, chatID, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var msg domain.Message
		err := rows.Scan(
			&msg.ID,
			&msg.ChatID, 
			&msg.SenderID, 
			&msg.Content, 
			&msg.CreatedAt, 
			&msg.ClientMessageID, 
			&msg.Sequence,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	
	return messages, nil
}