import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/client', () => ({
  default: { get: vi.fn(), patch: vi.fn(), post: vi.fn() }
}))
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
  useRoute: () => ({ params: {}, query: {} })
}))

import TaskTable from '@/components/TaskTable.vue'
import { useEpicsStore } from '@/stores/epics'
import { useTasksStore } from '@/stores/tasks'

async function mountTable(pinia) {
  const tasks = useTasksStore(pinia)
  tasks.tasks = [
    { id: 1, epic_id: 7, project: 'Лидер Спорт', status: 'new', unread_comments: 0, created_at: '2026-08-29T10:00:00Z' },
    { id: 2, epic_id: null, project: 'Лидер Спорт', status: 'new', unread_comments: 0, created_at: '2026-08-29T11:00:00Z' }
  ]
  tasks.total = 2
  const wrapper = mount(TaskTable, { global: { plugins: [pinia] } })
  await flushPromises()
  return { wrapper, tasks, epics: useEpicsStore(pinia) }
}

describe('TaskTable — колонка «Модуль»', () => {
  let pinia
  beforeEach(() => {
    vi.resetAllMocks()
    pinia = createPinia()
    setActivePinia(pinia)
  })

  it('задача с epic_id показывает имя модуля из epics-store; без epic_id — «—»', async () => {
    const { wrapper, epics } = await mountTable(pinia)
    epics.epics = [{ id: 7, project_id: 42, number: 2, name: 'Строчка', description: '', status: 'open' }]
    await wrapper.vm.$nextTick()
    expect(wrapper.vm.epicName({ epic_id: 7 })).toBe('Строчка')
    expect(wrapper.vm.epicName({ epic_id: null })).toBe(null)
    expect(wrapper.vm.epicName({ epic_id: 999 })).toBe(null)
    // колонка в разметке
    const headers = wrapper.findAll('th').map((th) => th.text())
    expect(headers).toContain('Модуль')
    // первая строка имеет Tag с именем, вторая — штрих
    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(2)
    // колонка «Модуль»: имя эпика в Tag первой строки, «—» во второй
    const headerIdx = wrapper.findAll('th').findIndex((th) => th.text() === 'Модуль')
    const epicCell = (row, i) => row.findAll('td')[i]
    expect(epicCell(rows[0], headerIdx).find('.p-tag').text()).toContain('Строчка')
    expect(epicCell(rows[1], headerIdx).find('.epic-none').text()).toBe('—')
  })

  it('колонка «Проект» — ссылка; клик фильтрует задачи по проекту (шаг 6)', async () => {
    const { wrapper, tasks } = await mountTable(pinia)
    vi.spyOn(tasks, 'setFilter').mockImplementation(() => {})
    const headerIdx = wrapper.findAll('th').findIndex((th) => th.text() === 'Проект')
    expect(headerIdx).toBeGreaterThanOrEqual(0)
    const link = wrapper.findAll('tbody tr')[0].findAll('td')[headerIdx].find('.project-link')
    expect(link.exists()).toBe(true)
    expect(link.text()).toBe('Лидер Спорт')

    await link.trigger('click')
    expect(tasks.setFilter).toHaveBeenCalledWith('project', 'Лидер Спорт')
  })
})
