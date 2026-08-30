import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'

// URL — source of truth для вкладки (шаг 13.2, баг 1): «К задачам» из
// «Проекты» и deep-link «?tab=active&project=…» должны реально переключать
// вкладку. До фикса этого не было (watch на URL не было) и 39 UI-тестов это
// не ловили — переходы табу здесь, по URL, проверяются впервые.
const tasksMock = {
  setFilter: vi.fn((k, v) => { tasksMock.filters[k] = v }),
  setStatuses: vi.fn((s) => { tasksMock.filters.statuses = s }),
  fetchTasks: vi.fn(async () => {}),
  filters: { project: '', epic_id: '', statuses: ['new', 'in_progress'], page: 1, per_page: 50 }
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
import { useEpicsStore } from '@/stores/epics'

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

describe('DashboardView — переход вкладок по URL (шаг 13.2)', () => {
  beforeEach(async () => {
    vi.restoreAllMocks()
    tasksMock.filters = { project: '', epic_id: '', statuses: ['new', 'in_progress'], page: 1, per_page: 50 }
    localStorage.clear()
  })

  it('deep-link ?tab=active&project=ТРК: сразу «Активные» + модули проекта + фильтр', async () => {
    const { wrapper } = await mountAt('/?tab=active&project=ТРК')
    await flushPromises()

    expect(wrapper.find('[data-testid="task-table"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="inbox-view"]').exists()).toBe(false)
    // ?project=… прилеплен к фильтру
    expect(tasksMock.filters.project).toBe('ТРК')
    // модули проекта загружены (эмуляция /projects/{id}/epics → опции в FilterBar)
    console.log('API-запросы:', apiGet.mock.calls.map((c) => c[0]))
    const epics = useEpicsStore()
    expect(epics.epics).toEqual([expect.objectContaining({ id: 11, name: 'E2E-модуль' })])
    // статусы активной вкладки
    expect(tasksMock.filters.statuses).toEqual(['new', 'in_progress'])
  })

  it('вкладка «Проекты»: проект-опции видны + переход на «Активные» по URL переключает вкладку', async () => {
    const { router, wrapper } = await mountAt('/?tab=projects')
    await flushPromises()

    // вкладки (TabBar) на месте
    expect(wrapper.findAll('.tab-bar button').length).toBe(6)
    expect(tasksMock.fetchTasks).not.toHaveBeenCalled()

    // имитируем ProjectsView.goToTasks(p): replace на «Активные» + фильтр
    tasksMock.filters.project = 'ТРК'
    await router.replace({ query: { tab: 'active', project: 'ТРК' } })
    await flushPromises()

    // вкладка реально переключилась (watch на route.query.tab) — баг 1 закрыт
    expect(wrapper.find('[data-testid="task-table"]').exists()).toBe(true)
    // applyTab -> setStatuses (не fetchTasks: он внутри TaskTable)
    expect(tasksMock.setStatuses).toHaveBeenCalledWith(['new', 'in_progress'])
  })

  it('возврат на «Проекты» через клик таба не теряет проект-фильтр (в store)', async () => {
    const { router, wrapper } = await mountAt('/?tab=active&project=ТРК')
    await flushPromises()
    expect(wrapper.find('[data-testid="task-table"]').exists()).toBe(true)

    // клик по вкладке «Проекты» — как в TabBar (кнопка с текстом)
    const tabButtons = wrapper.findAll('.tab-bar button')
    expect(tabButtons.length).toBeGreaterThan(0)
    const projectTab = tabButtons.find((t) => t.text().trim() === 'Проекты')
    expect(projectTab).toBeTruthy()
    await projectTab.trigger('click')
    await flushPromises()

    // проект-фильтр остаётся доступным (в store, на будущее переход)
    expect(tasksMock.filters.project).toBe('ТРК')
    // URL чистится: на вкладке «Проекты» ?project не нужен
    expect(router.currentRoute.value.query.project).toBeUndefined()
    // ProjectsView смонтировался (заменён tasks-table)
    expect(wrapper.find('[data-testid="task-table"]').exists()).toBe(false)
  })

  it('закрытая вкладка: неизвестный tab в URL → fallback на «Активные»', async () => {
    const { wrapper } = await mountAt('/?tab=???')
    await flushPromises()
    expect(wrapper.find('[data-testid="task-table"]').exists()).toBe(true)
  })
})
