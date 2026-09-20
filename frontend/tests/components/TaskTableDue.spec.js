// v0.25, шаг 7c: колонка «Срок» в списке задач — расцветка + pending-бейдж.
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
  const base = { project: 'Лидер Спорт', status: 'new', unread_comments: 0, created_at: '2026-08-29T10:00:00Z' }
  tasks.tasks = [
    { id: 1, ...base, due_date: '2026-09-16' }, // просрочено
    { id: 2, ...base, due_date: todayKey() }, // сегодня — ЛОКАЛЬНЫЙ день (dueClassOf считает локально; toISOString()=UTC → флейк 00:00–03:00 МСК, баг 2026-09-19)
    { id: 3, ...base, due_date: '2027-01-15' }, // будущее
    { id: 4, ...base, due_date: null, due_ai_pending: true, ai_due_date: '2027-01-15' }, // pending
    { id: 5, ...base, due_date: null } // без срока
  ]
  tasks.total = 5
  const wrapper = mount(TaskTable, {
    global: { plugins: [pinia, ToastService, [PrimeVue, {}]] }
  })
  await flushPromises()
  return { wrapper, tasks }
}

describe('TaskTable — колонка «Срок» (7c)', () => {
  let pinia
  beforeEach(() => {
    vi.resetAllMocks()
    pinia = createPinia()
    setActivePinia(pinia)
  })

  it('заголовок «Срок» присутствует; класс расцветки по due_date', async () => {
    const { wrapper } = await mountTable(pinia)
    const headers = wrapper.findAll('th').map((th) => th.text())
    expect(headers).toContain('Срок')

    const cell = (row) => row.find('[data-testid="due-cell"]')
    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(5)

    expect(cell(rows[0]).exists()).toBe(true)
    expect(cell(rows[0]).classes()).toContain('due--overdue')
    expect(cell(rows[1]).classes()).toContain('due--today')
    expect(cell(rows[2]).classes()).toContain('due--future')
    expect(cell(rows[4]).classes()).toContain('due--none')

    // Tag с датой там, где срок есть; «—» где нет.
    expect(cell(rows[0]).find('.due-tag').text()).toBe('2026-09-16')
    expect(cell(rows[4]).find('[data-testid="due-none"]').text()).toBe('—')
  })

  it('pending (due_ai_pending) — бейдж «срок?» (предложение AI ждёт решения)', async () => {
    const { wrapper } = await mountTable(pinia)
    const rows = wrapper.findAll('tbody tr')

    const pending = wrapper.findAll('[data-testid="due-pending-badge"]')
    expect(pending.length).toBe(1)
    expect(pending[0].text()).toContain('срок?')
    // именно в строке задачи 4 (не в датах других строк)
    const row4cell = rows[3].find('[data-testid="due-cell"]')
    expect(row4cell.exists()).toBe(true)
    expect(row4cell.find('[data-testid="due-pending-badge"]').exists()).toBe(true)
    expect(rows[0].find('[data-testid="due-pending-badge"]').exists()).toBe(false)
  })
})
