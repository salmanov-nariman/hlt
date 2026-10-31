package service_test

import (
	"chat/internal/domain"
	"chat/internal/websocket"
	"context"
)


type mockWSBroadcaster struct {
	broadcastEventFunc func(event websocket.Event)
}

func (m *mockWSBroadcaster) BroadcastEvent(event websocket.Event) {
	if m.broadcastEventFunc != nil {
		m.broadcastEventFunc(event)
	}
}

type mockChatRepository struct {
	getUsersInChatFunc func(ctx context.Context, chatID string) ([]string, error)
	saveMessageFunc func(ctx context.Context, msg *domain.Message) error
	createChatFunc func(ctx context.Context, invitedIDs []string) (string, error)
	getMessagesAfterFunc func(ctx context.Context, chatID string, afterSequence int64, limit int) ([]domain.Message, error)
}

func (m *mockChatRepository) GetUsersInChat(ctx context.Context, chatID string) ([]string, error) {
	return m.getUsersInChatFunc(ctx, chatID)
}

func (m *mockChatRepository) SaveMessage(ctx context.Context, msg *domain.Message) error {
	return m.saveMessageFunc(ctx, msg)
}

func (m *mockChatRepository) CreateChat(ctx context.Context, invitedIDs []string) (string, error) {
	return m.createChatFunc(ctx, invitedIDs)
}

func (m* mockChatRepository) GetMessagesAfter(ctx context.Context, chatID string, afterSequence int64, limit int) ([]domain.Message, error) {
	return m.getMessagesAfterFunc(ctx, chatID, afterSequence, limit)
}