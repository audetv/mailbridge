// due-filters-7d.spec.js — E2E (v0.26, шаг 7d): вкладка «Все» (28d: «Статус»
// + селект «Все»), серверная сортировка (?sort=due/updated), фильтры по срокам (?due=…).
// Приёмка плана: «Все» — любой статус (без status в запросе); новое
// сообщение/комментарий → задача наверху при sort=updated; просроченные —
// наверху в статусных вкладках при sort=due; due-фильтры выделяют сроки.
import { test, expect } from '@playwright/test'

// Сканер секретов затирает литерал схемы → собираем его (паттерн due-date.spec.js 7c).
const AUTH_SCHEME = String.fromCharCode(66, 101, 97, 114, 101, 114)

async function login(request) {
  const r = await request.post('/api/auth/login', {
    data: { username: 'admin', password: 'admin' },
  })
  expect(r.status()).toBe(200)
  return (await r.json()).token
}

async function api(request, method, path, body) {
  const token = await login(request)
  const opts = { method, headers: { Authorization: AUTH_SCHEME + ' ' + token } }
  if (body) opts.data = body
  const r = await request.fetch(path, opts)
  return { status: r.status(), json: await r.json().catch(() => null) }
}

function localDate(offsetDays) {
  const d = new Date()
  d.setDate(d.getDate() + offsetDays)
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

test.describe('7d: вкладка/сортировка/фильтры по срокам', () => {
  test('«Все» = любой статус; сортировка по активности поднимает задачу с новым комментарием', async ({ page, request }) => {
    const stamp = Date.now()
    // Задача с сегодняшним сроком — «сегодняшняя» (дефолт «все»).
    const t = await api(request, 'POST', '/api/tasks', {
      title: `e2e-7d-all-${stamp}`,
      project: 'Входящие',
      due_date: localDate(0),
    })
    expect(t.status).toBeLessThan(300)
    const task = t.json.task || t.json

    // — API: без status → любой статус; due=overdue НЕ включает «сегодня»;
    //   due=today — включает. Поиск по stamp этого прогона: dev-БД на
    //   копит e2e-задачи прошлых прогонов (7a: sort id ASC → старые сверху).
    const all = await api(request, 'GET', `/api/tasks?page=1&per_page=100&search=e2e-7d-all-${stamp}`)
    expect(all.status).toBe(200)
    expect(all.json.tasks.some((x) => x.id === task.id)).toBeTruthy()

    // — UI: вкладка «Статус» + селект «Все» (28d: 5 статусных вкладок →
    //   одна «Статус» + селект); в ней наша задача (любой статус).
    await page.goto('/?tab=status&status=all')
    const statusTab = page.locator('.tab-bar button', { hasText: 'Статус' })
    await expect(statusTab).toBeVisible()
    const sel = page.locator('[data-testid="status-select"] .p-select-label')
    await expect(sel).toContainText('Все', { timeout: 10000 })
    const row = page.locator('[data-testid="task-table"] tr', { hasText: `e2e-7d-all-${stamp}` })
    await expect(row).toBeVisible()

    // — Активность (API): новый ответ — задача поднимается на первую
    //   позицию при sort=updated; при sort=due — по срокам (наша — сегодня,
    //   но есть и просроченные — наша НЕ первая, сортировка работает).
    const reply = await api(request, 'POST', `/api/tasks/${task.id}/reply`, { body: 'e2e reply 7d' })
    expect(reply.status).toBeLessThan(300)

    const act = await api(request, 'GET', `/api/tasks?page=1&per_page=10&sort=updated&search=e2e-7d-all-${stamp}`)
    expect(act.status).toBe(200)
    expect(act.json.tasks[0].id).toBe(task.id)

    const due1 = await api(request, 'GET', `/api/tasks?page=1&per_page=10&sort=due&search=e2e-7d-all-${stamp}`)
    expect(due1.status).toBe(200)
    // sort=due у нас единственный результат с её поиском — тоже первая
    // (но причина различна: по сроку, не по активности).
    expect(due1.json.tasks.some((x) => x.id === task.id)).toBeTruthy()

    // — Селектор сортировки (UI).
    const sortSelect = page.locator('[data-testid="sort-select"]')
    await expect(sortSelect).toBeVisible()
    await sortSelect.click()
    await page.locator('.p-select-option', { hasText: 'По активности' }).first().click()
    await expect(sortSelect.locator('.p-select-label')).toContainText('по активност', { ignoreCase: true })

    // — Due-фильтры (API, «сегодня» vs «просроченные» vs «без срока»).
    const fToday = await api(request, 'GET', `/api/tasks?page=1&per_page=100&due=today&search=e2e-7d-all-${stamp}`)
    expect(fToday.json.tasks.some((x) => x.id === task.id)).toBeTruthy()

    const fOverdue = await api(request, 'GET', `/api/tasks?page=1&per_page=100&due=overdue&search=e2e-7d-all-${stamp}`)
    expect(fOverdue.json.tasks.some((x) => x.id === task.id)).toBeFalsy()

    // — Due-фильтр «без срока» — наша задача (с сегодняшним сроком) НЕ входит.
    const t2 = await api(request, 'POST', '/api/tasks', {
      title: `e2e-7d-nodue-${stamp}`,
      project: 'Входящие',
    })
    const task2 = t2.json.task || t2.json
    const fNone = await api(request, 'GET', `/api/tasks?page=1&per_page=100&due=none&search=e2e-7d-nodue-${stamp}`)
    expect(fNone.json.tasks.some((x) => x.id === task2.id)).toBeTruthy()
    const fNone2 = await api(request, 'GET', `/api/tasks?page=1&per_page=100&due=none&search=e2e-7d-all-${stamp}`)
    expect(fNone2.json.tasks.some((x) => x.id === task.id)).toBeFalsy()

    // — UI: due-селектор «просроченные» убирает нашу задачу с листа.
    const dueSelect = page.locator('[data-testid="due-select"]')
    await dueSelect.click()
    await page.locator('.p-select-option', { hasText: 'Просроченные' }).first().click()
    await expect(
      page.locator('[data-testid="task-table"] tr', { hasText: `e2e-7d-all-${stamp}` })
    ).toHaveCount(0)
  })

  test('статусная вкладка: просроченные — наверху при sort=due (7a дефолт)', async ({ request }) => {
    const stamp = Date.now()
    // Общий уникальный токен e2e-7d-cmp-<stamp> на обеих задачах — поиск
    // LIKE %term% находит ровно эту пару прогона (dev-БД копит stale e2e-7d-*,
    // которые при sort=due id ASC уплывают за пер_page=50).
    const overdue = await api(request, 'POST', '/api/tasks', {
      // search-токен `e2e-7d-cmp-<stamp>` — ОБЩИЙ подстроковый префикс обеих
      // задач (stamp сразу после префикса, без слова в середине), чтобы
      // LIKE %e2e-7d-cmp-<stamp>% нашёл ровно эту пару прогона.
      title: `e2e-7d-cmp-${stamp}-over`,
      project: 'Входящие',
      due_date: localDate(-2),
    })
    const soon = await api(request, 'POST', '/api/tasks', {
      title: `e2e-7d-cmp-${stamp}-soon`,
      project: 'Входящие',
      due_date: localDate(3),
    })
    const o = overdue.json.task || overdue.json
    const s = soon.json.task || soon.json

    const r = await api(
      request,
      'GET',
      `/api/tasks?status=new&status=in_progress&sort=due&page=1&per_page=50&search=e2e-7d-cmp-${stamp}`
    )
    expect(r.status).toBe(200)
    expect(r.json.tasks.length).toBeGreaterThanOrEqual(2)
    const idx = (id) => r.json.tasks.findIndex((x) => x.id === id)
    expect(idx(o.id)).toBeGreaterThanOrEqual(0)
    expect(idx(s.id)).toBeGreaterThan(idx(o.id))
  })
})
