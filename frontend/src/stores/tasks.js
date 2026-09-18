// Хранение задач: список, фильтры, CRUD
import { defineStore } from 'pinia'
import { ref } from 'vue'
import apiClient from '@/api/client'

export const useTasksStore = defineStore('tasks', () => {
  const tasks = ref([])
  const total = ref(0)
  const loading = ref(false)
  // Выбранные задачи (bulk actions, шаг 4): ОБЪЕКТЫ из tasks (PrimeVue v5
  // isSelected — по reference). Живёт в store, а не в TaskTable — чтобы
  // навигация «выделил → заглянул в задачу → назад» не сбрасывала выбор.
  const selectedTasks = ref([])
  const inboxCount = ref(0)
  const currentTask = ref(null)
  const currentComments = ref([])
  const currentAttachments = ref([])
  const filters = ref({
    project: '',
    epic_id: '',
    statuses: ['new', 'in_progress'],
    assignee: '',
    // Персоны (v0.24, шаг 6): UUID-фильтры по ролям (заказчик/исполнитель).
    requestor_id: '',
    assignee_id: '',
    search: '',
    // Срок (v0.26, шаг 7d): сортировка + фильтр по датам — серверные ключи
    // 7a («due»/«updated») и 7d («overdue|today|tomorrow|7d|30d|none|due_pending»).
    sort: 'due',
    due: '',
    page: 1,
    per_page: 50
  })

  // Вкладки (v0.26, шаг 7d): «Все» (любой статус) + статусные. Ключи — те же,
  // что во вкладках UI («completed» — статус выполненной). План: дефолт-
  // сортировка «Все» → по активности; статусные вкладки → по срокам.
  const TAB_STATUS = {
    all: [],
    active: ['new', 'in_progress'],
    backlog: ['backlog'],
    completed: ['completed'],
    closed: ['closed']
  }
  const TAB_SORT = {
    all: 'updated',
    active: 'due',
    backlog: 'due',
    completed: 'due',
    closed: 'due'
  }

  async function fetchTasks() {
    loading.value = true
    try {
      const params = { ...filters.value }
      // статусные вкладки — статусы; «Все» — пустой массив → параметр не
      // слать (без status бек-энд отдаёт любой статус). Сортировка и due —
      // всегда отправляем (дефолты: sort=due, due='').
      const statuses = filters.value.statuses || []
      delete params.statuses
      // пусто → не слать (иначе бекенд будет парсить '' как отсутствующий)
      if (!params.epic_id) delete params.epic_id
      if (!params.requestor_id) delete params.requestor_id
      if (!params.assignee_id) delete params.assignee_id
      if (!params.assignee) delete params.assignee
      if (!params.due) delete params.due
      const query = { ...params }
      // status: только если вкладка статусная (не «Все»); бэкенд принимает
      // повторяющиеся статусы (q["status"] — axios сериализует массив).
      if (statuses.length > 0) {
        query.status = statuses
      }
      const { data } = await apiClient.get('/tasks', {
        params: query
      })
      tasks.value = data.tasks
      total.value = data.total
      // Bulk-выбор живёт в store: после refetch (новые объекты) привязываем
      // по ID обратно к свежим references, иначе «назад из задачи» ломает
      // выбор (PrimeVue isSelected — по ссылке).
      if (selectedTasks.value.length) {
        const fresh = new Map(data.tasks.map((t) => [t.id, t]))
        selectedTasks.value = selectedTasks.value.map(
          (sel) => fresh.get(sel.id) || sel
        )
      }
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

  // Вкладка (v0.26, шаг 7d): «Все» + статусные. Сбрасываем due-фильтр и
  // применяем дефолтную сортировку вкладки (план: Все → активность,
  // статусные → сроки). Фильтр по статусам — ядро вкладки.
  function setTab(tabKey) {
    if (!Object.prototype.hasOwnProperty.call(TAB_STATUS, tabKey)) return
    filters.value.statuses = [...TAB_STATUS[tabKey]]
    filters.value.sort = TAB_SORT[tabKey]
    filters.value.due = ''
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

  // Bulk actions (v0.23, шаг 4). PATCH /api/tasks {ids, changes:{status?, project?}}.
  // Возвращает {count, ...}. Локально применяет изменения к листу (WS-ресинк
  // подтянет остальное). Выбор — в store (переживает навигацию TaskDetail).
  async function bulkUpdate(ids, changes) {
    const { data } = await apiClient.patch('/tasks', { ids, changes })
    const idsSet = new Set(ids)
    for (const t of tasks.value) {
      if (!idsSet.has(t.id)) continue
      if (changes.status) t.status = changes.status
      if (changes.project) t.project = changes.project
    }
    return data
  }

  // — Bulk-выбор в store (шаг 4 v0.23) —
  function clearSelection() {
    selectedTasks.value = []
  }

  // Toggle «выбрать весь список, который передали (текущая страница)».
  // Всегда присваиваем НОВЫЙ массив — PrimeVue отслеживает identity.
  function togglePageSelection(pageTasks) {
    const pageIds = new Set(pageTasks.map((t) => t.id))
    const allPage =
      pageTasks.length > 0 &&
      pageTasks.every((t) => selectedTasks.value.some((s) => s.id === t.id))
    if (allPage) {
      selectedTasks.value = selectedTasks.value.filter((t) => !pageIds.has(t.id))
    } else {
      const have = new Set(selectedTasks.value.map((t) => t.id))
      selectedTasks.value = [
        ...selectedTasks.value,
        ...pageTasks.filter((t) => !have.has(t.id))
      ]
    }
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
    setTab,
    TAB_STATUS,
    TAB_SORT,
    setEpic,
    fetchInboxCount,
    fetchTaskInbox,
    bulkUpdate,
    selectedTasks,
    clearSelection,
    togglePageSelection
  }
})
