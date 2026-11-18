package fcm

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)


type Client struct {
	fcmClient *messaging.Client
}

type PushSender interface {
	SendPush(ctx context.Context, deviceToken, title, body string) error
}

func NewClient(credentialsPath string) (*Client, error) {
	ctx := context.Background()

	opt := option.WithAuthCredentialsFile(option.ServiceAccount, credentialsPath)

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка инициализации fcm клиента: %w", err)
	}

	return &Client{fcmClient: client}, nil
}

func (c *Client) SendPush(ctx context.Context, deviceToken, title, body string) error {
	msg := &messaging.Message{
		Token: deviceToken,
		Notification: &messaging.Notification{
			Title: title,
			Body: body,
		},

		Data: map[string]string{
			"type": "new_chat_message",
		},
	}

	response, err := c.fcmClient.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("ошибка отправки fcm сообщения: %w", err)
	}

	log.Printf("Пуш успешно отправлен! MessageID: %s", response)
	return nil
}