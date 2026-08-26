# Contributing

## Dev Loop

```bash
git clone ...
cd mailbridge
make build
make test
make lint
```

## Правила коммитов

- Imperative: `feat:`, `fix:`, `docs:`, `refactor:`
- Кратко: что и зачем

## Правила веток

- **Прямой push / merge в `main` запрещён** — `main` защищена на GitHub, все изменения только через PR.
- Одна задача — отдельная ветка от актуального `main` (`fix/...`, `feat/...`, `docs/...`).
- Перед открытием PR — CI зелёное, `make test` локально.
- Мерж только после ревью; после мержа ветка удаляется.

## Перед PR

```bash
make lint && make test
```

## Планирование

- Текущий план: `PLAN.md` (этапы A–E до v1.0.0)
- Реализованные планы: `archive/` (только история)
- Архитектура: `docs/adr/`

## Изменение промптов

- Основной промпт: `configs/email-assistant-v2.Modelfile`
- Промпт в коде: `internal/ai/orchestrator.go`
- После изменения — пересоздать модель: `ollama create email-assistant-v2 -f configs/email-assistant-v2.Modelfile`
