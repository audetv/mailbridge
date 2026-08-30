// Хранение задач: список, фильтры, CRUD
import { defineStore } from 'pinia'
import { ref } from 'vue'
import apiClient from '@/api/client'

export const useTasksStore = defineStore('tasks', () => {
  const tasks = ref([])
  const total = ref(0)
  const loading = ref(false)
  const inboxCount = ref(0)
  const currentTask = ref(null)
  const currentComments = ref([])
  const currentAttachments = ref([])
  const filters = ref({
    project: '',
    epic_id: '',
    statuses: ['new', 'in_progress'],
    assignee: '',
    search: '',
    page: 1,
    per_page: 50
  })

  async function fetchTasks() {
    loading.value = true
    try {
      const params = { ...filters.value }
      delete params.statuses
      // пусто → не слать (иначе бекенд будет парсить '' как отсутствующий)
      if (!params.epic_id) delete params.epic_id
      const { data } = await apiClient.get('/tasks', {
        params: {
          ...params,
          status: filters.value.statuses
        }
      })
      tasks.value = data.tasks
      total.value = data.total
    } finally {
      loading.value = false
    }
  }

  async function fetchInboxCount() {
    try {
      const { data } = await apiClient.get('/inbox', {
        params: { status: 'unread', page: 1, per_page: 1 }
      })
      inboxCount.value = data.total || 0
    } catch {
      inboxCount.value = 0
    }
  }

  async function fetchTaskInbox(id) {
    const { data } = await apiClient.get(`/tasks/${id}/inbox`)
    return data
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

  async function replyTask(id, body, kind) {
    const payload = { body }
    if (kind) payload.kind = kind
    const { data } = await apiClient.post(`/tasks/${id}/reply`, payload)
    currentComments.value.push(data.comment)
    return data
  }

  // Утверждение ответа пользователю (ФАЗА 4): PATCH /api/comments/{id}/approve.
  // Бэкенд: 200 + {"comment": {...}} (approved=1). Idempotent. WS comment_approved.
  async function approveComment(commentId) {
    const { data } = await apiClient.patch(`/comments/${commentId}/approve`)
    const c = currentComments.value.find((x) => x.id === commentId)
    if (c) c.approved = 1
    return data
  }

  // WS-событие comment_approved: синхронизация бейджа «Утверждён».
  // event = {type, taskId, message, data: {…comment}}.
  function applyCommentApproved(event) {
    const c =
      currentComments.value.find((x) => x.id === event?.data?.id) ||
      currentComments.value.find((x) => x.id === event?.data?.comment?.id)
    if (c && typeof c.approved === 'undefined') c.approved = null
    if (c) c.approved = 1
  }

  // Ручное создание задачи. body: {title, project, description?, epic_id?}.
  // Статус всегда new (на сервере). Возвращает созданную задачу (data) с id.
  async function createTask(body) {
    const { data } = await apiClient.post('/tasks', body)
    return data
  }

  async function markAsRead(id) {
    await apiClient.post(`/tasks/${id}/mark-read`)
  }

  function setFilter(key, value) {
    filters.value[key] = value
    filters.value.page = 1
    fetchTasks()
  }

  function setStatuses(statuses) {
    filters.value.statuses = statuses
    filters.value.page = 1
    fetchTasks()
  }

  // Привязка / отвязка задачи к модулю (epic). epicId = null — отвязать.
  async function setEpic(taskId, epicId) {
    const { data } = await apiClient.patch(`/tasks/${taskId}`, { epic_id: epicId ?? null })
    const t = tasks.value.find((x) => x.id === taskId)
    if (t) t.epic_id = epicId ?? null
    if (currentTask.value && currentTask.value.id === taskId) {
      currentTask.value.epic_id = epicId ?? null
    }
    return data
  }

  return {
    tasks,
    total,
    loading,
    inboxCount,
    currentTask,
    currentComments,
    currentAttachments,
    filters,
    fetchTasks,
    fetchTask,
    createTask,
    updateTask,
    replyTask,
    approveComment,
    applyCommentApproved,
    markAsRead,
    setFilter,
    setStatuses,
    setEpic,
    fetchInboxCount,
    fetchTaskInbox
  }
})
