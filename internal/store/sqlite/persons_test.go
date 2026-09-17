package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
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

func TestPersons_6F_EnsurePersonFromIncoming(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// 1. Имя из FromName — авто-заполнение persons.name.
	p1, err := s.EnsurePersonFromIncoming(ctx, "Дмитрий", "dmitry@example.com")
	if err != nil || p1 == nil {
		t.Fatalf("EnsurePersonFromIncoming: %v / %v", err, p1)
	}
	if p1.Name != "Дмитрий" {
		t.Fatalf("Name: got %q", p1.Name)
	}

	// 2. Legacy-форма «Имя <email» БЕЗ закрывающей скобки (prod-БД):
	//    ParseFromHeader разбирает на (email, name) — персона создаётся,
	//    identity = чистый email (6-F п.3: чиним путь persons, не legacy).
	p3, err := s.EnsurePersonFromIncoming(ctx, "", "\u0418\u0432\u0430\u043d <ivan@x.ru")
	if err != nil || p3 == nil {
		t.Fatalf("bracketless legacy: %v / %v", err, p3)
	}
	if p3.Name != "\u0418\u0432\u0430\u043d" {
		t.Fatalf("bracketless legacy Name: got %q, want \u0418\u0432\u0430\u043d", p3.Name)
	}
	ident, err := s.FindIdentity(ctx, "email", "ivan@x.ru")
	if err != nil || ident == nil {
		t.Fatalf("FindIdentity(ivan@x.ru): %v / %v", err, ident)
	}
	if ident.PersonID != p3.ID {
		t.Fatalf("identity person: got %q, want %q", ident.PersonID, p3.ID)
	}

	// 3. Эвристика «машина»: no-reply@ без имени → org=машина.
	pm, errm := s.EnsurePersonFromIncoming(ctx, "", "no-reply@mail.example.com")
	if errm != nil || pm == nil {
		t.Fatalf("no-reply: %v / %v", errm, pm)
	}
	if pm.Org != "\u043c\u0430\u0448\u0438\u043d\u0430" {
		t.Errorf("no-reply Org: got %q", pm.Org)
	}

	// 4. Идемпотентность: (kind=email, value, case-fold) → та же персона, что case 1.
	p2, err := s.EnsurePersonFromIncoming(ctx, "Дмитрий", "DMITRY@Example.COM")
	if err != nil || p2 == nil {
		t.Fatalf("idempotent: %v / %v", err, p2)
	}
	if string(p2.ID) != string(p1.ID) {
		t.Fatalf("idempotent: different person id: %q vs %q", p2.ID, p1.ID)
	}
}

func TestPersons_6F_EnsurePersonByEmail_RejectsJunk(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()
	if p, err := s.EnsurePersonByEmail(ctx, "Имя <ivan@x.ru"); err != nil || p != nil {
		t.Fatalf("junk: got %v, %v; want nil, nil", p, err)
	}
	if p, err := s.EnsurePersonByEmail(ctx, "a.example.com"); err != nil || p != nil {
		t.Fatalf("no-at: got %v, %v; want nil, nil", p, err)
	}
}

// П.5 (6-F): self-heal — «грязная» identity (RFC822-хедер без «>») по этому
// email переименовывается в чистый; дубль не остаётся. Идемпотентно.
func TestPersons_6F_SelfHealDirtyIdentity(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	// Симулируем legacy-мусор: персона + identity «Имя <dirty@x.ru».
	pid := "test-self-heal-p1"
	if _, err := s.ExecForTest(ctx,
		`INSERT INTO persons (id, name, confirmed, is_internal, archived) VALUES (?, '', 1, 0, 0)`, pid); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ExecForTest(ctx,
		`INSERT INTO person_identities (id, person_id, kind, value, is_primary) VALUES (?, ?, 'email', 'Имя <dirty@x.ru', 1)`,
		"test-self-heal-i1", pid); err != nil {
		t.Fatal(err)
	}

	// EnsurePersonFromIncoming — чистый email → dirty переименовывается,
	// персона = та же (pid).
	p, err := s.EnsurePersonFromIncoming(ctx, "Имя", "dirty@x.ru")
	if err != nil || p == nil {
		t.Fatalf("ensure: %v / %v", err, p)
	}

	// Идентичности по email: ровно одна, value = чистый, person_id = pid.
	count, dirty := identityStats(ctx, t, s, "dirty@x.ru", "Имя <dirty@x.ru")
	if count != 1 {
		t.Fatalf("identities count: got %d, want 1", count)
	}
	if dirty != 0 {
		t.Fatalf("dirty identities: got %d, want 0", dirty)
	}
	personID, ok := identityPersonID(ctx, t, s, "dirty@x.ru")
	if !ok {
		t.Fatalf("identity person_id: missing")
	}
	if personID != pid {
		t.Fatalf("identity person_id: got %q, want %q", personID, pid)
	}

	// Идемпотентность: повторный вызов — не дублит, не ломает.
	if p2, err := s.EnsurePersonFromIncoming(ctx, "Имя", "dirty@X.RU"); err != nil || p2 == nil {
		t.Fatalf("re-ensure: %v / %v", err, p2)
	}
	if count, _ := identityStats(ctx, t, s, "dirty@x.ru", "Имя <dirty@x.ru"); count != 1 {
		t.Fatalf("re-ensure identities count: got %d, want 1", count)
	}
}

// 6-H (Red, prod 2026-09-17): чистая identity УЖЕ СУЩЕСТВУЕТ у одной персоны,
// а «сирота» с грязной identity (сырой хедер «Имя <email» без «>») — УДРУГОЙ.
// Self-heal обязан: НЕ упасть с UNIQUE(kind,value), вернуть чистую персону,
// убрать грязную identity, сирота не остаётся с identity на этот email;
// идемпотентно на повторе. (Прод: update шёл ДО дедуп-DELETE — коллизия,
// exit 1, crash-loop.)
func TestPersons_6H_SelfHealDirtyIdentityWhenCleanAlreadyExists(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	const (
		email     = "gusev@gc.example"
		cleanPID  = "person-6h-clean"  // «хозяин» чистой identity
		orphanPID = "person-6h-orphan" // сирота с грязной identity (prod: name/org пустые)
	)
	if _, err := s.ExecForTest(ctx,
		`INSERT INTO persons (id, name, confirmed, is_internal, archived) VALUES ('person-6h-clean', 'Гусев Алексей', 1, 0, 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ExecForTest(ctx,
		`INSERT INTO persons (id, name, confirmed, is_internal, archived) VALUES ('person-6h-orphan', '', 1, 0, 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ExecForTest(ctx,
		fmt.Sprintf(`INSERT INTO person_identities (id, person_id, kind, value, is_primary) VALUES ('person-6h-i1', 'person-6h-clean', 'email', %q, 1)`, email)); err != nil {
		t.Fatal(err)
	}
	dirtyVal := "Гусев Алексей <" + email
	if _, err := s.ExecForTest(ctx,
		fmt.Sprintf(`INSERT INTO person_identities (id, person_id, kind, value, is_primary) VALUES ('person-6h-i2', 'person-6h-orphan', 'email', %q, 0)`, dirtyVal)); err != nil {
		t.Fatal(err)
	}
	// Задача «сироты» (prod: 384/394/404): после merge должна оказаться
	// за «хозяином» чистой identity.
	if _, err := s.ExecForTest(ctx,
		fmt.Sprintf(`INSERT INTO tasks (message_id, subject, from_email, requestor_id, assignee_id) VALUES ('6h-task-1', 'Задача сироты', %q, %q, %q)`, email, orphanPID, orphanPID)); err != nil {
		t.Fatal(err)
	}

	// 1. Повторный self-heal — НЕ падает с UNIQUE, возвращает «хозяина»
	//    чистой identity (сирота — дубль того же человека).
	p, err := s.EnsurePersonFromIncoming(ctx, "Гусев Алексей", email)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if p == nil {
		t.Fatal("ensure: nil person")
	}
	if string(p.ID) != cleanPID {
		t.Fatalf("returned person: got %q, want %q", p.ID, cleanPID)
	}

	// 2. Сирота не остаётся с identity на этот email (ни грязной, ни чистой),
	//    чистая — одна, за настоящим владельцем.
	var n int
	if err := s.QueryRowForTest(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM person_identities WHERE value=%q OR value=%q`, email, dirtyVal)).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("identities for email: got %d rows, want exactly 1 (clean, owner's)", n)
	}
	if pid, ok := identityPersonID(ctx, t, s, email); !ok || pid != cleanPID {
		t.Fatalf("identity person_id: got %q (%v), want %q", pid, ok, cleanPID)
	}

	// 3. Задачи сироты — за настоящим владельцем (merge, а не «дубль живёт»).
	if tid, ok := taskIDForPerson(ctx, t, s, orphanPID); ok {
		if got, ok := taskOwner(ctx, t, s, tid); !ok || got != cleanPID {
			t.Fatalf("task %s owner: got %q (%v), want %q", tid, got, ok, cleanPID)
		}
	}

	// 4. Идемпотентность: повтор — не падает, не дуплит, та же персона.
	if p2, err := s.EnsurePersonFromIncoming(ctx, "", email); err != nil || p2 == nil || string(p2.ID) != cleanPID {
		t.Fatalf("re-ensure: %v / %v / %q", err, p2, cleanPID)
	}
	if n2 := identityCount(ctx, t, s, email); n2 != 1 {
		t.Fatalf("re-ensure identities: got %d, want 1", n2)
	}
}

// identityCount: сколько всего identity совпадает по чистому email.
func identityCount(ctx context.Context, t *testing.T, s *sqlite.Store, email string) int {
	t.Helper()
	var n int
	if err := s.QueryRowForTest(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM person_identities WHERE value = %q`, email)).Scan(&n); err != nil {
		t.Fatalf("identityCount: %v", err)
	}
	return n
}

// taskIDForPerson: первая (по id) задача, где персона — requestor или assignee.
func taskIDForPerson(ctx context.Context, t *testing.T, s *sqlite.Store, pid string) (string, bool) {
	t.Helper()
	var id int64
	err := s.QueryRowForTest(ctx,
		fmt.Sprintf(`SELECT id FROM tasks WHERE requestor_id = %q OR assignee_id = %q ORDER BY id LIMIT 1`, pid, pid)).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false
		}
		t.Fatalf("taskIDForPerson: %v", err)
	}
	return fmt.Sprintf("%d", id), true
}

// taskOwner: кто владелец (assignee) задачи; fallback — requestor.
func taskOwner(ctx context.Context, t *testing.T, s *sqlite.Store, taskID string) (string, bool) {
	t.Helper()
	var owner string
	err := s.QueryRowForTest(ctx,
		fmt.Sprintf(`SELECT coalesce(assignee_id, requestor_id) FROM tasks WHERE id = %q`, taskID)).Scan(&owner)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false
		}
		t.Fatalf("taskOwner: %v", err)
	}
	return owner, true
}

func identityStats(ctx context.Context, t *testing.T, s *sqlite.Store, clean, dirty string) (count, dirtyN int) {
	t.Helper()
	var n int
	if err := s.QueryRowForTest(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM person_identities WHERE kind='email' AND value = %q`, clean)).Scan(&n); err != nil {
		t.Fatalf("count clean: %v", err)
	}
	var dn int
	if err := s.QueryRowForTest(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM person_identities WHERE kind='email' AND value = %q`, dirty)).Scan(&dn); err != nil {
		t.Fatalf("count dirty: %v", err)
	}
	return n, dn
}

func identityPersonID(ctx context.Context, t *testing.T, s *sqlite.Store, value string) (string, bool) {
	t.Helper()
	var pid string
	err := s.QueryRowForTest(ctx,
		fmt.Sprintf(`SELECT person_id FROM person_identities WHERE kind='email' AND value = %q`, value)).Scan(&pid)
	return pid, err == nil
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
	// 6-F: identity = чистый email из extractor (RFC822-заголовок — мусор, отклоняется валидатором).
	p, err := s.EnsurePersonByEmail(ctx, "agusev@gcconsulting.ru")
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

// 6-H (часть 2, Red, prod 2026-09-17): авторы входящих в task_comments
// хранятся raw-строкой хедера («Имя <email>» — так пишет processor).
// auto-assign (ensurePersonInTx → upsertPersonByEmailInTx) обязан
// записать identity ТОЛЬКО с чистым email — сырой хедер в
// person_identities не должен попадать. (Прод: ровно так и попадал —
// source грязных identity.)
func TestPersons_6H_AutoAssignDoesNotStoreRawHeader(t *testing.T) {
	s, cleanup := setupStore(t)
	defer cleanup()
	ctx := context.Background()

	mustCreateTask(t, s, &store.Task{MessageID: "m6h2", Subject: "t6h2", Status: string(store.StatusNew)})
	rawAuth := "Гусев Алексей <agusev6h@gc.example>"
	comment := &store.TaskComment{
		TaskID:    1,
		Author:    rawAuth,
		Body:      "подтверждаю выполнение",
		Direction: "in",
	}
	if err := s.AddTaskComment(ctx, comment); err != nil {
		t.Fatalf("AddTaskComment: %v", err)
	}
	if err := s.SetTaskStatus(ctx, 1, "completed", "admin"); err != nil {
		t.Fatalf("SetTaskStatus: %v", err)
	}

	// 1. Чистая identity — ровно одна.
	const cleanEmail = "agusev6h@gc.example"
	if n := identityCount(ctx, t, s, cleanEmail); n != 1 {
		t.Fatalf("clean identity count: got %d, want 1", n)
	}
	// 2. Сырой хедер в identities НЕ записан.
	if n := identityCount(ctx, t, s, rawAuth); n != 0 {
		t.Fatalf("raw header in identities: got %d rows, want 0", n)
	}
	// 3. Персона создана (один email — одна персона).
	pid, ok := identityPersonID(ctx, t, s, cleanEmail)
	if !ok || pid == "" {
		t.Fatalf("identity owner: want non-empty person id, got ok=%v pid=%q", ok, pid)
	}
	// 4. Assignee связан с этой персоной.
	task, err := s.GetTask(ctx, 1)
	if err != nil || task == nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.AssigneeID == nil || *task.AssigneeID != store.PersonID(pid) {
		t.Fatalf("assignee: got %v, want %q", task.AssigneeID, pid)
	}
}
