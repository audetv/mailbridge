<template>
    <div class="task-detail" v-if="store.currentTask">
        <header class="task-header">
            <Button icon="pi pi-arrow-left" severity="secondary" text @click="goBack" />
            <h2>#{{ store.currentTask.id }} {{ store.currentTask.subject }}</h2>
        </header>

        <div class="task-grid">
            <div class="task-main">
                <Card>
                    <template #title>Описание</template>
                    <template #content>
                        <div class="task-meta">
                            <span><strong>От:</strong> {{ store.currentTask.from_email }}</span>
                            <span><strong>Дата:</strong> {{ formatDate(store.currentTask.created_at) }}</span>
                        </div>
                        <div class="task-body"
                            v-html="store.currentTask.body_html || escapeHtml(store.currentTask.body_text)"></div>

                        <div v-if="store.currentAttachments.length > 0" class="attachments">
                            <h4>Вложения</h4>
                            <div v-for="att in store.currentAttachments" :key="att.id" class="attachment-item">
                                <i class="pi pi-paperclip" />
                                <a :href="`/api/attachments/${att.storage_path}`" target="_blank">{{ att.filename }}</a>
                                <span class="size">{{ formatSize(att.size) }}</span>
                            </div>
                        </div>
                    </template>
                </Card>

                <Card>
                    <template #title>Комментарии</template>
                    <template #content>
                        <CommentList :comments="store.currentComments" />
                        <div class="reply-section">
                            <ReplyForm :taskId="store.currentTask.id" @sent="onReplySent" />
                        </div>
                    </template>
                </Card>
            </div>

            <div class="task-sidebar">
                <Card>
                    <template #content>
                        <div class="field">
                            <label>Проект</label>
                            <Select v-model="project" :options="projectOptions" optionLabel="label" optionValue="value"
                                @change="updateField('project', $event.value)" />
                        </div>
                        <div class="field">
                            <label>Статус</label>
                            <Select v-model="status" :options="statusOptions" optionLabel="label" optionValue="value"
                                @change="updateField('status', $event.value)" />
                        </div>
                        <div class="field">
                            <label>Приоритет</label>
                            <Select v-model="priority" :options="priorityOptions" optionLabel="label"
                                optionValue="value" @change="updateField('priority', $event.value)" />
                        </div>
                        <div class="field">
                            <label>Тип</label>
                            <Select v-model="type" :options="typeOptions" optionLabel="label" optionValue="value"
                                @change="updateField('type', $event.value)" />
                        </div>
                        <div class="field">
                            <label>Исполнитель</label>
                            <InputText v-model="assignee" @blur="updateField('assignee', assignee)" />
                        </div>
                    </template>
                </Card>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTasksStore } from '@/stores/tasks'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import CommentList from '@/components/CommentList.vue'
import ReplyForm from '@/components/ReplyForm.vue'

const route = useRoute()
const router = useRouter()
const store = useTasksStore()

const project = ref(null)
const status = ref(null)
const priority = ref(null)
const type = ref(null)
const assignee = ref('')

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

const statusOptions = [
    { label: 'Новая', value: 'new' },
    { label: 'В работе', value: 'in_progress' },
    { label: 'Решена', value: 'resolved' },
    { label: 'Закрыта', value: 'closed' }
]

const priorityOptions = [
    { label: 'Urgent', value: 'urgent' },
    { label: 'High', value: 'high' },
    { label: 'Medium', value: 'medium' },
    { label: 'Low', value: 'low' }
]

const typeOptions = [
    { label: 'Bug', value: 'bug' },
    { label: 'Feature', value: 'feature' },
    { label: 'Support', value: 'support' },
    { label: 'Access', value: 'access' },
    { label: 'SEO', value: 'seo' },
    { label: 'Content', value: 'content' }
]

onMounted(async () => {
    await store.fetchTask(route.params.id)
    syncFields()
    store.markAsRead(route.params.id)
})

watch(() => store.currentTask, syncFields)

function syncFields() {
    if (!store.currentTask) return
    project.value = store.currentTask.project
    status.value = store.currentTask.status
    priority.value = store.currentTask.priority
    type.value = store.currentTask.type
    assignee.value = store.currentTask.assignee
}

async function updateField(field, value) {
    await store.updateTask(route.params.id, { [field]: value })
}

function onReplySent() {
    // Комментарий уже добавлен в store.currentComments через replyTask
}

function goBack() {
  const tab = route.query.tab
  router.push({ path: '/', query: tab ? { tab } : {} })
}

function formatDate(dateStr) {
    if (!dateStr) return ''
    const d = new Date(dateStr)
    return d.toLocaleDateString('ru-RU') + ' ' + d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

function formatSize(bytes) {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function escapeHtml(text) {
    return text?.replace(/\n/g, '<br>') || ''
}
</script>

<style scoped>
.task-detail {
    min-height: 100vh;
    background: var(--p-surface-100);
}

.task-header {
    display: flex;
    align-items: center;
    gap: 1rem;
    padding: 1rem 2rem;
    background: var(--p-surface-0);
    border-bottom: 1px solid var(--p-surface-200);
}

.task-header h2 {
    margin: 0;
}

.task-grid {
    display: grid;
    grid-template-columns: 1fr 300px;
    gap: 1.5rem;
    padding: 1.5rem 2rem;
}

.task-main {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
}

.task-meta {
    display: flex;
    gap: 2rem;
    margin-bottom: 1rem;
    color: var(--p-text-muted-color);
    font-size: 1rem;
}

.task-body {
    line-height: 1.5;
    max-height: 500px;
    overflow-y: auto;
    padding: 0.5rem;
    border: 1px solid var(--p-surface-200);
    border-radius: 0.5rem;
    background: var(--p-surface-0);
}

.task-body :deep(p) {
    margin: 0 0 0.5rem 0;
}

.task-body :deep(br) {
    display: block;
    content: "";
    margin-top: 0.25rem;
}

.attachments {
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--p-surface-200);
}

.attachment-item {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    padding: 0.25rem 0;
}

.size {
    color: var(--p-text-muted-color);
    font-size: 0.85rem;
}

.reply-section {
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--p-surface-200);
}

.field {
    margin-bottom: 1rem;
}

.field label {
    display: block;
    margin-bottom: 0.25rem;
    font-weight: 600;
    font-size: 1rem;
}
</style>