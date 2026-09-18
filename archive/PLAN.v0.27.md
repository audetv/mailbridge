# PLAN — v0.27 (квитанции закрытых шагов)

Выпущено: **v0.27.0 (2026-09-18)** — шаг 8 «Версия в UI» (РЕЖИМ A, PR #60, squash `c6edd81`, CI green). Тег `v0.27.0` → `800a62f` (tip main) → GitHub Release, бинарник `mailbridge 0.27.0 (commit: 800a62f, built: 2026-09-18_10:41)` (Version check пройден локально). Тройка пройдена 2026-09-18: отчёт ✅, визуальная проверка на 192.168.1.71:8081 ✅ («оставляем, ок. принято»), явное «выпускай тэг» ✅. В живой `PLAN.md` — одна строка в «Выпущено» + курсор на **«Сегодня/спринты» (v0.28.0+)**.

## Шаг 8 — Версия в UI (v0.27.0) ✅ 2026-09-18 (РЕЖИМ A; PR #60 merged → `main` `c6edd81`)
**Решение (зафиксировано, режим B закрыт 2026-09-18, решения владельца):**
1. `GET /api/version` — **публичный** (без JWT), JSON `{version, commit, built}` из `internal/version` (ldflags). → `internal/web/version.go` (хендлер), `cmd/mailbridge/main.go` (роут); метод ≠ GET — 405.
2. UI — шапка Dashboard: версия в строке (рядом «● Онлайн/Офлайн»), **commit + time сборки — в нативном тултипе** (`title`, владелец уточнил 2026-09-18 и оставил, «устроит»). → бейдж `.version-badge` в `DashboardView.vue`.
3. Значения показывать **как есть** (`dev`/`none` допустимы) — полезно: в dev видно реальную сборку. ✓ (тесты unit + e2e)
4. `make run-dev` **вшить ldflags** (ранее bare `go run` → `dev`/`none`). → Makefile `run-dev` берёт `$(LDFLAGS)`.
**Приёмка:** бинарник-релиз = свой тег+commit ✓ (`0.27.0 (commit: 800a62f)`); dev = HEAD commit + время сборки ✓ (`0.26.0 (commit: 800a62f, built: …)`); `dev`/`none` не подавлены ✓; e2e: `GET /api/version` без auth = 200 + JSON + бейдж в шапке ✓ (`tests-e2e/version-header.spec.js` 2/2).
**Тесты:** Go `internal/web/version_test.go` 2/2; витест `frontend/tests/views/DashboardVersionBadge.spec.js` 4/4; e2e 2/2 (полный прогон 41 pass, flake `bulk-actions` — pre-existing, в изоляции green); `make lint` (golangci 0 issues) + `make test` + `npm run lint/test` — green; CI run `35331091376`: Lint pass + Test pass.
**Границы:** НЕ «проверка новой версии»/уведомления (отдельная идея, если понадобится). Доки: `docs/api.md` § «Версия (v0.27)», `ROADMAP.md` § «Версия в UI» → реализован, `CHANGELOG.md` [0.27.0].
