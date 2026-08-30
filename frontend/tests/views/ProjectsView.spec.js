import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'

// Общий объект роутера: заменяемо, чтобы assert'ить параметры replace()
const routerMock = { path: '/tasks' }

// Моки ДО импорта компонента, общий объект на все вызовы (паттерн CreateTaskDialog.spec.js)
const tasksMock = {
  setFilter: vi.fn(),
  filters: { project: '', epic_id: 7, statuses: ['new', 'in_progress'], page: 1, per_page: 50 },
  projects: []
}
const projectsMock = {
  projects: [
    { id: 1, name: 'Лидер Спорт', description: 'спорт', archived: false, created_at: '2026-08-29T10:00:00Z' }
  ],
  loading: false,
  error: '',
  fetchProjects: vi.fn(),
  createProject: vi.fn(),
  renameProject: vi.fn(),
  archiveProject: vi.fn(),
  unarchiveProject: vi.fn()
}

vi.mock('@/stores/tasks', () => ({ useTasksStore: () => tasksMock }))
vi.mock('@/stores/projects', () => ({ useProjectsStore: () => projectsMock }))
vi.mock('@/stores/epics', () => ({ useEpicsStore: () => ({ fetchEpics: vi.fn() }) }))
vi.mock('@/api/client', () => ({
  default: { get: vi.fn(), patch: vi.fn(), post: vi.fn() }
}))
vi.mock('vue-router', () => ({
  useRouter: () => { routerMock.push = vi.fn(); routerMock.replace = vi.fn(); return routerMock },
  useRoute: () => ({ params: {}, query: {} })
}))
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: vi.fn() }) }))

import ProjectsView from '@/views/ProjectsView.vue'

describe('ProjectsView — «К задачам» (шаг 6)', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    tasksMock.setFilter.mockReset()
  })

  it('кнопка «К задачам»: фильтр по проекту + переход на вкладку «Активные» по URL', async () => {
    const wrapper = mount(ProjectsView, { global: { plugins: [createPinia()] } })
    await flushPromises()

    const btn = wrapper
      .findAll('.p-button')
      .find((b) => b.text().includes('К задачам'))
    expect(btn).toBeTruthy()

    await btn.trigger('click')
    // фильтр — в сторе (значение-строка, не объект)
    expect(tasksMock.setFilter).toHaveBeenCalledWith('project', 'Лидер Спорт')
    // чужой модуль сброшен
    expect(tasksMock.filters.epic_id).toBe('')
    // переход — именно на вкладку «active» с проектом в URL (deep-link)
    expect(routerMock.replace).toHaveBeenCalledWith({
      query: { tab: 'active', project: 'Лидер Спорт' }
    })
  })
})
