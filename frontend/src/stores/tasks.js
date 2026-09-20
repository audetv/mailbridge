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
    // v0.28, шаг 28d: срез «План» (?plan=, серверная логика 28b). '' — выключен.
    plan: '',
    page: 1,
    per_page: 50
  })

  // v0.28, шаг 28d: вкладка «Статус» + селект. 5 статусных вкладок (v0.26, 7d)
  // → одна «Статус»; значения селекта (по плану: ?status=
  // all|active|backlog|completed|closed) → массив статусов запроса.
  const STATUS_FILTER = {
    all: [],
    active: ['new', 'in_progress'],
    backlog: ['backlog'],
    completed: ['completed'],
    closed: ['closed']
  }

  // v0.28, шаг 28e: вкладка-дропдаун «План» (решение владельца 2026-09-20,
  // аналогия Bootstrap «Tabs with dropdowns»): ТАБ — дропдаун-кнопка,
  // пункты — Все / Без плана / Сегодня / Завтра / Неделя, таб показывает
  // выбранный пункт. Значения — серверный срез 28b (?plan=) + «Все»
  // (без фильтра, 'all' → '') + «Без плана» ('none' → scheduled IS NULL).
  // «План» — БЕЗ status (любой статус).
  const PLAN_FILTER = { all: '', none: 'none', today: 'today', tomorrow: 'tomorrow', week: 'week' }

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
      if (!params.plan) delete params.plan
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

  // Вкладка-дропдаун (v0.28, шаг 28e): «Статус» (значение, дефолт «Активные»)
  // и «План» (значение, дефолт «Сегодня»). Смена вкладки сбрасывает
  // фильтр другой вкладки (plan ↔ due/status) — без перекрёстных фильтров.
  function setTab(tabKey, status = 'active') {
    if (tabKey === 'status') {
      const statuses = STATUS_FILTER[status] || []
      filters.value.statuses = [...statuses]
      filters.value.plan = ''
      filters.value.due = ''
      filters.value.sort = 'due'
      filters.value.page = 1
      fetchTasks()
      return
    }
    if (tabKey === 'plan') {
      // 'all' = без фильтра ( PLAN_FILTER.all = '' ); hasOwn — '' falsy,
      // поэтому не `|| 'today'` (сломал бы «Все»).
      const plan =
        Object.prototype.hasOwnProperty.call(PLAN_FILTER, status)
          ? PLAN_FILTER[status]
          : 'today'
      filters.value.plan = plan
      filters.value.statuses = [] // «План» — любой статус
      filters.value.due = ''
      filters.value.sort = 'due'
      filters.value.page = 1
      fetchTasks()
      return
    }
  }

  // Селект «Статус» (вкладка «Статус», v0.28/28d): значение ?status= → статусы.
  function setStatusFilter(status) {
    if (!Object.prototype.hasOwnProperty.call(STATUS_FILTER, status)) return
    filters.value.statuses = [...STATUS_FILTER[status]]
    filters.value.plan = ''
    filters.value.due = ''
    filters.value.page = 1
    fetchTasks()
  }

  // Селект «План» (вкладка «План», v0.28/28d): значение ?plan= → срез 28b.
  function setPlanFilter(plan) {
    if (!Object.prototype.hasOwnProperty.call(PLAN_FILTER, plan)) return
    filters.value.plan = PLAN_FILTER[plan]
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
    setStatusFilter,
    setPlanFilter,
    STATUS_FILTER,
    PLAN_FILTER,
    setEpic,
    fetchInboxCount,
    fetchTaskInbox,
    bulkUpdate,
    selectedTasks,
    clearSelection,
    togglePageSelection
  }
})
