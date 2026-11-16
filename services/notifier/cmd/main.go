package cmd

import (
	"context"
	"encoding/json"
	"log"
	"notifier/config"
	"notifier/internal/clients/user"
	"notifier/internal/fcm"
	"strings"

	"github.com/segmentio/kafka-go"
)

type ChatEvent struct {
	Type          string   `json:"type"`
	TargetUserIDs []string `json:"target_user_ids"`
}

func main() {
	ctx := context.Background()
	cfg := config.Load()

	fcmClient, err := fcm.NewClient("serviceAccountKey.json")
	if err != nil {
		log.Fatalf("[Notifier] Критическая ошибка инициализации FCM: %v", err)
	}

	userClient := user.NewMockClient()

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
			log.Printf("Ошибка чтения из Kafka: %v", err)
			continue
		}

		var event ChatEvent
		if json.Unmarshal(msg.Value, &event) == nil {
			processPushNotificaation(ctx, userClient, fcmClient, event)
		}
	}
}

func processPushNotificaation(
	ctx context.Context,
	tokenProvider user.TokenProvider,
	fcmClient *fcm.Client,
	event ChatEvent,
) {
	for _, targetID := range event.TargetUserIDs {

		deviceToken, err := tokenProvider.GetDeviceToken(ctx, targetID)

		if err != nil {
			if err == user.ErrTokenNotFound {
				log.Printf("[Notifier] У юзера %s нет токена (вышел из приложения)", targetID)
			} else {
					log.Printf("[Notifier] Ошибка получения токена для %s: %v", targetID, err)
			}
			continue
		}
		errPush := fcmClient.SendPush(ctx, deviceToken, "Новое сообщение", "Вам прислали сообщение в HLT")
		if errPush != nil {
			log.Printf("[Notifier] Ошибка отправки пуша юзеру %s: %v", targetID, errPush)
		}
	}
}
