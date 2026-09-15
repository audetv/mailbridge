package sqlite_test

import (
	"context"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
)

func TestPersons_MigrationCreatesTables(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()
	for _, table := range []string{"persons", "person_identities"} {
		exists, err := s.TableExists(ctx, table)
		if err != nil {
			t.Fatalf("TableExists(%s): %v", table, err)
		}
		if !exists {
			t.Errorf("%s table not created", table)
		}
	}
}

func TestPersons_Crud(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()
	internal := true
	p := &store.Person{Name: "Иван Петров", Org: "ООО Ромашка", IsInternal: internal}
	if err := s.CreatePerson(ctx, p); err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}
	if p.ID == "" {
		t.Fatal("CreatePerson: empty id")
	}
	got, err := s.GetPerson(ctx, p.ID)
	if err != nil || got == nil {
		t.Fatalf("GetPerson: %v / %v", err, got)
	}
	if got.Name != "Иван Петров" || got.Org != "ООО Ромашка" || !got.IsInternal {
		t.Fatalf("wrong person: %+v", got)
	}
	res, err := s.ListPersons(ctx, &store.PersonFilter{Search: "Ромаш"})
	if err != nil || res.Total != 1 {
		t.Fatalf("ListPersons: total=%v err=%v", res, err)
	}
	if len(res.Persons) != 1 || res.Persons[0].ID != p.ID {
		t.Fatalf("ListPersons rows: %+v", res.Persons)
	}
}

// Персона без имени («в процессе узнавания»): в списке виден её email,
// поиск находит её по email (канон §7.7.1).
func TestPersons_ListPrimaryEmailAndSearchByIdentity(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()
	p, err := s.EnsurePersonByEmail(ctx, "Гусев Алексей <agusev@gcconsulting.ru>")
	if err != nil || p == nil {
		t.Fatalf("EnsurePersonByEmail: %v / %v", err, p)
	}
	res, err := s.ListPersons(ctx, &store.PersonFilter{Page: 1, PerPage: 10})
	if err != nil || res.Total != 1 {
		t.Fatalf("ListPersons: %v / %v", res, err)
	}
	if len(res.Persons) != 1 || res.Persons[0].PrimaryEmail == "" {
		t.Fatalf("expected primary_email in list row, got %+v", res.Persons)
	}
	// Поиск по фрагменту email-идентичности.
	res, err = s.ListPersons(ctx, &store.PersonFilter{Search: "gcconsulting"})
	if err != nil || res.Total != 1 {
		t.Fatalf("search by identity: total=%v err=%v", res, err)
	}
	// Поиск по несуществующему email не находит.
	res, err = s.ListPersons(ctx, &store.PersonFilter{Search: "whoever@nowhere.example"})
	if err != nil || res.Total != 0 {
		t.Fatalf("search no-match: total=%v err=%v", res, err)
	}
}

func TestPersons_EnsureByEmail_Idempotent(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()
	p1, err := s.EnsurePersonByEmail(ctx, "User@Example.com")
	if err != nil || p1 == nil {
		t.Fatalf("EnsurePersonByEmail: %v / %v", err, p1)
	}
	p2, err := s.EnsurePersonByEmail(ctx, "user@example.com")
	if err != nil || p2 == nil {
		t.Fatalf("EnsurePersonByEmail #2: %v / %v", err, p2)
	}
	if p1.ID != p2.ID {
		t.Fatalf("different persons for same email: %q vs %q", p1.ID, p2.ID)
	}
	ident, err := s.FindIdentity(ctx, "email", "user@example.com")
	if err != nil || ident == nil {
		t.Fatalf("FindIdentity: %v / %v", err, ident)
	}
}

func TestPersons_IdentityAndSuggest(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()
	p, _ := s.EnsurePersonByEmail(ctx, "anna@corp.ru")
	if p == nil {
		t.Fatal("no person")
	}
	ident, err := s.AddPersonIdentity(ctx, &store.PersonIdentity{
		PersonID: p.ID, Kind: "phone", Value: "+7 900 000-00-00",
	})
	if err != nil || ident == nil {
		t.Fatalf("AddPersonIdentity: %v / %v", err, ident)
	}
	list, err := s.ListIdentities(ctx, p.ID)
	if err != nil || len(list) < 2 {
		t.Fatalf("ListIdentities: %v / %d", err, len(list))
	}
	if err := s.RemovePersonIdentity(ctx, ident.ID); err != nil {
		t.Fatalf("RemovePersonIdentity: %v", err)
	}
}

func TestPersons_Merge(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()
	src, _ := s.EnsurePersonByEmail(ctx, "ivan@corp.ru")
	tgt, _ := s.EnsurePersonByEmail(ctx, "Ivan.Petrov@corp.ru")
	if src == nil || tgt == nil {
		t.Fatal("ensure failed")
	}
	name := "Иван Петров"
	tgt.Name = name
	if err := s.UpdatePerson(ctx, tgt); err != nil {
		t.Fatalf("UpdatePerson: %v", err)
	}
	if err := s.MergePersons(ctx, src.ID, tgt.ID); err != nil {
		t.Fatalf("MergePersons: %v", err)
	}
	got, _ := s.GetPerson(ctx, src.ID)
	if got == nil || !got.Archived {
		t.Fatalf("source not archived: %+v", got)
	}
	gotTgt, _ := s.GetPerson(ctx, tgt.ID)
	if gotTgt == nil || gotTgt.Name != name {
		t.Fatalf("target wrong: %+v", gotTgt)
	}
	found, err := s.FindIdentity(ctx, "email", "ivan@corp.ru")
	if err != nil || found == nil {
		t.Fatalf("identity lost after merge: %v / %v", err, found)
	}
	if found.PersonID != tgt.ID {
		t.Fatalf("identity not moved to target: %q != %q", found.PersonID, tgt.ID)
	}
}

func TestPersons_SuggestCyrillic(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()
	p := &store.Person{Name: "Иван Петров"}
	if err := s.CreatePerson(ctx, p); err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}
	got, err := s.SuggestMatch(ctx, "email", "ivan@corp.ru", "Петров")
	if err != nil {
		t.Fatalf("SuggestMatch: %v", err)
	}
	if len(got) != 1 || got[0].ID != p.ID {
		t.Fatalf("expected suggestion by cyrillic name, got %d", len(got))
	}
}

func TestPersons_SuggestRespectsRejection(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()
	p := &store.Person{Name: "Иван Петров"}
	if err := s.CreatePerson(ctx, p); err != nil {
		t.Fatalf("CreatePerson: %v", err)
	}
	ident := &store.PersonIdentity{PersonID: p.ID, Kind: "email", Value: "ivan@corp.ru", Provenance: "auto"}
	saved, err := s.AddPersonIdentity(ctx, ident)
	if err != nil || saved == nil {
		t.Fatalf("AddPersonIdentity: %v / %v", err, saved)
	}
	got, err := s.SuggestMatch(ctx, "email", "ivan@corp.ru", "Петров")
	if err != nil || len(got) != 1 {
		t.Fatalf("expected 1 candidate, got %d / %v", len(got), err)
	}
	if err := s.RejectMatch(ctx, saved.ID, p.ID); err != nil {
		t.Fatalf("RejectMatch: %v", err)
	}
	got, err = s.SuggestMatch(ctx, "email", "ivan@corp.ru", "Петров")
	if err != nil {
		t.Fatalf("SuggestMatch after reject: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("rejected candidate must be filtered, got %d", len(got))
	}
}

func TestPersons_ListTasksByRequestor(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()
	p, _ := s.EnsurePersonByEmail(ctx, "client@demo.ru")
	mustCreateTask(t, s, &store.Task{MessageID: "m1", Subject: "t1", Status: string(store.StatusNew)})
	task, _ := s.GetTask(ctx, 1)
	if err := s.SetTaskPersonRoles(ctx, task.ID, &p.ID, nil); err != nil {
		t.Fatalf("SetTaskPersonRoles: %v", err)
	}
	res, err := s.ListTasks(ctx, &store.TaskFilter{RequestorID: &p.ID, Page: 1, PerPage: 10})
	if err != nil || len(res.Tasks) != 1 {
		t.Fatalf("ListTasks by requestor: %d / %v", len(res.Tasks), err)
	}
}

func TestPersons_AutoAssignOnClose(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()
	mustCreateTask(t, s, &store.Task{MessageID: "m2", Subject: "t2", Assignee: "client@demo.ru", Status: string(store.StatusNew)})
	comment := &store.TaskComment{
		TaskID:    1,
		Author:    "Client@DEMO.ru",
		Body:      "подтверждаю выполнение",
		Direction: "in",
	}
	if err := s.AddTaskComment(ctx, comment); err != nil {
		t.Fatalf("AddTaskComment: %v", err)
	}
	if err := s.SetTaskStatus(ctx, 1, "completed", "admin"); err != nil {
		t.Fatalf("SetTaskStatus: %v", err)
	}
	task, err := s.GetTask(ctx, 1)
	if err != nil || task == nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.AssigneeID == nil || *task.AssigneeID == "" {
		t.Fatal("expected auto-assigned person after close")
	}
	if person, _ := s.GetPerson(ctx, *task.AssigneeID); person == nil || person.Confirmed {
		t.Fatalf("unexpected auto-assigned person: %+v", person)
	}
}
