package plane_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/plane"
)

// testServerURL возвращает тестовый URL с workspace для совместимости.
// "http://127.0.0.1:PORT/test-workspace"
func testServerURL(srv *httptest.Server) string {
	return srv.URL + "/test-workspace"
}

func TestClient_CreateIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Error("missing or wrong API key header")
		}
		if !strings.Contains(r.URL.Path, "/issues/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if reqBody["name"] != "Test Issue" {
			t.Errorf("name = %v", reqBody["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(plane.Issue{
			ID:         "issue-uuid-1",
			SequenceID: "WEB-1",
			ProjectID:  "proj-uuid",
			Name:       "Test Issue",
			Priority:   "high",
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	issue, err := client.CreateIssue(ctx, &plane.CreateIssueRequest{
		ProjectID:   "proj-uuid",
		Name:        "Test Issue",
		Description: "Description text",
		Priority:    "high",
		Labels:      []string{"bug", "urgent"},
	})
	if err != nil {
		t.Fatalf("CreateIssue error: %v", err)
	}

	if issue.SequenceID != "WEB-1" {
		t.Errorf("SequenceID = %q, want %q", issue.SequenceID, "WEB-1")
	}
}

func TestClient_GetIssue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/issues/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(plane.Issue{
			ID:         "issue-uuid-2",
			SequenceID: "WEB-2",
			Name:       "Existing Issue",
			State:      "in_progress",
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	issue, err := client.GetIssue(ctx, "issue-uuid-2")
	if err != nil {
		t.Fatalf("GetIssue error: %v", err)
	}

	if issue.ID != "issue-uuid-2" {
		t.Errorf("ID = %q", issue.ID)
	}
}

func TestClient_AddComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/comments/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var reqBody map[string]string
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if reqBody["comment_html"] != "Test comment" {
			t.Errorf("comment_html = %q", reqBody["comment_html"])
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(plane.Comment{
			ID:      "comment-uuid",
			IssueID: "issue-uuid",
			Body:    "<p>Test comment</p>",
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	comment, err := client.AddComment(ctx, "issue-uuid", "Test comment")
	if err != nil {
		t.Fatalf("AddComment error: %v", err)
	}

	if comment.Body != "<p>Test comment</p>" {
		t.Errorf("Body = %q", comment.Body)
	}
}

func TestClient_GetProjects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/projects") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []plane.Project{
				{ID: "proj-1", Name: "ТРК"},
				{ID: "proj-2", Name: "Отель"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	projects, err := client.GetProjects(ctx)
	if err != nil {
		t.Fatalf("GetProjects error: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}
}

func TestClient_RetryOnServerError(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(plane.Issue{ID: "ok"}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	_, err := client.GetIssue(ctx, "any")
	if err != nil {
		t.Fatalf("GetIssue error after retry: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestClient_ClientErrorNoRetry(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	_, err := client.GetIssue(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404")
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt for 4xx, got %d", attempts)
	}
}

func TestExtractWorkspace(t *testing.T) {
	client := plane.NewClient("http://localhost/gc", "key")
	// Проверяем что клиент создался без ошибок
	if client == nil {
		t.Fatal("client is nil")
	}

	// Проверяем что GetProjects формирует правильный URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/workspaces/gc/projects/"
		if r.URL.Path != expectedPath {
			t.Errorf("path = %s, want %s", r.URL.Path, expectedPath)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"results": []plane.Project{}}); err != nil {
			t.Errorf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client = plane.NewClient(srv.URL+"/gc", "key")
	_, err := client.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("GetProjects error: %v", err)
	}
}
