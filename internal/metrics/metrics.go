// Package metrics предоставляет метрики приложения для Prometheus.
package metrics

import (
	"fmt"
	"sync/atomic"
)

// Metrics содержит счётчики приложения.
type Metrics struct {
	EmailsProcessed atomic.Int64
	IssuesCreated   atomic.Int64
	CommentsAdded   atomic.Int64
	EmailsSent      atomic.Int64
	Errors          atomic.Int64
}

// New создаёт новый экземпляр Metrics.
func New() *Metrics {
	return &Metrics{}
}

// PrometheusFormat возвращает метрики в формате Prometheus.
func (m *Metrics) PrometheusFormat() string {
	return fmt.Sprintf(
		`# HELP mailbridge_emails_processed_total Total number of emails processed
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
# HELP mailbridge_up Whether the service is up
# TYPE mailbridge_up gauge
mailbridge_up 1
`, m.EmailsProcessed.Load(),
		m.IssuesCreated.Load(),
		m.CommentsAdded.Load(),
		m.EmailsSent.Load(),
		m.Errors.Load(),
	)
}
