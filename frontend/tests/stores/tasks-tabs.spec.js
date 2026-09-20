// tasks-tabs.spec.js — v0.28, шаг 28e: вкладки-дропдауны.
// Модель (решения владельца 2026-09-20, аналогия Bootstrap «Tabs with
// dropdowns»): «Статус» и «План» — дропдаун-вкладки (таб = пункт выбора,
// таб показывает выбранный пункт; ОТДЕЛЬНОГО селекта НЕТ — 28d откат).
// «Статус»: Все/Активные/Бэклог/Выполненные/Закрытые → ?status=;
// «План»: Все (без фильтра)/Без плана (?plan=none)/Сегодня/Завтра/Неделя
// (?plan=, серверный срез 28b). Статусы и план-срез не взаимоисключающие,
// «План» не шлёт status.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const apiGet = vi.hoisted(() => vi.fn(async () => ({ data: { tasks: [], total: 0 } })))
vi.mock('@/api/client', () => ({ default: { get: apiGet, post: vi.fn(), patch: vi.fn(), delete: vi.fn() } }))

import { useTasksStore } from '@/stores/tasks'

function store() {
  setActivePinia(createPinia())
  return useTasksStore()
}

describe('v0.28/28e — store setTab / setStatusFilter / setPlanFilter', () => {
  beforeEach(() => {
    apiGet.mockClear()
  })

  it('STATUS_FILTER: 5 значений — все/активные/бэклог/выполненные/закрытые', () => {
    const s = store()
    expect(s.STATUS_FILTER).toEqual({
      all: [],
      active: ['new', 'in_progress'],
      backlog: ['backlog'],
      completed: ['completed'],
      closed: ['closed']
    })
  })

  it('PLAN_FILTER: all/none/today/tomorrow/week (28e: + Все/Без плана)', () => {
    const s = store()
    expect(['all', 'none', 'today', 'tomorrow', 'week']).toEqual(
      Object.keys(s.PLAN_FILTER).sort()
    )
    expect(s.PLAN_FILTER.all).toBe('') // «Все» — без фильтра
    expect(s.PLAN_FILTER.none).toBe('none') // «Без плана» — ?plan=none
  })

  it('setTab("status") — дефолт «Активные», без plan, due сброшен, status+fetch', () => {
    const s = store()
    s.setTab('status')
    expect(s.filters.statuses).toEqual(['new', 'in_progress'])
    expect(s.filters.plan).toBe('')
    expect(s.filters.due).toBe('')
    expect(s.filters.sort).toBe('created') // v0.29.0: дефолт — «По дате создания»
    expect(s.filters.page).toBe(1)
    const call = vi.mocked(apiGet).mock.lastCall
    expect(call[0]).toBe('/tasks')
    expect(call[1].params).toHaveProperty('status', ['new', 'in_progress'])
  })

  it('setTab("status","backlog") — статус-фильтр по значению селекта', () => {
    const s = store()
    s.setTab('status', 'backlog')
    expect(s.filters.statuses).toEqual(['backlog'])
  })

  it('setStatusFilter — переключает статус, обнуляет plan, fetch по status', () => {
    const s = store()
    s.setTab('plan', 'week') // сначала в плане
    expect(s.filters.plan).toBe('week')
    s.setStatusFilter('completed')
    expect(s.filters.statuses).toEqual(['completed'])
    expect(s.filters.plan).toBe('')
    const call = vi.mocked(apiGet).mock.lastCall
    expect(call[0]).toBe('/tasks')
    expect(call[1].params).toHaveProperty('status', ['completed'])
    expect(call[1].params).not.toHaveProperty('plan')
  })

  it('setTab("plan") — дефолт сегодня, БЕЗ status (любой статус), fetch с plan', () => {
    const s = store()
    s.setTab('plan')
    expect(s.filters.plan).toBe('today')
    expect(s.filters.statuses).toEqual([])
    const params = vi.mocked(apiGet).mock.lastCall[1].params
    expect(params.plan).toBe('today')
    expect(params).not.toHaveProperty('status')
  })

  it('setPlanFilter — меняет план-срез, остаётся без status, fetch', () => {
    const s = store()
    s.setTab('plan')
    apiGet.mockClear()
    s.setPlanFilter('tomorrow')
    expect(s.filters.plan).toBe('tomorrow')
    const params = vi.mocked(apiGet).mock.lastCall[1].params
    expect(params.plan).toBe('tomorrow')
    expect(params).not.toHaveProperty('status')
    expect(s.filters.statuses).toEqual([])
  })

  it('28e: setPlanFilter("all") — «Все» = без фильтра (plan не уходит в params)', () => {
    const s = store()
    s.setTab('plan')
    apiGet.mockClear()
    s.setPlanFilter('all')
    expect(s.filters.plan).toBe('')
    expect(vi.mocked(apiGet).mock.lastCall[1].params).not.toHaveProperty('plan')
  })

  it('28e: setPlanFilter("none") — «Без плана» → ?plan=none', () => {
    const s = store()
    s.setTab('plan')
    apiGet.mockClear()
    s.setPlanFilter('none')
    expect(s.filters.plan).toBe('none')
    expect(vi.mocked(apiGet).mock.lastCall[1].params).toHaveProperty('plan', 'none')
  })

  it('28e: setTab("plan","none") / ("plan","all") — пункт выбора из дропдауна', () => {
    const s = store()
    apiGet.mockClear()
    s.setTab('plan', 'none')
    expect(s.filters.plan).toBe('none')
    s.setTab('plan', 'all')
    expect(s.filters.plan).toBe('')
  })

  it('fetchTasks: план и due вместе (AND на сервере) — оба уходят в params', () => {
    const s = store()
    s.filters.due = 'today'
    s.filters.plan = 'week'
    s.fetchTasks()
    const params = vi.mocked(apiGet).mock.lastCall[1].params
    expect(params.plan).toBe('week')
    expect(params.due).toBe('today')
  })

  it('fetchTasks: пустой plan не слать (как due/epic_id)', () => {
    const s = store()
    s.filters.plan = ''
    s.fetchTasks()
    expect(vi.mocked(apiGet).mock.lastCall[1].params).not.toHaveProperty('plan')
  })

  it('несуществующий таб — no-op (не трогает фильтры, не fetch)', () => {
    const s = store()
    apiGet.mockClear()
    s.setTab('???')
    expect(apiGet).not.toHaveBeenCalled()
  })
})
