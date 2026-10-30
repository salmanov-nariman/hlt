package service_test

import (
	"chat/internal/domain"
	"chat/internal/service"
	"chat/internal/websocket"
	"errors"
	"strings"

	"context"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
)


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
	content := "Привет всем!"

	mockRepo := &mockChatRepository{
		getUsersInChatFunc: func(ctx context.Context, requestedRoomID string) ([]string, error) {
			return []string{"good_user_1", "good_user_2"}, nil
		},

		saveMessageFunc: func(ctx context.Context, msg *domain.Message) error {
			if msg.ChatID != chatID || msg.Content != content || msg.SenderID != senderID {
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

			if chatMsg.Content != content {
				t.Errorf("ожидался текст '%s', получено: '%s'", content, chatMsg.Content)
			}
			if chatMsg.ClientMessageID != clientMsgID {
				t.Errorf("ожидался ClientMessageID '%s', получено: '%s'", clientMsgID, chatMsg.ClientMessageID)
			}
		},
	}

	svc := service.NewChatService(mockWS, mockRepo)

	msg, err := svc.ProcessMessage(ctx, senderID, chatID, clientMsgID, content)

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


func TestChatService_SyncMessages_Forbidden(t *testing.T) {
	ctx := context.Background()
	chatID := "chat_1"
	userID := "hacker_1"

	mockRepo := &mockChatRepository{
		getUsersInChatFunc: func(ctx context.Context, requestedChatID string) ([]string, error) {
			return []string{"user_1", "user_2"}, nil
		},
		getMessagesAfterFunc: func(ctx context.Context, reqChatID string, reqAfterSeq int64, reqLimit int) ([]domain.Message, error) {
			t.Fatal("GetMessagesAfter не должен вызываться, если нет доступа")
			return nil, nil
		},
	}

	svc := service.NewChatService(nil, mockRepo)

	messages, err := svc.SyncMessages(ctx, userID, chatID, 0, 50)

	if err == nil {
		t.Fatal("ожидалась ошибка доступа, но метод вернул nil")
	}

	expectedErrorText := fmt.Sprintf("user %s is not member of chat %s", userID, chatID)
	if err.Error() != expectedErrorText {
		t.Errorf("ожидался текст ошибки '%s', получено: '%v'", expectedErrorText, err)
	}

	if messages != nil {
		t.Errorf("ожидалось nil сообщений, получено: %+v", messages)
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