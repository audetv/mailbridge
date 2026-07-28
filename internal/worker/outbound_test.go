package worker_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/audetv/mailbridge/internal/sender"
	"github.com/audetv/mailbridge/internal/store/sqlite"
	"github.com/audetv/mailbridge/internal/worker"
)

func TestEnqueueAcknowledgement(t *testing.T) {
	st, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore error: %v", err)
	}
	defer st.Close()
	_ = st.Migrate(context.Background())

	data := &sender.AcknowledgementData{
		To:                 "user@example.com",
		Subject:            "Test",
		InReplyToMessageID: "msg-1",
		IssueSequence:      "WEB-1",
		ProjectName:        "ТРК",
		TypeName:           "bug",
		Priority:           "high",
	}

	err = worker.EnqueueAcknowledgement(context.Background(), st, data)
	if err != nil {
		t.Fatalf("Enqueue error: %v", err)
	}

	items, err := st.GetPendingOutbox(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetPendingOutbox error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestEnqueueCommentReply(t *testing.T) {
	st, _ := sqlite.NewStore(":memory:")
	defer st.Close()
	_ = st.Migrate(context.Background())

	data := &sender.CommentReplyData{
		To:                 "user@example.com",
		Subject:            "Test",
		InReplyToMessageID: "msg-1",
		IssueSequence:      "WEB-1",
		CommentText:        "Ответ",
		CommentAuthor:      "Руслан",
	}

	err := worker.EnqueueCommentReply(context.Background(), st, data)
	if err != nil {
		t.Fatalf("Enqueue error: %v", err)
	}
}

func TestOutboundWorker_Creation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	st, _ := sqlite.NewStore(":memory:")
	defer st.Close()

	s := sender.NewSender(sender.Config{
		Server: "smtp.example.com",
		Port:   587,
		From:   "support@example.com",
	}, logger)

	w := worker.NewOutboundWorker(st, s, 15*time.Second, logger)
	if w == nil {
		t.Fatal("expected worker, got nil")
	}
}
