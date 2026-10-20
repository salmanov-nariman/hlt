package domain

import "time"


type Message struct {
	ID string
	RoomID string
	SenderID string
	Text string
	CreatedAT time.Time
} 
