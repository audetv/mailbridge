package web_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
	"github.com/audetv/mailbridge/internal/web"
)

// epicTest — окружение API-тестов: store + хендлер.
type epicTest struct {
	h  *web.EpicHandler
	st *sqlite.Store
}

func newEpicTest(t *testing.T) *epicTest {
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
	return &epicTest{h: web.NewEpicHandler(st, nil), st: st}
}

func (et *epicTest) mkProject(t *testing.T) *store.Project {
	t.Helper()
	p := &store.Project{Name: "Проект эпиков"}
	if err := et.st.CreateProject(context.Background(), p); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	return p
}

func (et *epicTest) mkTask(t *testing.T, project *store.Project, msgID string) *store.Task {
	t.Helper()
	task := &store.Task{MessageID: msgID, Subject: "s", BodyText: "b", Status: "active", Project: project.Name}
	if err := et.st.CreateTask(context.Background(), task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	return task
}

func reqWithBody(method, target, payload string, pathVals ...string) *http.Request {
	var r *http.Request
	if payload != "" {
		r = httptest.NewRequest(method, target, strings.NewReader(payload))
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	for i := 0; i+1 < len(pathVals); i += 2 {
		r.SetPathValue(pathVals[i], pathVals[i+1])
	}
	return r
}

func epURL(id int64) string   { return fmt.Sprintf("/api/epics/%d", id) }
func pjEPURL(id int64) string { return fmt.Sprintf("/api/projects/%d/epics", id) }

func TestEpicsAPI_CRUD(t *testing.T) {
	et := newEpicTest(t)
	proj := et.mkProject(t)

	// POST: создание (авто-номер 1, статус open)
	w := httptest.NewRecorder()
	et.h.CreateEpicList(w, reqWithBody(http.MethodPost, pjEPURL(proj.ID),
		`{"name":"Модуль A"}`, "id", fmt.Sprintf("%d", proj.ID)))
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status=%d body=%s", w.Code, w.Body.String())
	}
	var created store.Epic
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.Number != 1 || created.Status != "open" {
		t.Fatalf("created epic: %+v", created)
	}

	// POST: пустое имя → 400
	w = httptest.NewRecorder()
	et.h.CreateEpicList(w, reqWithBody(http.MethodPost, pjEPURL(proj.ID), `{"name":"  "}`, "id", fmt.Sprintf("%d", proj.ID)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty name: status=%d, want 400", w.Code)
	}

	// POST: несуществующий проект → 404
	w = httptest.NewRecorder()
	et.h.CreateEpicList(w, reqWithBody(http.MethodPost, "/api/projects/9999/epics", `{"name":"X"}`, "id", "9999"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("missing project: status=%d, want 404", w.Code)
	}

	// GET-список
	w = httptest.NewRecorder()
	et.h.ListEpicsList(w, reqWithBody(http.MethodGet, pjEPURL(proj.ID), "", "id", fmt.Sprintf("%d", proj.ID)))
	if w.Code != http.StatusOK {
		t.Fatalf("list: status=%d", w.Code)
	}
	var list []*store.Epic
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("list: %+v", list)
	}

	// GET-детали: прогресс 0 задач
	w = httptest.NewRecorder()
	et.h.GetEpicDetailHandler(w, reqWithBody(http.MethodGet, epURL(created.ID), "", "epic_id", fmt.Sprintf("%d", created.ID)))
	if w.Code != http.StatusOK {
		t.Fatalf("get detail: status=%d body=%s", w.Code, w.Body.String())
	}
	var detail web.GetEpicDetail
	if err := json.NewDecoder(w.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.Epic == nil || detail.ID != created.ID || detail.Progress == nil || detail.Progress.Total != 0 {
		t.Fatalf("detail: %+v", detail)
	}

	// PUT: обновление имени+статуса
	w = httptest.NewRecorder()
	et.h.UpdateEpicDetail(w, reqWithBody(http.MethodPut, epURL(created.ID),
		`{"name":"Модуль A v2","description":"опис","status":"in_progress"}`, "epic_id", fmt.Sprintf("%d", created.ID)))
	if w.Code != http.StatusOK {
		t.Fatalf("update: status=%d body=%s", w.Code, w.Body.String())
	}

	// PUT: невалидный статус → 400
	w = httptest.NewRecorder()
	et.h.UpdateEpicDetail(w, reqWithBody(http.MethodPut, epURL(created.ID),
		`{"name":"n","status":"bogus"}`, "epic_id", fmt.Sprintf("%d", created.ID)))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("bad status: status=%d, want 400", w.Code)
	}

	// DELETE
	w = httptest.NewRecorder()
	et.h.DeleteEpicDetail(w, reqWithBody(http.MethodDelete, epURL(created.ID), "", "epic_id", fmt.Sprintf("%d", created.ID)))
	if w.Code != http.StatusOK {
		t.Fatalf("delete: status=%d", w.Code)
	}
	if got, _ := et.st.GetEpic(context.Background(), created.ID); got != nil {
		t.Fatalf("epic still present after delete: %+v", got)
	}

	// DELETE повторный → 404
	w = httptest.NewRecorder()
	et.h.DeleteEpicDetail(w, reqWithBody(http.MethodDelete, epURL(created.ID), "", "epic_id", fmt.Sprintf("%d", created.ID)))
	if w.Code != http.StatusNotFound {
		t.Fatalf("second delete: status=%d, want 404", w.Code)
	}
}

func TestEpicsAPI_TaskLinking(t *testing.T) {
	et := newEpicTest(t)
	proj := et.mkProject(t)

	w := httptest.NewRecorder()
	et.h.CreateEpicList(w, reqWithBody(http.MethodPost, pjEPURL(proj.ID), `{"name":"М"}`, "id", fmt.Sprintf("%d", proj.ID)))
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status=%d body=%s", w.Code, w.Body.String())
	}
	var epic store.Epic
	if err := json.NewDecoder(w.Body).Decode(&epic); err != nil {
		t.Fatalf("decode: %v", err)
	}

	task := et.mkTask(t, proj, "epic-api-task-1")

	// POST: привязка
	w = httptest.NewRecorder()
	et.h.LinkTaskEpic(w, reqWithBody(http.MethodPost, fmt.Sprintf("/api/epics/%d/tasks/%d", epic.ID, task.ID), "", "epic_id", fmt.Sprintf("%d", epic.ID), "taskId", fmt.Sprintf("%d", task.ID)))
	if w.Code != http.StatusOK {
		t.Fatalf("link: status=%d body=%s", w.Code, w.Body.String())
	}
	got, _ := et.st.GetTask(context.Background(), task.ID)
	if got == nil || got.EpicID == nil || *got.EpicID != epic.ID {
		t.Fatalf("task after link: %+v", got)
	}

	// POST: нет задачи → 404
	w = httptest.NewRecorder()
	et.h.LinkTaskEpic(w, reqWithBody(http.MethodPost, fmt.Sprintf("/api/epics/%d/tasks/999999", epic.ID), "", "epic_id", fmt.Sprintf("%d", epic.ID), "taskId", "999999"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("link missing task: status=%d, want 404", w.Code)
	}

	// DELETE: отвязка
	w = httptest.NewRecorder()
	et.h.UnlinkTaskEpic(w, reqWithBody(http.MethodDelete, fmt.Sprintf("/api/epics/%d/tasks/%d", epic.ID, task.ID), "", "epic_id", fmt.Sprintf("%d", epic.ID), "taskId", fmt.Sprintf("%d", task.ID)))
	if w.Code != http.StatusOK {
		t.Fatalf("unlink: status=%d body=%s", w.Code, w.Body.String())
	}
	got, _ = et.st.GetTask(context.Background(), task.ID)
	if got == nil || got.EpicID != nil {
		t.Fatalf("task after unlink: %+v", got)
	}
}
