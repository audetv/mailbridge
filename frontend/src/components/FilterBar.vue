<template>
  <div class="filter-bar">
    <InputText v-model="search" placeholder="Поиск..." @input="onSearch" class="search-input" />
    <Select
      v-model="project"
      :options="projectOptions"
      optionLabel="label"
      optionValue="value"
      placeholder="Проект"
      @change="onChange('project', $event.value)"
      showClear
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useTasksStore } from '@/stores/tasks'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'

const store = useTasksStore()

const search = ref('')
const project = ref(null)

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

onMounted(() => {
  // Восстанавливаем фильтры из store при возврате со страницы задачи
  search.value = store.filters.search || ''
  project.value = store.filters.project || null
})

let searchTimeout
function onSearch() {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    store.setFilter('search', search.value)
  }, 300)
}

function onChange(key, value) {
  store.setFilter(key, value || '')
}
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
</style>
