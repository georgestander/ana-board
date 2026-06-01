package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/messages"
)

const DefaultMessageLimit = 200

type MemoryStore struct {
	mu       sync.RWMutex
	frames   map[string]board.Frame
	messages []messages.Message
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		frames: make(map[string]board.Frame),
	}
}

func (s *MemoryStore) SaveMessage(ctx context.Context, msg messages.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.messages = append([]messages.Message{msg}, s.messages...)
	if len(s.messages) > DefaultMessageLimit {
		s.messages[DefaultMessageLimit] = messages.Message{}
		s.messages = s.messages[:DefaultMessageLimit:DefaultMessageLimit]
	}

	return nil
}

func (s *MemoryStore) ListMessages(ctx context.Context, limit int) ([]messages.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 20
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit > len(s.messages) {
		limit = len(s.messages)
	}

	out := make([]messages.Message, limit)
	copy(out, s.messages[:limit])
	return out, nil
}

func (s *MemoryStore) CurrentFrame(ctx context.Context, boardID string) (board.Frame, error) {
	if err := ctx.Err(); err != nil {
		return board.Frame{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	frame, ok := s.frames[boardID]
	if !ok {
		return board.Frame{}, fmt.Errorf("frame for board %q was not found", boardID)
	}

	return frame, nil
}

func (s *MemoryStore) SaveCurrentFrame(ctx context.Context, boardID string, frame board.Frame) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.frames[boardID] = frame
	return nil
}
