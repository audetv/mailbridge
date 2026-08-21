<template>
  <div class="dashboard">
    <Toast position="top-right" />
    <header class="dashboard-header">
      <h1>Mailbridge</h1>
      <div class="header-right">
        <span class="connection-status" :class="{ connected: wsStore.connected }">
          {{ wsStore.connected ? '● Онлайн' : '○ Офлайн' }}
        </span>
        <span class="task-count">Всего задач: {{ store.total }}</span>
        <Button label="Выйти" severity="secondary" @click="handleLogout" />
      </div>
    </header>
    <main class="dashboard-content">
      <TabBar :tabs="tabItems" :activeTab="activeTab" @select="onTabSelect" />
      <FilterBar />
      <TaskTable />
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Toast from 'primevue/toast'
import Button from 'primevue/button'
import { useAuthStore } from '@/stores/auth'
import { useTasksStore } from '@/stores/tasks'
import { useWebSocket } from '@/stores/websocket'
import FilterBar from '@/components/FilterBar.vue'
import TaskTable from '@/components/TaskTable.vue'
import TabBar from '@/components/TabBar.vue'

const router = useRouter()
const route = useRoute()
const toast = useToast()
const authStore = useAuthStore()
const store = useTasksStore()
const wsStore = useWebSocket()

const activeTab = ref('active')

const tabItems = computed(() => [
  { key: 'active', label: 'Активные', count: 0 },
  { key: 'resolved', label: 'Решённые', count: 0 },
  { key: 'closed', label: 'Закрытые', count: 0 }
])

const tabStatuses = {
  active: ['new', 'in_progress'],
  resolved: ['resolved'],
  closed: ['closed']
}

onMounted(() => {
  wsStore.connect(authStore.token)
  
  // Определяем активную вкладку
  const tabFromUrl = route.query.tab
  const saved = localStorage.getItem('mailbridge_active_tab')
  
  let tab = 'active'
  if (tabFromUrl && tabStatuses[tabFromUrl]) {
    tab = tabFromUrl
  } else if (saved && tabStatuses[saved]) {
    tab = saved
  }
  
  activeTab.value = tab
  store.setStatuses(tabStatuses[tab])
})

onUnmounted(() => {
  wsStore.disconnect()
})

function onTabSelect(key) {
  activeTab.value = key
  localStorage.setItem('mailbridge_active_tab', key)
  router.replace({ query: { ...route.query, tab: key } })
  store.setStatuses(tabStatuses[key])
}

watch(() => wsStore.events.length, () => {
  const events = wsStore.events
  const latest = events[events.length - 1]
  if (!latest) return

  switch (latest.type) {
    case 'task_created':
      store.fetchTasks()
      toast.add({ severity: 'info', summary: latest.message, life: 5000 })
      break
    case 'task_updated':
      store.fetchTasks()
      toast.add({ severity: 'warn', summary: latest.message, life: 5000 })
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
  background: var(--p-surface-0);
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

.task-count {
  color: var(--p-text-muted-color);
}

.dashboard-content {
  padding: 2rem;
}
</style>