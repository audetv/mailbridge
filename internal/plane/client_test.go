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

func testServerURL(srv *httptest.Server) string {
	return srv.URL + "/test-workspace"
}

func TestClient_GetProjects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/projects") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []plane.Project{
				{ID: "proj-uuid-1", Name: "ТРК", Identifier: "TRK"},
				{ID: "proj-uuid-2", Name: "Отель", Identifier: "HOTEL"},
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
	if projects[0].Identifier != "TRK" {
		t.Errorf("Identifier = %s, want TRK", projects[0].Identifier)
	}
}

func TestClient_GetLabels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/labels") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []plane.Label{
				{ID: "label-uuid-1", Name: "Bug", Color: "#ff0000"},
				{ID: "label-uuid-2", Name: "Feature", Color: "#00ff00"},
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	labels, err := client.GetLabels(ctx, "proj-uuid")
	if err != nil {
		t.Fatalf("GetLabels error: %v", err)
	}
	if len(labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(labels))
	}
}

func TestClient_CreateLabel_New(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(plane.Label{
			ID:    "new-label-uuid",
			Name:  "Bug",
			Color: "#ff0000",
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	label, err := client.CreateLabel(ctx, "proj-uuid", &plane.CreateLabelRequest{
		Name:        "Bug",
		Color:       "#ff0000",
		Description: "Bug report",
	})
	if err != nil {
		t.Fatalf("CreateLabel error: %v", err)
	}
	if label.ID != "new-label-uuid" {
		t.Errorf("ID = %s, want new-label-uuid", label.ID)
	}
}

func TestClient_CreateLabel_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		if err := json.NewEncoder(w).Encode(plane.LabelConflictError{
			Error: "Label with the same name already exists in the project",
			ID:    "existing-label-uuid",
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	label, err := client.CreateLabel(ctx, "proj-uuid", &plane.CreateLabelRequest{
		Name:  "Bug",
		Color: "#ff0000",
	})
	if err != nil {
		t.Fatalf("CreateLabel error: %v", err)
	}
	if label.ID != "existing-label-uuid" {
		t.Errorf("ID = %s, want existing-label-uuid", label.ID)
	}
}

func TestClient_CreateWorkItem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/work-items") {
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
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(plane.WorkItem{
			ID:         "work-item-uuid",
			SequenceID: 1,
			ProjectID:  "proj-uuid",
			Name:       "Test Issue",
			Priority:   "high",
			Labels:     []string{"label-uuid"},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	item, err := client.CreateWorkItem(ctx, &plane.CreateWorkItemRequest{
		ProjectID:      "proj-uuid",
		Name:           "Test Issue",
		Description:    "<p>Description</p>",
		Priority:       "high",
		Labels:         []string{"label-uuid"},
		ExternalID:     "msg-001",
		ExternalSource: "mailbridge",
	})
	if err != nil {
		t.Fatalf("CreateWorkItem error: %v", err)
	}
	if item.ID != "work-item-uuid" {
		t.Errorf("ID = %s", item.ID)
	}
	if item.SequenceID != 1 {
		t.Errorf("SequenceID = %d, want 1", item.SequenceID)
	}
}

func TestClient_GetWorkItemByIdentifier(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/work-items/INBOX-1") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(plane.WorkItem{
			ID:         "work-item-uuid",
			SequenceID: 1,
			ProjectID:  "proj-uuid",
			Name:       "Existing Issue",
			State:      "in_progress",
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	item, err := client.GetWorkItemByIdentifier(ctx, "INBOX", 1)
	if err != nil {
		t.Fatalf("GetWorkItemByIdentifier error: %v", err)
	}
	if item.ID != "work-item-uuid" {
		t.Errorf("ID = %s", item.ID)
	}
}

func TestClient_AddComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/comments") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var reqBody map[string]string
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if reqBody["comment_html"] != "<p>Test comment</p>" {
			t.Errorf("comment_html = %q", reqBody["comment_html"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(plane.Comment{
			ID:   "comment-uuid",
			Body: "<p>Test comment</p>",
			Actor: &plane.Actor{
				ID:          "actor-uuid",
				DisplayName: "Test User",
			},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	comment, err := client.AddComment(ctx, "proj-uuid", "work-item-uuid", "<p>Test comment</p>", "msg-001")
	if err != nil {
		t.Fatalf("AddComment error: %v", err)
	}
	if comment.Body != "<p>Test comment</p>" {
		t.Errorf("Body = %q", comment.Body)
	}
	if comment.Actor == nil {
		t.Error("Actor is nil")
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
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []plane.Project{},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client := plane.NewClient(testServerURL(srv), "test-key")
	ctx := context.Background()

	_, err := client.GetProjects(ctx)
	if err != nil {
		t.Fatalf("GetProjects error after retry: %v", err)
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

	_, err := client.GetProjects(ctx)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt for 4xx, got %d", attempts)
	}
}

func TestExtractWorkspace(t *testing.T) {
	client := plane.NewClient("http://localhost/gc", "key")
	if client == nil {
		t.Fatal("client is nil")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/v1/workspaces/gc/projects/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []plane.Project{},
		}); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}))
	defer srv.Close()

	client = plane.NewClient(srv.URL+"/gc", "key")
	_, err := client.GetProjects(context.Background())
	if err != nil {
		t.Fatalf("GetProjects error: %v", err)
	}
}
