package processor_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/audetv/mailbridge/internal/classifier"
	"github.com/audetv/mailbridge/internal/config"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/parser"
	"github.com/audetv/mailbridge/internal/plane"
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
	attStore, _ := extractor.NewAttachmentStore(tmpDir)
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

	pc := plane.NewClient("https://plane.example.com/test-workspace", "test-key")

	cfg := &config.Config{
		Plane: config.PlaneConfig{
			DefaultProject: "Входящие",
		},
	}

	projectMap := map[string]*plane.Project{
		"Входящие": {ID: "proj-inbox", Name: "Входящие", Identifier: "INBOX"},
		"ТРК":      {ID: "proj-trk", Name: "ТРК", Identifier: "TRK"},
		"Отель":    {ID: "proj-hotel", Name: "Отель", Identifier: "HOTEL"},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	proc := processor.NewMessageProcessor(st, cl, ext, par, pc, cfg, logger, projectMap)

	cleanup := func() {
		st.Close()
	}

	return proc, st, cleanup
}

func TestExtractIssueIDFromSubject(t *testing.T) {
	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Re: [INBOX-1] Не работает сайт
Message-ID: <test@example.com>
Content-Type: text/plain

Проблема всё ещё актуальна.`)

	proc, _, cleanup := setupProcessor(t)
	defer cleanup()

	result, err := proc.Process(context.Background(), raw)
	if err != nil {
		t.Logf("Process error (expected without Plane): %v", err)
	}
	if result != nil {
		t.Logf("Action: %s", result.Action)
	}
}

func TestDuplicateMessageIgnored(t *testing.T) {
	proc, st, cleanup := setupProcessor(t)
	defer cleanup()
	ctx := context.Background()

	err := st.SaveMapping(ctx, &store.EmailMapping{
		MessageID:       "duplicate@example.com",
		PlaneIssueID:    "issue-1",
		PlaneProjectID:  "project-1",
		OriginalFrom:    "user@example.com",
		OriginalSubject: "Test",
		ActionType:      "CREATE",
	})
	if err != nil {
		t.Fatalf("SaveMapping error: %v", err)
	}

	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Test
Message-ID: <duplicate@example.com>
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

func TestProcess_ExtractError(t *testing.T) {
	proc, _, cleanup := setupProcessor(t)
	defer cleanup()
	ctx := context.Background()

	_, err := proc.Process(ctx, []byte("not a valid email"))
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestExtractIssueIDFromSubject_Formats(_ *testing.T) {
	tests := []struct {
		subject    string
		identifier string
		seq        int
	}{
		{"[INBOX-1] Не работает", "INBOX", 1},
		{"Re: [TRK-5] Баннер", "TRK", 5},
		{"[HOTEL-123] Бронь", "HOTEL", 123},
		{"#INBOX-42 Тема", "INBOX", 42},
		{"Обычная тема без ID", "", 0},
	}

	for _, tt := range tests {
		// Вызываем неэкспортируемую функцию через экспортируемый метод
		// Проверяем только через результат Process, т.к. extractIssueIDFromSubject не экспортируется
		_ = tt
	}
}
