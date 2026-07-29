package sqlite_test

import (
	"context"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
)

func setupStore(t *testing.T) (*sqlite.Store, func()) {
	t.Helper()

	s, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		s.Close()
		t.Fatalf("failed to migrate: %v", err)
	}

	cleanup := func() {
		s.Close()
	}

	return s, cleanup
}

func TestMigrate(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()

	ctx := context.Background()

	err := s.SaveMapping(ctx, &store.EmailMapping{
		MessageID:       "test-msg-id",
		PlaneIssueID:    "plane-issue-1",
		PlaneProjectID:  "plane-project-1",
		PlaneIssueSeq:   "INBOX-1",
		OriginalFrom:    "test@example.com",
		OriginalSubject: "Test Subject",
		ActionType:      "CREATE",
	})
	if err != nil {
		t.Fatalf("SaveMapping failed: %v", err)
	}
}

func TestSaveAndGetMapping(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	mapping := &store.EmailMapping{
		MessageID:        "msg-001",
		PlaneIssueID:     "issue-uuid-001",
		PlaneProjectID:   "project-uuid-001",
		PlaneIssueSeq:    "INBOX-1",
		OriginalFrom:     "user@example.com",
		OriginalSubject:  "Test",
		ThreadReferences: []string{"ref-1", "ref-2"},
		ActionType:       "CREATE",
	}

	err := s.SaveMapping(ctx, mapping)
	if err != nil {
		t.Fatalf("SaveMapping failed: %v", err)
	}

	got, err := s.GetMappingByMessageID(ctx, "msg-001")
	if err != nil {
		t.Fatalf("GetMappingByMessageID failed: %v", err)
	}

	if got == nil {
		t.Fatal("expected mapping, got nil")
	}
	if got.MessageID != mapping.MessageID {
		t.Errorf("MessageID = %s, want %s", got.MessageID, mapping.MessageID)
	}
	if got.PlaneProjectID != "project-uuid-001" {
		t.Errorf("PlaneProjectID = %s, want project-uuid-001", got.PlaneProjectID)
	}
	if got.PlaneIssueSeq != "INBOX-1" {
		t.Errorf("PlaneIssueSeq = %s, want INBOX-1", got.PlaneIssueSeq)
	}
	if len(got.ThreadReferences) != 2 {
		t.Errorf("ThreadReferences length = %d, want 2", len(got.ThreadReferences))
	}
}

func TestMessageExists(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	exists, err := s.MessageExists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("MessageExists failed: %v", err)
	}
	if exists {
		t.Error("expected false for nonexistent message")
	}

	err = s.SaveMapping(ctx, &store.EmailMapping{
		MessageID:       "msg-002",
		PlaneIssueID:    "issue-002",
		PlaneProjectID:  "project-002",
		OriginalFrom:    "user@example.com",
		OriginalSubject: "Test",
		ActionType:      "CREATE",
	})
	if err != nil {
		t.Fatalf("SaveMapping failed: %v", err)
	}

	exists, err = s.MessageExists(ctx, "msg-002")
	if err != nil {
		t.Fatalf("MessageExists failed: %v", err)
	}
	if !exists {
		t.Error("expected true for existing message")
	}
}

func TestFindMappingByReferences(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	err := s.SaveMapping(ctx, &store.EmailMapping{
		MessageID:        "ref-msg-1",
		PlaneIssueID:     "issue-003",
		PlaneProjectID:   "project-003",
		PlaneIssueSeq:    "INBOX-3",
		OriginalFrom:     "user@example.com",
		OriginalSubject:  "Test",
		ThreadReferences: []string{"thread-1", "thread-2"},
		ActionType:       "CREATE",
	})
	if err != nil {
		t.Fatalf("SaveMapping failed: %v", err)
	}

	m, err := s.FindMappingByReferences(ctx, []string{"ref-msg-1"})
	if err != nil {
		t.Fatalf("FindMappingByReferences failed: %v", err)
	}
	if m == nil {
		t.Fatal("expected mapping, got nil")
	}

	m, err = s.FindMappingByReferences(ctx, []string{"thread-1"})
	if err != nil {
		t.Fatalf("FindMappingByReferences failed: %v", err)
	}
	if m == nil {
		t.Fatal("expected mapping by thread reference, got nil")
	}
}

func TestGetLatestMappingByIssueID(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	err := s.SaveMapping(ctx, &store.EmailMapping{
		MessageID:       "msg-a",
		PlaneIssueID:    "issue-004",
		PlaneProjectID:  "project-004",
		PlaneIssueSeq:   "INBOX-4",
		OriginalFrom:    "user1@example.com",
		OriginalSubject: "First",
		ActionType:      "CREATE",
	})
	if err != nil {
		t.Fatalf("SaveMapping failed: %v", err)
	}

	err = s.SaveMapping(ctx, &store.EmailMapping{
		MessageID:       "msg-b",
		PlaneIssueID:    "issue-004",
		PlaneProjectID:  "project-004",
		PlaneIssueSeq:   "INBOX-4",
		OriginalFrom:    "user2@example.com",
		OriginalSubject: "Second",
		ActionType:      "REPLY",
	})
	if err != nil {
		t.Fatalf("SaveMapping failed: %v", err)
	}

	m, err := s.GetLatestMappingByIssueID(ctx, "issue-004")
	if err != nil {
		t.Fatalf("GetLatestMappingByIssueID failed: %v", err)
	}
	if m == nil {
		t.Fatal("expected mapping, got nil")
	}
	if m.MessageID != "msg-b" {
		t.Errorf("expected latest msg-b, got %s", m.MessageID)
	}
}

func TestDuplicateMessageID(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	mapping := &store.EmailMapping{
		MessageID:       "dup-msg",
		PlaneIssueID:    "issue-005",
		PlaneProjectID:  "project-005",
		OriginalFrom:    "user@example.com",
		OriginalSubject: "Test",
		ActionType:      "CREATE",
	}

	err := s.SaveMapping(ctx, mapping)
	if err != nil {
		t.Fatalf("first SaveMapping failed: %v", err)
	}

	err = s.SaveMapping(ctx, mapping)
	if err == nil {
		t.Fatal("expected error for duplicate message_id")
	}
}

func TestReplyLog(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	err := s.SaveReplyLog(ctx, &store.ReplyLog{
		MessageID:    "reply-001",
		InReplyTo:    "original-001",
		PlaneIssueID: "issue-006",
	})
	if err != nil {
		t.Fatalf("SaveReplyLog failed: %v", err)
	}

	exists, err := s.ReplyExists(ctx, "reply-001")
	if err != nil {
		t.Fatalf("ReplyExists failed: %v", err)
	}
	if !exists {
		t.Error("expected reply to exist")
	}

	exists, err = s.ReplyExists(ctx, "nonexistent-reply")
	if err != nil {
		t.Fatalf("ReplyExists failed: %v", err)
	}
	if exists {
		t.Error("expected reply not to exist")
	}
}

func TestOutbox(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	err := s.EnqueueOutbox(ctx, `{"to":"test@example.com","subject":"Hello"}`)
	if err != nil {
		t.Fatalf("EnqueueOutbox failed: %v", err)
	}

	items, err := s.GetPendingOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("GetPendingOutbox failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Status != "pending" {
		t.Errorf("expected status pending, got %s", items[0].Status)
	}

	err = s.MarkOutboxSent(ctx, items[0].ID)
	if err != nil {
		t.Fatalf("MarkOutboxSent failed: %v", err)
	}

	items, err = s.GetPendingOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("GetPendingOutbox failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 pending items, got %d", len(items))
	}
}

func TestMarkOutboxFailed(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	err := s.EnqueueOutbox(ctx, `{"test":true}`)
	if err != nil {
		t.Fatalf("EnqueueOutbox failed: %v", err)
	}

	items, _ := s.GetPendingOutbox(ctx, 1)

	err = s.MarkOutboxFailed(ctx, items[0].ID, "smtp timeout")
	if err != nil {
		t.Fatalf("MarkOutboxFailed failed: %v", err)
	}

	items, _ = s.GetPendingOutbox(ctx, 1)
	if len(items) != 0 {
		t.Errorf("expected 0 pending items, got %d", len(items))
	}
}

func TestPing(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	if err := s.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}
