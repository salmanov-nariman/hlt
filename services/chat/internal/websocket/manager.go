package websocket

import (
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
	}
)

type Manager struct {
	clients map[string]*Client
	sync.RWMutex
	broadcast chan Event
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

	userID := r.Header.Get("X-User-Id")
	if userID == "" {
		http.Error(w, "X-User-Id header is missing", http.StatusBadRequest)
		return
	}

	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(conn, m, userID)
	m.addClient(client)

	go client.readMessage()
	go client.writeMessage()
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client.userID] = client
	log.Printf("Client added. UserID: %s, Total connections: %d\n", client.userID, len(m.clients))
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client.userID]; ok {
		if err := client.connection.Close(); err != nil {
			log.Println(err)
		}
		delete(m.clients, client.userID)
		log.Printf("Client removed. UserID: %s\n", client.userID)
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

func (m *Manager) BroadcastEvent(event Event) {
	m.broadcast <- event
}
