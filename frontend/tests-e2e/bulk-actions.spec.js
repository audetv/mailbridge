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
      data: {
        title: `e2e-bulk-${stamp}-${i}`,
        // Поиск по unique stamp (см. ниже) находит обе задачи при любой
        // сортировке — срок не нужен (задачу видно по search LIKE %stamp%).
        project: 'Входящие',
      },
    })
    expect(create.status(), `POST /api/tasks → ${create.status()}`).toBeLessThan(300)
    const json = await create.json()
    const task = json.task || json
    ids.push(task.id)
  }

  // Лист задач → включаем чекбоксы обеих → панель bulk действий.
  await page.goto('/')
  await page.waitForSelector('[data-testid="task-table"]', { state: 'visible', timeout: 15000 })

  // 7a (v0.25.0): дефолтная сортировка «по срокам». Фильтруем поиск по
  // уникальному stamp ЭТОГО прогона — иначе широкое 'e2e-bulk-' попадает на
  // stale e2e-bulk-* прошлых прогонов (укопанные в dev-БД), и при sort=due
  // (равные сроки = id ASC = самые старые вверху) задачи этого прогона,
  // созданные последними, уходят за per_page=50 → picked == 0.
  const search = page.locator('.search-input')
  await search.fill(`e2e-bulk-${stamp}`)
  // Ждём, пока в листе появятся ОБЕ задачи этого прогона (по их уникальным
  // именам). Лист refetch-ится дебаунсом поиска → стабильность считывания
  // гарантируем ожиданием по именам, а не по rows.count() (гонка):
  //   rows.nth(i) на несуществующий i зависает на innerText.
  await expect(
    page.locator('tbody tr', { hasText: `e2e-bulk-${stamp}-1` }).first()
  ).toBeVisible({ timeout: 15000 })
  await expect(
    page.locator('tbody tr', { hasText: `e2e-bulk-${stamp}-2` }).first()
  ).toBeVisible({ timeout: 15000 })

  // PrimeVue .p-datatable-mask (loading overlay, full-screen) виден, пока идёт
  // асинхронный refetch по поиску — его клики перехватываются. Ждём, пока
  // маска скроется (opacity 0 / visibility hidden), чтобы чекбокс кликался.
  const maskGone = async (timeoutMs = 10000) => {
    const t0 = Date.now()
    while (Date.now() - t0 < timeoutMs) {
      const hidden = await page
        .locator('.p-datatable-mask')
        .first()
        .evaluate((el) => {
          const cs = getComputedStyle(el)
          return cs.visibility === 'hidden' || cs.opacity === '0'
        })
        .catch(() => true)
      if (hidden) return
      await page.waitForTimeout(50)
    }
    throw new Error('p-datatable-mask не скрылся за ' + timeoutMs + 'ms')
  }
  await maskGone()

  let picked = 0
  for (const n of [1, 2]) {
    const row = page.locator('tbody tr', { hasText: `e2e-bulk-${stamp}-${n}` }).first()
    await row.locator('input[type="checkbox"]').first().check()
    picked += 1
  }
  expect(picked, 'найдены обе созданные задачи (этот прогон) в листе').toBe(2)

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
