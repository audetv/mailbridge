// websocket-reconnect.spec.js — e2e Шаг 0 (v0.22.1): WS-надёжность.
// Сценарии (без F5):
//  1. разрыв → «вкладка в фоне» → возврат → данные актуальны
//     (задача, созданная на сервере во время разрыва, появляется в таблице),
//     индикатор «● Онлайн/○ Офлайн» честно отражает состояние;
//     возврат форсит реконнект СРАЗУ (visibilitychange), не дожидаясь backoff.
//  2. разрыв → широкое время → reconect по backoff-таймеру (onclose-путь).
// Механика: context.routeWebSocket — контроль соединения из теста:
//  mode 'down'  → новое соединение закрывается сразу (сервер «недоходит»);
//  mode 'up'    → проксирование на реальный бекенд (connectToServer).
import { test, expect } from '@playwright/test'

test.describe('WS-reliability — Шаг 0 (v0.22.1)', () => {
  test.describe.configure({ mode: 'serial' })

  let mode = 'up' // 'up' | 'down'
  let liveWs = null // последний WebSocketRoute (для close «с сервера»)

  async function login(request, username, password) {
    const r = await request.post('/api/auth/login', { data: { username, password } })
    expect(r.status()).toBe(200)
    return (await r.json()).token
  }

  async function api(request, method, path, body, token) {
    const headers = { 'Content-Type': 'application/json' }
    if (token) headers.Authorization = `Bearer ${token}`
    const r = await request.fetch(path, { method, data: body, headers })
    let json = null
    try {
      json = await r.json()
    } catch {
      /* нет json */
    }
    return { status: r.status(), json }
  }

  test.beforeEach(async ({ context }) => {
    mode = 'up'
    liveWs = null
    await context.routeWebSocket(/\/api\/ws$/, (ws) => {
      liveWs = ws
      if (mode === 'up') {
        ws.connectToServer()
      } else {
        ws.close({ code: 1011, reason: 'e2e: ws down' })
      }
    })
  })

  test('фон→возврат: данные актуальны БЕЗ F5, реконнект сразу, индикатор честный', async ({
    page,
    context,
    request
  }) => {
    const token = await login(request, 'admin', 'admin')

    await page.goto("/?tab=status&status=active")
    const indicator = page.locator('.connection-status')
    await expect(indicator).toHaveClass(/connected/, { timeout: 15000 })
    await expect(page.locator('[data-testid="task-table"]')).toBeVisible({ timeout: 15000 })

    // ── Разрыв: сервер «не доходит» ──────────────────────────────────
    mode = 'down'
    liveWs?.close({ code: 1011, reason: 'e2e: ws down' })
    // store: onclose → connected=false; новые попытки (backoff) уходят в down → закрываются.
    await expect(indicator).not.toHaveClass(/connected/, { timeout: 15000 })

    // ── На сервере во время разрыва появилась новая задача ────────────
    const projResp = await api(request, 'GET', '/api/projects?archived=false', null, token)
    const projectName = Array.isArray(projResp.json) && projResp.json.length ? projResp.json[0].name : null
    expect(projectName, 'нет проектов в seed').toBeTruthy()
    const title = `E2E-ws0 ${Date.now()}`
    // 7a дефолт: sort=due + IS NULL ASC … id ASC. Новая без срока (NULL)
    // уйдёт за per_page=50 из-за накопившихся stale e2e-строк с due → явный
    // минимальный срок держит её на 1‑й странице (top) при sort=due.
    const overdueMin = '2000-01-01'
    const created = await api(request, 'POST', '/api/tasks', { title, project: projectName, due_date: overdueMin }, token)
    expect(created.status, `POST /tasks -> ${JSON.stringify(created.json)}`).toBe(201)

    // Таблица её НЕ видит (сделана за время разрыва, WS-событие потеряно).
    await expect(page.locator('.subject-cell', { hasText: title })).toHaveCount(0)

    // ── «Вкладка в фоне» ─────────────────────────────────────────────
    await page.evaluate(() => {
      Object.defineProperty(document, 'visibilityState', { value: 'hidden', configurable: true })
      document.dispatchEvent(new Event('visibilitychange'))
    })

    // ── Возврат на вкладку ───────────────────────────────────────────
    mode = 'up'
    await page.evaluate(() => {
      Object.defineProperty(document, 'visibilityState', { value: 'visible', configurable: true })
      document.dispatchEvent(new Event('visibilitychange'))
    })

    // handleVisible форсирует реконнект СРАЗУ (без ожидания 3-сек backoff):
    // ждём online с жёстким 3-сек таймаутом — достаточно для локального сервера,
    // но НЕХВАТИЛО бы от backoff, если бы store ждал 3-го секунды.
    await expect(indicator).toHaveClass(/connected/, { timeout: 3000 })

    // resync: views перетянули данные → новая задача видна в таблице БЕЗ F5.
    await expect(page.locator('.subject-cell', { hasText: title })).toBeVisible({ timeout: 15000 })
    // «БЕЗ F5» проверяем по navigation-типу: "navigate" — первичная загрузка,
    // "reload"/"back_forward" — была перезагрузка (SPA-переходы сюда не попадают).
    const navType = await page.evaluate(() => performance.getEntriesByType('navigation').at(-1).type)
    expect(navType, 'F5 не делался (navigation type)').toBe('navigate')
  })

  test('onclose→backoff: реконнект без участия visibilitychange', async ({
    page,
    context,
    request
  }) => {
    await login(request, 'admin', 'admin')
    await page.goto("/?tab=status&status=active")
    const indicator = page.locator('.connection-status')
    await expect(indicator).toHaveClass(/connected/, { timeout: 15000 })

    mode = 'down'
    liveWs?.close({ code: 1011, reason: 'e2e: ws down' })
    await expect(indicator).not.toHaveClass(/connected/, { timeout: 15000 })

    // Вкладка остаётся видимой; store сам поднимает соединение по backoff
    // (первая попытка — через 3 с). mode снова 'up'.
    mode = 'up'
    await expect(indicator).toHaveClass(/connected/, { timeout: 15000 })
  })

  test('индикатор: при мёртвом канале НЕ врёт «онлайн»', async ({ page, context }) => {
    await page.goto('/?tab=inbox')
    const indicator = page.locator('.connection-status')
    await expect(indicator).toHaveClass(/connected/, { timeout: 15000 })

    mode = 'down'
    liveWs?.close({ code: 1011, reason: 'e2e: ws down' })
    // Во время «даун»: все попытки реконнекта закрываются в 'down',
    // connected=true ставится ТОЛЬКО по фрейму 'connected' → индикатор обязан
    // остаться/стать offline.
    await expect(indicator).not.toHaveClass(/connected/, { timeout: 10000 })
  })
})
