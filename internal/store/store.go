package store

import (
	"context"

	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/messages"
)

type Store interface {
	SaveMessage(ctx context.Context, msg messages.Message) error
	ListMessages(ctx context.Context, limit int) ([]messages.Message, error)
	CurrentFrame(ctx context.Context, boardID string) (board.Frame, error)
	SaveCurrentFrame(ctx context.Context, boardID string, frame board.Frame) error
}
