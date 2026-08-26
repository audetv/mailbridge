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

## Перед PR

```bash
make lint && make test
```

## Планирование

- Текущий план: `PLAN.md`
- AI-интеграция: `PLAN.ai-inbox.md`
- Архитектура: `docs/adr/`

## Изменение промптов

- Основной промпт: `configs/email-assistant-v2.Modelfile`
- Промпт в коде: `internal/ai/orchestrator.go`
- После изменения — пересоздать модель: `ollama create email-assistant-v2 -f configs/email-assistant-v2.Modelfile`
