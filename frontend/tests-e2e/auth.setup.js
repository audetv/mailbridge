// auth.setup.js — вход через UI, сохранение storageState для e2e-сценариев
import { test, expect } from '@playwright/test'

test('login via UI and save state', async ({ page, context }) => {
  await page.goto('/login')
  await page.getByPlaceholder('Логин').fill('admin')
  await page.getByPlaceholder('Пароль').fill('admin')
  await page.getByRole('button', { name: 'Войти' }).click()
  await expect(page).not.toHaveURL(/\/login/, { timeout: 15000 })
  const state = await context.storageState({ path: 'tests-e2e/.auth/state.json' })
  expect(state.origins.length).toBeGreaterThan(0)
})
