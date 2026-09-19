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
	rooms map[string]ClientList
	sync.RWMutex
	broadcast chan Event
}

func NewManager() *Manager {
	manager := &Manager{
		rooms:     make(map[string]ClientList),
		broadcast: make(chan Event),
	}

	go manager.routeMessage()

	return manager
}

func (m *Manager) ServeWS(w http.ResponseWriter, r *http.Request) {
	log.Println("new connection")

	roomID := r.URL.Query().Get("room")
	if roomID == "" {
		http.Error(w, "room parameter is missing", http.StatusBadRequest)
		return
	}

	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := NewClient(conn, m, roomID)
	m.addClient(client)

	go client.readMessage()
	go client.writeMessage()
}

func (m *Manager) addClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if m.rooms[client.roomID] == nil {
		m.rooms[client.roomID] = make(ClientList)
	}
	m.rooms[client.roomID][client] = true
}

func (m *Manager) removeClient(client *Client) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.rooms[client.roomID][client]; ok {
		if err := client.connection.Close(); err != nil {
			log.Println(err)
		}
		delete(m.rooms[client.roomID], client)

		if len(m.rooms[client.roomID]) == 0 {
			delete(m.rooms, client.roomID)
		}
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
				if roomClients, ok := m.rooms[event.RoomID]; ok {
					for client := range roomClients {
						client.egress <- data
					}
				}

				m.RUnlock()
			default:
				log.Printf("unknown event type: %v", event.Type)
			}
		}
	}
}
