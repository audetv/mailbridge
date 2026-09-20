// Playwright config — E2E (шаг 13.3)
// Стенд: dev-бекенд (make run-dev, :8081) + vite dev-сервер (прокси /api -> бек)
import { defineConfig } from '@playwright/test'

const BASE_URL = process.env.E2E_BASE_URL || 'http://127.0.0.1:5173'

export default defineConfig({
  testDir: 'tests-e2e',
  // ── Параллельность ВЫКЛЮЧЕНА (решение 2026-09-20, issue #75) ─────────────────
  // Все e2e-тесты работают ПРОТИВ ОДНОГО общего живого бэкенда + ОДНОЙ SQLite +
  // ОДНОГО WebSocket и все под тем же логином «admin». При workers>1 воркер B
  // создаёт задачу → бэкенд шлёт WS «task_created» ВСЕМ сокетам того же
  // «admin» → страница воркера A (открыта, слушает) реагирует
  // fetchTasks() СРЕДИ СВОИХ АССЕРШНОВ → loading=true → .p-datatable-mask
  // накрывает UI воркера A → тест A падает (маска/таймаут), а «битый» тест
  // прыгает от прогона к прогону. Факт (измерено): workers=1 → 45/45 стабильно;
  // workers=2 → падение в каждом из 3 прогонов, разный тест (bulk / «B: Проект»).
  // Корень — общее мутируемое состояние + кросс-контекстный WS, а не конкретный
  // тест → чинить «в лоб» нечего. Истинная параллельная изоляция требует
  // per-worker бэкенд/БД (задача back-log), поэтому suite выполняется СЕРИЙНО —
  // это детерминированно и быстро (≈58с). Если понадобится параллелизм —
  // сначала инфраструктурная задача об изоляции состояний.
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
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