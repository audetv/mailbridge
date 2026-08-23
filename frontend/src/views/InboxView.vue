<template>
  <div class="inbox-view">
    <div class="inbox-header">
      <h2>Лента входящих</h2>
      <Select v-model="statusFilter" :options="statusOptions" optionLabel="label" optionValue="value"
        placeholder="Все" @change="onFilterChange" showClear class="status-filter" />
    </div>
    <DataTable :value="store.items" :loading="store.loading" paginator :rows="20" :totalRecords="store.total"
      @page="onPage" lazy stripedRows @row-click="openItem" class="inbox-table">
      <Column field="received_at" header="Дата" style="width: 140px">
        <template #body="{ data }">{{ formatDate(data.received_at) }}</template>
      </Column>
      <Column field="from_contact" header="От кого" style="width: 200px">
        <template #body="{ data }">
          <div class="from-cell">
            <div :class="{ 'unread-text': data.status === 'unread' }">{{ data.from_name || data.from_contact }}</div>
            <div class="from-email">{{ data.from_contact }}</div>
          </div>
        </template>
      </Column>
      <Column field="subject" header="Тема">
        <template #body="{ data }">
          <div class="subject-cell">
            <Badge v-if="data.status === 'unread'" severity="info" size="small" />
            <span :class="{ 'unread-text': data.status === 'unread' }">{{ data.subject }}</span>
          </div>
        </template>
      </Column>
      <Column field="ai_processed" header="AI" style="width: 80px">
        <template #body="{ data }">
          <Tag v-if="data.ai_processed === 1" severity="success" value="OK" />
          <Tag v-else-if="data.ai_processed === -1" severity="danger" value="ERR" />
          <Tag v-else severity="warn" value="..." />
        </template>
      </Column>
      <Column header="" style="width: 100px">
        <template #body="{ data }">
          <Button icon="pi pi-inbox" text size="small" @click.stop="archiveItem(data)" title="В архив" />
          <Button icon="pi pi-plus" text size="small" @click.stop="createTask(data)" title="Создать задачу" />
        </template>
      </Column>
    </DataTable>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useInboxStore } from '@/stores/inbox'
import apiClient from '@/api/client'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Badge from 'primevue/badge'
import Button from 'primevue/button'
import Select from 'primevue/select'

const router = useRouter()
const toast = useToast()
const store = useInboxStore()

const statusFilter = ref(null)

const statusOptions = [
  { label: 'Непрочитанные', value: 'unread' },
  { label: 'Прочитанные', value: 'read' },
  { label: 'Архив', value: 'archived' }
]

onMounted(() => {
  store.fetchItems()
})

function onFilterChange() {
  store.setFilter('status', statusFilter.value || '')
}

function onPage(event) {
  store.filters.page = event.page + 1
  store.fetchItems()
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleDateString('ru-RU') + ' ' + d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

function openItem(event) {
  router.push(`/inbox/${event.data.id}`)
}

async function archiveItem(item) {
  try {
    await apiClient.post(`/inbox/${item.id}/archive`)
    toast.add({ severity: 'success', summary: 'В архив', life: 2000 })
    store.fetchItems()
    store.fetchUnreadCount()
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Ошибка', life: 3000 })
  }
}

async function createTask(item) {
  try {
    await apiClient.post(`/inbox/${item.id}/task`)
    toast.add({ severity: 'success', summary: 'Задача создана', life: 2000 })
    store.fetchItems()
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Ошибка', life: 3000 })
  }
}
</script>

<style scoped>
.inbox-view {
  padding: 0;
}

.inbox-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.inbox-header h2 {
  margin: 0;
}

.status-filter {
  width: 200px;
}

.subject-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.unread-text {
  font-weight: 600;
}

.from-cell {
  display: flex;
  flex-direction: column;
}

.from-email {
  font-size: 0.8rem;
  color: var(--p-text-muted-color);
}
</style>