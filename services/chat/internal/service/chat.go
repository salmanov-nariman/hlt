package service

import (
	"chat/internal/websocket"
	"encoding/json"
	"log"
)

type ChatService struct {
	wsManager *websocket.Manager
	// Позже сюда добавятся репозитории для БД и gRPC-клиенты
}

func NewChatService(wsManager *websocket.Manager) *ChatService {
	return &ChatService{wsManager: wsManager}
}

func (s *ChatService) ProcessMessage(senderID, roomID, text string) error {
	log.Printf("[DB] Сохраняем сообщение от %s в БД для чата %s...", senderID, roomID)
	log.Printf("[gRPC] Отправляем запрос GetDuoMembers(duo_id: %s) в Worker сервис...", roomID)

	targetUserIDs := []string{senderID, "a48e7c82-4b26-4d90-bda7-8b85f3f25abf"}
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
