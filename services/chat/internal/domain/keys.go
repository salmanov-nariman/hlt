package domain

import "time"

type UserPublicKey struct {
	ID        string
	UserID    string
	PublicKey []byte
	UpdatedAt time.Time
}
