package cmd

import (
	"context"
	"encoding/json"
	"log"
	"notifier/config"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

type ChatEvent struct {
	Type          string   `json:"type"`
	TargetUserIDs []string `json:"target_user_ids"`
}

func main() {
	ctx := context.Background()
	cfg := config.Load()

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect redis: %v", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: strings.Split(cfg.KafkaBrokers, ","),
		Topic:   cfg.KafkaTopic,
		GroupID: "push-notification-group",
	})
	defer reader.Close()

	log.Println("[Notifier] Сервис запущен, слушаем Кафку...")

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			continue
		}

		var event ChatEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			continue
		}

		if event.Type == "message.created" {
			processPushNotificaation(ctx, rdb, event)
		}
	}
}

func processPushNotificaation(ctx context.Context, rdb *redis.Client, event ChatEvent) {
	for _, targetID := range event.TargetUserIDs {
		_, err := rdb.Get(ctx, "online"+targetID).Result()

		if err == redis.Nil {
			// тут будет firebase
			log.Printf("Notification send to offline user: %s", targetID)
		} else if err != nil {
			log.Printf("Failed Redis for %s: %v", targetID, err)
		} else {
			log.Printf("User %s is online", targetID)
		}
	}
}
