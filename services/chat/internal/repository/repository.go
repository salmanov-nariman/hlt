package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
    ErrMessageConflict      = errors.New("client_message_id conflict with different text")
    ErrMessageAlreadyExists = errors.New("message already exists")
)

type ChatRepository struct {
	db *pgxpool.Pool
}

func NewChatRepository(db *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{db: db}
}



