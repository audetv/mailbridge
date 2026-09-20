// DashboardTasksTab.spec.js — URL — source of truth для вкладки (шаг 13.2) +
// v0.28/28e: вкладки-дропдауны (Bootstrap «Tabs with dropdowns», решение
// владельца 2026-09-20): «Статус» и «План» — дропдаун-кнопки, таб показывает
// выбранный пункт. ОТДЕЛЬНОГО селекта НЕТ (откат UI 28d).
// «Статус»: Все/Активные/Бэклог/Выполненные/Закрытые → ?status=;
// «План»: Все/Без плана/Сегодня/Завтра/Неделя → ?plan= (none/add 28e, срез 28b).
// Старые ключи ?tab=all/active/backlog/… — ломаются: unknown tab → дефолт.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'

const tasksMock = {
  setFilter: vi.fn((k, v) => { tasksMock.filters[k] = v }),
  setStatuses: vi.fn((s) => { tasksMock.filters.statuses = s }),
  // 28e: две формы вкладки-дропдаун — status (+ пункт) и plan (+ пункт).
  STATUS_FILTER: { all: [], active: ['new', 'in_progress'], backlog: ['backlog'], completed: ['completed'], closed: ['closed'] },
  PLAN_FILTER: { all: '', none: 'none', today: 'today', tomorrow: 'tomorrow', week: 'week' },
  setTab: vi.fn((tab, value = 'active') => {
    if (tab === 'status') {
      tasksMock.filters.statuses = [...(tasksMock.STATUS_FILTER[value] || [])]
      tasksMock.filters.plan = ''
      tasksMock.filters.due = ''
      tasksMock.filters.sort = 'due'
    } else if (tab === 'plan') {
      tasksMock.filters.plan = tasksMock.PLAN_FILTER[value] ?? 'today'
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
    tasksMock.filters.plan = tasksMock.PLAN_FILTER[v] ?? v
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

describe('DashboardView — вкладки-дропдауны v0.28/28e (Статус ▼ / План ▼ / URL)', () => {
  beforeEach(async () => {
    vi.restoreAllMocks()
    tasksMock.filters = { project: '', epic_id: '', statuses: ['new', 'in_progress'], sort: 'due', due: '', plan: '', page: 1, per_page: 50 }
    localStorage.clear()
  })

  it('5 вкладок в порядке: Лента, Статус, План, Проекты, Персоны; две — дропдауны', async () => {
    const { wrapper } = await mountAt('/')
    await flushPromises()
    // топ-уровневые кнопки: таб = пункту выбора (показывает ВЫБРАННЫЙ пункт:
    // по дефолту Статус → «Активные», План → «Сегодня»; 28e, решение владельца)
    const labels = wrapper.findAll('[data-testid^="tab-"]').map((b) => b.find('.tab-dd-label').exists() ? b.find('.tab-dd-label').text().trim() : b.text().trim())
    expect(labels).toEqual(['Лента', 'Активные', 'Сегодня', 'Проекты', 'Персоны'])
    // дропдауны именно в Статус и Плане
    expect(wrapper.find('[data-testid="tab-status"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="tab-plan"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="tab-inbox"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="tab-projects"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="tab-persons"]').exists()).toBe(true)
    // пунктов в меню вне дропдаунов нет (таб «Лента»/«Проекты»/«Персоны» — простые)
    const top = wrapper.findAll('[data-testid^="tab-"]').map((b) => b.text())
    expect(top.some((t) => t.trim() === 'Все')).toBe(false)
  })

  it('дефолт (без ?tab) — вкладка «Статус», пункт «Активные», fetch по статусам', async () => {
    const { wrapper } = await mountAt('/')
    await flushPromises()
    expect(wrapper.find('[data-testid="task-table"]').exists()).toBe(true)
    const statusBtn = wrapper.find('[data-testid="tab-status"]')
    expect(statusBtn.classes()).toContain('active')
    // таб показывает выбранный пункт — но по дефолту активная точка «Статус»:
    // пункт вкладки = «Активные» (selected)
    expect(tasksMock.setTab).toHaveBeenCalledWith('status', 'active')
    expect(tasksMock.filters.statuses).toEqual(['new', 'in_progress'])
    expect(tasksMock.filters.plan).toBe('')
  })

  it('deep-link ?tab=status&status=completed: пункт вкладки «Выполненные», fetch', async () => {
    const { wrapper } = await mountAt('/?tab=status&status=completed')
    await flushPromises()
    expect(tasksMock.setTab).toHaveBeenCalledWith('status', 'completed')
    expect(tasksMock.filters.statuses).toEqual(['completed'])
    // табу открываем → пункт «Выполненные» выделен (selected)
    const dd = wrapper.find('[data-testid="tab-status"]')
    await dd.trigger('click')
    await flushPromises()
    const menu = dd.element.parentElement
    const selected = Array.from(
      menu.querySelectorAll('[role="menuitemradio"]')
    ).find((b) => b.classList.contains('selected'))
    expect(selected).toBeTruthy()
    expect(selected.textContent).toContain('Выполненные')
  })

  it('deep-link ?tab=plan&plan=tomorrow: «План», пункт «Завтра», БЕЗ status, fetch с plan', async () => {
    const { wrapper } = await mountAt('/?tab=plan&plan=tomorrow')
    await flushPromises()
    expect(tasksMock.setTab).toHaveBeenCalledWith('plan', 'tomorrow')
    expect(tasksMock.filters.plan).toBe('tomorrow')
    expect(tasksMock.filters.statuses).toEqual([])
    const dd = wrapper.find('[data-testid="tab-plan"]')
    await dd.trigger('click')
    await flushPromises()
    const menu = dd.element.parentElement
    const selected = Array.from(
      menu.querySelectorAll('[role="menuitemradio"]')
    ).find((b) => b.classList.contains('selected'))
    expect(selected).toBeTruthy()
    expect(selected.textContent).toContain('Завтра')
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

  it('клик «План» + пункт (default «Сегодня») → URL ?tab=plan (без ?status=), статусы выключены', async () => {
    const { router, wrapper } = await mountAt('/?tab=status&status=completed')
    await flushPromises()
    const planBtn = wrapper.find('[data-testid="tab-plan"]')
    await planBtn.trigger('click') // открыть меню
    await flushPromises()
    const menu = planBtn.element.parentElement
    const today = Array.from(menu.querySelectorAll('[role="menuitemradio"]')).find((o) => o.textContent.includes('Сегодня'))
    await today.click()
    await flushPromises()
    expect(router.currentRoute.value.query.tab).toBe('plan')
    expect(router.currentRoute.value.query.status).toBeUndefined()
    expect(tasksMock.setTab).toHaveBeenLastCalledWith('plan', 'today')
    expect(tasksMock.filters.statuses).toEqual([])
    expect(tasksMock.filters.plan).toBe('today')
  })

  it('клик «Статус» из «План» + пункт «Активные» → URL ?tab=status (без ?plan=), статусы включены', async () => {
    const { router, wrapper } = await mountAt('/?tab=plan&plan=week')
    await flushPromises()
    const statusTab = wrapper.find('[data-testid="tab-status"]')
    await statusTab.trigger('click') // открыть меню
    await flushPromises()
    const menu = statusTab.element.parentElement
    const active = Array.from(menu.querySelectorAll('[role="menuitemradio"]')).find((o) => o.textContent.includes('Активные'))
    await active.click()
    await flushPromises()
    expect(router.currentRoute.value.query.tab).toBe('status')
    expect(router.currentRoute.value.query.status).toBe('active')
    expect(router.currentRoute.value.query.plan).toBeUndefined()
    expect(tasksMock.setTab).toHaveBeenLastCalledWith('status', 'active')
    expect(tasksMock.filters.statuses).toEqual(['new', 'in_progress'])
    expect(tasksMock.filters.plan).toBe('')
  })

  it('выбор пункта в «Статус»: «Бэклог» → ?status=backlog, fetch по статусам', async () => {
    const { router, wrapper } = await mountAt('/?tab=status&status=active')
    await flushPromises()
    const dd = wrapper.find('[data-testid="tab-status"]')
    await dd.trigger('click')
    await flushPromises()
    const menu = dd.element.parentElement
    const opt = Array.from(menu.querySelectorAll('[role="menuitemradio"]')).find((o) => o.textContent.includes('Бэклог'))
    await opt.click()
    await flushPromises()
    expect(tasksMock.setStatusFilter).toHaveBeenLastCalledWith('backlog')
    expect(tasksMock.filters.statuses).toEqual(['backlog'])
    expect(tasksMock.fetchTasks).toHaveBeenCalled()
    expect(router.currentRoute.value.query.status).toBe('backlog')
    expect(router.currentRoute.value.query.plan).toBeUndefined()
  })

  it('28e: пункт «План» → «Без плана» → ?plan=none', async () => {
    const { router, wrapper } = await mountAt('/?tab=plan&plan=today')
    await flushPromises()
    const dd = wrapper.find('[data-testid="tab-plan"]')
    await dd.trigger('click')
    await flushPromises()
    const menu = dd.element.parentElement
    const opt = Array.from(menu.querySelectorAll('[role="menuitemradio"]')).find((o) => o.textContent.includes('Без плана'))
    await opt.click()
    await flushPromises()
    expect(tasksMock.setPlanFilter).toHaveBeenLastCalledWith('none')
    expect(tasksMock.filters.plan).toBe('none')
    expect(router.currentRoute.value.query.plan).toBe('none')
  })

  it('28e: пункт «План» → «Все» → без фильтра (plan не в params)', async () => {
    const { router, wrapper } = await mountAt('/?tab=plan&plan=today')
    await flushPromises()
    const dd = wrapper.find('[data-testid="tab-plan"]')
    await dd.trigger('click')
    await flushPromises()
    const menu = dd.element.parentElement
    const opt = Array.from(menu.querySelectorAll('[role="menuitemradio"]')).find((o) => o.textContent.trim().replace(/^\S+\s+/, '') === 'Все')
    await opt.click()
    await flushPromises()
    expect(tasksMock.setPlanFilter).toHaveBeenLastCalledWith('all')
    expect(tasksMock.filters.plan).toBe('')
    expect(router.currentRoute.value.query.plan).toBe('all')
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
