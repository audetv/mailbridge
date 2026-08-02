<template>
    <Dialog v-model:visible="visible" modal :header="task?.subject || 'Задача'" :style="{ width: '800px' }"
        @hide="close">
        <div v-if="taskLoading" class="flex justify-content-center p-4">
            <ProgressSpinner />
        </div>

        <div v-else-if="task" class="task-card">
            <!-- Мета-информация -->
            <div class="task-meta">
                <div class="meta-row">
                    <span class="meta-label">Проект:</span>
                    <Select v-model="task.project" :options="projectOptions" optionLabel="label" optionValue="value"
                        @change="onChange('project', $event.value)" class="meta-select" />
                </div>
                <div class="meta-row">
                    <span class="meta-label">Статус:</span>
                    <Select v-model="task.status" :options="statusOptions" optionLabel="label" optionValue="value"
                        @change="onChange('status', $event.value)" class="meta-select" />
                </div>
                <div class="meta-row">
                    <span class="meta-label">Приоритет:</span>
                    <Tag :value="task.priority" :severity="prioritySeverity(task.priority)" />
                </div>
                <div class="meta-row">
                    <span class="meta-label">Тип:</span>
                    <Tag :value="task.type || '—'" />
                </div>
                <div class="meta-row">
                    <span class="meta-label">От:</span>
                    <span>{{ task.from_name || task.from_email }}</span>
                </div>
                <div class="meta-row">
                    <span class="meta-label">Дата:</span>
                    <span>{{ formatDate(task.created_at) }}</span>
                </div>
            </div>

            <!-- Описание -->
            <div class="task-body">
                <h3>Описание</h3>
                <div class="body-text" v-html="task.body_html || escapeHtml(task.body_text)"></div>
            </div>

            <!-- Вложения -->
            <div v-if="attachments.length" class="task-attachments">
                <h3>Вложения</h3>
                <div v-for="att in attachments" :key="att.id" class="attachment-item">
                    <i class="pi pi-paperclip" />
                    <span>{{ att.filename }} ({{ formatSize(att.size) }})</span>
                </div>
            </div>

            <!-- Комментарии -->
            <div class="task-comments">
                <h3>Комментарии ({{ comments.length }})</h3>
                <CommentList :comments="comments" />
            </div>

            <!-- Форма ответа -->
            <div class="task-reply">
                <h3>Ответить</h3>
                <ReplyForm @reply="onReply" />
            </div>
        </div>
    </Dialog>
</template>

<script setup>
import { computed } from 'vue'
import { useTasksStore } from '@/stores/tasks'
import Dialog from 'primevue/dialog'
import Select from 'primevue/select'
import Tag from 'primevue/tag'
import ProgressSpinner from 'primevue/progressspinner'
import CommentList from './CommentList.vue'
import ReplyForm from './ReplyForm.vue'

const props = defineProps({
    modelValue: { type: Boolean, default: false },
    taskId: { type: Number, default: 0 }
})

const emit = defineEmits(['update:modelValue'])

const store = useTasksStore()

const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
})

const task = computed(() => store.currentTask)
const comments = computed(() => store.comments)
const attachments = computed(() => store.attachments)
const taskLoading = computed(() => store.taskLoading)

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

import { watch } from 'vue'
watch(() => props.taskId, (id) => {
    if (id > 0) store.fetchTask(id)
}, { immediate: true })

function onChange(field, value) {
    store.updateTask(task.value.id, { [field]: value })
}

function onReply(body) {
    store.replyTask(task.value.id, body)
}

function close() {
    store.clearCurrentTask()
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

function prioritySeverity(p) {
    return { urgent: 'danger', high: 'warn', medium: 'info', low: 'success' }[p] || 'info'
}

function escapeHtml(text) {
    return text?.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\n/g, '<br>') || ''
}
</script>

<style scoped>
.task-card {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
}

.task-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    padding: 1rem;
    background: var(--p-surface-50);
    border-radius: 0.5rem;
}

.meta-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.meta-label {
    font-weight: 600;
    color: var(--p-text-muted-color);
    min-width: 80px;
}

.meta-select {
    width: 180px;
}

.task-body h3,
.task-attachments h3,
.task-comments h3,
.task-reply h3 {
    margin: 0 0 0.5rem 0;
}

.body-text {
    white-space: pre-wrap;
}

.attachment-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.25rem 0;
}
</style>