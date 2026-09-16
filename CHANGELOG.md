# Changelog

## [0.24.0] - 2026-09-15

### Fixed
- **persons identity (шаг 6-F, hotfix шага 6):** `person_identities.value` хранил сырой RFC822-хедер `Имя <email` (без закрывающей `>`; prod: 53/53 строки `inbox_items.from_contact` в этой форме) — 32/45 identity были отрывками, 36 персон безымянными. Разбор RFC822-отправителя (`ParseFromHeader`: name/email/display, edge: quoted-имя, bare email, legacy-форма без `>`, multi-address) — email в identity = единственный чистый адрес; `EnsurePersonFromIncoming(name, email)` — авто-заполнение `persons.name` из `from_name` + эвристика «машина» (no-reply/noreply/postfix/mailer/… → `org='машина'`); idempotent self-heal мусорной identity при повторном проходе; legacy-колонки (`from_contact`/`tasks.assignee`/`comments.author`) намеренно не троганы (решение владельца). Тесты: unit extractor+store+adapter+processor, e2e persons.spec.js G/H.
- **Fix (входит в 0.24.0):** `ListTasks` SELECT не перечислял `t.requestor_id`/`t.assignee_id` (фильтр `?requestor_id=`/`?assignee_id=` работал, но строки сериализовались с `requestor_id:null`) — добавлены обе колонки + scan в `sql.NullString` → `*PersonID`.

### Added
- **Ручное подтверждение персоны (v0.24, шаг 6-G):** кнопка «Подтвердить» на строке списка «Персоны» (для не-архивной `confirmed=false`) + в профиле персоны → `PUT /api/persons/{id}` **полным payload** `{name, org, is_internal, archived, confirmed:true}` (UpdatePerson — full-overwrite: частичный payload затёр бы имя/архив — защита на стороне store `confirmPerson`); тег-статус «подтверждена»/«не подтверждена» обновляется из ответа без доп. GET; toast успех/ошибка. Авто-подтверждение и «отмена подтверждения» — не включены (B-2, по данным наблюдения B-1). Бекенд без изменений (эндпоинт уже принимает `confirmed`). Тесты: `tests/stores/persons.spec.js` (4 сценария: полный payload, сохранение полей, локальный апдейт list/selected, сохранность при ошибке), e2e `persons.spec.js` I (клик → тег исчез + API `confirmed=true` + имя/орган не потеряны).
- **Персоны (v0.24, шаг 6, режим A):** справочник `persons` (id UUID, name, org, is_internal, confirmed, archived) + `person_identities` (person_id, kind, value, is_primary); Postgres-совместимая схема (TEXT/BIGINT/TIMESTAMP, без AUTOINCREMENT/FTS5, `ON CONFLICT DO NOTHING`, booleans `CHECK IN (0,1)`). API `/api/persons*`: CRUD, identities (primary email), `/api/persons-suggest` (fast-path + fuzzy) и `/api/persons-merge` `{target_id, source_id}` (source → archived, задачи reassigned на target). `tasks.requestor_id`/`tasks.assignee_id` (UUID) + `task_comments.person_id`; авто-создание персоны при первом контакте (`confirmed=false`, до подтверждения — email, не имя) и авто-назначение assignee при закрытии (`autoAssignLastConfirmer`, manual wins; best-effort). Backfill: `tasks.assignee` → `persons` + `tasks.requestor_id`. UI: вкладка «Персоны» (список + поиск + детали + identities + архив), selects «Заказчик»/«Исполнитель» в карточке задачи, колонки «Заказчик»/«Исполнитель» в таблице задач, фильтры по ролям + deep-link `?requestor_id=<uuid>` из «К задачам». Тесты: unit (fast-path/fuzzy/reject/merge/auto-assign-on-close), web `persons_test.go`, e2e `tests-e2e/persons.spec.js` (создание через UI, primary_email, связывание/отвязание ролей, deep-link + фильтр, merge).

## [0.23.0] - 2026-09-15

### Added
- **История статусов задачи (v0.23, шаг 2):** таблица `task_status_history` (task_id, from_status NULL при создании, to_status, by, created_at); единственный путь смены статуса — `Store.SetTaskStatus` (транзакция: UPDATE + строка истории при реальном переходе); пишут UI/workflow-кнопки (`by` из JWT) и AI-вердикты (`by=ai`); `GET /api/tasks/{id}/history` (хронология, `[]` если нет переходов); старые переходы не восстанавливаются
- **Копирование любого комментария (v0.23, шаг 3):** у каждого комментария в карточке задачи (reply/report/user_comment/ai_verdict) — две кнопки: «Копировать» (чистый текст: MD-символы `**`/`##`/`` ` `` сняты, переносы сохранены) и «MD» (дословно, с символами Markdown); скопированный текст вставляется в почтовый клиент / протокол совещаний (в Outlook MD с цветом/фоном искажается — поэтому по умолчанию чистый текст). Рендерер Markdown в карточке — отдельным шагом (3b)
- **Автолинкинг ссылок в теле комментария (v0.23, шаг 3):** голые URL (`https://…`) в комментариях отображаются как кликабельные ссылки, открывающиеся в **новой вкладке** (`target="_blank" rel="noopener"`); обычный текст вокруг остаётся текстом, переносы сохранены; без `v-html` (XSS-чисто: текст — text-ноды, ссылки — `<a>`). Поведение как в почтовых клиентах/Slack. Вложения не затронуты (у них свои `<a>`).
- **Markdown в карточке комментария (v0.23, шаг 3b):** комментарии с MD-синтаксисом (заголовки `##`, **жирный**, списки, task-lists `- [ ]`, `код`, `---`, цитаты, `[текст](url)`) рендерятся как форматированный HTML — `marked` (GFM, `target=_blank rel=noopener` на всех `<a>`) + `DOMPurify` (allowlist: `<script>`/`on*`/`javascript:` не проходят; только безопасный `<input type=checkbox disabled>`). Обычные (без MD) комментарии — прежний текст-рендер `linkify()` без `v-html` (нет XSS-поверхности); копирование (шаг 3) работает с обоих вариантов. Детектор MD + автолинкинг голых URL покрывают и plain-комментарии.
- **Bulk actions (v0.23, шаг 4):** массовые действия по задачам — checkbox per-row + «выбрать все на странице»; плавающая панель (появляется при ≥1 выбранном): «К статусу…» (5 статусов) и «К проекту…» (существующие проекты, ленивая подгрузка), применение — одним `PATCH /api/tasks {ids, changes:{status?,project?}}` (200 `{count, changed, statuses, errors}`; пустые ids/неизвестный статус → 400, неизвестный проект → 404; дубли ID дедуплятся; смена статуса — через `SetTaskStatus`, т.е. строка в `task_status_history` с `by=юзер`); WS — **одно** событие `batch_update` на всю операцию (не N × `task_updated`); UX: после bulk-действия **остаться на листе** (не прыжок на детальку). Тесты: `internal/web/bulk_tasks_test.go` (10 сценариев), `internal/store/sqlite/bulk_tasks_test.go`, e2e `tests-e2e/bulk-actions.spec.js` (2 задачи → «К В работе» → PATCH 200 `count:2`, чекбоксы сброшены, страница та же, строки истории на обеих), `tests/components/TaskTable.spec.js`.
- **Голые URL → ссылки (шаг 3, часть):** уже описано выше (автолинкинг).

### Changed
- **Формат AI-вердиктов (v0.23, шаг 5):** решение владельца по задаче 378 (правка 14.09 + уточнение 15.09): **AI-вердикт (`ai_verdict`) = ТОЛЬКО текст саммари** — убраны: спойлер «Оригинал письма», quote «Затронуто в письме» (из `verdict_json`), блок вложений письма («Там останется только вердикт», финальный фикс 15.09). **Спойлер «Оригинал письма» остался в саммари-комментариях `user` — и показывает ТОЛЬКО НОВУЮ ЧАСТЬ письма** (`newMessagePartOf`: diff с предыдущим письмом треда + отсечение подписи/«Кому:»/«Просьба при ответе…») + ссылка «Вся переписка — в ленте Inbox»; превью «Оригинальное письмо» в карточке задачи — та же новая часть. Проверено на реальных письмах (задача 378): письмо #474: 34 596 зн → **32 зн**; #473: 33 161 → **95 зн**. Хранение и цепочки (`threads`) не тронуты. Вложения у legacy AI-саммари `author='user'` (комментарий несёт письмо) — показываются как прежде. Тесты: vitest 119/119 (вкл. «AI-вердикт без вложений» и «legacy user с вложениями»), lint clean, Go green, e2e 26/26.

### Removed
- `frontend/src/utils/latest-message.js` (заменён `new-message-part.js` в шаге 5).

### v0.22.x fixes (hotfix-этап, без промежуточной версии)

#### Fixed
- Надёжность WebSocket (симптом: «данные приходят не сразу, пришлaл письмо заново»): реконнект с backoff 3с/10с/30с, форс-реконнект при возврате вкладки из фона и из bfcache; после восстановления соединения экраны (Дашборд, Задача, Лента) перетягивают данные БЕЗ F5; индикатор связи = подтверждение сервера (фрейм `connected` после auth), а не просто open сокета
- Вложения терялись при ручном создании задачи из письма: `POST /api/inbox/{id}/task` (`CreateTaskFromInbox`) и AI update/complete-пути (`completed_by`/`updated_by`) больше не создают задачу без вложений — `CopyInboxAttachmentsToTask` (идемпотентно, `INSERT OR IGNORE`); backfill исторических потерь — `data-fix/v0.22.1-backfill-task-attachments.sql` (запустить на проде при деплое)

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