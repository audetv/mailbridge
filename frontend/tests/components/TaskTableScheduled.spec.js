// v0.28, шаг 28c: колонка «План» в списке задач — scheduled_date.
//   Тон другой, не «дедлайн»: сегодня в плане — info (не warn); дальше — secondary.
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
import { useTasksStore } from '@/stores/tasks'
import { todayKey } from '@/utils/due-date'
import ToastService from 'primevue/toastservice'
import PrimeVue from 'primevue/config'

async function mountTable(pinia) {
  const tasks = useTasksStore(pinia)
  const base = { project: 'Лидер Спорт', status: 'new', unread_comments: 0, created_at: '2026-09-15T10:00:00Z' }
  const local = new Date()
  local.setDate(local.getDate() + 3)
  const inThreeDays = `${local.getFullYear()}-${String(local.getMonth() + 1).padStart(2, '0')}-${String(local.getDate()).padStart(2, '0')}`
  tasks.tasks = [
    { id: 1, ...base, due_date: '2026-01-01', scheduled_date: todayKey() }, // план сегодня
    { id: 2, ...base, due_date: null, scheduled_date: inThreeDays }, // план дальше
    { id: 3, ...base, due_date: todayKey(), scheduled_date: null } // без плана (срок есть)
  ]
  tasks.total = 3
  const wrapper = mount(TaskTable, {
    global: { plugins: [pinia, ToastService, [PrimeVue, {}]] }
  })
  await flushPromises()
  return { wrapper, tasks }
}

describe('TaskTable — колонка «План» (scheduled, 28c)', () => {
  let pinia
  beforeEach(() => {
    vi.resetAllMocks()
    pinia = createPinia()
    setActivePinia(pinia)
  })

  it('заголовок «План»; тег с датой там, где план есть; «—» где нет', async () => {
    const { wrapper } = await mountTable(pinia)
    const headers = wrapper.findAll('th').map((th) => th.text())
    expect(headers).toContain('План')

    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(3)

    const tag1 = rows[0].find('[data-testid="scheduled-tag"]')
    expect(tag1.exists()).toBe(true)
    expect(tag1.text()).toBe(todayKey())
    // тон «план сегодня» — info (синий), а не warn (жёлтый) как у дедлайна
    expect(tag1.classes()).toContain('p-tag-info')

    const tag2 = rows[1].find('[data-testid="scheduled-tag"]')
    expect(tag2.exists()).toBe(true)
    expect(tag2.classes()).toContain('p-tag-secondary')

    const none3 = rows[2].find('[data-testid="scheduled-none"]')
    expect(none3.exists()).toBe(true)
    expect(none3.text()).toBe('—')

    // «Срок» и «План» — разные колонки/тестиды: срок задачи 3 не попал в план
    expect(rows[2].find('[data-testid="due-cell"]').exists()).toBe(true)
  })
})
