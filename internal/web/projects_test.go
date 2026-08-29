package web_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
	"github.com/audetv/mailbridge/internal/web"
)

// projectsTest holds the API-under-test plus a live store.
type projectsTest struct {
	h       *web.ProjectHandler
	s       *sqlite.Store
	cleanup func()
}

func newProjectsTest(t *testing.T) *projectsTest {
	t.Helper()
	st, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		st.Close()
		t.Fatalf("Migrate: %v", err)
	}
	h := web.NewProjectHandler(st, nil)
	cleanup := func() { st.Close() }
	t.Cleanup(cleanup)
	return &projectsTest{h: h, s: st, cleanup: cleanup}
}

func (pt *projectsTest) decode(w *httptest.ResponseRecorder) *store.Project {
	var p store.Project
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		// best effort — the test will fail on wrong status regardless
		return nil
	}
	return &p
}

func TestProjectsAPI_CreateAndGet(t *testing.T) {
	pt := newProjectsTest(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/projects",
		strings.NewReader(`{"name":"Мой Проект","description":"деск"}`))
	pt.h.CreateProject(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", w.Code, w.Body.String())
	}
	p := pt.decode(w)
	if p == nil || p.Name != "Мой Проект" {
		t.Fatalf("bad project: %+v", p)
	}
	if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
		t.Error("timestamps not set")
	}

	// GET by id
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/projects/"+itoa64(p.ID), nil)
	req.SetPathValue("id", itoa64(p.ID))
	pt.h.GetProject(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get status=%d", w.Code)
	}
}

func TestProjectsAPI_DuplicateName_409(t *testing.T) {
	pt := newProjectsTest(t)
	body := `{"name":"Дубль"}`
	w := httptest.NewRecorder()
	pt.h.CreateProject(w, httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(body)))

	w = httptest.NewRecorder()
	pt.h.CreateProject(w, httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(body)))
	if w.Code != http.StatusConflict {
		t.Fatalf("duplicate status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestProjectsAPI_Validation(t *testing.T) {
	pt := newProjectsTest(t)
	tests := []struct {
		body string
		want int
	}{
		{`{"name":"   "}`, http.StatusBadRequest},
		{`{"description":"нет имени"}`, http.StatusBadRequest},
		{`not json at all`, http.StatusBadRequest},
		{`{"name":"` + strings.Repeat("я", 129) + `"}`, http.StatusBadRequest},
	}
	for _, tc := range tests {
		w := httptest.NewRecorder()
		pt.h.CreateProject(w, httptest.NewRequest(http.MethodPost, "/api/projects", strings.NewReader(tc.body)))
		if w.Code != tc.want {
			t.Errorf("body=%q status=%d want=%d", tc.body, w.Code, tc.want)
		}
	}
}

func TestProjectsAPI_UpdateAndRename(t *testing.T) {
	pt := newProjectsTest(t)
	w := httptest.NewRecorder()
	pt.h.CreateProject(w, httptest.NewRequest(http.MethodPost, "/api/projects",
		strings.NewReader(`{"name":"Исходный"}`)))
	p := pt.decode(w)

	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/projects/"+itoa64(p.ID),
		strings.NewReader(`{"name":"После","description":"новое"}`))
	req.SetPathValue("id", itoa64(p.ID))
	pt.h.UpdateProject(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", w.Code, w.Body.String())
	}
	upd := pt.decode(w)
	if upd.Name != "После" || upd.Description != "новое" {
		t.Errorf("after update: %+v", upd)
	}
}

func TestProjectsAPI_ArchiveUnarchive(t *testing.T) {
	pt := newProjectsTest(t)
	w := httptest.NewRecorder()
	pt.h.CreateProject(w, httptest.NewRequest(http.MethodPost, "/api/projects",
		strings.NewReader(`{"name":"Архив"}`)))
	p := pt.decode(w)

	// Archive
	w = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/projects/"+itoa64(p.ID), nil)
	req.SetPathValue("id", itoa64(p.ID))
	pt.h.ArchiveProject(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("archive status=%d", w.Code)
	}
	if !pt.decode(w).Archived {
		t.Error("project expected archived")
	}

	// Unarchive
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/projects/"+itoa64(p.ID)+"/unarchive", nil)
	req.SetPathValue("id", itoa64(p.ID))
	pt.h.UnarchiveProject(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unarchive status=%d", w.Code)
	}
	if pt.decode(w).Archived {
		t.Error("project expected unarchived")
	}
}

func TestProjectsAPI_ListFilters(t *testing.T) {
	pt := newProjectsTest(t)
	// Сид уже содержит «Входящие»; создаём свой и архивируем
	w := httptest.NewRecorder()
	pt.h.CreateProject(w, httptest.NewRequest(http.MethodPost, "/api/projects",
		strings.NewReader(`{"name":"Активный"}`)))
	activeP := pt.decode(w)

	w = httptest.NewRecorder()
	pt.h.CreateProject(w, httptest.NewRequest(http.MethodPost, "/api/projects",
		strings.NewReader(`{"name":"Мёртвый"}`)))
	deadP := pt.decode(w)

	// Archive the dead one
	w = httptest.NewRecorder()
	deadReq := httptest.NewRequest(http.MethodDelete, "/api/projects/"+itoa64(deadP.ID), nil)
	deadReq.SetPathValue("id", itoa64(deadP.ID))
	pt.h.ArchiveProject(w, deadReq)

	// List archived=true → 1
	w = httptest.NewRecorder()
	pt.h.ListProjects(w, httptest.NewRequest(http.MethodGet, "/api/projects?archived=true", nil))
	var arch []*store.Project
	if err := json.NewDecoder(w.Body).Decode(&arch); err != nil {
		t.Fatalf("decode archived: %v", err)
	}
	if len(arch) != 1 || arch[0].ID != deadP.ID {
		t.Errorf("archived list=%d items, want 1 (Мёртвый), got %+v", len(arch), arch)
	}

	// List archived=false → «Входящие» (сид) + «Активный»
	w = httptest.NewRecorder()
	pt.h.ListProjects(w, httptest.NewRequest(http.MethodGet, "/api/projects?archived=false", nil))
	var act []*store.Project
	if err := json.NewDecoder(w.Body).Decode(&act); err != nil {
		t.Fatalf("decode active: %v", err)
	}
	if len(act) != 2 {
		t.Errorf("active list=%d, want 2 (Входящие+Активный)", len(act))
	}
	for _, pr := range act {
		if pr.Archived {
			t.Errorf("archived leaked into active: %+v", pr)
		}
	}

	// Search «Актив»
	w = httptest.NewRecorder()
	pt.h.ListProjects(w, httptest.NewRequest(http.MethodGet, "/api/projects?search=Актив", nil))
	var f []*store.Project
	if err := json.NewDecoder(w.Body).Decode(&f); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if len(f) != 1 || f[0].ID != activeP.ID {
		t.Errorf("search=Актив → %d, want 1 (Активный)", len(f))
	}
}

func TestProjectsAPI_MethodNotAllowed(t *testing.T) {
	pt := newProjectsTest(t)
	w := httptest.NewRecorder()
	pt.h.ListProjects(w, httptest.NewRequest(http.MethodDelete, "/api/projects", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status=%d want=405", w.Code)
	}
	w = httptest.NewRecorder()
	pt.h.CreateProject(w, httptest.NewRequest(http.MethodGet, "/api/projects", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status=%d want=405", w.Code)
	}
}

func itoa64(i int64) string {
	b := []byte{}
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
