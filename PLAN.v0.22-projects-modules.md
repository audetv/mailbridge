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
> **Обновлено 2026-08-30 (решение §4):** `ai_verdict` — только история входящих писем (шаги 4–14);
> mailbridge-AI по комментариям НЕ вызывается (старый шаг 18 снят); `approved` — флаг комментария,
> а не статус задачи.

| kind | Направление | Назначение |
|---|---|---|
| `user_comment` | in/out | обычный комментарий (как сейчас) |
| `ai_verdict` | in | AI-вердикт входящего письма (история) |
| `report` | out | **внутренний отчёт**: что сделано, как, ссылки (commits/патчи) — для своих |
| `reply` | out | **черновик ответа пользователю**: простой язык, без тех. деталей |

- `approved INT NULL` на `task_comments` — модерационный флаг (admin-only approve, §4-решение 3).
- **Почта наружу НЕ отправляется** (отдельная будущая фича). Wiring отправки НЕ делаем (§9-1).

---

## 1. ФАЗА 1 — Проекты (CRUD)

| # | Шаг | Завис. | Статус |
|---|-----|--------|--------|
| 1 | Базовая линия: ветка `feat/...` от main; `make lint`, `make test`, `npm run lint` зелёные; **зафиксировать этот манифест коммитом** | — | `[x]` 64497ab (lint 0 issues, тесты green, npm green) |
| 2 | Миграция `projects` + сида из `configs/rules.yml` (секция `# Проекты`: ТРК, Отель, Лидер Спорт, Театр, Мебельный центр, Складской комплекс, Кафе, Ледовая арена, Корпоративные сайты, + `Входящие`) + из distinct `tasks.project`; **идемпотентно** (`IF NOT EXISTS` + upsert по `name`) | 1 | `[x]` 436a872 — таблица + идемпотентный сид из rules.yml «Проекты» + distinct tasks.project; тест TestMigrate_ProjectsTable / TestMigrate_SeedsDistinctTaskProjects |
| 3 | `store.Store`: модели `Project`, методы `CreateProject / GetProject / GetProjectByName / ListProjects(includeArchived) / UpdateProject / ArchiveProject`; sqlite-реализация + table-driven тесты (sqlite-инстанс) | 2 | `[x]` 436a872 — projects.go + 10 тестов (CRUD, дубли, архив, фильтры); `make test` PASS, `golangci-lint` 0 issues |
| 4 | API `/api/projects` (GET/POST), `/api/projects/{id}` (GET/PUT/DELETE→archive, ре-активация `POST /{id}/unarchive`); валидация имени (≠ пустое, ≤ 128, не дубль); WS-события `project_created/updated/archived` | 3 | `[x]` 436a872+ — web/projects.go + 6 тестов (CRUD/404/409/фильтры/WS), `golangci-lint` 0 issues, `go test ./...` green |
| 5 | UI: `views/ProjectsView.vue` (таблица, карточка создания, inline-редактирование, archiving), вкладки Dashboard; `stores/projects.js`; WS `project_*` → тосты + рефетч | 4 | `[x]` — ProjectsView + stores/projects.js + вкладка «Проекты» в Dashboard; `npm run lint` 0 errors, `npm run build` OK |
| 6 | Ссылка проекта на задачу: `GET /api/projects/{id}/tasks` (фильтр по `t.project = name`); UI: в задачах показываем проект-ссылку | 5 | `✅ dfc8daa` |

Заметки шага:
- (2) Сид обязан быть идемпотентным: второй запуск — 0 новых строк; имена читаем парсером `config.LoadRules` + отдельно секцию `# Проекты` (пока правил `RuleDef{Project}`) → отдельный `config.ProjectsFromRules() []string` (без дублей, trim).
- (3) `store.Store` интерфейс растёт: **нет фейков** в репо — все тесты на sqlite (совпадение с текущей практикой).
- (4) `DELETE /api/projects/{id}` = archive; hard-delete только пустого (`tasks` без ссылок + `epics` пустые) — до Fазы 2.

## 2. ФАЗА 2 — Модули (эпики)

| # | Шаг | Завис. | Статус |
|---|-----|--------|--------|
| 7 | Миграция `epics` + `tasks.epic_id INT NULL`; индексы; идемпотентно | 2 | `[x]` 2218397 — таблица + FK CASCADE + UNIQUE(project_id,number) + tasks.epic_id (SET NULL, ALTER для старых БД) + Task.EpicID; тесты TestMigrate_EpicsTable/Idempotency |
| 8 | `store`: `Epic` + CRUD + `SetTaskEpic` + прогресс (`EpicProgress{open,done,total}`); тесты | 7 | `[x]` — Epic struct + 7 методов + epicStatuses; тесты TestEpicCRUD/Progress/DeleteKeepsTasks |
| 9 | API `/api/projects/{id}/epics` (GET/POST), `/api/epics/{id}` (GET/PUT/DELETE); `POST/DELETE /api/epics/{id}/tasks/{taskId}`; прогресс в `GET /api/epics/{id}`; WS `epic_*` | 8 | `[x]` — EpicHandler (epics.go), роуты main.go (path epic_id), тесты TestEpicsAPI_CRUD/TaskLinking |
| 10 | UI: панель модулей в `ProjectView`/списке задач; фильтр задач по модулю; карточка прогресса (Bar) | 9 | `[ ]` |
| 11 | Задача ↔ модуль: поле `epic_id` в форме создания/редактирования задачи (UI в `TaskDetailView` + API в Фазе 3 `POST /api/tasks`) | 9 | `[ ]` |

Заметки шага:
- (7) `ON DELETE SET NULL` на `tasks.epic_id` — удаление модуля не бьёт по задачам.
- (9) DELETE epic = разрешён, если задач нет (иначе 409 + подсказка переноса/удаления задач).

## 3. ФАЗА 3 — Ручные задачи + AI-проект из БД

| # | Шаг | Завис. | Статус |
|---|-----|--------|--------|
| 12 | `POST /api/tasks` (ручное создание): `project` (из БД, iname), `title`, `body`, `type`, `priority`, `epic_id`; сообщение-ид = `manual-{uuid}` (schema `message_id NOT NULL UNIQUE` не трогаем); валидация project exists и нет archived | 5 | `[x]` 507456d (API) + 8cfa62c (Select-object fix) + 47f1f98 (archived → 400 + тест) |
| 13 | WS `task_created` + UI: форма «Новая задача» (modal) в Dashboard/ProjectView → `stores/tasks.js` | 12 | `[x]` 8c21b1f (WS task_created) + 7dc8b7a/54d5f4b (диалог + epic-поле) — см. решения §7(#11–12) |
| 13.1 | **BUG 2+3 (FilterBar)**: `@change="onProjectChange($event.value)"` (v5 эмитит `{originalEvent, value}`); опции проекта — из `projectsStore` (активные), не hardcoded; восстановление из store корректно | — | `[x]` bdb6299 — фикс багов 2+3 (change-объект) + опции из БД |
| 13.2 | **BUG 1 (таб-роутинг)**: `watch(() => route.query.tab, …)` в DashboardView — URL = source of truth («К задачам» работает, deep-link работает); `goToTasks` → `router.push({query:{tab:'active', project:'<имя>'}})` + сброс `epic_id`; стабильные slugs табов зафиксированы | 13.1 | `[x]` 589c5a4 — watch(route.query.tab) + deep-link ?project= + фикс Promise в FilterBar |
| 13.3 | **E2E-infra (Playwright)**: `@playwright/test` в `frontend/`, `playwright.config.js` (baseURL :5173), fixture-seed в `data/` (script `frontend/scripts/seed-dev.mjs` через API); smoke `auth.spec.js` (login + вкладки видимы) | 13.2 | `[x]` fd437a8 — config + auth.setup.js + seed-dev.mjs (идемпотентный) + auth.spec.js (smoke + deep-link) + npm e2e/e2e:seed; `npx playwright test` 3/3 green |
| 13.4 | **E2E-сценарии**: `projects-to-tasks.spec.js` — «Проекты → К задачам → «Активные» активна + filter project (все строки = проект) + селект-заполнен»; «фильтр проект/модуль в задаче-табах меняет список»; «создание из Проектов → задача с привязкой к проекту» (регрессия снятого бага 4); `npx playwright test` зелёный | 13.3 | `[x]` 967af40 + 9c0d50d — A–D spec: A «К задачам», B фильтр-проект, C фильтр-модуль (epic), D создание из Проектов; PrimeVue-хелперы (native mousedown/click, позиционные селекторы); `npx playwright test` 7/7 green; seed `seed-dev.mjs` расширен до «модуль + задача в модуле» — устойчивость к prod→dev копии БД |
| 13.5 | **AGENTS.md** (текст подготовили; apply — с явного разрешения user'а, т.к. protected): §Commands → «Тестирование» (unit + **e2e обязателен при UI-коммите, касающемся Filter/Tab/Select/Dialog/роутинга**; `e2e:seed` после prod→dev копии) + «Dev-окружение (порты)» (8080 prod — не трогать / 8081 dev / 5173 vite-прокси); §7(#11) закрыт | 13.4 | `[x]` 4452c58 — текст применён в AGENTS.md (user: «apply, я сам скажу») |
| 14 | **AI промты**: `orchestrator.SetProjects` теперь из `store.ListProjects()` (активные); при создании задачи из письма проект обязан ∈ активных проектов (валидация в `orchestrator.createTaskFromVerdict`) | 3, 4 | `[x]` 7fe9175 — SetProjectsProvider + activeProjectNames из БД |
| 15 | Fallback-проект при пустом AI-классификаторе: `config.Plane.DefaultProject` → `ai.DefaultProject = "Входящие"` (var, env `MAILBRIDGE_DEFAULT_PROJECT`) + fallback в `processor.go` | 14 | `[x]` 7fe9175 — resolveVerdictProject + fallback + тесты |
| 16 | Тестируем закрытие цикла: письмо (dev) → AI → задача с проектом из правил rules.yml; ручная задача в любом проекте — обе видимы в UI | 12, 14 | `[x]` `(B: dev env)` 2026-08-30 — решение «одна модель в памяти» + system/temperature из конфига; живой тест: письмо «Сбой бронирования номера» → AI `action:new, project:Отель, priority:high` (inbox_item_id=97, ~20 с) → задача видна в UI; ручной путь — e2e test D (7/7); код ollama/openai +system +temperature, config.go AI-секция, main.go чтение SYSTEM-файла, configs/email-assistant-v2.system.txt, config.env + example.env: qwen3.8-74k:latest + SYSTEM-файл + 0.1; unit system/temperature ×2 провайдера + config ×2; make test/lint/build green; SHA 1d23c71 |

Заметки шага:
- (13.1) Корень бага 2+3 (2026-08-30, баги 2 и 3 от user): PrimeVue 5.0.0 `Select @change` передаёт **объект-событие** `{originalEvent, value}` (`node_modules/primevue/select/index.mjs:941 updateModel: $emit('change',{originalEvent,value})`). `onProjectChange(value)` принимал объект → `setFilter('project', <объект>)` → axios-параметр `[object Object]` → бекенд возвращает НЕотфильтрованный список + placeholder не подставляется; `projectsIdByName(<объект>)` → null → `epicOptions=[]` → «No options available» (баг 3). Реф: соседний epic-`@change="onChange('epic_id', $event.value)"` immune — образец. Тест: `vi.mock` PrimeVue Select (emit с `{originalEvent, value}`) — regression тест в FilterBar.spec.js.
- (13.2) Корень бага 1 (2026-08-30): DashboardView читает `route.query.tab` только в `onMounted` (DashboardView.vue:107–128), watch-а НЕТ — `router.replace({tab})` после mount меняет URL, но v-else-блок задачи не монтируется (при этом store fetch ушёл — в devtools видно отфильтрованный GET, UI «застыл»). Fix: `watch(() => route.query.tab, applyTab)` + `applyTab` из `onMounted`-кода (DRY), `goToTasks` сбрасывает `epic_id` (чужой модуль из другого проекта). Решения §7(#9).
- (13.3–13.4) E2E: Playwright (frontend dev-dep), `baseURL http://localhost:5173`, `storageState` логин из auth.spec.js; реальные проекты из seed-скрипта (не из прод-синхронизации); sценарии 13.4 покрывают баги 1–3 + regression снятого бага 4 (§7(#12)). Acceptance: `npx playwright test` зелёный в headless (CI — Фаза 6). Решения §7(#10).
- (14) Критическое: промт строится из `o.projects` (оркестратор, строка ~299). После шага источник = БД, не Plane.
- (15) Код в tree (uncommitted): `ai.DefaultProject` var-константа + `resolveVerdictProject` (валидация вердикта ∈ активных, иначе fallback + slog warning), `processor.go` fallback `ai.DefaultProject` при пустом `classification.Project`; `main.go: SetProjectsProvider` (активные из SQLite) + env override `MAILBRIDGE_DEFAULT_PROJECT`.
- (16) `(!)` B — требует включённого AI (dev :8081); без dev-окружения отложить как P с причиной и растопить в Фазе 6.
- (16) **Ограничение среды (2026-08-30):** в dev нет второго GPU — `email-assistant-v2:latest` (тег = qwen3.8 + Системный промпт из `configs/email-assistant-v2.Modelfile`, веса те же 16.9 GB) конфликтует в памяти с моделью агента `qwen3.8-74k:latest`: Ollama держит одну модель → письмо выгружает модель агента (17 GB с диска) и обратно → обработка писем встаёт, пока работает агент (user останавливал агента — боль реальная). **Решение (user, 2026-08-30): одна модель в памяти** — mailbridge переходит на `MAILBRIDGE_AI_MODEL=qwen3.8-74k:latest` (тот же тег, что у агента), Системный промпт выносить из Modelfile в запрос Ollama (`"system"` / messages role=system), `temperature` не хардкодить, а взять из конфига (сейчас `0.1` — намеренно минимальное, для строгого JSON и меньше "креативности" — см. `ollama.go:38`, `openai.go:40`). De-gradation всегда откатываема (env). Modelfile остаётся в репозитории как источник системного промпта.
- (16) **Исполнение (2026-08-30):** (a) `AIConfig` + `AIConfig.SystemPrompt string` + `AIConfig.Temperature float64`; (b) `config.go`: `MAILBRIDGE_AI_SYSTEM` (default = чтение `configs/email-assistant-v2.system.txt` при отсутствии переменной — нет, проще: `MAILBRIDGE_AI_SYSTEM_FILE` путь к файлу) + `MAILBRIDGE_AI_TEMPERATURE` (default `0.1`); (c) `OllamaClient.Generate` — если `system != ""` → `"system": <строка>` в body; `temperature` из конфига; (d) `OpenAIClient` — `messages[0] = {role:"system", ...}` (при наличии), `temperature` из конфига; (e) `main.go`: чтение системного промпта из файла (если задан `AI.SystemPromptFile`) → `OllamaClient{system: …}`; (f) тесты: unit (system в body, temperature), integration-тест (реальная Ollama, письмо → AI → задача с проектом из правил; ручная задача видима в UI) — это и есть шаг 16, без e-mail.

## 4. ФАЗА 4 — Отчёт + Ответ (закрытие)

> **Решение 2026-08-30 (user, 4 вопроса закрыты):**
> 1. **Шаг 18 (AI-анализ комментария) — СНЯТ.** mailbridge-AI ничего не анализирует, не
>    рекомендует статуса, не дергается по комментариям (user: «Зачем AI-модель ревьюить мой отчёт?
>    Снять!»). Комментарии создаёт человек/агент, статусы задаёт человек.
> 2. **`approved` — флаг комментария, НЕ статус задачи.** Это внутренний статус модерации
>    конкретного действия — отправки ответа пользователю. Статусы задач остаются как есть
>    (new/backlog/in_progress/completed/closed); «Закрыл задачу» = кнопка, перевод в `completed` (или `closed`).
> 3. **Approve — сейчас admin-only.** Идеал: роль/permission в будущем RBAC; пока —
>    упрощение: admin. Реальной отправки нет (заморожено §9-1), нужна статистика — пока её нет.
> 4. **API не плодим:** расширяем существующий `POST /api/tasks/{id}/reply` полем `kind`
>    (`user_comment`|`report`|`reply`); новый endpoint только `PATCH /api/comments/{id}/approve`
>    (admin-only). Запись проектов/модулей API'ем в Фазе 4 НЕ делаем (только после явного запроса).
>
> Приоритет Фазы 4 (2026-08-30): **17.1 → 18 → 18.2 → 18.3 → 19** — сначала
> агент-юзер + два комментария (report/reply) к готовой задаче, approve последним.

| # | Шаг | Завис. | Статус |
|---|-----|--------|--------|
| 17.1 | Миграция `task_comments`: колонка `approved INT NULL` (дефолт NULL = не утверждён; идемпотентно `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` — SQLite без IF NOT EXISTS на ADD: миграция проверяет `PRAGMA table_info`). Колонки `direction`, `kind`, `inbox_item_id`, `verdict_json` — уже в DDL (шаг 3) — не трогаем. Индекс `idx_task_comments_kind` (если нет) | 3 | `[ ]` |
| 18 | UI: в `TaskDetailView` две секции **Отчёт** и **Ответ пользователю** (вид `CommentList` по `kind`+`direction`); бейдж «Утверждён» на `reply`-комментарии с `approved=1`; стили по kind/author (агент `hermes` визуально выделяется); кнопка «**Закрыл задачу**» = **уже существующие** `WorkflowButtons` (new/in_progress → `closed`, in_progress → `completed` — не пишем новое); кнопка «**Утвердил ответ**» → `PATCH /api/comments/{id}/approve` (только на `kind=reply`, admin-only) | 17.1 | `[ ]` |
| 18.2 | Агент-юзер `hermes`: `auth.go` — 2 юзера: admin (`MAILBRIDGE_AUTH_USER/PASS`, дефолт admin/admin) + **hermes** (логин фиксированный `hermes`, `MAILBRIDGE_AGENT_PASS` обязательно для активации; юзер существует только если пароль задан); login отдаёт username; `extractUserFromToken` уже поддерживает (токен `token-<user>-<date>`); тесты auth (table-driven) | — | `[ ]` |
| 18.3 | API-разрешение: `POST /api/tasks/{id}/reply` принимает `kind: user_comment\|report\|reply` (default `user_comment`; валидация; любой аутентичный юзер может писать; `direction` = `out`) | 18.2 | `[ ]` |
| 19 | `PATCH /api/comments/{id}/approve`: admin-only (не admin — 403), комментарий обязан быть `kind=reply` (иначе 400), `approved 0→1` (повторный approve идемпотентен); WS-событие `comment_approved` → UI бейдж в реальном времени | 18.3 | `[ ]` |
| 20 | **Приёмочный прогон** (кейс Лидер Спорт #66): dev `:8081` — агент (`hermes`) пишет `report` + `reply` к закрытой задаче → admin login в UI → видит оба комментария с бейджами → «Утвердил ответ» → бейдж появляется; WS-события приходят; approve от имени `hermes` → 403 | 19 | `[ ]` (B: dev) |

Заметки шага:
- **(20) `(!)` B** — dev `:8081`.
- **Отправка email (reply пользователю) — ВНЕ Фазы 4** (заморожено §9-1 до решения по отправке: SMTP/лог, авторизация,
  повторность). «Утвердил ответ» = только постановка `approved=1`; в логе/БД фиксируется как статистика.
- **Статус задач не расширяем** `awaiting_approval`/`approved` — такого решения больше нет.
- **Шаг 21 переночерован:** старый «17+18+19b+20+21» → новые `17.1/18/18.2/18.3/19/20`.
  Старый 18 (AI-анализ) удалён; старая 19b (agent) → новая 18.2; старый 19 → новые 18.3+19; старый 20 (UI) → новая 18; старый 21 (приёмка) → новая 20.
- **Plane-ссылка в отчёте** (шаг 7/8) — остаётся; в `reply` Plane-ссылку НЕ показывать (только в `report`).

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
| 29 | `AGENTS.md`: обновить инварианты v0.22 (plane — removed; новые kinds `report/reply`; проекты/модули; auth admin + agent-юзер; `MAILBRIDGE_LISTEN`). **AGENTS.md-разделы «тестирование» + «dev-окружение» уже на шаге 13.5** (раньше Фазы 6, решение §7(#11)). Текст правки подготовим на шаге 28 (apply — вручную, protected). | `[ ]` |
| 30 | Docs: `docs/api.md` (новые маршруты), `docs/data-model.md` (таблицы), `docs/ARCHITECTURE.md` (сборка проектов/модулей, срез Plane), `configs/config.example.env` (нет PLANE, нет WEBHOOK_SECRET, есть LISTEN, AGENT_USER/PASS) | `[ ]` |
| 31 | Обновить root `PLAN.md`: v0.22.0 → архив `archive/PLAN.v0.22-projects-modules.md`; новая активная версия; «Plane: удалён, v0.22.0» | `[ ]` |
| 32 | PR в main, green CI (Lint + Test), merge | `[ ]` |
| 33 | Релиз: тег `v0.22.0`, бинарник GitHub (release.yml — проверить), закрывающий PR | `[ ]` |

---

## 7. Вопросы / Решения пользователя (зафиксированы 2026-08-29)

| # | Вопрос | Решение |
|---|--------|---------|
| 9 | Вкладка-цель «К задачам» + deep-link? (2026-08-30) | **Вкладка «Активные»**, переход — по URL (`?tab=active`). **URL = source of truth для таба**: стабильные идентификаторы вкладок, `watch(route.query.tab)` в DashboardView, e2e ходит по URL (deep-link). Фильтры — в store; project в query — информационный (для e2e/ссылок). |
| 10 | E2E-тесты? (2026-08-30) | **Да, Playwright** (`frontend/`), против живого dev-стека (`make run-dev` :8081 + `npm run dev` :5173), fixture-данные — seed-скрипт в `data/`. Покрывает сценарии «кнопка → смена таба/фильтра → обновление списка», которые моки PrimeVue скрывают (реальные баги 1–3 прошли 39/39 vitest'ов). |
| 11 | AGENTS.md: правила тестирования + dev-окружение — когда? (2026-08-30) | **Раньше Фазы 6** (шаг 13.5, сразу после e2e): правило «UI-коммит несёт e2e-тест при касании Filter/Tab/Select/Dialog» + §3 порты 8080/8081/5173 + seed — вступает в силу сразу, к релизу уже живая практика. Заменяет шаг 29 (теперь: только инварианты v0.22). |
| 12 | Баг 4 «задача без привязки к проекту» (2026-08-30) | **Снят пользователем** — ошибка, всё работает. Регресс-покрытие — e2e-сценарий 13.4 (создание из Проектов). |
| 1 | UI «Эпик» vs «Модуль»? | UI = **Модуль**; API/БД/код = `epics`. Переименование до merge — дёшево. |
| 2 | Reply = отправка письма? | **Нет.** (2026-08-30 уточнено: AI-анализ по комментам снят, шаг 18 удалён). Reply = черновик-ответ пользователю (комментарий `kind=reply`); отправка — отдельно/позже (§9-1). `approved` — флаг комментария (admin-only), статус задачи НЕ расширяем. |
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
| Логин | `MAILBRIDGE_AUTH_USER/PASS` (дефолт admin/admin) | `internal/web/auth.go`; агент — `MAILBRIDGE_AGENT_USER=hermes` + `MAILBRIDGE_AGENT_PASS` (шаг 18.2) |

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
- **AI-очереди по комментариям — НЕ делаем** (2026-08-30, старый шаг 18 снят) — комментарии создаёт человек/агент без AI.
- **Гринн-критерий на каждый коммит:** не «просто компилируется», а `make lint && make test` + `npm run lint` (для фронта). CI обязателен.
- **AGENTS.md protected** — правим по согласованию (шаг 29).
- **Проект ≠ имя в БД.** Инвариант: `tasks.project` ∈ `projects.name` (валидация в API + AI-проверка). После archive — новые задачи в него НЕ создаются.
- **WS-события** — добавляем `project_*`, `epic_*`, `task_created`, `comment_approved` (Фазы 4); проверять на шаге 20 (приёмка Фазы 4).

## 11. Критерий «всё готово для v0.22.0»

- [ ] Все шаги 1–33 `[x]`, §9 пусто (или явно `P` c причиной и планом)
- [ ] `make lint && make test && make build` — зелёные
- [ ] `npm run lint && npm run build` — зелёные
- [ ] grep `plane\|PLANE` в `internal/ cmd/ configs/ docs/ frontend/src/` — 0 совпадений
- [ ] `docs/api.md`, `docs/data-model.md`, `docs/ARCHITECTURE.md` актуальны
- [ ] `configs/config.example.env` — без `MAILBRIDGE_PLANE_*` и `MAILBRIDGE_WEBHOOK_SECRET`
- [ ] PR в main, green CI (Lint + Test), merge
- [ ] Тег `v0.22.0`, бинарник GitHub (release.yml)
