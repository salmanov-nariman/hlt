package dto

// отправляется клиентам для обновления списка чатов
type ChatUpdatedEvent struct {
	ChatID      string          `json:"chat_id"`
	LastMessage MessageResponse `json:"last_message"`
}