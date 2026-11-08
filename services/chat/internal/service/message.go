package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"chat/internal/domain"
	"chat/internal/dto"
	"chat/internal/websocket"
)

func (s *ChatService) ProcessMessage(ctx context.Context, senderID, chatID, clientMessageID, content string) (*domain.Message, error) {
	memberIDs, err := s.chatRepository.GetUsersInChat(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	if !slices.Contains(memberIDs, senderID) {
		return nil, fmt.Errorf("user %s is not member of chat %s", senderID, chatID)
	}

	msg := &domain.Message{
		ChatID:          chatID,
		SenderID:        senderID,
		ClientMessageID: clientMessageID,
		Content:         content,
	}

	err = s.chatRepository.SaveMessageWithReadUpdate(ctx, msg)
	if err != nil {
		if errors.Is(err, domain.ErrMessageAlreadyExists) || errors.Is(err, domain.ErrMessageConflict) {
			return msg, err
		}
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	go func(m *domain.Message, mIDs []string) {
		bgCtx := context.WithoutCancel(ctx)

		msgDTO := dto.NewMessageResponse(m)
		payloadBytes, _ := json.Marshal(msgDTO)

		s.wsManager.BroadcastEvent(bgCtx, websocket.Event{
			Type:          websocket.EventTypeMessageCreated,
			ChatID:        chatID,
			Payload:       payloadBytes,
			TargetUserIDs: memberIDs,
		})

		// обновляем статус события, чтобы можно было обновить список чатов
		chatUpdatedPayload := dto.ChatUpdatedEvent{
			ChatID:      m.ChatID,
			LastMessage: *msgDTO,
		}

		updatedPayloadBytes, _ := json.Marshal(chatUpdatedPayload)

		s.wsManager.BroadcastEvent(bgCtx, websocket.Event{
			Type:          websocket.EventTypeChatUpdated,
			ChatID:        chatID,
			Payload:       updatedPayloadBytes,
			TargetUserIDs: memberIDs,
		})
	}(msg, memberIDs)

	return msg, nil
}

func (s *ChatService) SyncMessages(ctx context.Context, senderID, chatID string, afterSequence int64, limit int) ([]domain.Message, error) {
	memberIDs, err := s.chatRepository.GetUsersInChat(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	if !slices.Contains(memberIDs, senderID) {
		return nil, fmt.Errorf("user %s is not member of chat %s", senderID, chatID)
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.chatRepository.GetMessagesAfter(ctx, chatID, afterSequence, limit)
}

func (s *ChatService) GetChatMessages(ctx context.Context, chatID, userID string, cursor *int64, limit int) ([]domain.Message, *int64, error) {
	_, err := s.chatRepository.GetUsersInChat(ctx, chatID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil, ErrUserNotFound
		}

		return nil, nil, fmt.Errorf("access check failed: %w", err)
	}

	messages, err := s.chatRepository.GetChatMessages(ctx, chatID, cursor, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get messages from repo: %w", err)
	}

	var nextCursor *int64
	if len(messages) == limit && len(messages) > 0 {
		lastMsgSequence := messages[len(messages)-1].Sequence
		nextCursor = &lastMsgSequence
	}

	return messages, nextCursor, nil
}
