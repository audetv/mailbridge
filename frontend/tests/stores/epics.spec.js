import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

// API-клиент мокаем целиком — контракт зафиксирован в PLAN.steps10-11.md §1.1.
vi.mock('@/api/client', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

import apiClient from '@/api/client'
import { useEpicsStore } from '@/stores/epics'

const EPIC = {
  id: 1,
  project_id: 3,
  number: 2,
  name: 'Строчка',
  description: 'desc',
  status: 'open',
  created_at: '2026-08-29T10:00:00Z',
  updated_at: '2026-08-29T10:00:00Z'
}

function makeStore() {
  const store = useEpicsStore()
  store.setProject(3)
  return store
}

describe('stores/epics', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    setActivePinia(createPinia())
  })

  it('fetchEpics без projectId очищает список', async () => {
    const store = useEpicsStore()
    await store.fetchEpics(null)
    expect(store.epics).toEqual([])
    expect(apiClient.get).not.toHaveBeenCalled()
  })

  it('fetchEpics ставит список из GET /projects/{id}/epics', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [EPIC] })
    const store = useEpicsStore()
    await store.fetchEpics(3)
    expect(apiClient.get).toHaveBeenCalledWith('/projects/3/epics')
    expect(store.epics).toEqual([EPIC])
    expect(store.loading).toBe(false)
  })

  it('fetchEpics при ошибке кладёт message в error', async () => {
    vi.mocked(apiClient.get).mockRejectedValueOnce({
      response: { data: { error: 'boom' } }
    })
    const store = useEpicsStore()
    await store.fetchEpics(3)
    expect(store.error).toBe('boom')
    expect(store.epics).toEqual([])
  })

  it('createEpic — POST /projects/{id}/epics {name,status}, возвращает epic и перечитывает список', async () => {
    const created = { ...EPIC, id: 7, name: 'Новый' }
    vi.mocked(apiClient.post).mockResolvedValueOnce({ data: created })
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [created] })
    const store = useEpicsStore()
    const res = await store.createEpic(3, 'Новый', 'open')
    expect(apiClient.post).toHaveBeenCalledWith('/projects/3/epics', { name: 'Новый', status: 'open' })
    expect(res).toEqual(created)
    expect(store.epics).toEqual([created])
  })

  it('renameEpic — PUT /epics/{id} с полями name/description/status', async () => {
    const updated = { ...EPIC, name: 'Переименован' }
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: updated })
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [updated] })
    const store = makeStore()
    const res = await store.renameEpic(1, 'Переименован', 'd2', 'done')
    expect(apiClient.put).toHaveBeenCalledWith('/epics/1', {
      name: 'Переименован',
      description: 'd2',
      status: 'done'
    })
    expect(res).toEqual(updated)
  })

  it('deleteEpic — DELETE /epics/{id}; фильтрует себя, если был активным фильтром', async () => {
    vi.mocked(apiClient.delete).mockResolvedValueOnce({ data: null })
    // Поток get: setProject(3) → список; setFilter(1) → detail; deleteEpic → список.
    vi.mocked(apiClient.get)
      .mockResolvedValueOnce({ data: [EPIC] })
      .mockResolvedValueOnce({ data: { ...EPIC, progress: { total: 1, open: 1, done: 0 } } })
      .mockResolvedValueOnce({ data: [] })
    const store = makeStore()
    store.setFilter(1)
    await store.deleteEpic(1)
    expect(apiClient.delete).toHaveBeenCalledWith('/epics/1')
    expect(store.filterEpicId).toBeNull()
    expect(store.epics).toEqual([])
  })

  it('fetchDetail — API возвращает FLAT объект (эпик + progress), без обёртки epic', async () => {
    const flatDetail = { ...EPIC, progress: { total: 10, open: 4, done: 6 } }
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: flatDetail })
    const store = useEpicsStore()
    const res = await store.fetchDetail(1)
    expect(apiClient.get).toHaveBeenCalledWith('/epics/1')
    expect(res).toEqual(flatDetail)
    expect(store.detail).toMatchObject({ id: 1, name: 'Строчка', status: 'open' })
    expect(store.progress).toEqual({ total: 10, open: 4, done: 6 })
  })

  it('linkTask — POST /epics/{epicId}/tasks/{taskId} и перечитывает detail', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce({ data: null })
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { ...EPIC, progress: { total: 1, open: 1, done: 0 } }
    })
    const store = useEpicsStore()
    await store.linkTask(1, 42)
    expect(apiClient.post).toHaveBeenCalledWith('/epics/1/tasks/42')
    expect(store.progress).toEqual({ total: 1, open: 1, done: 0 })
  })

  it('unlinkTask — DELETE /epics/{epicId}/tasks/{taskId}', async () => {
    vi.mocked(apiClient.delete).mockResolvedValueOnce({ data: null })
    vi.mocked(apiClient.get).mockResolvedValueOnce({
      data: { ...EPIC, progress: { total: 0, open: 0, done: 0 } }
    })
    const store = useEpicsStore()
    await store.unlinkTask(1, 42)
    expect(apiClient.delete).toHaveBeenCalledWith('/epics/1/tasks/42')
    expect(store.progress).toEqual({ total: 0, open: 0, done: 0 })
  })

  it('setProject сбрасывает filter/detail/progress и перечитывает', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: [] })
    const store = useEpicsStore()
    store.filterEpicId = 1
    store.detail = EPIC
    store.progress = { total: 1, open: 1, done: 0 }
    store.setProject(5)
    expect(store.currentProjectId).toBe(5)
    expect(store.filterEpicId).toBeNull()
    expect(store.detail).toBeNull()
    expect(store.progress).toBeNull()
    expect(apiClient.get).toHaveBeenCalledWith('/projects/5/epics')
  })

  it('setFilter(null) очищает detail/progress', async () => {
    const store = makeStore()
    store.detail = EPIC
    store.progress = { total: 1, open: 1, done: 0 }
    store.setFilter(null)
    expect(store.filterEpicId).toBeNull()
    expect(store.detail).toBeNull()
    expect(store.progress).toBeNull()
  })

  it('epicById находит по id или null', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce({ data: [EPIC] })
    const store = useEpicsStore()
    await store.fetchEpics(3)
    expect(store.epicById(1)).toEqual(EPIC)
    expect(store.epicById(99)).toBeNull()
  })
})
