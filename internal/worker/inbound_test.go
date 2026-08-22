package worker_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/audetv/mailbridge/internal/classifier"
	"github.com/audetv/mailbridge/internal/config"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/mailbox"
	"github.com/audetv/mailbridge/internal/parser"
	"github.com/audetv/mailbridge/internal/plane"
	"github.com/audetv/mailbridge/internal/processor"
	"github.com/audetv/mailbridge/internal/store/sqlite"
	"github.com/audetv/mailbridge/internal/worker"
)

func TestInboundWorker_Creation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	reader := mailbox.NewReader(mailbox.Config{
		Server: "imap.example.com",
		Port:   993,
		Inbox:  "INBOX",
	}, logger)

	// Создаём минимальный processor для теста
	st, _ := sqlite.NewStore(":memory:")
	defer st.Close()
	_ = st.Migrate(context.Background())

	attStore, _ := extractor.NewAttachmentStore(t.TempDir())
	ext := extractor.NewExtractor(attStore)

	cl := classifier.NewRuleBasedClassifier(
		classifier.TestRules(),
		[]string{"срочно"},
		map[string]bool{"bug": true},
		map[string]bool{"high": true},
	)

	par := parser.NewFieldParser(
		map[string]bool{"bug": true},
		map[string]bool{"high": true},
	)

	_ = plane.NewClient("https://plane.example.com/test", "key")

	projectNameMap := map[string]string{
		"Входящие": "INBOX",
	}

	proc := processor.NewMessageProcessor(st, cl, ext, par, &config.Config{Plane: config.PlaneConfig{DefaultProject: "Входящие"}}, logger, projectNameMap, nil, nil, false, nil)

	w := worker.NewInboundWorker(reader, proc, 30*time.Second, logger)

	if w == nil {
		t.Fatal("expected worker, got nil")
	}
}
