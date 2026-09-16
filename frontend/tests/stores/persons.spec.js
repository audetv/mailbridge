import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

// API-клиент мокаем целиком (паттерн tests/stores/epics.spec.js).
vi.mock('@/api/client', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

import apiClient from '@/api/client'
import { usePersonsStore } from '@/stores/persons'

// Person из контракта GET /api/persons (internal/web/persons.go): bare-объект.
const PERSON = {
  id: 11,
  name: 'Иван',
  org: 'ООО Ромашка',
  is_internal: false,
  confirmed: false,
  archived: false,
  primary_email: 'ivan@example.com',
  created_at: '2026-09-16T00:00:00Z',
  updated_at: '2026-09-16T00:00:00Z'
}

describe('stores/persons — confirmPerson (шаг 6-G)', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    setActivePinia(createPinia())
  })

  it('PUT /persons/{id} — payload ПОЛНЫЙ: все 5 полей, confirmed=true (UpdatePerson — full-overwrite)', async () => {
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: { ...PERSON, confirmed: true } })
    const store = usePersonsStore()
    const updated = await store.confirmPerson(PERSON)
    expect(apiClient.put).toHaveBeenCalledWith('/persons/11', {
      name: 'Иван',
      org: 'ООО Ромашка',
      is_internal: false,
      confirmed: true,
      archived: false
    })
    expect(updated.confirmed).toBe(true)
  })

  it('имя/орган не теряются: payload = текущие поля персоны + confirmed=true', async () => {
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: { ...PERSON, confirmed: true } })
    const store = usePersonsStore()
    // Даже если персона из списка без primary_email — критичные 3 поля (name/org/is_internal) + archived на месте.
    const p = { id: 12, name: 'Пётр', org: 'БЦ', is_internal: true, confirmed: false, archived: true }
    await store.confirmPerson(p)
    const payload = vi.mocked(apiClient.put).mock.calls[0][1]
    expect(payload.name).toBe('Пётр')
    expect(payload.org).toBe('БЦ')
    expect(payload.is_internal).toBe(true)
    expect(payload.archived).toBe(true)
    expect(payload.confirmed).toBe(true)
  })

  it('локальное состояние обновлено: list и selected отражают confirmed=true без доп. GET', async () => {
    const updated = { ...PERSON, confirmed: true, updated_at: '2026-09-16T01:00:00Z' }
    vi.mocked(apiClient.put).mockResolvedValueOnce({ data: updated })
    const store = usePersonsStore()
    store.list = [{ ...PERSON }]
    store.selected = { ...PERSON }
    await store.confirmPerson(PERSON)
    expect(store.list.find((x) => x.id === PERSON.id).confirmed).toBe(true)
    expect(store.selected.confirmed).toBe(true)
  })

  it('ошибка → бросает (UI показывает локальный error), состояние не повреждено', async () => {
    vi.mocked(apiClient.put).mockRejectedValueOnce({ response: { data: { error: 'boom' } } })
    const store = usePersonsStore()
    store.list = [{ ...PERSON }]
    await expect(store.confirmPerson(PERSON)).rejects.toThrow()
    expect(store.list.find((x) => x.id === PERSON.id).confirmed).toBe(false)
  })
})
