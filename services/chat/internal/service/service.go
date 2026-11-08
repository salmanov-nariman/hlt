package service

import (
	"context"
	"time"

	"chat/internal/domain"
	"chat/internal/websocket"
)

type WSBroadcaster interface {
	BroadcastEvent(ctx context.Context, event websocket.Event)
}

type ChatRepository interface {
	GetUsersInChat(ctx context.Context, chatID string) ([]string, error)
	CreateDirectChat(ctx context.Context, userID1, userID2 string) (string, error)
	SaveMessageWithReadUpdate(ctx context.Context, msg *domain.Message) error
	GetMessagesAfter(ctx context.Context, chatID string, afterSequence int64, limit int) ([]domain.Message, error)
	UpdateReadStatus(ctx context.Context, chatID, userID string, sequence int64) error
	GetUserChats(ctx context.Context, userID string, limit int, cursorUpdatedAt *time.Time, cursorID *string) ([]domain.ChatSummary, error)
	GetDirectChat(ctx context.Context, userID1, userID2 string) (*string, error)
	GetChatByID(ctx context.Context, chatID, userID string) (domain.ChatSummary, error)
	GetChatMessages(ctx context.Context, chatID string, cursor *int64, limit int) ([]domain.Message, error)
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
