// DashboardVersionBadge.spec.js — v0.27, шаг 8: версия сборки в шапке.
// Публичный GET /api/version → бейдж «v<версия>» рядом «● Онлайн/Офлайн»;
// commit + built — в tooltip (title). Ошибка/без version — бейдж скрыт.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'

const tasksMock = {
  setTab: vi.fn(),
  fetchTasks: vi.fn(async () => {}),
  filters: { project: '', statuses: ['new', 'in_progress'], page: 1, per_page: 50 }
}
const wsMock = { connected: true, events: [], connect: vi.fn(), disconnect: vi.fn() }

// Мок API — vi.hoisted: vi.mock-фабрика поднимается выше переменных, поэтому
// shared-объект живёт в hoisted-блоке; тесты меняют version.value.
const { apiGet, version } = vi.hoisted(() => {
  const getVersion = { value: { version: '0.27.0', commit: 'f1e2d3c', built: '2026-09-18_15:31:07' } }
  const get = async (url) => {
    if (url === '/version') return { data: getVersion.value }
    if (url === '/projects') return { data: [] }
    if (url === '/inbox') return { data: { total: 0, items: [] } }
    return { data: { tasks: [], total: 0 } }
  }
  return { apiGet: vi.fn(get), version: getVersion }
})

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
vi.mock('@/views/InboxView.vue', () => ({ default: { template: '<div/>' } }))
vi.mock('@/components/TaskTable.vue', () => ({ default: { template: '<div/>' } }))
vi.mock('@/views/ProjectsView.vue', () => ({ default: { template: '<div/>' } }))
vi.mock('@/stores/epics', () => ({
  useEpicsStore: () => ({ fetchEpics: vi.fn(), currentProjectId: null })
}))
vi.mock('@/stores/projects', () => ({
  useProjectsStore: () => ({ projects: [], fetchProjects: vi.fn() })
}))
vi.mock('@/stores/inbox', () => ({
  useInboxStore: () => ({ unreadCount: 0, fetchUnreadCount: vi.fn(), fetchItems: vi.fn() })
}))

import DashboardView from '@/views/DashboardView.vue'

async function mountDashboard() {
  setActivePinia(createPinia())
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', name: 'dashboard', component: DashboardView }]
  })
  await router.push('/')
  await router.isReady()
  const wrapper = mount(DashboardView, { global: { plugins: [router] } })
  await flushPromises()
  return wrapper
}

describe('DashboardView — версия в шапке (v0.27, шаг 8)', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    version.value = { version: '0.27.0', commit: 'f1e2d3c', built: '2026-09-18_15:31:07' }
  })

  it('бейдж с версией рядом со статусом соединения; commit+built в tooltip', async () => {
    const wrapper = await mountDashboard()
    const badge = wrapper.find('.version-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('v0.27.0')
    expect(badge.attributes('title')).toContain('commit: f1e2d3c')
    expect(badge.attributes('title')).toContain('сборка: 2026-09-18_15:31:07')
    // рядом со строкой «● Онлайн/Офлайн» — в .header-right
    expect(wrapper.find('.header-right .version-badge').exists()).toBe(true)
  })

  it('значения «как есть»: dev/none показывается без подавления', async () => {
    version.value = { version: 'dev', commit: 'none', built: 'unknown' }
    const wrapper = await mountDashboard()
    const badge = wrapper.find('.version-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('vdev')
    expect(badge.attributes('title')).toContain('commit: none')
  })

  it('ответ без version — бейдж скрыт (не ломает шапку)', async () => {
    version.value = {}
    const wrapper = await mountDashboard()
    expect(wrapper.find('.version-badge').exists()).toBe(false)
  })

  it('ошибка запроса — бейдж скрыт', async () => {
    version.value = null
    apiGet.mockRejectedValueOnce(new Error('network 500'))
    const wrapper = await mountDashboard()
    expect(wrapper.find('.version-badge').exists()).toBe(false)
  })
})
