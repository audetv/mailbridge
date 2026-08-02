package web

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WSEvent представляет событие, отправляемое через WebSocket.
type WSEvent struct {
	Type     string `json:"type"`
	TaskID   int64  `json:"taskId,omitempty"`
	Username string `json:"username,omitempty"`
	Message  string `json:"message,omitempty"`
	Data     any    `json:"data,omitempty"`
}

// WSMessage представляет входящее сообщение от клиента.
type WSMessage struct {
	Type   string `json:"type"`
	TaskID int64  `json:"taskId,omitempty"`
}

// EventBroker управляет подписками на события WebSocket.
type EventBroker struct {
	subscribers map[string]map[chan WSEvent]struct{}
	mu          sync.RWMutex
}

// NewEventBroker создаёт новый EventBroker.
func NewEventBroker() *EventBroker {
	return &EventBroker{
		subscribers: make(map[string]map[chan WSEvent]struct{}),
	}
}

// Subscribe подписывает пользователя на события.
func (b *EventBroker) Subscribe(username string) chan WSEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan WSEvent, 64)
	if b.subscribers[username] == nil {
		b.subscribers[username] = make(map[chan WSEvent]struct{})
	}
	b.subscribers[username][ch] = struct{}{}

	// Также подписываем на общие события
	if username != "*" {
		if b.subscribers["*"] == nil {
			b.subscribers["*"] = make(map[chan WSEvent]struct{})
		}
		b.subscribers["*"][ch] = struct{}{}
	}

	return ch
}

// Unsubscribe отписывает пользователя.
func (b *EventBroker) Unsubscribe(username string, ch chan WSEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, ok := b.subscribers[username]; ok {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(b.subscribers, username)
		}
	}

	if subs, ok := b.subscribers["*"]; ok {
		delete(subs, ch)
	}
}

// Publish отправляет событие подписчикам.
func (b *EventBroker) Publish(event WSEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Отправляем адресно
	if event.Username != "" {
		for ch := range b.subscribers[event.Username] {
			select {
			case ch <- event:
			default:
			}
		}
	}

	// Отправляем всем
	for ch := range b.subscribers["*"] {
		select {
		case ch <- event:
		default:
		}
	}
}

// WSHandler обрабатывает WebSocket-соединения.
type WSHandler struct {
	broker   *EventBroker
	upgrader websocket.Upgrader
}

// NewWSHandler создаёт новый WSHandler.
func NewWSHandler(broker *EventBroker) *WSHandler {
	return &WSHandler{
		broker: broker,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
}

// ServeHTTP обрабатывает подключение WebSocket.
func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	username := extractUserFromToken(r)
	if username == "" {
		username = "anonymous"
	}

	ch := h.broker.Subscribe(username)
	defer h.broker.Unsubscribe(username, ch)

	// Отправляем приветственное событие
	if err := conn.WriteJSON(WSEvent{Type: "connected", Username: username, Message: "Подключено"}); err != nil {
		return
	}

	// Читаем входящие сообщения в отдельной горутине
	go func() {
		for {
			var msg WSMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			// Обработка команд от клиента
			switch msg.Type {
			case "ping":
				if err := conn.WriteJSON(WSEvent{Type: "pong"}); err != nil {
					return
				}
			}
		}
	}()

	// Отправляем события клиенту
	for event := range ch {
		if err := conn.WriteJSON(event); err != nil {
			return
		}
	}
}
