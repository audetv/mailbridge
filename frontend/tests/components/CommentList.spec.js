import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/api/client', () => ({
  default: { get: vi.fn(), post: vi.fn(), patch: vi.fn() }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(() => ({
    user: { username: 'admin' }
  }))
}))

import apiClient from '@/api/client'
import CommentList from '@/components/CommentList.vue'
import { useAuthStore } from '@/stores/auth'

const REPLY = {
  id: 11,
  task_id: 5,
  author: 'hermes',
  body: 'Готово',
  direction: 'out',
  kind: 'reply',
  approved: null,
  created_at: '2026-08-30T12:00:00Z'
}
const REPORT = {
  id: 12,
  task_id: 5,
  author: 'hermes',
  body: 'Выводы',
  direction: 'out',
  kind: 'report',
  approved: null,
  created_at: '2026-08-30T12:00:00Z'
}
const COMMENT = {
  id: 13,
  task_id: 5,
  author: 'admin',
  body: 'Просто комментарий',
  direction: 'out',
  kind: 'user_comment',
  approved: null,
  created_at: '2026-08-30T12:00:00Z'
}

function mockAttachments() {
  vi.mocked(apiClient.get).mockResolvedValue({ data: [] })
}

function mountList(comments) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const list = (comments || [fresh(REPLY), fresh(REPORT), fresh(COMMENT)]).map(c => ({ ...c }))
  const wrapper = mount(CommentList, {
    props: { comments: list },
    global: {
      plugins: [pinia],
      stubs: { 'vue-router': true }
    }
  })
  return wrapper
}

function fresh(o) {
  return { ...o }
}

describe('CommentList (ФАЗА 4 — бейджи + approve)', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    vi.mocked(useAuthStore).mockReturnValue({ user: { username: 'admin' } })
    mockAttachments()
  })

  it('показывает бейджи "Отчёт" и "Ответ пользователю" на соответствующих комментариях', () => {
    const wrapper = mountList()
    expect(wrapper.text()).toContain('Отчёт')
    expect(wrapper.text()).toContain('Ответ пользователю')
    expect(wrapper.findAll('.kind-badge').length).toBe(2)
  })

  it('показывает бейдж "Утверждён" на kind=reply с approved=1', () => {
    const wrapper = mountList([{ ...REPLY, approved: 1 }, REPORT])
    expect(wrapper.find('.approved-badge').exists()).toBe(true)
    expect(wrapper.text()).toContain('Утверждён')
  })

  it('не показывает бейджей/кнопки approve на kind=user_comment', () => {
    const wrapper = mountList([COMMENT])
    expect(wrapper.findAll('.kind-badge').length).toBe(0)
    expect(wrapper.find('.approve-btn').exists()).toBe(false)
  })

  it('кнопка "Утвердил ответ" видна только admin на kind=reply', () => {
    const wrapper = mountList([REPLY])
    expect(wrapper.find('.approve-btn').exists()).toBe(true)
    expect(wrapper.find('.approve-btn').text()).toContain('Утвердил ответ')
  })

  it('approve: PATCH /comments/11/approve + обновляет approved=1', async () => {
    vi.mocked(apiClient.patch).mockResolvedValueOnce({
      data: { comment: { ...REPLY, approved: 1 } }
    })
    const wrapper = mountList([REPLY])
    await wrapper.find('.approve-btn').trigger('click')
    await flushPromises()
    expect(apiClient.patch).toHaveBeenCalledWith('/comments/11/approve')
    expect(wrapper.find('.approved-badge').exists()).toBe(true)
  })

  it('approve: кнопка заменяется на "Ответ утверждён" после успешного approve', async () => {
    vi.mocked(apiClient.patch).mockResolvedValue({
      data: { comment: { ...REPLY, approved: 1 } }
    })
    const wrapper = mountList([{ ...REPLY, approved: null }])
    await wrapper.find('.approve-btn').trigger('click')
    await flushPromises()
    expect(wrapper.find('.approve-btn').exists()).toBe(false)
    expect(wrapper.text()).toContain('Ответ утверждён')
  })

  it('approve: не-admin — кнопка скрыта (mock authStore.user=hermes)', () => {
    vi.mocked(useAuthStore).mockReturnValue({ user: { username: 'hermes' } })
    const wrapper = mountList([REPLY])
    expect(wrapper.find('.approve-btn').exists()).toBe(false)
  })

  it('approve: 403 (не admin) — показывает ошибку бэкенда, не ломает UI', async () => {
    const alert = vi.fn()
    window.alert = alert
    vi.mocked(apiClient.patch).mockRejectedValueOnce({
      response: { data: { error: 'approve available only to admin' } }
    })
    const wrapper = mountList([REPLY])
    await wrapper.find('.approve-btn').trigger('click')
    await flushPromises()
    expect(alert).toHaveBeenCalledWith('approve available only to admin')
    expect(wrapper.find('.approved-badge').exists()).toBe(false)
    delete window.alert
  })
})
