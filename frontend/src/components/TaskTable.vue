<template>
  <DataTable :value="store.tasks" :loading="store.loading" paginator :rows="50" :totalRecords="store.total"
    @page="onPage" lazy stripedRows>
    <Column field="id" header="ID" style="width: 80px" sortable />
    <Column field="created_at" header="Дата" style="width: 150px" sortable>
      <template #body="{ data }">
        {{ formatDate(data.created_at) }}
      </template>
    </Column>
    <Column field="from_email" header="От кого" style="width: 200px" />
    <Column field="subject" header="Тема" />
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

function onPage(event) {
  store.filters.page = event.page + 1
  store.fetchTasks()
}
</script>