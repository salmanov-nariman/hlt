package domain

import "time"

type Message struct {
	ID              string
	ChatID          string
	SenderID        string
	ClientMessageID string
	CipherText      []byte
	Sequence        int64
	CreatedAt       time.Time
}
