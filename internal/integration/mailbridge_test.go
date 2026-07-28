// Package integration содержит сквозные тесты Mailbridge.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"strings"
	"testing"
	"time"

	"github.com/audetv/mailbridge/internal/config"
	"github.com/audetv/mailbridge/internal/plane"
	"github.com/audetv/mailbridge/internal/sender"
	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
)

// setupPlaneMock создаёт мок-сервер Plane API.
func setupPlaneMock(t *testing.T) (*httptest.Server, *plane.Client) {
	t.Helper()

	var createdIssues []plane.Issue
	var comments []plane.Comment

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// GET /api/workspaces/{workspace}/projects/
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/projects") && !strings.Contains(r.URL.Path, "/issues") {
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"results": []plane.Project{
					{ID: "proj-incoming", Name: "Входящие"},
					{ID: "proj-trk", Name: "ТРК"},
					{ID: "proj-hotel", Name: "Отель"},
				},
			}); err != nil {
				t.Errorf("encode error: %v", err)
			}
			return
		}

		// POST /api/workspaces/{workspace}/projects/{id}/issues/
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/projects/") && strings.Contains(r.URL.Path, "/issues/") {
			var req plane.CreateIssueRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode error: %v", err)
			}

			issue := plane.Issue{
				ID:         fmt.Sprintf("issue-%d", len(createdIssues)+1),
				SequenceID: fmt.Sprintf("WEB-%d", len(createdIssues)+1),
				ProjectID:  req.ProjectID,
				Name:       req.Name,
				Priority:   req.Priority,
				Labels:     req.Labels,
				CreatedAt:  time.Now(),
			}
			createdIssues = append(createdIssues, issue)

			if err := json.NewEncoder(w).Encode(issue); err != nil {
				t.Errorf("encode error: %v", err)
			}
			return
		}

		// POST /api/workspaces/{workspace}/issues/{id}/comments/
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/") && strings.Contains(r.URL.Path, "/comments/") {
			var req map[string]string
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode error: %v", err)
			}

			comment := plane.Comment{
				ID:        fmt.Sprintf("comment-%d", len(comments)+1),
				Body:      req["comment_html"],
				CreatedAt: time.Now(),
			}
			comments = append(comments, comment)

			if err := json.NewEncoder(w).Encode(comment); err != nil {
				t.Errorf("encode error: %v", err)
			}
			return
		}

		// GET /api/workspaces/{workspace}/issues/{id}/
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/") && !strings.Contains(r.URL.Path, "/comments/") {
			parts := strings.Split(strings.TrimRight(r.URL.Path, "/"), "/")
			id := parts[len(parts)-1]

			for _, issue := range createdIssues {
				if issue.ID == id || issue.SequenceID == id {
					if err := json.NewEncoder(w).Encode(issue); err != nil {
						t.Errorf("encode error: %v", err)
					}
					return
				}
			}

			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	client := plane.NewClient(srv.URL+"/test-workspace", "test-key")
	return srv, client
}

func TestFullCycle_EmailToIssue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	st, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore error: %v", err)
	}
	defer st.Close()
	_ = st.Migrate(context.Background())

	err = st.SaveMapping(context.Background(), &store.EmailMapping{
		MessageID:        "test-msg-1@mailbridge",
		PlaneIssueID:     "issue-1",
		PlaneIssueSeq:    "WEB-1",
		OriginalFrom:     "user@test.local",
		OriginalSubject:  "Test Subject",
		ThreadReferences: []string{"thread-1"},
		ActionType:       "CREATE",
	})
	if err != nil {
		t.Fatalf("SaveMapping error: %v", err)
	}

	mapping, err := st.GetMappingByMessageID(context.Background(), "test-msg-1@mailbridge")
	if err != nil {
		t.Fatalf("GetMappingByMessageID error: %v", err)
	}
	if mapping == nil {
		t.Fatal("mapping not found")
	}
	if mapping.PlaneIssueSeq != "WEB-1" {
		t.Errorf("PlaneIssueSeq = %s, want WEB-1", mapping.PlaneIssueSeq)
	}
}

func TestSMTPConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	addr := "localhost:3025"
	client, err := smtp.Dial(addr)
	if err != nil {
		t.Skipf("greenmail not available: %v", err)
	}
	defer client.Close()

	if err := client.Noop(); err != nil {
		t.Errorf("SMTP noop failed: %v", err)
	}
}

func TestPlaneMock(t *testing.T) {
	srv, client := setupPlaneMock(t)
	defer srv.Close()

	projects, err := client.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("GetProjects error: %v", err)
	}
	if len(projects) != 3 {
		t.Errorf("expected 3 projects, got %d", len(projects))
	}

	issue, err := client.CreateIssue(context.Background(), &plane.CreateIssueRequest{
		ProjectID: "proj-trk",
		Name:      "Не работает кабинет арендатора",
		Priority:  "high",
		Labels:    []string{"bug", "urgent"},
	})
	if err != nil {
		t.Fatalf("CreateIssue error: %v", err)
	}
	if issue.ID == "" {
		t.Error("issue ID is empty")
	}

	comment, err := client.AddComment(context.Background(), issue.ID, "Проверил, проблема с API")
	if err != nil {
		t.Fatalf("AddComment error: %v", err)
	}
	if comment.ID == "" {
		t.Error("comment ID is empty")
	}
}

func TestSender_Format(t *testing.T) {
	_, body := sender.FormatAcknowledgement(&sender.AcknowledgementData{
		To:                 "user@test.local",
		Subject:            "Не работает сайт",
		InReplyToMessageID: "original-msg",
		IssueSequence:      "WEB-123",
		ProjectName:        "ТРК",
		TypeName:           "bug",
		Priority:           "high",
	})
	if !strings.Contains(body, "MAILBRIDGE-INTERNAL") {
		t.Error("body should contain internal marker")
	}

	_, body = sender.FormatCommentReply(&sender.CommentReplyData{
		To:                 "user@test.local",
		Subject:            "Re: Не работает сайт",
		InReplyToMessageID: "original-msg",
		IssueSequence:      "WEB-123",
		CommentText:        "Проверил, проблема с сертификатом",
		CommentAuthor:      "Руслан",
	})
	if !strings.Contains(body, "Руслан") {
		t.Error("body should contain author name")
	}
}

func TestWebhookPayload(t *testing.T) {
	payload := map[string]interface{}{
		"event": "issue.comment.created",
		"payload": map[string]interface{}{
			"issue": map[string]interface{}{
				"id":          "issue-1",
				"sequence_id": "WEB-1",
				"name":        "Test Issue",
				"state":       "in_progress",
			},
			"comment": map[string]interface{}{
				"id":           "comment-1",
				"comment_html": "<p>Test comment</p>",
				"actor_detail": map[string]string{
					"display_name": "Руслан",
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed["event"] != "issue.comment.created" {
		t.Error("event mismatch")
	}
}

func TestConfig_Load(t *testing.T) {
	t.Setenv("MAILBRIDGE_IMAP_SERVER", "localhost")
	t.Setenv("MAILBRIDGE_IMAP_USER", "test@localhost")
	t.Setenv("MAILBRIDGE_IMAP_PASS", "test")
	t.Setenv("MAILBRIDGE_SMTP_SERVER", "localhost")
	t.Setenv("MAILBRIDGE_SMTP_FROM", "support@test.local")
	t.Setenv("MAILBRIDGE_PLANE_BASE_URL", "http://localhost:8089/test")
	t.Setenv("MAILBRIDGE_PLANE_API_KEY", "test-key")
	t.Setenv("MAILBRIDGE_WEBHOOK_SECRET", "test-secret")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.IMAP.Server != "localhost" {
		t.Errorf("IMAP server = %s", cfg.IMAP.Server)
	}
}
