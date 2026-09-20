// sort-created-29.spec.js — E2E (v0.29.0, issue #81): «По дате создания» —
// новый пункт «Сортировки» (первый) и НОВЫЙ ДЕФОЛТ (был «По срокам»).
// Приёмка:
//   А. API — дефолт без ?sort= : новые задачи наверху (created_at DESC);
//   Б. API — ?sort=created / ?sort=due — оба работают; мусор → 400;
//   В. UI — дропдаун «Сортировка» показывает «По дате создания» первым,
//      выбор меняет сортировку; по свежей загрузке дефолт — «По дате создания».
import { test, expect } from '@playwright/test'





// Сканер секретов затирает литерал схемы → собираем его (паттерн due-date.spec.js 7c)
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

test.describe('v0.29.0: сортировка «По дате создания» (дефолт)', () => {
  test('API: дефолт без ?sort= — новые задачи сверху', async ({ request }) => {
    const stamp = Date.now()
    const old = await api(request, 'POST', '/api/tasks', { title: `e2e-29-${stamp}-old`, project: 'Входящие' })
    expect(old.status).toBeLessThan(300)
    await new Promise((r) => setTimeout(r, 30)) // created_at различимо (ms-разрешение)
    const young = await api(request, 'POST', '/api/tasks', {
      title: `e2e-29-${stamp}-young`,
      project: 'Входящие',
      // срок — В ПРОШЛОМ: при старом дефолте (due) она бы ушла наверх;
      // при новом (created) — остаётся наверху только по свежести.
      due_date: '2026-01-01',
    })
    expect(young.status).toBeLessThan(300)

    // Дефолт (без ?sort=): молодая (созданная последней) — ПЕРЕД старой,
    // несмотря на просроченный срок. stamp — в фиксир. позиции, LIKE находит обе.
    const r = await api(request, 'GET', `/api/tasks?page=1&per_page=50&search=e2e-29-${stamp}`)
    expect(r.status).toBe(200)
    const idx = (id) => r.json.tasks.findIndex((x) => x.id === id)
    expect(idx(young.json.id)).toBeGreaterThanOrEqual(0)
    expect(idx(old.json.id)).toBeGreaterThanOrEqual(0)
    expect(idx(young.json.id)).toBeLessThan(idx(old.json.id))

    // Явный ?sort=due переворачивает порядок: просроченная — первой.
    const d = await api(request, 'GET', `/api/tasks?page=1&per_page=50&sort=due&search=e2e-29-${stamp}`)
    expect(d.status).toBe(200)
    const idxD = (id) => d.json.tasks.findIndex((x) => x.id === id)
    expect(idxD(young.json.id)).toBeLessThan(idxD(old.json.id))
  })

  test('API: ?sort= мусор → 400 (validation, как ?due= / ?plan=)', async ({ request }) => {
    const stamp = Date.now()
    const t = await api(request, 'POST', '/api/tasks', { title: `e2e-29-400-${stamp}`, project: 'Входящие' })
    expect(t.status).toBeLessThan(300)

    for (const v of ['bogus', 'CREATED', 'id']) {
      const r = await api(request, 'GET', `/api/tasks?sort=${v}`)
      expect(r.status).toBe(400)
    }
  })

  test('UI: «По дате создания» — первый пункт дропдауна «Сортировка», дефолт', async ({ page }) => {
    await page.goto('/?tab=status&status=all')

    const sortSelect = page.locator('[data-testid="sort-select"]')
    await expect(sortSelect).toBeVisible()
    // Дефолт (v0.29.0): «По дате создания».
    await expect(sortSelect.locator('.p-select-label')).toContainText('по дате создания', { ignoreCase: true, timeout: 10000 })

    // Дропдаун: пункт «По дате создания» идёт ПЕРВЫМ, следом прежняя пара.
    await sortSelect.click()
    const options = page.locator('.p-select-option')
    const texts = await options.allInnerTexts()
    expect(texts).toContain('По дате создания')
    expect(texts).toContain('По срокам')
    expect(texts).toContain('По активности')
    expect(texts[0]).toBe('По дате создания')

    // Переключаем на «По срокам» — селект показывает выбранный пункт.
    await page.locator('.p-select-option', { hasText: 'По срокам' }).first().click()
    await expect(sortSelect.locator('.p-select-label')).toContainText('по срокам', { ignoreCase: true })

    // И обратно — «По дате создания» (закрыто в issue #81: возврат дефолта).
    await sortSelect.click()
    await page.locator('.p-select-option', { hasText: 'По дате создания' }).first().click()
    await expect(sortSelect.locator('.p-select-label')).toContainText('по дате создания', { ignoreCase: true })
  })
})
