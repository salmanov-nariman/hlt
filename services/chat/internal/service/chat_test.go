package service_test

import (
	"chat/internal/service"
	"context"
	"slices"
	"testing"
)


func TestChatService_CreateChat_Success(t *testing.T) {
	ctx := context.Background()
	expectedRoomID := "new_chat_2"
	
	senderID := "user_1"
	invitedIDs := []string{"user_2", "user_2", "user_3"} 

	mockRepo := &mockChatRepository{
		createChatFunc: func(ctx context.Context, membersIDs []string) (string, error) {
			if len(membersIDs) != 3 {
				t.Errorf("ожидалось 3 уникальных участника, получено: %d", len(membersIDs))
			}
			
			expectedUsers := []string{"user_1", "user_2", "user_3"}
			for _, expectedUser := range expectedUsers {
				if !slices.Contains(membersIDs, expectedUser) {
					t.Errorf("участник %s потерян при сборке списка", expectedUser)
				}
			}

			return expectedRoomID, nil
		},
	}

	svc := service.NewChatService(nil, mockRepo)

	chatID, err := svc.CreateChat(ctx, senderID, invitedIDs)

	if err != nil {
		t.Fatalf("ожидался успех, но получена ошибка: %v", err)
	}
	if chatID != expectedRoomID {
		t.Errorf("ожидался ID комнаты %s, получено: %s", expectedRoomID, chatID)
	}
}

