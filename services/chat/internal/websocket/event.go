package websocket

import (
	"encoding/json"
)

const (
	EventTypeMessageCreated = "message.created"
	EventTypeChatUpdated    = "chat.updated"
	EventTypeChatRead       = "chat.read"
)

type Event struct {
	Type          string          `json:"type"`
	Payload       json.RawMessage `json:"payload"`
	ChatID        string          `json:"chat_id"`
	TargetUserIDs []string        `json:"-"`
}

