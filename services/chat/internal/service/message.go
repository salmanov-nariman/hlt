package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"chat/internal/domain"
	"chat/internal/repository"
	"chat/internal/websocket"
)

func (s *ChatService) ProcessMessage(ctx context.Context, senderID, chatID, clientMessageID, content string) (*domain.Message, error) {
	memberIDs, err := s.chatRepository.GetUsersInChat(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	
	if !slices.Contains(memberIDs, senderID) {
		return nil, fmt.Errorf("user %s is not member of room %s", senderID, chatID)
	}

	msg := &domain.Message{
		ChatID: chatID,
		SenderID: senderID,
		ClientMessageID: clientMessageID,
		Content: content,
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
		TargetUserIDs: memberIDs,
	})

	return msg, nil
}


func (s *ChatService) SyncMessages(ctx context.Context, senderID, chatID string, afterSequence int64, limit int) ([]domain.Message, error) {
	memberIDs, err := s.chatRepository.GetUsersInChat(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	if !slices.Contains(memberIDs, senderID) {
		return nil, fmt.Errorf("user %s is not member of room %s", senderID, chatID)
	}

	if limit <= 0 || limit > 100 {
		limit = 50 
	}

	return s.chatRepository.GetMessagesAfter(ctx, chatID, afterSequence, limit)
}
