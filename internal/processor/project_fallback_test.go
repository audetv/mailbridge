package processor_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/audetv/mailbridge/internal/ai"
	"github.com/audetv/mailbridge/internal/classifier"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/parser"
	"github.com/audetv/mailbridge/internal/processor"
	"github.com/audetv/mailbridge/internal/store/sqlite"
)

// fakeClassifier возвращает фиксированную классификацию.
type fakeClassifier struct {
	classification *classifier.Classification
}

func (f *fakeClassifier) Classify(_ context.Context, _ string, _ []string, _ []string) (*classifier.Classification, error) {
	return f.classification, nil
}

// TestCreateNewTask_ProjectFallback — шаг 15: если классификатор не определил
// проект, задача привязывается к MAILBRIDGE_DEFAULT_PROJECT (ai.DefaultProject),
// а не остаётся без проекта.
func TestCreateNewTask_ProjectFallback(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer st.Close()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	t.Setenv("MAILBRIDGE_DEFAULT_PROJECT", "Входящие (AI)")

	raw := []byte(
		"From: Ivan <ivan@example.com>\n" +
			"To: service@example.ru\n" +
			"Subject: Проблема с оплатой\n" +
			"Message-ID: <fallback-15-1@test.local>\n" +
			"\n" +
			"Текст письма: не могу оплатить счёт.\n",
	)

	proc := processor.NewMessageProcessor(
		st,
		&fakeClassifier{classification: &classifier.Classification{
			Project:    "", // — классификатор проект не определил
			Type:       "bug",
			Priority:   "medium",
			Confidence: 0.5,
		}},
		extractor.NewExtractor(nil),
		parser.NewFieldParser(nil, nil),
		nil,
		logger,
		nil,
		nil,
		false,
		nil,
		nil,
	)

	result, err := proc.Process(ctx, raw)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if result.Action != processor.ActionCreateIssue {
		t.Errorf("Action = %q, want %q", result.Action, processor.ActionCreateIssue)
	}
	if result.Task == nil {
		t.Fatal("Task = nil, want created")
	}
	if got, want := result.Task.Project, ai.DefaultProject; got != want {
		t.Errorf("Task.Project = %q, want %q (fallback)", got, want)
	}
}

// TestCreateNewTask_ClassifiedProjectWins — шаг 15 (regression): если классификатор
// определил проект, он используется, fallback не срабатывает.
func TestCreateNewTask_ClassifiedProjectWins(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer st.Close()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	t.Setenv("MAILBRIDGE_DEFAULT_PROJECT", "Входящие (AI)")

	raw := []byte(
		"From: Ivan <ivan@example.com>\n" +
			"To: service@example.ru\n" +
			"Subject: Проблема с оплатой\n" +
			"Message-ID: <fallback-15-2@test.local>\n" +
			"\n" +
			"Текст письма: не могу оплатить счёт.\n",
	)

	proc := processor.NewMessageProcessor(
		st,
		&fakeClassifier{classification: &classifier.Classification{
			Project:    "Лидер Спорт",
			Type:       "bug",
			Priority:   "medium",
			Confidence: 0.9,
		}},
		extractor.NewExtractor(nil),
		parser.NewFieldParser(nil, nil),
		nil,
		logger,
		nil,
		nil,
		false,
		nil,
		nil,
	)

	result, err := proc.Process(ctx, raw)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if result.Task == nil {
		t.Fatal("Task = nil, want created")
	}
	if got, want := result.Task.Project, "Лидер Спорт"; got != want {
		t.Errorf("Task.Project = %q, want %q", got, want)
	}
}
