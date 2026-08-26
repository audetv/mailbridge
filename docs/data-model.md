# Data Model — Mailbridge

## Обзор

SQLite (WAL-режим), файл `data/mailbridge.db`. Миграции в `internal/store/sqlite/sqlite.go`.

## Таблицы

### inbox_items — лента входящих

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| source | TEXT | Тип источника: email, telegram, web_form |
| source_id | TEXT | Уникальный ID внутри источника (Message-ID для email) |
| thread_id | TEXT | Цепочка (первый Message-ID или References[0]) |
| from_contact | TEXT | Email/имя отправителя |
| from_name | TEXT | Имя отправителя |
| subject | TEXT | Тема |
| body_text | TEXT | Очищенный текст (из HTML если есть) |
| body_html | TEXT | Оригинальный HTML |
| meta | TEXT (JSON) | Source-специфичные данные |
| received_at | TIMESTAMP | |
| ai_processed | INTEGER | 0=ожидает, 1=готово, -1=ошибка |
| ai_attempts | INTEGER | Количество попыток |
| ai_verdict | TEXT (JSON) | Массив вердиктов LLM |
| ai_summary | TEXT | Краткое резюме |
| status | TEXT | unread, read, archived |

Индексы: `thread_id`, `status`, `ai_processed`, UNIQUE(source, source_id).

### threads — цепочки

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| thread_id | TEXT UNIQUE | |
| source | TEXT | |
| subject | TEXT | |
| participants | TEXT (JSON) | |
| summary | TEXT | Резюме цепочки (обновляется LLM) |
| last_item_at | TIMESTAMP | |
| created_at, updated_at | TIMESTAMP | |

### tasks — задачи

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| message_id | TEXT UNIQUE | Уникальный ID (SourceID + -task-N) |
| subject | TEXT | |
| body_text | TEXT | Описание от AI |
| body_html | TEXT | |
| from_email, from_name | TEXT | |
| project | TEXT | |
| type | TEXT | bug, feature, support, access, seo, content |
| priority | TEXT | urgent, high, medium, low |
| status | TEXT | new, backlog, in_progress, completed, closed |
| assignee | TEXT | |
| thread_id | TEXT | Связь с цепочкой |
| source_email_id | TEXT | ID письма-источника |
| ai_verdict | TEXT (JSON) | Последний вердикт |

Индексы: `message_id`, `status`, `project`, `assignee`, `thread_id`, `source_email_id`.

### task_comments — комментарии

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| task_id | INTEGER FK | |
| author | TEXT | email, "ai", "user" |
| body | TEXT | Текст |
| direction | TEXT | in (от клиента), out (ответ) |
| kind | TEXT | user_comment, ai_verdict, system |
| inbox_item_id | INTEGER FK | Связь с входящим |
| verdict_json | TEXT | Полный JSON вердикта (для ai_verdict) |
| created_at | TIMESTAMP | |

### attachments — файлы (CAS)

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| hash | TEXT UNIQUE | SHA-256 содержимого |
| filename | TEXT | Оригинальное имя |
| content_type | TEXT | |
| size | INTEGER | |
| storage_path | TEXT | {hash[0:2]}/{hash[2:4]}/{hash} |
| created_at | TIMESTAMP | |

### Связующие таблицы

- **inbox_attachments** — inbox_item_id ↔ attachment_id
- **task_attachments** — task_id ↔ attachment_id
- **comment_attachments** — comment_id ↔ attachment_id
- **task_inbox_items** — task_id ↔ inbox_item_id (relation: created_from, updated_by, completed_by)

## ER-диаграмма (ASCII)

```
inbox_items ──┬── inbox_attachments ── attachments
              └── task_inbox_items ── tasks ── task_attachments ── attachments
                                          │
                                          └── task_comments ── comment_attachments ── attachments
```

## JSON-форматы

### inbox_items.meta

```json
{
  "message_id": "...",
  "to": "...",
  "cc": "",
  "references": ["...", "..."],
  "in_reply_to": "..."
}
```

### inbox_items.ai_verdict

```json
[
  {
    "action": "new|update|completed|none",
    "task_id": 42,
    "task": {
      "title": "...",
      "description": "...",
      "priority": "high",
      "project": "Входящие",
      "type": "bug",
      "source_email_id": "...",
      "image_note": "..."
    },
    "comment": "...",
    "summary": "..."
  }
]
```