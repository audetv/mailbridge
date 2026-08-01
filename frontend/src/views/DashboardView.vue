<template>
  <div class="dashboard">
    <header class="dashboard-header">
      <h1>Mailbridge</h1>
      <Button label="Выйти" severity="secondary" @click="handleLogout" />
    </header>
    <main class="dashboard-content">
      <Card>
        <template #content>
          <p>Дашборд в разработке. Задачи загружаются...</p>
        </template>
      </Card>
    </main>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Card from 'primevue/card'
import Button from 'primevue/button'

const router = useRouter()
const authStore = useAuthStore()

function handleLogout() {
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

.dashboard-content {
  padding: 2rem;
}
</style>