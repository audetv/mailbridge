import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/client', () => ({
  default: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn(), patch: vi.fn() }
}))
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: vi.fn() }) }))

import apiClient from '@/api/client'
import FilterBar from '@/components/FilterBar.vue'
import { useTasksStore } from '@/stores/tasks'
import { useEpicsStore } from '@/stores/epics'

const EPIC = { id: 7, project_id: 42, number: 2, name: 'Строчка', description: '', status: 'open' }

function mockApi() {
  vi.mocked(apiClient.get).mockImplementation((url) => {
    const u = String(url)
    if (u === '/tasks') return Promise.resolve({ data: { tasks: [], total: 0 } })
    if (u === '/projects') {
      return Promise.resolve({ data: [{ id: 42, name: 'Лидер Спорт', created_at: '' }] })
    }
    if (u.includes('/epics')) return Promise.resolve({ data: [EPIC] })
    return Promise.resolve({ data: [] })
  })
}

function mountBar() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const wrapper = mount(FilterBar, { global: { plugins: [pinia] } })
  return {
    wrapper,
    tasks: useTasksStore(pinia),
    epics: useEpicsStore(pinia)
  }
}

describe('FilterBar', () => {
  beforeEach(() => { vi.resetAllMocks() })

  it('селект «Модуль» есть и disabled пока не выбран проект', async () => {
    mockApi()
    const { wrapper } = mountBar()
    await flushPromises()
    const label = wrapper.find('.epic-select .p-select-label')
    expect(label.attributes('aria-disabled')).toBe('true')
  })

  it('выбор проекта: setFilter(project), загрузка /projects/{id}/epics, disabled снят', async () => {
    mockApi()
    const { wrapper, tasks, epics } = mountBar()
    await flushPromises()
    await wrapper.vm.onProjectChange('Лидер Спорт')
    await flushPromises()
    expect(tasks.filters.project).toBe('Лидер Спорт')
    expect(epics.epics).toEqual([EPIC])
    expect(apiClient.get).toHaveBeenCalledWith('/projects/42/epics')
    const label = wrapper.find('.epic-select .p-select-label')
    expect(label.attributes('aria-disabled')).toBe('false')
    // опции: #2 Строчка
    expect(wrapper.vm.epicOptions).toEqual([{ label: '#2 Строчка', value: '7' }])
  })

  it('выбор модуля: setFilter(epic_id) → GET /tasks?epic_id=7', async () => {
    mockApi()
    const { wrapper, tasks } = mountBar()
    await flushPromises()
    await wrapper.vm.onProjectChange('Лидер Спорт')
    await flushPromises()
    vi.mocked(apiClient.get).mockClear()
    await wrapper.vm.onChange('epic_id', '7')
    await flushPromises()
    expect(tasks.filters.epic_id).toBe('7')
    expect(apiClient.get).toHaveBeenCalledWith('/tasks', expect.objectContaining({
      params: expect.objectContaining({ epic_id: '7' })
    }))
  })

  it('сброс проекта: фильтр epic_id очищается', async () => {
    mockApi()
    const { wrapper, tasks } = mountBar()
    await flushPromises()
    await wrapper.vm.onProjectChange('Лидер Спорт')
    await flushPromises()
    await wrapper.vm.onChange('epic_id', '7')
    await flushPromises()
    await wrapper.vm.onProjectChange(null)
    await flushPromises()
    expect(tasks.filters.project).toBe('')
    expect(tasks.filters.epic_id).toBe('')
  })
})
