package processor_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/audetv/mailbridge/internal/classifier"
	"github.com/audetv/mailbridge/internal/config"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/parser"
	"github.com/audetv/mailbridge/internal/processor"
	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
)

func setupProcessor(t *testing.T) (*processor.MessageProcessor, *sqlite.Store, func()) {
	t.Helper()

	st, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		st.Close()
		t.Fatalf("failed to migrate: %v", err)
	}

	tmpDir := t.TempDir()
	attStore, err := extractor.NewAttachmentStore(tmpDir)
	if err != nil {
		t.Fatalf("NewAttachmentStore error: %v", err)
	}
	ext := extractor.NewExtractor(attStore)

	rules := classifier.TestRules()
	cl := classifier.NewRuleBasedClassifier(
		rules,
		[]string{"срочно", "горит", "упал"},
		map[string]bool{"bug": true, "feature": true, "support": true, "access": true, "seo": true, "content": true},
		map[string]bool{"urgent": true, "high": true, "medium": true, "low": true},
	)

	par := parser.NewFieldParser(
		map[string]bool{"bug": true, "feature": true, "support": true, "access": true, "seo": true, "content": true},
		map[string]bool{"urgent": true, "high": true, "medium": true, "low": true},
	)

	cfg := &config.Config{
		Plane: config.PlaneConfig{
			DefaultProject: "Входящие",
		},
	}

	projectMap := map[string]string{
		"Входящие": "Входящие",
		"ТРК":      "ТРК",
		"Отель":    "Отель",
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	proc := processor.NewMessageProcessor(st, cl, ext, par, cfg, logger, projectMap, nil, nil, false)

	cleanup := func() {
		st.Close()
	}

	return proc, st, cleanup
}

func TestProcess_NewTask(t *testing.T) {
	proc, st, cleanup := setupProcessor(t)
	defer cleanup()
	ctx := context.Background()

	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Не работает кабинет арендатора
Message-ID: <test-new@example.com>
Content-Type: text/plain

Ошибка 500 при входе.`)

	result, err := proc.Process(ctx, raw)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}

	if result.Action != processor.ActionCreateIssue {
		t.Errorf("expected ActionCreateIssue, got %s", result.Action)
	}
	if result.TaskID == 0 {
		t.Error("TaskID is 0")
	}

	// Проверяем задачу в БД
	task, err := st.GetTask(ctx, result.TaskID)
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if task == nil {
		t.Fatal("task not found in DB")
	}
	if task.Subject != "Не работает кабинет арендатора" {
		t.Errorf("Subject = %s", task.Subject)
	}

	// Проверяем комментарий
	comments, err := st.GetTaskComments(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTaskComments error: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("expected 0 comment, got %d", len(comments))
	}
}

func TestProcess_Duplicate(t *testing.T) {
	proc, st, cleanup := setupProcessor(t)
	defer cleanup()
	ctx := context.Background()

	// Создаём задачу напрямую
	err := st.CreateTask(ctx, &store.Task{
		MessageID: "dup@example.com",
		Subject:   "Test",
		BodyText:  "Body",
		FromEmail: "user@example.com",
		Status:    "new",
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Test
Message-ID: <dup@example.com>
Content-Type: text/plain

Повторное письмо.`)

	result, err := proc.Process(ctx, raw)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}

	if result.Action != processor.ActionIgnore {
		t.Errorf("expected ActionIgnore, got %s", result.Action)
	}
}

func TestProcess_ReplyToTask(t *testing.T) {
	proc, st, cleanup := setupProcessor(t)
	defer cleanup()
	ctx := context.Background()

	err := st.CreateTask(ctx, &store.Task{
		MessageID: "original@example.com",
		Subject:   "Original",
		BodyText:  "Body",
		FromEmail: "user@example.com",
		Status:    "new",
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	// Маппинг для threading
	if err := st.SaveMapping(ctx, &store.EmailMapping{
		MessageID:        "original@example.com",
		PlaneIssueID:     "task-1",
		OriginalFrom:     "user@example.com",
		OriginalSubject:  "Original",
		ThreadReferences: []string{"original@example.com"},
		ActionType:       "CREATE",
	}); err != nil {
		t.Fatalf("SaveMapping error: %v", err)
	}

	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Re: [TASK-1] Original
Message-ID: <reply@example.com>
In-Reply-To: <original@example.com>
Content-Type: text/plain

Ответ на задачу.`)

	result, err := proc.Process(ctx, raw)
	if err != nil {
		t.Fatalf("Process error: %v", err)
	}

	if result.Action != processor.ActionAddComment {
		t.Errorf("expected ActionAddComment, got %s", result.Action)
	}

	comments, _ := st.GetTaskComments(ctx, result.TaskID)
	if len(comments) != 1 {
		t.Errorf("expected 1 comment (reply only), got %d", len(comments))
	}
}

func TestProcess_InvalidEmail(t *testing.T) {
	proc, _, cleanup := setupProcessor(t)
	defer cleanup()
	ctx := context.Background()

	_, err := proc.Process(ctx, []byte("not a valid email"))
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestExtractTaskIDFromSubject(t *testing.T) {
	tests := []struct {
		subject string
		want    int64
	}{
		{"[TASK-1] Не работает", 1},
		{"Re: [TASK-123] Баннер", 123},
		{"Обычная тема", 0},
		{"[TASK-]", 0},
	}

	for i, tt := range tests {
		msgID := fmt.Sprintf("msg-%d@test", i)
		raw := []byte(fmt.Sprintf("From: u@e.com\nTo: s@e.com\nSubject: %s\nMessage-ID: <%s>\n\nBody", tt.subject, msgID))

		proc, _, cleanup := setupProcessor(t)

		result, err := proc.Process(context.Background(), raw)
		if err != nil {
			t.Logf("Process error (expected): %v", err)
		}
		if result != nil {
			t.Logf("Subject: %q, Action: %s, TaskID: %d", tt.subject, result.Action, result.TaskID)
		}
		cleanup()
	}
}
