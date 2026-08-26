# AGENTS.md

## Project

Mailbridge — сервис управления входящими обращениями из почты. Принимает письма через IMAP, анализирует через AI (Ollama/OpenAI), создаёт задачи, хранит историю цепочек. Стек: Go 1.26 (бекенд), Vue 3 + PrimeVue (фронтенд), SQLite (WAL). Версию смотрим по `git tag`. Подробнее: [README.md](README.md).

## Commands

### Backend (Go)

- `make build` — полная сборка: фронтенд + копирование статики + Go-бинарник
- `make test` — все тесты
- `make test-cover` — тесты с покрытием
- `make lint` — golangci-lint
- `make vet` — go vet
- `make fmt` — gofmt
- `make run` — запуск (production, порт из config.env)
- `make run-dev` — запуск в dev-режиме (порт берётся из `MAILBRIDGE_WEBHOOK_LISTEN`; vite-прокси при dev ждёт 8081 — т.е. в dev-конфиге `LISTEN=:8081`)
- `make clean` — очистка артефактов
- `make tidy` — go mod tidy

### Frontend (Vue)

- `cd frontend && npm run dev` — dev-сервер (Vite, порт 5173)
- `cd frontend && npm run build` — production-сборка
- `cd frontend && npm run preview` — предпросмотр собранного
- `cd frontend && npm run lint` / `npm run format` — ESLint 9 + Prettier (CI гоняет lint перед сборкой)

## Repo Map

```
cmd/mailbridge/          — точка входа, embed статики
internal/
  adapters/              — адаптеры источников (email)
  ai/                    — AI-клиент, оркестратор, очередь, воркер
  classifier/            — rules-based классификация + NLP
  config/                — загрузка конфигурации
  extractor/             — MIME-парсинг, очистка, вложения
  health/                — health-проверки
  logging/               — slog-обёртка
  mailbox/               — IMAP-клиент
  metrics/               — Prometheus-метрики
  parser/                — извлечение полей из текста
  plane/                 — клиент Plane API (отключён в маппинге задач, но env Plane **required** для валидации конфига — см. ADR-0001; удаление требует правки валидации в config.go)
  preprocessor/          — обработка вложений для AI
  processor/             — оркестрация обработки писем
  sender/                — SMTP-отправка
  app/                   — композиция кор/сервисов приложения
  attachments/           — работа с вложениями (CAS, SHA-256)
  integration/           — интеграционные тесты
  version/               — версия сборки (ldflags)
  store/                 — интерфейс хранилища
  store/sqlite/          — SQLite-реализация
  web/                   — HTTP-обработчики, API
  webhook/               — приём webhook'ов
  worker/                — воркеры (inbound, outbound)
frontend/                — Vue 3 SPA
  src/views/             — страницы
  src/components/        — компоненты
  src/stores/            — Pinia-сторы
  src/api/               — axios-клиент
configs/                 — конфигурация (env, rules.yml, Modelfile)
docs/                    — документация
data/                    — БД и вложения (НЕ коммитить)
```

## Go Conventions

- Ошибки: `fmt.Errorf("context: %w", err)` — всегда wrap, не терять контекст
- Логирование: `slog` (не `fmt.Println`, не `log.Printf` в бекенде)
- Тесты: table-driven, `t.Helper()` для setup-функций
- Пакеты: имя пакета соответствует директории
- Перед коммитом: `make lint && make test` обязательны

## Frontend Conventions

- Vue 3 Composition API (`<script setup>`)
- Компоненты: PrimeVue (DataTable, Card, Button, Select, Tag, Badge)
- Сторы: Pinia (`src/stores/`)
- HTTP: `src/api/client.js` (axios с JWT-интерсептором)
- WebSocket: `src/stores/websocket.js` (EventSource-подобный паттерн)
- Стили: scoped, CSS-переменные PrimeVue (`--p-surface-*`)
- После правок фронта: `make build` (статику копирует в `cmd/mailbridge/static/`)
- Линтер обязателен: `npm run lint` без ошибок — и в CI перед сборкой (npm run lint → eslint .)

## Config & Secrets

- Конфигурация: `configs/config.env` (НЕ коммитить, в .gitignore)
- Образец: `configs/config.example.env` (обновлять при добавлении параметров)
- Секреты: только через env-переменные `MAILBRIDGE_*` (словарь — `configs/config.example.env`)
- `data/` — БД и вложения, НЕ коммитить
- Обязательные для запуска (валидация в `internal/config/config.go`): `MAILBRIDGE_IMAP_SERVER/USER/PASS`, `MAILBRIDGE_SMTP_SERVER/FROM`, `MAILBRIDGE_PLANE_BASE_URL/API_KEY`, `MAILBRIDGE_WEBHOOK_SECRET` — без них бинарник не стартует
- `MAILBRIDGE_AUTH_USER/PASS` имеют дефолт `admin/admin` — **всегда** задавать свои в production

## Docs Index

| Файл | Когда читать |
|------|--------------|
| [README.md](README.md) | Первое знакомство |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Понимание системы |
| [docs/data-model.md](docs/data-model.md) | Работа с БД |
| [docs/api.md](docs/api.md) | Работа с API |
| [docs/ai-pipeline.md](docs/ai-pipeline.md) | AI-обработка, промпты |
| [docs/operations.md](docs/operations.md) | Деплой, запуск, отладка |
| [docs/adr/](docs/adr/) | Архитектурные решения |
| [PLAN.md](PLAN.md) | Текущий план разработки (v0.21.x → v1.0.0) |
| [archive/](archive/) | Реализованные планы (v0–16, v17–21, реархитектура) — только история |

## Workflow

1. Создать отдельную ветку от `main` (`fix/...`, `feat/...`, `docs/...`)
2. Внести изменения (Go: `make lint && make test`)
3. Фронтенд: `cd frontend && npm run dev` для проверки
4. Коммит: imperative, кратко
5. PR в `main`
6. **Дождаться зелёного CI** (джобы Lint + Test) — merge запрещён без green CI
7. После мержа — **релиз**: тег `v0.X.Y` + бинарник на GitHub (подробнее: `CONTRIBUTING.md`, «Правила релизов»)

> **Правило веток (с 2026-08-26):** прямой merge/push в `main` запрещён — `main` защищена на GitHub.
> Все изменения только через отдельную ветку под задачу + PR.
> **Green CI обязателен** (required status check `CI` на `main`). **Релизы — минорные `0.x.y` до v1**; после мержа — тег + бинарник.

## Критические инварианты

- Миграции БД должны быть идемпотентными (`IF NOT EXISTS`)
- `email_mapping` удалена — использовать `inbox_items`
- Вложения: CAS (SHA-256), файл хранится по hash, имя в БД
- AI-обработка: через очередь, не синхронно
- Вердикты: строгий JSON с полем `action`
- `task_comments.kind` — `user_comment` или `ai_verdict`

## Hermes (agent-specific)

- Файл контекста: только `AGENTS.md` (не создавать `.hermes.md`, `CLAUDE.md`)
- Лимит файла 20K символов — держать кратко
- Skill `mailbridge-dev` — точные команды и pitfalls
- При работе с БД: DDL-миграции в `internal/store/sqlite/migrations.go` (`sqlite.go` — только Open + WAL)