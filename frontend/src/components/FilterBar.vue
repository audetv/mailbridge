<template>
    <div class="filter-bar">
        <InputText v-model="search" placeholder="Поиск..." @input="onSearch" class="search-input" />
        <Select v-model="project" :options="projectOptions" optionLabel="label" optionValue="value" placeholder="Проект"
            @change="onChange('project', $event.value)" showClear />
        <Select v-model="status" :options="statusOptions" optionLabel="label" optionValue="value" placeholder="Статус"
            @change="onChange('status', $event.value)" showClear />
    </div>
</template>

<script setup>
import { ref } from 'vue'
import { useTasksStore } from '@/stores/tasks'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'

const store = useTasksStore()

const search = ref('')
const project = ref(null)
const status = ref(null)

const projectOptions = [
    { label: 'Входящие', value: 'Входящие' },
    { label: 'ТРК', value: 'ТРК' },
    { label: 'Отель', value: 'Отель' },
    { label: 'Фитнес-клуб', value: 'Фитнес-клуб' },
    { label: 'Театр', value: 'Театр' },
    { label: 'Мебельный центр', value: 'Мебельный центр' },
    { label: 'Кафе', value: 'Кафе' },
    { label: 'Ледовая арена', value: 'Ледовая арена' },
    { label: 'Корпоративные сайты', value: 'Корпоративные сайты' }
]

const statusOptions = [
    { label: 'Новая', value: 'new' },
    { label: 'В работе', value: 'in_progress' },
    { label: 'Решена', value: 'resolved' },
    { label: 'Закрыта', value: 'closed' }
]

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