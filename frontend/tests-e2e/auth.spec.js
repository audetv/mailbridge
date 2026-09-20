// auth.spec.js — smoke после авторизации (шаг 13.3) + 28d (5 вкладок,
// «Статус» + селект вместо 5 статусных вкладок).
// Логин сам по себе покрывает setup-проект (auth.setup.js), который стартует
// ДО chromium и сохраняет storageState. Здесь — только состояние ПОСЛЕ авторизации:
//  - вкладки Dashboard видимы (5: Лента, Статус, План, Проекты, Персоны)
//  - deep-link «?tab=status&status=active» сразу на «Статус»/«Активные»
import { test, expect } from '@playwright/test'

test('dashboard after login: все 5 вкладок видимы (28d)', async ({ page }) => {
  await page.goto('/')
  // Авторизация уже есть (storageState) — остаёмся на dashборде, не падаем на /login
  await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 })
  await expect(page.getByRole('button', { name: 'Выйти' })).toBeVisible({ timeout: 10000 })

  for (const label of ['Лента', 'Статус', 'План', 'Проекты', 'Персоны']) {
    await expect(page.locator('.tab-bar button', { hasText: label }).first()).toBeVisible()
  }
})

test('deep-link: ?tab=status&status=active сразу на «Статус»/«Активные» (28d)', async ({ page }) => {
  await page.goto('/?tab=status&status=active')
  await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 })

  const statusTab = page.locator('.tab-bar button', { hasText: 'Статус' }).first()
  await expect(statusTab).toHaveClass(/active/, { timeout: 10000 })
  // селект «Статус» показывает «Активные»
  const sel = page.locator('[data-testid="status-select"] .p-select-label')
  await expect(sel).toContainText('Активные', { timeout: 10000 })
})

test('deep-link: ?tab=plan&plan=today сразу на «План»/«Сегодня» (28d)', async ({ page }) => {
  await page.goto('/?tab=plan&plan=today')
  await expect(page).not.toHaveURL(/\/login/, { timeout: 10000 })

  const planTab = page.locator('.tab-bar button', { hasText: 'План' }).first()
  await expect(planTab).toHaveClass(/active/, { timeout: 10000 })
  const sel = page.locator('[data-testid="plan-select"] .p-select-label')
  await expect(sel).toContainText('Сегодня', { timeout: 10000 })
  // приёмка 28d: deep-link переживает reload (URL — source of truth)
  await page.reload({ waitUntil: 'load' })
  await expect(planTab).toHaveClass(/active/, { timeout: 10000 })
  await expect(sel).toContainText('Сегодня', { timeout: 10000 })
})
