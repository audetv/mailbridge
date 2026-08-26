# Баг #1: календарные приглашения Exchange (text/calendar) → пустое письмо

> Ссылка: [баг-репорт](1.md). Дата: 2026-08-26. Статус: план готов, не реализован.

## Корневая причина (верифицировано по данным)

1. **Структура письма** (проверено по `data/emails/1.eml`, 9 КБ):
   `multipart/alternative` → `text/plain` (пусто), `text/html` (только `<p>&nbsp;</p>`),
   `text/calendar; method=REQUEST` (iCalendar, 1 575 байт, внутри **envelope, не вложение**).
2. **enmime v1.3.0**: часть без `Content-Disposition` и не `text/*` — попадает в
   `env.OtherParts` (проверено кодом библиотеки + живым парсингом письма):
   `OtherParts[0]: ct=text/calendar, size=1575, содержит BEGIN:VEVENT, SUMMARY`.
   В `env.Attachments` она **не попадает** — поэтому ни экстрактор, ни AI её не видят.
3. **Лог AI** (`data/ai-debug/...1787728313.md`): в секции «НОВОЕ ПИСЬМО» тело пустое →
   модель корректно ответила «нет текста» → задача не создана. Модель не виновата.
4. **Содержимое события** (извлечено): `SUMMARY: онлайн продажи`,
   `DTSTART 2026-08-26 14:30 (Russian Standard Time)` → `DTEND 15:00`,
   `LOCATION: у Алексея`, `ORGANIZER: Бухтина А.А.`, `ATTENDEE: Кузьмина П.В., Гусев…`.
5. **Доп. дефект существующего ICal-парсера** (`internal/preprocessor/ics.go`, работает
   только по **пути к файлу**): строки iCal складываются по RFC 5545 (CRLF + пробел),
   а парсер читает построчно без unfolding → `ORGANIZER` обрезается (`value="M"`),
   `ATTENDEE` теряются. На простых «чистых» .ics (как fixture в тестах) это не видно.

### Место, где нужно менять (проверено по коду)

- `internal/extractor/extractor.go` — `Extract()` не смотрит `env.OtherParts` вообще.
- `internal/adapters/email_adapter.go:57-64` — **перезаписывает** `BodyText` текстом из
  HTML, если HTML не пуст. Значит прилеплять календарный текст нужно не в `BodyText`
  экстрактора, а **отдельным полем** и присоединять в адаптере после всех перезаписей.
- `internal/ai/orchestrator.go` — промпт строится из `email.BodyText`; вложения проходят
  через `preprocessor.ProcessAttachment(path, filename)` (только по пути).
- `data/` в `.gitignore` — fixture'ы из `data/emails/` в тестах **нельзя** использовать
  напрямую, нужны копии в `testdata/` (заверен, что `data/` игнорируется, строка 24).

## План исправления

### Целевое поведение

Письмо-приглашение (METHOD:REQUEST/CANCEL) попадает в inbox с `BodyText`, содержащим
секцию `[Событие]` (SUMMARY, время, место, организатор, участники), и AI вердикт
формируется по содержимому события, а не «пустое письмо».

### Шаг 1 — ICal из байтов + unfolding (preprocessor)

- `internal/preprocessor/ics.go`:
  - вынести ядро в `extractICalFromData(data []byte) string` (сейчас только `os.ReadFile`);
    `extractICalText(path)` становится обёрткой;
  - добавить exported `ExtractICal(data []byte) string`;
  - **unfolding по RFC 5545 §3.1**: объединять строки, следующие за строкой
    «CRLF + пробел/таб», до разбора;
  - добавить `METHOD:***` и `DESCRIPTION` события в `formatICalEvent`
    (CANCEL должен быть видим — иначе AI не поймёт, что встреча отменена);
- тест: `ics_test.go` → новый кейс `TestExtractICal_ExchangeFolded`:
 ORGANIZER;CN=… со сгибом строки, TZID-параметры, METHOD — ожидается полный ORGANIZER.

### Шаг 2 — извлечь календарные части в extractor

- `internal/extractor/extractor.go`:
  - новое поле `Calendar string` в `ExtractedEmail` (отдельно от `BodyText` — чтобы
    адаптер не потерял при HTML-перезаписи);
  - в `Extract()`: перебор `env.OtherParts`, фильтр
    `strings.HasPrefix(p.ContentType, "text/calendar")`,
    `extractor` → вызов `preprocessor.ExtractICal(p.Content)` (импорт без циклов —
    проверено: preprocessor не зависит от extractor);
  - несколько `text/calendar` частей склеить последовательно.
- **не** сохраняем calendar-часть как вложение (enmime её в Attachments не кладёт и
  так); опционально (out of scope, но пометить) — дублировать `.ics` вложения для UI.

### Шаг 3 — присоединить в адаптере

- `internal/adapters/email_adapter.go`: после блока HTML→text:
  ```go
  if email.Calendar != "" {
      bodyText = strings.TrimSpace(bodyText)
      if bodyText != "" { bodyText += "\n\n" }
      bodyText += email.Calendar
  }
  ```
  — т.е. календарная секции добавляется **всегда**, независимо от HTML-ветки.

### Шаг 4 — промпт AI (минимально)

- `internal/ai/orchestrator.go` + `configs/email-assistant-v2.Modelfile`: в раздел с
  правилами одна строка: «Текст может содержать секцию `[Событие]` — это приглашение
  в календарь; создай задачу-напоминание о встрече (или отмену), опираясь на её данные.»
  (Модель уже умеет решать — ей просто нужен контекст; правило лишь фиксирует поведение.)

### Шаг 5 — тесты

1. **extractor**: `testdata/calendar_invite.eml` (копия `data/emails/1.eml`,
   заверенность в git) → `Extract` → `Calendar` содержит «онлайн продажи», `14:30`,
   «у Алексея»; `BodyText` остаётся пустым (как есть).
2. **adapter**: то же письмо → `Parse().InboxItem.BodyText` содержит `[СОБЫТИЕ] онлайн продажи`.
3. **регрессия**: существующие `email_adapter_test.go` и `extractor_test.go` без изменений.
4. `make lint && make test` + CI (npm-lint не затронут).

### Шаг 6 — ручная приёмка + документация

1. Локально: запустить сервис, повторную отправку `1.eml` и `2.eml` (dedup по
   SourceID/MessageID: если записи 61/62 уже лежат с пустым телом — обновить их
   `body_text` через `UPDATE inbox_items SET body_text=? WHERE id IN (61,62)` и
   повторно прогнать через AI-очередь, либо удалить и пересоздать — решим по поведению
   store на месте).
2. Проверить: в inbox текст события виден, в `data/ai-debug/` промпт с телом, вердикт
   адекватный («new»-задача-встреча или «information»).
3. `docs/ai-pipeline.md`: блок «Календарные приглашения (text/calendar)»;
   `CHANGELOG.md`: новый unreleased/0.21.0 → Fixed «Приглашения Exchange-календаря
   больше не теряются: события извлекаются в текст письма».
4. Закрыть `docs/issues/1.md` (статус: исправлено в v0.21.0).

## Риски и границы

- **Unfolding** — без него ORGANIZER/ATTENDEE ломаются на реальных Exchange-письмах
  (продемонстрировано на 1.eml); обязательный под-пункт шага 1.
- **VTIMEZONE-шум** — парсер и сейчас берёт только VEVENT-блок, остаётся корректным.
- **Несколько событий** в одном календаре — обрабатывается (цикл по VEVENT-блокам).
- **METHOD:CANCEL** — в текст добавляется, поведение AI по правилам промпта.
- **Не затрагивает**: SMTP-out, Plane-legacy, адаптеры telegram/web_form (этап B плана).

## Оценка объёма

~150–200 строк (включая тесты). Один коммит «fix: Exchange-календарные приглашения:
извлечение text/calendar в текст письма (issue #1)» + CHANGELOG.
