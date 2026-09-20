// tasks-tabs.spec.js — v0.28, шаг 28d: вкладка «Статус» + селект / «План» + селект.
// Модель (решения владельца 2026-09-19): 5 статусных вкладок → ОДНА «Статус» +
// селект (Все/Активные/Бэклог/Выполненные/Закрытые → ?status=); + вкладка
// «План» + селект (Сегодня/Завтра/Неделя → ?plan=, серверный срез 28b).
// Статусные статусы и план-срез — не взаимоисключающие, «План» не шлёт status.
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const apiGet = vi.hoisted(() => vi.fn(async () => ({ data: { tasks: [], total: 0 } })))
vi.mock('@/api/client', () => ({ default: { get: apiGet, post: vi.fn(), patch: vi.fn(), delete: vi.fn() } }))

import { useTasksStore } from '@/stores/tasks'

function store() {
  setActivePinia(createPinia())
  return useTasksStore()
}

describe('v0.28/28d — store setTab / setStatusFilter / setPlanFilter', () => {
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

  it('PLAN_FILTER: today/tomorrow/week (значения серверного среза 28b)', () => {
    const s = store()
    expect(['today', 'tomorrow', 'week']).toEqual(Object.keys(s.PLAN_FILTER).sort())
  })

  it('setTab("status") — дефолт «Активные», без plan, due сброшен, status+fetch', () => {
    const s = store()
    s.setTab('status')
    expect(s.filters.statuses).toEqual(['new', 'in_progress'])
    expect(s.filters.plan).toBe('')
    expect(s.filters.due).toBe('')
    expect(s.filters.sort).toBe('due')
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
