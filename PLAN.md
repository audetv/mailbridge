# Mailbridge — план (текущий)

> **Методология** — `CONTRIBUTING.md`, раздел «Методология разработки» (режимы, уровни, чек-листы, стоп-гейт). Этот файл — только курс: где стоим и что дальше.
> **Схема v0.23**: живой план = только открытые шаги. Выпущенные/завершённые блоки уходят в `archive/`, в этом файле — **одна строка** в «Выпущено».
> **Рестарт сессии (промпт)**: «план `site/mailbridge/PLAN.md`, skill `mailbridge-dev`; с какого шага читать — „Статус“ в шапке; чек-лист СТАРТА».

## Статус (курсор — читать первым)

**Дата обновления:** 2026-09-16
**Версия:** в разработке **v0.24**; **v0.23.0 — выпущен 2026-09-15** (тег по решению владельца; блок 0–5 в `archive/PLAN.v0.23.md`)
**Текущий шаг:** **шаг 7 (Due date/SLA) — РЕЖИМ B, старт после решения владельца** (код — только после фиксации решения в шаге 7).
**Предыдущий шаг (завершён):** **шаг 6-F (hotfix чистота identity RFC822) — CLOSED 2026-09-16** (режим A; ParseFromHeader + EnsurePersonFromIncoming + эвристика «машина» + idempotent self-heal; квитанция в шаге 6-F).
**Предыдущий (завершён):** **шаг 6 (Персоны) — CLOSED 2026-09-15** (режим A; к релизу v0.24.0 вместе со шагом 7).
**Пред. шаг:** v0.22.0 (тег 2026-08-30) — архив.

## Что уже работает (карта — новые шаги строить поверх этого)

v0.24 (в разработке): **Шаг 6 Персоны** (CLOSED 2026-09-15): `persons`+`person_identities` (UUID, Postgres-совместимая схема), ролевые ссылки `tasks.requestor_id/assignee_id` + `task_comments.author_person_id`, авто-создание персоны при первом контакте + авто-назначение assignee при закрытии, UI-вкладка «Персоны» + selects/колонки Заказчик/Исполнитель + deep-link `?requestor_id=`. Шаг 7 (Due/SLA) — впереди.
v0.23.0 (блок закрыт, 2026-09-15, полный текст в `archive/PLAN.v0.23.md`):
входящий поток IMAP + MIME-парсинг + вложения (extraction) · ws-подписки · история статусов (`task_status_history`, `SetTaskStatus`) · копирование/MD-рендер/автолинкинг комментариев · bulk actions (статус/проект, `PATCH /api/tasks`, WS `batch_update`) · **AI-вердикт = только саммари; спойлер с НОВОЙ ЧАСТЬЮ письма — в саммари `user`; вложения legacy user-саммари остаются** · AI-очереди retry 1м/5м/15м/1ч.
v0.22.0: проекты/модули, срез Plane, outbound SMTP, темы, observability — `archive/PLAN.v0.22-...md`.

## Отдельные решения (зафиксированы, действуют независимо от шагов)

1. **Хранение (`attachments`) — не трогаем** (решение владельца, 15.09; баги legacy — только отдельными шагами/багами).
2. **AI-вердикт — только саммари**; спойлер письма — в саммари `user` (решение 15.09, закрыт шагом 5).
3. **Патч-версии = только hotfix.** Фиче-блок = **только MINOR**: v0.22.1 не создавалась (фиксы вошли в v0.23.0).
4. **PR с fix по багам — допустим** (в рамках шага: fix-коммит → green CI → merge → план/CHANGELOG).
5. **Правило 8:** тег делает владелец, но **PR-мерж + CHANGELOG/план — агент** (расширено 15.09: PR #27 = PR #26 + «тег нет» → PR #27).
6. **Правило 10:** строго один шаг; fix — только по багам, не «заодно» (решение владельца 2026-09-13: «1 шаг = 1 цель + фиксы по багу/ревью, не более»).

## Закрытые (квитанции — в архиве)
- **v0.23 — шаги 0–5:** ws-reconnect · вложения (CAS+API) · история статусов · копирование/MD/автолинки · bulk · AI-вердикт = саммари — полный текст с дискуссионными пометками: `archive/PLAN.v0.23.md`
> **Нумерация** — сквозная: шаги 0–5 = v0.23 (архив), 6–7 = v0.24 (ниже).

## Открытые шаги

### [x] Шаг 6 — Персоны: справочник + assignee на персонах + авто-назначение (v0.24)
**Статус: CLOSED (реализация завершена 2026-09-15, режим A) → к релизу v0.24.0.**

**Квитанция (реализация, 2026-09-15):** Справочник `persons`+`person_identities` (UUID, Postgres-совместимая схема) + backfill из legacy-текста; API `/api/persons*` (CRUD, identities, suggest/merge, `PUT /api/tasks/{id}/persons`); processor Rule 7 (авто-создание + авто-assign при закрытии); UI: вкладка «Персоны», selects/колонки «Заказчик»/«Исполнитель», фильтры + deep-link `?requestor_id=`. Баг этого шага исправлен: `ListTasks` не перечислял `t.requestor_id/assignee_id` (фильтр работал, строки — `null`) → добавлены + scan. Тесты: unit (fast-path/fuzzy/merge/auto-assign), web `persons_test.go`, e2e `persons.spec.js` (6 passed; полный e2e 31). **CHANGELOG ✓ [0.24.0].** Коммиты: `6d0916c` (store), `7fae2b5` (web), `29c7a36` (processor), фронт+fix — текущий. *Граница (решение владельца): `confirmed`-статус и раздел «Не подтверждённые»/match_rejections UI — задел в модели, не блокирует релиз; к v0.24.1 по опыту.*

**Цель.** Действующие лица процесса (персоны) — справочник `persons` + идентичности `person_identities`; задачи ссылаются на персонах UUID, не на тексте; исполнитель проставляется автоматически при закрытии задачи подтверждением по письму.

**Модель (кратко; канон — `docs/ontology.md` §7.7/7.7.1):**
- `persons`: id (uuid) · name (display) · org · **is_internal** (hint «своя команда», дешёво искореняем, решение: принимает) · **confirmed** (узнана vs авто-гипотеза) · archived.
- `person_identities`: kind (`email`/`phone` на старте; `telegram`/`vk`/`github`/`gitea` — будущие) · value · external_id (устойчивый id провайдера; ник/email — secondary) · is_primary · provenance (auto/suggested/manual); UNIQUE(kind, value).
- **Роль НЕ поле персоны** — контекст связи: `task.requestor_id` / `task.assignee_id` / `comment.author_person_id` → `persons.id`. Одна персона — разные роли в разных записях.
- Идентичность — НЕ учётная запись; мост к RBAC = будущий `users.person_id` (задел, не код).

**Решения владельца (зафиксировано 2026-09-15):**
1. Наименование — **«Персона»** (канон онтологии §7.7).
2. Порядок v0.24: **6 персоны → 7 due/SLA → 8 спринты** (персоны — фундамент: из них «кому долг, когда»; без них due/SLA и будущие источники стоят).
3. Кинды в v0.24: **email + phone**; telegram/github/gitea/vk — модель в онтологии, код — источник-шаги 9+.
4. Fast-path: (kind, value) найден → **привязать молча** (это знание, не гипотеза); ошибки — обработаем при прецеденте.
5. Предложение нового контакта: fuzzy по имени → карточка [Да/Нет] в UI; **«Нет» → `match_rejections`** (не повторять), плюс **отмена мисклика**: toast «Отменить» (5–10 с) + раздел **«Не подтверждённые»** (связать/объединить/отвязать) + профиль персоны (add/unlink identity).
6. Авто-назначение: триггер — переход задачи в `done`/`closed`; условие — последнее подтверждение-письмо от P, P не assignee; **ручное > авто**; след — `task_status_history.by = email(P)` (наблюдаемость).
7. Персона авто-создаётся при первом контакте (confirmed=false; UI до подтверждения показывaет email, не пустое имя).
8. AI в идентификации — НЕ в v0.24 (rule-based + ручной; к AI — по опыту). RBAC/входы — НЕ в v0.24 (отдельный шаг, онтология-мост).
9. Backfill: distinct из `tasks.from_email`/`tasks.assignee`(legacy текст)/`task_comments.author` → персоны + идентичности; legacy `tasks.assignee` (текст) остаётся до конца шага, UI/API — на `person_id` (совместимость).

**План реализации (режим A):**
1. Миграции: `persons`, `person_identities`, `match_rejections`, FK-колонки `tasks.requestor_id/assignee_id`, `task_comments.author_person_id` (идемпотентно, IF NOT EXISTS).
2. Backfill-миграция (первопрохождение: персоны из текущих данных) + индекс UNIQUE(kind,value).
3. Store/API: `GET/POST/PATCH /api/persons`, `GET /api/persons/{id}/identities`, suggestions + accept/reject, merge, unlink; task-поля `requestor_id`/`assignee_id` (legacy-фильтр `?assignee=` работает по email).
4. Процессор: auto-assignee при закрытии (правило п.6), `task_status_history.by`.
5. UI: справочник «Персоны» (список/профиль), раздел «Не подтверждённые», карточки-предложения; задача: requestor (имя/email), assignee — **Select по персонам** (не InputText), бейдж «авто-назначено»; колонки/фильтры «исполнитель/заказчик».
6. Тесты: unit (fast-path, fuzzy, reject, merge, backfill, авто-assignee + приоритет ручного) + e2e (связывание/отвязание/авто-назначение при закрытии). e2e-сид — дополнить персонами.
7. CHANGELOG-буллит + квитанция шага `[x]` в этом PR.

**Риски/границы:** is_internal — только hint; identity без имени = «персона в процессе узнавания» (UI показывaет email); команды (info@) — open question в онтологии, не в v0.24.

**Решение владельца (2026-09-15): схема — под Postgres, SQLite лишь слой.** Будущие слои: Postgres + ManticoreSearch (поиск). Правила для новых таблиц:
- UUID (`persons.id`, FK) = `TEXT` в SQLite-реализации (тип уровня реализации, в Postgres станет `UUID`); PK — без `AUTOINCREMENT`.
- Booleans — `INTEGER NOT NULL DEFAULT 0 CHECK (col IN (0,1))` (в Postgres → `BOOLEAN`).
- Идентичность email: normalization case-fold в app-слое при записи + lookup; уникальность — `UNIQUE(kind, value)` на уровне таблицы (диалект-нейтрально).
- FK — `ON DELETE SET NULL` + `IS NOT NULL` где семантика требует; archived — «вместо удаления» (закрытые задачи не теряют ссылку).
- Движки поиска/fuzzy (LIKE/inmemory) — НЕ в схеме: ManticoreSearch — будущий слой, SQLite-LIKE — только dev-заглушка в app-слое.
- Backfill — идемпотентно через `UNIQUE` + `INSERT OR IGNORE` (диалект SQLite; в Postgres — `ON CONFLICT DO NOTHING`).
- Правило: Store-интерфейс — единственный диалект-aware слой; бизнес-код (processor/web) не знает диалект.

### [x] Шаг 6-F — хотфикс: чистота `person_identities` + разбор RFC822-отправителя (баг шага 6) — РЕЖИМ A
**Добавлен 2026-09-15 после теста владельца на dev. Статус: решения зафиксированы (владелец подтвердил), исполнение — сессия от 2026-09-16.**

**КВИТАНЦИЯ (CLOSED 2026-09-16):** реализовано по scope п.1–5,7–8; ветка `hotfix/6-f-clean-identity`:
- `extractor.go` — `ParseFromHeader(name, email, display)`: quoted-имя, bare email, legacy-форма **без** закрывающей `>`, multi-address (первый адрес) — `extractor_6f_test.go` (+регресс-кeйсы).
- `sqlite/persons.go` — `EnsurePersonFromIncoming(name, email)`: строгий email-контракт (`EnsurePersonByEmail` на грязном входе — error, не мусор), авто-имя из `from_name`, идемпотентный self-heal мусорной identity (value → чистый email), эвристика «машина» (no-reply/noreply/postfix/mailer/… → `org='машина'`); `FindIdentity`/`Suggest` не падают на identity.
- `adapters`/`processor` — входящий → `EnsurePersonFromIncoming(FromName, FromEmail)` (не сырой хедер); legacy-колонки не тронуты (решение владельца).
- Dev-БД (п.5, порядок «код → DROP persons → бинарник»): identity = **только чистые email** (ГЛОБ-проверка: 0 грязных из 38), no-reply-персоны → `org='машина'`, имена авто из `from_name`.
- Тесты (квитанция п.7): unit extractor/store/adapter/processor `go test ./...` green; `make lint` 0 issues; vitest 119/119; e2e `persons.spec.js` **8/8** (A–E + **G**: legacy-форма inbox → задача → персона с чистым identity + авто-имем; **H**: no-reply → `org='машина'`).
- Отклонение от СТАРТ-порядка (уточнить владельцу): dev-БД уже была поднята до-фикс бинарником (инцидент 2026-09-15) — DROP persons/identities + пересборка/рестарт исправленным бинарником выполнен (п.5), порядок «сначала код» сохранён в итоге.

**СТАРТ (новый агент, без анализа — порядок зафиксирован, отклонения НЕЛЬЗЯ):**
```
1. read_file PLAN.md → этот шаг (6-F). Всё уже решено — только исполнять.
2. НЕ поднимать dev (8081/5173) до п.5 включительно — до-фикс backfill регенерирует ломаные персоны (инцидент 2026-09-15, см. проблему п.4).
3. TDD по scope п.1–3,5 (сначала тесты красные):
   - internal/extractor/ — parseFromHeader (name, email, display) + тесты edge-cases;
   - internal/store/sqlite/persons.go — жёсткий email-контракт + EnsurePersonFromIncoming (авто-имя, эвристика «машина») + тесты;
   - internal/processor — вызов EnsurePersonFromIncoming(name, email).
4. make lint && make test (go) + cd frontend && npm run lint && npm run build && npm test.
5. go build -o /tmp/mailbridge-dev ./cmd/mailbridge → в dev: sqlite3 data/mailbridge.db "DROP TABLE person_identities; DROP TABLE persons;" → поднять dev (/tmp/mailbridge-dev, :8081) → проверить: identity = чистые email, name заполнен, no-reply → org='машина' (в prod persons ещё нет — это нормальный backfill-сценарий).
6. cd frontend && npm run e2e:seed → npx playwright test persons.spec.js (+ новые test G/H из п.7).
7. CHANGELOG п.8 → commit → PR hotfix/6-f-clean-identity → CI → merge (правила CONTRIBUTING + skill mailbridge-dev; помнить: make lint НЕ включает vitest).
8. Отметить шаг [x] + квитанцию, курсор → шаг 7.
```
**Инцидент-память (читай перед стартом):** dev-БД = копия prod + ломаные персоны — это ОЖИДАЕМОЕ состояние, не баг копирования; чиним кодом, а БД — DROP persons/identities + рестарт исправленным бинарником (п.5).

**Проблема (найдена в owner-тесте, подтверждено по prod-БД):**
1. `inbox_items.from_contact` — ВСЕ строки вида `Имя <email` **без закрывающей `>`** (prod: 53 distinct из вида, 0 с `>`): `strings.Trim(decoded, "<> ")` в `extractor.go:152` (cutset режет обе скобки с обоих концов).
2. `EnsurePersonByEmail`-family (`sqlite/persons.go:396`, `normalizeEmail` 219) принимает ВХОД как email, но получает сырой RFC822-хедер `имя <email@x.ru` → identity value = мусор → нарушает `UNIQUE(kind,value)`-натурный ключ, ломает `SuggestMatch`/`primary_email`/label-формулу; dev-БД: 32 из 45 identity = отрывки, 36 безымянных персон.
3. **Исправляется НЕ в `inbox_items.from_contact`/`tasks.assignee` (legacy — остаётся как есть; решение владельца 2026-09-15: в dev-разгаре старые записи не чиним, главное — новые)** — только путь persons.
4. **Инцидент 2026-09-15 (важен для понимания):** после пересадки копии prod в dev, dev был поднят **на старом (до-фикс) бинарнике** — авто-backfill при старте (таблиц persons не было) **снова создал 36 ломаных персон** (created_at 20:23) из тех же отрывков (`tasks.from_email`, `assignee`, `comments.author`). Служебный вывод: **БД — расходный материал, чиним сначала КОД**, dev поднимать только исправленным бинарником, иначе backfill регенерирует мусор.

**Решения (владелица, 2026-09-15):**
1. **Чиним сейчас, отдельным хотфикс-шагом (стратегия (a))** — до выхода на шаг 7, чтобы шаг 7 строился на чистом фундаменте.
2. **Автоподстановка имени** при авто-создании персон из входящих: `from_name` (у extractor уже парсится, 53/53 в prod заполнен) → `persons.name` (пустое имя → заполнить). **Эвристика «машин/но-репл»: локаль email `no-reply|noreply|no_?reply|postfix|mailer|donotreply|auto(no)?`** → имя всё равно ставим (это «name/псевдоним»), но помечаем `org=` `«машина»` и не пытаемся «узнать человека»; такие персоны — отдельный анализ позже (решение владельца: в реальном использовании такие email есть — support/gcarenda, Selectel, SpaceWeb и т.п.).
3. **Чиним только `person_identities` + авто-имя**, не legacy-колонки (`from_contact`, `tasks.assignee`, `task_comments.author`) — те остаются как есть (решение владельца: приемлемо на dev, в prod step 6 ещё не выпущен).

**Scope (режим A):**
1. **Extractor — структура, не строки:**
   - `extractor.go` — `From: cleanHeader(GetHeader("From"))` → заменить на разбор RFC822: выдать `FromName string`, `FromEmail string`, `FromDisplay string` (совместимый legacy-текст `имя <email>` **с** закрывающей скобкой для `inbox_items.from_contact` — но для отладки человек-читаемый).
   - `extractNameFromEmail` + новый `extractEmailFromFrom` — общий `parseFromHeader(from string) (name, email, display string)`.
   - `adapters/email_adapter.go` — `FromContact: display` (совместимость UI-история), `FromName: name`, + `FromEmail: email` (новый, struct-поле `store.InboxItem`) и `store.Task.FromEmail` (новый — для persons-моста).
   - `extractor_test.go`/`adapters/email_adapter_test.go` + fixture: `«Имя <a@b.ru» → (name=Имя, email=a@b.ru, display=Имя <a@b.ru>)`; edge: `«<a@b.ru» → (name='', email=a@b.ru)`; `a@b.ru → (name='', email=a@b.ru)`; quoted `"Имя Фамилия" <a@b.ru>`; multi-address (take first).
2. **Store/Persons:**
   - `store.go` (interface) — `EnsurePersonByEmail` **остаётся** (контракт: email); + новый/дополненный `EnsurePersonFromIncoming(ctx, fromName, fromEmail string) (*Person, error)` — принимает разбор из extractor (не сырой хедер).
   - `sqlite/persons.go` — `EnsurePersonByEmail` **жёстко** требует email-вид (`@` + `.` после @), иначе error (не мусор в identity). Новый helper `personFromIncoming(name, email)` — idempotent: identity(email) → person; если person.name пуст и имя пришло — автозаполняем (provenance: `name_from_incoming` в `match_rejections`-style note НЕ нужно — просто `updated_at`).
   - **Эвристика машина/но-репл:** локаль email по `no-reply|noreply|auto( no)?|postfix|mailer|bot|donotreply|mailer-daemon|abuse|spamtraps?(list|)?@` (regex) + имя «машина» в org — помечаем `persons.org` = «машина» (не имя); UI-бейдж «машина» (optional, low-pri).
   - `FindPersonByEmail` + `SuggestMatch` (если есть) — не должны падать на identity-виде.
3. **Processor:** `processor.go:252/313` — `EnsurePersonByEmail(email.From)` → **`EnsurePersonFromIncoming(name, email)`** (в `processor.go` `email.FromName` уже есть после шага 1; использовать его).
4. **Dev-БД — УПРАЩЁНО (решение владельца 2026-09-15, «пропустить этот шаг»):** пересадка как отдельный шаг **отменяется — БД больше НЕ КОПИРУЕМ**. Текущее состояние dev-БД уже годится: это копия prod (507 inbox) + авто-backfill (36 ломаных персон). Порядок: **сначала фикс кода** (п. 1–3, 5) + тесты → сборка бинарника → **сброс персон в dev** (`sqlite3 data/mailbridge.db "DROP TABLE person_identities; DROP TABLE persons;"`) → рестарт dev (исправленным бинарником) → backfill перезапустится (идемпотентно) → **проверяем**: все identity = чистые email, `name` автозаполнен, no-reply → `org='машина'`. ТОЛЬКО потом `npm run e2e:seed` + e2e. **Правило: dev НЕ поднимать до готовности фикса — любой старт до-фикс бинарника регенерирует ломаные персоны.**
5. **Идемпотентное one-off** (обязательный, но на новой копии — no-op): в код добавить (по решению владельца: миграция в code, не SQL-скрипт): при `EnsurePersonFromIncoming` — если identity по email уже есть но value не чистый email → переименовать identity value → чистый email (idempotent, идемпотентный). Это страховка на случай prod-деплой с уже существующей БД с мусором (например, если владелец закинет v0.24.0 на прод).
6. **UI** — low-pri, не блокирует: бейдж «машина» в `PersonsView` + `TaskTable` (org=`«машина»` → иконка ⚙️); name авто — уже отображается (label).
7. **TDD:** unit-тесты:
   - `extractor_test.go` — `parseFromHeader` все edge cases (quoted, bare email, no-`>`, multi).
   - `sqlite/persons_test.go` — `EnsurePersonByEmail("имя <a@b.ru>")` → **error** (не мусор); `EnsurePersonFromIncoming(«Имя», «a@b.ru») → person.name=«Имя»`; idempotence (2 calls, 1 person).
   - `adapters/email_adapter_test.go` — `FromContact`/`FromName`/`FromEmail` при разных From.
   - `processor_test.go` — `EnsurePersonFromIncoming` вызывается с (name, email) от `email` struct (не с хедером).
   - **e2e:** persons.spec.js — новый test G: «входящее письмо (via seed API, `from_email=«Имя <x@y.z»`) → auto-create person.name=«Имя», identity.email=x@y.z, label формула работает»; **test H:** no-reply case (name=«машина», org-бейдж).
8. **CHANGELOG** — `[0.24.0] Fixed:` bullet: `persons identity: чистый email вместо RFC822-хедера; авто-имя из from_name; эвристика «машина» для no-reply.`
9. **Квитанция (чек-лист закрытия):** `make lint && make test` (go golangci + go test) + `cd frontend && npm run lint && npm run build && npm test` (vitest!) + e2e persons full (включая G/H) + PR (branch `hotfix/6-f-clean-identity`) + merge + squash-тест `e2e:seed` + `git push`. **Обязательный pitfall:** `make lint` НЕ включает vitest (см. mailbridge-dev SKILL).
10. **Отложенные (не блокируют):** чистка legacy `inbox_items.from_contact`/`tasks.assignee` (шаг 9+), UI «машина» бейдж (по опыту), Manticore (после Postgres), `confirmed`/match_rejections (v0.24.1 по опыту).

**Риски:** `@`-вид email в `EnsurePersonByEmail` — не ломать (E2E A — уже проверяет). `FromName` для `inbox_items` — не ломать (UI-история). Processor — не сломать (auto-assign по email).

**Чек-лист закрытия:** CONTRIBUTING.md §Методология.

### [ ] Шаг 7 — Due date: дата «до…/нужно до…» + SLA (v0.24) — РЕЖИМ B

**Цель.** Задача знает дату, до которой нужно ответить/выполнить («до 18.09») — статус/SLA/напоминания. (Зависит от шага 6: SLA меряется «кому» — на персонах.)

**Открытые вопросы (решения владельца — не зафиксированы):**
1. Поле `due_date` (дата) vs `due` (дата+время)?
2. Источник: **AI извлекает сам** (промпт: «найти deadline») или явное поле в UI / оба?
3. Напоминания: механизм и сроки, через что (WS, пометка задачи) — на фундаменте истории статусов (в0.23 шаг 2).
4. SLA: «в работе — ответить до…», по какому признаку персоны (is_internal?) — решение после шага 6.
5. В UI: колонка в таблице / бейдж / сортировка?

**Решение:** НЕ ЗАФИКСИРОВАНО (до решения — только обсуждение, код НЕ начинать).

**План шага:**
*После решения:* 1. schema/миграции (due_date на tasks + история) 2. API (GET/PUT, bulk?) 3. AI (извлечение + `DueDate` в вердикт) 4. UI (бейдж/колонка, сортировка) 5. SLA/напоминания (отдельная цель, если решим) 6. тесты + CHANGELOG + закрыть по чек-листу

**Чек-лист закрытия шага:** (см. CONTRIBUTING.md §Методология)

### [ ] Шаг 8 — «Сегодня» / спринты: что отвечать сегодня (v0.24+) — РЕЖИМ B

**Цель.** Ответ на вопрос владельца «что отвечать сегодня»: приоритизация (due из шага 7, приоритет AI, возраст, «новый»/«ожидание»).

**Открытые вопросы:** 1. Что считаем «сегодня»: due_date = сегодня + просроченные? (зависит от шага 7) 2. «Спринт» терминологически (как в Plane) vs обычный список? 3. Дашборд-вид: отдельная вкладка / сортировка-фильтры / «моё»?

**Решение:** не зафиксировано (зависит от шага 7).

**План шага:**
*После решения:* 1. UI (вкладка «Сегодня» / сортировка) 2. API (сортировка/фильтр) 3. тесты

## Что НЕ делаем (границы, чтобы не предлагать повторно)
- «Починить storage» отдельным планом — НЕ делаем (решение владельца; баги legacy только отдельными шагами/багами)
- Переписывать legacy AI-саммари (`author='user'`, `ai_processed=-1`) — НЕ делаем; новые вердикты — только по шагу 5
- Дубли письма (dedup) в цепочке — НЕ делаем (решение владельца: дубли ок)
- Plane как бэкенд задач — НЕ делаем (удалён в v0.22, ADR-0001)
- v1.0.0 сейчас — НЕ делаем (v1.0.0 далеко — v0.x-циклы, см. v0.23-блок)

## Выпущено (индекс, одна строка на версию)
- **v0.24 (2026-09-15)** — в разработке (текущий план; шаги 6–8).
- **v0.23.0 (2026-09-15)** — шаги 0–5: ws · вложения (extraction) · история статусов · MD-карта · bulk · вердикты (только саммари) → `archive/PLAN.v0.23.md`, CHANGELOG [0.23.0]
- **v0.22.0 (2026-08-30)** — проекты/модули, Plane-срез, outbound, темы → `archive/PLAN.v0.22-projects-modules.md`, CHANGELOG
- **v0.21.0 / v20–v17** — helpdesk + AI → `archive/PLAN.v17-21.md`; **v0–16** (Plane-based) → `archive/PLAN.v0-plane-era.md`

## Методология
→ `CONTRIBUTING.md` раздел «Методология разработки» (режимы A/B/C, уровни «шаг»/«версия», чек-листы СТАРТА/ЗАКРЫТИЯ/ВЫПУСКА, стоп-гейт, нумерация, передача между сессиями).
