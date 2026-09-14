import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

// Шаг 0 (v0.22.1): WebSocket-надёльность стора.
// Мок WebSocket: контролируем open/close/error вручную, счётчик инстансов.
class MSocket {
  static instances = []
  constructor(url) {
    this.url = url
    this.readyState = 0 // CONNECTING
    this.sent = []
    this.onopen = null
    this.onclose = null
    this.onerror = null
    this.onmessage = null
    MSocket.instances.push(this)
  }
  send(data) {
    this.sent.push(data)
  }
  close() {
    if (this.readyState === 3) return
    this.readyState = 3
    if (this.onclose) this.onclose()
  }
  _open() {
    if (this.readyState !== 0) return
    this.readyState = 1
    if (this.onopen) this.onopen()
    // Сервер шлёт подтверждающий фрейм после auth (internal/web/websocket.go)
    if (this.onmessage) {
      this.onmessage({ data: JSON.stringify({ type: 'connected', message: 'Подключено' }) })
    }
  }
  _fail() {
    if (this.onerror) this.onerror(new Error('network'))
  }
  _close() {
    this.close()
  }
}

import { useWebSocket } from '@/stores/websocket'

function makeStore() {
  setActivePinia(createPinia())
  return useWebSocket()
}

describe('stores/websocket — надёжность (шаг 0, v0.22.1)', () => {
  beforeEach(() => {
    MSocket.instances = []
    vi.stubGlobal('WebSocket', MSocket)
    vi.stubGlobal('console', { ...console, warn: vi.fn(), error: vi.fn() })
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  it('connect создаёт сокет и шлёт auth при open; первое событие — НЕ resync', async () => {
    const store = makeStore()
    store.connect('tok-123')

    expect(MSocket.instances).toHaveLength(1)
    expect(MSocket.instances[0].url).toContain('/api/ws')

    MSocket.instances[0]._open()
    expect(store.connected).toBe(true)
    expect(MSocket.instances[0].sent).toEqual([JSON.stringify({ type: 'auth', token: 'tok-123' })])
    // Первое соединение: mount уже всё подгрузил, resync НЕ нужен
    expect(store.events.some((e) => e.type === 'resync')).toBe(false)
    store.disconnect()
  })

  it('onclose → реконнект после 3с; повторный open ЭМИТИРУЕТ resync', async () => {
    vi.useFakeTimers()
    const store = makeStore()
    store.connect('tok')
    MSocket.instances[0]._open()
    expect(MSocket.instances[0].sent).toHaveLength(1)
    store.events.length = 0

    MSocket.instances[0]._close()
    expect(store.connected).toBe(false)
    // пока таймер не отработал — нового сокета нет
    expect(MSocket.instances).toHaveLength(1)

    vi.advanceTimersByTime(3000)
    expect(MSocket.instances).toHaveLength(2)
    expect(store.connected).toBe(false)

    MSocket.instances[1]._open()
    expect(store.connected).toBe(true)
    const resyncs = store.events.filter((e) => e.type === 'resync')
    expect(resyncs).toHaveLength(1)
    expect(store.events.some((e) => e.type === 'connected')).toBe(true)
    // подписчик onEvent тоже получает resync
    store.disconnect()
  })

  it('backoff: 3с → 10с → 30с → cap 30с (без лавины)', async () => {
    vi.useFakeTimers()
    const store = makeStore()
    store.connect('tok')
    MSocket.instances[0]._open()
    store.events.length = 0

    // 1-я неудача: 3000
    MSocket.instances[0]._close()
    vi.advanceTimersByTime(2999)
    expect(MSocket.instances).toHaveLength(1)
    vi.advanceTimersByTime(1)
    expect(MSocket.instances).toHaveLength(2)

    // 2-я неудача: 10000
    MSocket.instances[1]._close()
    vi.advanceTimersByTime(9999)
    expect(MSocket.instances).toHaveLength(2)
    vi.advanceTimersByTime(1)
    expect(MSocket.instances).toHaveLength(3)

    // 3-я неудача: 30000
    MSocket.instances[2]._close()
    vi.advanceTimersByTime(29999)
    expect(MSocket.instances).toHaveLength(3)
    vi.advanceTimersByTime(1)
    expect(MSocket.instances).toHaveLength(4)

    // 4-я неудача: cap — снова 30000
    MSocket.instances[3]._close()
    vi.advanceTimersByTime(29999)
    expect(MSocket.instances).toHaveLength(4)
    vi.advanceTimersByTime(1)
    expect(MSocket.instances).toHaveLength(5)
    store.disconnect()
  })

  it('onerror → сокет закрыт, реконнект идёт (без мёртвого сокета)', async () => {
    vi.useFakeTimers()
    const store = makeStore()
    store.connect('tok')
    const first = MSocket.instances[0]
    first._open()
    first._fail()

    // onerror → close() → onclose → scheduled
    expect(first.readyState).toBe(3)
    expect(store.connected).toBe(false)

    vi.advanceTimersByTime(3000)
    expect(MSocket.instances).toHaveLength(2)
    MSocket.instances[1]._open()
    // resync, т.к. было разрывное подключение
    expect(store.events.some((e) => e.type === 'resync')).toBe(true)
    store.disconnect()
  })

  it('handleVisible: видимая вкладка + нет живого сокета → немедленный reconnect (без ожидания таймера)', async () => {
    const store = makeStore()
    store.connect('tok')
    MSocket.instances[0]._close()
    expect(store.connected).toBe(false)
    expect(MSocket.instances).toHaveLength(1)

    // Таймер на 3с ЕЩЁ НЕ отработал — handler форсит соединение сейчас.
    // (happy-dom по умолчанию visibilityState === 'visible')
    store.handleVisible()
    expect(MSocket.instances).toHaveLength(2)

    MSocket.instances[1]._open()
    expect(store.connected).toBe(true)
    store.disconnect()
  })

  it('refcount: disconnect одного из двух consumers не убивает сокет', async () => {
    const store = makeStore()
    store.connect('tok-A')
    store.connect('tok-B') // второй view, уехал в другой экран
    const sock = MSocket.instances[0]
    sock._open()

    store.disconnect() //_consumer A ушёл
    expect(sock.readyState).toBe(1) // сокет жив (CLOSED = 3)
    expect(store.connected).toBe(true)

    store.disconnect() // последний — сокет закрыт
    expect(store.connected).toBe(false)
  })

  it('disconnect очищает состояние: reconnectTimer, backoff, resync-флаг', async () => {
    vi.useFakeTimers()
    const store = makeStore()
    store.connect('tok')
    MSocket.instances[0]._open()
    MSocket.instances[0]._close() // → таймер 3с pending
    store.disconnect()

    // pending-таймер не обязан создать сокет после полного отключения
    vi.advanceTimersByTime(60000)
    expect(MSocket.instances).toHaveLength(1)

    // повторный connect — с нуля: первое open НЕ даёт resync
    store.connect('tok2')
    expect(MSocket.instances).toHaveLength(2)
    store.events.length = 0
    MSocket.instances[1]._open()
    expect(store.connected).toBe(true)
    expect(store.events.some((e) => e.type === 'resync')).toBe(false)
    store.disconnect()
  })

  it('onEvent: off() отписывает; сломанный listener отписывается сам (без повторных исключений)', async () => {
    const store = makeStore()
    store.connect('tok')
    MSocket.instances[0]._open()

    const seen = []
    const off = store.onEvent((e) => seen.push(e.type), 'test:ok')
    store.onEvent(() => {
      throw new Error('listener bug')
    }, 'test:bad')

    // Разрыв → handleVisible форсит реконнект → повторный open → resync.
    MSocket.instances[0]._close()
    store.handleVisible()
    MSocket.instances[1]._open()

    expect(store.events.filter((e) => e.type === 'resync')).toHaveLength(1)
    // Живой listener получил resync; сломанный — пойман и отписан warn'ом.
    expect(seen).toContain('resync')
    expect(console.warn).toHaveBeenCalledWith(
      'ws listener error',
      'test:bad',
      expect.any(Error)
    )

    // off() отписывает следующего живого подписчика
    off()
    const before = seen.length
    MSocket.instances[1]._close()
    store.handleVisible()
    MSocket.instances[2]._open()
    expect(seen.length).toBe(before)
    store.disconnect()
  })
})
