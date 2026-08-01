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
	"github.com/audetv/mailbridge/internal/metrics"
	"github.com/audetv/mailbridge/internal/parser"
	"github.com/audetv/mailbridge/internal/plane"
	"github.com/audetv/mailbridge/internal/processor"
	"github.com/audetv/mailbridge/internal/sender"
	"github.com/audetv/mailbridge/internal/store/sqlite"
	"github.com/audetv/mailbridge/internal/version"
	"github.com/audetv/mailbridge/internal/web"
	"github.com/audetv/mailbridge/internal/webhook"
	"github.com/audetv/mailbridge/internal/worker"
)

func main() {
	fmt.Println(version.Info())

	// ---------------------------------------------------------------------------
	// Конфигурация
	// ---------------------------------------------------------------------------
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

	rulesCfg, err := config.LoadRules(cfg.NLP.RulesFile)
	if err != nil {
		logger.Error("failed to load rules", "error", err)
		os.Exit(1)
	}
	logger.Info("rules loaded",
		"file", cfg.NLP.RulesFile,
		"rules_count", len(rulesCfg.Rules),
	)

	// ---------------------------------------------------------------------------
	// Хранилище
	// ---------------------------------------------------------------------------
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

	attStore, err := extractor.NewAttachmentStore(cfg.Attachments.Dir)
	if err != nil {
		logger.Error("failed to create attachment store", "error", err)
		os.Exit(1)
	}

	// ---------------------------------------------------------------------------
	// Метрики
	// ---------------------------------------------------------------------------
	m := metrics.New()

	// ---------------------------------------------------------------------------
	// Plane Client
	// ---------------------------------------------------------------------------
	planeClient := plane.NewClient(cfg.Plane.BaseURL, cfg.Plane.APIKey)
	projectMap := loadProjectMap(planeClient, logger)
	logger.Info("projects loaded from Plane", "count", len(projectMap))

	// ---------------------------------------------------------------------------
	// NLP и классификация
	// ---------------------------------------------------------------------------
	rules := classifier.ConvertRules(rulesCfg.Rules)
	cl := classifier.NewRuleBasedClassifier(
		rules,
		rulesCfg.UrgencyPatterns,
		rulesCfg.ValidTypesMap(),
		rulesCfg.ValidPrioritiesMap(),
	)

	par := parser.NewFieldParser(
		rulesCfg.ValidTypesMap(),
		rulesCfg.ValidPrioritiesMap(),
	)

	// ---------------------------------------------------------------------------
	// Extractor и Processor
	// ---------------------------------------------------------------------------
	ext := extractor.NewExtractor(attStore)

	// Преобразуем projectMap в map[string]string (имя → идентификатор)
	projectNameMap := make(map[string]string, len(projectMap))
	for name, proj := range projectMap {
		projectNameMap[name] = proj.Identifier
	}
	if len(projectNameMap) == 0 {
		projectNameMap["Входящие"] = "INBOX"
	}

	proc := processor.NewMessageProcessor(
		st, cl, ext, par, cfg, logger, projectNameMap,
	)

	// ---------------------------------------------------------------------------
	// Email Sender
	// ---------------------------------------------------------------------------
	emailSender := sender.NewSender(sender.Config{
		Server:   cfg.SMTP.Server,
		Port:     cfg.SMTP.Port,
		User:     cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
		TLS:      cfg.SMTP.TLS,
	}, logger)

	// ---------------------------------------------------------------------------
	// Mail Reader
	// ---------------------------------------------------------------------------
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

	imapOk := false
	if err := mailReader.Connect(); err != nil {
		logger.Warn("failed to connect to IMAP, inbound mail disabled", "error", err)
	} else {
		imapOk = true
		defer func() {
			if err := mailReader.Disconnect(); err != nil {
				logger.Error("failed to disconnect IMAP", "error", err)
			}
		}()
	}

	// ---------------------------------------------------------------------------
	// Health-проверки
	// ---------------------------------------------------------------------------
	healthSrv := health.NewServer(cfg.Webhook.Listen)

	healthSrv.Register(health.NewNamedCheck("database", func(ctx context.Context) error {
		return st.Ping(ctx)
	}))

	healthSrv.Register(health.NewNamedCheck("imap", func(_ context.Context) error {
		m.SetIMAPConnected(imapOk && mailReader.IsConnected())
		if !mailReader.IsConnected() {
			return fmt.Errorf("imap not connected")
		}
		return nil
	}))

	healthSrv.Register(health.NewNamedCheck("plane", func(ctx context.Context) error {
		reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_, err := planeClient.GetProjects(reqCtx)
		m.SetPlaneAvailable(err == nil)
		return err
	}))

	// ---------------------------------------------------------------------------
	// HTTP-сервер (health + webhook + metrics)
	// ---------------------------------------------------------------------------
	mux := http.NewServeMux()

	// Health
	mux.Handle("/health", healthSrv.Handler())
	mux.Handle("/ready", healthSrv.Handler())

	// Metrics
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		m.SetIMAPConnected(imapOk && mailReader.IsConnected())
		_, _ = w.Write([]byte(m.PrometheusFormat()))
	})

	// Auth API
	authHandler := web.NewAuthHandler()
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/me", authHandler.Me)

	taskHandler := web.NewTaskHandler(st)
	mux.HandleFunc("/api/tasks", taskHandler.ListTasks)
	mux.HandleFunc("/api/tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			taskHandler.GetTask(w, r)
		case http.MethodPatch:
			taskHandler.UpdateTask(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/tasks/{id}/reply", taskHandler.ReplyTask)

	// Webhook
	whHandler := webhook.NewHandler(st, cfg.Webhook.Secret, logger)
	mux.Handle("/webhook", whHandler)

	// ---------------------------------------------------------------------------
	// Воркеры
	// ---------------------------------------------------------------------------
	inboundWorker := worker.NewInboundWorker(mailReader, proc, cfg.IMAP.ScanInterval, logger)
	outboundWorker := worker.NewOutboundWorker(st, emailSender, 15*time.Second, logger)

	// ---------------------------------------------------------------------------
	// Контекст с graceful shutdown
	// ---------------------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// ---------------------------------------------------------------------------
	// Запуск HTTP
	// ---------------------------------------------------------------------------
	httpServer := &http.Server{Addr: cfg.Webhook.Listen, Handler: mux}

	go func() {
		logger.Info("http server listening", "addr", cfg.Webhook.Listen)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", "error", err)
		}
	}()

	// ---------------------------------------------------------------------------
	// Запуск воркеров
	// ---------------------------------------------------------------------------
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
		"imap_connected", imapOk,
		"imap_addr", fmt.Sprintf("%s:%d", cfg.IMAP.Server, cfg.IMAP.Port),
		"smtp_addr", fmt.Sprintf("%s:%d", cfg.SMTP.Server, cfg.SMTP.Port),
		"plane_url", cfg.Plane.BaseURL,
		"scan_interval", cfg.IMAP.ScanInterval,
	)

	// ---------------------------------------------------------------------------
	// Ожидание сигнала
	// ---------------------------------------------------------------------------
	sig := <-sigCh
	logger.Info("received signal, shutting down", "signal", sig)
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http server shutdown error", "error", err)
	}

	if err := inboundWorker.Stop(shutdownCtx); err != nil {
		logger.Error("failed to stop inbound worker", "error", err)
	}
	if err := outboundWorker.Stop(shutdownCtx); err != nil {
		logger.Error("failed to stop outbound worker", "error", err)
	}

	logger.Info("mailbridge stopped")
}

// loadProjectMap загружает проекты из Plane и строит карту имя → UUID.
func loadProjectMap(client *plane.Client, logger *slog.Logger) map[string]*plane.Project {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projects, err := client.GetProjects(ctx)
	if err != nil {
		logger.Warn("failed to load projects from Plane, project mapping disabled", "error", err)
		return nil
	}

	projectMap := make(map[string]*plane.Project, len(projects))
	for i := range projects {
		projectMap[projects[i].Name] = &projects[i]
		logger.Debug("mapped project", "name", projects[i].Name, "id", projects[i].ID, "identifier", projects[i].Identifier)
	}

	return projectMap
}
