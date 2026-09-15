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
- `make run-dev` — запуск в dev-режиме (порт берётся из `MAILBRIDGE_LISTEN`; vite-прокси при dev ждёт 8081 — т.е. в dev-конфиге `LISTEN=:8081`)
- `make clean` — очистка артефактов
- `make tidy` — go mod tidy

### Frontend (Vue)

- `cd frontend && npm run dev` — dev-сервер (Vite, порт 5173)
- `cd frontend && npm run build` — production-сборка
- `cd frontend && npm run preview` — предпросмотр собранного
- `cd frontend && npm run lint` / `npm run format` — ESLint 9 + Prettier (CI гоняет lint перед сборкой)

### Тестирование

- Unit: `make test` (Go) + `cd frontend && npm test` (vitest) — перед коммитом, который трогает код
- **E2E (Playwright в `frontend/`)**: обязателен при UI-коммите, касающемся **Filter / Tab / Select / Dialog / роутинга вкладок** — `cd frontend && npx playwright test` (против живого dev-стека: moка PrimeVue скрывают реальные баги)
- Стек для e2e: `make run-dev` (:8081) + `cd frontend && npm run dev` (:5173); данные — сид
  `cd frontend && npm run e2e:seed` (идемпотентный: проекты + модуль «Сайт ТРК» + задачи).
  **После КОПИРОВАНИЯ БД из прод в dev — ОБЯЗАТЕЛЬНО прогнать `e2e:seed` перед e2e.**

### Dev-окружение (порты)

- `:8080` = **ПРОД** mailbridge — НЕ трогать (не kill/restart, не писать в его БД); действия на проде — у пользователя
- `:8081` = dev-бекенд (`make run-dev`); `:5173` = vite dev (прокси на 8081)
- Агенту допустимо поднимать/останавливать dev-бекенд для тестов (по необходимости)

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
- Обязательные для запуска (валидация в `internal/config/config.go`): `MAILBRIDGE_IMAP_SERVER/USER/PASS`, `MAILBRIDGE_SMTP_SERVER/FROM` — без них бинарник не стартует (Plane/webhook-secret сняты в v0.22.0)
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
| [PLAN.md](PLAN.md) | **Единственный живой план** (курсор + квитанции закрытых + открытые шаги + «Выпущено») |
| [archive/](archive/) | Закрытые блоки по версиям (v0–16, v17–21, v0.22, v0.23) — только история |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Правила коммитов/CI/релизов + **§Методология разработки** (режимы A/B/C, чек-листы, стоп-гейт) |

## Workflow

1. **Курс:** шапка «Статус» в `PLAN.md` + режим (A разработка / B обсуждение / C баг — `CONTRIBUTING.md` §Методология)
2. Ветка от `main`: `feat/stepN-<slug>` (A) / `fix/stepN-<slug>` (C) — **не** `docs/...` (конфликт с папкой `docs/`)
3. Изменения (Go: `make lint && make test`; фронты: `npm run dev` + e2e по триггеру)
4. Коммит: imperative, кратко
5. PR в `main` **+ в том же PR** CHANGELOG-буллит + план (квитанция шага `[x]`)
6. **Green CI** (Lint + Test) — merge запрещён без green
7. **Стоп-гейт:** тег `v0.X.Y`/релиз — **только после явного решения владельца** (`CONTRIBUTING.md`, «Правила релизов» + §Методология)

> **Правило веток (с 2026-08-26):** прямой merge/push в `main` запрещён — `main` защищена на GitHub.
> Все изменения только через отдельную ветку под задачу + PR.
> **Green CI обязателен** (required status check `CI` на `main`).
> **Нумерация (с 2026-09-15):** до v1.0.0 — **только MINOR** (`0.x.y`); PATCH = только hotfix. **Тег ставит владелец** после явного решения (стоп-гейт: следующий шаг до него не стартует).
> **Правило 8:** тег — владелец; PR-мерж + CHANGELOG/план-квитанция — агент (в том же PR, что и код шага).
> **Правило 10 (1 шаг = 1 цель):** фиксы — только по багу/ревью (CI, покрытие), «заодно» нельзя.

## Критические инварианты (v0.22.0)

- Миграции БД должны быть идемпотентными (`IF NOT EXISTS`)
- `email_mapping` удалена — использовать `inbox_items`
- Вложения: CAS (SHA-256), файл хранится по hash, имя в БД
- AI-обработка: через очередь, не синхронно
- Вердикты: строгий JSON с полем `action`
- `task_comments.kind` — `user_comment` | `ai_verdict` (история входящих) | `report` (внутренний отчёт) | `reply` (черновик ответа)
- `task_comments.approved` — флаг комментария (admin-only `PATCH /api/comments/{id}/approve`), NOT статус задачи
- Проекты → Модули (API/UI: `epics`/`epic_id`) → Задачи; задача без модуля допустима
- Auth: 2 юзера — admin (`MAILBRIDGE_AUTH_USER/PASS`) + агент `hermes` (`MAILBRIDGE_AGENT_PASS` обязателен для активации)
- **Срез Plane (v0.22.0):** Plane/webhook удалены (ADR-0001); проекты для AI — из SQLite; отправка e-mail пользователю НЕ идёт (reply = только черновик)

## Hermes (agent-specific)

- Файл контекста: только `AGENTS.md` (не создавать `.hermes.md`, `CLAUDE.md`)
- Лимит файла 20K символов — держать кратко
- Skill `mailbridge-dev` — точные команды и pitfalls
- При работе с БД: DDL-миграции в `internal/store/sqlite/migrations.go` (`sqlite.go` — только Open + WAL)