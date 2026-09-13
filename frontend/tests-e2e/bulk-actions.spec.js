// bulk-actions.spec.js — Смоук (v0.23, шаг 4): выбор N задач в листе →
// «К статусу…» → PATCH /api/tasks {ids, changes:{status}} → 200 {count}
// + одна строка task_status_history на задачу (by=admin).
// Idempotent: свои задачи на каждом прогоне (unique subject).
import { test, expect } from '@playwright/test'

async function login(request) {
  const r = await request.post('/api/auth/login', { data: { username: 'admin', password: 'admin' } })
  expect(r.status()).toBe(200)
  return (await r.json()).token
}

test('bulk status change from UI applies to all selected', async ({ page, request }) => {
  const token = await login(request)
  const auth = { Authorization: ['Bearer', token].join(' ') }

  // Две свежие задачи (уходят в «Новые» — вкладка «Активные» видна).
  const stamp = Date.now()
  const ids = []
  for (const i of [1, 2]) {
    const create = await request.post('/api/tasks', {
      headers: auth,
      data: { title: `e2e-bulk-${i}-${stamp}`, project: 'Входящие' },
    })
    expect(create.status(), `POST /api/tasks → ${create.status()}`).toBeLessThan(300)
    const json = await create.json()
    const task = json.task || json
    ids.push(task.id)
  }

  // Лист задач → включаем чекбоксы обеих → панель bulk действий.
  await page.goto('/')
  await page.waitForSelector('[data-testid="task-table"]', { state: 'visible', timeout: 15000 })

  const rows = page.locator('tbody tr')
  await expect(rows).not.toHaveCount(0, 'пустой список задач')

  let picked = 0
  const count = await rows.count()
  for (let i = 0; i < count && picked < 2; i++) {
    const title = await rows.nth(i).innerText()
    if (title.includes(`e2e-bulk-`)) {
      await rows.nth(i).locator('input[type="checkbox"]').first().check()
      picked += 1
    }
  }
  expect(picked, 'найдены обе созданные задачи в листе').toBe(2)

  const panel = page.getByTestId('bulk-panel')
  await expect(panel).toBeVisible({ timeout: 5000 })

  // «К В работе» → Применить.
  await panel.getByTestId('bulk-status').selectOption('in_progress')
  const respPromise = page.waitForResponse(
    (r) => r.request().method() === 'PATCH' && r.url().endsWith('/api/tasks'),
    { timeout: 15000 }
  )
  await panel.getByTestId('bulk-apply-status').click()
  const resp = await respPromise
  expect(resp.status(), 'PATCH /api/tasks → 200').toBe(200)
  const body = await resp.json()
  expect(body.count).toBe(2)

  // История: обе задачи получили new → in_progress (by=admin).
  for (const id of ids) {
    const h = await request.get(`/api/tasks/${id}/history`, { headers: auth })
    expect(h.status()).toBe(200)
    const rows = (await h.json()).history
    const last = rows[rows.length - 1]
    expect(last.from_status).toBe('new')
    expect(last.to_status).toBe('in_progress')
    expect(last.by).toBe('admin')
  }
})
