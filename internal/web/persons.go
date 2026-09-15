package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/audetv/mailbridge/internal/store"
)

// PersonHandler обрабатывает запросы к справочнику Персон (v0.24, шаг 6, Режим A).
// Канон: docs/ontology.md §7.7 / §7.7.1.
type PersonHandler struct {
	store  store.Store
	broker *EventBroker
}

// NewPersonHandler создаёт PersonHandler (broker может быть nil).
func NewPersonHandler(st store.Store, broker *EventBroker) *PersonHandler {
	return &PersonHandler{store: st, broker: broker}
}

// publishWS — событие всем подписчикам (broker nil — тихо пропускаем).
func (h *PersonHandler) publishWS(eventType string, data any, message string) {
	if h.broker == nil || data == nil {
		return
	}
	h.broker.Publish(WSEvent{Type: eventType, Message: message, Data: data})
}

// ListPersons — GET /api/persons. Query: search, confirmed, archived, is_internal, page, per_page.
func (h *PersonHandler) ListPersons(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	q := r.URL.Query()
	filter := &store.PersonFilter{
		Search: strings.TrimSpace(q.Get("search")),
	}
	if v, ok := parseBoolQuery(q.Get("confirmed")); ok {
		filter.Confirmed = &v
	}
	if v, ok := parseBoolQuery(q.Get("archived")); ok {
		filter.Archived = &v
	}
	if v, ok := parseBoolQuery(q.Get("is_internal")); ok {
		filter.IsInternal = &v
	}
	filter.Page, _ = strconv.Atoi(q.Get("page"))
	filter.PerPage, _ = strconv.Atoi(q.Get("per_page"))

	res, err := h.store.ListPersons(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if res.Persons == nil {
		res.Persons = []*store.Person{}
	}
	writeJSON(w, http.StatusOK, res)
}

// GetPerson — GET /api/persons/{id}.
func (h *PersonHandler) GetPerson(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, ok := parsePersonID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid person id")
		return
	}
	p, err := h.store.GetPerson(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "person not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// CreatePerson — POST /api/persons. Body: name, org, is_internal, email.
// При email персона сразу получает идентичность kind=email (provenance=manual).
func (h *PersonHandler) CreatePerson(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Name       string `json:"name"`
		Org        string `json:"org"`
		IsInternal bool   `json:"is_internal"`
		Email      string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	p := &store.Person{
		Name:       strings.TrimSpace(body.Name),
		Org:        strings.TrimSpace(body.Org),
		IsInternal: body.IsInternal,
	}
	if err := h.store.CreatePerson(r.Context(), p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Email → идентичность (если задан и не занят): best-effort с понятной ошибкой.
	if body.Email != "" {
		if id, err := h.store.FindIdentity(r.Context(), "email", body.Email); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		} else if id == nil {
			ident := &store.PersonIdentity{
				PersonID:   p.ID,
				Kind:       "email",
				Value:      strings.ToLower(strings.TrimSpace(body.Email)),
				IsPrimary:  true,
				Provenance: "manual",
			}
			if _, err := h.store.AddPersonIdentity(r.Context(), ident); err != nil {
				writeError(w, http.StatusConflict, "email identity already exists")
				return
			}
		} else {
			writeError(w, http.StatusConflict, "email identity already exists")
			return
		}
	}

	created, err := h.store.GetPerson(r.Context(), p.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.publishWS("person_created", created, fmt.Sprintf("Персона создана: %s", displayName(created)))
	writeJSON(w, http.StatusCreated, created)
}

// UpdatePerson — PUT /api/persons/{id}. Поля тела опциональны (присутствует — обновить).
func (h *PersonHandler) UpdatePerson(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, ok := parsePersonID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid person id")
		return
	}
	existing, err := h.store.GetPerson(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "person not found")
		return
	}
	var body struct {
		Name       string `json:"name"`
		Org        string `json:"org"`
		IsInternal bool   `json:"is_internal"`
		Confirmed  bool   `json:"confirmed"`
		Archived   bool   `json:"archived"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	existing.Name = strings.TrimSpace(body.Name)
	existing.Org = strings.TrimSpace(body.Org)
	existing.IsInternal = body.IsInternal
	existing.Confirmed = body.Confirmed
	existing.Archived = body.Archived

	if err := h.store.UpdatePerson(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	updated, err := h.store.GetPerson(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.publishWS("person_updated", updated, fmt.Sprintf("Персона обновлена: %s", displayName(updated)))
	writeJSON(w, http.StatusOK, updated)
}

// ArchivePerson — POST /api/persons/{id}/archive. Restore (вместо удаления).
func (h *PersonHandler) ArchivePerson(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, ok := parsePersonID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid person id")
		return
	}
	p, err := h.store.GetPerson(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "person not found")
		return
	}
	p.Archived = true
	if err := h.store.UpdatePerson(r.Context(), p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	archived, _ := h.store.GetPerson(r.Context(), id)
	if archived == nil {
		writeError(w, http.StatusNotFound, "person not found")
		return
	}
	h.publishWS("person_archived", archived, fmt.Sprintf("Персона заархивирована: %s", displayName(archived)))
	writeJSON(w, http.StatusOK, archived)
}

// UnarchivePerson — POST /api/persons/{id}/unarchive.
func (h *PersonHandler) UnarchivePerson(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, ok := parsePersonID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid person id")
		return
	}
	p, err := h.store.GetPerson(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "person not found")
		return
	}
	p.Archived = false
	if err := h.store.UpdatePerson(r.Context(), p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	restored, _ := h.store.GetPerson(r.Context(), id)
	if restored == nil {
		writeError(w, http.StatusNotFound, "person not found")
		return
	}
	h.publishWS("person_unarchived", restored, fmt.Sprintf("Персона восстановлена: %s", displayName(restored)))
	writeJSON(w, http.StatusOK, restored)
}

// ListIdentities — GET /api/persons/{id}/identities.
func (h *PersonHandler) ListIdentities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, ok := parsePersonID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid person id")
		return
	}
	idents, err := h.store.ListIdentities(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if idents == nil {
		idents = []*store.PersonIdentity{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"identities": idents})
}

// AddIdentity — POST /api/persons/{id}/identities. Body: kind, value, external_id.
// UNIQUE(kind, value) → 409, если идентичность уже закреплена за другой персоной.
func (h *PersonHandler) AddIdentity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, ok := parsePersonID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid person id")
		return
	}
	var body struct {
		Kind       string `json:"kind"`
		Value      string `json:"value"`
		ExternalID string `json:"external_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	body.Kind = strings.TrimSpace(body.Kind)
	body.Value = strings.TrimSpace(body.Value)
	if body.Kind == "" || body.Value == "" {
		writeError(w, http.StatusBadRequest, "kind and value are required")
		return
	}
	if body.Kind == "email" {
		body.Value = strings.ToLower(body.Value)
	}

	existing, err := h.store.FindIdentity(r.Context(), body.Kind, body.Value)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing != nil && existing.PersonID != id {
		writeError(w, http.StatusConflict, "identity already linked to another person")
		return
	}

	ident := &store.PersonIdentity{
		PersonID:   id,
		Kind:       body.Kind,
		Value:      body.Value,
		ExternalID: strings.TrimSpace(body.ExternalID),
		Provenance: "manual",
	}
	added, err := h.store.AddPersonIdentity(r.Context(), ident)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	p, _ := h.store.GetPerson(r.Context(), id)
	h.publishWS("person_identity_added", added, fmt.Sprintf("Идентичность добавлена: %s %s", body.Kind, body.Value))
	if p != nil {
		h.publishWS("person_updated", p, fmt.Sprintf("Персона обновлена: %s", displayName(p)))
	}
	writeJSON(w, http.StatusCreated, added)
}

// SuggestMatches — POST /api/persons/suggest-matches.
// Body: email (или kind+value), name. Возвращает кандидаты на слияние
// (fast-path/fuzzy по канону §7.7.1) — UI решает: принять (merge) или отклонить.
func (h *PersonHandler) SuggestMatches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	kind, value := body.Kind, body.Value
	if body.Email != "" {
		kind, value = "email", strings.ToLower(strings.TrimSpace(body.Email))
	}
	value = strings.TrimSpace(value)
	if value == "" {
		writeError(w, http.StatusBadRequest, "email or kind+value is required")
		return
	}

	// Fast-path: идентичность уже привязана — это не «предложение», а факт.
	if existing, err := h.store.FindIdentity(r.Context(), kind, value); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	} else if existing != nil {
		p, err := h.store.GetPerson(r.Context(), existing.PersonID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if p == nil {
			writeError(w, http.StatusNotFound, "linked person not found")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"linked": true,
			"reason": "fast_path",
			"person": p,
		})
		return
	}

	candidates, err := h.store.SuggestMatch(r.Context(), kind, value, body.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if candidates == nil {
		candidates = []*store.Person{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"linked": false, "candidates": candidates})
}

// MergePersons — POST /api/persons-merge. Body: {target_id, source_id} —
// дубль source_id вливается в target_id (identities + роли переезжают;
// источник остаётся заархивированным ghost, история не теряется).
func (h *PersonHandler) MergePersons(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		TargetID string `json:"target_id"`
		SourceID string `json:"source_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	id, ok := parsePersonIDValue(body.TargetID)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid target_id")
		return
	}
	srcID, ok := parsePersonIDValue(body.SourceID)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid source_id")
		return
	}
	if srcID == id {
		writeError(w, http.StatusConflict, "cannot merge a person into itself")
		return
	}
	if err := h.store.MergePersons(r.Context(), srcID, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "source person not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	target, _ := h.store.GetPerson(r.Context(), id)
	if target != nil {
		h.publishWS("person_merged", target, fmt.Sprintf("Персоны слиты в: %s", displayName(target)))
	}
	writeJSON(w, http.StatusOK, target)
}

// SetTaskPersonRoles — PUT /api/tasks/{id}/persons. Body: {requestor_id?, assignee_id?}.
// Роль задаётся по персонам (шаг 6): поле, отсутствующее в теле — не меняется;
// явный null — отвязывается. Legacy assignee (email) при этом не трогается.
func (h *PersonHandler) SetTaskPersonRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	taskID, ok := parseID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	var body struct {
		RequestorID *string `json:"requestor_id"`
		AssigneeID  *string `json:"assignee_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	var reqID, asgID *store.PersonID
	if body.RequestorID != nil {
		if *body.RequestorID != "" {
			v, ok := parsePersonIDValue(*body.RequestorID)
			if !ok {
				writeError(w, http.StatusBadRequest, "invalid requestor_id")
				return
			}
			reqID = &v
		} else {
			reqID = new(store.PersonID) // явный null — отвязать
		}
	}
	if body.AssigneeID != nil {
		if *body.AssigneeID != "" {
			v, ok := parsePersonIDValue(*body.AssigneeID)
			if !ok {
				writeError(w, http.StatusBadRequest, "invalid assignee_id")
				return
			}
			asgID = &v
		} else {
			asgID = new(store.PersonID) // явный null — отвязать
		}
	}
	if reqID == nil && asgID == nil {
		writeError(w, http.StatusBadRequest, "requestor_id or assignee_id is required")
		return
	}
	if err := h.store.SetTaskPersonRoles(r.Context(), taskID, reqID, asgID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	task, err := h.store.GetTask(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// --- helpers ---

// parsePersonID — {id} из пути (UUID-строка, не число — не parseID).
func parsePersonID(r *http.Request) (store.PersonID, bool) {
	v := r.PathValue("id")
	return parsePersonIDValue(v)
}

// isHexChar — hex-символ (0-9, a-f, A-F). Выведено в функцию: staticcheck QF1001.
func isHexChar(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// parsePersonIDValue — валидация UUID-подобной строки персоны.
// UUID v4: 36 символов, 4 дефиса (8-4-4-4-12). Не строим regex: проверка структурная.
func parsePersonIDValue(v string) (store.PersonID, bool) {
	v = strings.TrimSpace(v)
	if len(v) != 36 || strings.Count(v, "-") != 4 {
		return "", false
	}
	// Структура UUID v4: hex(8-4-4-4-12); проверено, что все остальные
	// символы hex — выше (строка не пустая, длина фиксированная).
	segments := []int{8, 4, 4, 4, 12}
	idx := 0
	for i, seg := range segments {
		if i > 0 {
			if idx >= len(v) || v[idx] != '-' {
				return "", false
			}
			idx++
		}
		end := idx + seg
		if end > len(v) {
			return "", false
		}
		for j := idx; j < end; j++ {
			if !isHexChar(v[j]) {
				return "", false
			}
		}
		idx = end
	}
	return store.PersonID(v), true
}

// displayName — имя или email-идентичность для WS-сообщений.
func displayName(p *store.Person) string {
	if p != nil && p.Name != "" {
		return p.Name
	}
	return "персона"
}

// parseBoolQuery — "true"/"1" → true, "false"/"0" → false, иначе (val,false).
func parseBoolQuery(v string) (bool, bool) {
	switch strings.ToLower(v) {
	case "true", "1":
		return true, true
	case "false", "0":
		return false, true
	default:
		return false, false
	}
}
