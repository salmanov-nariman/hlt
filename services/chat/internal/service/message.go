package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"

	"chat/internal/domain"
	"chat/internal/dto"
	"chat/internal/websocket"

	"github.com/redis/go-redis/v9"
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

	go func(m *domain.Message, mIDs []string, sID string) {
		bgCtx := context.WithoutCancel(ctx)

		var onlineUserIDs []string
        var offlineUserIDs []string

		for _, memberID := range mIDs {
			if memberID == senderID {
				continue
			}

			_, err := s.rdb.Get(bgCtx, "online:"+memberID).Result()
			if err == redis.Nil {
                offlineUserIDs = append(offlineUserIDs, memberID)
            } else if err == nil {
                onlineUserIDs = append(onlineUserIDs, memberID)
            } else {
                log.Printf("[ChatSvc] Redis error for user %s: %v", memberID, err)
                offlineUserIDs = append(offlineUserIDs, memberID)
            }
		}

		s.publishEvents(bgCtx, m, onlineUserIDs, offlineUserIDs)

	}(msg, memberIDs, senderID)

	return msg, nil
}

func (s *ChatService) publishEvents(ctx context.Context, m *domain.Message, onlineIDs, offlineIDs []string) {
	msgDTO := dto.NewMessageResponse(m)
	payloadBytes, _ := json.Marshal(msgDTO)

	if len(onlineIDs) > 0 {
		err := s.kafkaProducer.PublishToTopic(ctx, "websocket-events", websocket.Event{
			Type: websocket.EventTypeMessageCreated,
			ChatID: m.ChatID,
			Payload: payloadBytes,
			TargetUserIDs: onlineIDs,
		})
		if err != nil {
			log.Printf("[ChatService] Error publishing MessageCreated: %v", err)
		}

		chatUpdatedPayload := dto.ChatUpdatedEvent{
			ChatID: m.ChatID,
			LastMessage: *msgDTO,
		}
		updatedPayloadBytes, _ := json.Marshal(chatUpdatedPayload)
		
		err = s.kafkaProducer.PublishToTopic(ctx, "websocket-events", websocket.Event{
			Type: websocket.EventTypeChatUpdated,
			ChatID: m.ChatID,
			Payload: updatedPayloadBytes,
			TargetUserIDs: onlineIDs,
		})
		if err != nil {
			log.Printf("[ChatService] Error publishing ChatUpdated: %v", err)
		}
	}

	if len(offlineIDs) > 0 {
		err := s.kafkaProducer.PublishToTopic(ctx, "push-events", websocket.Event{
			Type: "push-notification",
			ChatID: m.ChatID,
			Payload: payloadBytes,
			TargetUserIDs: offlineIDs,
		})
		if err != nil {
			log.Printf("[ChatService] Error publishing push notification: %v", err)
		}
	}
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
