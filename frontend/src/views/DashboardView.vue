<template>
  <div class="dashboard">
    <Toast position="top-right" />
    <header class="dashboard-header">
      <h1>Mailbridge</h1>
      <div class="header-right">
        <span class="connection-status" :class="{ connected: wsStore.connected }">
          {{ wsStore.connected ? '● Онлайн' : '○ Офлайн' }}
        </span>
        <span class="task-count">Задачи: {{ activeCount }}</span>
        <span class="inbox-count">Входящие: {{ inboxStore.unreadCount }}</span>
        <Button label="Выйти" severity="secondary" @click="handleLogout" />
        <Button :icon="themeStore.isDark ? 'pi pi-sun' : 'pi pi-moon'" text @click="themeStore.toggleTheme()"
          title="Переключить тему" />
      </div>
    </header>
    <main class="dashboard-content">
      <TabBar :tabs="tabItems" :activeTab="activeTab" @select="onTabSelect" />
      <InboxView v-if="activeTab === 'inbox'" />
      <template v-else>
        <FilterBar />
        <TaskTable />
      </template>
    </main>
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
import { useWebSocket } from '@/stores/websocket'
import { useInboxStore } from '@/stores/inbox'
import apiClient from '@/api/client'
import FilterBar from '@/components/FilterBar.vue'
import TaskTable from '@/components/TaskTable.vue'
import TabBar from '@/components/TabBar.vue'
import InboxView from '@/views/InboxView.vue'

const themeStore = useThemeStore()
const router = useRouter()
const route = useRoute()
const toast = useToast()
const authStore = useAuthStore()
const store = useTasksStore()
const wsStore = useWebSocket()
const inboxStore = useInboxStore()

const activeTab = ref('active')
const activeCount = ref(0)

const tabItems = computed(() => [
  { key: 'inbox', label: 'Лента', count: 0 },
  { key: 'active', label: 'Активные', count: 0 },
  { key: 'backlog', label: 'Бэклог', count: 0 },
  { key: 'completed', label: 'Выполненные', count: 0 },
  { key: 'closed', label: 'Закрытые', count: 0 }
])

const tabStatuses = {
  active: ['new', 'in_progress'],
  backlog: ['backlog'],
  completed: ['completed'],
  closed: ['closed']
}

onMounted(() => {
  wsStore.connect(authStore.token)

  const tabFromUrl = route.query.tab
  const saved = localStorage.getItem('mailbridge_active_tab')

  let tab = 'active'
  if (tabFromUrl && (tabStatuses[tabFromUrl] || tabFromUrl === 'inbox')) {
    tab = tabFromUrl
  } else if (saved && (tabStatuses[saved] || saved === 'inbox')) {
    tab = saved
  }

  activeTab.value = tab
  if (tab !== 'inbox') {
    store.setStatuses(tabStatuses[tab])
  }

  fetchActiveCount()
  inboxStore.fetchUnreadCount()
})

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

function onTabSelect(key) {
  activeTab.value = key
  localStorage.setItem('mailbridge_active_tab', key)
  router.replace({ query: { ...route.query, tab: key } })
  if (key !== 'inbox') {
    store.setStatuses(tabStatuses[key])
  }
}

watch(() => wsStore.events.length, () => {
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
    case 'inbox_created':
      inboxStore.fetchItems()
      inboxStore.fetchUnreadCount()
      toast.add({ severity: 'info', summary: latest.message, life: 5000 })
      break
    case 'connected':
      toast.add({ severity: 'success', summary: latest.message, life: 2000 })
      break
  }
})

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
  background: var(--p-content-background);
  border-bottom: 1px solid var(--p-surface-200);
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
  color: var(--p-text-muted-color);
}

.connection-status.connected {
  color: var(--p-green-500);
}

.task-count,
.inbox-count {
  color: var(--p-text-muted-color);
  font-size: 1rem;
}

.dashboard-content {
  padding: 2rem;
}
</style>