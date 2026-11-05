package repository

import (
	"chat/internal/domain"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

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

func (r *ChatRepository) CreateDirectChat(ctx context.Context, userID1, userID2 string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var chatID string

	if err := tx.QueryRow(ctx, `INSERT INTO chats (revision) VALUES (1) RETURNING id`).Scan(&chatID); err != nil {
		return "", fmt.Errorf("failed to insert chat: %w", err)
	}

	queryMembers := `
		INSERT INTO chat_members (chat_id, user_id, last_read_sequence)
		VALUES ($1, $2, 0), ($1, $3, 0)
	`

	_, err = tx.Exec(ctx, queryMembers, chatID, userID1, userID2)
	if err != nil {
		return "", fmt.Errorf("failed to insert chat members: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return chatID, nil
}

func (r *ChatRepository) GetDirectChat(ctx context.Context, userID1, userID2 string) (*string, error) {
	query := `
		SELECT cm1.chat_id
		FROM chat_members cm1
		JOIN chat_members cm2 ON cm1.chat_id = cm2.chat_id
		WHERE cm1.user_id = $1 AND cm2.user_id = $2
		LIMIT 1;
	`

	var chatID string
	err := r.db.QueryRow(ctx, query, userID1, userID2).Scan(&chatID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to check direct chat: %w", err)
	}
	return &chatID, nil
}

func (r *ChatRepository) GetChatByID(ctx context.Context, chatID, userID string) (domain.ChatSummary, error) {
	query := `
		SELECT 
			c.id AS chat_id,
			c.updated_at AS chat_updated_at,
			c.revision AS chat_revision,
			c.last_sequence AS chat_last_sequence,
			cm.last_read_sequence AS user_last_read_sequence,
			m.id AS msg_id,
			m.sender_id AS msg_sender_id,
			m.content AS msg_content,
			m.created_at AS msg_created_at,
			m.client_message_id AS msg_client_message_id,
			m.sequence AS msg_sequence
		FROM chats c
		JOIN chat_members cm ON c.id = cm.chat_id
		LEFT JOIN LATERAL (
			SELECT id, sender_id, content, created_at, client_message_id, sequence
			FROM messages
			WHERE chat_id = c.id
			ORDER BY sequence DESC
			LIMIT 1
		) m ON true
		WHERE c.id = $1 AND cm.user_id = $2
	`

	var s domain.ChatSummary
	var chatLastSeq, userLastReadSeq int64
	var msgID, msgSenderID, msgClientMessageID, msgContent *string
	var msgCreatedAt *time.Time
	var msgSequence *int64

	err := r.db.QueryRow(ctx, query, chatID, userID).Scan(
		&s.ID,
		&s.UpdatedAt,
		&s.Revision,
		&chatLastSeq,
		&userLastReadSeq,
		&msgID,
		&msgSenderID,
		&msgContent,
		&msgCreatedAt,
		&msgClientMessageID,
		&msgSequence,
	)

	if err != nil {
		return domain.ChatSummary{}, fmt.Errorf("failed to get chat by id: %w", err)
	}

	if chatLastSeq > userLastReadSeq {
		s.UnreadCount = int(chatLastSeq - userLastReadSeq)
	} else {
		s.UnreadCount = 0
	}

	if msgID != nil {
		s.LastMessage = &domain.Message{
			ID:              *msgID,
			ChatID:          s.ID,
			SenderID:        *msgSenderID,
			ClientMessageID: *msgClientMessageID,
			Content:         *msgContent,
			CreatedAt:       *msgCreatedAt,
			Sequence:        *msgSequence,
		}
	}

	return s, nil
}

func (r *ChatRepository) updateLastReadSequence(ctx context.Context, db DBEngine, chatID, userID string, sequence int64) error {
	query := `
		UPDATE chat_members
		SET last_read_sequence = GREATEST(last_read_sequence, $1)
		WHERE chat_id = $2 AND user_id = $3
	`

	_, err := db.Exec(ctx, query, sequence, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to update last_read_sequence: %w", err)
	}

	return nil
}

func (r *ChatRepository) UpdateReadStatus(ctx context.Context, chatID, userID string, sequence int64) error {
	return r.updateLastReadSequence(ctx, r.db, chatID, userID, sequence)
}

func (r *ChatRepository) GetUserChats(ctx context.Context, userID string, limit int, cursorUpdatedAt *time.Time, cursorID *string) ([]domain.ChatSummary, error) {
	query := `
		SELECT 
			c.id AS chat_id,
			c.updated_at AS chat_updated_at,
			m.id AS msg_id,
			m.sender_id AS msg_sender_id,
			m.content AS msg_content,
			m.created_at AS msg_created_at,
			m.client_message_id AS msg_client_message_id,
			m.sequence AS msg_sequence,
			cm.last_read_sequence AS user_last_read_sequence
		FROM chats c
		JOIN chat_members cm ON c.id = cm.chat_id
		LEFT JOIN LATERAL (
			SELECT id, sender_id, content, created_at, client_message_id, sequence
			FROM messages
			WHERE chat_id = c.id
			ORDER BY created_at DESC
			LIMIT 1
		) m ON true
		WHERE cm.user_id = $1
			AND ($2::timestamptz IS NULL OR (c.updated_at, c.id) < ($2::timestamptz, $3::uuid))
		ORDER BY c.updated_at DESC, c.id DESC
		LIMIT $4;
	`

	rows, err := r.db.Query(ctx, query, userID, cursorUpdatedAt, cursorID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query user chats: %w", err)
	}
	defer rows.Close()

	var summaries []domain.ChatSummary
	for rows.Next() {
		var s domain.ChatSummary
		var chatUpdatedAt time.Time
		var msgID, msgSenderID, msgContent, msgClientMessageID *string
		var msgSequence *int64
		var msgCreatedAt *time.Time
		var userLastSequence int64

		err = rows.Scan(
			&s.ID,
			&chatUpdatedAt,
			&msgID, // Если в БД NULL, сюда запишется nil
			&msgSenderID,
			&msgContent,
			&msgCreatedAt,
			&msgClientMessageID,
			&msgSequence,
			&userLastSequence,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan chat summary: %w", err)
		}

		if msgID != nil {
			s.LastMessage = &domain.Message{
				ID:              *msgID,
				ChatID:          s.ID,
				SenderID:        *msgSenderID,
				Content:         *msgContent,
				CreatedAt:       *msgCreatedAt,
				Sequence:        *msgSequence,
				ClientMessageID: *msgClientMessageID,
			}
			s.UpdatedAt = chatUpdatedAt

			if *msgSequence > userLastSequence {
				s.UnreadCount = int(*msgSequence - userLastSequence)
			} else {
				s.UnreadCount = 0
			}
		} else {
			s.UpdatedAt = chatUpdatedAt
			s.UnreadCount = 0
		}

		summaries = append(summaries, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return summaries, nil
}
