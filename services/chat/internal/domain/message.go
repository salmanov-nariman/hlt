package domain

import "time"


type Message struct {
	ID string
	ChatID string
	SenderID string
	ClientMessageID string
	Text string
	Sequence int64
	CreatedAT time.Time
} 
