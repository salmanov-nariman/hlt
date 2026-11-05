package domain

import "time"

type ChatSummary struct {
	ID          string
	LastMessage *Message
	UnreadCount int
	UpdatedAt   time.Time
	Revision    int
}
