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

	s.wsManager.BroadcastEvent(websocket.Event{
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