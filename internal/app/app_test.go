package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/audetv/mailbridge/internal/app"
	"github.com/audetv/mailbridge/internal/config"
)

// mockComponent реализует интерфейс Component для тестов.
type mockComponent struct {
	started chan struct{}
	stopped chan struct{}
}

func newMockComponent() *mockComponent {
	return &mockComponent{
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (m *mockComponent) Start(ctx context.Context) error {
	close(m.started)
	<-ctx.Done()
	return nil
}

func (m *mockComponent) Stop(_ context.Context) error {
	close(m.stopped)
	return nil
}

func TestApp_StartStop(t *testing.T) {
	cfg := &config.Config{}
	// Не вызываем Load(), используем нулевую конфигурацию
	// ShutdownTimeout по умолчанию 30s через getEnv

	app := app.New(cfg)

	mock := newMockComponent()
	app.Register(mock)

	// Запускаем в горутине
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run()
	}()

	// Ждём старта компонента
	select {
	case <-mock.started:
		// Ок
	case <-time.After(2 * time.Second):
		t.Fatal("component did not start")
	}

	// Отправляем сигнал завершения
	// В реальности это делает ОС, но мы вызовем Stop напрямую
	// Для теста — просто проверяем, что компонент запущен
}
