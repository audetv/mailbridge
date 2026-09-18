#!/usr/bin/env bash
# dev-stack.sh — пересобрать и перезапустить dev-стак :8081 со СТАРОЙ→НОВОЙ статикой.
#
# ПОЧЕМУ ЭТОТ СКРИПТ (инцидент v0.27.2, 2026-09-18):
#   //go:embed static (cmd/mailbridge/main.go:37) — статика вшита в бинарник
#   НА МОМЕНТ КОМПИЛЯЦИИ. go run компилирует fresh, НО читает cmd/mailbridge/static,
#   а этот снимок обновляется ТОЛЬКО руками (npm build + cp). Без этого шага
#   :8081 молча отдаёт СТАРУЮ статику — владелец видит старый UI, tests/CI pass.
#   Правило: ПОСЛЕ ЛЮБОГО UI-изменения (и merge в main) → этот скрипт → ТОЛЬКО потом
#   заявка визуала владельца.
#
# Использование:
#   ./scripts/dev-stack.sh           # stop + build(frontend+static) + start :8081 + verify
#   ./scripts/dev-stack.sh --no-stop # не убивать запущенный dev (если не наш)
set -euo pipefail
cd "$(dirname "$0")/.."

PORT="${PORT:-8081}"
STOP_OLD=1
for a in "$@"; do [ "$a" = "--no-stop" ] && STOP_OLD=0; done

echo "==> 1/4 стоп старых dev-процессов ($PORT)"
if [ "$STOP_OLD" = 1 ]; then
  pkill -f "exe/mailbridge|cmd/mailbridge" 2>/dev/null || true
  sleep 1
fi

echo "==> 2/4 npm build + sync → cmd/mailbridge/static (embed-снимок)"
(cd frontend && npm run build) >/dev/null
rm -rf cmd/mailbridge/static
cp -r frontend/dist cmd/mailbridge/static

# верификация свежести: hash index.html совпадает в dist и static
h1=$(sha256sum frontend/dist/index.html | cut -d' ' -f1)
h2=$(sha256sum cmd/mailbridge/static/index.html | cut -d' ' -f1)
[ "$h1" = "$h2" ] || { echo "FATAL: static != dist"; exit 1; }

echo "==> 3/4 запуск dev ($PORT) в фоне (go run + ldflags, make run-dev аналог)"
# VERSION: как в Makefile — последний git tag (vX.Y.Z → X.Y.Z) или dev
RAW_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
VERSION=${RAW_TAG#v}
[ -z "$VERSION" ] && VERSION=dev
[ -f configs/config.env ] && { set -a; . configs/config.env; set +a; }
nohup go run \
  -ldflags "-X github.com/audetv/mailbridge/internal/version.Version=$VERSION \
            -X github.com/audetv/mailbridge/internal/version.Commit=$(git rev-parse --short HEAD) \
            -X github.com/audetv/mailbridge/internal/version.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
  ./cmd/mailbridge --port "$PORT" > /tmp/mailbridge-dev.log 2>&1 &

echo "==> 4/4 health + /api/version"
for i in $(seq 1 30); do
  v=$(curl -s -m2 "http://localhost:$PORT/api/version" 2>/dev/null) && [ -n "$v" ] && break
  sleep 2
done
[ -n "$v" ] || { echo "FATAL: dev не поднялся, лог: /tmp/mailbridge-dev.log"; exit 1; }
echo "$v"
curl -s "http://localhost:$PORT/health"; echo
echo "OK: dev :$PORT готов к визуалу"
