package service

import (
	"chat/internal/grpc_client/worker"
	"chat/internal/websocket"
	"context"
	"encoding/json"
	"fmt"
	"log"
)

type ChatService struct {
	wsManager    *websocket.Manager
	workerClient worker.WorkerClient
}

func NewChatService(wsManager *websocket.Manager, workerClient worker.WorkerClient) *ChatService {
	return &ChatService{
		wsManager:    wsManager,
		workerClient: workerClient,
	}
}

func (s *ChatService) ProcessMessage(ctx context.Context, senderID, roomID, text string) error {
	log.Printf("[DB] Сохраняем сообщение от %s в БД для чата %s...", senderID, roomID)
	log.Printf("[gRPC] Отправляем запрос GetDuoMembers(duo_id: %s) в Worker сервис...", roomID)

	targetUserIDs, err := s.workerClient.GetDuoMembers(ctx, roomID, senderID)
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
