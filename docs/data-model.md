# Data Model — Mailbridge

## Обзор

SQLite (WAL-режим), файл `data/mailbridge.db`. Миграции в `internal/store/sqlite/migrations.go` (идемпотентные: `IF NOT EXISTS` + `ALTER TABLE ... ADD COLUMN` с проверкой наличие).

> **v0.22.0:** иерархия Проекты → Модули(`epics`) → Задачи; `reply_log`, `email_mapping`, `plane_*` удалены (ADR-0001).

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

### projects — проекты (v0.22.0)

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| name | TEXT UNIQUE | Название проекта |
| description | TEXT | |
| archived | INTEGER | 0/1 — soft-archive (не удаление) |
| created_at, updated_at | TIMESTAMP | |

### epics — модули проекта (v0.22.0)

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| project_id | INTEGER FK → projects(id) (ON DELETE CASCADE) | |
| name | TEXT | Название модуля |
| description | TEXT | |
| number | INTEGER | Порядковый номер внутри проекта |
| status | TEXT | open, in_progress, done |
| created_at, updated_at | TIMESTAMP | |

Ограничение: UNIQUE(project_id, number).

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
| epic_id | INTEGER FK → epics(id) (ON DELETE SET NULL) | Связь задачи с модулем (nullable) |

Индексы: `message_id`, `status`, `project`, `assignee`, `thread_id`, `source_email_id`, `epic_id`.

### task_comments — комментарии

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| task_id | INTEGER FK | |
| author | TEXT | email, "ai", "user" |
| body | TEXT | Текст |
| direction | TEXT | in (от клиента), out (ответ) |
| kind | TEXT | user_comment, ai_verdict (история входящих), report (внутренний отчёт), reply (черновик ответа) |
| inbox_item_id | INTEGER FK | Связь с входящим |
| verdict_json | TEXT | Полный JSON вердикта (для ai_verdict) |
| approved | INTEGER | 0/NULL = не утверждённый, 1 = утверждённый (admin-only `PATCH /api/comments/{id}/approve`) |
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

### outbox — очередь исходящих

| Поле | Тип | Описание |
|------|-----|----------|
| id | INTEGER PK | |
| payload | TEXT | JSON-нагрузка (отправитель/получатель/сообщение) |
| status | TEXT | pending, sent, failed |
| attempts | INTEGER | Количество попыток |
| last_attempt_at | TIMESTAMP | |
| created_at | TIMESTAMP | |

### task_reads — прочитанные задачи (чтения по юзеру)

| Поле | Тип | Описание |
|------|-----|----------|
| task_id | INTEGER PK (часть) | |
| username | TEXT PK (часть) | Кто прочитал |
| read_at | TIMESTAMP | |

### Связующие таблицы

- **inbox_attachments** — inbox_item_id ↔ attachment_id
- **task_attachments** — task_id ↔ attachment_id
- **comment_attachments** — comment_id ↔ attachment_id
- **task_inbox_items** — task_id ↔ inbox_item_id (relation: created_from, updated_by, completed_by)

## ER-диаграмма (ASCII)

```
projects ── epics ──┐
                    ├── tasks ── task_attachments ── attachments
inbox_items ──┬── inbox_attachments ── attachments
              └── task_inbox_items
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