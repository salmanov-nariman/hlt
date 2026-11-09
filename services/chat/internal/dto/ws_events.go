package dto

// ChatUpdatedEvent отправляется клиентам для обновления списка чатов (Inbox)
type ChatUpdatedEvent struct {
	ChatID      string          `json:"chat_id"`
	LastMessage MessageResponse `json:"last_message"`
}