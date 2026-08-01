package web

import (
	"encoding/json"
	"sync"
)

// TaskEvent представляет событие изменения задачи.
type TaskEvent struct {
	Type   string `json:"type"` // "task_created", "task_updated", "task_comment"
	TaskID int64  `json:"task_id"`
}

// EventBroker реализует pub/sub для событий задач.
type EventBroker struct {
	subscribers map[chan []byte]struct{}
	mu          sync.RWMutex
}

// NewEventBroker создаёт новый EventBroker.
func NewEventBroker() *EventBroker {
	return &EventBroker{
		subscribers: make(map[chan []byte]struct{}),
	}
}

// Subscribe создаёт подписку на события.
func (b *EventBroker) Subscribe() chan []byte {
	ch := make(chan []byte, 10)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe удаляет подписку.
func (b *EventBroker) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	delete(b.subscribers, ch)
	close(ch)
	b.mu.Unlock()
}

// Publish отправляет событие всем подписчикам.
func (b *EventBroker) Publish(event TaskEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.subscribers {
		select {
		case ch <- data:
		default:
			// Пропускаем если канал заполнен
		}
	}
}
