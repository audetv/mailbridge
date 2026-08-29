# Mailbridge v0.22.0 — Проекты, Модули (эпики), ручные задачи, срез Plane

**Манифест разработки.** Ведётся ПО ШАГАМ: 1 шаг = 1 коммит = `make lint && make test` (+ `npm run lint` для фронта) зелёные.
Локальная ветка даёт свободу манёвра: статусы и коммиты в этом файле — локальные, корректировка плана допустима, но любой сдвиг фиксируется здесь (что, почему, какие шаги затронуты), статус не меняется молча.

**Целевая версия:** v0.22.0 (тег после merge PR в main, green CI обязателен).

---

## 0. Соглашения

### Легенда статусов шага
- `[ ]` — todo
- `[x]` — done (обязательно: хеш/сообщение коммита рядом)
- `[P]` — **отложено** (обязательно: причина и на какой шаг перенос)
- `[B]` — блокирован (на что ждём)
- `(!)` — шаг содержит риск/особенность, см. «Заметки шага»; при возврате к шагу перечитать
- Замороженные (отложенные) тесты/код — **отдельный блок §9 «Заморожено»**: что, в каком шаге отложено, в каком шаге растаивать. При достижении расмораживающего шага — тесты обязаны стать зелёными ДО коммита.

### Правила движения
1. Начать шаг: прочитать §8 (окружение) + Заметки шага + §9 (рассолить своё, если пора).
2. Критерий «шаг готов»: lint+test зелёные локально; фронт — `npm run lint` без ошибок; commit (imperative, краткий); `[x]` + hash; отметить отложенное/особенности.
3. Не начинать следующий шаг, пока текущий не `[x]`. Исключение — явные `P/B` с переносом.
4. Зависимости: стрелки `→ N` в шаге означают «сначала шаг N».

### Терминология (принято 2026-08-29)
| Контекст | Имя |
|---|---|
| UI / отчёты / письма / анализ | **Модуль** |
| Go-код, БД, API | **epic / epics** (`epic_id`, `/epics`) |
| Проект | Project / `projects` / «Проект» (едино) |

> При необходимости переименовать API/UI в «модуль» — пока API не опубликован и merge не сделан, переименование дёшево (replace в маршрутах/полях/компонентах). После v0.22.0 — уже breaking.

### Модель данных (принято)
```
projects(id, name UNIQUE, description, archived INT DEFAULT 0, created_at, updated_at)
epics(id, project_id INT REF projects(id) ON DELETE CASCADE,
      name, description, status 'open|in_progress|done', created_at, updated_at)
tasks.epic_id INT NULL REF epics(id) ON DELETE SET NULL   -- новые задачи: из письма и ручные
```
- `tasks.project` остаётся **TEXT-именем** (без FK — уже есть живые данные; инвариант: имя ∈ `projects.name`).
- Иерархия: `Проект → Модуль(эпик) → Задача`. Задача без модуля допустима.
- Прогресс эпика считается: `done = задачи со статусом completed|closed` внутри эпика.
- Удаление проекта = soft-archive; удаление эпика разрешено (задачи остаются, `epic_id → NULL`).
- `reply_log` (вместе с `plane_issue_id`) и `SaveReplyLog/ReplyExists` — **мёртвый код** (нет живых вызователей, проверено) — удаляются в Фазе 5.

### Comment kinds (расширяемый набор)
| kind | Направление | Назначение |
|---|---|---|
| `user_comment` | in/out | обычный комментарий (как сейчас) |
| `ai_verdict` | in | AI-вердикт |
| `report` | out | **внутренний отчёт**: что сделано, как, ссылки (commits/патчи) — для своих |
| `reply` | out | **черновик ответа пользователю**: простой язык, без тех. деталей |

- AI анализирует комментарий (`report`/`reply`) асинхронно (те же retry-политики очереди) и вернёт `ai_verdict` + рекомендацию статуса; **почта наружу НЕ отправляется** (отправка — отдельная будущая фича, пока не реализуем).
- Базовый механизм «approve → наружу»: статус задачи `awaiting_approval` + `approved` — фантомные до подключения отправки; wiring отправки НЕ делаем (отложить, §9).

---

## 1. ФАЗА 1 — Проекты (CRUD)

| # | Шаг | Завис. | Статус |
|---|-----|--------|--------|
| 1 | Базовая линия: ветка `feat/...` от main; `make lint`, `make test`, `npm run lint` зелёные; **зафиксировать этот манифест коммитом** | — | `[x]` 64497ab (lint 0 issues, тесты green, npm green) |
| 2 | Миграция `projects` + сида из `configs/rules.yml` (секция `# Проекты`: ТРК, Отель, Лидер Спорт, Театр, Мебельный центр, Складской комплекс, Кафе, Ледовая арена, Корпоративные сайты, + `Входящие`) + из distinct `tasks.project`; **идемпотентно** (`IF NOT EXISTS` + upsert по `name`) | 1 | `[x]` 436a872 — таблица + идемпотентный сид из rules.yml «Проекты» + distinct tasks.project; тест TestMigrate_ProjectsTable / TestMigrate_SeedsDistinctTaskProjects |
| 3 | `store.Store`: модели `Project`, методы `CreateProject / GetProject / GetProjectByName / ListProjects(includeArchived) / UpdateProject / ArchiveProject`; sqlite-реализация + table-driven тесты (sqlite-инстанс) | 2 | `[x]` 436a872 — projects.go + 10 тестов (CRUD, дубли, архив, фильтры); `make test` PASS, `golangci-lint` 0 issues |
| 4 | API `/api/projects` (GET/POST), `/api/projects/{id}` (GET/PUT/DELETE→archive, ре-активация `POST /{id}/unarchive`); валидация имени (≠ пустое, ≤ 128, не дубль); WS-события `project_created/updated/archived` | 3 | `[x]` 436a872+ — web/projects.go + 6 тестов (CRUD/404/409/фильтры/WS), `golangci-lint` 0 issues, `go test ./...` green |
| 5 | UI: `views/ProjectsView.vue` (таблица, карточка создания, inline-редактирование, archiving), вкладки Dashboard; `stores/projects.js`; WS `project_*` → тосты + рефетч | 4 | `[x]` — ProjectsView + stores/projects.js + вкладка «Проекты» в Dashboard; `npm run lint` 0 errors, `npm run build` OK |
| 6 | Ссылка проекта на задачу: `GET /api/projects/{id}/tasks` (фильтр по `t.project = name`); UI: в задачах показываем проект-ссылку | 5 | `[ ]` |

Заметки шага:
- (2) Сид обязан быть идемпотентным: второй запуск — 0 новых строк; имена читаем парсером `config.LoadRules` + отдельно секцию `# Проекты` (пока правил `RuleDef{Project}`) → отдельный `config.ProjectsFromRules() []string` (без дублей, trim).
- (3) `store.Store` интерфейс растёт: **нет фейков** в репо — все тесты на sqlite (совпадение с текущей практикой).
- (4) `DELETE /api/projects/{id}` = archive; hard-delete только пустого (`tasks` без ссылок + `epics` пустые) — до Fазы 2.

## 2. ФАЗА 2 — Модули (эпики)

| # | Шаг | Завис. | Статус |
|---|-----|--------|--------|
| 7 | Миграция `epics` + `tasks.epic_id INT NULL`; индексы; идемпотентно | 2 | `[x]` 5f1d2b0 — таблица + FK + UNIQUE(project_id,number) + tasks.epic_id (SET NULL, ALTER для старых БД) + Task.EpicID; тесты TestEpicsMigration/Idempotency |
| 8 | `store`: `Epic` + CRUD + `SetTaskEpic` + прогресс (`EpicProgress{open,done,total}`); тесты | 7 | `[ ]` |
| 9 | API `/api/projects/{id}/epics` (GET/POST), `/api/epics/{id}` (GET/PUT/DELETE); `POST/DELETE /api/epics/{id}/tasks/{taskId}`; прогресс в `GET /api/epics/{id}`; WS `epic_*` | 8 | `[ ]` |
| 10 | UI: панель модулей в `ProjectView`/списке задач; фильтр задач по модулю; карточка прогресса (Bar) | 9 | `[ ]` |
| 11 | Задача ↔ модуль: поле `epic_id` в форме создания/редактирования задачи (UI в `TaskDetailView` + API в Фазе 3 `POST /api/tasks`) | 9 | `[ ]` |

Заметки шага:
- (7) `ON DELETE SET NULL` на `tasks.epic_id` — удаление модуля не бьёт по задачам.
- (9) DELETE epic = разрешён, если задач нет (иначе 409 + подсказка переноса/удаления задач).

## 3. ФАЗА 3 — Ручные задачи + AI-проект из БД

| # | Шаг | Завис. | Статус |
|---|-----|--------|--------|
| 12 | `POST /api/tasks` (ручное создание): `project` (из БД, iname), `title`, `body`, `type`, `priority`, `epic_id`; сообщение-ид = `manual-{uuid}` (schema `message_id NOT NULL UNIQUE` не трогаем); валидация project exists и нет archived | 5 | `[ ]` |
| 13 | WS `task_created` + UI: форма «Новая задача» (modal) в Dashboard/ProjectView → `stores/tasks.js` | 12 | `[ ]` |
| 14 | **AI промты**: `orchestrator.SetProjects` теперь из `store.ListProjects()` (активные); при создании задачи из письма проект обязан ∈ активных проектов (валидация в `orchestrator.createTaskFromVerdict`) | 3, 4 | `[ ]` |
| 15 | Fallback-проект при пустом AI-классификаторе: `config.Plane.DefaultProject` → константа `processor.DefaultProject = "Входящие"` + (опц.) конфиг `MAILBRIDGE_DEFAULT_PROJECT` | 14 | `[ ]` |
| 16 | Тестируем закрытие цикла: письмо (dev) → AI → задача с проектом из правил rules.yml; ручная задача в любом проекте — обе видимы в UI | 12, 14 | `[ ]` (B: dev env) |

Заметки шага:
- (14) Критическое: промт строится из `o.projects` (оркестратор, строка ~299). После шага источник = БД, не Plane.
- (16) `(!)` B — требует включённого AI (dev :8081); без dev-окружения отложить как P с причиной и растопить в Фазе 6.

## 4. ФАЗА 4 — Отчёт + Ответ (закрытие)

| # | Шаг | Завис. | Статус |
|---|-----|--------|--------|
| 17 | Миграции `task_comments`: колонки `ai_verdict TEXT`, `approved INT NULL` (без изменения существующих); `kind` расширен (валидация в коде, БД без CONSTRAINT) + индекс `idx_task_comments_kind` | 3 | `[ ]` |
| 18 | AI-анализ комментария: `ai.CommentAnalysis(ctx, comment)`: verdict + рекомендуемый статус (completed / awaiting_approval); асинхронно через очередь (новые типы events: `comment_new` / `comment_reply`); retry на те же backoff | 14 | `[ ]` |
| 19 | API: `POST /api/tasks/{id}/comments` (add kind), `PUT /api/tasks/{id}` (update status, incl. `completed`/`awaiting_approval`), `PATCH /api/comments/{id}/approve` (approve flag → статус `approved`); статусы: добавить `awaiting_approval`, `approved` в `store/status.go` + `IsActive/IsArchived` | 18 | `[ ]` |
| 19b | Агент-юзер `hermes`: `auth.go` — 2 юзера: admin (`MAILBRIDGE_AUTH_USER/PASS`, default admin/admin) + **hermes** (логин фиксир. `hermes`, `MAILBRIDGE_AGENT_PASS` обязателен для активации; юзер существует только если пароль задан); login отдаёт username; `extractUserFromToken` уже поддерживает (токен `token-<user>-<date>`); тесты auth (table-driven) | 19 | `[ ]` |
| 20 | UI: в `TaskDetailView` — две карточки (Отчёт / Ответ пользователю); кнопки «Закрыл задачу», «Утвердил ответ» (approve); статус-бейдж `awaiting_approval`/`approved`; стили по kind/author (агент выделяем) | 19 | `[ ]` |
| 21 | **Приёмочный прогон**: dev env (login), `inbox/86` → `report` + `reply` + close → видим в UI, статус closed; WS-события приходят | 15, 20 | `[ ]` (B: dev) |

Заметки шага:
- (18) AI-анализ: промт отдельно от входящего; verdict: `{status: "completed"|"in_progress"|"awaiting_approval", reason}`; применяем только `status` + `ai_verdict`; `approved` — руками (человек), не AI.
- (21) `(!)` B — dev :8081.

## 5. ФАЗА 5 — Срез Plane (последний!)

> Зависимость: Фазы 1–4 завершены/рассолены — иначе нет замены для источника проектов и fallback.

| # | Шаг | Завис. | Статус |
|---|-----|--------|--------|
| 22 | `config`: удалить `PlanePlaneConfig` (`MAILBRIDGE_PLANE_BASE_URL/API_KEY/DEFAULT_PROJECT`) и их валидацию; обновить `config.example.env`; `config.env` (локально, не коммитится) — убрать | 15 | `[ ]` |
| 23 | Удалить пакет `internal/plane/` + импорты + `loadProjectMap` в `main.go`; `orchestrator.SetProjects` из БД (закрыть шаг 14, если P) | 14 | `[ ]` |
| 24 | Удалить health-check «plane» (`main.go`, `server_test.go`), метрики `mailbridge_plane_available` + `SetPlaneAvailable` (internal/metrics + все callers) | 23 | `[ ]` |
| 25 | **Webhook**: удалить `/webhook` + `internal/webhook/` (Планомерный аудит: секреты/тайм-зона см. §7-В); переименовать `config.HTTP` + `MAILBRIDGE_LISTEN` (было `MAILBRIDGE_WEBHOOK_LISTEN` — фактический HTTP-порт, путаница); `Makefile run/run-dev` обновить; `config.example.env` | 24 | `[ ]` |
| 26 | Удалить `reply_log` (таблица + `SaveReplyLog/ReplyExists` + `plane_issue_id`); `outbound.go` — функции `EnqueueAcknowledgement/CommentReply/StatusChange` (если мёртвы после шагов 22–25 — удалить; иначе оставить) | 25 | `[ ]` |
| 27 | **Чистая зелёная финальная**: `make lint && make test && make build`; `npm run lint && npm run build`; grep «plane/PLANE» — 0 совпадений в Go/UI/configs/docs; PR → green CI | 22–26 | `[ ]` |

Заметки шага:
- (22) `(!)` Путь: `Plane.*` есть на строках 169–174 (`Validate`) и 16, 61, 124–126 (`config/PlaneConfig`).
- (25) `(!)` КРИТИЧНО: `cfg.Webhook.Listen` = **фактический HTTP-порт** (main.go:242, 359). Секрет же — только webhook-плане (мёртвый; `validateSignature` — placeholder, `return nil`). Переименование обязано сохранить порт; тест интеграции с `:8081` (dev) в CI — проверить.

## 6. ФАЗА 6 — Финализация + релиз

| # | Шаг | Статус |
|---|-----|--------|
| 28 | Рассолить: все `P/B` шаги (16, 21, если были P в Фазе 5); тесты → зелёные | `[ ]` |
| 29 | `AGENTS.md`: обновить инварианты (plane — removed; новые kinds; «проекты/модули»; auth: admin + agent-юзер). Текст правки подготовим на шаге 28 (apply — вручную, protected). | `[ ]` |
| 30 | Docs: `docs/api.md` (новые маршруты), `docs/data-model.md` (таблицы), `docs/ARCHITECTURE.md` (сборка проектов/модулей, срез Plane), `configs/config.example.env` (нет PLANE, нет WEBHOOK_SECRET, есть LISTEN, AGENT_USER/PASS) | `[ ]` |
| 31 | Обновить root `PLAN.md`: v0.22.0 → архив `archive/PLAN.v0.22-projects-modules.md`; новая активная версия; «Plane: удалён, v0.22.0» | `[ ]` |
| 32 | PR в main, green CI (Lint + Test), merge | `[ ]` |
| 33 | Релиз: тег `v0.22.0`, бинарник GitHub (release.yml — проверить), закрывающий PR | `[ ]` |

---

## 7. Вопросы / Решения пользователя (зафиксированы 2026-08-29)

| # | Вопрос | Решение |
|---|--------|---------|
| 1 | UI «Эпик» vs «Модуль»? | UI = **Модуль**; API/БД/код = `epics`. Переименование до merge — дёшево. |
| 2 | Reply = отправка письма? | **Нет.** AI-анализ + verdict + статусы; отправка — отдельно/позже. Базовый мерж `approve→outbound` не делать. |
| 3 | Удаление проекта с задачами? | **Soft-archive**. Hard — только пустого. |
| 4 | Имена проектов сида? | Из `configs/rules.yml` секция `# Проекты` (9 названий) + `Входящие`. |
| 5 | Агент-логин? | Отдельный юзер (env): логин **`hermes`**, `MAILBRIDGE_AGENT_USER/PASS`; **не RBAC** (2 пользователя: admin + hermes). |
| 6 | `/webhook` + `MAILBRIDGE_WEBHOOK_SECRET`? | **Удалить** (мёртвый). `MAILBRIDGE_WEBHOOK_LISTEN` — **это HTTP-порт**, переименовать в `MAILBRIDGE_LISTEN`. |
| 7 | Версия фичи? | **v0.22.0** — один тег после merge. |
| 8 | Инварианты AGENTS.md? | Да, добавляем. |

---

## 8. Окружение (зафиксировано при планировании, 2026-08-29)

| Инструмент | Версия | Комментарий |
|---|---|---|
| Go | 1.26.x | `go.mod` |
| `golangci-lint` | **v2.7.2** (build go1.26) | Установлен в `$(go env GOPATH)/bin`; `PATH=$(go env GOPATH)/bin:$PATH`; `make lint` = `golangci-lint run ./...`; на HEAD: **0 issues** |
| Node | v26.7.0 | |
| npm | 11.19.0 | `frontend/` |
| CI | `.github/workflows/ci.yml` | go 1.26, golangci-lint-action v9, green CI обязателен |
| БД | SQLite (WAL) | `data/` (локально, не в git) |
| Dev-порт | 8081 (vite-прокси) `make run-dev` | Prod — `MAILBRIDGE_LISTEN` (после шага 25) |
| Логин | `MAILBRIDGE_AUTH_USER/PASS` (дефолт admin/admin) | `internal/web/auth.go`; агент — `MAILBRIDGE_AGENT_USER=hermes` + `MAILBRIDGE_AGENT_PASS` (шаг 19b) |

**При старте любой сессии:**
```sh
cd ~/work/mailbridge
export PATH=$(go env GOPATH)/bin:$PATH
make lint && make test     # Go
cd frontend && npm run lint   # фронт
```

## 9. Заморожено / отложено (пересматривать при достижении)

| # | Что | Когда заморозили | Рассолить | Комментарий |
|---|-----|------------------|-----------|-------------|
| 9.1 | Автоотправка `report`/`reply` наружу (SMTP) | Планирование v0.22 | Будущая фича | Пока только approve-флаг + статус; статистика/опыт |
| 9.2 | Источники писем ≠ email (Telegram, webform) | Планирование v0.22 | Будущий план (PLAN.v0.23) | Outbound сейчас только SMTP |
| 9.3 | RBAC / роли / боты | Планирование v0.22 | Будущая фича | Пока admin + agent (2 пользователя) |
| —   | (в процессе) — добавляем сюда все `P/B` по мере хода | | | |

## 10. Риски и грабли

- **`store.Store` — без фейков.** Все тесты на sqlite-инстансе. Новый метод → новый тест на sqlite (table-driven).
- **`message_id NOT NULL UNIQUE`** в `tasks`. Ручная задача → `manual-{uuid}`. ALTER таблицы под NULL-уникальность — дорого, не надо.
- **Порт = `cfg.Webhook.Listen`** (пока). Переименование (шаг 25) — обновить Makefile, config.env, config.example.env, документацию.
- **AI-очереди:** `ai.Queue` сейчас только inbox_items (int64). Комментарий — новый тип events или параллельная очередь. Решаем на шаге 18 (не ломать inbox-поток).
- **Гринн-критерий на каждый коммит:** не «просто компилируется», а `make lint && make test` + `npm run lint` (для фронта). CI обязателен.
- **AGENTS.md protected** — правим по согласованию (шаг 29).
- **Проект ≠ имя в БД.** Инвариант: `tasks.project` ∈ `projects.name` (валидация в API + AI-проверка). После archive — новые задачи в него НЕ создаются.
- **WS-события** — добавляем `project_*`, `epic_*`, `task_created`; проверять на шаге 21.

## 11. Критерий «всё готово для v0.22.0»

- [ ] Все шаги 1–33 `[x]`, §9 пусто (или явно `P` c причиной и планом)
- [ ] `make lint && make test && make build` — зелёные
- [ ] `npm run lint && npm run build` — зелёные
- [ ] grep `plane\|PLANE` в `internal/ cmd/ configs/ docs/ frontend/src/` — 0 совпадений
- [ ] `docs/api.md`, `docs/data-model.md`, `docs/ARCHITECTURE.md` актуальны
- [ ] `configs/config.example.env` — без `MAILBRIDGE_PLANE_*` и `MAILBRIDGE_WEBHOOK_SECRET`
- [ ] PR в main, green CI (Lint + Test), merge
- [ ] Тег `v0.22.0`, бинарник GitHub (release.yml)
