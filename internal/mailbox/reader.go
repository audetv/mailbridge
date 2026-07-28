// Package mailbox предоставляет IMAP-клиент для чтения почты.
package mailbox

import (
	"context"
	"fmt"
	"log/slog"

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
	config Config
	logger *slog.Logger
	client *imapclient.Client
}

// NewReader создаёт новый Reader.
func NewReader(cfg Config, logger *slog.Logger) *Reader {
	return &Reader{
		config: cfg,
		logger: logger,
	}
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
		return fmt.Errorf("failed to connect to IMAP: %w", err)
	}

	if err := r.client.Login(r.config.User, r.config.Password).Wait(); err != nil {
		r.client.Close()
		return fmt.Errorf("failed to login: %w", err)
	}

	r.logger.Info("connected to IMAP", "server", addr, "user", r.config.User)

	if err := r.ensureFolder(r.config.Archive); err != nil {
		r.logger.Warn("failed to ensure archive folder", "folder", r.config.Archive, "error", err)
	}
	if err := r.ensureFolder(r.config.Errors); err != nil {
		r.logger.Warn("failed to ensure errors folder", "folder", r.config.Errors, "error", err)
	}

	return nil
}

// FetchUnseen возвращает непрочитанные письма из inbox.
func (r *Reader) FetchUnseen(_ context.Context) ([]*RawEmail, error) {
	if r.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	_, err := r.client.Select(r.config.Inbox, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select inbox: %w", err)
	}

	criteria := imap.SearchCriteria{
		NotFlag: []imap.Flag{imap.FlagSeen},
	}

	searchData, err := r.client.UIDSearch(&criteria, nil).Wait()
	if err != nil {
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

	fetchCmd := r.client.Fetch(uidSet, fetchOptions)
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
	if r.client == nil {
		return fmt.Errorf("not connected")
	}

	uidSet := imap.UIDSetNum(uid)

	_, err := r.client.Copy(uidSet, folder).Wait()
	if err != nil {
		return fmt.Errorf("failed to copy to %s: %w", folder, err)
	}

	delFlags := imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}
	if err := r.client.Store(uidSet, &delFlags, nil).Close(); err != nil {
		return fmt.Errorf("failed to store delete flags: %w", err)
	}

	expungeCmd := r.client.Expunge()
	_, _ = expungeCmd.Collect()

	return nil
}

func (r *Reader) ensureFolder(folder string) error {
	if r.client == nil {
		return fmt.Errorf("not connected")
	}

	_, err := r.client.Select(folder, nil).Wait()
	if err == nil {
		return nil
	}

	return r.client.Create(folder, nil).Wait()
}

// Disconnect закрывает соединение.
func (r *Reader) Disconnect() error {
	if r.client != nil {
		return r.client.Logout().Wait()
	}
	return nil
}
