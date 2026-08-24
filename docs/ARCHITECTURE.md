# Архитектура Mailbridge

## Обзор системы

Mailbridge — сервис для трансформации входящих email в структурированные задачи с помощью AI-анализа. Состоит из Go-бекенда, Vue 3 фронтенда и опционально локальной LLM (Ollama).

## Компоненты

```
┌─────────────────────────────────────────────────────┐
│                  Frontend (Vue 3)                   │
│  Dashboard, Inbox, Tasks, WebSocket, Dark Theme     │
└──────────────────────┬──────────────────────────────┘
                       │ HTTP / WebSocket
┌──────────────────────▼──────────────────────────────┐
│                  Go Backend                          │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────┐ │
│  │  IMAP Client │→ │  Processor   │→ │  AI Queue  │ │
│  └─────────────┘  └──────┬───────┘  └─────┬──────┘ │
│                          │                 │        │
│                    ┌─────▼─────┐    ┌──────▼─────┐  │
│                    │ SQLite DB │    │  Ollama    │  │
│                    │           │    │  (LLM)     │  │
│                    └───────────┘    └────────────┘  │
└─────────────────────────────────────────────────────┘
```

## Схема данных

### inbox_items — лента входящих

Универсальное хранилище входящих сообщений. Каждый источник (email, telegram, webhook) создаёт записи здесь.

- `source` — тип источника (email, telegram, web_form)
- `source_id` — уникальный ID внутри источника (Message-ID для email)
- `meta` — JSON с source-специфичными данными
- `ai_processed` — 0=ожидает, 1=обработано, -1=ошибка
- `ai_verdict` — JSON с вердиктами LLM

### threads — цепочки

Группировка входящих. Для email — это цепочка писем (References).

- `thread_id` — уникальный идентификатор цепочки
- `summary` — резюме цепочки, обновляется LLM

### tasks — задачи

Создаются из входящих после AI-анализа или вручную.

- `status`: new, backlog, in_progress, completed, closed
- `thread_id` — связь с цепочкой
- `source_email_id` — ID письма-источника

### attachments — вложения

Универсальная таблица файлов с дедупликацией.

- `hash` — SHA-256 содержимого, уникальный
- `storage_path` — путь в CAS: `{hash[0:2]}/{hash[2:4]}/{hash}`
- `filename` — оригинальное имя для отображения

Связи:
- `inbox_attachments` — входящее ↔ вложение
- `task_attachments` — задача ↔ вложение
- `comment_attachments` — комментарий ↔ вложение

### task_inbox_items — связь задач с лентой

Многие-ко-многим. Одна задача может ссылаться на несколько входящих, одно входящее может породить несколько задач.

## Поток обработки письма

```
Письмо приходит
  │
  ▼
IMAP Client (MailReader)
  │
  ▼
Extractor (MIME-парсинг, сохранение вложений)
  │
  ▼
EmailAdapter (создание InboxItem, hash вложений)
  │
  ▼
MessageProcessor (сохранение в БД)
  │
  ├── AI включён?
  │   ├── Да → AIQueue.Enqueue()
  │   │         │
  │   │         ▼
  │   │       AI Worker
  │   │         │
  │   │         ├── Загрузка InboxItem + вложений
  │   │         ├── Формирование промпта
  │   │         ├── Вызов LLM (Ollama/OpenAI)
  │   │         ├── Парсинг вердиктов
  │   │         ├── Применение: создание/обновление задач
  │   │         └── Обновление summary цепочки
  │   └── Нет → Rules-based классификация
  │
  ▼
WebSocket-событие
  │
  ▼
Frontend обновляется
```

## AI-интеграция

### Очередь

- Канал в памяти (`chan int64`)
- Загрузка необработанных при старте из БД
- Retry с backoff: 1м → 5м → 15м → 1ч
- После 5 попыток — `ai_processed = -1`

### Вердикты LLM

```json
{
  "verdicts": [
    {
      "action": "new",
      "task": {
        "title": "...",
        "description": "...",
        "priority": "high",
        "project": "Входящие",
        "type": "bug"
      }
    },
    {
      "action": "update",
      "task_id": 42,
      "updates": {"priority": "urgent", "add_comment": "..."}
    },
    {
      "action": "completed",
      "task_id": null,
      "task": {...},
      "comment": "Работа закрыта"
    },
    {
      "action": "none",
      "summary": "Информационное сообщение"
    }
  ]
}
```

### Действия по вердиктам

| action | Поведение |
|--------|-----------|
| `new` | Создать задачу |
| `update` | Обновить существующую задачу |
| `completed` | Завершить или создать завершённую задачу |
| `none` | Ничего — письмо остаётся в ленте |

## Вложения

### Content-Addressable Storage

- Файл хранится по SHA-256: `data/attachments/{hash[0:2]}/{hash[2:4]}/{hash}`
- Оригинальное имя в БД `attachments.filename`
- Дедупликация: одинаковые файлы хранятся один раз

### Поддерживаемые типы

| Тип | Обработка |
|-----|-----------|
| .txt .csv .md .log | Текст |
| .png .jpg .jpeg .gif .webp | Base64 → vision |
| .docx | Извлечение текста |
| .xlsx | Извлечение текста из листов |
| .pdf | pdftotext, fallback на изображения |
| .rtf | unrtf или простой парсер |
| .ics | Парсинг iCalendar |
| .pptx | Извлечение текста из слайдов |

## API

### Аутентификация

```
POST /api/auth/login  {username, password} → {token}
GET  /api/auth/me     → {username}
```

### Задачи

```
GET    /api/tasks                    — список с фильтрами
GET    /api/tasks/{id}               — задача + комментарии + вложения
PATCH  /api/tasks/{id}               — обновить
POST   /api/tasks/{id}/reply         — ответ клиенту
POST   /api/tasks/{id}/mark-read     — отметить прочитанной
GET    /api/tasks/{id}/inbox         — входящие, связанные с задачей
GET    /api/tasks/{id}/attachments   — вложения задачи
DELETE /api/tasks/{id}/attachments/{attId} — открепить
```

### Лента входящих

```
GET  /api/inbox                       — список с фильтрами
GET  /api/inbox/{id}                  — элемент ленты
POST /api/inbox/{id}/read             — отметить прочитанным
POST /api/inbox/{id}/unread           — вернуть в непрочитанные
POST /api/inbox/{id}/archive          — в архив
POST /api/inbox/{id}/task             — создать задачу из ленты
GET  /api/inbox/{id}/attachments      — вложения
GET  /api/inbox/{id}/tasks            — связанные задачи
```

### Вложения

```
GET /api/attachments/{hash_path}/{filename} — скачать/открыть
```

## Конфигурация

### config.env

```env
# IMAP
MAILBRIDGE_IMAP_SERVER=...
MAILBRIDGE_IMAP_PORT=993
MAILBRIDGE_IMAP_USER=...
MAILBRIDGE_IMAP_PASS=...
MAILBRIDGE_IMAP_TLS=true

# SMTP
MAILBRIDGE_SMTP_SERVER=...
MAILBRIDGE_SMTP_PORT=587
MAILBRIDGE_SMTP_FROM=...

# AI
MAILBRIDGE_AI_ENABLED=true
MAILBRIDGE_AI_PROVIDER=ollama
MAILBRIDGE_AI_BASE_URL=http://localhost:11434
MAILBRIDGE_AI_MODEL=email-assistant-v2

# Auth
MAILBRIDGE_AUTH_USER=admin
MAILBRIDGE_AUTH_PASS=...
```

## Деплой

### Systemd (user-сервис)

```ini
[Unit]
Description=Mailbridge
After=network.target

[Service]
Type=simple
WorkingDirectory=%h/apps/mailbridge
ExecStart=%h/apps/mailbridge/bin/mailbridge
EnvironmentFile=%h/apps/mailbridge/configs/config.env
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
```

```bash
systemctl --user daemon-reload
systemctl --user enable mailbridge
systemctl --user start mailbridge
loginctl enable-linger $USER
```

### Nginx (опционально)

```nginx
server {
    listen 80;
    server_name mail.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }
}
```

## Решения и обоснования

| Решение | Почему |
|---------|--------|
| SQLite | Один пользователь/маленькая команда, простота |
| CAS для вложений | Дедупликация, экономия места |
| Очередь AI | Не блокирует IMAP при медленной LLM |
| WebSocket | Мгновенное обновление UI |
| Vue 3 + PrimeVue | Компоненты из коробки, тёмная тема |
| goquery для HTML | Надёжное извлечение текста из писем |
