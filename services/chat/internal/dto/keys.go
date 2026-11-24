package dto

import "time"

type SaveKeyRequest struct {
	PublicKey []byte `json:"public_key"`
}

type UserPublicKeyResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	PublicKey []byte    `json:"public_key"`
	UpdatedAt time.Time `json:"updated_at"`
}
