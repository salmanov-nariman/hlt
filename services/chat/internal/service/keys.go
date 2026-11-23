package service

import (
	"chat/internal/domain"
	"context"
	"fmt"
	"slices"
)


func (s *ChatService) SaveKey(ctx context.Context, userID string, publicKey []byte) error {
	if len(publicKey) != 32 {
		return domain.ErrInvalidPublicKey
	}

	if err := s.chatRepository.SaveKey(ctx, userID, publicKey); err != nil {
		return fmt.Errorf("save key: %w", err)
	}

	return nil
}

func (s *ChatService) GetLatestKey(ctx context.Context, chatID, senderID, targetUserID string) (domain.UserPublicKey, error) {
	membersIDs, err := s.chatRepository.GetUsersInChat(ctx, chatID)
	if err != nil {
		return domain.UserPublicKey{}, fmt.Errorf("failed to get users in chat: %w", err)
	}

	if !slices.Contains(membersIDs, targetUserID) || !slices.Contains(membersIDs, senderID) {
		return domain.UserPublicKey{}, fmt.Errorf("chat %s access denied: %w", chatID, domain.ErrUserNotInChat)
	}

	return s.chatRepository.GetLatestKey(ctx, targetUserID)
}