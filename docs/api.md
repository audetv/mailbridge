# API — Mailbridge

## Аутентификация

JWT-подобный токен. Заголовок: `Authorization: Bearer token-USERNAME-YYYYMMDD`.

### Эндпоинты

- `POST /api/auth/login` — вход: `{username, password}` → `{token, user}`
- `GET /api/auth/me` — текущий пользователь

## Задачи

### `GET /api/tasks`

Query-параметры:
- `page` — номер страницы (default 1)
- `per_page` — размер страницы (default 50, max 200)
- `status` — может повторяться: `?status=new&status=in_progress`
- `project`, `assignee`, `type`, `priority` — фильтры
- `search` — поиск по теме/тексту/email

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
{"project": "Отель", "status": "in_progress", "priority": "high", "type": "bug", "assignee": "Иванов"}
```

### `POST /api/tasks/{id}/reply`

Body: `{"body": "Текст ответа"}`

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
| `connected` | `{message, username}` |
| `task_created` | `{task_id, message, data}` |
| `task_updated` | `{task_id, message, data}` |
| `inbox_created` | `{message, data}` |

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
