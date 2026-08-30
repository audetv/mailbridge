import { ref } from 'vue'
import { defineStore } from 'pinia'

export const useWebSocket = defineStore('websocket', () => {
  const connected = ref(false)
  const events = ref([])
  let ws = null
  let reconnectTimer = null
  const listeners = new Set()

  function emit(event) {
    events.value.push(event)
    // Ограничиваем историю
    if (events.value.length > 100) events.value.shift()
    for (const fn of listeners) {
      try {
        fn(event)
      } catch (e) {
        console.error('ws listener error', e)
      }
    }
  }

  function connect(token) {
    if (ws) return

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${location.host}/api/ws`
    ws = new WebSocket(url)

    ws.onopen = () => {
      connected.value = true
      ws.send(JSON.stringify({ type: 'auth', token }))
    }

    ws.onclose = () => {
      connected.value = false
      ws = null
      reconnectTimer = setTimeout(() => connect(token), 3000)
    }

    ws.onmessage = (e) => {
      const event = JSON.parse(e.data)
      emit(event)
    }
  }

  function disconnect() {
    if (reconnectTimer) clearTimeout(reconnectTimer)
    if (ws) {
      ws.close()
      ws = null
    }
    connected.value = false
  }

  function markAsRead(taskId) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'mark_read', taskId }))
    }
  }

  // Подписка на WS-события (view'ы). Возвращает функцию отписки.
  function onEvent(fn) {
    listeners.add(fn)
    return () => listeners.delete(fn)
  }

  return { connected, events, connect, disconnect, markAsRead, onEvent }
})
