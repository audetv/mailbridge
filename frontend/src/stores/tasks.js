import { defineStore } from 'pinia'
import { ref } from 'vue'
import apiClient from '@/api/client'

export const useTasksStore = defineStore('tasks', () => {
    const tasks = ref([])
    const total = ref(0)
    const loading = ref(false)
    const filters = ref({
        project: '',
        status: '',
        assignee: '',
        search: '',
        page: 1,
        perPage: 50
    })

    let eventSource = null
    let reconnectTimer = null

    const RECONNECT_DELAY = 3000
    const MAX_RECONNECT_DELAY = 30000
    let currentDelay = RECONNECT_DELAY

    // ── Данные ───────────────────────────────────────────────────

    async function fetchTasks({ silent = false } = {}) {
        if (!silent) loading.value = true
        try {
            const { data } = await apiClient.get('/tasks', { params: filters.value })
            tasks.value = data.tasks
            total.value = data.total
        } finally {
            if (!silent) loading.value = false
        }
    }

    function setFilter(key, value) {
        filters.value[key] = value
        filters.value.page = 1
        fetchTasks()
    }

    // ── SSE ──────────────────────────────────────────────────────

    function connectEvents() {
        cleanup()

        eventSource = new EventSource('/api/tasks/events')

        eventSource.onmessage = (e) => {
            try {
                const event = JSON.parse(e.data)
                if (['task_created', 'task_updated', 'task_comment'].includes(event.type)) {
                    fetchTasks({ silent: true })
                }
            } catch {
                // игнорируем ошибки парсинга
            }
        }

        eventSource.onopen = () => {
            console.log('SSE connected')
            currentDelay = RECONNECT_DELAY
        }

        eventSource.onerror = () => {
            console.warn('SSE error, readyState:', eventSource?.readyState)

            if (eventSource && eventSource.readyState === EventSource.CLOSED) {
                scheduleReconnect()
            }
        }
    }

    function scheduleReconnect() {
        cleanup()
        if (reconnectTimer) return

        console.log(`SSE reconnect in ${currentDelay / 1000}s`)
        reconnectTimer = setTimeout(() => {
            reconnectTimer = null
            connectEvents()
        }, currentDelay)

        currentDelay = Math.min(currentDelay * 2, MAX_RECONNECT_DELAY)
    }

    function cleanup() {
        if (eventSource) {
            eventSource.onmessage = null
            eventSource.onopen = null
            eventSource.onerror = null
            eventSource.close()
            eventSource = null
        }
        if (reconnectTimer) {
            clearTimeout(reconnectTimer)
            reconnectTimer = null
        }
    }

    function disconnectEvents() {
        cleanup()
    }

    // ── Экспорт ──────────────────────────────────────────────────

    return {
        tasks,
        total,
        loading,
        filters,
        fetchTasks,
        setFilter,
        connectEvents,
        disconnectEvents
    }
})