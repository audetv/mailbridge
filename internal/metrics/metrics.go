// Package metrics предоставляет метрики приложения для Prometheus.
package metrics

import (
	"fmt"
	"sync/atomic"
)

// Metrics содержит счётчики и состояния приложения.
type Metrics struct {
	EmailsProcessed atomic.Int64
	IssuesCreated   atomic.Int64
	CommentsAdded   atomic.Int64
	EmailsSent      atomic.Int64
	Errors          atomic.Int64

	IMAPConnected  atomic.Int32 // 1 = connected, 0 = disconnected
	PlaneAvailable atomic.Int32 // 1 = available, 0 = unavailable
}

// New создаёт новый экземпляр Metrics.
func New() *Metrics {
	return &Metrics{}
}

// SetIMAPConnected устанавливает состояние IMAP-подключения.
func (m *Metrics) SetIMAPConnected(connected bool) {
	if connected {
		m.IMAPConnected.Store(1)
	} else {
		m.IMAPConnected.Store(0)
	}
}

// SetPlaneAvailable устанавливает доступность Plane API.
func (m *Metrics) SetPlaneAvailable(available bool) {
	if available {
		m.PlaneAvailable.Store(1)
	} else {
		m.PlaneAvailable.Store(0)
	}
}

// PrometheusFormat возвращает метрики в формате Prometheus.
func (m *Metrics) PrometheusFormat() string {
	return fmt.Sprintf(
		`# HELP mailbridge_up Whether the service is up
# TYPE mailbridge_up gauge
mailbridge_up 1
# HELP mailbridge_emails_processed_total Total number of emails processed
# TYPE mailbridge_emails_processed_total counter
mailbridge_emails_processed_total %d
# HELP mailbridge_issues_created_total Total number of issues created
# TYPE mailbridge_issues_created_total counter
mailbridge_issues_created_total %d
# HELP mailbridge_comments_added_total Total number of comments added
# TYPE mailbridge_comments_added_total counter
mailbridge_comments_added_total %d
# HELP mailbridge_emails_sent_total Total number of emails sent
# TYPE mailbridge_emails_sent_total counter
mailbridge_emails_sent_total %d
# HELP mailbridge_errors_total Total number of errors
# TYPE mailbridge_errors_total counter
mailbridge_errors_total %d
# HELP mailbridge_imap_connected IMAP connection status (1=connected)
# TYPE mailbridge_imap_connected gauge
mailbridge_imap_connected %d
# HELP mailbridge_plane_available Plane API availability (1=available)
# TYPE mailbridge_plane_available gauge
mailbridge_plane_available %d
`,
		m.EmailsProcessed.Load(),
		m.IssuesCreated.Load(),
		m.CommentsAdded.Load(),
		m.EmailsSent.Load(),
		m.Errors.Load(),
		m.IMAPConnected.Load(),
		m.PlaneAvailable.Load(),
	)
}
