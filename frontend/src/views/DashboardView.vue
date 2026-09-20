<template>
  <div class="dashboard">
    <Toast position="top-right" />
    <header class="dashboard-header">
      <h1>Mailbridge</h1>
      <div class="header-right">
        <span class="connection-status" :class="{ connected: wsStore.connected }">
          {{ wsStore.connected ? '● Онлайн' : '○ Офлайн' }}
        </span>
        <span v-if="versionInfo" class="version-badge" :title="versionTip">v{{ versionInfo.version }}</span>
        <span class="task-count">Задачи: {{ activeCount }}</span>
        <span class="inbox-count">Входящие: {{ inboxStore.unreadCount }}</span>
        <Button label="Выйти" severity="secondary" @click="handleLogout" />
        <Button
          :icon="themeStore.isDark ? 'pi pi-sun' : 'pi pi-moon'"
          text
          @click="themeStore.toggleTheme()"
          title="Переключить тему"
        />
      </div>
    </header>
    <main class="dashboard-content">
      <TabBar
        :tabs="tabItems"
        :activeTab="activeTab"
        @select="onTabSelect"
        @select-option="onTabSelectOption"
      />
      <InboxView v-if="activeTab === 'inbox'" />
      <ProjectsView v-else-if="activeTab === 'projects'" />
      <PersonsView v-else-if="activeTab === 'persons'" />
      <template v-else>
        <div class="tasks-toolbar">
          <Button
            v-if="activeTab === 'status' || activeTab === 'plan'"
            label="Создать задачу"
            icon="pi pi-plus"
            @click="createTaskDialogOpen = true"
          />
        </div>
        <FilterBar v-if="activeTab === 'status' || activeTab === 'plan'" />
        <TaskTable v-if="activeTab === 'status' || activeTab === 'plan'" />
      </template>
    </main>

    <!-- Диалог создания задачи (кнопка на вкладке «Статус»): проект выбирается -->
    <CreateTaskDialog
      :visible="createTaskDialogOpen"
      :projects="projectsStore.projects"
      @cancel="createTaskDialogOpen = false"
      @success="onTaskCreated"
    />
  </div>
</template>

<script setup>
import { useThemeStore } from '@/stores/theme'
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Toast from 'primevue/toast'
import Button from 'primevue/button'
import { useAuthStore } from '@/stores/auth'
import { useTasksStore } from '@/stores/tasks'
import { useProjectsStore } from '@/stores/projects'
import { useWebSocket } from '@/stores/websocket'
import { useInboxStore } from '@/stores/inbox'
import { useEpicsStore } from '@/stores/epics'
import apiClient from '@/api/client'
import FilterBar from '@/components/FilterBar.vue'
import TaskTable from '@/components/TaskTable.vue'
import TabBar from '@/components/TabBar.vue'
import InboxView from '@/views/InboxView.vue'
import ProjectsView from '@/views/ProjectsView.vue'
import PersonsView from '@/views/PersonsView.vue'
import CreateTaskDialog from '@/components/CreateTaskDialog.vue'

const themeStore = useThemeStore()
const router = useRouter()
const route = useRoute()
const toast = useToast()
const authStore = useAuthStore()
const store = useTasksStore()
const projectsStore = useProjectsStore()
const wsStore = useWebSocket()
const inboxStore = useInboxStore()
const epicsStore = useEpicsStore()

const activeTab = ref('status')
const activeCount = ref(0)
const createTaskDialogOpen = ref(false)

// v0.27, шаг 8: версия сборки в шапке (рядом «Онлайн/Офлайн»).
// GET /api/version — публичный (без JWT); значения «как есть» (dev/none ок),
// commit + built — в tooltip (title). Ошибка/нет ответа — бейдж скрыт.
const versionInfo = ref(null)
const versionTip = computed(() => {
  if (!versionInfo.value) return ''
  const v = versionInfo.value
  return `mailbridge v${v.version}\ncommit: ${v.commit}\nсборка: ${v.built} (UTC)`
})

async function fetchVersion() {
  try {
    const { data } = await apiClient.get('/version')
    if (data && data.version) {
      versionInfo.value = data
    }
  } catch {
    versionInfo.value = null
  }
}

// Задача создана из диалога: диалог уже ушёл на /tasks/:id —
// здесь только актуализируем счётчики, чтобы «Задачи: N» не устарело.
function onTaskCreated() {
  fetchActiveCount()
  store.fetchTasks()
}

const tabItems = computed(() => [
  { key: 'inbox', label: 'Лента', count: 0 },
  { key: 'status', label: 'Статус', count: 0, options: STATUS_OPTIONS, selected: statusFilterValue.value },
  { key: 'plan', label: 'План', count: 0, options: PLAN_OPTIONS, selected: planFilterValue.value },
  { key: 'projects', label: 'Проекты', count: 0 },
  { key: 'persons', label: 'Персоны', count: 0 }
])

// v0.28, шаг 28e: вкладки-дропдауны (Bootstrap «Tabs with dropdowns»,
// решение владельца 2026-09-20): «Статус» и «План» — дропдаун-кнопки,
// пункты — списки ниже; таб показывает выбранный пункт. Мапы значения →
// запрос — в store (tasks.STATUS_FILTER / tasks.PLAN_FILTER); здесь только
// опции и UI-рефы. PLAN: + «Все» (без фильтра) / «Без плана» (?plan=none).
const STATUS_OPTIONS = [
  { label: 'Все', value: 'all' },
  { label: 'Активные', value: 'active' },
  { label: 'Бэклог', value: 'backlog' },
  { label: 'Выполненные', value: 'completed' },
  { label: 'Закрытые', value: 'closed' }
]
const PLAN_OPTIONS = [
  { label: 'Все', value: 'all' },
  { label: 'Без плана', value: 'none' },
  { label: 'Сегодня', value: 'today' },
  { label: 'Завтра', value: 'tomorrow' },
  { label: 'Неделя', value: 'week' }
]
const statusFilterValue = ref('active')
const planFilterValue = ref('today')
// hasOwn — store-мапы не прототипные, но «all»: [] — falsy; hasOwnProperty
// честнее `!!map[v]`.
const statusKeyValid = (v) =>
  v != null && Object.prototype.hasOwnProperty.call(store.STATUS_FILTER, v)
const planKeyValid = (v) =>
  v != null && Object.prototype.hasOwnProperty.call(store.PLAN_FILTER, v)

const isKnownTab = (k) =>
  k === 'inbox' || k === 'status' || k === 'plan' || k === 'projects' || k === 'persons'

// URL — source of truth: deep-link «?project=…» переживает reload.
// Сидим в setup (до onMounted дочернего FilterBar) — селект «Проект»
// сразу покажет выбранный проект.
if (typeof route.query.project === 'string' && route.query.project !== '') {
  store.filters.project = route.query.project
  store.filters.page = 1
}
// Deep-link «?requestor_id=…» (кадр «К задачам» из Персон, шаг 6) — фильтр
// заказчика переживает reload.
if (typeof route.query.requestor_id === 'string' && route.query.requestor_id !== '') {
  store.filters.requestor_id = route.query.requestor_id
  store.filters.page = 1
}

// URL — source of truth для вкладки + селекта (вкладка «Статус» →
// ?status=, вкладка «План» → ?plan=). applyTab вызывается и при mount, и на
// каждый переход (router.push/replace), поэтому «К задачам»/ссылка «Проект»
// из таблицы и deep-link «?tab=plan&plan=today» переключают вкладку и
// селект корректно. Старые ключи ?tab=all/active/backlog/completed/closed —
// ломаются (без миграции, решение владельца): unknown → дефолт «Статус».
function applyTab(tab, statusVal, planVal) {
  if (!isKnownTab(tab)) return
  activeTab.value = tab
  if (tab === 'inbox') return
  if (tab === 'status') {
    const sv = statusKeyValid(statusVal) ? statusVal : 'active'
    statusFilterValue.value = sv
    store.setTab('status', sv)
  } else {
    // 'all' легитимно: без фильтра (deffalt 'today' только для мусора/отсутствия).
    const pv = planKeyValid(planVal) ? planVal : 'today'
    planFilterValue.value = pv
    store.setTab('plan', pv)
  }
}

function applyTabFromQuery() {
  const t = route.query.tab
  applyTab(
    typeof t === 'string' ? t : 'status',
    typeof route.query.status === 'string' ? route.query.status : undefined,
    typeof route.query.plan === 'string' ? route.query.plan : undefined
  )
}

onMounted(() => {
  wsStore.connect(authStore.token)
  fetchVersion()

  const tabFromUrl = route.query.tab
  const saved = localStorage.getItem('mailbridge_active_tab')

  let tab = 'status'
  let statusVal = undefined
  let planVal = undefined

  // URL — source of truth, fallback — localStorage (механика из 7d).
  if (typeof tabFromUrl === 'string' && isKnownTab(tabFromUrl)) {
    tab = tabFromUrl
    statusVal = typeof route.query.status === 'string' ? route.query.status : undefined
    planVal = typeof route.query.plan === 'string' ? route.query.plan : undefined
  } else if (saved && isKnownTab(saved)) {
    tab = saved
  }

  applyTab(tab, statusVal, planVal)
  fetchActiveCount()
  inboxStore.fetchUnreadCount()
})

// Переход по URL (goToTasks/ссылка проекта/deep-link) — переключаем вкладку
// + селект. watch на ?tab= и на ?status=/?plan= — один watch ловит всё:
// ?tab=status&status=all (селект «Все»), ?tab=plan&plan=week и т.д.
watch(
  () => [route.query.tab, route.query.status, route.query.plan],
  ([tab, statusVal, planVal]) => {
    if (tab === undefined && statusVal === undefined && planVal === undefined) return
    if (!isKnownTab(tab)) return // не наша вкладка — не трогаем (и старые ?tab=active — ломаются)
    if (tab === activeTab.value) return
    applyTabFromQuery()
  }
)

onUnmounted(() => {
  wsStore.disconnect()
})

async function fetchActiveCount() {
  try {
    const { data } = await apiClient.get('/tasks', {
      params: { status: ['new', 'in_progress'], page: 1, per_page: 1 }
    })
    activeCount.value = data.total || 0
  } catch {
    activeCount.value = 0
  }
}

// Клик по вкладке (TabBar @select). URL — ?tab=<key>; для «Статус»/«План» —
// ещё ?status= / ?plan= (дефолт: active / today). Старые ?status=/?plan= из
// другого таб-контекста — очищаем (без перекрёстных фильтров).
function onTabSelect(key) {
  activeTab.value = key
  localStorage.setItem('mailbridge_active_tab', key)
  const query = { ...route.query, tab: key }
  if (key === 'projects' || key === 'inbox') {
    delete query.project
    delete query.status
    delete query.plan
  }
  if (key === 'status') {
    query.status = statusFilterValue.value
    delete query.plan
  }
  if (key === 'plan') {
    query.plan = planFilterValue.value
    delete query.status
  }
  router.replace({ query })
  if (key === 'status') {
    store.setTab('status', statusFilterValue.value)
  } else if (key === 'plan') {
    store.setTab('plan', planFilterValue.value)
  }
}

// Выбор пункта во вкладке-дропдауне (TabBar @select-option). Клик по
// КНОПКЕ вкладки — только открывает меню (Bootstrap-паттерн); вкладка
// становится активной + fetch — при выборе пункта. URL — source of truth:
// ?status= / ?plan= переписывается, параметр другой вкладки — удаляется.
// «Статус»: статусы; «План»: срез ?plan= (all = без фильтра, 28e: + none).
function onTabSelectOption(key, value) {
  if (key === 'status') {
    if (!statusKeyValid(value)) return
    statusFilterValue.value = value
    if (activeTab.value === 'status') store.setStatusFilter(value)
    else store.setTab('status', value) // смена вкладки — сбросить план/due
  } else if (key === 'plan') {
    if (!planKeyValid(value)) return
    planFilterValue.value = value
    if (activeTab.value === 'plan') store.setPlanFilter(value)
    else store.setTab('plan', value)
  } else {
    return
  }
  if (activeTab.value !== key) {
    activeTab.value = key
    localStorage.setItem('mailbridge_active_tab', key)
  }
  const query = { ...route.query, tab: key }
  if (key === 'status') {
    query.status = value
    delete query.plan
  } else {
    query.plan = value
    delete query.status
  }
  router.replace({ query })
}

watch(
  () => wsStore.events.length,
  () => {
    const events = wsStore.events
    const latest = events[events.length - 1]
    if (!latest) return

    switch (latest.type) {
      case 'task_created':
        store.fetchTasks()
        fetchActiveCount()
        toast.add({ severity: 'info', summary: latest.message, life: 5000 })
        break
      case 'task_updated':
        store.fetchTasks()
        fetchActiveCount()
        toast.add({ severity: 'warn', summary: latest.message, life: 5000 })
        break
      case 'batch_update': {
        // Шаг 4 (v0.23): один пакетный WS на bulk-операцию (не N событий).
        // Данные — список затронутых task ID; перетягиваем список.
        store.fetchTasks()
        fetchActiveCount()
        toast.add({ severity: 'success', summary: latest.message, life: 3000 })
        break
      }
      case 'inbox_created':
        inboxStore.fetchItems()
        inboxStore.fetchUnreadCount()
        toast.add({ severity: 'info', summary: latest.message, life: 5000 })
        break
      // connected — ТОЛЬКО индикатор в шапе (точка). Тост «Соединено/
      // восстановлено» — убран по просьбе владельца: при rapid-click на
      // задачах каждое подключение показывало баннер — шум без смысла.
      case 'resync': {
        // Шаг 0 (v0.22.1): переподключились — события за время разрыва
        // могут быть потеряны → перетягиваем данные всех вкладок. БЕЗ F5.
        // Тост «Список обновлён» убран (там же): данные тихо подтянулись,
        // баннер маскирует вкладки.
        store.fetchTasks()
        fetchActiveCount()
        inboxStore.fetchItems()
        inboxStore.fetchUnreadCount()
        break
      }
      case 'project_created':
      case 'project_updated':
      case 'project_archived':
      case 'project_unarchived':
        projectsStore.fetchProjects({ archived: 'false' })
        toast.add({ severity: 'info', summary: latest.message, life: 3000 })
        break
      case 'epic_created':
      case 'epic_updated':
      case 'epic_deleted': {
        // перечитываем список модулей, если уже загружен для проекта
        if (epicsStore.currentProjectId) {
          epicsStore.fetchEpics(epicsStore.currentProjectId)
        }
        toast.add({ severity: 'info', summary: latest.message, life: 3000 })
        break
      }
    }
  }
)

function handleLogout() {
  wsStore.disconnect()
  authStore.logout()
  router.push('/login')
}
</script>

<style scoped>
.dashboard {
  min-height: 100vh;
}

.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 2rem;
  background: var(--mb-surface);
  border-bottom: 1px solid var(--mb-border);
}

.dashboard-header h1 {
  margin: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.connection-status {
  font-size: 0.85rem;
  color: var(--mb-text-muted);
}

.connection-status.connected {
  color: var(--mb-primary);
}

.task-count,
.inbox-count {
  color: var(--mb-text-muted);
  font-size: 1rem;
}

.version-badge {
  font-size: 0.8rem;
  color: var(--mb-text-muted);
  border: 1px solid var(--mb-border);
  border-radius: 0.375rem;
  padding: 0.1rem 0.45rem;
  cursor: default;
}

.dashboard-content {
  padding: 2rem;
}

.tab-filter-row {
  margin-bottom: 0.75rem;
}

.tab-filter-select {
  width: 180px;
}
</style>
