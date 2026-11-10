package websocket

import (
	"chat/internal/httputil"
	"chat/middleware"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	websocketUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type Manager struct {
	clients map[string]*Client // мапа клиентов, которые в онлайне
	sync.RWMutex
	broadcast chan Event // почтовый ящик, который связывает бизнес-логику с миром вебсокетов
}

func NewManager() *Manager {
	manager := &Manager{
		clients:   make(map[string]*Client),
		broadcast: make(chan Event),
	}

	go manager.routeMessage()

	return manager
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	log.Println("new connection")

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		httputil.JSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация")
		return
	}

	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(userID, conn, m)
	m.addClient(client)

	go client.readMessage()
	go client.writeMessage()
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client.ID] = client
	log.Printf("Client added. UserID: %s, Total connections: %d\n", client.ID, len(m.clients))
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client.ID]; ok {
		if err := client.connection.Close(); err != nil {
			log.Println(err)
		}
		delete(m.clients, client.ID)
		log.Printf("Client removed. UserID: %s\n", client.ID)
	}
}

func (m *Manager) routeMessage() {
	for event := range m.broadcast {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("error marshaling event to json: %v", err)
			continue
		}

		m.RLock()
		// Пробегаемся по списку получателей

		for _, targetID := range event.TargetUserIDs {
			// Если получатель сейчас в сети (есть в мапе)
			if client, ok := m.clients[targetID]; ok {
				client.egress <- data
			}
		}
		m.RUnlock()
	}
}

func (m *Manager) BroadcastEvent(ctx context.Context, event Event) {
	select {
	case m.broadcast <- event:
		// все хорошо, событие улетело в канал
	case <-ctx.Done():
		// завершаем работу горутины если контекст отменился (например сервак отключился)
	}

}
