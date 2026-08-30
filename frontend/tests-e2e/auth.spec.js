// auth.spec.js — smoke после авторизации (шаг 13.3)
// Логин сам по себе покрывает setup-проект (auth.setup.js), который стартует
// ДО chromium и сохраняет storageState. Здесь — только состояние ПОСЛЕ авторизации:
//  - вкладки Dashboard видимы
//  - deep-link «?tab=active» сразу на «Активные»
import { test, expect } from '@playwright/test'

test('dashboard after login: все 6 вкладок видимы', async ({ page }) => {
  await page.goto('/')
  // Авторизация уже есть (storageState) — остаёмся на dashборде, не падаем на /login
  await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 })
  await expect(page.getByRole('button', { name: 'Выйти' })).toBeVisible({ timeout: 10000 })

  for (const label of ['Лента', 'Активные', 'Бэклог', 'Выполненные', 'Закрытые', 'Проекты']) {
    await expect(page.locator('.tab-bar button', { hasText: label }).first()).toBeVisible()
  }
})

test('deep-link: ?tab=active сразу на «Активные»', async ({ page }) => {
  await page.goto('/?tab=active')
  await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 })

  const active = page.locator('.tab-bar button', { hasText: 'Активные' }).first()
  await expect(active).toHaveClass(/active/, { timeout: 10000 })
})
