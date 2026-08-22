<template>
    <div class="inbox-item" v-if="item">
        <header class="item-header">
            <Button icon="pi pi-arrow-left" severity="secondary" text @click="goBack" />
            <h2>{{ item.subject }}</h2>
        </header>

        <div class="item-grid">
            <div class="item-main">
                <!-- Вердикты AI -->
                <Card v-if="parsedVerdicts.length > 0" class="verdicts-card">
                    <template #title>Вердикты AI</template>
                    <template #content>
                        <div v-for="(v, i) in parsedVerdicts" :key="i" class="verdict-item">
                            <Tag :value="verdictLabel(v.action)" :severity="verdictSeverity(v.action)" />
                            <div v-if="v.task" class="verdict-task">
                                <strong>{{ v.task.title }}</strong>
                                <div class="verdict-desc">{{ v.task.description }}</div>
                                <div class="verdict-meta">
                                    <Tag :value="`Проект: ${v.task.project}`" severity="info" />
                                    <Tag :value="`Тип: ${v.task.type}`" severity="secondary" />
                                    <Tag :value="`Приоритет: ${v.task.priority}`" severity="warn" />
                                </div>
                            </div>
                            <div v-if="v.summary" class="verdict-summary">{{ v.summary }}</div>
                        </div>
                    </template>
                </Card>

                <!-- Связанные задачи -->
                <Card v-if="linkedTasks.length > 0" class="linked-tasks">
                    <template #title>Связанные задачи</template>
                    <template #content>
                        <div v-for="link in linkedTasks" :key="link.task_id" class="linked-task">
                            <router-link :to="`/tasks/${link.task_id}?tab=inbox`">
                                Задача #{{ link.task_id }}
                            </router-link>
                        </div>
                    </template>
                </Card>

                <!-- Оригинальное письмо -->
                <Card>
                    <template #title>Письмо</template>
                    <template #content>
                        <div class="item-meta">
                            <span><strong>От:</strong> {{ item.from_name }} ({{ item.from_contact }})</span>
                            <span><strong>Дата:</strong> {{ formatDate(item.received_at) }}</span>
                        </div>
                        <div class="item-body" v-html="item.body_html || escapeHtml(item.body_text)"></div>
                    </template>
                </Card>
            </div>

            <div class="item-sidebar">
                <Card>
                    <template #content>
                        <div class="actions">
                            <Button label="Отметить непрочитанным" icon="pi pi-envelope" severity="secondary"
                                @click="markUnread" class="w-full mb-2" />
                            <Button label="В архив" icon="pi pi-inbox" severity="secondary"
                                @click="archive" class="w-full mb-2" />
                            <Button label="Создать задачу" icon="pi pi-plus" @click="createTask" class="w-full" />
                        </div>
                    </template>
                </Card>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import apiClient from '@/api/client'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Tag from 'primevue/tag'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const item = ref(null)
const linkedTasks = ref([])

const parsedVerdicts = computed(() => {
    if (!item.value?.ai_verdict || item.value.ai_verdict === '[]') return []
    try {
        return JSON.parse(item.value.ai_verdict)
    } catch {
        return []
    }
})

onMounted(async () => {
    const { data } = await apiClient.get(`/inbox/${route.params.id}`)
    item.value = data
    await apiClient.post(`/inbox/${route.params.id}/read`)

    try {
        const tasksData = await apiClient.get(`/inbox/${route.params.id}/tasks`)
        linkedTasks.value = tasksData.data
    } catch { /* ignore */ }
})

function verdictLabel(action) {
    const labels = { new: 'Новая задача', update: 'Обновление', completed: 'Выполнено', none: 'Информация' }
    return labels[action] || action
}

function verdictSeverity(action) {
    const map = { new: 'info', update: 'warn', completed: 'success', none: 'secondary' }
    return map[action] || 'info'
}

function escapeHtml(text) {
    return text?.replace(/\n/g, '<br>') || ''
}

function formatDate(dateStr) {
    if (!dateStr) return ''
    const d = new Date(dateStr)
    return d.toLocaleDateString('ru-RU') + ' ' + d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

async function markUnread() {
    await apiClient.post(`/inbox/${route.params.id}/unread`)
    toast.add({ severity: 'info', summary: 'Отмечено непрочитанным', life: 2000 })
}

async function archive() {
    await apiClient.post(`/inbox/${route.params.id}/archive`)
    toast.add({ severity: 'success', summary: 'В архив', life: 2000 })
    router.push('/?tab=inbox')
}

async function createTask() {
    try {
        await apiClient.post(`/inbox/${route.params.id}/task`)
        toast.add({ severity: 'success', summary: 'Задача создана', life: 2000 })
        router.push('/')
    } catch (e) {
        toast.add({ severity: 'error', summary: 'Ошибка', life: 3000 })
    }
}

function goBack() {
    router.push('/?tab=inbox')
}
</script>

<style scoped>
.inbox-item {
    min-height: 100vh;
    background: var(--p-surface-100);
}

.item-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 1rem 2rem;
    background: var(--p-surface-0);
    border-bottom: 1px solid var(--p-surface-200);
}

.item-header h2 {
    margin: 0;
}

.item-grid {
    display: grid;
    grid-template-columns: 1fr 300px;
    gap: 1.5rem;
    padding: 1.5rem 2rem;
}

.item-main {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
}

.item-meta {
    display: flex;
    gap: 2rem;
    margin-bottom: 1rem;
    color: var(--p-text-muted-color);
    font-size: 1rem;
}

.item-body {
    line-height: 1.5;
    max-height: 500px;
    overflow-y: auto;
    padding: 0.5rem;
    border: 1px solid var(--p-surface-200);
    border-radius: 0.5rem;
    background: var(--p-surface-0);
}

.item-body :deep(p) {
    margin: 0 0 0.5rem 0;
}

.item-body :deep(br) {
    display: block;
    content: "";
    margin-top: 0.25rem;
}

.verdicts-card {
    margin-bottom: 0;
}

.verdict-item {
    margin-bottom: 0.75rem;
}

.verdict-task {
    margin-top: 0.5rem;
}

.verdict-desc {
    margin-top: 0.25rem;
    font-size: 1rem;
}

.verdict-meta {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
    margin-top: 0.5rem;
}

.verdict-summary {
    margin-top: 0.5rem;
    font-size: 1rem;
    color: var(--p-text-muted-color);
}

.linked-task {
    padding: 0.25rem 0;
}

.linked-task a {
    text-decoration: none;
    color: var(--p-primary-color);
    font-weight: 600;
}

.actions {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
}

.w-full {
    width: 100%;
}
</style>