package service

import (
	"context"
	"time"

	"chat/internal/domain"

	"github.com/redis/go-redis/v9"
)

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
	SaveKey(ctx context.Context, userID string, publicKey []byte) error
	GetLatestKey(ctx context.Context, userID string) (domain.UserPublicKey, error)
}

type Producer interface {
	PublishToTopic(ctx context.Context, topic string, event any) error
}

type ChatService struct {
	chatRepository ChatRepository
	kafkaProducer  Producer
	rdb            *redis.Client
}

func NewChatService(producer Producer, repo ChatRepository, rdb *redis.Client) *ChatService {
	return &ChatService{
		kafkaProducer:  producer,
		chatRepository: repo,
		rdb:            rdb,
	}
}
