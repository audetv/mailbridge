# Квитанция hotfix v0.29.1 (шаг 29.1, issue #87) — 2026-11-25

### [x] 29.1 — Хотфикс: openai-клиент не передаёт images (issue #87, v0.29.1) — CLOSED 2026-11-25

- **Баг (выпущенный, задел прод-путь после свитча на strata):** `internal/ai/openai.go:47` — параметр изображений отброшен (`_ []string`); оркестратор (`orchestrator.go:189`) массив `images` передаёт всегда, ollama-клиент клал его в `images` запроса `/api/generate` (vision работал 3 мес.), openai-клиент — нет. Замер прода с 25.11 20:18: 26 писем с картинками, 0 непустых `image_note`.
- **Решение:** только `openai.go`: user-сообщение → мультимодальный content `[{type:text},{type:image_url,"data:image/<mime>;base64,…"}]` (strata/llama.cpp принимает — проверено живым тестом «Красный»), MIME из decoded-байтов (`http.DetectContentType`), дефолт image/png. Без рефакторинга.
- **Исполнение:** ветка `hotfix/openai-vision` от тега `v0.29.0`, коммит `2da678d`: RED→GREEN `TestOpenAIClient_WithImages` (форма payload при images>0) + `WithoutImages` (регресс: без картинок content = строка). `make lint` 0, все Go-пакеты ok; CI (Lint/Test) pass. PR **#88** (squash `9ba75a5`), issue **#87 CLOSED**.
- **Приёмка (тем же сценарием бага, regression):** живое fixture-письмо с PNG-вложением (текст «VISION TEST 29.1») через dev :8081 на strata → вердикт `"image_note":"VISION TEST 29.1"` дословно + созданная задача; fixture-строки из dev-БД вычищены.
- **После выпуска:** прод :8080 обновлён `deploy-released.sh v0.29.1` (команда владельца «close тег deploy», 25.11).
