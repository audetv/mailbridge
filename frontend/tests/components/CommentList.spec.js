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

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '5' } }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() })
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

function mountList(comments, inboxItems) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const list = (comments || [fresh(REPLY), fresh(REPORT), fresh(COMMENT)]).map(c => ({ ...c }))
  const wrapper = mount(CommentList, {
    props: { comments: list, inboxItems: (inboxItems || []).map(i => ({ ...i })) },
    global: {
      plugins: [pinia],
      stubs: {
        'vue-router': true,
        RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }
      }
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

  it('кнопка "Утвердить ответ" видна только admin на kind=reply', () => {
    const wrapper = mountList([REPLY])
    expect(wrapper.find('.approve-btn').exists()).toBe(true)
    expect(wrapper.find('.approve-btn').text()).toContain('Утвердить ответ')
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


// Шаг 5 v0.23 (решение владельца 14.09, задача 378): AI-вердикт = ТОЛЬКО
// саммари-текст комментария. Письмо показывается СВЕРХУ в карточке
// «Оригинальное письмо» (уже новая часть, см. new-message-part.spec.js) —
// дубль письма/цитаты/вложений ВНУТРИ AI-комментария убран.
describe('CommentList (шаг 5 v0.23 — AI-вердикт = только саммари)', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    vi.mocked(useAuthStore).mockReturnValue({ user: { username: 'admin' } })
    mockAttachments()
  })

  const INBOX_ITEM = {
    id: 128,
    from_name: 'Целищева Виктория',
    from_contact: 'vcel@example.com',
    subject: 'Сроки',
    body_text: 'Алексей, сообщите сроки, когда ждать ответ. Ждём до конца недели.\nВика'
  }
  const VERDICT_COMMENT = {
    ...COMMENT,
    id: 50,
    author: 'ai',
    kind: 'ai_verdict',
    inbox_item_id: 128,
    verdict_json: JSON.stringify({ quote: 'сообщите сроки, когда ждать ответ' }),
    body: 'Задача обновлена: клиент уточнила, что нужен новый документ для сайта.'
  }

  it('AI-вердикт: виден ТОЛЬКО саммари-текст, без quote-цитаты и без detail письма', () => {
    const wrapper = mountList([VERDICT_COMMENT], [INBOX_ITEM])
    expect(wrapper.find('.comment-body').text()).toContain('Задача обновлена: клиент уточнила')
    expect(wrapper.find('.comment-quote').exists()).toBe(false)
    expect(wrapper.find('.comment-original').exists()).toBe(false)
    expect(wrapper.find('details').exists()).toBe(false)
  })

  it('AI-вердикт с verdict_json и inbox_item: без дубля письма (quote из verdict не выносится)', () => {
    const wrapper = mountList([VERDICT_COMMENT], [INBOX_ITEM])
    // даже при наличии quote в verdict_json и inbox_item в comment-блоке письма нет
    expect(wrapper.find('.comment-quote').exists()).toBe(false)
    expect(wrapper.find('.comment-original').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('сообщите сроки, когда ждать ответ')
  })

  it('AI-вердикт без inbox_item (пусто inboxItems): UI не ломается, только текст', () => {
    const wrapper = mountList([VERDICT_COMMENT], [])
    expect(wrapper.find('.comment-quote').exists()).toBe(false)
    expect(wrapper.find('.comment-original').exists()).toBe(false)
    expect(wrapper.find('.comment-body').exists()).toBe(true)
  })

  // Решение #1 (2026-09-14): legacy-комментарии старого AI-пайплайна писались
  // под author='user' (83 строки). Не мигрируем: UI выводит реального
  // отправителя из inbox_item_id + бейдж «AI-саммари».
  it('legacy author="user": показывает реального отправителя + бейдж AI-саммари', () => {
    const legacy = {
      ...COMMENT,
      id: 52,
      author: 'user',
      direction: 'in',
      inbox_item_id: 128,
      verdict_json: undefined,
      body: 'Клиент уточнил сроки по правкам сайта.'
    }
    const wrapper = mountList([legacy], [INBOX_ITEM])
    expect(wrapper.find('.author').text()).not.toContain('user')
    expect(wrapper.find('.author').text()).toContain('Целищева Виктория')
    expect(wrapper.find('.ai-summary-badge').exists()).toBe(true)
    expect(wrapper.find('.ai-summary-badge').text()).toBe('AI-саммари')
    // саммари-комментарий (author='user') НОСИТ detail письма — ТОЛЬКО новая часть
    // (спойлер: «то, что написали в этом сообщении»; вся история — по ссылке в Inbox)
    expect(wrapper.find('.comment-original').exists()).toBe(true)
    const body = wrapper.find('.comment-original-body')
    expect(body.text()).toContain('сообщите сроки, когда ждать ответ')
    expect(body.text()).toContain('Вся переписка — в ленте')
    // router-link → <a href="/inbox/128"> (стабб в mountList)
    expect(body.find('a[href="/inbox/128"]').exists()).toBe(true)
    expect(wrapper.find('summary').text()).toContain('Целищева Виктория')
  })

  // Хотфикс v0.27.2: саммари после 15.09 (коммит 7f65a3a) пишутся
  // senderLabel(email) — реальный автор, kind=user_comment, direction=in,
  // inbox_item_id заполнен. Предикат — через kind, не через author.
  const NEW_SUMMARY = {
    ...COMMENT,
    id: 60,
    author: 'Гусев Алексей',
    direction: 'in',
    inbox_item_id: 128,
    body: 'Клиент уточнил, что нужен новый документ для сайта.'
  }

  it('новый саммари (kind=user_comment, direction=in, inbox_item_id): реальный автор + бейдж AI-саммари + detail «Оригинал письма»', () => {
    const wrapper = mountList([NEW_SUMMARY], [INBOX_ITEM])
    expect(wrapper.find('.author').text()).toContain('Гусев Алексей')
    expect(wrapper.find('.ai-summary-badge').exists()).toBe(true)
    expect(wrapper.find('.ai-summary-badge').text()).toBe('AI-саммари')
    expect(wrapper.find('.comment-original').exists()).toBe(true)
    expect(wrapper.find('.comment-original-body').text()).toContain('сообщите сроки, когда ждать ответ')
    expect(wrapper.find('summary').text()).toContain('Целищева Виктория')
  })

  it('новый саммари БЕЗ inboxItem: без detail и без краша', () => {
    const wrapper = mountList([{ ...NEW_SUMMARY, id: 61 }], [])
    expect(wrapper.find('.comment-original').exists()).toBe(false)
    expect(wrapper.find('.comment-body').exists()).toBe(true)
  })

  // Страховка (урок из разбора, PLAN.md «Хотфикс v0.27.2» п.1): гейт НЕ может
  // стоять на direction=in + inbox_item_id — у ai_verdict тоже есть оба поля.
  it('ai_verdict (direction=in, inbox_item_id есть): detail НЕ появляется даже при новом гейте', () => {
    const wrapper = mountList([VERDICT_COMMENT], [INBOX_ITEM])
    expect(wrapper.find('.comment-original').exists()).toBe(false)
    expect(wrapper.find('.ai-summary-badge').exists()).toBe(false)
  })

  it('user_comment с direction=out (внутренний): БЕЗ бейджа AI-саммари и detail', () => {
    const inner = { ...COMMENT, id: 62, direction: 'out', inbox_item_id: 128 }
    const wrapper = mountList([inner], [INBOX_ITEM])
    expect(wrapper.find('.ai-summary-badge').exists()).toBe(false)
    expect(wrapper.find('.comment-original').exists()).toBe(false)
  })

  it('legacy author="user" БЕЗ inboxItem → «автор письма», без краха', () => {
    const legacy = { ...COMMENT, id: 53, author: 'user', inbox_item_id: undefined, body: 'Саммари.' }
    const wrapper = mountList([legacy], [])
    expect(wrapper.find('.author').text()).toContain('автор письма')
    expect(wrapper.find('.ai-summary-badge').exists()).toBe(true)
  })

  // Финальный фикс (15.09): вложения УБИРАЮТСЯ ТОЛЬКО из AI-вердикта.
  const ATTS = [{ id: 1, filename: 'фото.png', storage_path: 't5/a1' }]

  it('AI-вердикт: вложения НЕ показываются даже если у коммента есть файлы', async () => {
    const getMock = vi.mocked(apiClient.get)
    getMock.mockReset()  // сброс mockResolvedValue из beforeEach
    getMock.mockImplementation((url) =>
      String(url).includes('/comments/50/attachments')
        ? Promise.resolve({ data: ATTS })
        : Promise.resolve({ data: [] }))
    const wrapper = mountList([VERDICT_COMMENT], [INBOX_ITEM])
    await flushPromises()
    expect(getMock).toHaveBeenCalledWith(expect.stringContaining('/comments/50/attachments'))
    expect(wrapper.find('.comment-attachments').exists()).toBe(false)
    expect(wrapper.find('.comment-attachment-item').exists()).toBe(false)
  })

  it('legacy AI-саммари (author="user"): вложения ОСТАЮТСЯ', async () => {
    const getMock = vi.mocked(apiClient.get)
    getMock.mockReset()
    getMock.mockImplementation((url) =>
      String(url).includes('/comments/54/attachments')
        ? Promise.resolve({ data: ATTS })
        : Promise.resolve({ data: [] }))
    const legacy = {
      ...COMMENT, id: 54, author: 'user', direction: 'in',
      inbox_item_id: 128, body: 'Клиент уточнил, что нужен новый документ.'
    }
    const wrapper = mountList([legacy], [INBOX_ITEM])
    await flushPromises()
    expect(wrapper.find('.comment-attachments').exists()).toBe(true)
    expect(wrapper.find('.comment-attachment-item a').text()).toBe('фото.png')
  })
})


// Хотфикс v0.27.3: цитата (quote) из письма — дословные 1–3 строки,
// хранятся в verdict_json (строка-JSON {"quote": "…"}) коммента-саммари.
// Рендер — блок «Затронуто в письме» МЕЖДУ бейджем «AI-саммари» (header)
// и body (саммари). Источники: свой verdict_json → fallback ai_verdict-дубль
// того же inbox_item_id. Пусто/нет/битый JSON → блок не рендерится.
// ai_verdict-комментарии quote НЕ несут (решение 15.09).
describe('CommentList (хотфикс v0.27.3 — цитата в AI-саммари)', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    vi.mocked(useAuthStore).mockReturnValue({ user: { username: 'admin' } })
    mockAttachments()
  })

  const INBOX_ITEM = {
    id: 140,
    from_name: 'Мария Клиентова',
    from_contact: 'mariya@mail.ru',
    subject: 'Бесплатно?',
    body_text: 'Добрый день! Подскажите, разработка будет бесплатная?\nМир'
  }
  const QUOTE_TEXT = 'разработка будет бесплатная'
  // Новый саммари (формат после 15.09): kind=user_comment, direction=in,
  // inbox_item_id, verdict_json = строка-JSON с quote.
  const QUOTE_SUMMARY = {
    ...COMMENT,
    id: 70,
    author: 'Мария Клиентова',
    direction: 'in',
    inbox_item_id: 140,
    verdict_json: JSON.stringify({ quote: QUOTE_TEXT }),
    body: 'Клиент уточняет, что разработка будет бесплатной.'
  }
  const QUOTE_VERDICT_DUPLICATE = {
    ...COMMENT,
    id: 71,
    author: 'ai',
    kind: 'ai_verdict',
    direction: 'in',
    inbox_item_id: 140,
    verdict_json: JSON.stringify({ quote: QUOTE_TEXT }),
    body: 'Клиент уточняет, что разработка будет бесплатной.'
  }

  it('саммари user_comment с verdict_json.quote: блок «Затронуто в письме» + текст quote; body — чистый саммари', () => {
    const wrapper = mountList([QUOTE_SUMMARY], [INBOX_ITEM])
    const block = wrapper.find('.comment-quote')
    expect(block.exists()).toBe(true)
    expect(block.text()).toContain('Затронуто в письме')
    expect(block.text()).toContain(QUOTE_TEXT)
    // body — саммари по-прежнему, quote в body не дублируется
    expect(wrapper.find('.comment-body').text()).toContain('Клиент уточняет, что разработка будет бесплатной.')
    // бейдж + detail v0.27.2 — не сломаны
    expect(wrapper.find('.ai-summary-badge').exists()).toBe(true)
    expect(wrapper.find('.comment-original').exists()).toBe(true)
  })

  it('ai_verdict-дубль (тот же inbox_item_id, тот же quote): quote НЕ показывается', () => {
    const wrapper = mountList([QUOTE_SUMMARY, QUOTE_VERDICT_DUPLICATE], [INBOX_ITEM])
    const blocks = wrapper.findAll('.comment-quote')
    // quote ровно ОДИН раз (в саммари), дубль в ai_verdict — нет
    expect(blocks.length).toBe(1)
    expect(blocks[0].text()).toContain(QUOTE_TEXT)
    // второй комментарий (ai_verdict) — блока не имеет
    const verdictComment = wrapper.findAll('.comment')[1]
    expect(verdictComment.find('.comment-quote').exists()).toBe(false)
  })

  it('fallback: саммари без quote + ai_verdict того же inbox_item_id с quote → блок виден', () => {
    const summaryNoQuote = { ...QUOTE_SUMMARY, id: 72, verdict_json: undefined }
    const wrapper = mountList([summaryNoQuote, QUOTE_VERDICT_DUPLICATE], [INBOX_ITEM])
    const block = wrapper.find('.comment-quote')
    expect(block.exists()).toBe(true)
    expect(block.text()).toContain('Затронуто в письме')
    expect(block.text()).toContain(QUOTE_TEXT)
  })

  it('пусто: quote="" → блок отсутствует (без ошибок)', () => {
    const wrapper = mountList([{ ...QUOTE_SUMMARY, id: 73, verdict_json: JSON.stringify({ quote: '' }) }], [INBOX_ITEM])
    expect(wrapper.find('.comment-quote').exists()).toBe(false)
    expect(wrapper.find('.comment-body').exists()).toBe(true)
  })

  it('пусто: verdict_json отсутствует → блок отсутствует', () => {
    const wrapper = mountList([{ ...QUOTE_SUMMARY, id: 74, verdict_json: undefined }], [INBOX_ITEM])
    expect(wrapper.find('.comment-quote').exists()).toBe(false)
  })

  it('пусто: битый JSON в verdict_json → блок отсутствует, UI не ломается', () => {
    const wrapper = mountList([{ ...QUOTE_SUMMARY, id: 75, verdict_json: '{not a json' }], [INBOX_ITEM])
    expect(wrapper.find('.comment-quote').exists()).toBe(false)
    expect(wrapper.find('.comment-body').exists()).toBe(true)
  })

  it('legacy author="user" (без quote в verdict_json): блока quote нет, бейдж + detail как есть (регрессия v0.27.2)', () => {
    const legacy = {
      ...COMMENT, id: 76, author: 'user', direction: 'in',
      inbox_item_id: 140, verdict_json: undefined,
      body: 'Клиент уточняет условия.'
    }
    const wrapper = mountList([legacy], [INBOX_ITEM])
    expect(wrapper.find('.comment-quote').exists()).toBe(false)
    expect(wrapper.find('.ai-summary-badge').exists()).toBe(true)
    expect(wrapper.find('.comment-original').exists()).toBe(true)
  })

  it('ai_verdict-комментарий (даже с quote в verdict_json): блока нет — вердикт = только саммари (решение 15.09)', () => {
    const wrapper = mountList([QUOTE_VERDICT_DUPLICATE], [INBOX_ITEM])
    expect(wrapper.find('.comment-quote').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Затронуто в письме')
  })
})
