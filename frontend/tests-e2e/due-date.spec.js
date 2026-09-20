// due-date.spec.js — E2E (v0.25, шаг 7c): подтверждение AI-срока в UI.
// AI-предложение сеется API-ключом due_ai_set (тестовый сид, без LLM);
// решения — КЛИКАМИ в UI карточки задачи (data-testid due-ai-*).
// А. Принять: due_ai_pending → due_date = ai_due_date, due_source='ai',
//    pending снят, AI-строка исчезла.
// Б. Отклонить: pending снят, срок НЕ устанавливается, ai_due_date
//    сохраняется (база ошибок AI).
// В. Изменить: «Изменить» предзаполняет DatePicker значением AI; новая дата
//    = ручное решение → due_source='manual', pending снят.
// Параллель-безопасно: unique-задачи (stamp), как в task-history-smoke.
import { test, expect } from '@playwright/test'

const AUTH_SCHEME = String.fromCharCode(66, 105, 97, 114, 101, 114)

async function login(request) {
  const r = await request.post('/api/auth/login', { data: { username: 'admin', password: 'admin' } })
  expect(r.status()).toBe(200)
  return (await r.json()).token
}

async function api(request, method, path, body) {
  const token = await login(request)
  const opts = { method, headers: { Authorization: `${AUTH_SCHEME} ${token}` } }
  if (body) opts.data = body
  const r = await request.fetch(path, opts)
  return { status: r.status(), json: await r.json().catch(() => null) }
}

// Unique-задача + посев AI-предложения срока (тестовый сид due_ai_set).
async function seedTask(request, aiDue) {
  const stamp = Date.now()
  const t = await api(request, 'POST', '/api/tasks', { title: `e2e-due7c-${stamp}`, project: 'Входящие' })
  expect(t.status, `POST /api/tasks → ${t.status}`).toBeLessThan(300)
  const task = t.json.task || t.json
  const seed = await api(request, 'PATCH', `/api/tasks/${task.id}`, { due_ai_set: aiDue })
  expect(seed.status, `PATCH due_ai_set → ${seed.status}`).toBe(200)
  const seeded = seed.json.task || seed.json
  expect(seeded.due_ai_pending).toBeTruthy()
  expect(seeded.ai_due_date).toBe(aiDue)
  return task
}

// Открыть карточку задачи и дождаться AI-строки (pending).
async function openPending(page, taskId) {
  await page.goto(`/tasks/${taskId}`)
  await page.waitForSelector('.task-detail', { state: 'visible', timeout: 15000 })
  const aiRow = page.locator('[data-testid="due-ai-row"]')
  await expect(aiRow, 'AI-строка с предложением срока').toBeVisible({ timeout: 10000 })
  return aiRow
}

// ── А: Принять ─────────────────────────────────────────────────────────────
test('7c-A: «Принять» — срок = AI-значение, source=ai, pending снят', async ({ page, request }) => {
  const AI = '2027-03-31'
  const task = await seedTask(request, AI)

  await openPending(page, task.id)
  await expect(page.locator('[data-testid="due-ai-suggestion"]')).toContainText(AI)

  const respPromise = page.waitForResponse(
    (r) => r.request().method() === 'PATCH' && r.url().includes(`/api/tasks/${task.id}`),
    { timeout: 15000 }
  )
  await page.locator('[data-testid="due-ai-accept"]').click()
  const resp = await respPromise
  expect(resp.status(), 'PATCH accept → 200').toBe(200)

  // pending снят → AI-строка и кнопки исчезли.
  await expect(page.locator('[data-testid="due-ai-row"]')).toBeHidden({ timeout: 5000 })
  // Источник — AI.
  await expect(page.locator('[data-testid="due-source-ai"]')).toBeVisible()

  // API-подтверждение итога.
  const g = await api(request, 'GET', `/api/tasks/${task.id}`)
  expect(g.status).toBe(200)
  const t = g.json.task || g.json
  expect(t.due_date).toBe(AI)
  expect(t.due_source).toBe('ai')
  expect(t.due_ai_pending).toBeFalsy()
  expect(t.ai_due_date).toBe(AI)
})

// ── Б: Отклонить ───────────────────────────────────────────────────────────
test('7c-B: «Отклонить» — срок не ставится, AI-значение сохраняется', async ({ page, request }) => {
  const AI = '2027-04-15'
  const task = await seedTask(request, AI)

  await openPending(page, task.id)
  const respPromise = page.waitForResponse(
    (r) => r.request().method() === 'PATCH' && r.url().includes(`/api/tasks/${task.id}`),
    { timeout: 15000 }
  )
  await page.locator('[data-testid="due-ai-reject"]').click()
  const resp = await respPromise
  expect(resp.status(), 'PATCH reject → 200').toBe(200)

  await expect(page.locator('[data-testid="due-ai-row"]')).toBeHidden({ timeout: 5000 })

  const g = await api(request, 'GET', `/api/tasks/${task.id}`)
  expect(g.status).toBe(200)
  const t = g.json.task || g.json
  expect(t.due_ai_pending).toBeFalsy()
  expect(!t.due_date, 'отклонённое предложение не должно ставить срок').toBeTruthy()
  expect(t.ai_due_date, 'ai_due_date остаётся (база ошибок AI)').toBe(AI)
})

// ── В: Изменить ────────────────────────────────────────────────────────────
test('7c-C: «Изменить» — AI-значение предзаполнено, новая дата = manual', async ({ page, request }) => {
  const AI = '2027-05-20'
  const MANUAL = '2027-06-30'
  const task = await seedTask(request, AI)

  await openPending(page, task.id)
  await page.locator('[data-testid="due-ai-edit"]').click()

  // «Изменить» предзаполнило DatePicker значением AI (dd.mm.yyyy).
  const input = page.locator('.due-field input').first()
  const aiText = '20.05.2027'
  await expect
    .poll(async () => (await input.inputValue()), {
      timeout: 5000,
      message: `DatePicker не предзаполнен ${aiText} (факт: ${(await input.inputValue())})`
    })
    .toBe(aiText)

  // Подбираем СВОЮ дату: заменяем день в поле (dd.mm.yy → 30.06.2027).
  // waitForResponse ДО действия — change-хендлер сработает на blur.
  const respPromise = page.waitForResponse(
    (r) => r.request().method() === 'PATCH' && r.url().includes(`/api/tasks/${task.id}`),
    { timeout: 15000 }
  )
  await input.fill('30.06.2027')
  await input.blur()
  const resp = await respPromise
  expect(resp.status(), 'PATCH manual-due → 200').toBe(200)

  // pending снят, источник — вручную.
  await expect(page.locator('[data-testid="due-ai-row"]')).toBeHidden({ timeout: 5000 })
  await expect(page.locator('[data-testid="due-source-manual"]')).toBeVisible()

  const g = await api(request, 'GET', `/api/tasks/${task.id}`)
  expect(g.status).toBe(200)
  const t = g.json.task || g.json
  expect(t.due_date).toBe(MANUAL)
  expect(t.due_source).toBe('manual')
  expect(t.due_ai_pending).toBeFalsy()
  expect(t.ai_due_date).toBe(AI) // AI-значение не затёрто решением человека
})

// ── D: Ручная дата КЛИКОМ по календарю (регресс: в v5 DatePicker нет «change») ─
test('7c-D: клик по календарю в карточке задачи сохраняет срок (manual)', async ({ page, request }) => {
  const task = await seedTask(request, '2027-07-01')

  await openPending(page, task.id)

  // Открываем календарь кликом по полю; оверлей — в портале вне .due-field.
  const input = page.locator('.due-field input').first()
  await input.click()
  const todayCell = page.locator('td[data-p-today="true"]').first()
  await expect(todayCell, 'ячейка «сегодня» в календаре').toBeVisible({ timeout: 5000 })

  const respPromise = page.waitForResponse(
    (r) => r.request().method() === 'PATCH' && r.url().includes(`/api/tasks/${task.id}`),
    { timeout: 15000 }
  )
  await todayCell.click()
  const resp = await respPromise
  expect(resp.status(), 'PATCH calendar-click → 200').toBe(200)

  // «Сегодня» = каноническая дата в формате сервера (YYYY-MM-DD).
  const today = await page.evaluate(() => {
    const d = new Date()
    const p = (n) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
  })

  const g = await api(request, 'GET', `/api/tasks/${task.id}`)
  expect(g.status).toBe(200)
  const t = g.json.task || g.json
  expect(t.due_date, 'дата, выбранная кликом, сохранена').toBe(today)
  expect(t.due_source).toBe('manual')
})
