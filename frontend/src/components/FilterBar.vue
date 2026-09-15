<template>
  <div class="filter-bar">
    <InputText v-model="search" placeholder="Поиск..." @input="onSearch" class="search-input" />
    <Select
      v-model="project"
      :options="projectOptions"
      optionLabel="label"
      optionValue="value"
      placeholder="Проект"
      @change="onProjectChange($event.value)"
      showClear
    />
    <Select
      v-model="epic"
      :options="epicOptions"
      optionLabel="label"
      optionValue="value"
      placeholder="Модуль"
      :disabled="!project"
      @change="onChange('epic_id', $event.value)"
      showClear
      class="epic-select"
    />
    <!-- Персоны (v0.24, шаг 6): фильтры «Заказчик»/«Исполнитель» по персонам
         (UUID); legacy-поле assignee (email) — отдельно, без фильтра здесь. -->
    <Select
      v-model="requestor"
      :options="personOptions"
      optionLabel="label"
      optionValue="value"
      placeholder="Заказчик (персона)"
      @change="onChange('requestor_id', $event.value)"
      showClear
      class="person-select"
    />
    <Select
      v-model="assigneePerson"
      :options="personOptions"
      optionLabel="label"
      optionValue="value"
      placeholder="Исполнитель (персона)"
      @change="onChange('assignee_id', $event.value)"
      showClear
      class="person-select"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useTasksStore } from '@/stores/tasks'
import { useEpicsStore } from '@/stores/epics'
import { useProjectsStore } from '@/stores/projects'
import { usePersonsStore } from '@/stores/persons'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'

const store = useTasksStore()
const epicsStore = useEpicsStore()
const projectsStore = useProjectsStore()
const personsStore = usePersonsStore()

const search = ref('')
const project = ref(null)
const epic = ref(null)
const requestor = ref(null)
const assigneePerson = ref(null)

const epicOptions = ref([])

// Персона-опции (v0.24, шаг 6): имя → org → email (для «в процессе узнавания»).
const personOptions = computed(() =>
  personsStore.list.map((p) => ({
    label: p.name || p.org || p.primary_email || 'персона',
    value: p.id
  }))
)

// Опции «Проект» — из store (активные, archived=false), не хардкод:
// проекты создаются/переименовываются/архивируются в рантайме,
// статический список живёт вне БД (в т.ч. «Входящие» приходит из БД).
const projectOptions = computed(() =>
  projectsStore.projects.map((p) => ({ label: p.name, value: p.name }))
)

onMounted(async () => {
  if (projectsStore.projects.length === 0) {
    await projectsStore.fetchProjects({ archived: 'false' })
  }
  // Персоны (v0.24, шаг 6) — для фильтров Заказчик/Исполнитель.
  if (personsStore.list.length === 0) {
    await personsStore.fetchPersons().catch(() => {})
  }
  // Восстанавливаем фильтры из store при возврате со страницы задачи
  search.value = store.filters.search || ''
  project.value = store.filters.project || null
  requestor.value = store.filters.requestor_id || null
  assigneePerson.value = store.filters.assignee_id || null
  if (project.value) {
    const projectId = await projectsIdByName(project.value)
    await loadEpicOptions(projectId)
  }
})

// Проекты «Проект → Проект» (задачи) хранят текстовое имя; модули (эпики)
// привязаны к числовому id из /api/projects. Ищем id по имени из проектов-стора.
async function projectsIdByName(name) {
  if (projectsStore.projects.length === 0) {
    await projectsStore.fetchProjects()
  }
  const p = projectsStore.projects.find((x) => x.name === name)
  return p ? p.id : null
}

async function loadEpicOptions(projectId) {
  if (!projectId) {
    epicOptions.value = []
    return
  }
  await epicsStore.fetchEpics(projectId)
  epicOptions.value = epicsStore.epics.map((e) => ({
    label: `#${e.number} ${e.name}`,
    value: String(e.id)
  }))
}

let searchTimeout
function onSearch() {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    store.setFilter('search', search.value)
  }, 300)
}

async function onProjectChange(value) {
  // сбрасываем фильтр по модулю при смене проекта — модули другого проекта
  project.value = value ?? null
  epic.value = null
  store.filters.epic_id = ''
  store.setFilter('project', value || '')
  if (value) {
    const projectId = await projectsIdByName(value)
    await loadEpicOptions(projectId)
  } else {
    epicOptions.value = []
  }
}

function onChange(key, value) {
  store.setFilter(key, value || '')
}

defineExpose({ onProjectChange, onChange, epicOptions })
</script>

<style scoped>
.filter-bar {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
}

.search-input {
  flex: 1;
}

.epic-select {
  width: 220px;
}

.person-select {
  width: 200px;
}
</style>
