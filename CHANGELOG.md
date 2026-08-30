# Changelog

## [0.22.0] - 2026-08-30

**Проекты → Модули → Задачи, Approve, Срез Plane (ADR-0001).** Plane/webhook из системы удалены целиком.

### Added
- Иерархия **Проекты → Модули → Задачи** (API/UI: `epics`, «Модули»): таблицы `projects`/`epics` (UNIQUE `name`, `UNIQUE(project_id, number)`), `tasks.epic_id` (nullable, `ON DELETE SET NULL`); задача без модуля допустима
- API проектов: список (`archived=`/`search=`), CRUD, SOFT-archive (`DELETE`) + `unarchive`, «в задачи» (`GET /api/projects/{id}/tasks`); WS `project_updated/archived/unarchived`
- API модулей: CRUD, прогресс (`done/total/pct`), привязка/отвязка задач (`POST|DELETE /api/epics/{id}/tasks/{taskId}`); WS `epic_*`
- UI: вкладка «Проекты» (ProjectsView), панель модулей (EpicPanel, Bar-прогресс), фильтр «Проект»/«Модуль», колонка «Модуль» в задачах, поле «Модуль» в форме задачи, кнопка «К задачам»; URL — source of truth (`?tab=`, deep-link `?project=`)
- Ручное создание задач: `POST /api/tasks` (диалог; archived-проект → `400`)
- AI: контекст проектов **из SQLite** в промпт + fallback-проект `MAILBRIDGE_DEFAULT_PROJECT` (проект «Входящие» из AI не создаётся)
- «Одна модель в памяти»: system-prompt (`MAILBRIDGE_AI_SYSTEM_FILE`/`_PROMPT`) и `MAILBRIDGE_AI_TEMPERATURE` из конфига (вынесено из Modelfile)
- **Approve (черновики ответов)**: `task_comments.approved` — флаг **комментария** (не статус задачи); `PATCH /api/comments/{id}/approve` — admin-only, idempotent, WS `comment_approved`; комментарии с `kind`: `user_comment` (входящие) / `report` (внутренний отчёт) / `reply` (черновик ответа)
- UI Approve: бейджи «Отчёт»/«Ответ пользователю», кнопка «Ответ утвердить» (в CommentList; в ReplyForm её нет — дубль/мисклик исключён); e2e: hermes отчитался → admin утверждает → бейдж «Утверждено»
- Агентский юзер `hermes` — включается при заданном `MAILBRIDGE_AGENT_PASS` (иначе login → 401)
- E2E-инфраструктура: Playwright + `e2e:seed` (идемпотентный: проекты + модуль + задачи, устойчив к prod→dev-копии БД) + прогон A–D «проект ↔ задачи» (7/7) + approve-сценарий (11/11)
- Юнит-тесты фронтенда: vitest + happy-dom + @vue/test-utils (53/53)

### Changed
- Конфиг: `MAILBRIDGE_LISTEN` (порт HTTP) вместо `MAILBRIDGE_WEBHOOK_LISTEN`; секция «HTTP» в `config.example.env`
- `task_comments.kind`: `system` заменён семантикой `report|reply` (ai_verdict для входящих сохранён)
- E2E: `approve.spec` читает `MAILBRIDGE_AGENT_PASS` из `configs/config.env` (env-переменная опциональна) — тесты проходят в любом shell
- Доки: `docs/api.md` (проекты/модули, approve, WS — по фактическому контракту), `docs/data-model.md` (таблицы + связи v0.22), `docs/ARCHITECTURE.md` (иерархия, API, env), `AGENTS.md` — инварианты v0.22.0

### Fixed
- PrimeVue 5: `Select`/filter слал объект (change-объект) в значения — фильтры «Проект», диалоги «Модуль», `kind` в reply (400 «комментарий не создавался») — `optionValue`/`event.value`
- FilterBar: Promise-баг; после F5 user терялся (в памяти) — admin-only кнопки невидимы → restore `user` с `/auth/me`
- Epics: backfill `description`/`status` при миграции, `epic.description` на создании
- WS-сообщение: «восстанавлизован» → «восстановлен»
- e2e: устойчивость сидов к копированию БД prod→dev

### Removed
- Пакеты `internal/plane/`, `internal/webhook/` и все зависимости от них
- Конфиг: `MAILBRIDGE_PLANE_BASE_URL`, `MAILBRIDGE_PLANE_API_KEY`, `MAILBRIDGE_WEBHOOK_SECRET`; обязательными при запуске остаются только IMAP/SMTP (см. `config.example.env`)
- Метрика `mailbridge_plane_available` и health-check `plane`
- Таблица `reply_log` (на старых БД — идемпотентный `DROP` в миграции), `SaveReplyLog`/`ReplyExists`, `plane_issue_id`
- Отправка e-mail пользователю из mailbridge — **не идёт** (`reply` — только черновик комментария; ADR-0001)

### Security
- Auth: два юзера — `admin` (`MAILBRIDGE_AUTH_USER/PASS`, дефолт `admin/admin` — **обязательно** сменить в production) и `hermes` (агент; `MAILBRIDGE_AGENT_PASS` обязателен для активации)
- Approve — права только у admin (агент — `403`)

## [0.21.2] - 2026-08-26

### Added
- Подкоманда `./build/mailbridge version` — печатает вшитую версию и выходит без загрузки конфига

### Fixed
- Release-воркфлоу: проверка версии бинарника (сравнение с тегом); `make version` без `v`-префикса в `VERSION`

## [0.21.1] - 2026-08-26

### Added
- Workflow `Release` (`.github/workflows/release.yml`): тег `v*` автоматически собирает бинарник и публикует GitHub Release

### Fixed
- gofmt: `ics.go` (`s[i+1]`) и выравнивание комментариев в `ics_real_test.go` — CI Lint зелёный

### Changed
- Правила веток/CI/релизов в CONTRIBUTING.md/AGENTS.md/operations.md: PR-only, green CI обязателен, релизы минорные до v1

## [0.21.0] - 2026-08-26

### Fixed
- Приглашения Exchange-календаря (text/calendar) больше не теряются: событие извлекается
  в секцию `[СОБЫТИЕ]` и попадает в текст письма для AI (issue #1, docs/issues/1.md);
  парсер iCalendar поддерживает unfolding строк (RFC 5545 §3.1) — ранее реальные
  Exchange-писма давали обрезанный ORGANIZER и потерянных ATTENDEE.

## [0.20.2] - 2026-08-26

### Added
- Токены темы (`--mb-*` CSS custom properties) в `global.css`: светлая палитра slate, тёмная — neutral zinc (без сизового оттенка)
- Персистентность темы: `.dark` применяется до роутинга и сохраняется после F5
- Линтер фронтенда: ESLint 9 (flat config, vue3 + prettier) + Prettier; `npm run lint` в CI

### Fixed
- Тёмная тема: устранён «синеватый» фон (slate в dark-режиме вместо нейтрального zinc)
- Мелкие линт-правки в FilterBar/StatusBadge/TaskTable/TaskDetailView

## [0.20.1] - 2026-08-25

### Fixed
- Порядок секций в промпте — история цепочки до вложений
- Лимиты на размер извлекаемого текста из вложений

## [0.20.0] - 2026-08-24

### Added
- Цепочки писем: AI видит контекст thread'а
- Комментарии с kind=ai_verdict, inbox_item_id, verdict_json
- Привязка вложений к комментариям
- API для вложений комментариев

### Fixed
- Очистка JSON от BOM
- Пагинация входящих (perPage)

## [0.19.1] - 2026-08-23

### Added
- Извлечение текста из HTML-писем через goquery

## [0.19.0] - 2026-08-23

### Added
- Лента входящих (inbox_items)
- AI-очередь с retry
- Content-addressable storage для вложений
- Тёмная тема
- Production-сборка с embed фронтенда