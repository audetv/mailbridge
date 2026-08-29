import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'

// Моки ДО импорта компонента. Один общий объект на все вызовы use*Store —
// иначе мокап компонента и мокап теста были бы разными объектами.
const tasksMock = { createTask: vi.fn() }
const projectsMock = { projectByName: vi.fn() }
const epicsMock = { fetchEpics: vi.fn() }

vi.mock('@/stores/tasks', () => ({
  useTasksStore: () => tasksMock,
}))
vi.mock('@/stores/projects', () => ({
  useProjectsStore: () => projectsMock,
}))
vi.mock('@/stores/epics', () => ({
  useEpicsStore: () => epicsMock,
}))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: vi.fn() }) }))

import CreateTaskDialog from '@/components/CreateTaskDialog.vue'

// Экспонирование через defineExpose: @vue/test-utils разворачивает ref'ы,
// поэтому w.vm.title — уже строка (без .value), w.vm.canSubmit — boolean.
function mountDialog(props = {}) {
  return mount(CreateTaskDialog, {
    props: { visible: true, ...props },
    global: { plugins: [createPinia()] },
  })
}

describe('CreateTaskDialog', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    // tasksMock — общий модульный объект: сбрасываем calls и имплементации
    // между тестами, иначе «успешный» submit предыдущего теста догоняет сюда.
    tasksMock.createTask.mockReset()
  })

  it('без заголовка сабмит заблокирован', () => {
    const w = mountDialog({ projects: [{ name: 'A' }], initialProject: 'A' })
    w.vm.title = ''
    expect(w.vm.canSubmit).toBe(false)
  })

  it('после ввода заголовка сабмит разрешён (проект из контекста)', () => {
    const w = mountDialog({ projects: [{ name: 'A' }], initialProject: 'A' })
    w.vm.title = 'Заголовок'
    expect(w.vm.canSubmit).toBe(true)
  })

  it('submit → createTask c проектом из контекста, без epic_id', async () => {
    tasksMock.createTask.mockResolvedValue({ id: 42 })
    const w = mountDialog({ projects: [], initialProject: 'A', lockProject: true })
    w.vm.title = 'Заголовок'
    await w.vm.submit()
    expect(tasksMock.createTask).toHaveBeenCalledTimes(1)
    const payload = tasksMock.createTask.mock.calls[0][0]
    expect(payload.title).toBe('Заголовок')
    expect(payload.project).toBe('A')
    expect(payload.epic_id).toBeUndefined()
  })

  it('при выбранном модуле payload несёт epic_id', async () => {
    tasksMock.createTask.mockResolvedValue({ id: 43 })
    const w = mountDialog({
      projects: [{ name: 'A' }],
      initialProject: 'A',
      epicOptions: [{ value: 7, label: 'Модуль 7' }],
    })
    w.vm.title = 'Заголовок'
    w.vm.setEpic(7)
    await w.vm.submit()
    expect(tasksMock.createTask).toHaveBeenCalledTimes(1)
    expect(tasksMock.createTask.mock.calls[0][0].epic_id).toBe(7)
  })

  it('ошибка createTask → success не эмится', async () => {
    tasksMock.createTask.mockRejectedValue(new Error('boom'))
    const w = mountDialog({ projects: [], initialProject: 'A' })
    w.vm.title = 'Заголовок'
    await w.vm.submit()
    expect(w.emitted('success')).toBeUndefined()
  })

  it('lockProject=true → Select «Проект» не показывается', () => {
    const locked = mountDialog({
      projects: [{ name: 'A' }, { name: 'B' }],
      initialProject: 'A',
      lockProject: true,
    })
    const free = mountDialog({
      projects: [{ name: 'A' }, { name: 'B' }],
      initialProject: 'A',
      lockProject: false,
    })
    expect(locked.vm.showProjectSelect).toBe(false)
    expect(free.vm.showProjectSelect).toBe(true)
  })
})
