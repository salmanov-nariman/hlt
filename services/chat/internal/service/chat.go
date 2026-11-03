package service

import (
	"chat/internal/domain"
	"chat/internal/websocket"
	"context"
	"encoding/json"
	"fmt"
	"slices"
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


func (s *ChatService) GetUserChats(ctx context.Context, userID string) ([]domain.ChatSummary, error) {
	chats, err := s.chatRepository.GetUserChats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("chat service - failed to get user chats: %w", err)
	}
	return chats, nil
}
