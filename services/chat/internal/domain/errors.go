package domain

import "errors"

//chats
var (
	ErrChatNotFound  = errors.New("chat not found")
	ErrUserNotInChat = errors.New("user not in chat")
)

// messages
var (
	ErrMessageConflict      = errors.New("client_message_id conflict")
	ErrMessageAlreadyExists = errors.New("message already exists")
)
