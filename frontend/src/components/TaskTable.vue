<template>
  <DataTable :value="store.tasks" :loading="store.loading" paginator :rows="50" :totalRecords="store.total"
    @page="onPage" lazy stripedRows :rowClass="rowClass" @row-click="onRowClick">
    <Column field="id" header="ID" style="width: 80px" />
    <Column field="created_at" header="Дата" style="width: 150px">
      <template #body="{ data }">
        {{ formatDate(data.created_at) }}
      </template>
    </Column>
    <Column field="from_email" header="От кого" style="width: 200px" />
    <Column field="subject" header="Тема">
      <template #body="{ data }">
        <div class="subject-cell">
          <Badge v-if="data.unread_comments > 0" :value="data.unread_comments" severity="info" size="small"
            class="unread-badge" />
          <span>{{ data.subject }}</span>
        </div>
      </template>
    </Column>
    <Column field="type" header="Тип" style="width: 100px">
      <template #body="{ data }">
        <Tag :value="data.type" v-if="data.type" />
      </template>
    </Column>
    <Column field="priority" header="Приоритет" style="width: 100px">
      <template #body="{ data }">
        <Tag :value="data.priority" :severity="prioritySeverity(data.priority)" />
      </template>
    </Column>
    <Column field="project" header="Проект" style="width: 150px" />
    <Column field="status" header="Статус" style="width: 120px">
      <template #body="{ data }">
        <Tag :value="statusLabel(data.status)" :severity="statusSeverity(data.status)" />
      </template>
    </Column>
    <Column field="assignee" header="Исполнитель" style="width: 120px" />
  </DataTable>
</template>

<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useTasksStore } from '@/stores/tasks'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Badge from 'primevue/badge'

const store = useTasksStore()
const router = useRouter()

onMounted(() => {
  store.fetchTasks()
})

function formatDate(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleDateString('ru-RU') + ' ' + d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

function statusLabel(status) {
  const labels = { new: 'Новая', in_progress: 'В работе', resolved: 'Решена', closed: 'Закрыта' }
  return labels[status] || status
}

function statusSeverity(status) {
  const map = { new: 'info', in_progress: 'warn', resolved: 'success', closed: 'secondary' }
  return map[status] || 'info'
}

function prioritySeverity(priority) {
  const map = { urgent: 'danger', high: 'warn', medium: 'info', low: 'success' }
  return map[priority] || 'info'
}

function rowClass(task) {
  return task.unread_comments > 0 ? 'task-unread' : ''
}

function onRowClick(event) {
  router.push(`/tasks/${event.data.id}`)
}

function onPage(event) {
  store.filters.page = event.page + 1
  store.fetchTasks()
}
</script>

<style scoped>
.subject-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.unread-badge {
  flex-shrink: 0;
}

:deep(.task-unread) {
  font-weight: 600;
  background: var(--p-primary-50) !important;
}

:deep(.task-unread td:first-child) {
  border-left: 3px solid var(--p-primary-500);
}
</style>