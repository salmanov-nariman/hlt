package domain

import "errors"

// chats
var (
	ErrChatNotFound  = errors.New("chat not found")
	ErrUserNotInChat = errors.New("user not in chat")
)

// messages
var (
	ErrMessageConflict      = errors.New("client_message_id conflict")
	ErrMessageAlreadyExists = errors.New("message already exists")
)

// keys
var (
	ErrPublicKeyNotFound = errors.New("public key not found")
	ErrInvalidPublicKey  = errors.New("invalid public key")
)

var ErrInvalidCipherText = errors.New("invalid cipher text")
