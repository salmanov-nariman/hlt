package service

import (
	"chat/internal/domain"
	"chat/internal/websocket"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

var (
	ErrInvalidPartner = errors.New("invalid partner: cannot create chat with yourself")
	ErrUserNotFound   = errors.New("user not found")
)

func (s* ChatService) CreateOrGetDirectChat(ctx context.Context, senderID, partnerID string) (domain.ChatSummary, bool, error) {
	if senderID == partnerID {
		return domain.ChatSummary{}, false, ErrInvalidPartner
	}

	existingChatID, err := s.chatRepository.GetDirectChat(ctx, senderID, partnerID)
	if err != nil {
		return domain.ChatSummary{}, false, fmt.Errorf("failed to check existing chat: %w", err)
	}

	var chatID string
	var isNew bool
	
	if existingChatID != nil {
		chatID = *existingChatID
		isNew = false
	} else {
		chatID, err = s.chatRepository.CreateDirectChat(ctx, senderID, partnerID)
		if err != nil {
			//имитация ошибки отсутствия user, пока не прикрутил user service
			if strings.Contains(err.Error(), "foreign key constraint") {
				return domain.ChatSummary{}, false, ErrUserNotFound
			}
			return domain.ChatSummary{}, false, fmt.Errorf("failed to create direct chat: %w", err)
		}
		isNew = true
	}

	chat, err := s.chatRepository.GetChatByID(ctx, chatID, senderID)
	if err != nil {
		return domain.ChatSummary{}, false, fmt.Errorf("failed to get created chat: %w", err)
	}

	// TODO: Обогатить данные профилем партнера через UserProvider

	return chat, isNew, nil
}

func (s *ChatService) UpdateReadStatus(ctx context.Context, chatID, userID string, sequence int64) error {
	membersIDs, err := s.chatRepository.GetUsersInChat(ctx, chatID)
	if err != nil {
		return fmt.Errorf("failed to get users in chat: %w", err)
	}

	if !slices.Contains(membersIDs, userID) {
		return fmt.Errorf("user %s is not member of chat %s", userID, chatID)
	}

	err = s.chatRepository.UpdateReadStatus(ctx, chatID, userID, sequence)
	if err != nil {
		return fmt.Errorf("failed to update read status in db: %w", err)
	}

	payload := map[string]any{
		"chat_id": chatID,
		"user_id": userID,
		"last_read_sequence": sequence,
	}

	payloadBytes, err := json.Marshal(payload)

	s.wsManager.BroadcastEvent(ctx, websocket.Event{
		Type: websocket.EventTypeChatRead,
		ChatID: chatID,
		TargetUserIDs: membersIDs,
		Payload: payloadBytes,
	})

	return nil
}


var ErrInvalidCursor = errors.New("invalid cursor")

func (s *ChatService) GetUserChats(ctx context.Context, userID string, limit int, cursor string) ([]domain.ChatSummary, *string, error) {
	var cursorUpdatedAt *time.Time
	var cursorID *string
	var err error

	if cursor != "" {
		cursorUpdatedAt, cursorID, err = decodeCursor(cursor)
		if err != nil {
			return nil, nil, err
		} 
	}

	chats, err := s.chatRepository.GetUserChats(ctx, userID, limit, cursorUpdatedAt, cursorID)
	if err != nil {
		return nil, nil, fmt.Errorf("chat service - failed to get user chats: %w", err)
	}

	var nextCursor *string
	if len(chats) == limit && len(chats) > 0 {
		lastChat := chats[len(chats)-1]

		encoded := encodeCursor(lastChat.UpdatedAt, lastChat.ID)
		nextCursor = &encoded
	}

	return chats, nextCursor, nil
}

func encodeCursor(t time.Time, id string) string {
	cursorStr := fmt.Sprintf("%s|%s", t.Format(time.RFC3339Nano), id)
	return base64.RawURLEncoding.EncodeToString([]byte(cursorStr))
}

func decodeCursor(cursor string) (*time.Time, *string, error) {
	decodedBytes, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, nil, ErrInvalidCursor
	}

	parts := strings.Split(string(decodedBytes), "|")
	if len(parts) != 2 {
		return nil, nil, ErrInvalidCursor
	}

	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, nil, ErrInvalidCursor
	}

	return &t, &parts[1], nil 
}

func (s *ChatService) GetChat(ctx context.Context, chatID, userID string) (domain.ChatSummary, error) {
	//сначала проверяем состоит ли юзер в чате
	userIDs, err := s.chatRepository.GetUsersInChat(ctx, chatID)
	if err != nil {
		return domain.ChatSummary{}, fmt.Errorf("failed to get users in chat: %w", err)
	}
	if !slices.Contains(userIDs, userID) {
		return domain.ChatSummary{}, domain.ErrUserNotInChat
	}

	//достаем объект чата
	chat, err := s.chatRepository.GetChatByID(ctx, chatID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrChatNotFound) {
			return domain.ChatSummary{}, domain.ErrChatNotFound
		}
		return domain.ChatSummary{}, fmt.Errorf("failed to get chat from repo: %w", err)
	}

	return chat, nil
}