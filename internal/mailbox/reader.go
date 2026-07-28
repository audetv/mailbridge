// Package mailbox предоставляет IMAP-клиент для чтения почты.
package mailbox

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// Config содержит настройки подключения к IMAP.
type Config struct {
	Server   string
	Port     int
	User     string
	Password string
	TLS      bool
	Inbox    string
	Archive  string
	Errors   string
}

// RawEmail содержит сырое письмо и его UID.
type RawEmail struct {
	UID  imap.UID
	Data []byte
}

// Reader читает письма из IMAP-ящика.
type Reader struct {
	config    Config
	logger    *slog.Logger
	client    *imapclient.Client
	mu        sync.RWMutex
	connected bool
}

// NewReader создаёт новый Reader.
func NewReader(cfg Config, logger *slog.Logger) *Reader {
	return &Reader{
		config: cfg,
		logger: logger,
	}
}

// IsConnected возвращает текущее состояние подключения.
func (r *Reader) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.connected
}

// setConnected обновляет состояние подключения.
func (r *Reader) setConnected(state bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connected = state
}

// Connect устанавливает соединение с IMAP-сервером.
func (r *Reader) Connect() error {
	addr := fmt.Sprintf("%s:%d", r.config.Server, r.config.Port)

	var err error
	if r.config.TLS {
		r.client, err = imapclient.DialTLS(addr, nil)
	} else {
		r.client, err = imapclient.DialStartTLS(addr, nil)
	}
	if err != nil {
		r.setConnected(false)
		return fmt.Errorf("failed to connect to IMAP: %w", err)
	}

	if err := r.client.Login(r.config.User, r.config.Password).Wait(); err != nil {
		r.client.Close()
		r.setConnected(false)
		return fmt.Errorf("failed to login: %w", err)
	}

	r.setConnected(true)
	r.logger.Info("connected to IMAP", "server", addr, "user", r.config.User)

	if err := r.ensureFolder(r.config.Archive); err != nil {
		r.logger.Warn("failed to ensure archive folder", "folder", r.config.Archive, "error", err)
	}
	if err := r.ensureFolder(r.config.Errors); err != nil {
		r.logger.Warn("failed to ensure errors folder", "folder", r.config.Errors, "error", err)
	}

	return nil
}

// Reconnect пытается переподключиться с exponential backoff.
func (r *Reader) Reconnect() error {
	r.mu.Lock()
	if r.client != nil {
		r.client.Close()
		r.client = nil
	}
	r.connected = false
	r.mu.Unlock()

	backoff := 1 * time.Second
	maxBackoff := 5 * time.Minute
	attempts := 0

	for {
		attempts++
		r.logger.Info("attempting IMAP reconnect", "attempt", attempts, "backoff", backoff)

		if err := r.Connect(); err != nil {
			r.logger.Warn("IMAP reconnect failed", "attempt", attempts, "error", err)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		r.logger.Info("IMAP reconnected", "attempts", attempts)
		return nil
	}
}

// FetchUnseen возвращает непрочитанные письма из inbox.
func (r *Reader) FetchUnseen(_ context.Context) ([]*RawEmail, error) {
	r.mu.RLock()
	client := r.client
	connected := r.connected
	r.mu.RUnlock()

	if client == nil || !connected {
		return nil, fmt.Errorf("not connected")
	}

	_, err := client.Select(r.config.Inbox, nil).Wait()
	if err != nil {
		r.setConnected(false)
		return nil, fmt.Errorf("failed to select inbox: %w", err)
	}

	criteria := imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen},
	}

	searchData, err := client.UIDSearch(&criteria, nil).Wait()
	if err != nil {
		r.setConnected(false)
		return nil, fmt.Errorf("failed to search unseen: %w", err)
	}

	uids := searchData.AllUIDs()
	if len(uids) == 0 {
		return nil, nil
	}

	r.logger.Info("found unseen messages", "count", len(uids))

	uidSet := imap.UIDSetNum(uids...)

	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{{}},
	}

	fetchCmd := client.Fetch(uidSet, fetchOptions)
	defer fetchCmd.Close()

	var rawEmails []*RawEmail
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		buf, err := msg.Collect()
		if err != nil {
			r.logger.Error("failed to collect message", "error", err)
			continue
		}

		for _, section := range buf.BodySection {
			rawEmails = append(rawEmails, &RawEmail{
				UID:  buf.UID,
				Data: section.Bytes,
			})
		}
	}

	return rawEmails, nil
}

// MarkProcessed помечает письмо как обработанное и перемещает в Archive.
func (r *Reader) MarkProcessed(uid imap.UID) error {
	return r.moveMessage(uid, r.config.Archive)
}

// MarkErrored помечает письмо как ошибочное и перемещает в Errors.
func (r *Reader) MarkErrored(uid imap.UID) error {
	return r.moveMessage(uid, r.config.Errors)
}

func (r *Reader) moveMessage(uid imap.UID, folder string) error {
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("not connected")
	}

	uidSet := imap.UIDSetNum(uid)

	_, err := client.Copy(uidSet, folder).Wait()
	if err != nil {
		return fmt.Errorf("failed to copy to %s: %w", folder, err)
	}

	delFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	if err := client.Store(uidSet, &delFlags, nil).Close(); err != nil {
		return fmt.Errorf("failed to store delete flags: %w", err)
	}

	expungeCmd := client.Expunge()
	_, _ = expungeCmd.Collect()

	return nil
}

func (r *Reader) ensureFolder(folder string) error {
	r.mu.RLock()
	client := r.client
	r.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("not connected")
	}

	_, err := client.Select(folder, nil).Wait()
	if err == nil {
		return nil
	}

	return client.Create(folder, nil).Wait()
}

// Disconnect закрывает соединение.
func (r *Reader) Disconnect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.connected = false
	if r.client != nil {
		return r.client.Logout().Wait()
	}
	return nil
}
