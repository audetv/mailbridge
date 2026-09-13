// task-history-smoke.spec.js — Смоук (v0.23, шаг 2): смена статуса из UI
// оставляет строку в task_status_history (by=имя), GET /history отдаёт хронологию.
// Idempotent: свой task на каждом прогоне (unique subject) — не зависит от
// seed/состав dev-БД и не влезает в UNIQUE(tasks.message_id) чужих inbox-ссылок.
import { test, expect } from '@playwright/test'

async function login(request) {
  const r = await request.post('/api/auth/login', { data: { username: 'admin', password: 'admin' } })
  expect(r.status()).toBe(200)
  return (await r.json()).token
}

test('status change from UI writes history row', async ({ page, request }) => {
  const token = await login(request)
  const auth = { Authorization: `Bearer ${token}` }

  // Чистая задача (без inbox-связи — unique message_id гарантирован;
  // project — «Входящие»-проект, создаётся сам бэкендом при отсутствии)
  const stamp = Date.now()
  const title = `e2e-history-smoke-${stamp}`
  const create = await request.post('/api/tasks', {
    headers: auth,
    data: { title, project: 'Входящие' },
  })
  expect(create.status(), `POST /api/tasks → ${create.status()}`).toBeLessThan(300)
  // Ответ — прямой task: {id, ...} (обёртки {task:{...}} нет)
  const createdJson = await create.json()
  const task = createdJson.task || createdJson
  expect(task.id).toBeTruthy()

  // UI: страница задачи → workflow-кнопка («В работу»/«Закрыть») → PATCH /api/tasks/{id}
  await page.goto(`/tasks/${task.id}`)
  await page.waitForSelector('.task-detail', { state: 'visible', timeout: 15000 })
  await page.waitForTimeout(300)
  const btnWork = page.getByRole('button', { name: 'В работу' }).first()
  const btnClose = page.getByRole('button', { name: 'Закрыть' }).first()
  const respPromise = page.waitForResponse(
    (r) => r.request().method() === 'PATCH' && r.url().includes(`/api/tasks/${task.id}`),
    { timeout: 15000 }
  )
  if ((await btnWork.count()) > 0 && await btnWork.isVisible()) {
    await btnWork.click()
    const expectedTo = 'in_progress'
    await assertHistory(request, auth, task.id, expectedTo)
  } else {
    expect(await btnClose.count(), 'workflow-кнопки не видны на странице задачи').toBeGreaterThan(0)
    await btnClose.click()
    const expectedTo = 'closed'
    await assertHistory(request, auth, task.id, expectedTo)
  }
  const resp = await respPromise
  expect(resp.status(), 'PATCH /api/tasks/{id} → 200').toBe(200)

  async function assertHistory(req, headers, id, expectedTo) {
    const h = await req.get(`/api/tasks/${id}/history`, { headers })
    expect(h.status()).toBe(200)
    const rows = (await h.json()).history
    expect(rows.length).toBeGreaterThanOrEqual(1)
    const last = rows[rows.length - 1]
    expect(last.to_status).toBe(expectedTo)
    expect(last.from_status).toBe('new')
    expect(last.by).toBe('admin')
    expect(last.task_id).toBe(id)
  }
})
