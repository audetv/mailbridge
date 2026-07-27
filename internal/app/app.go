// Package app управляет жизненным циклом приложения Mailbridge.
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/audetv/mailbridge/internal/config"
)

// Component описывает компонент с жизненным циклом.
type Component interface {
	// Start запускает компонент. Должен блокироваться до ошибки или отмены ctx.
	Start(ctx context.Context) error
	// Stop останавливает компонент с таймаутом из ctx.
	Stop(ctx context.Context) error
}

// App представляет приложение Mailbridge.
type App struct {
	cfg        *config.Config
	components []Component
	cancel     context.CancelFunc
}

// New создаёт новый экземпляр App.
func New(cfg *config.Config) *App {
	return &App{
		cfg:        cfg,
		components: make([]Component, 0),
	}
}

// Register добавляет компонент в список управляемых.
func (a *App) Register(c Component) {
	a.components = append(a.components, c)
}

// Run запускает приложение и ждёт сигналов завершения.
func (a *App) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	defer cancel()

	// Канал для сигналов ОС
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Канал для ошибок компонентов
	errCh := make(chan error, len(a.components))

	// Запускаем все компоненты
	for _, c := range a.components {
		go func(component Component) {
			if err := component.Start(ctx); err != nil {
				errCh <- fmt.Errorf("component start error: %w", err)
			}
		}(c)
	}

	// Ждём сигнал или ошибку
	select {
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "received signal: %v, shutting down...\n", sig)
		cancel()
	case err := <-errCh:
		fmt.Fprintf(os.Stderr, "component error: %v, shutting down...\n", err)
		cancel()
	}

	// Останавливаем компоненты в обратном порядке
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout())
	defer shutdownCancel()

	var shutdownErr error
	for i := len(a.components) - 1; i >= 0; i-- {
		if err := a.components[i].Stop(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error stopping component: %v\n", err)
			shutdownErr = err
		}
	}

	return shutdownErr
}

// Config возвращает конфигурацию приложения.
func (a *App) Config() *config.Config {
	return a.cfg
}

// Shutdown инициирует корректное завершение приложения.
func (a *App) Shutdown() {
	a.cancel()
}
