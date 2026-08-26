# Changelog

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