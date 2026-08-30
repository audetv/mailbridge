<template>
  <Card class="epic-panel">
    <template #title>Модули — {{ project.name }}</template>
    <template #subtitle>Модули (эпики) проекта и прогресс по задачам</template>

    <template #content>
      <div v-if="list.length" class="epics-list">
        <div v-for="e in list" :key="e.id" class="epic-row">
          <span class="epic-number">#{{ e.number }}</span>
          <span class="epic-name">{{ e.name }}</span>
          <div class="epic-progress">
            <ProgressBar
              v-if="barOf(e) !== null"
              :value="barOf(e)"
              class="bar"
              style="width: 130px"
              :showValue="false"
            />
            <span v-else class="muted">прогресс —</span>
          </div>
          <SelectButton
            v-model="draft[e.id]"
            :options="statusOptions"
            optionLabel="label"
            optionValue="value"
            :defaultModelValue="e.status"
            size="small"
            :allowEmpty="false"
            aria-label="Статус модуля"
            @change="saveStatus(e)"
          />
          <Button
            label="Удалить"
            severity="danger"
            text
            size="small"
            icon="pi pi-trash"
            :loading="busy"
            @click="remove(e)"
          />
        </div>
      </div>
      <div v-else class="muted empty">Модулей пока нет — создайте первый ниже.</div>

      <div class="create-row">
        <InputText
          v-model="newName"
          placeholder="Имя модуля"
          maxlength="128"
          @keyup.enter="add"
        />
        <Button
          label="Добавить модуль"
          icon="pi pi-plus"
          :loading="busy"
          :disabled="!newName.trim()"
          @click="add"
        />
      </div>

      <Message v-if="error" severity="error" :closable="true" @close="error = ''" class="err">
        {{ error }}
      </Message>
    </template>
  </Card>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useToast } from 'primevue/usetoast'
import Card from 'primevue/card'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import ProgressBar from 'primevue/progressbar'
import SelectButton from 'primevue/selectbutton'
import Message from 'primevue/message'
import { useEpicsStore } from '@/stores/epics'

const props = defineProps({
  project: { type: Object, required: true } // { id, name }
})

const store = useEpicsStore()
const toast = useToast()

const list = computed(() => store.epics)
const busy = ref(false)
const error = ref('')
const newName = ref('')
const bars = reactive({}) // epicId -> процент 0..100
const draft = reactive({}) // epicId -> статус (draft для SelectButton)

const statusOptions = [
  { label: 'Открыт', value: 'open' },
  { label: 'В работе', value: 'in_progress' },
  { label: 'Готов', value: 'done' }
]

function barOf(e) {
  return bars[e.id] !== undefined ? bars[e.id] : null
}

function syncDraft() {
  for (const e of store.epics) {
    draft[e.id] = e.status
  }
}

async function loadAll() {
  error.value = ''
  Object.keys(bars).forEach((k) => delete bars[k])
  await store.fetchEpics(props.project.id)
  syncDraft()
  // Прогресс по каждому модулю (detail: flat + progress). Неполный прогресс не ломает список.
  await Promise.all(
    store.epics.map(async (e) => {
      try {
        const d = await store.fetchDetail(e.id)
        const p = d && d.progress
        if (p && p.total > 0) {
          bars[e.id] = Math.round((p.done / p.total) * 100)
        }
      } catch {
        /* без прогресса — строка всё равно видна */
      }
    })
  )
}

async function add() {
  const name = newName.value.trim()
  if (!name || busy.value) return
  busy.value = true
  error.value = ''
  try {
    const created = await store.createEpic(props.project.id, name, 'open')
    newName.value = ''
    syncDraft()
    try {
      const d = await store.fetchDetail(created.id)
      const p = d && d.progress
      if (p && p.total > 0) bars[created.id] = Math.round((p.done / p.total) * 100)
    } catch {
      /* ок */
    }
    toast.add({ severity: 'success', summary: `Модуль создан: ${name}`, life: 3000 })
  } catch (e) {
    error.value = e.response?.data?.error || 'Не удалось создать модуль'
  } finally {
    busy.value = false
  }
}

async function saveStatus(e) {
  const status = draft[e.id]
  if (!status || status === e.status) return
  try {
    await store.renameEpic(e.id, e.name, e.description, status)
    await store.fetchEpics(props.project.id)
    syncDraft()
    toast.add({ severity: 'success', summary: `Статус «${e.name}» обновлён`, life: 2500 })
  } catch (err) {
    draft[e.id] = e.status // откат
    toast.add({ severity: 'error', summary: err.response?.data?.error || 'Ошибка смены статуса', life: 4000 })
  }
}

async function remove(e) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    await store.deleteEpic(e.id)
    syncDraft()
    delete bars[e.id]
    toast.add({ severity: 'info', summary: `Модуль удалён: ${e.name}`, life: 3000 })
  } catch (err) {
    error.value = err.response?.data?.error || 'Не удалось удалить модуль'
  } finally {
    busy.value = false
  }
}

watch(
  () => props.project.id,
  () => loadAll()
)

onMounted(loadAll)
</script>

<style scoped>
.epic-panel {
  width: 100%;
}

.epics-list {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
  margin-bottom: 1rem;
}

.epic-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.epic-number {
  font-weight: 700;
  color: var(--mb-text-muted);
  min-width: 2.6rem;
}

.epic-name {
  font-weight: 600;
  min-width: 160px;
}

.epic-progress {
  min-width: 140px;
}

.muted {
  color: var(--mb-text-muted);
}

.empty {
  padding: 0.5rem 0 1rem;
}

.create-row {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  flex-wrap: wrap;
}

.err {
  margin-top: 0.75rem;
}
</style>
