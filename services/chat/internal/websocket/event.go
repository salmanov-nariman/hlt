package websocket

import "encoding/json"

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	RoomID  string          `json:"-"`
}

type ChatMessage struct {
	Text string `json:"text"`
}
