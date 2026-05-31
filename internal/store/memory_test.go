package store

import (
	"context"
	"testing"
	"time"

	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/messages"
)

func TestMemoryStoreSavesAndReadsCurrentFrame(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	frame, err := board.NewFrame(board.DefaultRows, board.DefaultCols)
	if err != nil {
		t.Fatalf("NewFrame returned error: %v", err)
	}

	if err := frame.Set(1, 2, board.Cell{Symbol: "A"}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	if err := store.SaveCurrentFrame(ctx, "default", frame); err != nil {
		t.Fatalf("SaveCurrentFrame returned error: %v", err)
	}

	got, err := store.CurrentFrame(ctx, "default")
	if err != nil {
		t.Fatalf("CurrentFrame returned error: %v", err)
	}

	cell, err := got.CellAt(1, 2)
	if err != nil {
		t.Fatalf("CellAt returned error: %v", err)
	}

	if cell.Symbol != "A" {
		t.Fatalf("cell = %q, want %q", cell.Symbol, "A")
	}
}

func TestMemoryStoreListsNewestMessagesFirst(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	first := messages.Message{ID: "1", Text: "FIRST", CreatedAt: time.Now()}
	second := messages.Message{ID: "2", Text: "SECOND", CreatedAt: time.Now()}

	if err := store.SaveMessage(ctx, first); err != nil {
		t.Fatalf("SaveMessage returned error: %v", err)
	}

	if err := store.SaveMessage(ctx, second); err != nil {
		t.Fatalf("SaveMessage returned error: %v", err)
	}

	got, err := store.ListMessages(ctx, 10)
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}

	if got[0].ID != "2" {
		t.Fatalf("first message ID = %q, want %q", got[0].ID, "2")
	}
}

func TestMemoryStoreListMessagesHonorsLimit(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	for _, id := range []string{"1", "2", "3"} {
		if err := store.SaveMessage(ctx, messages.Message{ID: id, Text: id}); err != nil {
			t.Fatalf("SaveMessage returned error: %v", err)
		}
	}

	got, err := store.ListMessages(ctx, 2)
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
}
