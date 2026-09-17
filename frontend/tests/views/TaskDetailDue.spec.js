// v0.25, шаг 7c: детальная задача — поле «Срок» + решение по AI-предложению.
//   accept  → PATCH {due_ai_resolve:"accept"}  (due_date := ai_due_date, source=ai)
//   reject  → PATCH {due_ai_resolve:"reject"}  (pending снят, срок не ставится)
//   edit    → AI-значение предзаполнено в DatePicker (PATCH не шлём — только у edit)
//   смена даты в поле → PATCH {due_date:"YYYY-MM-DD"|"null"} (ручное = manual, backend)
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { reactive } from 'vue'

vi.mock('@/api/client', () => ({
  default: { get: vi.fn(), patch: vi.fn(), post: vi.fn(), delete: vi.fn() }
}))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  useRoute: () => ({ params: { id: '9' }, query: {} })
}))
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: vi.fn() }) }))

const tasksHolder = reactive({
  task: null,
  updateTask: vi.fn().mockResolvedValue({ task: {} })
})
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
const epicsHold = { epics: [], fetchEpics: vi.fn().mockResolvedValue(undefined) }
vi.mock('@/stores/epics', () => ({
  useEpicsStore: () => ({ epics: epicsHold.epics, fetchEpics: (id) => epicsHold.fetchEpics(id) })
}))
vi.mock('@/stores/persons', () => ({
  usePersonsStore: () => ({
    list: [],
    persons: [],
    fetchPersons: vi.fn().mockResolvedValue(undefined)
  })
}))
vi.mock('@/components/CommentList.vue', () => ({ default: { template: '<div>comments</div>' } }))
vi.mock('@/components/ReplyForm.vue', () => ({ default: { template: '<div>reply</div>' } }))
vi.mock('@/components/WorkflowButtons.vue', () => ({ default: { template: '<div>wf</div>' } }))

import TaskDetailView from '@/views/TaskDetailView.vue'

const TASK = {
  id: 9,
  subject: 'AI-срок',
  project: 'Деск X',
  status: 'new',
  priority: 'medium',
  type: 'feature',
  assignee: null,
  epic_id: null,
  from_email: 'a@b.c',
  created_at: '2026-08-29T10:00:00Z',
  body_html: null,
  body_text: 'описание',
  due_date: null,
  due_source: null,
  ai_due_date: '2027-01-15',
  due_ai_pending: true
}

function mountView(task) {
  // setState ПЕРЕД mount (watch(() => store.currentTask) — смена объекта после
  // mount не перерисовывает template, только syncFields — поэтому каждый
  // сценарий монтирует СВОЙ инстанс со своим task).
  tasksHolder.task = task ? { ...task } : { ...TASK }
  tasksHolder.updateTask = vi.fn().mockResolvedValue({ task: tasksHolder.task })
  epicsHold.fetchEpics = vi.fn().mockResolvedValue(undefined)
  const pinia = createPinia()
  setActivePinia(pinia)
  return mount(TaskDetailView, { global: { plugins: [pinia] } })
}

describe('TaskDetailView — срок + AI-решение (7c)', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    epicsHold.epics = []
  })

  it('pending-предложение: бейдж с ai_due_date + 3 кнопки (принять/изменить/отклонить)', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('[data-testid="due-ai-row"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="due-ai-suggestion"]').text()).toContain('2027-01-15')
    expect(wrapper.find('[data-testid="due-ai-accept"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="due-ai-edit"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="due-ai-reject"]').exists()).toBe(true)
    // подсветка pending строки поля
    expect(wrapper.find('.due-field--pending').exists()).toBe(true)
  })

  it('без pending: AI-кнопки не показываются', async () => {
    const wrapper = mountView({ ...TASK, due_ai_pending: null })
    await flushPromises()
    expect(wrapper.find('[data-testid="due-ai-row"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="due-ai-accept"]').exists()).toBe(false)
  })

  it('«Принять» → PATCH {due_ai_resolve:"accept"}; датпикер берёт новый срок из ответа', async () => {
    tasksHolder.updateTask = vi.fn().mockResolvedValue({ task: { ...TASK, due_date: '2027-01-15', due_ai_pending: false } })
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('[data-testid="due-ai-accept"]').trigger('click')
    await flushPromises()
    expect(tasksHolder.updateTask).toHaveBeenCalledWith('9', { due_ai_resolve: 'accept' })
  })

  it('«Отклонить» → PATCH {due_ai_resolve:"reject"}', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('[data-testid="due-ai-reject"]').trigger('click')
    await flushPromises()
    expect(tasksHolder.updateTask).toHaveBeenCalledWith('9', { due_ai_resolve: 'reject' })
  })

  it('«Изменить» — предзаполняет AI-значение, PATCH не шлёт', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.find('[data-testid="due-ai-edit"]').trigger('click')
    await flushPromises()
    expect(tasksHolder.updateTask).not.toHaveBeenCalled()
    const d = wrapper.vm.dueDateValue
    expect(d).toBeInstanceOf(Date)
    expect(d.getFullYear()).toBe(2027)
    expect(d.getMonth()).toBe(0)
    expect(d.getDate()).toBe(15)
  })

  it('ручная запись даты → PATCH {due_date}; null (снять срок) → {due_date: null}', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.vm.saveDueDate(new Date(2027, 1, 20))
    expect(tasksHolder.updateTask).toHaveBeenLastCalledWith('9', { due_date: '2027-02-20' })

    tasksHolder.updateTask = vi.fn().mockResolvedValue({ task: { due_date: null } })
    await wrapper.vm.saveDueDate(null)
    expect(tasksHolder.updateTask).toHaveBeenLastCalledWith('9', { due_date: null })
  })

  it('due_source отображается: AI → «срок: AI», manual → «срок: вручную»', async () => {
    const wrapper = mountView({ ...TASK, due_date: '2027-01-15', due_source: 'ai', due_ai_pending: false })
    await flushPromises()
    expect(wrapper.find('[data-testid="due-source-ai"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="due-source-manual"]').exists()).toBe(false)

    const wrapper2 = mountView({ ...TASK, due_date: '2027-01-15', due_source: 'manual', due_ai_pending: false })
    await flushPromises()
    expect(wrapper2.find('[data-testid="due-source-manual"]').exists()).toBe(true)
  })
})
