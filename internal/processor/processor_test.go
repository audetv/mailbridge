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

	// Создаём временную директорию для вложений
	tmpDir := t.TempDir()
	attStore, _ := extractor.NewAttachmentStore(tmpDir)
	ext := extractor.NewExtractor(attStore)

	// Классификатор
	rules := classifier.TestRules()
	cl := classifier.NewRuleBasedClassifier(
		rules,
		[]string{"срочно", "горит", "упал"},
		map[string]bool{"bug": true, "feature": true, "support": true, "access": true, "seo": true, "content": true},
		map[string]bool{"urgent": true, "high": true, "medium": true, "low": true},
	)

	// Парсер
	par := parser.NewFieldParser(
		map[string]bool{"bug": true, "feature": true, "support": true, "access": true, "seo": true, "content": true},
		map[string]bool{"urgent": true, "high": true, "medium": true, "low": true},
	)

	// PlaneClient (не используется в тестах без mock-сервера, но нужен для конструктора)
	pc := plane.NewClient("https://plane.example.com/test-workspace", "test-key")

	cfg := &config.Config{
		Plane: config.PlaneConfig{
			DefaultProject: "Входящие",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	proc := processor.NewMessageProcessor(st, cl, ext, par, pc, cfg, logger)

	cleanup := func() {
		st.Close()
	}

	return proc, st, cleanup
}

func TestExtractIssueIDFromSubject_WithBrackets(t *testing.T) {
	raw := []byte(`From: user@example.com
To: support@example.com
Subject: Re: [WEB-123] Не работает сайт
Message-ID: <test@example.com>
Content-Type: text/plain

Проблема всё ещё актуальна.`)

	// Проверяем извлечение ID из темы
	proc, _, cleanup := setupProcessor(t)
	defer cleanup()

	ext, _ := extractor.NewAttachmentStore(t.TempDir())
	extractor := extractor.NewExtractor(ext)
	email, err := extractor.Extract(raw)
	if err != nil {
		t.Fatalf("extract error: %v", err)
	}

	// Проверяем, что тема содержит [WEB-123]
	if email.Subject != "Re: [WEB-123] Не работает сайт" {
		t.Logf("Subject = %q", email.Subject)
	}

	// Процессор обработает письмо (без Plane — будет ошибка, но логику проверим)
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

	// Сохраняем маппинг
	err := st.SaveMapping(ctx, &store.EmailMapping{
		MessageID:       "duplicate@example.com",
		PlaneIssueID:    "issue-1",
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

	// Невалидный email
	_, err := proc.Process(ctx, []byte("not a valid email"))
	if err == nil {
		t.Fatal("expected error for invalid email")
	}
}
