# AI Pipeline — Mailbridge

## Обзор

Письма обрабатываются через LLM (Ollama/OpenAI) асинхронно, через очередь. AI анализирует содержимое и возвращает структурированные вердикты.

## Поток обработки

```
Письмо → InboxItem → AIQueue → Worker → LLM → Вердикты → Действия
```

## Компоненты

### AIQueue

- Канал в памяти (`chan int64`)
- При старте загружает необработанные из БД (`ai_processed = 0`)
- Retry с backoff: 1м → 5м → 15м → 1ч
- После 5 попыток: `ai_processed = -1`

### Worker

- Слушает очередь
- Загружает InboxItem + вложения
- Формирует промпт
- Вызывает LLM
- Применяет вердикты
- Обновляет summary цепочки

### Orchestrator

- `ProcessEmail` — сбор контекста, вызов LLM, парсинг
- `ApplyVerdicts` — применение решений к БД
- `UpdateSummary` — обновление резюме цепочки

## Промпт

Формируется в `internal/ai/orchestrator.go`, метод `buildPromptWithThreadContext`.

Секции (порядок важен):

1. ДОСТУПНЫЕ ПРОЕКТЫ
2. ЗАДАЧИ ЦЕПОЧКИ
3. ИСТОРИЯ ЦЕПОЧКИ
4. НОВОЕ ПИСЬМО
5. СОДЕРЖИМОЕ ВЛОЖЕНИЙ (обрезанное)
6. JSON-инструкция

## Вердикты

JSON-формат:

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
        "type": "bug",
        "source_email_id": "...",
        "image_note": "..."
      }
    },
    {
      "action": "update",
      "task_id": 42,
      "updates": {
        "priority": "urgent",
        "add_comment": "...",
        "change_status": "in_progress"
      }
    },
    {
      "action": "completed",
      "task_id": 42,
      "task": {...},
      "comment": "..."
    },
    {
      "action": "none",
      "summary": "..."
    }
  ]
}
```

## Действия по вердиктам

| action | Поведение |
|--------|-----------|
| `new` | Создать задачу |
| `update` | Обновить существующую + комментарий |
| `completed` | Завершить + комментарий |
| `none` | Ничего, только в inbox_items.ai_verdict |

## Календарные приглашения (text/calendar)

Приглашения Exchange приходят как `multipart/alternative` с пустыми `text/plain`/`text/html`
и частью `text/calendar` (iCalendar, RFC 5545), которую enmime размещает в `env.OtherParts`,
а не в вложениях.

Обработка (v0.21.0, issue #1):

1. `extractor`: части `text/calendar` из `env.OtherParts` извлекаются в поле
   `ExtractedEmail.Calendar` (вызывает `preprocessor.ExtractICal`).
2. `preprocessor/ics.go`: парсер с unfolding строк (RFC 5545 §3.1) — без него
   реальные Exchange-строки `ORGANIZER;CN=…`/`ATTENDEE`, сложенные из-за переноса, ломались.
   Вывод — секция `[СОБЫТИЕ] …` (тема, метод REQUEST/CANCEL, время, описание, место,
   организатор, участники).
3. `adapters/email_adapter.go`: если `Calendar` не пусто — секция приклеивается к
   `InboxItem.BodyText` (ПЕРЫЙ блок, после `Текст:`), т.е. AI видит событие в теле письма.
4. Промпт (orchestrator + Modelfile): правило — `[СОБЫТИЕ]` ≠ «пустое письмо»:
   REQUEST → задача-напоминание о встрече; CANCEL → закрыть/отменить задачу.

## Вложения для AI

| Тип | Обработка |
|-----|-----------|
| .txt .csv .md .log | Текст (лимит 10K символов) |
| .png .jpg .jpeg .gif .webp | Base64 → vision |
| .docx | Извлечение текста (лимит 10K) |
| .xlsx | 500 строк, 20 листов, 10K символов |
| .pdf | pdftotext (10K), скан → 5 страниц PNG |
| .rtf | unrtf |
| .ics | Парсинг iCalendar |
| .pptx | Извлечение текста слайдов |

Суммарный лимит текстовых вложений: 50K символов.

## Модель

- Локально: Ollama, модель `qwen3.8-74k:latest` — **та же модель, что и у hermes-агента**: одна в памяти, без выгрузок (решение §7 #16, 2026-08-30)
- Системный промпт: `configs/email-assistant-v2.system.txt` (источник — Modelfile), передаётся в каждом `/api/generate` в поле `system`
- `configs/email-assistant-v2.Modelfile` — оставлен как история SYSTEM-промпта (superseded)
- OpenAI-совместимые: Cloud.ru, OpenAI API (system — `messages[0]`)

## Настройка

```env
MAILBRIDGE_AI_ENABLED=true
MAILBRIDGE_AI_PROVIDER=ollama
MAILBRIDGE_AI_BASE_URL=http://localhost:11434
MAILBRIDGE_AI_MODEL=qwen3.8-74k:latest
MAILBRIDGE_AI_SYSTEM_FILE=configs/email-assistant-v2.system.txt
MAILBRIDGE_AI_TEMPERATURE=0.1   # строгий JSON, минимум креатива (минимум осознанный выбор)
```

## Отладка

- Логи: `[AI] Промпт отправлен...`, `[AI] Ответ LLM...`
- Файлы: `data/ai-debug/*.md` — полный промпт и ответ