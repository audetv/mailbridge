// DashboardTasksTab.spec.js — URL — source of truth для вкладки (шаг 13.2) +
// v0.28/28d: вкладки Инбокс/Статус/План/Проекты/Персоны; «Статус» + селект
// (?status=all|active|backlog|completed|closed), «План» + селект (?plan=
// today|tomorrow|week, срез 28b). Старые ключи ?tab=all/active/backlog/… —
// ломаются (без миграции, решение владельца 2026-09-19): unknown tab → дефолт.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'

const tasksMock = {
  setFilter: vi.fn((k, v) => { tasksMock.filters[k] = v }),
  setStatuses: vi.fn((s) => { tasksMock.filters.statuses = s }),
  // 28d: две формы вкладки — status (+ селект-значение) и plan (+ значение).
  STATUS_FILTER: { all: [], active: ['new', 'in_progress'], backlog: ['backlog'], completed: ['completed'], closed: ['closed'] },
  PLAN_FILTER: { today: 'today', tomorrow: 'tomorrow', week: 'week' },
  setTab: vi.fn((tab, value = 'active') => {
    if (tab === 'status') {
      tasksMock.filters.statuses = [...(tasksMock.STATUS_FILTER[value] || [])]
      tasksMock.filters.plan = ''
      tasksMock.filters.due = ''
      tasksMock.filters.sort = 'due'
    } else if (tab === 'plan') {
      tasksMock.filters.plan = tasksMock.PLAN_FILTER[value] || 'today'
      tasksMock.filters.statuses = []
      tasksMock.filters.due = ''
      tasksMock.filters.sort = 'due'
    }
    tasksMock.filters.page = 1
    tasksMock.fetchTasks()
  }),
  setStatusFilter: vi.fn((v) => {
    tasksMock.filters.statuses = [...(tasksMock.STATUS_FILTER[v] || [])]
    tasksMock.filters.plan = ''
    tasksMock.fetchTasks()
  }),
  setPlanFilter: vi.fn((v) => {
    tasksMock.filters.plan = v
    tasksMock.filters.due = ''
    tasksMock.fetchTasks()
  }),
  fetchTasks: vi.fn(async () => {}),
  filters: { project: '', epic_id: '', statuses: ['new', 'in_progress'], sort: 'due', due: '', plan: '', page: 1, per_page: 50 }
}
const wsMock = { connected: false, events: [], connect: vi.fn(), disconnect: vi.fn() }

// Мок API — vi.hoisted: vi.mock-фабрика поднимается выше переменных,
// поэтому функции живут в hoisted-блоке.
const apiGet = vi.hoisted(() =>
  vi.fn(async (url) => {
    if (url === '/projects') {
      return { data: [{ id: 7, name: 'ТРК', description: '', archived: false }] }
    }
    if (url === '/projects/7/epics') {
      return { data: [{ id: 11, number: 3, name: 'E2E-модуль' }] }
    }
    if (url === '/inbox') return { data: { total: 0, items: [] } }
    return { data: { tasks: [], total: 0 } }
  })
)

vi.mock('@/stores/tasks', () => ({ useTasksStore: () => tasksMock }))
vi.mock('@/stores/websocket', () => ({ useWebSocket: () => wsMock }))
vi.mock('@/api/client', () => ({
  default: { get: apiGet, post: vi.fn(), patch: vi.fn(), delete: vi.fn() }
}))
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: vi.fn() }) }))
vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ token: 't', user: { username: 'admin' }, logout: vi.fn() })
}))
vi.mock('@/components/CreateTaskDialog.vue', () => ({ default: { template: '<div/>' } }))
vi.mock('@/views/InboxView.vue', () => ({ default: { template: '<div data-testid="inbox-view"/>' } }))
vi.mock('@/components/TaskTable.vue', () => ({ default: { template: '<div data-testid="task-table"/>' } }))
vi.mock('@/views/ProjectsView.vue', () => ({ default: { template: '<div data-testid="projects-view"/>' } }))

import DashboardView from '@/views/DashboardView.vue'

async function mountAt(path) {
  setActivePinia(createPinia())
  // navigation ДО mount: route.query (для deep-link ?project=…) должен быть
  // готов до setup компонента, иначе seed фильтра в setup его не увидит.
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: DashboardView },
      { path: '/login', name: 'login', component: { template: '<div/>' } },
      { path: '/tasks/:id', name: 'task-detail', component: { template: '<div/>' } }
    ]
  })
  await router.push(path)
  await router.isReady()
  const wrapper = mount(DashboardView, { global: { plugins: [router] } })
  return { router, wrapper }
}

describe('DashboardView — вкладки v0.28/28d (Статус+селект / План+селект / URL)', () => {
  beforeEach(async () => {
    vi.restoreAllMocks()
    tasksMock.filters = { project: '', epic_id: '', statuses: ['new', 'in_progress'], sort: 'due', due: '', plan: '', page: 1, per_page: 50 }
    localStorage.clear()
  })

  it('5 вкладок в порядке: Лента, Статус, План, Проекты, Персоны', async () => {
    const { wrapper } = await mountAt('/')
    await flushPromises()
    const labels = wrapper.findAll('.tab-bar button').map((b) => b.text().trim()).map((t) => t.replace(/\d+$/, '').trim())
    expect(labels).toEqual(['Лента', 'Статус', 'План', 'Проекты', 'Персоны'])
    // старых вкладак нет
    const all = wrapper.findAll('.tab-bar button').map((b) => b.text())
    expect(all.some((t) => t.trim() === 'Все')).toBe(false)
    expect(all.some((t) => t.trim() === 'Активные')).toBe(false)
    expect(all.some((t) => t.trim() === 'Бэклог')).toBe(false)
  })

  it('дефолт (без ?tab) — вкладка «Статус», селект «Активные», fetch по статусам', async () => {
    const { wrapper } = await mountAt('/')
    await flushPromises()
    expect(wrapper.find('[data-testid="task-table"]').exists()).toBe(true)
    const statusBtn = wrapper.findAll('.tab-bar button').find((b) => b.text().includes('Статус'))
    expect(statusBtn.classes()).toContain('active')
    expect(tasksMock.setTab).toHaveBeenCalledWith('status', 'active')
    expect(tasksMock.filters.statuses).toEqual(['new', 'in_progress'])
    expect(tasksMock.filters.plan).toBe('')
  })

  it('deep-link ?tab=status&status=completed: «Статус» + селект «Выполненные» + fetch', async () => {
    const { wrapper } = await mountAt('/?tab=status&status=completed')
    await flushPromises()
    expect(tasksMock.setTab).toHaveBeenCalledWith('status', 'completed')
    expect(tasksMock.filters.statuses).toEqual(['completed'])
    const sel = wrapper.find('[data-testid="status-select"] .p-select-label')
    expect(sel.text()).toContain('Выполненные')
  })

  it('deep-link ?tab=plan&plan=tomorrow: «План» + селект «Завтра», БЕЗ status, fetch с plan', async () => {
    const { wrapper } = await mountAt('/?tab=plan&plan=tomorrow')
    await flushPromises()
    expect(tasksMock.setTab).toHaveBeenCalledWith('plan', 'tomorrow')
    expect(tasksMock.filters.plan).toBe('tomorrow')
    expect(tasksMock.filters.statuses).toEqual([])
    const sel = wrapper.find('[data-testid="plan-select"] .p-select-label')
    expect(sel.text()).toContain('Завтра')
  })

  it('старый ?tab=active ломается (без миграции): fallback на «Статус»', async () => {
    await mountAt('/?tab=active')
    await flushPromises()
    expect(tasksMock.setTab).toHaveBeenCalledWith('status', 'active')
  })

  it('старые ?tab=all / backlog / completed / closed — все ломаются → «Статус»', async () => {
    for (const old of ['all', 'backlog', 'completed', 'closed']) {
      vi.restoreAllMocks()
      tasksMock.filters = { project: '', epic_id: '', statuses: ['new', 'in_progress'], sort: 'due', due: '', plan: '', page: 1, per_page: 50 }
      await mountAt('/?tab=' + old)
      await flushPromises()
      expect(tasksMock.setTab).toHaveBeenLastCalledWith('status', 'active')
    }
  })

  it('клик «План» → URL ?tab=plan (без ?status=), store — план-срез, статусы выключены', async () => {
    const { router, wrapper } = await mountAt('/?tab=status&status=completed')
    await flushPromises()
    const planTab = wrapper.findAll('.tab-bar button').find((b) => b.text().trim() === 'План')
    await planTab.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.tab).toBe('plan')
    expect(router.currentRoute.value.query.status).toBeUndefined()
    expect(tasksMock.setTab).toHaveBeenLastCalledWith('plan', 'today')
    expect(tasksMock.filters.statuses).toEqual([])
    expect(tasksMock.filters.plan).toBe('today')
  })

  it('клик «Статус» из «План» → URL ?tab=status (без ?plan=), статусы снова включены', async () => {
    const { router, wrapper } = await mountAt('/?tab=plan&plan=week')
    await flushPromises()
    const statusTab = wrapper.findAll('.tab-bar button').find((b) => b.text().includes('Статус'))
    await statusTab.trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.tab).toBe('status')
    expect(router.currentRoute.value.query.status).toBe('active')
    expect(router.currentRoute.value.query.plan).toBeUndefined()
    expect(tasksMock.setTab).toHaveBeenLastCalledWith('status', 'active')
    expect(tasksMock.filters.statuses).toEqual(['new', 'in_progress'])
    expect(tasksMock.filters.plan).toBe('')
  })

  it('селект «Статус»: выбор «Бэклог» → ?status=backlog, fetch по статусам', async () => {
    await mountAt('/?tab=status&status=active')
    await flushPromises()
    tasksMock.setStatusFilter('backlog')
    expect(tasksMock.filters.statuses).toEqual(['backlog'])
    expect(tasksMock.fetchTasks).toHaveBeenCalled()
  })

  it('«Создать задачу» виден во вкладках «Статус» и «План», НЕ в «Ленте»', async () => {
    for (const [tab, visible] of [
      ['status', true],
      ['plan', true],
      ['inbox', false]
    ]) {
      vi.restoreAllMocks()
      tasksMock.filters = { project: '', epic_id: '', statuses: ['new', 'in_progress'], sort: 'due', due: '', plan: '', page: 1, per_page: 50 }
      const { wrapper } = await mountAt('/?tab=' + tab)
      await flushPromises()
      const createBtn = wrapper.findAll('button').some((b) => b.text().includes('Создать задачу'))
      expect(createBtn, `tab=${tab}`).toBe(visible)
    }
  })

  it('вкладках «Статус»/«План» — TaskTable; в Ленте/Проектах/Персонах — вью', async () => {
    for (const [path, expectTable] of [
      ['/?tab=status&status=active', true],
      ['/?tab=plan&plan=today', true],
      ['/?tab=inbox', false],
      ['/?tab=projects', false],
      ['/?tab=persons', false]
    ]) {
      vi.restoreAllMocks()
      tasksMock.filters = { project: '', epic_id: '', statuses: ['new', 'in_progress'], sort: 'due', due: '', plan: '', page: 1, per_page: 50 }
      const { wrapper } = await mountAt(path)
      await flushPromises()
      expect(wrapper.find('[data-testid="task-table"]').exists(), path).toBe(expectTable)
    }
  })
})