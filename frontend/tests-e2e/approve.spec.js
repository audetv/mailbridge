// approve.spec.js — E2E-приёмка ФАЗЫ 4 (шаг 20)
// Кейс: агент hermes пишет kind=report и kind=reply на задачу;
// admin видит бейджи «Отчёт» / «Ответ пользователю», жмёт «Утвердил ответ»,
// появляется бейдж «Утверждён». API-guards: approve от hermes → 403,
// approve kind=report → 400, повторный approve → 200 (idempotent).
//
// Пароль агента: env MAILBRIDGE_AGENT_PASS (configs/config.env, не git).
// Если env не задан — читаем из configs/config.env, чтобы тест не падал
// в чистом shell без `source configs/config.env`.
import { test, expect } from '@playwright/test'
import { existsSync, readFileSync } from 'node:fs'

function agentPassFromConfigEnv() {
  const p = new URL('../../configs/config.env', import.meta.url).pathname
  if (!existsSync(p)) return ''
  const m = readFileSync(p, 'utf8').match(/^\s*MAILBRIDGE_AGENT_PASS\s*=\s*(\S+)\s*$/m)
  return m ? m[1] : ''
}

const BASE = process.env.E2E_BASE_URL || 'http://127.0.0.1:5173'
const AGENT_PASS = process.env.MAILBRIDGE_AGENT_PASS || agentPassFromConfigEnv()
const MARK = `E2E-phase4-${Date.now()}`

test.describe.configure({ mode: 'serial' })

let taskId
let reportCommentId
let replyCommentId
let agentToken
let adminToken
let requestCtx

test.afterAll(async () => {
  if (requestCtx) await requestCtx.dispose()
})

// ── Хелперы API ───────────────────────────────────────────────────────────
async function api(request, method, path, body, token) {
  const headers = { 'Content-Type': 'application/json' }
  if (token) headers.Authorization = `Bearer ${token}`
  const r = await request.fetch(path, {
    method,
    data: body,
    headers
  })
  let json = null
  try {
    json = await r.json()
  } catch {
    /* text */
  }
  return { status: r.status(), json }
}

async function login(request, username, password) {
  const { status, json } = await api(
    request,
    'POST',
    '/api/auth/login',
    { username, password },
    null
  )
  expect(status, `login ${username}`).toBe(200)
  return json.token
}

test.beforeAll(async ({ playwright }) => {
  expect(AGENT_PASS, 'MAILBRIDGE_AGENT_PASS — задать в env/конфиге').toBeTruthy()
  requestCtx = await playwright.request.newContext({ baseURL: BASE })
  const request = requestCtx

  adminToken = await login(request, 'admin', 'admin')
  agentToken = await login(request, 'hermes', AGENT_PASS)

  // Задача: берём первую активную (seed гарантирует существование).
  const { status, json } = await api(request, 'GET', '/api/tasks?status=new&per_page=1', null, adminToken)
  expect(status).toBe(200)
  const tasks = json.tasks || json
  expect(tasks.length, 'нет задач new — seed?').toBeGreaterThan(0)
  taskId = tasks[0].id

  // Агент: отчёт + ответ пользователю.
  const rep = await api(request, 'POST', `/api/tasks/${taskId}/reply`, { body: `отчёт ${MARK}`, kind: 'report' }, agentToken)
  expect(rep.status, `hermes POST report (${rep.json})`).toBe(200)
  reportCommentId = rep.json.comment.id

  const rep2 = await api(request, 'POST', `/api/tasks/${taskId}/reply`, { body: `ответ ${MARK}`, kind: 'reply' }, agentToken)
  expect(rep2.status, `hermes POST reply (${rep2.json})`).toBe(200)
  replyCommentId = rep2.json.comment.id
})

test('UI: admin видит бейджи «Отчёт»/«Ответ пользователю» и утверждает ответ', async ({ page }) => {
  await page.goto(`/tasks/${taskId}`)

  // Карточка с нашим текстом (контейнер списка — .comments, карточка — .comment).
  const card = page.locator('.comment', { hasText: `ответ ${MARK}` }).first()
  await expect(card).toBeVisible({ timeout: 10000 })
  await expect(card.locator('.kind-badge', { hasText: 'Ответ пользователю' })).toBeVisible()
  await expect(card.locator('.approved-badge')).toHaveCount(0)

  // Утверждаем.
  await card.locator('.approve-btn', { hasText: 'Утвердить ответ' }).click()

  // Бейдж «Утверждён» появляется (локальная мутация или WS).
  await expect(card.locator('.approved-badge', { hasText: 'Утверждён' })).toBeVisible({ timeout: 5000 })
  await expect(card.locator('.approve-btn')).toHaveCount(0)

  // Отчёт тоже видим с бейджем «Отчёт», без approve-кнопки.
  const repCard = page.locator('.comment', { hasText: `отчёт ${MARK}` }).first()
  await expect(repCard.locator('.kind-badge', { hasText: 'Отчёт' })).toBeVisible()
  await expect(repCard.locator('.approve-btn')).toHaveCount(0)
})

test('UI: reload — approved живёт в БД (approved=1 после перезагрузки)', async ({ page }) => {
  await page.goto(`/tasks/${taskId}`)
  const card = page.locator('.comment', { hasText: `ответ ${MARK}` }).first()
  await expect(card.locator('.approved-badge', { hasText: 'Утверждён' })).toBeVisible({ timeout: 10000 })
})

test('UI: селект kind — выбор «Ответ пользователю» уходит в запрос как строка kind=reply', async ({ page }) => {
  await page.goto(`/tasks/${taskId}`)
  await page.waitForSelector('.reply-form', { timeout: 10000 })

  const body = `ui-select ${MARK}`
  const replyBodies = []
  page.on('request', (r) => {
    if (r.method() === 'POST' && r.url().endsWith('/reply')) {
      replyBodies.push(JSON.parse(r.postData() || '{}'))
    }
  })

  // Выбор вида через PrimeVue Select: клик по поле → опция в дропдауне.
  await page.locator('.kind-select').click()
  await page.locator('.p-select-option, [role="option"]', { hasText: 'Ответ пользователю' }).first().click()

  await page.locator('textarea').first().fill(body)
  await page.getByRole('button', { name: 'Отправить' }).click()

  const card = page.locator('.comment', { hasText: body }).first()
  // Комментарий создан (до фикса бэкенд отвечал 400 и карточки не было).
  await expect(card, 'комментарий из формы с выбранным kind виден в ленте').toBeVisible({ timeout: 10000 })
  // Бейдж «Ответ пользователю» — kind сохранился на бэкенде.
  await expect(card.locator('.kind-badge', { hasText: 'Ответ пользователю' })).toBeVisible()

  // Запрос шёл с kind как строка, а не объектом {value,label}.
  const sent = replyBodies.find((b) => b.body === body)
  expect(sent, 'запрос POST /reply с нашим текстом').toBeTruthy()
  expect(sent.kind, 'kind в payload — строка').toBe('reply')
})

test('API guards: hermes approve → 403, kind=report → 400, повтор → 200', async ({ request }) => {
  // 403 — агенту approve недоступен.
  const forbidden = await api(request, 'PATCH', `/api/comments/${replyCommentId}/approve`, null, agentToken)
  expect(forbidden.status).toBe(403)
  expect(forbidden.json.error).toContain('admin')

  // 400 — отчёт (kind=report) нельзя утверждать.
  const badkind = await api(request, 'PATCH', `/api/comments/${reportCommentId}/approve`, null, adminToken)
  expect(badkind.status).toBe(400)
  expect(badkind.json.error).toContain('kind=reply')

  // 200 — повторный approve idempotent.
  const again = await api(request, 'PATCH', `/api/comments/${replyCommentId}/approve`, null, adminToken)
  expect(again.status).toBe(200)
  expect(again.json.comment.approved).toBe(1)
})
