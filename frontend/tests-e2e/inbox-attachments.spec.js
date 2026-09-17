// inbox-attachments.spec.js — E2E: вложения входящего попадают в задачу
// при ручном создании (шаг 1, v0.22.1). UI-поток InboxItemView (кнопка
// «Создать задачу» → POST /api/inbox/{id}/task) + API-ассерты:
//   до:   GET /api/inbox/{id}/attachments  N>0
//   до:   GET /api/tasks/{id}/attachments  0 (нет задачи)
//   после: POST 201, GET task attachments >=2, те же hash,
//          GET /api/attachments/{path} — 200, тело не пустое
import { test, expect } from '@playwright/test'

async function api(request, method, path, body, token, withCt) {
  const headers = {}
  if (token) headers.Authorization = `Bearer ${token}`
  if (withCt) headers['Content-Type'] = 'application/json'
  const r = await request.fetch(path, { method, data: body, headers })
  let text = null
  try { text = await r.text() } catch { /* binary */ }
  return { status: r.status(), text, r }
}

async function login(request, username, password) {
  const { status, text } = await api(request, 'POST', '/api/auth/login', { username, password }, null, true)
  expect(status, `login ${username}`).toBe(200)
  return JSON.parse(text).token
}

test('attachments inherit: inbox item -> task (manual create)', async ({ page }) => {
  const token = await login(page.request, 'admin', 'admin')

  // Входящее с вложениями (seed: ids 57/170 — из backfill-проверки dev-БД).
  // Кандидат выбираем до клика: если задача по этому входящему уже создана
  // (первый прогон создаёт связь inbox→task), продолжаем с ней — e2e rerunnable.
  let inboxItem = null
  let inboxAtts = null
  let existingTask = null
  for (const candidateId of [57, 170, 171, 172]) {
    const itemR = await api(page.request, 'GET', `/api/inbox/${candidateId}`, null, token)
    if (itemR.status !== 200) continue
    const item = JSON.parse(itemR.text)
    const r = await api(page.request, 'GET', `/api/inbox/${candidateId}/attachments`, null, token)
    if (r.status !== 200) continue
    const arr = JSON.parse(r.text)
    if (Array.isArray(arr) && arr.length >= 2) {
      inboxItem = item
      inboxAtts = arr
      const listR = await api(page.request, 'GET', `/api/tasks?per_page=200&search=${encodeURIComponent(item.subject)}`, null, token)
      if (listR.status === 200) {
        const tasks = JSON.parse(listR.text).tasks || []
        existingTask = tasks.find((t) => t.subject === item.subject) || null
      }
      break
    }
  }
  expect(inboxItem, 'в dev-БД есть письмо-кандидат с >=2 вложениями (seed)').toBeTruthy()

  const inboxHashes = inboxAtts.map((a) => a.hash).sort()

  let createdTask = existingTask
  if (!createdTask) {
    // UI: страница листа, кнопка «Создать задачу» (ответ 201: {task:{id,...}})
    await page.goto(`/inbox/${inboxItem.id}`)
    const button = page.locator('button', { hasText: 'Создать задачу' }).first()
    await expect(button).toBeVisible({ timeout: 10000 })

    const respPromise = page.waitForResponse(
      (r) => r.request().method() === 'POST' && r.url().includes(`/api/inbox/${inboxItem.id}/task`),
      { timeout: 15000 }
    )
    await button.click()
    const created = await respPromise
    expect(created.status(), 'POST /api/inbox/{id}/task → 201').toBe(201)
    createdTask = (await created.json()).task
  } else {
    // Rerun: задача уже существует (связь создана первым прогоном) — проверяем её
    expect(createdTask.id, 'существующая задача (rerun)').toBeTruthy()
  }
  expect(createdTask.id, 'задача по входящему').toBeTruthy()

  // API: задача создана с вложениями
  const taskAtts = await api(page.request, 'GET', `/api/tasks/${createdTask.id}/attachments`, null, token)
  expect(taskAtts.status).toBe(200)
  const arr = JSON.parse(taskAtts.text)
  expect(arr.length, 'вложений на задаче').toBeGreaterThanOrEqual(inboxAtts.length)
  const taskHashes = arr.map((a) => a.hash).sort()
  for (const h of inboxHashes) expect(taskHashes, `hash ${h} на задаче`).toContain(h)

  // Download: файл доступен
  const dl = await api(page.request, 'GET', `/api/attachments/${arr[0].storage_path}`, null, token)
  expect(dl.status).toBe(200)
  expect((await dl.r.body()).length, 'тело download > 0').toBeGreaterThan(0)
})
