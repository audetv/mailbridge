// Persons (v0.24, шаг 6) — реализация справочника Персон и идентичностей.
//
// Postgres-совместимость (решение владельца 2026-09-15):
//   - PK persons.id / person_identities.id — UUID (в SQLite — TEXT; в Postgres — тип UUID).
//     Идентификаторы генерируются приложением (crypto/rand, UUID v4), НЕ базой
//     — без AUTOINCREMENT, переносится на Postgres одной строкой.
//   - Booleans — INTEGER 0/1 (в Postgres → BOOLEAN).
//   - FK — ON DELETE SET NULL (arhiv — «вместо удаления»).
//   - UNIQUE(kind, value) — natural key (диалект-нейтрально).
//   - Диалект-зависимости (INSERT OR IGNORE) изолированы только в этом слое.
package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/audetv/mailbridge/internal/store"
)

// newPersonID генерирует UUID v4. Приложение-источник идентификаторов:
// на Postgres тот же формат впишется в нативный тип UUID без конвертации.
func newPersonID() (store.PersonID, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("failed to generate person id: %w", err)
	}
	// UUID v4: версия 4, вариант 1 (RFC 4122).
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return store.PersonID(fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])), nil
}

// personIDToSQL — nil-aware преобразование для SQL-записи (nil → NULL).
func personIDToSQL(id *store.PersonID) interface{} {
	if id == nil {
		return nil
	}
	return string(*id)
}

// boolToInt определён в projects.go; intToBool — только здесь.
func intToBool(i int) bool { return i != 0 }

// migratePersons — схема Персон (v0.24, шаг 6). Идемпотентно (IF NOT EXISTS).
// SQL — диалект-нейтральный (Postgres-совместим); единственная SQLite-особенность
// — INSERT OR IGNORE в backfill (изолирована в этом файле).
func (s *Store) migratePersons(ctx context.Context) error {
	dml := []string{
		`CREATE TABLE IF NOT EXISTS persons (
			id TEXT NOT NULL PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			org TEXT NOT NULL DEFAULT '',
			is_internal INTEGER NOT NULL DEFAULT 0 CHECK (is_internal IN (0,1)),
			confirmed INTEGER NOT NULL DEFAULT 0 CHECK (confirmed IN (0,1)),
			archived INTEGER NOT NULL DEFAULT 0 CHECK (archived IN (0,1)),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS person_identities (
			id TEXT NOT NULL PRIMARY KEY,
			person_id TEXT NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
			kind TEXT NOT NULL,
			value TEXT NOT NULL,
			external_id TEXT NOT NULL DEFAULT '',
			is_primary INTEGER NOT NULL DEFAULT 0 CHECK (is_primary IN (0,1)),
			provenance TEXT NOT NULL DEFAULT 'auto',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE (kind, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_person_identities_person ON person_identities(person_id)`,
		// Отрицательное знание: «это НЕ эта персона» (онтология §7.7.1).
		`CREATE TABLE IF NOT EXISTS match_rejections (
			identity_id TEXT NOT NULL,
			person_id TEXT NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
			rejected_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (identity_id, person_id)
		)`,
	}
	for _, stmt := range dml {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("persons migration failed: %w", err)
		}
	}

	// FK-колонки задач и комментариев — роли как контекст связи (онтология §7.7).
	fkColumns := map[string]struct {
		table string
		col   string
		typ   string
	}{
		"tasks.requestor_id":             {"tasks", "requestor_id", "TEXT REFERENCES persons(id) ON DELETE SET NULL"},
		"tasks.assignee_id":              {"tasks", "assignee_id", "TEXT REFERENCES persons(id) ON DELETE SET NULL"},
		"task_comments.author_person_id": {"task_comments", "author_person_id", "TEXT REFERENCES persons(id) ON DELETE SET NULL"},
	}
	for _, c := range fkColumns {
		has, err := s.columnExists(ctx, c.table, c.col)
		if err != nil {
			return fmt.Errorf("failed to check column %s.%s: %w", c.table, c.col, err)
		}
		if !has {
			if _, err := s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", c.table, c.col, c.typ)); err != nil {
				return fmt.Errorf("failed to add column %s.%s: %w", c.table, c.col, err)
			}
		}
	}
	_ = ctx // контекст используется выше; молчание linter'а

	return nil
}

// backfillPersons — первохождение: персоны из текущих данных (решение 9).
// Идемпотентно: UNIQUE(kind, value) + INSERT OR IGNORE; повторный запуск
// не создаёт дублей и не затирает подтверждённые личности.
//
// Источники: tasks.from_email (заказчик), tasks.assignee (legacy-текст,
// если это email), task_comments.author (если это email).
func (s *Store) backfillPersons(ctx context.Context) error {
	// 1. Персона + идентичность по каждому email'у из задач и комментариев.
	var emails []string
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT from_email FROM tasks
		WHERE from_email IS NOT NULL AND from_email <> ''`)
	if err != nil {
		return fmt.Errorf("backfill: list task emails: %w", err)
	}
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			_ = rows.Close()
			return err
		}
		emails = append(emails, e)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	_ = rows.Close()

	assigneeEmails, err := s.collectEmails(ctx, `
		SELECT DISTINCT a.assignee FROM tasks a
		WHERE a.assignee IS NOT NULL AND a.assignee <> '' AND instr(a.assignee, '@') > 0`)
	if err != nil {
		return fmt.Errorf("backfill: list assignee emails: %w", err)
	}
	emails = append(emails, assigneeEmails...)

	commentEmails, err := s.collectEmails(ctx, `
		SELECT DISTINCT c.author FROM task_comments c
		WHERE c.author IS NOT NULL AND c.author <> '' AND instr(c.author, '@') > 0`)
	if err != nil {
		return fmt.Errorf("backfill: list comment author emails: %w", err)
	}
	emails = append(emails, commentEmails...)

	for _, email := range emails {
		if _, err := s.EnsurePersonByEmail(ctx, email); err != nil {
			return fmt.Errorf("backfill: ensure person %q: %w", email, err)
		}
	}

	// 2. Привязываем роли уже существующих задач (requestor/assignee) —
	// идемпотентно: обновляем только пустые FK-колонки.
	if _, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET requestor_id = (
			SELECT pi.person_id FROM person_identities pi
			WHERE pi.kind = 'email' AND LOWER(pi.value) = LOWER(tasks.from_email) LIMIT 1
		)
		WHERE requestor_id IS NULL AND from_email IS NOT NULL AND from_email <> ''`); err != nil {
		return fmt.Errorf("backfill: link requestor: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET assignee_id = (
			SELECT pi.person_id FROM person_identities pi
			WHERE pi.kind = 'email' AND LOWER(pi.value) = LOWER(tasks.assignee) LIMIT 1
		)
		WHERE assignee_id IS NULL AND tasks.assignee IS NOT NULL AND tasks.assignee <> ''
		  AND instr(tasks.assignee, '@') > 0`); err != nil {
		return fmt.Errorf("backfill: link assignee: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
		UPDATE task_comments SET author_person_id = (
			SELECT pi.person_id FROM person_identities pi
			WHERE pi.kind = 'email' AND LOWER(pi.value) = LOWER(task_comments.author) LIMIT 1
		)
		WHERE author_person_id IS NULL AND task_comments.author IS NOT NULL AND task_comments.author <> ''
		  AND instr(task_comments.author, '@') > 0`); err != nil {
		return fmt.Errorf("backfill: link comment authors: %w", err)
	}

	return nil
}

// collectEmails — вспомогательный: distinct email-поля.
func (s *Store) collectEmails(ctx context.Context, query string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return out, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// normalizeEmail — case-fold при записи/поиске (решение по Postgres-схеме):
// Postgres UNIQUE case-sensitive, как и SQLite — нормализуем на уровне приложения.
func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

// --- Persons CRUD ---

func (s *Store) CreatePerson(ctx context.Context, p *store.Person) error {
	if p.ID == "" {
		id, err := newPersonID()
		if err != nil {
			return err
		}
		p.ID = id
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO persons (id, name, org, is_internal, confirmed, archived)
		VALUES (?, ?, ?, ?, ?, ?)`,
		string(p.ID), p.Name, p.Org,
		boolToInt(p.IsInternal), boolToInt(p.Confirmed), boolToInt(p.Archived))
	if err != nil {
		return fmt.Errorf("failed to create person: %w", err)
	}
	return nil
}

func (s *Store) GetPerson(ctx context.Context, id store.PersonID) (*store.Person, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, org, is_internal, confirmed, archived, created_at, updated_at
		FROM persons WHERE id = ?`, string(id))
	return scanPerson(row)
}

func scanPerson(row interface{ Scan(...interface{}) error }) (*store.Person, error) {
	p := &store.Person{}
	var isInternal, confirmed, archived int
	err := row.Scan(&p.ID, &p.Name, &p.Org, &isInternal, &confirmed, &archived, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan person: %w", err)
	}
	p.IsInternal = intToBool(isInternal)
	p.Confirmed = intToBool(confirmed)
	p.Archived = intToBool(archived)
	return p, nil
}

// scanPersonWithPrimaryEmail — как scanPerson + primary_email (колонка №7).
// NULL → пустая строка (JSON omitempty скроет пустое значение).
func scanPersonWithPrimaryEmail(row interface{ Scan(...interface{}) error }) (*store.Person, error) {
	p := &store.Person{}
	var isInternal, confirmed, archived int
	var email interface{}
	err := row.Scan(&p.ID, &p.Name, &p.Org, &isInternal, &confirmed, &archived, &email, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan person: %w", err)
	}
	if email != nil {
		p.PrimaryEmail, _ = email.(string)
	}
	p.IsInternal = intToBool(isInternal)
	p.Confirmed = intToBool(confirmed)
	p.Archived = intToBool(archived)
	return p, nil
}

func (s *Store) ListPersons(ctx context.Context, filter *store.PersonFilter) (*store.PersonListResult, error) {
	if filter == nil {
		filter = &store.PersonFilter{Page: 1, PerPage: 50}
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 200 {
		filter.PerPage = 50
	}

	where := []string{"1=1"}
	var args []interface{}
	if filter.Search != "" {
		// Диалектная заметка (Postgres): SQLite без unicode-lowercase — LIKE
		// без LOWER() (lower() — ASCII-only fallback); в Postgres → ILIKE /
		// citext нормализация. Диалект — только в store-реализации (правило схемы).
		// Поиск и по имени/организации, и по identity (email/phone): персона
		// «в процессе узнавания» имеет имя пустое — email единственный вход
		// человека в справочник (канон §7.7.1).
		like := "%" + filter.Search + "%"
		where = append(where, "(p.name LIKE ? OR p.org LIKE ? OR EXISTS ("+
			"SELECT 1 FROM person_identities pi WHERE pi.person_id = p.id AND pi.value LIKE ?))")
		args = append(args, like, like, like)
	}
	for _, f := range []struct {
		col string
		val *bool
	}{{"p.confirmed", filter.Confirmed}, {"p.archived", filter.Archived}, {"p.is_internal", filter.IsInternal}} {
		if f.val != nil {
			where = append(where, f.col+" = ?")
			args = append(args, boolToInt(*f.val))
		}
	}
	whereSQL := strings.Join(where, " AND ")

	var total int64
	countRow := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM persons p WHERE "+whereSQL, args...)
	if err := countRow.Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count persons: %w", err)
	}

	offset := (filter.Page - 1) * filter.PerPage
	// primary_email — исполнимость строки: персона без имени «в процессе
	// узнавания» (name='') — email единственная отображаемая деталь (канон §7.7.1).
	// COALESCE: основная (is_primary=1) → самая свежая email → NULL (нет
	// email вообще). Чистый SQL — Postgres-совместимо.
	query := `SELECT p.id, p.name, p.org, p.is_internal, p.confirmed, p.archived,
			COALESCE(
				(SELECT pi.value FROM person_identities pi
				 WHERE pi.person_id = p.id AND pi.kind = 'email' AND pi.is_primary = 1
				 LIMIT 1),
				(SELECT pi2.value FROM person_identities pi2
				 WHERE pi2.person_id = p.id AND pi2.kind = 'email'
				 ORDER BY pi2.created_at DESC LIMIT 1)
			) AS primary_email,
			p.created_at, p.updated_at
		FROM persons p WHERE ` + whereSQL + ` ORDER BY p.name COLLATE NOCASE LIMIT ? OFFSET ?`
	rows, err := s.db.QueryContext(ctx, query, append(args, filter.PerPage, offset)...)
	if err != nil {
		return nil, fmt.Errorf("failed to list persons: %w", err)
	}
	defer rows.Close()

	var persons []*store.Person
	for rows.Next() {
		p, err := scanPersonWithPrimaryEmail(rows)
		if err != nil {
			return nil, err
		}
		persons = append(persons, p)
	}
	if persons == nil {
		persons = []*store.Person{}
	}
	return &store.PersonListResult{
		Persons: persons,
		Total:   total,
		Page:    filter.Page,
		PerPage: filter.PerPage,
	}, nil
}

func (s *Store) UpdatePerson(ctx context.Context, p *store.Person) error {
	p.UpdatedAt = time.Now()
	res, err := s.db.ExecContext(ctx, `
		UPDATE persons SET name = ?, org = ?, is_internal = ?, confirmed = ?, archived = ?, updated_at = ?
		WHERE id = ?`,
		p.Name, p.Org, boolToInt(p.IsInternal), boolToInt(p.Confirmed), boolToInt(p.Archived),
		p.UpdatedAt, string(p.ID))
	if err != nil {
		return fmt.Errorf("failed to update person: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("person %q not found", p.ID)
	}
	return nil
}

// --- Fast-path / авто-создание (решения 4, 7) ---

// ensurePersonByEmail — internal: (kind=email, value=normalize) → персона.
// Если нет — создаёт персону (confirmed=false, решение 7) + идентичность
// (provenance='auto'). Идемпотентно через UNIQUE(kind, value).
func (s *Store) EnsurePersonByEmail(ctx context.Context, email string) (*store.Person, error) {
	email = normalizeEmail(email)
	if email == "" {
		return nil, nil
	}
	// Fast-path: идентичность уже есть?
	row := s.db.QueryRowContext(ctx, `
		SELECT person_id FROM person_identities WHERE kind = 'email' AND value = ?`, email)
	var personID string
	switch err := row.Scan(&personID); err {
	case nil:
		return s.GetPerson(ctx, store.PersonID(personID))
	case sql.ErrNoRows:
		// продолжаем
	default:
		return nil, fmt.Errorf("failed to look up email identity: %w", err)
	}

	// Персона-гипотеза (confirmed=false) + идентичность (provenance='auto').
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	pid, err := newPersonID()
	if err != nil {
		return nil, err
	}
	iid, err := newPersonID()
	if err != nil {
		return nil, err
	}
	// INSERT OR IGNORE — диалект-особенность (Postgres: ON CONFLICT DO NOTHING).
	// Гонки на backfill идемпотентны: UNIQUE(kind, value) — natural key.
	if _, err = tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO persons (id, name, org, is_internal, confirmed, archived)
		VALUES (?, '', '', 0, 0, 0)`, string(pid)); err != nil {
		return nil, fmt.Errorf("failed to insert person: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO person_identities (id, person_id, kind, value, external_id, is_primary, provenance)
		VALUES (?, ?, 'email', ?, '', 0, 'auto')`,
		string(iid), string(pid), email); err != nil {
		return nil, fmt.Errorf("failed to insert identity: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	// Возвращаем фактическую персону (при гошке — уже существующую).
	return s.FindPersonByEmail(ctx, email)
}

// upsertPersonByEmailInTx — то же, что EnsurePersonByEmail, но в уже открытой tx
// (для атомарности auto-assign с переходом статуса). Возвращает (personID, found, err):
// found=false — email пуст (нечего делать); personID пуст + err=nil — персона не
// создана из-за незначительной ошибки (авто-ассигмент best-effort, не фейлит tx).
func upsertPersonByEmailInTx(ctx context.Context, tx *sql.Tx, email string) (string, bool, error) {
	email = normalizeEmail(email)
	if email == "" {
		return "", false, nil
	}
	// Fast-path: идентичность уже есть?
	var personID string
	switch err := tx.QueryRowContext(ctx,
		`SELECT person_id FROM person_identities WHERE kind = 'email' AND value = ?`, email).Scan(&personID); err {
	case nil:
		return personID, true, nil
	case sql.ErrNoRows:
		// продолжаем
	default:
		return "", false, err
	}

	// Персона-гипотеза (confirmed=false) + идентичность (provenance='auto').
	pid, err := newPersonID()
	if err != nil {
		return "", false, err
	}
	iid, err := newPersonID()
	if err != nil {
		return "", false, err
	}
	if _, err = tx.ExecContext(ctx,
		`INSERT INTO persons (id, name, org, is_internal, confirmed, archived)
		 VALUES (?, '', '', 0, 0, 0)`, string(pid)); err != nil {
		return "", false, err
	}
	if _, err = tx.ExecContext(ctx,
		`INSERT INTO person_identities (id, person_id, kind, value, external_id, is_primary, provenance)
		 VALUES (?, ?, 'email', ?, '', 0, 'auto')`,
		string(iid), string(pid), email); err != nil {
		return "", false, err
	}
	return string(pid), true, nil
}

func (s *Store) FindPersonByEmail(ctx context.Context, email string) (*store.Person, error) {
	email = normalizeEmail(email)
	if email == "" {
		return nil, nil
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT p.id, p.name, p.org, p.is_internal, p.confirmed, p.archived, p.created_at, p.updated_at
		FROM persons p
		JOIN person_identities pi ON pi.person_id = p.id
		WHERE pi.kind = 'email' AND pi.value = ?
		ORDER BY p.created_at ASC LIMIT 1`, email)
	return scanPerson(row)
}

// --- Identities (профиль персоны: add/unlink) ---

func (s *Store) AddPersonIdentity(ctx context.Context, ident *store.PersonIdentity) (*store.PersonIdentity, error) {
	if _, err := s.GetPerson(ctx, ident.PersonID); err != nil {
		return nil, err
	}
	if ident.ID == "" {
		id, err := newPersonID()
		if err != nil {
			return nil, err
		}
		ident.ID = id
	}
	if ident.Kind == "email" {
		ident.Value = normalizeEmail(ident.Value)
	}
	if ident.Provenance == "" {
		ident.Provenance = "manual"
	}
	ident.CreatedAt = time.Now()
	// INSERT OR IGNORE — natural key UNIQUE(kind, value); повторный линк идемпотентен.
	res, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO person_identities (id, person_id, kind, value, external_id, is_primary, provenance)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		string(ident.ID), string(ident.PersonID), ident.Kind, ident.Value,
		ident.ExternalID, boolToInt(ident.IsPrimary), ident.Provenance)
	if err != nil {
		return nil, fmt.Errorf("failed to add identity: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Уже существует — возвращаем существующую (совместимость fast-path'а).
		existing, err2 := s.FindIdentity(ctx, ident.Kind, ident.Value)
		if err2 != nil || existing == nil {
			return nil, fmt.Errorf("identity (%s, %s) not created or found", ident.Kind, ident.Value)
		}
		return existing, nil
	}
	return ident, nil
}

func (s *Store) ListIdentities(ctx context.Context, personID store.PersonID) ([]*store.PersonIdentity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, person_id, kind, value, external_id, is_primary, provenance, created_at
		FROM person_identities WHERE person_id = ? ORDER BY is_primary DESC, created_at ASC`, string(personID))
	if err != nil {
		return nil, fmt.Errorf("failed to list identities: %w", err)
	}
	defer rows.Close()
	var out []*store.PersonIdentity
	for rows.Next() {
		ident, err := scanIdentity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ident)
	}
	if out == nil {
		out = []*store.PersonIdentity{}
	}
	return out, rows.Err()
}

func scanIdentity(row interface{ Scan(...interface{}) error }) (*store.PersonIdentity, error) {
	ident := &store.PersonIdentity{}
	var isPrimary int
	err := row.Scan(&ident.ID, &ident.PersonID, &ident.Kind, &ident.Value,
		&ident.ExternalID, &isPrimary, &ident.Provenance, &ident.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	ident.IsPrimary = intToBool(isPrimary)
	return ident, nil
}

func (s *Store) RemovePersonIdentity(ctx context.Context, id store.PersonID) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM person_identities WHERE id = ?`, string(id))
	if err != nil {
		return fmt.Errorf("failed to unlink identity: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("identity %q not found", id)
	}
	return nil
}

func (s *Store) FindIdentity(ctx context.Context, kind, value string) (*store.PersonIdentity, error) {
	if kind == "email" {
		value = normalizeEmail(value)
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT id, person_id, kind, value, external_id, is_primary, provenance, created_at
		FROM person_identities WHERE kind = ? AND value = ?
		ORDER BY created_at ASC LIMIT 1`, kind, value)
	return scanIdentity(row)
}

// --- Предложения / отрицательное знание (решения 4–5) ---

// SuggestMatch — fuzzy по имени (rule-based, AI — НЕ в v0.24).
// Возвращает кандидатов без учёта match_rejections (отрицательное знание).
func (s *Store) SuggestMatch(ctx context.Context, identityKind, identityValue, name string) ([]*store.Person, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return []*store.Person{}, nil
	}
	like := "%" + name + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.name, p.org, p.is_internal, p.confirmed, p.archived, p.created_at, p.updated_at
		FROM persons p
		LEFT JOIN match_rejections r
			ON r.person_id = p.id
			AND r.identity_id IN (
				SELECT i.id FROM person_identities i WHERE i.kind = ? AND i.value = ?
			)
		WHERE (p.name LIKE ? OR p.name LIKE ?)
		  AND p.archived = 0 AND r.person_id IS NULL
		ORDER BY p.confirmed DESC, p.name COLLATE NOCASE
		LIMIT 10`, identityKind, identityValue, like, name+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to suggest matches: %w", err)
	}
	defer rows.Close()
	var out []*store.Person
	for rows.Next() {
		p, err := scanPerson(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if out == nil {
		out = []*store.Person{}
	}
	return out, rows.Err()
}

// AcceptMatch — привязывает identity к person (provenance='suggested').
// Идемпотентно: UNIQUE(kind, value) не позволяет дублей; при пере-привязке
// идентичность меняется на новую персону.
func (s *Store) AcceptMatch(ctx context.Context, identityID, personID store.PersonID) error {
	ident, err := s.findIdentityByID(ctx, identityID)
	if err != nil {
		return err
	}
	if ident == nil {
		return fmt.Errorf("identity %q not found", identityID)
	}
	// Пере-привязка к новой персоне: обновляем person_id.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `
		UPDATE person_identities SET person_id = ?, provenance = 'suggested' WHERE id = ?`,
		string(personID), string(identityID)); err != nil {
		return fmt.Errorf("failed to attach identity: %w", err)
	}
	// Убираем отвергнутые совпадения для этой пары (исправление мисклика).
	if _, err = tx.ExecContext(ctx, `
		DELETE FROM match_rejections WHERE identity_id = ? AND person_id = ?`,
		string(identityID), string(personID)); err != nil {
		return fmt.Errorf("failed to clear rejection: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// RejectMatch — отрицательное знание: не повторять предложение (решение 5).
// Идемпотентно через PRIMARY KEY (identity_id, person_id).
func (s *Store) RejectMatch(ctx context.Context, identityID, personID store.PersonID) error {
	// INSERT OR IGNORE — диалект-особенность (Postgres: ON CONFLICT DO NOTHING).
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO match_rejections (identity_id, person_id) VALUES (?, ?)`,
		string(identityID), string(personID))
	if err != nil {
		return fmt.Errorf("failed to reject match: %w", err)
	}
	return nil
}

func (s *Store) findIdentityByID(ctx context.Context, id store.PersonID) (*store.PersonIdentity, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, person_id, kind, value, external_id, is_primary, provenance, created_at
		FROM person_identities WHERE id = ?`, string(id))
	return scanIdentity(row)
}

// --- Слияние персон ---

// MergePersons — идентичности source переносятся к target (решение 5: section «Не подтверждённые»).
// Ссылки задач и комментариев переключаются на target; source — archived (не удаляется).
func (s *Store) MergePersons(ctx context.Context, sourceID, targetID store.PersonID) error {
	if sourceID == targetID {
		return nil
	}
	// Проверяем существование обеих (GetPerson возвращает (nil, nil) при отсутствии).
	sp, err := s.GetPerson(ctx, sourceID)
	if err != nil || sp == nil {
		return fmt.Errorf("source person not found")
	}
	tp, err := s.GetPerson(ctx, targetID)
	if err != nil || tp == nil {
		return fmt.Errorf("target person not found")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Удаляем идентичности target'а с конфликтующим natural key (UNIQUE).
	if _, err = tx.ExecContext(ctx, `
		DELETE FROM person_identities
		WHERE person_id = ? AND (kind, value) IN (SELECT kind, value FROM person_identities WHERE person_id = ?)`,
		string(targetID), string(sourceID)); err != nil {
		return fmt.Errorf("failed to resolve identity conflicts: %w", err)
	}
	// 2. Переносим идентичности source → target (INSERT OR IGNORE — идемпотентно).
	if _, err = tx.ExecContext(ctx, `
		UPDATE person_identities SET person_id = ? WHERE person_id = ?`,
		string(targetID), string(sourceID)); err != nil {
		return fmt.Errorf("failed to move identities: %w", err)
	}
	// 3. Ссылки задач.
	for _, col := range []string{"requestor_id", "assignee_id"} {
		if _, err = tx.ExecContext(ctx, fmt.Sprintf(
			"UPDATE tasks SET %s = ? WHERE %s = ?", col, col),
			string(targetID), string(sourceID)); err != nil {
			return fmt.Errorf("failed to re-link task.%s: %w", col, err)
		}
	}
	// 4. Ссылки комментариев.
	if _, err = tx.ExecContext(ctx, `
		UPDATE task_comments SET author_person_id = ? WHERE author_person_id = ?`,
		string(targetID), string(sourceID)); err != nil {
		return fmt.Errorf("failed to re-link comments: %w", err)
	}
	// 5. Очистка match_rejections по source (стало неактуально).
	if _, err = tx.ExecContext(ctx, `
		DELETE FROM match_rejections WHERE person_id = ?`, string(sourceID)); err != nil {
		return fmt.Errorf("failed to clean rejections: %w", err)
	}
	// 6. Source — в архив (вместо удаления; онтология §7.7 archived).
	if _, err = tx.ExecContext(ctx, `
		UPDATE persons SET archived = 1, updated_at = ? WHERE id = ?`,
		time.Now(), string(sourceID)); err != nil {
		return fmt.Errorf("failed to archive source: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// --- Роли задач (вместо текста assignee) ---

// SetTaskPersonRoles привязывает/отвязывает роли задачи (решения 4–6).
// Semantics: requestorID/assigneeID nil — сохранять текущее;
// &""(пустойID) — отвязать (SET NULL); ненулевой ID — задать.
func (s *Store) SetTaskPersonRoles(ctx context.Context, taskID int64, requestorID, assigneeID *store.PersonID) error {
	// Проверяем существование задачи.
	if _, err := s.GetTask(ctx, taskID); err != nil {
		return err
	}

	// Определяем новое значение: nil → нет; иначе → значение.
	setRequestor := false
	reqVal := ""
	if requestorID != nil {
		setRequestor = true
		reqVal = string(*requestorID)
	}
	setAssignee := false
	assignVal := ""
	if assigneeID != nil {
		setAssignee = true
		assignVal = string(*assigneeID)
	}

	if setRequestor {
		if reqVal == "" {
			if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET requestor_id = NULL WHERE id = ?`, taskID); err != nil {
				return fmt.Errorf("failed to unlink requestor: %w", err)
			}
		} else {
			if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET requestor_id = ? WHERE id = ?`, reqVal, taskID); err != nil {
				return fmt.Errorf("failed to set requestor: %w", err)
			}
		}
	}
	if setAssignee {
		if assignVal == "" {
			if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET assignee_id = NULL WHERE id = ?`, taskID); err != nil {
				return fmt.Errorf("failed to unlink assignee: %w", err)
			}
		} else {
			if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET assignee_id = ? WHERE id = ?`, assignVal, taskID); err != nil {
				return fmt.Errorf("failed to set assignee: %w", err)
			}
		}
	}
	return nil
}
