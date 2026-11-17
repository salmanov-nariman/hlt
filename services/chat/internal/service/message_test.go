package service_test

import (
	"chat/internal/domain"
	"chat/internal/dto"
	"chat/internal/service"
	"chat/internal/websocket"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)


func TestChatService_ProcessMessage_UserNotMember(t *testing.T) {
	ctx := context.Background()
	chatID := "chat_1"
	senderID := "bad_user_1"
	clientMsgID := "client_uuid_1"

	rdb, mr := setupTestRedis(t)
	defer mr.Close()

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

	svc := service.NewChatService(nil, mockRepo, rdb)

	msg, err := svc.ProcessMessage(ctx, senderID, chatID, clientMsgID, "Пытаюсь что-то написать")

	if err == nil {
		t.Fatal("ожидалась ошибка доступа, но метод вернул nil")
	}
	if msg != nil {
		t.Errorf("ожидалось, что сообщение будет nil, получено: %+v", msg)
	}

	expectedErrorText := fmt.Sprintf("user %s is not member of chat %s", senderID, chatID)
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

	rdb, mr := setupTestRedis(t)
	defer mr.Close()

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
		updateReadStatusFunc: func(ctx context.Context, reqChatID, reqUserID string, reqSeq int64) error {
            return nil
        },
	}

	mr.Set("online:good_user_2", "1")

	eventChan := make(chan interface{}, 2)

	mockPublisher := &mockEventPublisher{
		publishToTopicFunc: func(ctx context.Context, topic string, event interface{}) error {
			if topic == "websocket-events" {
				eventChan <- event
			}
			return nil
		},
	}

	svc := service.NewChatService(mockPublisher, mockRepo, rdb)

	msg, err := svc.ProcessMessage(ctx, senderID, chatID, clientMsgID, content)

	if err != nil {
		t.Fatalf("ожидался успех, но получена ошибка: %v", err)
	}
	if msg == nil {
		t.Fatal("ожидалось сообщение, но получен nil")
	}

	var capturedEvents []interface{}
	for range 2 {
		select {
		case event := <-eventChan:
			capturedEvents = append(capturedEvents, event)
		case <-time.After(1 * time.Second):
			t.Fatalf("Таймаут: ожидалось событие вебсокета, но горутина его не прислала")
		}
	}

	expectedUsers := []string{"good_user_2"}

	event1 := capturedEvents[0].(websocket.Event) // message.created
    if event1.Type != websocket.EventTypeMessageCreated || event1.ChatID != chatID {
        t.Errorf("ошибка в первом событии (message.created): %+v", event1)
    }

    if !slices.Equal(event1.TargetUserIDs, expectedUsers) {
        t.Errorf("ожидались получатели %v для event1, получено %v", expectedUsers, event1.TargetUserIDs)
    }

	var chatMsg dto.MessageResponse
	if err := json.Unmarshal(event1.Payload, &chatMsg); err != nil {
        t.Errorf("ошибка распаковки Payload: %v", err)
    }

	if chatMsg.Content != content {
		t.Errorf("ожидался текст '%s', получено: '%s'", content, chatMsg.Content)
	}

	if chatMsg.ClientMessageID != clientMsgID {
		t.Errorf("ожидался ClientMessageID '%s', получено: '%s'", clientMsgID, chatMsg.ClientMessageID)
	}

	event2 := capturedEvents[1].(websocket.Event) // chat.updated
    if event2.Type != websocket.EventTypeChatUpdated || event2.ChatID != chatID {
        t.Errorf("ошибка во втором событии (chat.updated): %+v", event2)
    }
    if !slices.Equal(event2.TargetUserIDs, expectedUsers) {
        t.Errorf("ожидались получатели %v для event2, получено %v", expectedUsers, event2.TargetUserIDs)
    }
}


func TestChatService_SyncMessages_Forbidden(t *testing.T) {
	ctx := context.Background()
	chatID := "chat_1"
	userID := "hacker_1"

	rdb, mr := setupTestRedis(t)
	defer mr.Close()

	mockRepo := &mockChatRepository{
		getUsersInChatFunc: func(ctx context.Context, requestedChatID string) ([]string, error) {
			return []string{"user_1", "user_2"}, nil
		},
		getMessagesAfterFunc: func(ctx context.Context, reqChatID string, reqAfterSeq int64, reqLimit int) ([]domain.Message, error) {
			t.Fatal("GetMessagesAfter не должен вызываться, если нет доступа")
			return nil, nil
		},
	}

	svc := service.NewChatService(nil, mockRepo, rdb)

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

	rdb, mr := setupTestRedis(t)
	defer mr.Close()

	mockRepo := &mockChatRepository{
		getUsersInChatFunc: func(ctx context.Context, chatID string) ([]string, error) {
			return nil, errors.New("db connections lost")
		},
	}

	svc := service.NewChatService(nil, mockRepo, rdb)

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

func TestChatService_UpdateReadStatus_Success(t *testing.T) {
    ctx := context.Background()
    chatID := "chat_1"
    userID := "good_user_1"
    sequence := int64(15)

	rdb, mr := setupTestRedis(t)
	defer mr.Close()

    mockRepo := &mockChatRepository{
        getUsersInChatFunc: func(ctx context.Context, requestedRoomID string) ([]string, error) {
            return []string{"good_user_1", "good_user_2"}, nil
        },

        updateReadStatusFunc: func(ctx context.Context, reqChatID, reqUserID string, reqSeq int64) error {
            if reqChatID != chatID || reqUserID != userID || reqSeq != sequence {
                t.Errorf("неверные данные для обновления: chatID=%s, userID=%s, seq=%d", reqChatID, reqUserID, reqSeq)
            }
            return nil
        },
    }

	eventChan := make(chan interface{}, 1)
	mockPublisher := &mockEventPublisher{
		publishToTopicFunc: func(ctx context.Context, topic string, event interface{}) error {
			eventChan <- event
			return nil
		},
	}

    svc := service.NewChatService(mockPublisher, mockRepo, rdb)

    err := svc.UpdateReadStatus(ctx, chatID, userID, sequence)

    if err != nil {
        t.Fatalf("ожидался успех, но получена ошибка: %v", err)
    }

	select {
	case capturedEvent := <-eventChan:
		event := capturedEvent.(websocket.Event)
		if event.ChatID != chatID {
			t.Errorf("неверный chatID в событии: %s", event.ChatID)
		}
		expectedUsers := []string{"good_user_1", "good_user_2"}
		if !slices.Equal(event.TargetUserIDs, expectedUsers) {
			t.Errorf("ожидались получатели %v, получено %v", expectedUsers, event.TargetUserIDs)
		}
	case <-time.After(1 * time.Second):
		t.Error("Таймаут: статус прочтения не был отправлен в брокер")
	}
}

func setupTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("не удалось запустить миниредис: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return rdb, mr
}