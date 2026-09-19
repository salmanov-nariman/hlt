package websocket

import (
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
	clients ClientList
	sync.RWMutex
	broadcast chan []byte
}

func NewManager() *Manager {
	manager := &Manager{
		clients:   make(ClientList),
		broadcast: make(chan []byte),
	}

	go manager.routeMessage()

	return manager
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	log.Println("new connection")

	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(conn, m)
	m.addClient(client)

	go client.readMessage()
	go client.writeMessage()
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	m.clients[client] = true
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.clients[client]; ok {
		delete(m.clients, client)
	}
}

func (m *Manager) routeMessage() {
	for {
		select {
		case payload := <-m.broadcast:
			m.RLock()
			for client := range m.clients {
				client.egress <- payload
			}
			m.RUnlock()
		}
	}
}
