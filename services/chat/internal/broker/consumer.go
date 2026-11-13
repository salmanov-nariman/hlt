package broker

import (
	"chat/internal/websocket"
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader    *kafka.Reader
	wsManager *websocket.Manager
}

func NewConsumer(brokers []string, topic string, wsManager *websocket.Manager) *Consumer {
	groupID := "ws-nodes-" + uuid.NewString()

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,
	})
	return &Consumer{
		reader:    r,
		wsManager: wsManager,
	}
}

func (c *Consumer) Start(ctx context.Context) {
	log.Println("[Kafka Consumer] Subscribed to websocket events...")
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("[Kafka Consumer] Stopping (context canceled)")
				return
			}
			log.Printf("[Kafka Consumer] Error reading from Kafka: %v", err)
			continue
		}

		var event websocket.Event
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("[Kafka Consumer] Error parsing event: %v", err)
			continue
		}

		c.wsManager.BroadcastEvent(ctx, event)
	}
}

func (c *Consumer) Close() {
	c.reader.Close()
}