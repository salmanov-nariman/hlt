package service

import (
	"context"

	"chat/internal/websocket"
	"chat/internal/domain"
)

type WSBroadcaster interface {
	BroadcastEvent(event websocket.Event)
}

type ChatRepository interface {
	GetUsersInChat(ctx context.Context, chatID string) ([]string, error)
	CreateChat(ctx context.Context, membersIDs []string) (string, error)
	SaveMessage(ctx context.Context, msg *domain.Message) error
	GetMessagesAfter(ctx context.Context, chatID string, afterSequence int64, limit int) ([]domain.Message, error)
}

type ChatService struct {
	wsManager      WSBroadcaster
	chatRepository ChatRepository
}

func NewChatService(wsManager WSBroadcaster, repo ChatRepository) *ChatService {
	return &ChatService{
		wsManager:      wsManager,
		chatRepository: repo,
	}
}
