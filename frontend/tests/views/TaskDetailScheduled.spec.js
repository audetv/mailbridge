// v0.28, шаг 28c: детальная задача — поле «План» (scheduled_date).
//   «когда делаю» — отдельное поле от «Срок» (due_date: дедлайн/обещание).
//   Установка/смена → PATCH {scheduled_date: "YYYY-MM-DD"}; снять → null.
//   due НЕ трогается (независимые поля — контракт 28a).
//   Тон другой, не «дедлайн»: план сегодня — info (синий), дедлайн сегодня — warn (жёлтый).
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
vi.mock('@/stores/projects', () => ({
  useProjectsStore: () => ({
    projects: [{ id: 3, name: 'Деск X' }],
    projectByName: () => null,
    fetchProjects: vi.fn().mockResolvedValue(undefined)
  })
}))
vi.mock('@/stores/epics', () => ({
  useEpicsStore: () => ({ epics: [], fetchEpics: vi.fn().mockResolvedValue(undefined) })
}))
vi.mock('@/stores/persons', () => ({
  usePersonsStore: () => ({
    list: [], persons: [],
    fetchPersons: vi.fn().mockResolvedValue(undefined),
    setTaskRoles: vi.fn().mockResolvedValue({})
  })
}))
vi.mock('@/components/CommentList.vue', () => ({ default: { template: '<div>comments</div>' } }))
vi.mock('@/components/ReplyForm.vue', () => ({ default: { template: '<div>reply</div>' } }))
vi.mock('@/components/WorkflowButtons.vue', () => ({ default: { template: '<div>wf</div>' } }))

import TaskDetailView from '@/views/TaskDetailView.vue'

const TASK = {
  id: 9,
  subject: 'Две даты',
  project: 'Деск X',
  status: 'new',
  priority: 'medium',
  type: 'feature',
  assignee: null,
  epic_id: null,
  from_email: 'a@b.c',
  created_at: '2026-09-15T10:00:00Z',
  body_html: null,
  body_text: 'описание',
  due_date: '2026-09-15',
  due_source: 'manual',
  ai_due_date: null,
  due_ai_pending: false,
  scheduled_date: '2026-09-19'
}

function mountView(task) {
  tasksHolder.task = task ? { ...task } : { ...TASK }
  tasksHolder.updateTask = vi.fn().mockResolvedValue({ task: tasksHolder.task })
  const pinia = createPinia()
  setActivePinia(pinia)
  return mount(TaskDetailView, { global: { plugins: [pinia] } })
}

describe('TaskDetailView — «План» (scheduled_date, 28c)', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('поле «План» с DatePicker; дата из задачи предзаполнена', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.find('[data-testid="scheduled-label"]').text()).toBe('План')
    expect(wrapper.find('[data-testid="scheduled-datepicker"]').exists()).toBe(true)
    // предзаполнение: канон 2026-09-19 → локальный Date
    const v = wrapper.vm.scheduledDateValue
    expect(v).toBeInstanceOf(Date)
    expect(v.getFullYear()).toBe(2026)
    expect(v.getMonth()).toBe(8)
    expect(v.getDate()).toBe(19)
  })

  it('установка даты → PATCH {scheduled_date}; due при этом НЕ трогается', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.vm.saveScheduledDate(new Date(2027, 1, 20))
    expect(tasksHolder.updateTask).toHaveBeenLastCalledWith('9', { scheduled_date: '2027-02-20' })
    // только scheduled_date — ни due_date, ни due_ai_resolve в PATCH нет
    const patch = tasksHolder.updateTask.mock.calls.at(-1)[1]
    expect(Object.keys(patch)).toEqual(['scheduled_date'])
  })

  it('null (снять план) → PATCH {scheduled_date: null}', async () => {
    const wrapper = mountView()
    await flushPromises()
    tasksHolder.updateTask = vi.fn().mockResolvedValue({ task: { scheduled_date: null } })
    await wrapper.vm.saveScheduledDate(null)
    expect(tasksHolder.updateTask).toHaveBeenLastCalledWith('9', { scheduled_date: null })
  })

  it('без scheduled — datePicker пуст (null); срок при этом отображается как есть', async () => {
    const wrapper = mountView({ ...TASK, scheduled_date: null })
    await flushPromises()
    expect(wrapper.vm.scheduledDateValue).toBeNull()
    // due-поле живёт своей жизнью: его значение не зависит от scheduled
    const dv = wrapper.vm.dueDateValue
    expect(dv).toBeInstanceOf(Date)
    expect(dv.getDate()).toBe(15)
  })
})
