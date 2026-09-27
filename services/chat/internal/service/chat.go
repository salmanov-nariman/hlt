package service

import (
	"chat/internal/repository"
	"chat/internal/websocket"
	"context"
	"encoding/json"
	"fmt"
	"log"
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

func (s *ChatService) ProcessMessage(ctx context.Context, senderID, roomID string, text string) error {
	log.Printf("[DB] Сохраняем сообщение от %s в БД для чата %s...", senderID, roomID)
	log.Printf("[gRPC] Отправляем запрос GetDuoMembers(duo_id: %s) в Worker сервис...", roomID)

	targetUserIDs, err := s.chatRepository.GetUsersInChat(ctx, roomID)
	if err != nil {
		return fmt.Errorf("access denied or worker error: %w", err)
	}
	log.Printf("[gRPC] Ответ от Worker сервиса получен. Участники: %v", targetUserIDs)

	chatMsg := websocket.ChatMessage{Text: text}
	payloadBytes, _ := json.Marshal(chatMsg)

	log.Printf("[WebSocket] Передаем событие в менеджер для рассылки...")
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
