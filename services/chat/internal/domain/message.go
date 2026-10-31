package domain

import "time"


type Message struct {
	ID string
	ChatID string
	SenderID string
	ClientMessageID string
	Content string
	Sequence int64
	CreatedAT time.Time
} 
