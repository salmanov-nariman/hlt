package service

import (
	"chat/internal/domain"
	"chat/internal/websocket"
	"context"
	"encoding/json"
	"fmt"
	"slices"
)

type WSBroadcaster interface {
	BroadcastEvent(event websocket.Event)
}

type ChatRepository interface {
	GetUsersInChat(ctx context.Context, chatID string) ([]string, error)
	CreateChat(ctx context.Context, membersIDs []string) (string, error)
	SaveMessage(ctx context.Context, msg *domain.Message) error
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

func (s *ChatService) ProcessMessage(ctx context.Context, senderID, roomID string, text string) error {
	targetUserIDs, err := s.chatRepository.GetUsersInChat(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}
	
	if !slices.Contains(targetUserIDs, senderID) {
		return fmt.Errorf("user %s is not member of room %s", senderID, roomID)
	}

	msg := &domain.Message{
		RoomID: roomID,
		SenderID: senderID,
		Text: text,
	}

	if err := s.chatRepository.SaveMessage(ctx, msg); err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	chatMsg := websocket.ChatMessage{Text: text}

	payloadBytes, _ := json.Marshal(chatMsg)
	s.wsManager.BroadcastEvent(websocket.Event{
		Type:          "new_message",
		RoomID:        roomID,
		Payload:       payloadBytes,
		TargetUserIDs: targetUserIDs,
	})

	return nil
}

func (s *ChatService) CreateChat(ctx context.Context, senderID string, invitedIDs []string) (string, error) {
	uniqueUsers := make(map[string]struct{})

	uniqueUsers[senderID] = struct{}{}

	for _, userID := range invitedIDs {
		uniqueUsers[userID] = struct{}{}
	}

	finalMembersIDs := make([]string, 0, len(uniqueUsers))
	for uid := range uniqueUsers {
		finalMembersIDs = append(finalMembersIDs, uid)
	}

	chatID, err := s.chatRepository.CreateChat(ctx, finalMembersIDs)
	if err != nil {
		return "", fmt.Errorf("failed to create chat in db: %w", err)
	}

	return chatID, nil
}
