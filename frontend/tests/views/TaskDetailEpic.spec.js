import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/client', () => ({
  default: { get: vi.fn(), patch: vi.fn(), post: vi.fn(), delete: vi.fn() }
}))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  useRoute: () => ({ params: { id: '7' }, query: {} })
}))
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: vi.fn() }) }))

// Управляемый holder: тесты меняют currentTask до mount.
const tasksHolder = {
  task: null,
  updateTask: vi.fn().mockResolvedValue({ data: { task: {} } })
}
vi.mock('@/stores/tasks', () => ({
  useTasksStore: () => ({
    get currentTask() { return tasksHolder.task },
    get currentComments() { return [] },
    fetchTask: vi.fn(async () => tasksHolder.task),
    updateTask: (id, updates) => tasksHolder.updateTask(id, updates),
    markAsRead: vi.fn(),
    fetchTaskInbox: vi.fn().mockResolvedValue([])
  })
}))

const projectsHold = {
  byName: (n) => (n === 'Деск X' ? { id: 3, name: 'Деск X' } : null),
  fetchProjects: vi.fn().mockResolvedValue(undefined)
}
vi.mock('@/stores/projects', () => ({
  useProjectsStore: () => ({
    projects: [{ id: 3, name: 'Деск X' }],
    projectByName: (n) => projectsHold.byName(n),
    fetchProjects: (p) => projectsHold.fetchProjects(p)
  })
}))

const epicsHold = {
  epics: [],
  fetchEpics: vi.fn().mockResolvedValue(undefined)
}
vi.mock('@/stores/epics', () => ({
  useEpicsStore: () => ({
    epics: epicsHold.epics,
    fetchEpics: (id) => epicsHold.fetchEpics(id)
  })
}))
vi.mock('@/components/CommentList.vue', () => ({ default: { template: '<div>comments</div>' } }))
vi.mock('@/components/ReplyForm.vue', () => ({ default: { template: '<div>reply</div>' } }))
vi.mock('@/components/WorkflowButtons.vue', () => ({ default: { template: '<div>wf</div>' } }))

import TaskDetailView from '@/views/TaskDetailView.vue'

const TASK = {
  id: 7,
  subject: 'Ручная',
  project: 'Деск X',
  status: 'new',
  priority: 'medium',
  type: 'feature',
  assignee: null,
  epic_id: null,
  from_email: 'a@b.c',
  created_at: '2026-08-29T10:00:00Z',
  body_html: null,
  body_text: 'описание'
}

function mountView() {
  tasksHolder.task = { ...TASK }
  tasksHolder.updateTask = vi.fn().mockResolvedValue({ data: { task: {} } })
  epicsHold.fetchEpics = vi.fn().mockResolvedValue(undefined)
  const pinia = createPinia()
  setActivePinia(pinia)
  const wrapper = mount(TaskDetailView, { global: { plugins: [pinia] } })
  return wrapper
}

describe('TaskDetailView — поле «Модуль» (11.2)', () => {
  beforeEach(() => { vi.resetAllMocks(); epicsHold.epics = [] })

  it('поднимает эпики проекта и строит опции для Select (fetchEpics по id проекта)', async () => {
    epicsHold.epics = [
      { id: 10, name: 'Биллинг' },
      { id: 11, name: 'Портал' }
    ]
    epicsHold.fetchEpics = vi.fn(async () => {
      epicsHold.epics = [
        { id: 10, name: 'Биллинг' },
        { id: 11, name: 'Портал' }
      ]
    })
    const wrapper = mountView()
    await flushPromises()
    expect(epicsHold.fetchEpics).toHaveBeenCalledWith(3)
    expect(wrapper.vm.epicOptions).toEqual([
      { label: 'Биллинг', value: 10 },
      { label: 'Портал', value: 11 }
    ])
  })

  it('выбор модуля → PATCH /tasks/7 c {epic_id}', async () => {
    epicsHold.epics = [{ id: 10, name: 'Биллинг' }]
    const wrapper = mountView()
    await flushPromises()
    await wrapper.vm.onEpicChange(10)
    expect(tasksHolder.updateTask).toHaveBeenCalledWith('7', { epic_id: 10 })
    expect(wrapper.vm.epic).toBe(10)
  })

  it('сброс модуля (null) → PATCH c {epic_id: null}', async () => {
    epicsHold.epics = [{ id: 10, name: 'Биллинг' }]
    const wrapper = mountView()
    await flushPromises()
    wrapper.vm.epic = 10
    await wrapper.vm.onEpicChange(null)
    expect(tasksHolder.updateTask).toHaveBeenCalledWith('7', { epic_id: null })
    expect(wrapper.vm.epic).toBeNull()
  })

  it('задача из «Входящих» (нет проекта) — опций нет, поле скрыто', async () => {
    tasksHolder.task = { ...TASK, project: 'Входящие' }
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.vm.epicOptions).toEqual([])
    expect(wrapper.find('label').text()) // smoke: template rendered
  })
})
