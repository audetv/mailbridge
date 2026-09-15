// Package store — типы и интерфейс для сущностей Персона (Person) и её идентичностей.
// v0.24, шаг 6 (Режим A). Канон: docs/ontology.md §7.7 / §7.7.1.
//
// Принципы Postgres-совместимости (решение владельца 2026-09-15):
//   - PersonID — UUID-строка; в SQLite хранится как TEXT, в Postgres — нативный UUID.
//   - Booleans — INTEGER 0/1 в SQLite, BOOLEAN в Postgres; на уровне Go — bool.
//   - Связи через FK + ON DELETE SET NULL; archived — «вместо удаления».
//   - ФTS5/SQLite-специфика не используется; поиск (dev-заглушка LIKE) — только в реализации.
package store

import "time"

// PersonID — идентификатор персоны (UUID). Строчное представление,
// совпадает как с SQLite TEXT-хранилищем, так и с Postgres UUID типом.
type PersonID string

// Person — персона (действующее лицо процесса). Канон §7.7.
//
// Персона НЕ имеет роли: роль — это контекст связи
// (task.requestor_id / task.assignee_id / task_comments.author_person_id).
// Идентичность — НЕ учётная запись (мост к RBAC — будущий users.person_id).
type Person struct {
	// ID = UUID v4 (сгенерирован приложением, не базой — Postgres-совместимо).
	ID PersonID `json:"id"`

	// Name — отображаемое имя (ФИО/псевдоним). Не ключ.
	// Пустая строка = «персона в процессе узнавания» (UI показывает email).
	Name string `json:"name"`

	// Org — организация (опционально).
	Org string `json:"org,omitempty"`

	// IsInternal — hint «своя команда / внешний». Опциональная зависимость,
	// дешёво искореняема (решение владельца 2026-09-15).
	IsInternal bool `json:"is_internal"`

	// Confirmed = узнана / подтверждена; false = авто-созданная гипотеза
	// (первый контакт, подтверждение ещё не получено).
	Confirmed bool `json:"confirmed"`

	// Archived = «вместо удаления»: закрытые задачи не теряют ссылку.
	Archived bool `json:"archived"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PersonIdentity — идентичность персоны (точка входа к ней из источника).
// Канон §7.7.1.
//
// Одна персона — N идентичностей; идентичность — НЕ учётная запись и НЕ роль.
// UNIQUE(kind, value) — natural key слияния (диалект-нейтрально, Postgres-совместимо).
type PersonIdentity struct {
	// ID — UUID v4.
	ID PersonID `json:"id"`

	// PersonID — FK → persons.id (ON DELETE CASCADE).
	PersonID PersonID `json:"person_id"`

	// Kind — тип источника: email · phone (v0.24); telegram · vk · github · gitea (футурные).
	// Каждое новое значение = новая модель ядра не меняется.
	Kind string `json:"kind"`

	// Value — читаемое значение: email, phone (E.164), login/ним.
	// Для email — нормализуется (case-fold) на уровне приложения при записи и поиске.
	Value string `json:"value"`

	// ExternalID — устойчивый id провайдера (telegram user_id, github id).
	// Поисковая привязка идёт по нему прежде всего; ник/email могут меняться.
	ExternalID string `json:"external_id,omitempty"`

	// IsPrimary — является ли основной.
	IsPrimary bool `json:"is_primary"`

	// Provenance — происхождение: auto (первый контакт) · suggested (предложение принято/отвергнуто) · manual (ручной линк).
	Provenance string `json:"provenance"`

	CreatedAt time.Time `json:"created_at"`
}

// PersonFilter — параметры поиска/фильтрации справочника Персон.
type PersonFilter struct {
	// Search — частичное совпадение по name/org (LIKE; dev-заглушка, будущее — ManticoreSearch).
	Search string

	// Confirmed: nil = все, true = только подтверждённые, false = только непроверенные.
	Confirmed *bool

	// Archived: nil = все, false = только неархивные, true = только архив.
	Archived *bool

	// IsInternal: nil = все, true/false = фильтр по «своя команда».
	IsInternal *bool

	// Page/PerPage — пагинация (1-indexed).
	Page    int
	PerPage int
}

// PersonListResult — результат запроса списка Персон.
type PersonListResult struct {
	Persons []*Person `json:"persons"`
	Total   int64     `json:"total"`
	Page    int       `json:"page"`
	PerPage int       `json:"per_page"`
}

// MatchSuggestion — proposed match: «эта идентичность, может быть, относится к этой персоне».
// Генерируется fast-path (kind, value найден → привязать молча) или fuzzy-предложением.
type MatchSuggestion struct {
	// IdentityID — идентичность, по которой идёт предложение.
	IdentityID PersonID `json:"identity_id"`

	// PersonID — предложенная персона.
	PersonID PersonID `json:"person_id"`

	// Reason — пояснение: "fast_path" | "fuzzy_name" | "fuzzy_email".
	Reason string `json:"reason"`

	// At — когда предложение было сделано.
	At time.Time `json:"at"`
}

// MatchRejection — отказ на предложение слияния (отрицательное знание).
// «Это НЕ эта персона» запоминается, не стирается: система не повторяет отвергнутое предложение.
// Канон §7.7.1 (отрицательное знание).
type MatchRejection struct {
	// IdentityID — идентичность, которая НЕ относится к PersonID.
	IdentityID PersonID `json:"identity_id"`

	// PersonID — персона, к которой идентичность НЕ привязана.
	PersonID PersonID `json:"person_id"`

	// At — когда отторгнуто.
	At time.Time `json:"at"`
}
