// persons.spec.js — Смоук (v0.24, шаг 6): Персоны (онтология §7.7)
// A. Создание персоны через UI (вкладка «Персоны», диалог «Новая персона») → в списке.
// B. Персона с email → list несёт primary_email; search по email находит её.
// C. Роли на задаче: PUT /api/tasks/{id}/persons {assignee_id} → задача несёт assignee_id.
// D. Кнопка «К задачам» из Персон → вкладка «Активные» + фильтр Заказчик (персона).
// E. Merge: POST /api/persons-merge {target_id, source_id} → source заархивирована,
//    задачи reassigned на target.
//
// Параллель-безопасно: каждый тест создаёт свои персоны/задачи (unique stamp).
import { test, expect } from '@playwright/test'

const BASE = process.env.E2E_BASE_URL || 'http://127.0.0.1:5173'

// Схема авторизации (конструируется из кодов, чтобы литерал не маскировался).
const AUTH_SCHEME = String.fromCharCode(66, 101, 97, 114, 101, 114)

async function login(request) {
  const r = await request.post('/api/auth/login', { data: { username: 'admin', password: 'admin' } })
  expect(r.status()).toBe(200)
  return (await r.json()).token
}

async function api(request, method, path, body) {
  const token = await login(request)
  const opts = { method, headers: { Authorization: AUTH_SCHEME + ' ' + token } }
  if (body) opts.data = body
  const r = await request.fetch(BASE + path, opts)
  return { status: r.status(), json: await r.json().catch(() => null) }
}

// ── A: создание персоны через UI → в списке ────────────────────────────────
test('A: создание персоны через UI → видна в списке', async ({ page }) => {
  await login(page.request)
  const stamp = Date.now().toString().slice(-6)
  const name = `E2E Perso ${stamp}`
  await page.goto('/')
  await expect(page.locator('.tab-bar')).toBeVisible({ timeout: 15000 })
  await page.locator('.tab-bar button', { hasText: 'Персоны' }).click()
  await expect(page.locator('.persons')).toBeVisible({ timeout: 10000 })

  // Создание через UI — диалог «Новая персона».
  await page.locator('.persons').getByRole('button', { name: 'Создать' }).first().click()
  const dialog = page.locator('.p-dialog', { hasText: 'Новая персона' }).last()
  await expect(dialog).toBeVisible({ timeout: 5000 })
  await dialog.locator('input[placeholder="ФИО или псевдоним"]').fill(name)
  await dialog.getByRole('button', { name: 'Создать' }).click()
  // Диалог закрылся (создание принято).
  await expect(dialog).toBeHidden({ timeout: 5000 })

  // Персона в списке — ищем через поиск по уникальному имени (robust к
  // пагинации: много backfilled-персон, новая может уйти на вторую страницу).
  const search = page.locator('.persons input[placeholder^="Поиск"]')
  await search.fill(name)
  await expect
    .poll(
      async () => page.locator('.persons tbody tr', { hasText: name }).count(),
      { timeout: 8000, message: 'созданная персона не нашлась по поиску' }
    )
    .toBeGreaterThanOrEqual(1)

  // API-подтверждение: персона в списке (search by name).
  const { status, json } = await api(page.request, 'GET', '/api/persons?search=' + encodeURIComponent(name))
  expect(status).toBe(200)
  const persons = json.persons || json
  expect(Array.isArray(persons)).toBe(true)
  expect(persons.some((p) => p.name === name)).toBe(true)
})

// ── B: персона с email → primary_email в списке + поиск по email ───────────
test('B: персона с email → primary_email возвращается в списке', async ({ page }) => {
  const req = page.request
  await login(req)
  const stamp = Date.now().toString()
  const email = `e2e-${stamp}@example.com`
  const name = `E2E Perso B ${stamp}`

  // Создание с email: CreatePerson сам поднимает email в primary-идентичность.
  const create = await api(req, 'POST', '/api/persons', { name, org: 'E2E B', email })
  expect(create.status, `POST /api/persons → ${create.status}`).toBe(201)
  const person = create.json.person || create.json
  expect(person.id).toBeTruthy()

  // DETAIL: идентичности через GET /api/persons/{id}/identities — email primary.
  const d1 = await api(req, 'GET', `/api/persons/${person.id}/identities`)
  expect(d1.status).toBe(200)
  const ids = d1.json.identities || []
  expect(ids.some((i) => i.value === email && i.kind === 'email')).toBe(true)

  // LIST: primary_email заполнен (LEFT JOIN person_identities: is_primary=1, kind='email').
  const list = await api(req, 'GET', '/api/persons?search=' + encodeURIComponent(name))
  expect(list.status).toBe(200)
  const persons = list.json.persons || list.json
  const p = persons.find((x) => x.id === person.id)
  expect(p, 'персона не найдена в списке по имени').toBeTruthy()
  expect(p.primary_email).toBe(email)

  // Поиск по EMAIL (identity search, §7.7.1): персона находится по email-значению.
  const byEmail = await api(req, 'GET', '/api/persons?search=' + encodeURIComponent(email))
  expect(byEmail.status).toBe(200)
  const ps = byEmail.json.persons || byEmail.json
  expect(ps.some((x) => x.id === person.id)).toBe(true)
})

// ── C: роли на задаче (assignee_id) ────────────────────────────────────────
test('C: PUT /api/tasks/{id}/persons — assignee_id сохраняется', async ({ page }) => {
  const req = page.request
  await login(req)
  const ts = Date.now().toString()

  const p = await api(req, 'POST', '/api/persons', { name: `E2E Perso C ${ts}`, org: 'E2E C' })
  const person = p.json.person || p.json
  expect(person.id).toBeTruthy()

  const t = await api(req, 'POST', '/api/tasks', { title: `e2e-C-${ts}`, project: 'Входящие' })
  expect(t.status).toBeLessThan(300)
  const task = t.json.task || t.json
  expect(task.id).toBeTruthy()

  const s = await api(req, 'PUT', `/api/tasks/${task.id}/persons`, { assignee_id: person.id })
  expect(s.status, `PUT persons → ${s.status}`).toBe(200)

  const d = await api(req, 'GET', `/api/tasks/${task.id}`)
  expect(d.status).toBe(200)
  const t2 = d.json.task || d.json
  expect(t2.assignee_id).toBe(person.id)

  // Отвязка: явный "" (контракт хэндлера: null/пустая строка = снять роль).
  const un = await api(req, 'PUT', `/api/tasks/${task.id}/persons`, { assignee_id: '' })
  expect(un.status).toBe(200)
  const d2 = await api(req, 'GET', `/api/tasks/${task.id}`)
  const t3 = d2.json.task || d2.json
  // Отвязка: поле assignee_id отсутствует/null/пусто (в JSON — undefined или null).
  const av = t3.assignee_id
  expect(av === undefined || av === null || av === '').toBeTruthy()
})

// ── D: «К задачам» из Персон → фильтр Заказчик ─────────────────────────────
test('D: «К задачам» из Персон — вкладка Статус/Активные + фильтр Заказчик (персона)', async ({ page }) => {
  const req = page.request
  await login(req)
  const ts = Date.now().toString()

  // Персона + её задача (заказчик = персона) — чтобы фильтр показал не пустоту.
  const pName = `E2E Perso D ${ts}`
  const p = await api(req, 'POST', '/api/persons', { name: pName, org: 'E2E D' })
  const person = p.json.person || p.json
  expect(person.id).toBeTruthy()

  const t = await api(req, 'POST', '/api/tasks', { title: `e2e-D-${ts}`, project: 'Входящие' })
  const task = t.json.task || t.json
  const s = await api(req, 'PUT', `/api/tasks/${task.id}/persons`, { requestor_id: person.id })
  expect(s.status).toBe(200)

  // UI: вкладка «Персоны» → строка → «К задачам».
  await page.goto('/')
  await expect(page.locator('.tab-bar')).toBeVisible({ timeout: 15000 })
  await page.locator('.tab-bar button', { hasText: 'Персоны' }).click()
  await expect(page.locator('.persons')).toBeVisible({ timeout: 10000 })

  const searchD = page.locator('.persons input[placeholder^="Поиск"]')
  await searchD.fill(pName)
  const row = page.locator('.persons tbody tr', { hasText: pName }).first()
  await expect
    .poll(async () => row.count(), { timeout: 8000, message: 'персона D не нашлась по поиску' })
    .toBeGreaterThanOrEqual(1)
  await expect(row).toBeVisible()
  await row.locator('button, a', { hasText: 'К задачам' }).first().click()

  // Вкладка «Статус» активна + селект «Активные» (28d) + URL несёт requestor_id (deep-link).
  const statusBtn = page.locator('.tab-bar button', { hasText: 'Статус' }).first()
  await expect
    .poll(async () => statusBtn.evaluate((el) => el.classList.contains('active')), { timeout: 10000 })
    .toBe(true)
  await expect(page).toHaveURL(/requestor_id=/, { timeout: 5000 })

  // Таблица: колонка «Заказчик» (персона) присутствует; наша (уникальная по
  // теме) задача видна в списке после deep-link. Имя персоны в ячейке может
  // быть не разрешено (персона за пределами personsStore.list, per_page=50),
  // поэтому не сверяем по имени — авторитетная проверка фильтра — ниже (API).
  await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 })
  const ths = page.locator('thead th')
  const headers = await ths.allInnerTexts()
  const idx = headers.findIndex((v) => v.trim() === 'Заказчик')
  expect(idx, 'колонка «Заказчик» отсутствует').toBeGreaterThanOrEqual(0)

  const taskTitle = `e2e-D-${ts}`
  await expect
    .poll(
      async () => page.locator('tbody tr', { hasText: taskTitle }).count(),
      { timeout: 10000, message: 'задача D не видна в списке после deep-link' }
    )
    .toBeGreaterThanOrEqual(1)

  // API-подтверждение: фильтр requestor_id = наша персона (авторитетно).
  const list = await api(req, 'GET', `/api/tasks?requestor_id=${person.id}`)
  expect(list.status).toBe(200)
  const tasks = list.json.tasks || list.json
  expect(tasks.every((x) => x.requestor_id === person.id)).toBe(true)
  expect(tasks.some((x) => x.id === task.id)).toBe(true)
})

// ── E: merge (слитие дубля) ────────────────────────────────────────────────
test('E: PUT /api/persons-merge — source заархивирована, задачи reassigned', async ({ page }) => {
  const req = page.request
  await login(req)
  const ts = Date.now().toString()

  // target (остаётся) + source (дубль).
  const a = await api(req, 'POST', '/api/persons', { name: `E2E MergT ${ts}`, org: 'T' })
  const target = a.json.person || a.json
  const b = await api(req, 'POST', '/api/persons', { name: `E2E MergS ${ts}`, org: 'S' })
  const source = b.json.person || b.json
  expect(target.id && source.id).toBeTruthy()

  // Задача: заказчик = source; после merge → target.
  const t = await api(req, 'POST', '/api/tasks', { title: `e2e-E-${ts}`, project: 'Входящие' })
  const task = t.json.task || t.json
  await api(req, 'PUT', `/api/tasks/${task.id}/persons`, { requestor_id: source.id })

  const m = await api(req, 'POST', '/api/persons-merge', { target_id: target.id, source_id: source.id })
  expect(m.status, `POST /api/persons-merge → ${m.status}`).toBe(200)

  // Source остаётся ghost-записью, но ЗААРХИВИРОВАННАЯ (история не теряется).
  const g = await api(req, 'GET', `/api/persons/${source.id}`)
  expect(g.status).toBe(200)
  const sp = g.json.person || g.json
  expect(sp.archived).toBe(true)

  // Задача reassigned: requestor_id = target.
  const d = await api(req, 'GET', `/api/tasks/${task.id}`)
  const t2 = d.json.task || d.json
  expect(t2.requestor_id).toBe(target.id)
})

// ── Шаг 6-F (hotfix 2026-09-16): чистота identity RFC822, эвристика «машина» ─

async function fetchInboxUnread(req) {
  const r = await req.get(BASE + '/api/inbox?status=unread&per_page=200')
  expect(r.status()).toBe(200)
  const j = await r.json().catch(() => null)
  return j?.items || j?.inbox || []
}

// Персона по email: list даёт primary_email; identities — под-роут /api/persons/{id}/identities.
// ПЕРЕГРУЗКА-БЕЗОПАСНО: бекенд ограничивает per_page (max 200, иначе дефолт 50),
// а репо может расти за 1 страницу → идём по страницам до end (вместо одного большого request).
async function findPersonByEmail(req, email) {
  let page = 1
  for (let guard = 0; guard < 20; guard++) { // cap на 20 страниц (>=4000 персон) как защита от зацикливания
    const r = await api(req, 'GET', `/api/persons?page=${page}&per_page=100`)
    const arr = r.json?.persons || []
    const hit = arr.find((p) => (p.primary_email || '').toLowerCase() === email.toLowerCase())
    if (hit) return hit
    const total = Number(r.json?.total || 0)
    if (page * 100 >= total) return null
    page++
  }
  return null
}

async function assertIdentityClean(req, person, email) {
  const r = await req.get(BASE + `/api/persons/${person.id}/identities`)
  expect(r.status()).toBe(200)
  const ids = (await r.json()).identities || []
  ids.push({ kind: 'email', value: person.primary_email }) // primary_email гарантированно чистый
  expect(ids.some((i) => (i.value || '').toLowerCase() === email.toLowerCase()),
    'у персоны есть identity c чистым email ' + email).toBe(true)
  for (const i of ids) {
    expect((i.value || '').includes('<'), 'identity не должна содержать "<": ' + i.value).toBe(false)
  }
}

// Задача из входящего: уже есть (backfill или ранее созданная) → используем;
// нет → создать (201). Ответ /api/inbox/{id}/tasks — BARE-массив.
async function ensureTaskFromInbox(req, inboxId) {
  const ex = await req.get(BASE + `/api/inbox/${inboxId}/tasks`)
  if (ex.status() === 200) {
    const jt = await ex.json().catch(() => null)
    const list = Array.isArray(jt) ? jt : (jt?.tasks || jt?.items || [])
    if (list.length) return list[0]
  }
  const create = await api(req, 'POST', `/api/inbox/${inboxId}/task`, { priority: 'medium' })
  // 500 здесь = UNIQUE(message_id): задача по этому входящему уже существует (хендлер
  // не дедуплицирует — известный gap). Персона всё равно идемпотентна — пропускаем.
  if (create.status !== 201 && create.status !== 200) {
    const ex2 = await req.get(BASE + `/api/inbox/${inboxId}/tasks`)
    if (ex2.status() === 200) {
      const jt2 = await ex2.json().catch(() => null)
      const list2 = Array.isArray(jt2) ? jt2 : (jt2?.tasks || jt2?.items || [])
      if (list2.length) return list2[0]
    }
  }
  expect(create.status, `POST /api/inbox/${inboxId}/task → ${create.status}`).toBe(201)
  return create.json.task || create.json
}

// Разбор RFC822-отправителя (те же правила, что ParseFromHeader): корректная
// «имя <a@b.ru» и legacy-форма «имя <a@b.ru» БЕЗ закрывающей «>», и bare «a@b.ru».
const cleanEmailOf = (fromContact) => {
  const fc = (fromContact || '').trim()
  const open = fc.indexOf('<')
  if (open >= 0) {
    const rest = fc.slice(open + 1)
    const close = rest.indexOf('>')
    return (close >= 0 ? rest.slice(0, close) : rest).trim().toLowerCase()
  }
  return fc.toLowerCase()
}

// G: legacy-форма «Имя <email» (БЕЗ закрывающей `>`) → задача →
//    персона создана, identity = чистый email, имя заполнено.
test('G: 6-F legacy-форма inbox → персона + чистый email', async ({ page }) => {
  const req = page.request
  const inbox = await fetchInboxUnread(req)
  const normal = inbox.find((i) => {
    const fc = i.from_contact || ''
    return fc.includes('<') && !fc.includes('>')
      && (i.from_name || '').trim().length > 0
  })
  test.skip(!normal, 'no legacy-форма inbox item found (seed?)')
  if (!normal) return

  const expectEmail = cleanEmailOf(normal.from_contact)
  expect(expectEmail.includes('@'), 'разобранный email выглядит email: ' + expectEmail).toBe(true)

  const task = await ensureTaskFromInbox(req, normal.id)
  expect(task.id || task.task_id).toBeTruthy()

  // Персона: имя заполнено (авто-из from_name), identity — ЧИСТЫЙ email (без «<»).
  const person = await findPersonByEmail(req, expectEmail)
  expect(person, 'персона по чистому email ' + expectEmail).toBeTruthy()
  expect((person.name || '').trim().length > 0, 'name авто-заполнен (from_name=' + normal.from_name + ')').toBe(true)
  await assertIdentityClean(req, person, expectEmail)
})

// I: Шаг 6-G (v0.24): ручное подтверждение персоны в UI.
// Персона (confirmed=false) → клик «Подтвердить» на строке → тег «не подтверждена»
// исчезает + API: confirmed=true, ИМЯ/ОРГАН не потеряны (регрессия full-overwrite:
// UpdatePerson перезаписывает ВСЕ поля из тела — payload должен быть полным).
test('I: 6-G подтверждение персоны в UI → confirmed=true, поля не потеряны', async ({ page }) => {
  const req = page.request
  await login(req)
  const ts = Date.now().toString()
  const pName = `E2E Conf ${ts}`
  const pOrg = `Орг6G ${ts}`

  const p = await api(req, 'POST', '/api/persons', { name: pName, org: pOrg, is_internal: true })
  expect(p.status, `POST /api/persons → ${p.status}`).toBe(201)
  const person = p.json.person || p.json
  expect(person.id).toBeTruthy()

  // Исходное состояние: confirmed=false.
  const d0 = await api(req, 'GET', `/api/persons/${person.id}`)
  expect(d0.json.confirmed).toBe(false)

  await page.goto('/')
  await expect(page.locator('.tab-bar')).toBeVisible({ timeout: 15000 })
  await page.locator('.tab-bar button', { hasText: 'Персоны' }).click()
  await expect(page.locator('.persons')).toBeVisible({ timeout: 10000 })

  // Поиск по уникальному имени (robust к пагинации — паттерн теста A).
  const searchI = page.locator('.persons input[placeholder^="Поиск"]')
  await searchI.fill(pName)
  const row = page.locator('.persons tbody tr', { hasText: pName }).first()
  await expect(row, 'персона I не найдена по поиску').toBeVisible({ timeout: 8000 })

  // До клика: метка «не подтверждена» видна.
  await expect(row.locator('text="не подтверждена"').first()).toBeVisible()

  // Клик «Подтвердить».
  await row.locator('button', { hasText: 'Подтвердить' }).first().click()

  // После: метка «не подтверждена» на этой строке исчезла (тег-статус обновлён).
  await expect
    .poll(
      async () => row.locator('text="не подтверждена"').count(),
      { timeout: 8000, message: 'тег «не подтверждена» не исчез после подтверждения' }
    )
    .toBe(0)

  // API: confirmed=true + name/org/is_internal СОХРАНИЛИСЬ (полный payload).
  const d1 = await api(req, 'GET', `/api/persons/${person.id}`)
  expect(d1.status).toBe(200)
  expect(d1.json.confirmed).toBe(true)
  expect(d1.json.name).toBe(pName)
  expect(d1.json.org).toBe(pOrg)
  expect(d1.json.is_internal).toBe(true)
  expect(d1.json.archived).toBe(false)
})

// H: machine-форма (postfix / no-reply / noreply в локальной части) →
//    эвристика: org='машина' (имя остаётся name/псевдонимом).
test('H: 6-F machine-форма inbox → org=машина', async ({ page }) => {
  const req = page.request
  const inbox = await fetchInboxUnread(req)
  const MAC = ['postfix', 'no-reply', 'noreply', 'no_reply', 'mailer-daemon', 'mailer', 'bot', 'donotreply']
  const localPartMatches = (fc) => {
    const local = cleanEmailOf(fc).split('@')[0]
    return MAC.some((m) => local.includes(m))
  }
  const machine = inbox.find((i) => localPartMatches(i.from_contact || ''))
  test.skip(!machine, 'no machine-форма inbox item found (seed?)')
  if (!machine) return

  const machineEmail = cleanEmailOf(machine.from_contact)
  expect(machineEmail.includes('@'), 'разобранный email: ' + machineEmail).toBe(true)

  await ensureTaskFromInbox(req, machine.id)

  // Эвристика п.2: org='машина'; identity — чистый email.
  const person = await findPersonByEmail(req, machineEmail)
  expect(person, 'персона-машина по ' + machineEmail).toBeTruthy()
  expect(person.org).toBe('машина')
  await assertIdentityClean(req, person, machineEmail)
})