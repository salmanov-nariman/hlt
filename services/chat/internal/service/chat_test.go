package service_test

import (
	"chat/internal/domain"
	"chat/internal/service"
	"context"
	"errors"
	"testing"
)


func TestChatService_CreateOrGetDirectChat(t *testing.T) {
	ctx := context.Background()
	senderID := "user_1"

	t.Run("ошибка при попытке создать чат с самим собой", func(t *testing.T) {
		svc := service.NewChatService(nil, nil)

		_, _, err := svc.CreateOrGetDirectChat(ctx, senderID, senderID)

		if !errors.Is(err, service.ErrInvalidPartner) {
			t.Errorf("ожидалась ошибка %v, получена: %v", service.ErrInvalidPartner, err)
		}
	})

	t.Run("успешный возврат существующего чата", func(t *testing.T) {
		partnerID := "user_2"
		expectedRoomID := "existing_chat_1"

		mockRepo := &mockChatRepository{
			getDirectChatFunc: func(ctx context.Context, u1, u2 string) (*string, error) {
				return &expectedRoomID, nil // Имитируем, что чат уже найден
			},
			getChatByIDFunc: func(ctx context.Context, chatID, userID string) (domain.ChatSummary, error) {
				return domain.ChatSummary{ID: chatID}, nil
			},
		}

		svc := service.NewChatService(nil, mockRepo)

		chat, isNew, err := svc.CreateOrGetDirectChat(ctx, senderID, partnerID)

		if err != nil {
			t.Fatalf("ожидался успех, но получена ошибка: %v", err)
		}
		if isNew {
			t.Errorf("ожидалось isNew = false, получено true")
		}
		if chat.ID != expectedRoomID {
			t.Errorf("ожидался ID чата %s, получено: %s", expectedRoomID, chat.ID)
		}
	})

	t.Run("успешное создание нового чата", func(t *testing.T) {
		partnerID := "user_3"
		expectedRoomID := "new_chat_2"

		mockRepo := &mockChatRepository{
			getDirectChatFunc: func(ctx context.Context, u1, u2 string) (*string, error) {
				return nil, nil 
			},
			createDirectChatFunc: func(ctx context.Context, u1, u2 string) (string, error) {
				if u1 != senderID || u2 != partnerID {
					t.Errorf("ожидались участники %s и %s, получено: %s, %s", senderID, partnerID, u1, u2)
				}
				return expectedRoomID, nil
			},
			getChatByIDFunc: func(ctx context.Context, chatID, userID string) (domain.ChatSummary, error) {
				return domain.ChatSummary{ID: chatID}, nil
			},
		}

		svc := service.NewChatService(nil, mockRepo)

		chat, isNew, err := svc.CreateOrGetDirectChat(ctx, senderID, partnerID)

		if err != nil {
			t.Fatalf("ожидался успех, но получена ошибка: %v", err)
		}
		if !isNew {
			t.Errorf("ожидалось isNew = true, получено false")
		}
		if chat.ID != expectedRoomID {
			t.Errorf("ожидался ID чата %s, получено: %s", expectedRoomID, chat.ID)
		}
	})
}

