#!/usr/bin/env bash
# dev-stack.sh — обновить dev :8081 (статика + бинарник) и перезапустить.
#
# ПОЧЕМУ ЭТОТ СКРИПТ (инцидент v0.27.2, 2026-09-18): статика вшита в бинарник
# через //go:embed cmd/mailbridge/static — снимок НА МОМЕНТ КОМПИЛЯЦИИ. Без
# `make build` (npm build + cp dist → static + go build) :8081 молча отдаёт
# СТАРУЮ статику: владелец видит старый UI, тесты/CI pass.
#
# Использование: ПОСЛЕ ЛЮБОГО merge в main (или UI-изменений) и ДО заявки
# визуала владельца:   ./scripts/dev-stack.sh
# (Живой UI-девелопмент — vite :5173 + `make run-dev`, этот скрипт не нужен.)
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> стоп старого dev"
pkill -f "exe/mailbridge|build/mailbridge" 2>/dev/null || true
sleep 1

echo "==> make build (frontend → static → бинарник, version/commit из git)"
make build
[ -f configs/config.env ] && { set -a; . configs/config.env; set +a; }
nohup make run > /tmp/mailbridge-dev.log 2>&1 &

echo "==> health + /api/version"
v=""
for i in $(seq 1 30); do
  v=$(curl -s -m2 "http://localhost:${MAILBRIDGE_LISTEN#:}/api/version" 2>/dev/null) && [ -n "$v" ] && break
  sleep 2
done
[ -n "$v" ] || { echo "FATAL: dev не поднялся, лог: /tmp/mailbridge-dev.log"; tail -5 /tmp/mailbridge-dev.log; exit 1; }
echo "$v"
echo "OK: dev ${MAILBRIDGE_LISTEN} готов к визуалу"
