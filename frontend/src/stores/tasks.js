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

    function setFilter(key, value) {
        filters.value[key] = value
        filters.value.page = 1
        fetchTasks()
    }

    return { tasks, total, loading, filters, fetchTasks, setFilter }
})