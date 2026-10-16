package service

import (
	"chat/internal/repository"
	"chat/internal/websocket"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

type ChatService struct {
	wsManager      *websocket.Manager
	chatRepository repository.ChatRepository
}

func NewChatService(wsManager *websocket.Manager, repo repository.ChatRepository) *ChatService {
	return &ChatService{
		wsManager:      wsManager,
		chatRepository: repo,
	}
}

func (s *ChatService) ProcessMessage(
    ctx context.Context,
    senderID string,
    roomID string,
    text string,
) error {

    targetUserIDs, err := s.chatRepository.GetUsersInChat(ctx, roomID)
    if err != nil {
        return fmt.Errorf("failed to get chat members: %w", err)
    }

    isMember := false

    for _, uid := range targetUserIDs {
        if uid == senderID {
            isMember = true
            break
        }
    }

    if !isMember {
        return fmt.Errorf("sender is not a member of this chat")
    }

    if strings.TrimSpace(text) == "" {
        return fmt.Errorf("message cannot be empty")
    }

    err = s.chatRepository.SaveMessage(
        ctx,
        roomID,
        senderID,
        text,
    )
    if err != nil {
        return fmt.Errorf("failed to save message: %w", err)
    }

    log.Printf(
        "[DB] Message saved successfully: sender=%s, chat=%s",
        senderID,
        roomID,
    )

    chatMsg := websocket.ChatMessage{
        Text: text,
    }

    payloadBytes, err := json.Marshal(chatMsg)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }

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
