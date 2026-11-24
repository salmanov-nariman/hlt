package repository

import (
	"chat/internal/domain"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)


func (r *ChatRepository) SaveKey(ctx context.Context, userID string, publicKey []byte) error {
	query := `
		INSERT INTO user_public_keys (user_id, public_key)
		VALUES ($1, $2)
		ON CONFLICT (user_id, public_key) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, userID, publicKey)
	if err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	return nil
}

func (r *ChatRepository) GetLatestKey(ctx context.Context, userID string) (domain.UserPublicKey, error) {
	query := `
		SELECT id, user_id, public_key, updated_at
		FROM user_public_keys
		WHERE user_id = $1
		ORDER BY updated_at DESC, id DESC
		LIMIT 1;
	`

	var key domain.UserPublicKey
	err := r.db.QueryRow(ctx, query, userID).Scan(&key.ID, &key.UserID, &key.PublicKey, &key.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.UserPublicKey{}, domain.ErrPublicKeyNotFound
		}
		return domain.UserPublicKey{}, fmt.Errorf("failed to get public key: %w", err)
	}

	return key, nil
}