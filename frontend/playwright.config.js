// Playwright config — E2E (шаг 13.3)
// Стенд: dev-бекенд (make run-dev, :8081) + vite dev-сервер (прокси /api -> бек)
import { defineConfig } from '@playwright/test'

const BASE_URL = process.env.E2E_BASE_URL || 'http://127.0.0.1:5173'

export default defineConfig({
  testDir: 'tests-e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 2,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: BASE_URL,
    actionTimeout: 15000,
    navigationTimeout: 20000,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure'
  },
  projects: [
    // setup: вход через UI, сохранение storageState
    { name: 'setup', testMatch: /auth\.setup\.js/ },
    {
      name: 'chromium',
      use: {
        viewport: { width: 1366, height: 900 },
        storageState: 'tests-e2e/.auth/state.json'
      },
      dependencies: ['setup']
    }
  ]
})