// WebSocket-стор: соединение + auth + события для UI.
// Надёжность (step 0, v0.22.1):
// - onerror + onclose → реконнект с backoff 3с/10с/30с (cap), без лавины;
// - visibilitychange/pageshow → если вкладка видимая, а соединения нет —
//   форс-реконнект (в замороженном табе Chrome таймеры троттлятся —
//   событие могло «проскочить», а таймер сработать в вакууме);
// - при повторном open — событие `resync`: views перетягивают данные
//   (за первое соединение при mount всё и так подгружается);
// - `off()`-обёртка подписки: view обязан отписаться в onUnmounted, иначе
//   убитый Vue-компонент продолжает ловить события (resync → 3 GET-а в
//   консоль на каждый реконнект, URL-заглушки и т.п.);
// - refcount на connect/disconnect: переход Dashboard → Inbox не убивает
//   соединение, которое нужен другой consumer.
import { ref } from 'vue'
import { defineStore } from 'pinia'

// Backoff: 3с → 10с → 30с, дальше cap (без лавины при долгом down)
const RECONNECT_DELAYS = [3000, 10000, 30000]

export const useWebSocket = defineStore('websocket', () => {
  const connected = ref(false)
  const events = ref([])
  const listeners = new Set()

  let ws = null
  let reconnectTimer = null
  let refCount = 0
  let delayIdx = 0
  let everConnected = false
  // token последнего connect() — при реконнекте передаём его снова
  let authToken = ''
  // Счётчик неудачных подключений подряд — для диагностического warn без spam.
  let deadConns = 0

  function emit(event) {
    events.value.push(event)
    // Ограничиваем историю
    if (events.value.length > 100) events.value.shift()
    for (const [fn, label] of listeners) {
      try {
        fn(event)
      } catch (e) {
        // console.error здесь создавал бы собственный шум в консоли — warn.
        console.warn('ws listener error', label, e)
        // Сломанный listener (исключительно из-за его бага) отписываем,
        // чтобы не бросать исключения на каждом событии.
        listeners.delete([fn, label])
      }
    }
  }

  function resetTimer() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  function scheduleReconnect(delay) {
    if (reconnectTimer || refCount === 0) return
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      if (refCount > 0) ensureSocket()
    }, delay)
  }

  function ensureSocket() {
    if (ws || reconnectTimer) return

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${location.host}/api/ws`
    ws = new WebSocket(url)

    ws.onopen = () => {
      if (authToken) ws.send(JSON.stringify({ type: 'auth', token: authToken }))
      if (deadConns >= 3) {
        // «После N неудач — предупреждение»: видно в консоли, что соединение
        // нестабильно, без spam от каждого close.
        console.warn(
          `ws: ${deadConns} неудачных подключений подряд — соединение нестабильно`
        )
        deadConns = 0
      }
      // Индикатор загорится после ПОДТВЕРЖДЕНИЯ сервера (фрейм connected
      // после auth) — «связь есть» ≠ «сокет открыт». Первый раз при mount
      // всё и так подгружено (fetch в onMounted) → resync только при
      // ПОВТОРНОМ подключении: за разрыв могли прийти события, которых
      // UI не видел.
      if (everConnected) {
        emit({ type: 'resync', message: 'Переподключено — данные обновлены' })
      }
      everConnected = true
      delayIdx = 0
    }

    ws.onclose = () => {
      connected.value = false
      ws = null
      deadConns += 1
      // backoff растёт с каждой неудачной попыткой; после успешного open
      // обнуляется (delayIdx = 0 в onopen)
      scheduleReconnect(RECONNECT_DELAYS[Math.min(delayIdx, RECONNECT_DELAYS.length - 1)])
      delayIdx = Math.min(delayIdx + 1, RECONNECT_DELAYS.length - 1)
    }

    ws.onerror = (e) => {
      // Без он-хендлера ошибка молча пропадала. Браузер обычно сам «догаляет»
      // сокетом (onclose → реконнект идёт оттуда); close() здесь — страховка
      // на случай «застрявшего» onClose (close идемпотентен).
      console.warn('ws error', e)
      if (ws) ws.close()
    }

    ws.onmessage = (e) => {
      let event
      try {
        event = JSON.parse(e.data)
      } catch {
        return
      }
      if (event?.type === 'connected') {
        // Сервер подтвердил auth: соединение рабочее.
        connected.value = true
      }
      emit(event)
    }
  }

  function killSocket() {
    resetTimer()
    connected.value = false
    // everConnected НЕ сбрасываем здесь: «было разрывное подключение»
    // решается на верхнем уровне (teardown сбрасывает, реконнект — нет).
    if (ws) {
      // onclose сработает сам, он идемпотентен
      const sock = ws
      ws = null
      sock.close()
    }
    delayIdx = 0
  }

  // Полный снос состояния: первое успешное соединение после «с нуля»
  // не считается реконнектом — resync не нужен.
  function teardown() {
    killSocket()
    everConnected = false
  }

  function connect(token) {
    refCount += 1
    authToken = token
    // Consumer вернулся (или первый) — немедленный старт, без ожидания
    // pending-таймера от прошлой неудачной попытки.
    resetTimer()
    ensureSocket()
  }

  // disconnect() — «этот consumer больше не слушает». Сокет закрывается
  // только когда последний consumer отпустил (refCount → 0).
  function disconnect() {
    if (refCount > 0) refCount -= 1
    if (refCount > 0) return
    teardown()
  }

  function markAsRead(taskId) {
    if (ws && ws.readyState === 1 /* OPEN */) {
      ws.send(JSON.stringify({ type: 'mark_read', taskId }))
    }
  }

  // Подписка на WS-события (view'ы). label — что это за подписчик
  // (для диагностического warn в emit; без него — имя функции).
  // Возвращает функцию отписки: ОБЯЗАН быть вызвана в onUnmounted —
  // живой стор живёт дольше компонента, и «забытая» подписка будет
  // дёргать мёртвый view на каждом resync.
  function onEvent(fn, label) {
    const pair = [fn, label || (fn && fn.name) || 'listener']
    listeners.add(pair)
    return () => listeners.delete(pair)
  }

  // ── Надёжность: возврат из фоновой вкладки ─────────────────────────────
  // Вкладка была заморожена/скрыта (throttled-таймеры, потерянное событие)
  // → при видимости форсим реконнект, если соединения нет/оно не открыто.
  function handleVisible() {
    if (document.visibilityState !== 'visible') return
    if (refCount === 0) return
    if (ws && ws.readyState === 1 /* OPEN */) return
    // Pending-таймер сбрасываем: во фоновом табе браузер троттлил
    // setTimeout (30с+), и «реконнект уже в работе» — может и не быть.
    // everConnected НЕ трогаем: был разрыв → при open emit resync.
    killSocket()
    ensureSocket()
  }

  if (typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', () => handleVisible())
    // bfcache-возврат: сокет мёртв, таймеры «в вакууме» — чистый старт.
    // everConnected НЕ сбрасываем: были события, которые UI мог не получить
    // → resync при open (аналогично handleVisible).
    window.addEventListener('pageshow', (e) => {
      if (!e?.persisted) return
      killSocket()
      if (refCount > 0) ensureSocket()
    })
  }

  return { connected, events, connect, disconnect, markAsRead, onEvent, handleVisible }
})
