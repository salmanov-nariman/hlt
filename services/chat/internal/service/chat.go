package service

import (
	"chat/internal/domain"
	"chat/internal/repository"
	"chat/internal/websocket"
	"context"
	"encoding/json"
	"errors"
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

func (s *ChatService) ProcessMessage(ctx context.Context, senderID, chatID, clientMessageID, text string) (*domain.Message, error) {
	targetUserIDs, err := s.chatRepository.GetUsersInChat(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	
	if !slices.Contains(targetUserIDs, senderID) {
		return nil, fmt.Errorf("user %s is not member of room %s", senderID, chatID)
	}

	msg := &domain.Message{
		ChatID: chatID,
		SenderID: senderID,
		ClientMessageID: clientMessageID,
		Text: text,
	}

	err = s.chatRepository.SaveMessage(ctx, msg)
	if err != nil {
		if errors.Is(err, repository.ErrMessageAlreadyExists) || errors.Is(err, repository.ErrMessageConflict) {
			return msg, err
		}
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	payloadBytes, _ := json.Marshal(msg)
	s.wsManager.BroadcastEvent(websocket.Event{
		Type:          websocket.EventTypeMessageCreated,
		ChatID:        chatID,
		Payload:       payloadBytes,
		TargetUserIDs: targetUserIDs,
	})

	return msg, nil
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
