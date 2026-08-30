package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"log/slog"

	"github.com/audetv/mailbridge/internal/adapters"
	"github.com/audetv/mailbridge/internal/ai"
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
	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
	"github.com/audetv/mailbridge/internal/version"
	"github.com/audetv/mailbridge/internal/web"
	"github.com/audetv/mailbridge/internal/webhook"
	"github.com/audetv/mailbridge/internal/worker"
)

//go:embed static
var staticFiles embed.FS

func main() {
	// Подкоманда version: печатает вшитую версию и выходит без загрузки конфига
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version.Info())
		return
	}

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
	// Контекст с graceful shutdown
	// ---------------------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

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
	// Websocket
	// ---------------------------------------------------------------------------
	broker := web.NewEventBroker()

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

	// ---------------------------------------------------------------------------
	// AI Client (если включён в конфиге)
	// ---------------------------------------------------------------------------
	var aiClient ai.Client
	if cfg.AI.Enabled {
		switch cfg.AI.Provider {
		case "ollama":
			aiClient = ai.NewOllamaClient(cfg.AI.BaseURL, cfg.AI.Model)
		case "openai":
			aiClient = ai.NewOpenAIClient(cfg.AI.BaseURL, cfg.AI.APIKey, cfg.AI.Model)
		}
	}

	var orchestrator *ai.Orchestrator
	if aiClient != nil {
		orchestrator = ai.NewOrchestrator(aiClient, st)
	}
	// Шаг 15: fallback-проект переопределяется через MAILBRIDGE_DEFAULT_PROJECT
	if d := os.Getenv("MAILBRIDGE_DEFAULT_PROJECT"); d != "" {
		ai.DefaultProject = d
	}

	if orchestrator != nil {
		// Фаза 3 шаг 14: проекты для AI берутся из внутренней БД (aktivные), не из Plane.
		orchestrator.SetProjectsProvider(func(ctx context.Context) ([]string, error) {
			archived := false
			projects, err := st.ListProjects(ctx, &store.ProjectFilter{Archived: &archived})
			if err != nil {
				return nil, err
			}
			names := make([]string, 0, len(projects))
			for _, p := range projects {
				if p.Name != "" {
					names = append(names, p.Name)
				}
			}
			// Fallback-проект всегда доступен для AI, даже если БД пуста.
			names = append(names, ai.DefaultProject)
			return names, nil
		})
	}

	var aiQueue *ai.Queue
	var aiWorker *ai.Worker
	if aiClient != nil {
		aiQueue = ai.NewQueue(st, 100)
		aiWorker = ai.NewWorker(aiQueue, orchestrator, st, logger, broker)

		// Загружаем pending при старте
		if err := aiQueue.LoadPending(context.Background()); err != nil {
			logger.Error("failed to load pending AI items", "error", err)
		}

		// Запускаем воркер
		go aiWorker.Start(ctx)
	}

	emailAdapter := adapters.NewEmailAdapter(ext, st, cfg.Attachments.Dir)
	proc := processor.NewMessageProcessor(
		st, cl, ext, par, cfg, logger, projectNameMap, broker, orchestrator, cfg.AI.Enabled, emailAdapter, aiQueue,
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

	// Раздача статики SPA
	distFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		logger.Error("failed to load static files", "error", err)
		os.Exit(1)
	}
	fileServer := http.FileServer(http.FS(distFS))

	// Отдаём index.html для маршрутов Vue Router
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// API-запросы не трогаем
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}

		// Проверяем существует ли файл в embed.FS
		cleanPath := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
		if cleanPath != "" && cleanPath != "." {
			if _, err := fs.Stat(distFS, cleanPath); os.IsNotExist(err) {
				r.URL.Path = "/"
			}
		}

		fileServer.ServeHTTP(w, r)
	})

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

	taskHandler := web.NewTaskHandler(st, broker)
	// Projects API
	projectHandler := web.NewProjectHandler(st, broker)

	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			projectHandler.ListProjects(w, r)
		case http.MethodPost:
			projectHandler.CreateProject(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			projectHandler.GetProject(w, r)
		case http.MethodPut:
			projectHandler.UpdateProject(w, r)
		case http.MethodDelete:
			projectHandler.ArchiveProject(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/projects/{id}/unarchive", projectHandler.UnarchiveProject)
	mux.HandleFunc("/api/projects/{id}/tasks", projectHandler.ListProjectTasks) // Фаза 1 шаг 6: ссылка проекта на задачи

	// Epics API (модули)
	epicHandler := web.NewEpicHandler(st, broker)
	mux.HandleFunc("/api/projects/{id}/epics", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			epicHandler.ListEpicsList(w, r)
		case http.MethodPost:
			epicHandler.CreateEpicList(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/epics/{epic_id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			epicHandler.GetEpicDetailHandler(w, r)
		case http.MethodPut:
			epicHandler.UpdateEpicDetail(w, r)
		case http.MethodDelete:
			epicHandler.DeleteEpicDetail(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/epics/{epic_id}/tasks/{taskId}", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			epicHandler.LinkTaskEpic(w, r)
		case http.MethodDelete:
			epicHandler.UnlinkTaskEpic(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/inbox", taskHandler.ListInbox)
	mux.HandleFunc("/api/inbox/{id}", taskHandler.GetInboxItem)
	mux.HandleFunc("/api/inbox/{id}/attachments", taskHandler.GetInboxAttachments)
	mux.HandleFunc("/api/inbox/{id}/tasks", taskHandler.GetInboxItemTasks)
	mux.HandleFunc("/api/inbox/{id}/read", taskHandler.UpdateInboxStatus)
	mux.HandleFunc("/api/inbox/{id}/unread", taskHandler.UpdateInboxStatus)
	mux.HandleFunc("/api/inbox/{id}/archive", taskHandler.UpdateInboxStatus)
	mux.HandleFunc("/api/inbox/{id}/task", taskHandler.CreateTaskFromInbox)
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			taskHandler.CreateTask(w, r)
			return
		}
		taskHandler.ListTasks(w, r)
	})
	mux.HandleFunc("/api/tasks/{id}/attachments", taskHandler.GetTaskAttachments)
	mux.HandleFunc("/api/tasks/{id}/attachments/{attId}", taskHandler.UnlinkTaskAttachment)
	mux.HandleFunc("/api/tasks/{id}/inbox", taskHandler.GetTaskInboxItems)
	mux.HandleFunc("/api/tasks/{id}/comments/{commentId}/attachments", taskHandler.GetCommentAttachments)
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
	// Помечаем задачу прочитанной
	mux.HandleFunc("/api/tasks/{id}/mark-read", taskHandler.MarkRead)
	// Вложения
	mux.HandleFunc("/api/attachments/{path...}", taskHandler.GetAttachment)

	// WebSocket
	wsHandler := web.NewWSHandler(broker)
	mux.Handle("/api/ws", wsHandler)

	// Webhook
	whHandler := webhook.NewHandler(st, cfg.Webhook.Secret, logger)
	mux.Handle("/webhook", whHandler)

	// ---------------------------------------------------------------------------
	// Воркеры
	// ---------------------------------------------------------------------------
	inboundWorker := worker.NewInboundWorker(mailReader, proc, cfg.IMAP.ScanInterval, logger)
	outboundWorker := worker.NewOutboundWorker(st, emailSender, 15*time.Second, logger)

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
