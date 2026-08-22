<template>
  <div class="inbox-view">
    <div class="inbox-header">
      <h2>Лента входящих</h2>
      <div class="inbox-filters">
        <Select v-model="statusFilter" :options="statusOptions" optionLabel="label" optionValue="value"
          placeholder="Все" @change="onFilterChange" showClear />
      </div>
    </div>
    <DataTable :value="items" :loading="loading" paginator :rows="20" :totalRecords="total"
      @page="onPage" lazy stripedRows>
      <Column field="id" header="ID" style="width: 60px" />
      <Column field="received_at" header="Дата" style="width: 140px">
        <template #body="{ data }">{{ formatDate(data.received_at) }}</template>
      </Column>
      <Column field="from_contact" header="От кого" style="width: 180px" />
      <Column field="subject" header="Тема">
        <template #body="{ data }">
          <div class="subject-cell">
            <Badge v-if="data.status === 'unread'" severity="info" size="small" />
            <span :class="{ 'unread-text': data.status === 'unread' }">{{ data.subject }}</span>
          </div>
        </template>
      </Column>
      <Column field="ai_processed" header="AI" style="width: 60px">
        <template #body="{ data }">
          <Tag v-if="data.ai_processed === 1" severity="success" value="OK" />
          <Tag v-else-if="data.ai_processed === -1" severity="danger" value="ERR" />
          <Tag v-else severity="warn" value="..." />
        </template>
      </Column>
      <Column header="Действия" style="width: 160px">
        <template #body="{ data }">
          <Button icon="pi pi-eye" text size="small" @click="openItem(data)" />
          <Button icon="pi pi-inbox" text size="small" @click="archiveItem(data)" />
          <Button icon="pi pi-plus" text size="small" @click="createTask(data)" />
        </template>
      </Column>
    </DataTable>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import apiClient from '@/api/client'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Badge from 'primevue/badge'
import Button from 'primevue/button'
import Select from 'primevue/select'

const router = useRouter()
const toast = useToast()

const items = ref([])
const total = ref(0)
const loading = ref(false)
const statusFilter = ref(null)
const page = ref(1)

const statusOptions = [
  { label: 'Непрочитанные', value: 'unread' },
  { label: 'Прочитанные', value: 'read' },
  { label: 'Архив', value: 'archived' }
]

onMounted(() => {
  fetchItems()
})

async function fetchItems() {
  loading.value = true
  try {
    const params = { page: page.value, per_page: 20 }
    if (statusFilter.value) params.status = statusFilter.value
    const { data } = await apiClient.get('/inbox', { params })
    items.value = data.items
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function onFilterChange() {
  page.value = 1
  fetchItems()
}

function onPage(event) {
  page.value = event.page + 1
  fetchItems()
}

function formatDate(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleDateString('ru-RU') + ' ' + d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

function openItem(item) {
  router.push(`/inbox/${item.id}`)
}

async function archiveItem(item) {
  try {
    await apiClient.post(`/inbox/${item.id}/archive`)
    toast.add({ severity: 'success', summary: 'В архив', life: 2000 })
    fetchItems()
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Ошибка', life: 3000 })
  }
}

async function createTask(item) {
  try {
    await apiClient.post(`/inbox/${item.id}/task`)
    toast.add({ severity: 'success', summary: 'Задача создана', life: 2000 })
    fetchItems()
  } catch (e) {
    toast.add({ severity: 'error', summary: 'Ошибка', life: 3000 })
  }
}
</script>

<style scoped>
.inbox-view {
  padding: 1rem;
}

.inbox-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.subject-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.unread-text {
  font-weight: 600;
}
</style>