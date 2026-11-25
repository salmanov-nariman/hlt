package service_test

import (
	"bytes"
	"chat/internal/domain"
	"chat/internal/service"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestChatService_SaveKey_InvalidPublicKey(t *testing.T) {
	tests := []struct {
		name      string
		publicKey []byte
	}{
		{name: "invalid key nil", publicKey: nil},
		{name: "invalid key empty", publicKey: []byte{}},
		{name: "invalid key too short", publicKey: bytes.Repeat([]byte{1}, 31)},
		{name: "invalid key too long", publicKey: bytes.Repeat([]byte{1}, 33)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockChatRepository{
				saveKeyFunc: func(ctx context.Context, userID string, publicKey []byte) error {
					t.Fatal("SaveKey не должен вызываться при невалидном publicKey")
					return nil
				},
			}

			svc := service.NewChatService(nil, mockRepo, nil)
			err := svc.SaveKey(t.Context(), "good_user_1", tt.publicKey)

			require.ErrorIs(t, err, domain.ErrInvalidPublicKey)
		})
	}
}

func TestChatService_SaveKey_Success(t *testing.T) {
	called := false
	expectedUserID := "good_user_1"
	expectedPublicKey := bytes.Repeat([]byte{1}, 32) // repeat возвращает массив байтов из копий значений заданной длины

	mockRepo := &mockChatRepository{
		saveKeyFunc: func(ctx context.Context, userID string, publicKey []byte) error {
			called = true

			require.Equal(t, userID, expectedUserID)
			require.Equal(t, publicKey, expectedPublicKey)

			return nil
		},
	}

	svc := service.NewChatService(nil, mockRepo, nil)
	err := svc.SaveKey(t.Context(), expectedUserID, expectedPublicKey)
	require.Nil(t, err)
	require.True(t, called)
}

func TestChatService_SaveKey_RepoError(t *testing.T) {
	repoErr := errors.New("db connection failed")
	userID := "good_user_1"
	publicKey := bytes.Repeat([]byte{1}, 32)

	mockRepo := &mockChatRepository{
		saveKeyFunc: func(ctx context.Context, userID string, publicKey []byte) error {
			return repoErr
		},
	}

	svc := service.NewChatService(nil, mockRepo, nil)
	err := svc.SaveKey(t.Context(), userID, publicKey)
	require.ErrorIs(t, err, repoErr)
}

func TestChatService_GetLatestKey_Success(t *testing.T) {
	var callCountGetKey, callCountGetUsers int
	chatID := "chat_1"
	senderID := "good_user_1"
	targetUserID := "good_user_2"
	expectedKey := domain.UserPublicKey{
		ID:        "key_1",
		UserID:    targetUserID,
		PublicKey: bytes.Repeat([]byte{1}, 32),
		UpdatedAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	mockRepo := &mockChatRepository{
		getLatestKeyFunc: func(ctx context.Context, userID string) (domain.UserPublicKey, error) {
			callCountGetKey++

			require.Equal(t, userID, targetUserID)

			key := expectedKey
			key.PublicKey = bytes.Clone(expectedKey.PublicKey)

			return key, nil
		},

		getUsersInChatFunc: func(ctx context.Context, actualChatID string) ([]string, error) {
			callCountGetUsers++
			require.Equal(t, chatID, actualChatID)
			return []string{senderID, targetUserID}, nil
		},
	}

	svc := service.NewChatService(nil, mockRepo, nil)
	actualKey, err := svc.GetLatestKey(t.Context(), chatID, senderID, targetUserID)

	require.Nil(t, err)
	require.Equal(t, 1, callCountGetUsers)
	require.Equal(t, 1, callCountGetKey)
	require.Equal(t, expectedKey, actualKey)
}

func TestChatService_GetLatestKey_GetUsersInChatError(t *testing.T) {
	var callCountGetUsers int

	repoErr := errors.New("db connection failed")

	chatID := "chat_1"
	senderID := "good_user_1"
	targetUserID := "good_user_2"

	expectedKey := domain.UserPublicKey{}

	mockRepo := &mockChatRepository{
		getUsersInChatFunc: func(ctx context.Context, actualChatID string) ([]string, error) {
			require.Equal(t, actualChatID, chatID)
			callCountGetUsers++
			return []string{}, repoErr
		},

		getLatestKeyFunc: func(ctx context.Context, userID string) (domain.UserPublicKey, error) {
			t.Fatal("метод repo.GetLatestKey не должен вызываться при ошибке получения пользователей чата")
			return expectedKey, nil
		},
	}

	svc := service.NewChatService(nil, mockRepo, nil)
	actualKey, err := svc.GetLatestKey(t.Context(), chatID, senderID, targetUserID)

	require.ErrorIs(t, err, repoErr)
	require.Equal(t, 1, callCountGetUsers)
	require.Equal(t, expectedKey, actualKey)
}

func TestChatService_GetLatestKey_AccessDenied(t *testing.T) {
	chatID := "chat_1"
	senderID := "good_user_1"
	targetUserID := "good_user_2"

	expectedKey := domain.UserPublicKey{}

	tests := []struct {
		name        string
		chatMembers []string
	}{
		{name: "target not in chat", chatMembers: []string{senderID, "user_3"}},
		{name: "sender not in chat", chatMembers: []string{targetUserID, "user_3"}},
		{name: "both not in chat", chatMembers: []string{"user_3", "user_4"}},
		{name: "empty chat members", chatMembers: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var callCountGetUsers int

			mockRepo := &mockChatRepository{
				getUsersInChatFunc: func(ctx context.Context, actualChatID string) ([]string, error) {
					callCountGetUsers++
					require.Equal(t, chatID, actualChatID)
					return tt.chatMembers, nil
				},

				getLatestKeyFunc: func(ctx context.Context, userID string) (domain.UserPublicKey, error) {
					t.Fatalf("метод repo.GetLatestKey не должен вызываться при отсутствии хотя бы одного пользователя")
					return expectedKey, nil
				},
			}

			svc := service.NewChatService(nil, mockRepo, nil)
			actualKey, err := svc.GetLatestKey(t.Context(), chatID, senderID, targetUserID)

			require.ErrorIs(t, err, domain.ErrUserNotInChat)
			require.Equal(t, 1, callCountGetUsers)
			require.Equal(t, expectedKey, actualKey)
		})
	}
}

func TestChatService_GetLatestKey_RepoError(t *testing.T) {
	chatID := "chat_1"
	senderID := "user_1"
	targetUserID := "user_2"
	expectedKey := domain.UserPublicKey{}

	repoError := errors.New("db connection failed")

	tests := []struct {
		name string
		err  error
	}{
		{name: "public key not found", err: domain.ErrPublicKeyNotFound},
		{name: "db error", err: repoError},
	}

	for _, tt := range tests {
		var callCountGetUsers, callCountGetKeys int

		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &mockChatRepository{
				getUsersInChatFunc: func(
					ctx context.Context,
					actualChatID string,
				) ([]string, error) {
					callCountGetUsers++
					require.Equal(t, chatID, actualChatID)
					return []string{senderID, targetUserID}, nil
				},

				getLatestKeyFunc: func(
					ctx context.Context,
					userID string,
				) (domain.UserPublicKey, error) {
					callCountGetKeys++
					require.Equal(t, targetUserID, userID)
					return expectedKey, tt.err
				},
			}

			svc := service.NewChatService(nil, mockRepo, nil)
			actualKey, err := svc.GetLatestKey(t.Context(), chatID, senderID, targetUserID)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, 1, callCountGetUsers)
			require.Equal(t, 1, callCountGetKeys)
			require.Equal(t, expectedKey, actualKey)
		})
	}

}
