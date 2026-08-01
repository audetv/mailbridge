<template>
  <div class="dashboard">
    <header class="dashboard-header">
      <h1>Mailbridge</h1>
      <div class="header-right">
        <span class="task-count">Всего задач: {{ store.total }}</span>
        <Button label="Выйти" severity="secondary" @click="handleLogout" />
      </div>
    </header>
    <main class="dashboard-content">
      <FilterBar />
      <TaskTable @row-click="openTask" />
    </main>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useTasksStore } from '@/stores/tasks'
import FilterBar from '@/components/FilterBar.vue'
import TaskTable from '@/components/TaskTable.vue'
import Button from 'primevue/button'

const router = useRouter()
const authStore = useAuthStore()
const store = useTasksStore()

function handleLogout() {
  authStore.logout()
  router.push('/login')
}

function openTask(task) {
  // Будет реализовано в этапе 19.3
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

.task-count {
  color: var(--p-text-muted-color);
}

.dashboard-content {
  padding: 2rem;
}
</style>