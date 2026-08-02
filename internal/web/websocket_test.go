package web_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/audetv/mailbridge/internal/web"
	"github.com/gorilla/websocket"
)

func TestEventBroker_SubscribePublish(t *testing.T) {
	broker := web.NewEventBroker()
	ch := broker.Subscribe("testuser")
	defer broker.Unsubscribe("testuser", ch)

	event := web.WSEvent{
		Type:     "task_created",
		TaskID:   42,
		Username: "testuser",
		Message:  "Новая задача",
	}

	broker.Publish(event)

	select {
	case received := <-ch:
		if received.Type != "task_created" {
			t.Errorf("Type = %s, want task_created", received.Type)
		}
		if received.TaskID != 42 {
			t.Errorf("TaskID = %d, want 42", received.TaskID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestEventBroker_Broadcast(t *testing.T) {
	broker := web.NewEventBroker()
	ch1 := broker.Subscribe("user1")
	ch2 := broker.Subscribe("user2")
	defer broker.Unsubscribe("user1", ch1)
	defer broker.Unsubscribe("user2", ch2)

	event := web.WSEvent{
		Type:    "broadcast",
		Message: "Всем",
	}

	broker.Publish(event)

	// Оба должны получить (через подписку "*")
	select {
	case <-ch1:
	case <-time.After(1 * time.Second):
		t.Error("user1 did not receive broadcast")
	}

	select {
	case <-ch2:
	case <-time.After(1 * time.Second):
		t.Error("user2 did not receive broadcast")
	}
}

func TestEventBroker_AddressEvent(t *testing.T) {
	broker := web.NewEventBroker()
	ch1 := broker.Subscribe("user1")
	ch2 := broker.Subscribe("user2")
	defer broker.Unsubscribe("user1", ch1)
	defer broker.Unsubscribe("user2", ch2)

	event := web.WSEvent{
		Type:     "assigned",
		TaskID:   1,
		Username: "user1",
		Message:  "Только user1",
	}

	broker.Publish(event)

	// user1 должен получить и через адресную и через broadcast
	select {
	case <-ch1:
	case <-time.After(1 * time.Second):
		t.Error("user1 did not receive addressed event")
	}

	// user2 тоже получит через broadcast ("*")
	select {
	case <-ch2:
	case <-time.After(1 * time.Second):
		t.Error("user2 did not receive broadcast event")
	}
}

func TestEventBroker_Unsubscribe(t *testing.T) {
	broker := web.NewEventBroker()
	ch := broker.Subscribe("testuser")
	broker.Unsubscribe("testuser", ch)

	event := web.WSEvent{Type: "test"}
	broker.Publish(event)

	select {
	case <-ch:
		t.Error("received event after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// Ожидаемо — события нет
	}
}

func TestEventBroker_BufferedChannel(t *testing.T) {
	broker := web.NewEventBroker()
	ch := broker.Subscribe("testuser")
	defer broker.Unsubscribe("testuser", ch)

	// Отправляем больше событий чем размер буфера
	for i := 0; i < 70; i++ {
		broker.Publish(web.WSEvent{Type: "test", TaskID: int64(i)})
	}

	// Первые 64 должны быть в буфере, остальные отброшены
	time.Sleep(100 * time.Millisecond)

	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:

	if count < 64 {
		t.Errorf("expected at least 64 events, got %d", count)
	}
}

func TestWSHandler_Connect(t *testing.T) {
	broker := web.NewEventBroker()
	handler := web.NewWSHandler(broker)

	srv := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial error: %v", err)
	}
	defer conn.Close()

	// Читаем приветственное событие
	var event web.WSEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("ReadJSON error: %v", err)
	}

	if event.Type != "connected" {
		t.Errorf("expected 'connected', got %s", event.Type)
	}
	if event.Username != "anonymous" {
		t.Errorf("Username = %s, want anonymous", event.Username)
	}
}

func TestWSHandler_PingPong(t *testing.T) {
	broker := web.NewEventBroker()
	handler := web.NewWSHandler(broker)

	srv := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial error: %v", err)
	}
	defer conn.Close()

	// Пропускаем connected
	if err := conn.ReadJSON(&web.WSEvent{}); err != nil {
		return
	}

	// Отправляем ping
	msg := web.WSMessage{Type: "ping"}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	// Читаем pong
	var event web.WSEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("ReadJSON error: %v", err)
	}

	if event.Type != "pong" {
		t.Errorf("expected 'pong', got %s", event.Type)
	}
}

func TestWSHandler_ReceiveBroadcast(t *testing.T) {
	broker := web.NewEventBroker()
	handler := web.NewWSHandler(broker)

	srv := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial error: %v", err)
	}
	defer conn.Close()

	// Пропускаем connected
	if err := conn.ReadJSON(&web.WSEvent{}); err != nil {
		return
	}

	// Публикуем событие
	broker.Publish(web.WSEvent{Type: "test_event", Message: "Hello"})

	var event web.WSEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("ReadJSON error: %v", err)
	}

	if event.Type != "test_event" {
		t.Errorf("Type = %s, want test_event", event.Type)
	}
}

func TestWSHandler_JSONEncoding(t *testing.T) {
	event := web.WSEvent{
		Type:     "task_created",
		TaskID:   42,
		Username: "testuser",
		Message:  "Новая задача #42",
		Data:     map[string]string{"key": "value"},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded web.WSEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Type != "task_created" {
		t.Errorf("Type = %s", decoded.Type)
	}
	if decoded.TaskID != 42 {
		t.Errorf("TaskID = %d", decoded.TaskID)
	}
}
