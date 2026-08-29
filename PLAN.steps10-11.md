# Подплан v0.22 — Фаза 2, шаги 10–11 (Проект → Модуль → Задача)

Статус: **ШАГ 10 ГОТОВ (2026-08-29, коммиты ed4a80e→1963152); шаг 11 — в работе**. Дата: 2026-08-29.
Метод: каждый подшаг = 1 коммит; acceptance = зелёные тесты + линт до коммита.
Правило сессии: НЕ перечитывать код — контракт ниже финален, вопросы только в §4.

---

## 1. ЗАФИКСИРОВАННЫЙ API CONTRACT (не менять, не «переоткрывать»)

### 1.1 Эпики (коммиты шагов 7–9, `db5d4e7`) — ГОТОВО
| Метод | Путь | Request | Response |
|---|---|---|---|
| GET | `/api/projects/{projectId}/epics` | — | `200 [Epic]` |
| GET | `/api/epics/{id}` | — | `200 EpicDetail` (flat, см. ниже) |
| POST | `/api/projects/{projectId}/epics` | `{name, description?}` | `201 Epic` |
| PUT | `/api/epics/{id}` | `{name?, description?, status?}` | `200 Epic` |
| DELETE | `/api/epics/{id}` | — | `204`; задачи эпика → `epic_id = NULL` |

```jsonc
// Epic
{"id":1,"project_id":3,"number":2,"name":"Строчка","description":"...","status":"open",
 "created_at":"2026-08-29T10:00:00Z","updated_at":"2026-08-29T10:00:00Z"}
// EpicDetail = Epic (flat) + progress
{"id":1,"project_id":3,"number":2,"name":"Строчка","description":"...","status":"open",
 "created_at":"...","updated_at":"...","progress":{"total":10,"open":4,"done":6}}
```
- WS-события: `epic_created`, `epic_updated`, `epic_deleted` (из `internal/web/epics.go`).
- **Валидные статусы модуля (FROZEN, из `store.go` + `epics.go`):** `open | in_progress | done`; дефолт `open`. (Список `active|done|closed|archived` из предыдущих черновиков — НЕВАЛИДЕН, не использовать.)
- `Epic` JSON-форма (FROZEN): `{"id", "project_id", "number", "name", "description", "status", "created_at", "updated_at"}`.
- `EpicDetail` JSON-форма (FROZEN, flat): `Epic` + `"progress": {"total", "open", "done"}` — поля эпика ВЕРХНЕГО уровня, обёртки `epic` НЕТ (структура `GetEpicDetail` в `internal/web/epics.go`).
- Ошибки: как в существующих handlers (`404 {"error":"not found"}`, `400` валидация).

### 1.2 Задачи × эпики (шаг 10) — бекенд ЗАКОММИБЛЕН (`d51056b`)
- `GET /api/tasks?epic_id={int}` → `200 [Task]`; `Task` несёт `epic_id: int|null` (имя модуля фронт берёт из epics-store, см. 10.5).
- `PATCH /api/tasks/{id}` — body допускает `epic_id: int|null` (`null` → отвязать).
- Тест: `TestListTasks_EpicFilter` (в `internal/store/sqlite/epics_test.go`).
- Статус: `go test ./internal/...` green, `golangci-lint` 0 issues.

### 1.3 Ручное создание задач (шаг 11) — НОВЫЙ ЭНДПОИНТ
- `POST /api/tasks` — контракт зафиксирован в **§1.4** (реальный, после сверки со схемой): обязательны `title` + `project` (имя), опционально `description`, `epic_id`.
- `message_id` = `"manual-" + 16 hex` (crypto/rand; не коллидит с реальными email-сообщениями; UNIQUE по таблице соблюдается).
- Статус создаётся всегда как `new` — в запросе не передаётся.
- Реальная схема (проверено по коду, 2026-08-29): задача сшивается с проектом по **имени** (`tasks.project`, TEXT) — поля `project_id` в tasks НЕТ, поэтому запрос принимает `project` (имя). `due_date` в схеме задач НЕТ. `description` → `body_text`.
- Валидация: `title` пуст/>500 или `project` пуст → `400`; проект не найден → `404`; `epic_id` не существует или чужого проекта → `400`.
- Поведение: после создания → WS `task_created` (паттерн уже есть в `processor.go`/`ai/worker.go` — `WSEvent{Type, TaskID, Message, Data}`), задача → `201`.

### 1.4 ИСПРАВЛЕНИЕ КОНТРАКТА (2026-08-29, после реализации 11.1)
Исходный черновик §1.3 предполагал `project_id` — это неверно для текущей схемы. Фактический закоммиченный контракт (`507456d`):
```jsonc
// POST /api/tasks — Request (title и project — обязательны; остальное — опционально)
{"title":"Зафиксировать контракт","project":"Деск X",
 "description":"...","epic_id":1}
// Response 201 — созданная задача (стандартная JSON-форма Task)
```

---

## 2. ШАГ 10 — «Панель модулей + фильтр по модулю»

Правило тестов (согласовано 2026-08-29):
- **10.0 — тестовая обвязка фронта** (vitest + happy-dom + @vue/test-utils, `npm test`, пилотный тест стора, `npm test` в `ci.yml` джоба lint).
- **Тесты идут в том же коммите, что и код** — для 10.3–10.6 и 11.2–11.3.
- Покритичность (дефолт, согласован): тесты сторов + юнит-тесты ключевых компонентов (фильтр «Модуль», колонка, WS-handlers, диалог создания задачи). Остальные правки — без отдельных тестов. E2E — не сейчас (отдельная фаза).

| # | Комит (сообщение) | Файлы | Что делаем | Acceptance | Статус |
|---|---|---|---|---|---|
| 10.0 | `chore: frontend — vitest + happy-dom + @vue/test-utils (v0.22 step 10)` | `frontend/package.json` (devDeps), `frontend/vitest.config.js`, `frontend/tests/setup.js`, `frontend/tests/stores/epics.spec.js`, `.github/workflows/ci.yml` | devDeps + `"test": "vitest run"` + конфиг happy-dom + setup (PrimeVue-тема, localStorage) + 12 тестов стора + `npm test` в джобе lint CI | `npm test` + `npm run lint` + `npm run build` green | ✅ `ed4a80e` + `2a077ed` |
| 10.1 | `feat: tasks — epic_id filter + PATCH epic_id (v0.22 step 10)` | `internal/store/store.go`, `internal/store/sqlite/sqlite.go`, `internal/web/api.go`, `internal/store/sqlite/epics_test.go` | Код УЖЕ в working tree (EpicID в TaskFilter/SQL/allowlist + тест). Только финальный линт и коммит. | `go test ./internal/...` + `golangci-lint` green | ✅ `d51056b` |
| 10.2 | `feat: ui — epics store (v0.22 step 10)` | `frontend/src/stores/epics.js` | Стор готов (124 строки, flat API) + фикс `fetchDetail` (flat, не `data.epic`). | eslint 0 ошибок | ✅ `db2eed6` |
| 10.3 | `feat: ui — epic panel in projects view (v0.22 step 10)` | `frontend/src/components/EpicPanel.vue` (новый), `frontend/src/views/ProjectsView.vue`, `frontend/tests/components/EpicPanel.spec.js` | Компонент: список (номер, имя, Tag статуса, ProgressBar), создание, смена статуса, удаление + кнопка «Модули» в таблице проектов + 6 тестов. | lint+test+build; видно панель с 1+ модулем | ✅ `b68b840` |
| 10.4 | `feat: ui — epic filter in filter bar (v0.22 step 10)` | `frontend/src/components/FilterBar.vue`, `frontend/src/stores/tasks.js`, `frontend/tests/components/FilterBar.spec.js` | Select «Модуль» (disabled без проекта), опции из `/projects/{id}/epics`, `epic_id → GET /tasks?epic_id=`; фикс perPage→per_page; 4 теста. | lint+test+build; фильтр реально уменьшает список | ✅ `141d803` |
| 10.5 | `feat: ui — epic column in task table (v0.22 step 10)` | `frontend/src/components/TaskTable.vue`, `frontend/tests/components/TaskTable.spec.js` | Колонка «Модуль»: имя из epics-store (`epicById`), `—` если нет; 1 тест. | lint+test+build | ✅ `d786c5d` |
| 10.6 | `feat: ui — epic WS events in dashboard (v0.22 step 10)` | `frontend/src/views/DashboardView.vue`, `frontend/tests/views/DashboardEpicWS.spec.js`, `frontend/tests/setup.js` | `epic_created/updated/deleted` → перечитывание стора epics (если проект открыт) + 3 теста. | lint+test+build | ✅ `1963152` |

> Порядок строго 10.1 → 10.6, по одному коммиту за раз. UI-коммиты не требуют Go-тестов, но требуют `npm run lint && npm run build`.

## 3. ШАГ 11 — «Ручная задача + поле «Модуль» в задаче»

**Решено пользователем (2026-08-29):**
- Кнопка «Создать задачу» — в **2 местах**: DashboardView (вкладка «Активные задачи») и ProjectsView (список проектов).
- Диалог создания: **обязательно — только заголовок** (+ проект: из контекста в ProjectsView, выбор в Dashboard). Всё остальное необязательно. **Статус всегда `new`** — в диалоге выбора статуса нет.
- После создания → **автопереход на `/tasks/{id}`** — там уже форма полного редактирования (сайдбар TaskDetailView: проект, статус, приоритет, тип, исполнитель; + «Модуль» из 11.2). «Режим редактирования» — существующая страница задачи, ничего нового строить не надо.

| # | Коммит (сообщение) | Файлы | Что делаем | Acceptance | Статус |
|---|---|---|---|---|---|
| 11.1 | `feat: api — manual task creation POST /api/tasks (v0.22 step 11)` | `internal/web/api.go`, `internal/web/api_createtask_test.go` (новый), `internal/web/api_test.go` (helper), `cmd/mailbridge/main.go` | Handler `CreateTask`: контракт §1.4 (`title`+`project` обязательны; `description`, `epic_id` опц.; `epic_id` не существует/чужой → 400). `message_id = manual-{16hex}`, статус `new`. WS `task_created`. Тесты: OK/минимум/валидация/404/эпик-валидация/405. | `go test ./internal/...` + lint green | ✅ `507456d` |
| 11.2 | `feat: ui — module field in task detail (v0.22 step 11)` | `frontend/src/views/TaskDetailView.vue` + тест | Select «Модуль» в сайдбаре (эпики проекта задачи, сброс → `epic_id:null`) → `PATCH /api/tasks/{id}`. Тест: выбор модуля шлёт PATCH с epic_id, сброс шлёт null. | test+lint+build | ⬜ |
| 11.3 | `feat: ui — create task dialog in dashboard and projects (v0.22 step 11)` | `frontend/src/components/CreateTaskDialog.vue` (новый) + `DashboardView.vue` + `ProjectsView.vue` + тест | Dialog «Создать задачу»: заголовок (обязательно) + проект (из контекста ProjectsView / Select в Dashboard) + модуль (опция, эпиками проекта) → `POST /api/tasks` → redirect `/tasks/{id}`. Кнопки в обоих местах. Тест: валидация title, payload, redirect. | test+lint+build | ⬜ |

Финальная проверка всей фазы: `make test && make lint && npm run lint && npm run build`, затем — только после твоего «ок»: PR → CI → merge → tag v0.22.0 (по общему PLAN).

## 4. ВОПРОСЫ — РЕШЕНЫ (2026-08-29)

1. ✅ **Поля ручной задачи:** обязательно `title` (+ `project_id` из контекста/выбора); опц. `description`, `epic_id`. Статус всегда `new`.
2. ✅ **Где «Создать задачу»:** диалог (компонент `CreateTaskDialog`), кнопка в DashboardView и ProjectsView. После создания — переход на задачу.
