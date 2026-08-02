import { defineStore } from 'pinia'
import { ref } from 'vue'
import apiClient from '@/api/client'

export const useTasksStore = defineStore('tasks', () => {
    const tasks = ref([])
    const total = ref(0)
    const loading = ref(false)
    const currentTask = ref(null)
    const currentComments = ref([])
    const currentAttachments = ref([])
    const filters = ref({
        project: '',
        status: '',
        assignee: '',
        search: '',
        page: 1,
        perPage: 50
    })

    async function fetchTasks() {
        loading.value = true
        try {
            const { data } = await apiClient.get('/tasks', { params: filters.value })
            tasks.value = data.tasks
            total.value = data.total
        } finally {
            loading.value = false
        }
    }

    async function fetchTask(id) {
        const { data } = await apiClient.get(`/tasks/${id}`)
        currentTask.value = data.task
        currentComments.value = data.comments
        currentAttachments.value = data.attachments
        return data
    }

    async function updateTask(id, updates) {
        const { data } = await apiClient.patch(`/tasks/${id}`, updates)
        currentTask.value = data.task
        return data
    }

    async function replyTask(id, body) {
        const { data } = await apiClient.post(`/tasks/${id}/reply`, { body })
        currentComments.value.push(data.comment)
        return data
    }

    function setFilter(key, value) {
        filters.value[key] = value
        filters.value.page = 1
        fetchTasks()
    }

    return { tasks, total, loading, currentTask, currentComments, currentAttachments, filters, fetchTasks, fetchTask, updateTask, replyTask, setFilter }
})