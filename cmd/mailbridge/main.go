package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"

	"github.com/audetv/mailbridge/internal/classifier"
	"github.com/audetv/mailbridge/internal/config"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/health"
	"github.com/audetv/mailbridge/internal/logging"
	"github.com/audetv/mailbridge/internal/mailbox"
	"github.com/audetv/mailbridge/internal/parser"
	"github.com/audetv/mailbridge/internal/plane"
	"github.com/audetv/mailbridge/internal/processor"
	"github.com/audetv/mailbridge/internal/sender"
	"github.com/audetv/mailbridge/internal/store/sqlite"
	"github.com/audetv/mailbridge/internal/version"
	"github.com/audetv/mailbridge/internal/webhook"
	"github.com/audetv/mailbridge/internal/worker"
)

func main() {
	fmt.Println(version.Info())

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := logging.New(cfg.Logging.Level, cfg.Logging.Format)
	slog.SetDefault(logger)

	logger.Info("starting mailbridge",
		"version", version.Short(),
		"commit", version.Commit,
	)

	// Загружаем правила классификации
	rulesCfg, err := config.LoadRules(cfg.NLP.RulesFile)
	if err != nil {
		logger.Error("failed to load rules", "error", err)
		os.Exit(1)
	}
	logger.Info("rules loaded",
		"file", cfg.NLP.RulesFile,
		"rules_count", len(rulesCfg.Rules),
	)

	// Хранилище
	st, err := sqlite.NewStore(cfg.Storage.DSN)
	if err != nil {
		logger.Error("failed to create store", "error", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := st.Migrate(context.Background()); err != nil {
		logger.Error("failed to migrate store", "error", err)
		os.Exit(1)
	}
	logger.Info("database migrated")

	// Хранилище вложений
	attStore, err := extractor.NewAttachmentStore(cfg.Attachments.Dir)
	if err != nil {
		logger.Error("failed to create attachment store", "error", err)
		os.Exit(1)
	}

	// Extractor
	ext := extractor.NewExtractor(attStore)

	// Plane Client
	planeClient := plane.NewClient(cfg.Plane.BaseURL, cfg.Plane.APIKey)

	// Загружаем проекты Plane и строим карту
	projectMap := loadProjectMap(planeClient, logger)
	logger.Info("projects loaded from Plane", "count", len(projectMap))

	// Classifier
	rules := classifier.ConvertRules(rulesCfg.Rules)
	cl := classifier.NewRuleBasedClassifier(
		rules,
		rulesCfg.UrgencyPatterns,
		rulesCfg.ValidTypesMap(),
		rulesCfg.ValidPrioritiesMap(),
	)

	// Field Parser
	par := parser.NewFieldParser(
		rulesCfg.ValidTypesMap(),
		rulesCfg.ValidPrioritiesMap(),
	)

	// Message Processor
	proc := processor.NewMessageProcessor(
		st,
		cl,
		ext,
		par,
		planeClient,
		cfg,
		logger,
	)

	// Mail Reader
	mailReader := mailbox.NewReader(mailbox.Config{
		Server:   cfg.IMAP.Server,
		Port:     cfg.IMAP.Port,
		User:     cfg.IMAP.User,
		Password: cfg.IMAP.Password,
		TLS:      cfg.IMAP.TLS,
		Inbox:    cfg.IMAP.Inbox,
		Archive:  cfg.IMAP.Archive,
		Errors:   cfg.IMAP.Errors,
	}, logger)

	// IMAP — опционально, не падаем
	if err := mailReader.Connect(); err != nil {
		logger.Warn("failed to connect to IMAP, inbound mail disabled", "error", err)
		// Не вызываем os.Exit(1)
	} else {
		defer func() {
			if err := mailReader.Disconnect(); err != nil {
				logger.Error("failed to disconnect IMAP", "error", err)
			}
		}()
	}

	// Email Sender
	emailSender := sender.NewSender(sender.Config{
		Server:   cfg.SMTP.Server,
		Port:     cfg.SMTP.Port,
		User:     cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		TLS:      cfg.SMTP.TLS,
	}, logger)

	// Health сервер
	healthSrv := health.NewServer(cfg.Webhook.Listen)

	// Webhook Handler
	whHandler := webhook.NewHandler(st, cfg.Webhook.Secret, logger)

	// Health-проверки
	healthSrv.Register(health.NewNamedCheck("config", func(_ context.Context) error {
		return nil
	}))
	healthSrv.Register(health.NewNamedCheck("database", func(ctx context.Context) error {
		return st.Ping(ctx)
	}))

	// Монтируем webhook на /webhook
	mux := http.NewServeMux()
	mux.Handle("/health", healthSrv.Handler())
	mux.Handle("/ready", healthSrv.Handler())
	mux.Handle("/metrics", healthSrv.Handler())
	mux.Handle("/webhook", whHandler)

	// Inbound Worker
	inboundWorker := worker.NewInboundWorker(mailReader, proc, cfg.IMAP.ScanInterval, logger)

	// Outbound Worker
	outboundWorker := worker.NewOutboundWorker(st, emailSender, 15*time.Second, logger)

	// Контекст с graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// HTTP-сервер
	httpServer := &http.Server{
		Addr:    cfg.Webhook.Listen,
		Handler: mux,
	}

	// Запускаем HTTP 1
	go func() {
		logger.Info("http server listening", "addr", cfg.Webhook.Listen)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
		}
	}()

	// Запускаем воркеры
	go func() {
		if err := inboundWorker.Start(ctx); err != nil {
			logger.Error("inbound worker error", "error", err)
		}
	}()

	go func() {
		if err := outboundWorker.Start(ctx); err != nil {
			logger.Error("outbound worker error", "error", err)
		}
	}()

	logger.Info("mailbridge started",
		"imap", fmt.Sprintf("%s:%d", cfg.IMAP.Server, cfg.IMAP.Port),
		"smtp", fmt.Sprintf("%s:%d", cfg.SMTP.Server, cfg.SMTP.Port),
		"plane", cfg.Plane.BaseURL,
		"scan_interval", cfg.IMAP.ScanInterval,
	)

	// Ждём сигнал
	sig := <-sigCh
	logger.Info("received signal, shutting down", "signal", sig)
	cancel()

	// Graceful shutdown HTTP
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", "error", err)
	}

	// Останавливаем воркеры
	if err := inboundWorker.Stop(shutdownCtx); err != nil {
		logger.Error("failed to stop inbound worker", "error", err)
	}
	if err := outboundWorker.Stop(shutdownCtx); err != nil {
		logger.Error("failed to stop outbound worker", "error", err)
	}

	logger.Info("mailbridge stopped")
}

// loadProjectMap загружает проекты из Plane и строит карту имя → UUID.
func loadProjectMap(client *plane.Client, logger *slog.Logger) map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projects, err := client.GetProjects(ctx)
	if err != nil {
		logger.Warn("failed to load projects from Plane, project mapping disabled", "error", err)
		return nil
	}

	projectMap := make(map[string]string, len(projects))
	for _, p := range projects {
		projectMap[p.Name] = p.ID
		logger.Debug("mapped project", "name", p.Name, "id", p.ID)
	}

	return projectMap
}
