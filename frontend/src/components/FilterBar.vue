<template>
  <div class="filter-bar">
    <InputText v-model="search" placeholder="Поиск..." @input="onSearch" class="search-input" />
    <Select
      v-model="project"
      :options="projectOptions"
      optionLabel="label"
      optionValue="value"
      placeholder="Проект"
      @change="onProjectChange"
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
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useTasksStore } from '@/stores/tasks'
import { useEpicsStore } from '@/stores/epics'
import { useProjectsStore } from '@/stores/projects'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'

const store = useTasksStore()
const epicsStore = useEpicsStore()
const projectsStore = useProjectsStore()

const search = ref('')
const project = ref(null)
const epic = ref(null)

const epicOptions = ref([])

const projectOptions = [
  { label: 'Входящие', value: 'Входящие' },
  { label: 'ТРК', value: 'ТРК' },
  { label: 'Отель', value: 'Отель' },
  { label: 'Лидер Спорт', value: 'Лидер Спорт' },
  { label: 'Театр', value: 'Театр' },
  { label: 'Мебельный центр', value: 'Мебельный центр' },
  { label: 'Кафе', value: 'Кафе' },
  { label: 'Ледовая арена', value: 'Ледовая арена' },
  { label: 'Корпоративные сайты', value: 'Корпоративные сайты' }
]

onMounted(async () => {
  // Восстанавливаем фильтры из store при возврате со страницы задачи
  search.value = store.filters.search || ''
  project.value = store.filters.project || null
  if (project.value) {
    await loadEpicOptions(projectsIdByName(project.value))
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
</style>
