package dto

import "time"

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
