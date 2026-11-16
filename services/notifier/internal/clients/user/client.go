package user

import (
	"context"
	"errors"
	"time"
)

var ErrTokenNotFound = errors.New("device token not found")

type TokenProvider interface {
	GetDeviceToken(ctx context.Context, userID string) (string, error)
}

type MockClient struct {
	fakeTokens map[string]string //user_id: device_token
}

func NewMockClient() *MockClient {
	return &MockClient{
		fakeTokens: map[string]string{
			"user-1": "test_token_123",
			"user-2": "test_token_456",
		},
	}
}

func (m *MockClient) GetDeviceToken(ctx context.Context, userID string) (string, error) {
	time.Sleep(50 * time.Millisecond)

	token, exists := m.fakeTokens[userID]
	if !exists {
		return "", ErrTokenNotFound
	}

	return token, nil
}