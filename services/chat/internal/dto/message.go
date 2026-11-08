package dto

import (
	"chat/internal/domain"
	"time"
)

type MessageResponse struct {
	ID              string    `json:"id"`
	ChatID          string    `json:"chat_id"`
	SenderID        string    `json:"sender_id"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"created_at"`
	ClientMessageID string    `json:"client_message_id"`
	Sequence        int64     `json:"sequence"`
}

type PostMessageRequest struct {
	ChatID          string `json:"chat_id"`
	ClientMessageID string `json:"client_message_id"`
	Text            string `json:"text"`
}

type PaginatedChatMessagesResponse struct {
	Messages []MessageResponse `json:"messages"`
	NextCursor *int64 `json:"next_cursor"`
}

func NewMessageResponse(msg *domain.Message) *MessageResponse {
	if msg == nil {
		return nil
	}
	return &MessageResponse{
		ID:              msg.ID,
		ChatID:          msg.ChatID,
		SenderID:        msg.SenderID,
		Content:         msg.Content,
		CreatedAt:       msg.CreatedAt,
		ClientMessageID: msg.ClientMessageID,
		Sequence:        msg.Sequence,
	}
}