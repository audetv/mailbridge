package sqlite_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
)

func TestGetInboxItemsByThread(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём несколько входящих в одном thread
	for i := 0; i < 3; i++ {
		item := &store.InboxItem{
			Source:   "email",
			SourceID: fmt.Sprintf("msg-%d", i),
			ThreadID: "thread-1",
			Status:   "unread",
		}
		if err := s.CreateInboxItem(ctx, item); err != nil {
			t.Fatalf("CreateInboxItem error: %v", err)
		}
	}

	items, err := s.GetInboxItemsByThread(ctx, "thread-1")
	if err != nil {
		t.Fatalf("GetInboxItemsByThread error: %v", err)
	}
	if len(items) != 3 {
		t.Errorf("expected 3 items, got %d", len(items))
	}
}
