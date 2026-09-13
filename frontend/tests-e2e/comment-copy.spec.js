// comment-copy.spec.js — Смоук UI (v0.23, шаг 3): у каждого комментария
// кнопки «Копировать (чистый текст)» и «MD» кладут в clipboard
// соответствующую версию body (navigator.clipboard — мок).
//
// Задача #38 — в dev-БД 3 комментария, у первого MD-символы (**, списки, ##),
// что даёт реальное различие между режимами 'text' и 'md'.
import { test, expect } from '@playwright/test'

const TASK_URL = '/tasks/38'

test.describe('copy buttons on comments (step 3)', () => {
  test('clipboard icon + MD chip appear near every comment (≥2)', async ({ page }) => {
    await page.goto(TASK_URL)
    await page.waitForSelector('.comment', { timeout: 10_000 })
    const n = await page.locator('.comment').count()
    expect(n).toBeGreaterThanOrEqual(2)
    // У каждого комментария — 2 кнопки (расширено: не только reply).
    await expect(page.locator('.comment .copy-btn')).toHaveCount(n * 2)
    await expect(page.locator('.comment .pi-clipboard').first()).toBeVisible()
    await expect(page.locator('.comment .copy-btn', { hasText: 'MD' }).first()).toBeVisible()
  })

  test('MD button writes the RAW body verbatim (symbols ** ## - kept)', async ({ page }) => {
    await page.context().grantPermissions(['clipboard-read', 'clipboard-write'])
    await page.goto(TASK_URL)
    await page.locator('.comment-body').first().waitFor({ state: 'visible', timeout: 10_000 })

    // Снимаем "сырое" тело из DOM (white-space: pre-wrap сохраняет \n)
    const raw = await page.locator('.comment-body').first().evaluate((el) => el.innerText)

    const mdBtn = page.locator('.comment').first().locator('.copy-btn', { hasText: /^MD$/ })
    await mdBtn.click()

    const clip = await page.evaluate(async () => navigator.clipboard.readText())
    expect(clip).toBe(raw)
  })

  test('clean-text button strips MD symbols, keeps newlines', async ({ page }) => {
    await page.context().grantPermissions(['clipboard-read', 'clipboard-write'])
    await page.goto(TASK_URL)
    await page.locator('.comment-body').first().waitFor({ state: 'visible', timeout: 10_000 })

    const btn = page.locator('.comment').first().locator('.copy-btn:has(.pi-clipboard)')
    await btn.click()

    const clip = await page.evaluate(async () => navigator.clipboard.readText())
    expect(clip.length).toBeGreaterThan(10)
    expect(clip).toContain('\n')            // переносы сохранены
    // MD-символы сняты: жирные **, заголовки ## (начало строки)
    expect(clip).not.toMatch(/\*\*/)
    expect(clip).not.toMatch(/(^|\n)\s*#{1,3} /)
  })

  test('feedback: "MD" briefly becomes a check mark, then reverts', async ({ page }) => {
    await page.context().grantPermissions(['clipboard-read', 'clipboard-write'])
    await page.goto(TASK_URL)
    await page.locator('.comment-body').first().waitFor({ state: 'visible', timeout: 10_000 })

    // Стабильная ссылка: 2-я кнопка копирования внутри первого комментария
    // (1-я — чистый текст, 2-я — MD). Позиция не меняется при смене текста.
    const mdBtn = page.locator('.comment').first().locator('.copy-btn').nth(1)
    await mdBtn.click()
    await expect(mdBtn).toHaveText('✓ MD', { timeout: 2_000 })
    await page.waitForTimeout(2300)
    await expect(mdBtn).toHaveText('MD')
  })
})
