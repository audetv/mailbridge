import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { reactive } from 'vue'

// Общий управляемый мок websocket-стора. events — РЕАКТИВНЫЙ МАССИВ (реальный стор
// через pinia возвращает развёрнутый ref-array, так что wsStore.events — обычный массив).
const wsEvents = reactive([])
const wsMock = { connected: true, events: wsEvents, connect: vi.fn(), disconnect: vi.fn(), markAsRead: vi.fn() }

vi.mock('@/stores/websocket', () => ({
  useWebSocket: () => wsMock
}))

vi.mock('@/api/client', () => ({
  default: { get: vi.fn(), patch: vi.fn(), post: vi.fn(), del: vi.fn(), delete: vi.fn() }
}))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  useRoute: () => ({ params: {}, query: {} })
}))
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: vi.fn() }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ token: 't', logout: vi.fn() }) }))
vi.mock('@/stores/theme', () => ({ useThemeStore: () => ({ isDark: false, toggleTheme: vi.fn() }) }))
vi.mock('@/stores/inbox', () => ({
  useInboxStore: () => ({ unreadCount: 0, items: [], fetchUnreadCount: vi.fn(), fetchItems: vi.fn() })
}))
vi.mock('@/stores/projects', () => ({
  useProjectsStore: () => ({ projects: [], error: '', fetchProjects: vi.fn() })
}))
vi.mock('@/views/InboxView.vue', () => ({ default: { template: '<div>inbox</div>' } }))
vi.mock('@/views/ProjectsView.vue', () => ({ default: { template: '<div>projects</div>' } }))
vi.mock('@/components/FilterBar.vue', () => ({ default: { template: '<div>filter</div>' } }))
vi.mock('@/components/TaskTable.vue', () => ({ default: { template: '<div>tasks</div>' } }))
vi.mock('@/components/TabBar.vue', () => ({ default: { template: '<div>tabs</div>' } }))

import apiClient from '@/api/client'
import { useEpicsStore } from '@/stores/epics'
import DashboardView from '@/views/DashboardView.vue'

describe('DashboardView — WS epic_* ветка', () => {
  let pinia
  let epics

  beforeEach(() => {
    vi.resetAllMocks()
    wsEvents.splice(0, wsEvents.length)
    wsMock.connected = true
    pinia = createPinia()
    setActivePinia(pinia)
    vi.mocked(apiClient.get).mockResolvedValue({ data: { total: 0 } })
    epics = useEpicsStore(pinia)
  })

  it('epic_updated: перечитываем список модулей для активного проекта', async () => {
    const fetchEpicsSpy = vi.spyOn(epics, 'fetchEpics').mockResolvedValue(undefined)
    epics.currentProjectId = 3

    const wrapper = mount(DashboardView, { global: { plugins: [pinia] } })
    await flushPromises()

    // Эмулируем arrival события из WS: watcher в DashboardView следит за wsStore.events (length)
    wsEvents.push({ type: 'epic_updated', message: 'Модуль обновлён' })
    await flushPromises()
    await flushPromises()

    expect(fetchEpicsSpy).toHaveBeenCalledTimes(1)
    expect(fetchEpicsSpy).toHaveBeenCalledWith(3)
    expect(wrapper.find('.dashboard').exists()).toBe(true)
  })

  it('epic_created и epic_deleted тоже перечитывают список модулей (по одному событию на цикл воркера)', async () => {
    const fetchEpicsSpy = vi.spyOn(epics, 'fetchEpics').mockResolvedValue(undefined)
    epics.currentProjectId = 9

    mount(DashboardView, { global: { plugins: [pinia] } })
    await flushPromises()

    // воркер DashboardView обрабатывает только ПОСЛЕДНЕЕ событие за цикл —
    // проверяем каждый тип отдельно
    wsEvents.push({ type: 'epic_created', message: 'x' })
    await flushPromises()
    wsEvents.splice(0, wsEvents.length)
    await flushPromises()
    wsEvents.push({ type: 'epic_deleted', message: 'x' })
    await flushPromises()

    expect(fetchEpicsSpy).toHaveBeenCalledTimes(2)
    expect(fetchEpicsSpy).toHaveBeenNthCalledWith(1, 9)
    expect(fetchEpicsSpy).toHaveBeenNthCalledWith(2, 9)
  })

  it('epic_* без активного проекта: перечитываний нет', async () => {
    const fetchEpicsSpy = vi.spyOn(epics, 'fetchEpics').mockResolvedValue(undefined)
    epics.currentProjectId = null

    mount(DashboardView, { global: { plugins: [pinia] } })
    await flushPromises()

    wsEvents.push({ type: 'epic_updated', message: 'x' })
    await flushPromises()

    expect(fetchEpicsSpy).not.toHaveBeenCalled()
  })
})
