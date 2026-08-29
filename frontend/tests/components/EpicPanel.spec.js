import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/client', () => ({
  default: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() }
}))
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: vi.fn() }) }))

import apiClient from '@/api/client'
import EpicPanel from '@/components/EpicPanel.vue'
import { useEpicsStore } from '@/stores/epics'

const EPIC = { id: 1, project_id: 3, number: 1, name: 'Строчка', description: 'desc', status: 'open' }

function mockApi() {
  vi.mocked(apiClient.get).mockImplementation((url) => {
    if (String(url).includes('/projects/')) return Promise.resolve({ data: [EPIC] })
    return Promise.resolve({ data: { ...EPIC, progress: { total: 4, open: 1, done: 3 } } })
  })
}

function mountPanel() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const wrapper = mount(EpicPanel, {
    props: { project: { id: 3, name: 'Проект' } },
    global: { plugins: [pinia] }
  })
  return { wrapper, store: useEpicsStore() }
}

describe('EpicPanel', () => {
  beforeEach(() => { vi.resetAllMocks() })

  it('показывает список: номер, имя, статус', async () => {
    mockApi()
    const { wrapper } = mountPanel()
    await flushPromises()
    expect(wrapper.find('.epic-name').text()).toBe('Строчка')
    expect(wrapper.find('.epic-number').text()).toContain('#1')
    expect(wrapper.text()).toContain('Проект') // заголовок панели
  })

  it('показывает прогресс-бар (75% = 3 из 4 done)', async () => {
    mockApi()
    const { wrapper } = mountPanel()
    await flushPromises()
    expect(wrapper.find('.p-progressbar').exists()).toBe(true)
    const pct = wrapper.find('.p-progressbar-value').attributes('style')
    expect(pct).toContain('width: 75%')
  })

  it('пустой проект: плейсхолдер + форма создания', async () => {
    vi.mocked(apiClient.get).mockImplementation((url) => {
      if (String(url).includes('/projects/')) return Promise.resolve({ data: [] })
      return Promise.resolve({ data: EPIC })
    })
    const { wrapper } = mountPanel()
    await flushPromises()
    expect(wrapper.find('.empty').exists()).toBe(true)
    expect(wrapper.find('input[placeholder="Имя модуля"]').exists()).toBe(true)
  })

  it('создание: POST /projects/3/epics с {name}', async () => {
    mockApi()
    vi.mocked(apiClient.post).mockResolvedValueOnce({ data: { ...EPIC, id: 99 } })
    const { wrapper } = mountPanel()
    await flushPromises()
    await wrapper.find('input[placeholder="Имя модуля"]').setValue('Новый')
    await wrapper.find('.create-row .p-button').trigger('click')
    await flushPromises()
    expect(apiClient.post).toHaveBeenCalledWith('/projects/3/epics', { name: 'Новый', status: 'open' })
  })

  it('смена статуса: PUT /epics/1 с {name, description, status}', async () => {
    mockApi()
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: { ...EPIC, status: 'done' } })
    const { wrapper } = mountPanel()
    await flushPromises()
    // выбрать «Готов» в SelectButton (тоглы внутри .p-selectbutton)
    const sb = wrapper.findAll('.p-selectbutton .p-togglebutton')
    await sb[2].trigger('click')
    await flushPromises()
    expect(apiClient.put).toHaveBeenCalledWith(
      '/epics/1',
      expect.objectContaining({ status: 'done', name: 'Строчка', description: 'desc' })
    )
  })

  it('удаление: DELETE /epics/1', async () => {
    mockApi()
    vi.mocked(apiClient.delete).mockResolvedValueOnce({ data: null })
    const { wrapper } = mountPanel()
    await flushPromises()
    const delBtn = wrapper.findAll('.p-button').find((b) => b.text().includes('Удалить'))
    await delBtn?.trigger('click')
    await flushPromises()
    expect(apiClient.delete).toHaveBeenCalledWith('/epics/1')
  })
})
