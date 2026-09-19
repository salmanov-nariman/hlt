package websocket

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type ClientList map[*Client]bool
type Client struct {
	connection *websocket.Conn
	manager    *Manager
	egress     chan []byte
}

func NewClient(conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		connection: conn,
		manager:    manager,
		egress:     make(chan []byte),
	}
}

func (c *Client) readMessage() {
	defer func() {
		c.manager.removeClient(c)
	}()

	for {
		messageType, payload, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}
		var event Event
		if err := json.Unmarshal(payload, &event); err != nil {
			log.Printf("received invalid JSON: %v", err)
			continue
		}

		c.manager.broadcast <- event

		log.Println(messageType)
		log.Println(string(payload))
	}
}

func (c *Client) writeMessage() {
	defer func() {
		c.manager.removeClient(c)
	}()

	for {
		select {
		case message, ok := <-c.egress:
			if !ok {
				if err := c.connection.WriteMessage(websocket.CloseMessage, nil); err != nil {
					log.Printf("connection closed: %v", err)

				}
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("failed to send message: %v", err)
			}
			log.Println("message sent")
		}
	}
}
