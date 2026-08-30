import { defineStore } from 'pinia'
import { ref } from 'vue'
import apiClient from '@/api/client'

// Модули (эпики) проекта: список, CRUD, прогресс.
// WS-события epic_* потребляются DashboardView (fetchEpics + тосты).
export const useEpicsStore = defineStore('epics', () => {
  const epics = ref([])
  const loading = ref(false)
  const error = ref('')
  const detail = ref(null)
  const progress = ref(null)
  const filterEpicId = ref(null)

  // Список модулей проекта (по ID проекта)
  async function fetchEpics(projectId) {
    if (!projectId) {
      epics.value = []
      return
    }
    loading.value = true
    error.value = ''
    try {
      const { data } = await apiClient.get(`/projects/${projectId}/epics`)
      epics.value = data
    } catch (e) {
      error.value = e.response?.data?.error || e.message
      epics.value = []
    } finally {
      loading.value = false
    }
  }

  async function refresh(projectId) {
    if (!projectId) return
    await fetchEpics(projectId)
    if (filterEpicId.value) {
      await fetchDetail(filterEpicId.value)
    }
  }

  async function createEpic(projectId, name, status = 'open') {
    const { data } = await apiClient.post(`/projects/${projectId}/epics`, { name, status })
    await fetchEpics(projectId)
    return data
  }

  async function renameEpic(id, name, description, status) {
    const { data } = await apiClient.put(`/epics/${id}`, { name, description, status })
    await refresh(currentProjectId.value)
    return data
  }

  async function deleteEpic(id) {
    const { data } = await apiClient.delete(`/epics/${id}`)
    if (filterEpicId.value === id) setFilter(null)
    await fetchEpics(currentProjectId.value)
    return data
  }

  // Детали + прогресс (Bar). API возвращает FLAT объект: поля эпика + progress
  // (см. PLAN.steps10-11.md §1.1 и internal/web/epics.go GetEpicDetail).
  async function fetchDetail(id) {
    const { data } = await apiClient.get(`/epics/${id}`)
    detail.value = data
    progress.value = data.progress || { open: 0, done: 0, total: 0 }
    return data
  }

  // Привязка / отвязка задачи
  async function linkTask(epicId, taskId) {
    await apiClient.post(`/epics/${epicId}/tasks/${taskId}`)
    await fetchDetail(epicId)
  }

  async function unlinkTask(epicId, taskId) {
    await apiClient.delete(`/epics/${epicId}/tasks/${taskId}`)
    await fetchDetail(epicId)
  }

  const currentProjectId = ref(null)

  function setProject(id) {
    currentProjectId.value = id
    filterEpicId.value = null
    detail.value = null
    progress.value = null
    fetchEpics(id)
  }

  function setFilter(epicId) {
    filterEpicId.value = epicId
    if (epicId) {
      fetchDetail(epicId)
    } else {
      detail.value = null
      progress.value = null
    }
  }

  function epicById(id) {
    return epics.value.find((e) => e.id === id) || null
  }

  return {
    epics,
    loading,
    error,
    detail,
    progress,
    filterEpicId,
    currentProjectId,
    fetchEpics,
    refresh,
    createEpic,
    renameEpic,
    deleteEpic,
    fetchDetail,
    linkTask,
    unlinkTask,
    setProject,
    setFilter,
    epicById
  }
})
