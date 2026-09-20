// v0.28/28e: «Назад» на странице задачи возвращает ВСЁ исходное состояние
// вкладки (вкладка + её значение + фильтры) — не сбрасывает к дефолтам
// (решение владельца 2026-09-20: «моя выбор должен быть запомнен»).
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { reactive } from 'vue'

const routeMock = { params: { id: '9' }, query: {} }
const routerMock = { push: vi.fn(), replace: vi.fn() }
vi.mock('@/api/client', () => ({
  default: { get: vi.fn(), patch: vi.fn(), post: vi.fn(), delete: vi.fn() }
}))
vi.mock('vue-router', () => ({
  useRouter: () => routerMock,
  useRoute: () => routeMock
}))
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: vi.fn() }) }))

const tasksHolder = reactive({ task: null })
vi.mock('@/stores/tasks', () => ({
  useTasksStore: () => ({
    get currentTask() { return tasksHolder.task },
    get currentComments() { return [] },
    fetchTask: vi.fn(async () => tasksHolder.task),
    updateTask: vi.fn().mockResolvedValue({ task: {} }),
    markAsRead: vi.fn(),
    fetchTaskInbox: vi.fn().mockResolvedValue([])
  })
}))
vi.mock('@/stores/projects', () => ({
  useProjectsStore: () => ({
    projects: [], projectByName: () => null, fetchProjects: vi.fn().mockResolvedValue(undefined)
  })
}))
vi.mock('@/stores/epics', () => ({
  useEpicsStore: () => ({ epics: [], fetchEpics: vi.fn().mockResolvedValue(undefined) })
}))
vi.mock('@/stores/persons', () => ({
  usePersonsStore: () => ({ list: [], persons: [], fetchPersons: vi.fn().mockResolvedValue(undefined) })
}))
vi.mock('@/components/CommentList.vue', () => ({ default: { template: '<div></div>' } }))
vi.mock('@/components/ReplyForm.vue', () => ({ default: { template: '<div></div>' } }))
vi.mock('@/components/WorkflowButtons.vue', () => ({ default: { template: '<div></div>' } }))

import TaskDetailView from '@/views/TaskDetailView.vue'

const TASK = {
  id: 9, subject: 'тест', project: null, status: 'new', priority: 'medium',
  type: 'feature', assignee: null, epic_id: null, from_email: 'a@b.c',
  created_at: '2026-08-29T10:00:00Z', body_html: null, body_text: 'desc',
  due_date: null, due_source: null, ai_due_date: null, due_ai_pending: false
}

describe('v0.28/28e — «Назад» возвращает сохранённое состояние вкладки', () => {
  let push
  beforeEach(() => {
    vi.resetAllMocks()
    routeMock.query = {}
    tasksHolder.task = { ...TASK }
    push = routerMock.push
  })

  it('?tab=plan&plan=tomorrow&project=X → «Назад» гонит всё это обратно (не дефолты)', async () => {
    routeMock.query = { tab: 'plan', plan: 'tomorrow', project: 'Деск X' }
    const pinia = createPinia()
    setActivePinia(pinia)
    const wrapper = mount(TaskDetailView, { global: { plugins: [pinia] } })
    await flushPromises()
    wrapper.find('header .p-button').trigger('click')
    await flushPromises()
    expect(push).toHaveBeenCalledWith({
      path: '/',
      query: { tab: 'plan', plan: 'tomorrow', project: 'Деск X' }
    })
  })

  it('без сохранённого query → чистый корень', async () => {
    routeMock.query = {}
    const pinia = createPinia()
    setActivePinia(pinia)
    const wrapper = mount(TaskDetailView, { global: { plugins: [pinia] } })
    await flushPromises()
    wrapper.find('header .p-button').trigger('click')
    await flushPromises()
    expect(push).toHaveBeenCalledWith({ path: '/', query: {} })
  })
})
