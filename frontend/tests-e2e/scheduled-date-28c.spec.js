// scheduled-date-28c.spec.js — E2E (v0.28, шаг 28c): «План» в UI карточки задачи.
// Сценарии владельца из режима B (контракт 28c):
//   А. due на неделю впереди → scheduled = сегодня (кalendar-клик) — due не меняется.
//   Б. due = сегодня → scheduled = завтра; due остаётся как есть
//      (формально просрочена завтра; поля независимы — контракты 28a/28b).
// Установка dates КЛИКОМ/записью в DatePicker; API-ключи: POST/PATCH (28a), GET.
import { test, expect } from '@playwright/test'

async function login(request) {
  const r = await request.post('/api/auth/login', { data: { username: 'admin', password: 'admin' } })
  expect(r.status()).toBe(200)
  return (await r.json()).token
}

const AUTH_SCHEME = String.fromCharCode(66, 105, 97, 114, 101, 114)

async function api(request, method, path, body) {
  const token = await login(request)
  const opts = { method, headers: { Authorization: `${AUTH_SCHEME} ${token}` } }
  if (body) opts.data = body
  const r = await request.fetch(path, opts)
  return { status: r.status(), json: await r.json().catch(() => null) }
}

const p = (n) => String(n).padStart(2, '0')
function canon(offsetDays) {
  const d = new Date()
  d.setDate(d.getDate() + offsetDays)
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}
/** канон → dd.mm.yyyy (текст DatePicker, dateFormat="dd.mm.yy"). */
function ddmyy(iso) {
  const [y, m, d] = iso.split('-')
  return `${d}.${m}.${y}`
}

async function openCard(page, taskId) {
  await page.goto(`/tasks/${taskId}`)
  await page.waitForSelector('.task-detail', { state: 'visible', timeout: 15000 })
  await expect(page.locator('[data-testid="scheduled-datepicker"]'), 'поле «План»').toBeVisible({ timeout: 10000 })
}

test('28c-A: due на неделю → scheduled = сегодня (клик календаря); due не трогается', async ({ page, request }) => {
  const DUE = canon(7)
  const stamp = Date.now()
  const t = await api(request, 'POST', '/api/tasks', { title: `e2e-28c-a-${stamp}`, project: 'Входящие' })
  expect(t.status, `POST /api/tasks → ${t.status}`).toBeLessThan(300)
  const created = t.json.task || t.json
  const seed = await api(request, 'PATCH', `/api/tasks/${created.id}`, { due_date: DUE })
  expect(seed.status).toBe(200)
  const task = seed.json.task || seed.json

  await openCard(page, task.id)

  // Клик по календарю: ячейка «сегодня».
  const input = page.locator('.scheduled-field input').first()
  await input.click()
  const todayCell = page.locator('td[data-p-today="true"]').first()
  await expect(todayCell).toBeVisible({ timeout: 5000 })

  const respPromise = page.waitForResponse(
    (r) => r.request().method() === 'PATCH' && r.url().includes(`/api/tasks/${task.id}`),
    { timeout: 15000 }
  )
  await todayCell.click()
  const resp = await respPromise
  expect(resp.status()).toBe(200)

  const g = await api(request, 'GET', `/api/tasks/${task.id}`)
  expect(g.status).toBe(200)
  const t2 = g.json.task || g.json
  expect(t2.scheduled_date, 'план = сегодня').toBe(canon(0))
  expect(t2.due_date, 'due не тронут').toBe(DUE)
})

test('28c-B: due = сегодня → scheduled = завтра; due остаётся как есть (независимые поля)', async ({ page, request }) => {
  const stamp = Date.now()
  const t = await api(request, 'POST', '/api/tasks', { title: `e2e-28c-b-${stamp}`, project: 'Входящие', due_date: canon(0) })
  expect(t.status).toBeLessThan(300)
  const task = t.json.task || t.json
  expect(task.due_date).toBe(canon(0))

  await openCard(page, task.id)

  // Датапикер: «завтра» записью (dd.mm.yy) + blur = сохранить (механика 7c-C).
  const input = page.locator('.scheduled-field input').first()
  const respPromise = page.waitForResponse(
    (r) => r.request().method() === 'PATCH' && r.url().includes(`/api/tasks/${task.id}`),
    { timeout: 15000 }
  )
  await input.fill(ddmyy(canon(1)))
  await input.blur()
  const resp = await respPromise
  expect(resp.status(), 'PATCH scheduled → 200').toBe(200)

  const g = await api(request, 'GET', `/api/tasks/${task.id}`)
  expect(g.status).toBe(200)
  const t2 = g.json.task || g.json
  expect(t2.scheduled_date, 'план = завтра').toBe(canon(1))
  expect(t2.due_date, 'due остаётся как есть (сегодня)').toBe(canon(0))
})
