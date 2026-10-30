package service_test

import (
	"chat/internal/domain"
	"chat/internal/service"
	"chat/internal/websocket"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)


type mockWSBroadcaster struct {
	broadcastEventFunc func(event websocket.Event)
}

func (m *mockWSBroadcaster) BroadcastEvent(event websocket.Event) {
	if m.broadcastEventFunc != nil {
		m.broadcastEventFunc(event)
	}
}

type mockChatRepository struct {
	getUsersInChatFunc func(ctx context.Context, chatID string) ([]string, error)
	saveMessageFunc func(ctx context.Context, msg *domain.Message) error
	createChatFunc func(ctx context.Context, invitedIDs []string) (string, error)
}

func (m *mockChatRepository) GetUsersInChat(ctx context.Context, chatID string) ([]string, error) {
	return m.getUsersInChatFunc(ctx, chatID)
}

func (m *mockChatRepository) SaveMessage(ctx context.Context, msg *domain.Message) error {
	return m.saveMessageFunc(ctx, msg)
}

func (m *mockChatRepository) CreateChat(ctx context.Context, invitedIDs []string) (string, error) {
	return m.createChatFunc(ctx, invitedIDs)
}


func TestChatService_ProcessMessage_UserNotMember(t *testing.T) {
	ctx := context.Background()
	chatID := "chat_1"
	senderID := "bad_user_1"
	clientMsgID := "client_uuid_1"

	mockRepo := &mockChatRepository{
		getUsersInChatFunc: func(ctx context.Context, requestedRoomID string) ([]string, error) {
			if requestedRoomID != chatID {
				t.Errorf("ожидался запрос для комнаты %s, получено: %s", chatID, requestedRoomID)
			}
			return []string{"good_user_1", "good_user_2"}, nil
		},

		saveMessageFunc: func(ctx context.Context, msg *domain.Message) error {
			t.Fatal("SaveMessage не должен вызываться, если пользователя нет в чате")
			return nil
		},
	}

	svc := service.NewChatService(nil, mockRepo)

	msg, err := svc.ProcessMessage(ctx, senderID, chatID, clientMsgID, "Пытаюсь что-то написать")

	if err == nil {
		t.Fatal("ожидалась ошибка доступа, но метод вернул nil")
	}
	if msg != nil {
		t.Errorf("ожидалось, что сообщение будет nil, получено: %+v", msg)
	}

	expectedErrorText := fmt.Sprintf("user %s is not member of room %s", senderID, chatID)
	if err.Error() != expectedErrorText {
		t.Errorf("ожидался текст ошибки '%s', получено: '%v'", expectedErrorText, err)
	}
}


func TestChatService_ProcessMessage_Success(t *testing.T) {
	ctx := context.Background()
	chatID := "chat_1"
	senderID := "good_user_1"
	clientMsgID := "client_uuid_1"
	text := "Привет всем!"

	mockRepo := &mockChatRepository{
		getUsersInChatFunc: func(ctx context.Context, requestedRoomID string) ([]string, error) {
			return []string{"good_user_1", "good_user_2"}, nil
		},

		saveMessageFunc: func(ctx context.Context, msg *domain.Message) error {
			if msg.ChatID != chatID || msg.Text != text || msg.SenderID != senderID {
				t.Errorf("неверные данные для сохранения: %+v", msg)
			}
			msg.ID = "msg_1"
			return nil
		},
	}

	wsCalled := false
	mockWS := &mockWSBroadcaster{
		broadcastEventFunc: func(event websocket.Event) {
			wsCalled = true
			if event.Type != websocket.EventTypeMessageCreated || event.ChatID != chatID {
				t.Errorf("неверный тип сообщения или chatID: %+v", event)
			}

			if len(event.TargetUserIDs) == 0 {
				t.Errorf("ошибка: пустой TargetUserIDs")
			}
			
			expectedUsers := []string{"good_user_1", "good_user_2"}
			if !slices.Equal(event.TargetUserIDs, expectedUsers) {
				t.Errorf("ожидались получатели %v, получено %v", expectedUsers, event.TargetUserIDs)
			}

			var chatMsg domain.Message
			if err := json.Unmarshal(event.Payload, &chatMsg); err != nil {
            	t.Errorf("ошибка распаковки Payload: %v", err)
        	}

			if chatMsg.Text != text {
				t.Errorf("ожидался текст '%s', получено: '%s'", text, chatMsg.Text)
			}
			if chatMsg.ClientMessageID != clientMsgID {
				t.Errorf("ожидался ClientMessageID '%s', получено: '%s'", clientMsgID, chatMsg.ClientMessageID)
			}
		},
	}

	svc := service.NewChatService(mockWS, mockRepo)

	msg, err := svc.ProcessMessage(ctx, senderID, chatID, clientMsgID, text)

	if err != nil {
		t.Fatalf("ожидался успех, но получена ошибка: %v", err)
	}
	if msg == nil {
		t.Fatal("ожидалось сообщение, но получен nil")
	}
	if msg.ID != "msg_1" {
		t.Errorf("ожидался ID 'msg_1', получено: '%s'", msg.ID)
	}

	if !wsCalled {
		t.Error("ожидалось, что сообщение будет отправлено в вебсокет, но BroadcastEvent не был вызван")
	}
}

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

func TestChatService_ProcessMessage_DBError(t *testing.T) {
	ctx := context.Background()

	mockRepo := &mockChatRepository{
		getUsersInChatFunc: func(ctx context.Context, chatID string) ([]string, error) {
			return nil, errors.New("db connections lost")
		},
	}

	svc := service.NewChatService(nil, mockRepo)

	msg, err := svc.ProcessMessage(ctx, "user_1", "chat_1", "client_uuid_1", "something test")

	if err == nil {
		t.Fatal("ожидалась ошибка базы данных, но метод вернул nil")
	}
	if msg != nil {
		t.Errorf("ожидалось, что сообщение будет nil, получено: %+v", msg)
	}

	expectedErrPrefix := "failed to get users"
	if !strings.HasPrefix(err.Error(), expectedErrPrefix) {
		t.Errorf("ожидалось, что ошибка начнется с '%s', получено: '%v'", expectedErrPrefix, err)
	}
}