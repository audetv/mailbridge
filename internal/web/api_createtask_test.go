package web_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
	"github.com/audetv/mailbridge/internal/web"
)

type createTaskEnv struct {
	h  *web.TaskHandler
	st *sqlite.Store
}

func newCreateTaskEnv(t *testing.T) *createTaskEnv {
	t.Helper()
	st, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		st.Close()
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return &createTaskEnv{h: web.NewTaskHandler(st, web.NewEventBroker()), st: st}
}

func (e *createTaskEnv) mkProject(t *testing.T, name string) *store.Project {
	t.Helper()
	p := &store.Project{Name: name}
	if err := e.st.CreateProject(context.Background(), p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	return p
}

func (e *createTaskEnv) mkEpic(t *testing.T, p *store.Project) *store.Epic {
	t.Helper()
	e1 := &store.Epic{ProjectID: p.ID, Name: "Модуль"}
	if err := e.st.CreateEpic(context.Background(), e1); err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}
	return e1
}

func (e *createTaskEnv) post(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	e.h.CreateTask(w, httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(body)))
	return w
}

func (e *createTaskEnv) decodeTask(t *testing.T, w *httptest.ResponseRecorder) *store.Task {
	t.Helper()
	var tk store.Task
	if err := json.NewDecoder(w.Body).Decode(&tk); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	return &tk
}

func TestCreateTask_OK(t *testing.T) {
	e := newCreateTaskEnv(t)
	p := e.mkProject(t, "Деск X")
	ew := e.mkEpic(t, p)
	desc := "текст описания"

	w := e.post(t, `{"title":"Новая задача","project":"Деск X","description":"`+desc+`","epic_id":`+itoa64(ew.ID)+`}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	tk := e.decodeTask(t, w)
	if tk.Subject != "Новая задача" || tk.Project != "Деск X" {
		t.Errorf("bad fields: %+v", tk)
	}
	if tk.Status != "new" {
		t.Errorf("status = %q, want new", tk.Status)
	}
	if tk.BodyText != desc {
		t.Errorf("body = %q, want %q", tk.BodyText, desc)
	}
	if tk.EpicID == nil || *tk.EpicID != ew.ID {
		t.Errorf("epic_id = %v, want %d", tk.EpicID, ew.ID)
	}
	if !regexp.MustCompile(`^manual-[0-9a-f]{16}$`).MatchString(tk.MessageID) {
		t.Errorf("message_id = %q, want manual-<16hex>", tk.MessageID)
	}
	if tk.ID == 0 || tk.CreatedAt.IsZero() {
		t.Errorf("id/timestamps not set: %+v", tk)
	}
}

func TestCreateTask_Minimal(t *testing.T) {
	e := newCreateTaskEnv(t)
	e.mkProject(t, "Деск Y")

	w := e.post(t, `{"title":"Минимум","project":"Деск Y"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	tk := e.decodeTask(t, w)
	if tk.Status != "new" || tk.EpicID != nil || tk.BodyText != "" {
		t.Errorf("unexpected defaults: %+v", tk)
	}
}

func TestCreateTask_Validation(t *testing.T) {
	e := newCreateTaskEnv(t)
	e.mkProject(t, "Деск Z")
	cases := []struct {
		body string
		want int
	}{
		{`{"title":"","project":"Деск Z"}`, http.StatusBadRequest},
		{`{"title":"  ","project":"Деск Z"}`, http.StatusBadRequest},
		{`{"title":"` + strings.Repeat("т", 501) + `"}`, http.StatusBadRequest}, // нет проекта
		{`{"project":"Деск Z"}`, http.StatusBadRequest},                         // нет title
		{`{"title":"x"}`, http.StatusBadRequest},                                // нет project
		{`bad json`, http.StatusBadRequest},
	}
	for _, c := range cases {
		if w := e.post(t, c.body); w.Code != c.want {
			t.Errorf("body=%q status=%d want=%d", c.body, w.Code, c.want)
		}
	}
}

func TestCreateTask_ProjectNotFound(t *testing.T) {
	e := newCreateTaskEnv(t)
	w := e.post(t, `{"title":"x","project":"Нет такого"}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestCreateTask_EpicValidation(t *testing.T) {
	e := newCreateTaskEnv(t)
	p := e.mkProject(t, "Деск A")
	foreign := e.mkProject(t, "Деск B")
	eown := e.mkEpic(t, p)
	efor := e.mkEpic(t, foreign)

	if w := e.post(t, `{"title":"x","project":"Деск A","epic_id":9999}`); w.Code != http.StatusBadRequest {
		t.Errorf("missing epic status=%d want 400", w.Code)
	}
	if w := e.post(t, `{"title":"x","project":"Деск A","epic_id":`+itoa64(efor.ID)+`}`); w.Code != http.StatusBadRequest {
		t.Errorf("foreign epic status=%d want 400", w.Code)
	}
	// чужой epic у валидного проекта — 400, задача НЕ создаётся
	if w := e.post(t, `{"title":"x","project":"Деск B","epic_id":`+itoa64(eown.ID)+`}`); w.Code != http.StatusBadRequest {
		t.Errorf("cross-project epic status=%d want 400", w.Code)
	}
}

func TestCreateTask_MethodNotAllowed(t *testing.T) {
	e := newCreateTaskEnv(t)
	w := httptest.NewRecorder()
	e.h.CreateTask(w, httptest.NewRequest(http.MethodGet, "/api/tasks", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want 405", w.Code)
	}
}
