package ai

import (
	"context"
	"log"

	"github.com/audetv/mailbridge/internal/store"
)

// AIQueue представляет очередь обработки входящих через LLM.
type Queue struct {
	channel chan int64 // IDs inbox_items
	store   store.Store
}

// NewQueue создаёт новую очередь.
func NewQueue(st store.Store, bufferSize int) *Queue {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	return &Queue{
		channel: make(chan int64, bufferSize),
		store:   st,
	}
}

// Enqueue добавляет ID входящего в очередь.
func (q *Queue) Enqueue(inboxItemID int64) {
	select {
	case q.channel <- inboxItemID:
	default:
		log.Printf("[AIQueue] buffer full, dropping %d", inboxItemID)
	}
}

// Channel возвращает канал очереди.
func (q *Queue) Channel() <-chan int64 {
	return q.channel
}

// LoadPending загружает необработанные входящие из БД при старте.
func (q *Queue) LoadPending(ctx context.Context) error {
	items, err := q.store.GetPendingAIItems(ctx)
	if err != nil {
		return err
	}

	for _, item := range items {
		q.Enqueue(item.ID)
	}

	log.Printf("[AIQueue] loaded %d pending items", len(items))
	return nil
}
