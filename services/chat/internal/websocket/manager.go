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
	clients ClientList
	sync.RWMutex
	broadcast chan Event
}

func NewManager() *Manager {
	manager := &Manager{
		clients:   make(ClientList),
		broadcast: make(chan Event),
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
		case event := <-m.broadcast:
			switch event.Type {
			case "send_message":
				var chtMsg ChatMessage
				if err := json.Unmarshal(event.Payload, &chtMsg); err != nil {
					log.Printf("error unmarshaling payload: %v", err)
					continue
				}
				if chtMsg.Text == "/summary" {
					log.Println("command intercepted: user requested summary!")
					// add summary logic here:
					continue
				}
				data, err := json.Marshal(event)
				if err != nil {
					log.Printf("error marshaling event to json: %v", err)
					continue
				}
				m.RLock()
				for client := range m.clients {
					client.egress <- data
				}
				m.RUnlock()
			default:
				log.Printf("unknown event type: %v", event.Type)
			}
		}
	}
}
