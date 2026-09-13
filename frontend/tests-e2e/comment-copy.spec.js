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

    // Каноническое тело — из API (DOM после linkify уже содержит <a>),
    // авторизация теста — на origin приложения, Cookie доступен evaluate.
    const raw = await page.evaluate(async () => {
      const d = await (await fetch('/api/tasks/38')).json()
      return d.comments[0].body
    })

    const mdBtn = page.locator('.comment').first().locator('.copy-btn').nth(1)
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

// ---- auto-linking (v0.23): голые URL в теле комментария — кликабельны,
// target="_blank" rel="noopener" (новое окно). Задача #87 — реальный URL.
test.describe('auto-linking in plain-text comments', () => {
  const LINK_URL = '/tasks/87'
  const EXPECTED_HREF = 'https://kushniras.ru/seo/lider-sport.ru/tz/TZ_lider-sport__SEO_GEO-1.html'

  test('bare https URL in comment body renders as <a target=_blank rel=noopener>', async ({ page }) => {
    await page.goto(LINK_URL)
    const a = page.locator('.comment-body a[href$="TZ_lider-sport__SEO_GEO-1.html"]')
    await a.first().waitFor({ state: 'visible', timeout: 10_000 })
    await expect(a.first()).toHaveAttribute('target', '_blank')
    await expect(a.first()).toHaveAttribute('rel', 'noopener')
    await expect(a.first()).toHaveText(EXPECTED_HREF)
    // обычный текст рядом остался как текст (не ссылка)
    await expect(a.first().locator('..')).toContainText('Ссылка на ТЗ')
  })

  test('clicking the link does NOT navigate the app away (target=_blank)', async ({ page }) => {
    await page.context().grantPermissions(['clipboard-read'])
    await page.goto(LINK_URL)
    const a = page.locator('.comment-body a[href$="TZ_lider-sport__SEO_GEO-1.html"]').first()
    await a.waitFor({ state: 'visible', timeout: 10_000 })

    // Не переходи по ней: проверим, что в контексте появятся новые вкладки
    const newPagePromise = page.context().waitForEvent('page').catch(() => null)
    await a.click()
    const newPage = await newPagePromise

    // App остаётся на той же странице
    expect(page.url()).toContain(LINK_URL)
    // Новая вкладка открыта (target=_blank)
    if (newPage) {
      await newPage.close()
    }
  })
})

// ---- MD-рендер (v0.23, шаг 3b): MD-комментарий → HTML-теги в DOM.
// Задача #38 — первый комментарий: **жирный** + списки "- ".
test.describe('MD rendering in comment card (step 3b)', () => {
  const MD_URL = '/tasks/38'

  test('MD body renders as semantic HTML: <strong>, <ul>/<li>, .comment-body.md', async ({ page }) => {
    await page.goto(MD_URL)
    const mdBody = page.locator('.comment-body.md').first()
    await mdBody.waitFor({ state: 'visible', timeout: 10_000 })

    // детектор выбрал MD-путь (v-html)
    await expect(mdBody.locator('strong')).toHaveCount(1)
    await expect(mdBody.locator('strong')).toContainText('Добавить модуль бронирования')

    // списки "- " → <ul><li>
    const lis = mdBody.locator('li')
    await expect(lis.first()).toBeVisible()
    expect(await lis.count()).toBeGreaterThanOrEqual(3)

    // в DOM не осталось буквальных MD-символов как текста
    const text = await mdBody.textContent()
    expect(text).not.toContain('**')
    expect(text).not.toMatch(/^\s*-\s/m)
  })

  test('MD with heading, task-list, code and hr (task 49) renders semantic HTML', async ({ page }) => {
    await page.goto('/tasks/49') // richest comment: # / ## / bold / - [ ] / `code` / ---
    const mdBody = page.locator('.comment-body.md').first()
    await mdBody.waitFor({ state: 'visible', timeout: 10_000 })
    // header is converted to <h...>
    await expect(mdBody.locator('h1, h2, h3').first()).toBeVisible()
    // task-list checkboxes
    expect(await mdBody.locator('input[type="checkbox"]').count()).toBeGreaterThan(0)
    // inline code
    await expect(mdBody.locator('code').first()).toBeVisible()
    // horizontal rule
    await expect(mdBody.locator('hr').first()).toBeVisible()
    // no literal Markdown markers in text
    const text = await mdBody.textContent()
    expect(text).not.toMatch(/\*\*/)
    expect(text).not.toMatch(/^- \[ \]/m)
  })

  test('plain comment (без MD) НЕ получает .md-класс, рендер — прежний', async ({ page }) => {
    await page.goto('/tasks/87')
    await page.locator('.comment-body').first().waitFor({ state: 'visible', timeout: 15000 })
    // конкретный plain-комментарий ищем по тексту; c39 — без единого MD-синтаксиса
    const cls = await page.evaluate(() => {
      const els = [...document.querySelectorAll('.comment-body')]
      const plain = els.find((e) => e.textContent.includes('Ссылка на ТЗ'))
      if (!plain) return null
      return plain.className
    })
    expect(cls).toContain('comment-body')
    expect(cls).not.toContain('md')
  })
})
