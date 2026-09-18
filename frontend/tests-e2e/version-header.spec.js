// version-header.spec.js — v0.27, шаг 8: версия сборки в UI (РЕЖИМ A).
// Принятие: GET /api/version без auth = 200 + JSON {version, commit, built};
// шапка Dashboard отображает строку версии рядом «● Онлайн/Офлайн».
import { test, expect } from '@playwright/test'

test('GET /api/version без auth = 200 + JSON {version, commit, built}', async ({ page }) => {
  // window.fetch — без заголовка Authorization (axios-интерсептор не задействован)
  const res = await page.request.get('/api/version')
  expect(res.status()).toBe(200)
  const body = await res.json()
  expect(typeof body.version).toBe('string')
  expect(body.version.length).toBeGreaterThan(0)
  expect(body.commit).toBeTruthy()
  expect(body.built).toBeTruthy()
})

test('шапка: бейдж версии рядом со статусом соединения', async ({ page }) => {
  await page.goto('/')
  await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 })

  // API вернул версию → бейдж v<version> в .header-right
  const apiBody = await (await page.request.get('/api/version')).json()
  const badge = page.locator('.header-right .version-badge')
  await expect(badge).toBeVisible({ timeout: 10000 })
  await expect(badge).toHaveText(`v${apiBody.version}`)
  // commit + time сборки — в tooltip (title)
  await expect(badge).toHaveAttribute('title', new RegExp(`commit: ${apiBody.commit}`))
  await expect(badge).toHaveAttribute('title', new RegExp(`сборка: ${apiBody.built}`))
})
