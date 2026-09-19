#!/usr/bin/env bash
# plan-gate.sh — МЕХАНИЧЕСКИЙ ГЕЙТ гигиены живого плана (смена «напоминание» → «замок»).
# Правило из CONTRIBUTING: живой PLAN.md = только открытые шаги (### [ ]) + ОДНА строка
# на закрытый/выпущенный шаг (квитанция → archive/). Этот скрипт проверяет это
# МЕХАНИЧЕСКИ, что чек-листы СТАРТА (п.5) и ВЫПУСКА (п.3) требуют.
#
# Использование:
#   scripts/plan-gate.sh start                 # ДО старта шага (прогоняем чек-лист СТАРТА п.5)
#   scripts/plan-gate.sh close v0.X.Y          # ПОСЛЕ тега v0.X.Y (квитанция должна быть уже
#                                              # в archive/, в живом плане — строки-индексы)
# Выход: 0 = PASS, 1 = FAIL (с списком нарушенных пунктов).

set -u
cd "$(dirname "$0")/.."

MODE="${1:-}"
TARGET="${2:-}"
PLAN="PLAN.md"

FAILS=()
PASS() { printf '  ok  %s\n' "$1"; }
FAIL() { printf '  ERR %s\n' "$1"; FAILS+=("$1"); }

last_tag() { git tag --list 'v0.*' --sort=-v:refname | head -1; }
ver() { # v0.27.2 -> 0.27.2
  printf '%s\n' "$1" | sed 's/^v//';
}
le() { # a <= b (поверк через sort -V)
  [ "$(printf '%s\n%s\n' "$1" "$2" | sort -V | head -1)" = "$1" ];
}

case "$MODE" in
  close)
    [ -n "$TARGET" ] || { echo "Использование: plan-gate.sh close v0.X.Y"; exit 1; }
    echo "== plan-gate close $TARGET =="
    if ! git rev-parse -q --verify "$TARGET" >/dev/null; then
      FAIL "тег $TARGET НЕ существует (п.3 ВЫПУСКА: тег по явной команде владельца)";
    else
      PASS "тег $TARGET существует"
    fi
    TVER="$(ver "$TARGET")"
    # 1) Строка в индексе «Выпущено»
    if awk '/^## Выпущено/,0' "$PLAN" | grep -q "v$TVER"; then
      PASS "индекс «Выпущено» содержит v$TVER"
    else
      FAIL "в «Выпущено» НЕТ строки v$TVER (п.3: одна строка на версию)"
    fi
    # 2) Живой план НЕ содержит развёрнутой квитанции: все строки ### [x] с v$TVER — строки-индексы (с «квитанция:» + archive/)
    BLOCKS=$(grep -n "^### \[x\].*v$TVER" "$PLAN" || true)
    if [ -z "$BLOCKS" ]; then
      PASS "в живом плане нет блока ### [x] v$TVER (квитанция полностью в архиве)"
    else
      while IFS=: read -r LN _; do
        [ -n "$LN" ] || continue
        LINE=$(sed -n "${LN}p" "$PLAN" | head -c 500)
        if printf '%s\n' "$LINE" | grep -q 'квитанци' && printf '%s\n' "$LINE" | grep -q 'archive/'; then
          REF=$(printf '%s\n' "$LINE" | grep -o 'archive/[A-Za-z0-9./_-]*' | head -1)
          if [ -f "$REF" ]; then
            PASS "строка ### [x] v$TVER — индекс с корректной ссылкой на $REF"
          else
            FAIL "строка ### [x] v$TVER (строка $LN) ссылается на несуществующий $REF"
          fi
        else
          FAIL "развёрнутая квитанция v$TVER ВЕСЬ ЕЩЁ в живом $PLAN (строка $LN). Правило: квитанция → archive/, в живом — ОДНА строка со ссылкой."
        fi
      done <<< "$BLOCKS"
    fi
    # 3) CHANGELOG-секция
    if grep -q "^## \[$TVER\]" CHANGELOG.md; then
      PASS "CHANGELOG: секция [$TVER] есть"
    else
      FAIL "CHANGELOG: секция [$TVER] НЕТ (п.2 ВЫПУСКА)"
    fi
    # 4) Чистое дерево
    if [ -z "$(git status --porcelain)" ]; then
      PASS "git status — чистое дерево"
    else
      FAIL "git status НЕ чист (незакоммиченные изменения)"
    fi
    ;;
  start)
    echo "== plan-gate start =="
    LTAG=$(last_tag)
    [ -n "$LTAG" ] && echo "  (последний релиз: $LTAG)"
    # Правило п.5 СТАРТА: живой план чист от квитанций выпущенных версий.
    CHECKED=0; WARNED=0
    while IFS= read -r H; do
      VER=$(echo "$H" | grep -oE 'v0\.[0-9]+\.[0-9]+' | tail -1)
      LINE=$(echo "$H")
      if [ -z "$VER" ]; then
        continue
      fi
      CHECKED=$((CHECKED+1))
      if [ -n "$LTAG" ] && le "$(ver "$VER")" "$(ver "$LTAG")" ]; then
        # Закрытый и ОБЫКНОВЕННО ВЫПУЩЕННЫЙ (<= последнего релиза) шаг: только строка-индекс.
        if printf '%s\n' "$LINE" | grep -q 'квитанци' && printf '%s\n' "$LINE" | grep -q 'archive/'; then
          REF=$(echo "$LINE" | grep -o 'archive/[A-Za-z0-9./_-]*' | head -1)
          if [ -f "$REF" ]; then
            PASS "$VER <= $LTAG: индекс строка -> $REF"
          else
            FAIL "$VER <= $LTAG: строка-индекс ссылается на несуществующий $REF"
          fi
        else
          FAIL "ГЛЮКАЖ: развёрнутая квитанция $VER (<= $LTAG — уже выпущено) ВЕСЬ ЕЩЁ в живом $PLAN. Старт запрещён до doc-PR «архивировать $VER» (п.5)."
        fi
      else
        WARNED=$((WARNED+1))
      fi
    done < <(grep '^### \[x\]' "$PLAN")
    if [ "$CHECKED" -eq 0 ]; then
      echo "  (в живом плане нет закрытых ### [x] блоков — ок)"
    elif [ "$WARNED" -gt 0 ]; then
      echo "  warn  $WARNED закрытых ### [x] без/с версией выше $LTAG (ожидается: закрыт, но ещё не выпущен) — разрешено до тега"
    fi
    # Лимит индекса «Выпущено» (макс 5 последних + строка-указатель; старое → CHANGELOG/archive).
    REL=$(grep -c '^- \*\*v0\.[0-9]*\.[0-9]* (' "$PLAN" || true)
    if [ "${REL:-0}" -gt 5 ]; then
      FAIL "«Выпущено» — $REL строк релизов; лимит = 5 последних (+1 указатель). Старые уберите в CHANGELOG.md (один указатель)."
    else
      PASS "«Выпущено» — ${REL:-0} строк релизов (≤5 — лимит соблюдён)"
    fi
    if [ -z "$(git status --porcelain | grep -v '^??' || true)" ]; then
      PASS "git status — не-трекованных локальных изменений нет"
    else
      FAIL "git status НЕ чист — чек-лист СТАРТА п.4 (посторонние локальные изменения)"
    fi
    ;;
  *)
    echo "Использование: plan-gate.sh start | close v0.X.Y"
    exit 1
    ;;
esac

echo
if [ ${#FAILS[@]} -eq 0 ]; then
  echo "PASS — гейт пройден"
  exit 0
else
  echo "FAIL — ${#FAILS[@]} нарушенных пунктов:"
  for f in "${FAILS[@]}"; do echo "  - $f"; done
  exit 1
fi
