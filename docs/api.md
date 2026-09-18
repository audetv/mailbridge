# API — Mailbridge

## Аутентификация

JWT-подобный токен. Заголовок: `Authorization: Bearer token-USERNAME-YYYYMMDD`.

### Эндпоинты

- `POST /api/auth/login` — вход: `{username, password}` → `{token, user:{username}}`
- `GET /api/auth/me` — текущий пользователь → `{username}`

Два пользователя: `MAILBRIDGE_AUTH_USER` (admin; дефолт `admin`) и `hermes` (агент, включается при заданном `MAILBRIDGE_AGENT_PASS`). Права approve — только admin.

## Проекты и модули (v0.22)

Проекты → Модули (API: `epics`, `Epic.name`) → Задачи. Задача без модуля допустима. AI-контекст проектов — из SQLite (Plane удалён в v0.22.0, ADR-0001). Поля проекта: `{id, name, description, archived, created_at, updated_at}`. Поля модуля: `{id, project_id, name, description, number, status: open|in_progress|done, created_at, updated_at}`.

### `GET /api/projects`

Список проектов (массив, «сырой» JSON). Query: `archived=true|false` — фильтр; по умолчанию активные; `search` — по названию.

### `POST /api/projects`

Создать: `{name, description?}` → `201` проект. `400` без `name`, `409` при дубле `name`.

### `GET /api/projects/{id}`

Проект (`200`, «сырой» объект). `404`, если нет.

### `PUT /api/projects/{id}`

Обновить: `{name, description?}` → проект. `400` валидация, `409` дубль `name`.

### `DELETE /api/projects/{id}` — архивирование (SOFT-DELETE)

`archived=true`; задачи остаются. `404`, если нет.

### `POST /api/projects/{id}/unarchive` — разархивация (v0.22)

Возвращает проект в активные. `404`, если нет.

### `GET /api/projects/{id}/tasks`

Задачи проекта (фильтр `project = name`; пагинация `?page&per_page`): `{tasks:[...], total, page, per_page}` (те же `tasks`, что в `GET /api/tasks`). `404`, если проект нет.

### `GET /api/projects/{id}/epics`

Модули проекта (массив «сырых» эпиков). `404`, если проект нет.

### `POST /api/projects/{id}/epics`

Создать модуль: `{name, description?, status?}` → `201` модуль. `400` без `name`, `409` дубль `number`.

### `GET /api/epics/{epic_id}`

`{...epic fields..., progress:{total, open, done}}` — прогресс задач модуля.

### `PUT /api/epics/{epic_id}`

Обновить: `{name, description?, status?}` → `200` модуль. `404`, если нет.

### `DELETE /api/epics/{epic_id}`

Удалить (связи задач с модулем рвутся: `epic_id = NULL`). `404`, если нет.

### `POST /api/epics/{epic_id}/tasks/{taskId}`

Привязать задачу к модулю. `404`, если нет.

### `DELETE /api/epics/{epic_id}/tasks/{taskId}`

Отвязать задачу от модуля. `404`, если нет.

## Задачи

### `GET /api/tasks`

Query-параметры:
- `page` — номер страницы (default 1)
- `per_page` — размер страницы (default 50, max 200)
- `status` — может повторяться: `?status=new&status=in_progress`
- `project`, `assignee`, `type`, `priority` — фильтры
- `search` — поиск по теме/тексту/email
- `sort` (v0.25, шаг 7a) — `due` (default) — по сроку возраст, задачи без срока в конце (просроченные выше); при одинаковом сроке (в т.ч. «без срока») — **новые созданные выше** (`created_at DESC`, v0.25.1); `updated` — по дате обновления, новые сверху. Неизвестное значение → `400`.

Ответ:

```json
{
  "tasks": [
    {
      "id": 1,
      "message_id": "...",
      "subject": "...",
      "body_text": "...",
      "from_email": "...",
      "project": "Входящие",
      "type": "bug",
      "priority": "high",
      "status": "new",
      "assignee": "",
      "due_date": "2026-12-01",
      "unread_comments": 2,
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "total": 42,
  "page": 1,
  "per_page": 50
}
```

### `GET /api/tasks/{id}`

Ответ:

```json
{
  "task": {...},
  "comments": [
    {
      "id": 1,
      "task_id": 1,
      "author": "user@example.com",
      "body": "...",
      "direction": "in",
      "kind": "user_comment",
      "approved": null,
      "inbox_item_id": 5,
      "verdict_json": "",
      "created_at": "..."
    }
  ],
  "attachments": [...]
}
```

### `PATCH /api/tasks/{id}`

Body — допустимые поля:

```json
{"project": "Отель", "status": "in_progress", "priority": "high", "type": "bug", "assignee": "Иванов", "due_date": "2026-12-31"}
```

`due_date` (v0.25, шаг 7a) — срок задачи, дата `"YYYY-MM-DD"` (без времени, канон: онтология v0.5.2 §7.5). Установить — строка; снять — `null` или пустая строка. Не-канонический формат (`DD.MM.YYYY`, `2026-13-01`, `2026-02-31`, время `...T00:00:00Z`) → `400`. Ручная установка пишет `due_source = 'manual'`.

`status` (статус) — единственный путь смены: атомарно обновляет `tasks.status` и при реальном переходе (`from != to`) пишет строку в `task_status_history` с `by` = имя текущего пользователя (из JWT). → обновлённая `task`.

### `GET /api/tasks/{id}/history` (v0.23)

Хронология переходов статуса задачи: `[]` если переходов ещё не было.

```json
{"history": [
  {"id":1,"task_id":370,"from_status":null,"to_status":"in_progress","by":"admin","created_at":"2026-09-13T17:50:00.000Z"},
  {"id":2,"task_id":370,"from_status":"in_progress","to_status":"completed","by":"ai","created_at":"2026-09-13T18:00:12.000Z"}
]}
```

`by` — имя пользователя из JWT либо строка `ai` (AI-вердикт, v0.23). Идентичный повторный `PATCH` (тот же `status`) строку не пишет.

### `PATCH /api/tasks` (v0.23, bulk)

Массовое действие над списком задач одним запросом. Body:

```json
{"ids": [1, 2, 3], "changes": {"status": "backlog", "project": "Отель"}}
```

- `ids` — `[int]`, минимум 1 (`400` иначе); дубли игнорируются.
- `changes` — `status?` (backlog/new/in_progress/completed/closed) и/или `project?` (имя существующего проекта); пустое/отсутствующее → `400`.
- `status` идёт через тот же `SetTaskStatus`, что и одиночный `PATCH /api/tasks/{id}`: при реальном переходе — строка в `task_status_history` (`by` — юзер из JWT).
- Ответ: `200 {"count": N, "changed": N, ...}` (число затронутых задач). **Все** переданные ID не найдены → `404`; частично — существующие обновляются, отсутствующие не считаются в `count`.
- WS: **одно** событие `batch_update` (`{event, count, source}`) на всю операцию — не N событий по одной.

### `POST /api/tasks` (ручное создание)

Body: `{"title": "Тема (обяз., 1..500)", "project": "Имя проекта (обяз., существующий)", "description": "текст (опц.)", "epic_id": 3, "due_date": "2026-12-31"}`.
`due_date` (v0.25, шаг 7a) — `"YYYY-MM-DD"`, не-канон → `400`, задача не создаётся. Статус всегда `new` → `201 {task}`; WS `task_created`.

### `POST /api/tasks/{id}/reply`

Body: `{"body": "Текст ответа", "kind": "user_comment"}` — `kind` опционален: `user_comment` (по умолчанию) | `report` (внутренний отчёт) | `reply` (черновик ответа пользователю). Черновик НЕ отправляется — только в историю задачи (Срез Plane, v0.22.0). → `{"comment": {...}}`

### `PATCH /api/comments/{id}/approve` (v0.22)

Утверждение черновика ответа. Только `kind=reply` (иначе `400`), только **admin** (иначе `403`). Idempotent: повторный approve не ошибка. → обновлённый `{comment}` + WS `comment_approved`

### `POST /api/tasks/{id}/mark-read`

Отмечает задачу прочитанной для текущего пользователя.

### `GET /api/tasks/{id}/inbox`

Связанные входящие.

### `GET /api/tasks/{id}/attachments`

Вложения задачи.

### `DELETE /api/tasks/{id}/attachments/{attId}`

Открепить вложение.

### `GET /api/tasks/{id}/comments/{commentId}/attachments`

Вложения комментария.

## Лента входящих

### `GET /api/inbox`

Query: `status` (unread/read/archived), `page`, `perPage` (или `per_page`).

Ответ:

```json
{
  "items": [...],
  "total": 42,
  "page": 1,
  "per_page": 20
}
```

### `GET /api/inbox/{id}`

Элемент ленты.

### `POST /api/inbox/{id}/read`

Отметить прочитанным.

### `POST /api/inbox/{id}/unread`

Вернуть в непрочитанные.

### `POST /api/inbox/{id}/archive`

В архив.

### `POST /api/inbox/{id}/task`

Создать задачу из ленты.

### `GET /api/inbox/{id}/attachments`

Вложения входящего.

### `GET /api/inbox/{id}/tasks`

Задачи, связанные с входящим.

## Вложения

### `GET /api/attachments/{hash_path}/{filename}`

Скачать/открыть файл. `hash_path` = `{hash[0:2]}/{hash[2:4]}/{hash}`. `filename` опционален — для браузера.

## WebSocket

### `GET /api/ws`

События от сервера:

| Тип | Payload |
|-----|---------|
| `connected` | `{type, message, username}` |
| `task_created` | `{type, taskId, message, data:{task}}` |
| `task_updated` | `{type, taskId, message, data:{task}}` |
| `inbox_created` | `{type, message, data:{inbox_item}}` |
| `comment_approved` | `{type, taskId, message, data:{comment}}` |
| `project_created` / `project_updated` / `project_archived` / `project_unarchived` | `{type, message, data:{project}}` |
| `epic_created` / `epic_updated` / `epic_deleted` | `{type, message, data:{epic}}` |
| `epic_task_linked` / `epic_task_unlinked` | `{type, message}` |

Входящие сообщения от клиента:
- `{"type": "ping"}` → ответ `{"type": "pong"}`
- `{"type": "mark_read", "task_id": 42}` (зарезервировано)

## Коды ошибок

| Код | Описание |
|-----|----------|
| 400 | Невалидный запрос |
| 401 | Не авторизован |
| 404 | Не найдено |
| 500 | Внутренняя ошибка |
