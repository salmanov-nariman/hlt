package service_test

import (
	"chat/internal/domain"
	"context"
	"time"
)

type mockEventPublisher struct {
	publishToTopicFunc func(ctx context.Context, topic string, event interface{}) error
}

func (m *mockEventPublisher) PublishToTopic(ctx context.Context, topic string, event interface{}) error {
	if m.publishToTopicFunc(ctx, topic, event) != nil {
		return m.publishToTopicFunc(ctx, topic, event)
	}
	return nil
}

type mockChatRepository struct {
	getUsersInChatFunc func(ctx context.Context, chatID string) ([]string, error)
	saveMessageFunc func(ctx context.Context, msg *domain.Message) error
	createDirectChatFunc func(ctx context.Context, uid1, uid2 string) (string, error)
	getMessagesAfterFunc func(ctx context.Context, chatID string, afterSequence int64, limit int) ([]domain.Message, error)
	updateReadStatusFunc func(ctx context.Context, chatID, userID string, sequence int64) error
	getUserChatsFunc func(ctx context.Context, userID string, limit int, cursorUpdatedAt *time.Time, cursorID *string) ([]domain.ChatSummary, error)
	getDirectChatFunc func(ctx context.Context, uid1, uid2 string) (*string, error)
	getChatByIDFunc func(ctx context.Context, chatID, userID string) (domain.ChatSummary, error)
	getChatMessagesFunc func(ctx context.Context, chatID string, cursor *int64, limit int) ([]domain.Message, error)
	saveKeyFunc func(ctx context.Context, userID string, publicKey []byte) error
	getLatestKeyFunc func(ctx context.Context, userID string) (domain.UserPublicKey, error)
}

func (m *mockChatRepository) GetUsersInChat(ctx context.Context, chatID string) ([]string, error) {
	return m.getUsersInChatFunc(ctx, chatID)
}

func (m *mockChatRepository) SaveMessageWithReadUpdate(ctx context.Context, msg *domain.Message) error {
	return m.saveMessageFunc(ctx, msg)
}

func (m *mockChatRepository) CreateDirectChat(ctx context.Context, uid1, uid2 string) (string, error) {
	return m.createDirectChatFunc(ctx, uid1, uid2)
}

func (m* mockChatRepository) GetMessagesAfter(ctx context.Context, chatID string, afterSequence int64, limit int) ([]domain.Message, error) {
	return m.getMessagesAfterFunc(ctx, chatID, afterSequence, limit)
}

func (m *mockChatRepository) UpdateReadStatus(ctx context.Context, chatID, userID string, sequence int64) error {
    return m.updateReadStatusFunc(ctx, chatID, userID, sequence)
}

func (m *mockChatRepository) GetUserChats(ctx context.Context, userID string, limit int, cursorUpdatedAt *time.Time, cursorID *string) ([]domain.ChatSummary, error) {
	return m.getUserChatsFunc(ctx, userID, limit, cursorUpdatedAt, cursorID)
}

func (m *mockChatRepository) GetDirectChat(ctx context.Context, uid1, uid2 string) (*string, error) {
	return m.getDirectChatFunc(ctx, uid1, uid2)
}

func (m *mockChatRepository) GetChatByID(ctx context.Context, chatID, userID string) (domain.ChatSummary, error) {
	return m.getChatByIDFunc(ctx, chatID, userID)
}

func (m *mockChatRepository) GetChatMessages(ctx context.Context, chatID string, cursor *int64, limit int) ([]domain.Message, error) {
	return m.getChatMessagesFunc(ctx, chatID, cursor, limit)
}

func (m *mockChatRepository) SaveKey(ctx context.Context, userID string, publicKey []byte) error {
	return m.saveKeyFunc(ctx, userID, publicKey)
}

func (m *mockChatRepository) GetLatestKey(ctx context.Context, userID string) (domain.UserPublicKey, error) {
	return m.getLatestKeyFunc(ctx, userID)
}