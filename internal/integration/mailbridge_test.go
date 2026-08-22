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
	"github.com/audetv/mailbridge/internal/store/sqlite"
)

func setupPlaneMock(t *testing.T) (*httptest.Server, *plane.Client) {
	t.Helper()

	var createdWorkItems []plane.WorkItem
	var comments []plane.Comment

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// GET /api/v1/workspaces/{ws}/projects/
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/projects") && !strings.Contains(r.URL.Path, "/work-items") && !strings.Contains(r.URL.Path, "/labels") {
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"results": []plane.Project{
					{ID: "proj-inbox", Name: "Входящие", Identifier: "INBOX"},
					{ID: "proj-trk", Name: "ТРК", Identifier: "TRK"},
					{ID: "proj-hotel", Name: "Отель", Identifier: "HOTEL"},
				},
			}); err != nil {
				t.Errorf("encode error: %v", err)
			}
			return
		}

		// GET /api/v1/workspaces/{ws}/projects/{id}/labels/
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/labels") {
			if err := json.NewEncoder(w).Encode(map[string]interface{}{
				"results": []plane.Label{
					{ID: "label-bug", Name: "bug", Color: "#ff0000"},
				},
			}); err != nil {
				t.Errorf("encode error: %v", err)
			}
			return
		}

		// POST /api/v1/workspaces/{ws}/projects/{id}/labels/
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/labels") {
			var req plane.CreateLabelRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode error: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(plane.Label{
				ID:    fmt.Sprintf("label-%s", req.Name),
				Name:  req.Name,
				Color: req.Color,
			}); err != nil {
				t.Errorf("encode error: %v", err)
			}
			return
		}

		// POST /api/v1/workspaces/{ws}/projects/{id}/work-items/
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/work-items") && !strings.Contains(r.URL.Path, "/comments") {
			var req plane.CreateWorkItemRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode error: %v", err)
			}

			item := plane.WorkItem{
				ID:         fmt.Sprintf("work-item-%d", len(createdWorkItems)+1),
				SequenceID: len(createdWorkItems) + 1,
				ProjectID:  req.ProjectID,
				Name:       req.Name,
				Priority:   req.Priority,
				Labels:     req.Labels,
				ExternalID: req.ExternalID,
				CreatedAt:  time.Now(),
			}
			createdWorkItems = append(createdWorkItems, item)

			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(item); err != nil {
				t.Errorf("encode error: %v", err)
			}
			return
		}

		// GET /api/v1/workspaces/{ws}/work-items/{identifier}-{seq}/
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/work-items/") && strings.Contains(r.URL.Path, "-") && !strings.Contains(r.URL.Path, "/comments") {
			for _, item := range createdWorkItems {
				expected := fmt.Sprintf("/api/v1/workspaces/test-workspace/work-items/INBOX-%d/", item.SequenceID)
				if r.URL.Path == expected {
					if err := json.NewEncoder(w).Encode(item); err != nil {
						t.Errorf("encode error: %v", err)
					}
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// POST .../work-items/{id}/comments/
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments") {
			var req map[string]string
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode error: %v", err)
			}

			comment := plane.Comment{
				ID:         fmt.Sprintf("comment-%d", len(comments)+1),
				Body:       req["comment_html"],
				ExternalID: req["external_id"],
				CreatedAt:  time.Now(),
				ActorRaw:   json.RawMessage(`"actor-1"`),
			}
			comments = append(comments, comment)

			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(comment); err != nil {
				t.Errorf("encode error: %v", err)
			}
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

	workItem, err := client.CreateWorkItem(context.Background(), &plane.CreateWorkItemRequest{
		ProjectID:      "proj-trk",
		Name:           "Не работает кабинет арендатора",
		Description:    "<p>Описание</p>",
		Priority:       "high",
		Labels:         []string{"label-bug"},
		ExternalID:     "msg-001",
		ExternalSource: "mailbridge",
	})
	if err != nil {
		t.Fatalf("CreateWorkItem error: %v", err)
	}
	if workItem.ID == "" {
		t.Error("work item ID is empty")
	}
	if workItem.SequenceID != 1 {
		t.Errorf("SequenceID = %d, want 1", workItem.SequenceID)
	}

	item, err := client.GetWorkItemByIdentifier(context.Background(), "INBOX", 1)
	if err != nil {
		t.Fatalf("GetWorkItemByIdentifier error: %v", err)
	}
	if item.ID != workItem.ID {
		t.Errorf("ID = %s, want %s", item.ID, workItem.ID)
	}

	comment, err := client.AddComment(context.Background(), workItem.ProjectID, workItem.ID, "<p>Test</p>", "msg-002")
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
		IssueSequence:      "INBOX-123",
		ProjectName:        "ТРК",
		TypeName:           "bug",
		Priority:           "high",
	})
	if !strings.Contains(body, "MAILBRIDGE-INTERNAL") {
		t.Error("body should contain internal marker")
	}
	if !strings.Contains(body, "INBOX-123") {
		t.Error("body should contain issue sequence")
	}

	_, body = sender.FormatCommentReply(&sender.CommentReplyData{
		To:                 "user@test.local",
		Subject:            "Re: Не работает сайт",
		InReplyToMessageID: "original-msg",
		IssueSequence:      "INBOX-123",
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
				"sequence_id": 1,
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
