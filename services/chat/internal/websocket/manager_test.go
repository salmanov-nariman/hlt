package websocket

import (
	"chat/middleware"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

func setupTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("не удалось запустить miniredis: %v", err)
	}
	
	t.Cleanup(mr.Close)

	return redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
}

func TestManager_ServeWS_ConnectAndDisconnect(t *testing.T) {
	rdb := setupTestRedis(t)
	m := NewManager(rdb)

	// оборачиваем ServeWS в auth мидлвейр
	handler := middleware.AuthMiddleware(http.HandlerFunc(m.ServeWS)) 

	// поднимаем сервер в тесте 
	server := httptest.NewServer(handler)
	defer server.Close()

	// http:// -> ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// инициализация клиента
	dialer := websocket.DefaultDialer
	
	headers := make(http.Header)
	headers.Add("X-User-Id", "user_1")

	conn, _, err := dialer.Dial(wsURL, headers) // иммитируем подключение к вс
	if err != nil {
		t.Fatalf("не удалось подключиться к вебсокету: %v", err)
	} 

	time.Sleep(50 * time.Millisecond)

	m.RLock()
	if len(m.clients) != 1 {
		t.Errorf("Ожидался 1 клиент в мапе, получено: %d", len(m.clients))
	}
	m.RUnlock()

	conn.Close()

	time.Sleep(50 * time.Millisecond)

	m.RLock()
	if len(m.clients) != 0 {
		t.Errorf("Ожидалось 0 клиентов после отключения, получено: %d", len(m.clients))
	}
	m.RUnlock()
}

func TestManager_BroadcastEvent(t *testing.T) {
	rdb := setupTestRedis(t)
	m := NewManager(rdb)

	handler := middleware.AuthMiddleware(http.HandlerFunc(m.ServeWS))
	server := httptest.NewServer(handler)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// подключаем двух пользователей
	conn1, _, _ := websocket.DefaultDialer.Dial(wsURL, http.Header{"X-User-Id": []string{"user_1"}})
	defer conn1.Close()

	conn2, _, _ := websocket.DefaultDialer.Dial(wsURL, http.Header{"X-User-Id": []string{"user_2"}})
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)

	event := Event{
		Type: EventTypeMessageCreated,
		TargetUserIDs: []string{"user_1"},
	}

	m.BroadcastEvent(context.Background(), event)

	// проверяем, что user_1 получил сообщение
	conn1.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, msg, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("user_1 должен был получить сообщение, но произошла ошибка: %v", err)
	}
	if len(msg) == 0 {
		t.Error("user_1 получил пустое сообщение")
	}

	// проверяем, что user_2 ничего не получил
	conn2.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = conn2.ReadMessage()
	if err == nil {
		t.Error("user_2 не должен был получить сообщение, но получил его")
	}
}

func TestManager_ServeWS_MissingUserID(t *testing.T) {
	rdb := setupTestRedis(t)
	m := NewManager(rdb)
	handler := middleware.AuthMiddleware(http.HandlerFunc(m.ServeWS))
	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

	if err == nil {
		t.Fatal("Ожидалась ошибка при подключении без X-User-Id, но подключение прошло успешно")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Ожидался статус 401, получен: %d", resp.StatusCode)
	}
}