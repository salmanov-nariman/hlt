package dto

import "time"

type ChatSummaryResponse struct {
	ID          string           `json:"id"`
	LastMessage *MessageResponse `json:"last_message"`
	UnreadCount int              `json:"unread_count"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Revision    int              `json:"revision"`
}

type PaginatedChatsResponse struct {
	Items      []ChatSummaryResponse `json:"items"`
	NextCursor *string               `json:"next_cursor"`
}
