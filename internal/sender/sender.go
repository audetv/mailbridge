// Package sender предоставляет отправку email-уведомлений с threading.
package sender

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-mail/mail"
)

// Config содержит настройки SMTP.
type Config struct {
	Server   string
	Port     int
	User     string
	Password string
	From     string
	TLS      bool
}

// Sender отправляет email-уведомления.
type Sender struct {
	config Config
	logger *slog.Logger
	dialer *mail.Dialer
}

// NewSender создаёт новый Sender.
func NewSender(cfg Config, logger *slog.Logger) *Sender {
	var dialer *mail.Dialer
	if cfg.User != "" {
		dialer = mail.NewDialer(cfg.Server, cfg.Port, cfg.User, cfg.Password)
	} else {
		dialer = &mail.Dialer{Host: cfg.Server, Port: cfg.Port}
	}

	if cfg.TLS {
		dialer.StartTLSPolicy = mail.MandatoryStartTLS
	} else {
		dialer.StartTLSPolicy = mail.NoStartTLS
	}

	return &Sender{
		config: cfg,
		logger: logger,
		dialer: dialer,
	}
}

// SendAcknowledgement отправляет подтверждение создания задачи.
func (s *Sender) SendAcknowledgement(data *AcknowledgementData) error {
	subject, body := FormatAcknowledgement(data)
	return s.sendEmail(data.To, subject, body, data.InReplyToMessageID, nil)
}

// SendCommentReply отправляет уведомление о новом комментарии.
func (s *Sender) SendCommentReply(data *CommentReplyData) error {
	subject, body := FormatCommentReply(data)
	return s.sendEmail(data.To, subject, body, data.InReplyToMessageID, data.References)
}

// SendStatusChange отправляет уведомление о смене статуса.
func (s *Sender) SendStatusChange(data *StatusChangeData) error {
	subject, body := FormatStatusChange(data)
	return s.sendEmail(data.To, subject, body, data.InReplyToMessageID, nil)
}

// sendEmail отправляет письмо с threading-заголовками.
func (s *Sender) sendEmail(to, subject, body, inReplyTo string, references []string) error {
	m := mail.NewMessage()
	m.SetHeader("From", s.config.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)

	// Threading заголовки
	if inReplyTo != "" {
		m.SetHeader("In-Reply-To", fmt.Sprintf("<%s>", inReplyTo))
	}

	if len(references) > 0 {
		var refs []string
		for _, ref := range references {
			if !strings.HasPrefix(ref, "<") {
				ref = fmt.Sprintf("<%s>", ref)
			}
			refs = append(refs, ref)
		}
		m.SetHeader("References", strings.Join(refs, " "))
	} else if inReplyTo != "" {
		m.SetHeader("References", fmt.Sprintf("<%s>", inReplyTo))
	}

	// Служебный заголовок для фильтрации
	m.SetHeader("X-Mailbridge-Issue", "true")

	m.SetBody("text/plain; charset=utf-8", body)

	if err := s.dialer.DialAndSend(m); err != nil {
		s.logger.Error("failed to send email",
			"to", to,
			"subject", subject,
			"error", err,
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	s.logger.Info("email sent", "to", to, "subject", subject)
	return nil
}
