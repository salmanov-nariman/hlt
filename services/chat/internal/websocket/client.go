package websocket

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	pongWait   = 60 * time.Second  
	pingPeriod = (pongWait * 9) / 10
	writeWait  = 10 * time.Second
)

type Client struct {
	ID         string
	connection *websocket.Conn
	manager    *Manager
	egress     chan []byte
}

func NewClient(userID string, conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		ID:         userID,
		connection: conn,
		manager:    manager,
		egress:     make(chan []byte, 256),
	}
}

func (c *Client) readMessage() {
	defer func() {
		c.manager.removeClient(c)
	}()

	c.connection.SetReadDeadline(time.Now().Add(pongWait))

	c.connection.SetPongHandler(func(string) error {
		c.connection.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error reading message: %v", err)
			}
			break
		}
	}
}

func (c *Client) writeMessage() {
	ticker := time.NewTicker(pingPeriod) // он будет тикать каждые 54 секунды

	defer func() {
		ticker.Stop()
		c.manager.removeClient(c)
	}()

	for {
		select {
		case message, ok := <-c.egress:
			c.connection.SetWriteDeadline(time.Now().Add(writeWait)) // если долго отправляется сообщение, больше 10 сек, то обрываем соединение

			if !ok {
				c.connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.connection.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("failed to send message: %v", err)
				return // Выходим, сокет сломан

			}

		case <-ticker.C:
			c.connection.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
