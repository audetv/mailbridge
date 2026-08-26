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
- **Green CI обязателен:** merge только после того, как Actions/CI полностью зелёная (оба джоба CI — Lint и Test — успешны). CI красная = merge запрещён, сначала починить на ветке.
- Мерж только после ревью; после мержа ветка удаляется.

## Правила CI/CD (gate)

- CI: `.github/workflows/ci.yml` → джобы `Lint` (golangci-lint + npm lint + build) и `Test` (`go test -race`).
- `main` защищена **required status check `CI`** → GitHub не даст mergнуть PR без зелёного CI.
- `go` lint (golangci-lint, gofmt) и `npm run lint` обязаны проходить до мержа.
- Коммит не считается готовым, пока `gh run list` для его head sha не green.
- После мержа убедиться, что push-запуск на `main` тоже green (иначе чинить отдельной веткой).

## Правила релизов

- **Релиз — минорный `0.x.y` (до v1.0.0).** До v1 меняется архитектура и API, поэтому релизы минорные; выход v1 — отдельное решение.
- После мержа PR в `main` делаем:
  1. CHANGELOG: секция `[0.X.Y]` с датой (обязательно, входит в notes релиза),
  2. **Тег `v0.X.Y` на смерженный commit main** — он запускает workflow `Release` (`.github/workflows/release.yml`): `make build` → GitHub Release + бинарник (asset `mailbridge`) автоматически,
  3. Проверка: `gh run list --workflow release.yml --limit 1` green, `gh release view v0.X.Y --json assets`.
- Версия в бинарнике (после мержа PR со сменой кода):
  тег `v0.X.Y` → workflow `Release` (fetch с полной историей) → `git describe --tags --abbrev=0` → Makefile `VERSION` (префикс `v` срезан: `0.X.Y`) → ldflags → `internal/version`;
  проверка: шаг «Version check» сравнивает вывод `./build/mailbridge version` с тегом.
- Локальный (offline) вариант: `make build && gh release create v0.X.Y build/mailbridge --title v0.X.Y` (см. `docs/operations.md`, «Релизы»).

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
