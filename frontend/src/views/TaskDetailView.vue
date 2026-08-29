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
            <div
              class="task-body"
              @click="handleImageClick"
              v-html="store.currentTask.body_html || escapeHtml(store.currentTask.body_text)"
            ></div>

            <div v-if="taskAttachments.length > 0" class="attachments">
              <h4>Вложения</h4>
              <div v-for="att in taskAttachments" :key="att.id" class="attachment-item">
                <i class="pi pi-paperclip" />
                <a
                  :href="`/api/attachments/${att.storage_path}/${encodeURIComponent(att.filename)}`"
                  target="_blank"
                  >{{ att.filename }}</a
                >
                <span class="size">{{ formatSize(att.size) }}</span>
                <Button
                  icon="pi pi-times"
                  text
                  size="small"
                  severity="danger"
                  @click="unlinkAttachment(att.id)"
                  title="Открепить"
                />
              </div>
            </div>
          </template>
          <template #footer>
            <WorkflowButtons :currentStatus="status" @transition="onWorkflowTransition" />
          </template>
        </Card>

        <!-- Оригинальное письмо — аккордеон -->
        <Card v-if="inboxItems.length > 0" class="inbox-context">
          <template #title>
            <div class="inbox-context-header">
              <span>📧 Оригинальное письмо</span>
              <i
                :class="expanded ? 'pi pi-chevron-up' : 'pi pi-chevron-down'"
                @click="expanded = !expanded"
              />
            </div>
          </template>
          <template #content>
            <div v-if="expanded">
              <div v-for="item in inboxItems" :key="item.id" class="inbox-context-item">
                <div class="inbox-context-meta">
                  <span class="inbox-context-subject">{{ item.subject }}</span>
                  <router-link :to="`/inbox/${item.id}`" class="inbox-link">Открыть в ленте</router-link>
                </div>
                <div
                  class="item-body"
                  @click="handleImageClick"
                  v-html="item.body_html || escapeHtml(item.body_text)"
                ></div>
              </div>
            </div>
            <div v-else>
              <div v-for="item in inboxItems" :key="item.id" class="inbox-context-item">
                <div class="inbox-context-meta">
                  <span class="inbox-context-subject">{{ item.subject }}</span>
                  <router-link :to="`/inbox/${item.id}`" class="inbox-link">Открыть в ленте</router-link>
                </div>
                <div class="item-body-preview" v-html="previewHtml(item)"></div>
                <Button
                  v-if="needExpand(item)"
                  label="Показать полностью"
                  text
                  size="small"
                  @click="expanded = true"
                  class="expand-btn"
                />
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
              <Select
                v-model="project"
                :options="projectOptions"
                optionLabel="label"
                optionValue="value"
                @change="updateField('project', $event.value)"
              />
            </div>
            <div class="field">
              <label>Статус</label>
              <Select
                v-model="status"
                :options="statusOptions"
                optionLabel="label"
                optionValue="value"
                @change="updateField('status', $event.value)"
              />
            </div>
            <div class="field">
              <label>Приоритет</label>
              <Select
                v-model="priority"
                :options="priorityOptions"
                optionLabel="label"
                optionValue="value"
                @change="updateField('priority', $event.value)"
              />
            </div>
            <div class="field">
              <label>Тип</label>
              <Select
                v-model="type"
                :options="typeOptions"
                optionLabel="label"
                optionValue="value"
                @change="updateField('type', $event.value)"
              />
            </div>
            <div class="field" v-if="epicOptions.length > 0">
              <label>Модуль</label>
              <Select
                v-model="epic"
                :options="epicOptions"
                optionLabel="label"
                optionValue="value"
                @change="onEpicChange($event.value)"
              />
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
import { ref, watch, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useTasksStore } from '@/stores/tasks'
import { useProjectsStore } from '@/stores/projects'
import { useEpicsStore } from '@/stores/epics'
import apiClient from '@/api/client'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import CommentList from '@/components/CommentList.vue'
import ReplyForm from '@/components/ReplyForm.vue'
import WorkflowButtons from '@/components/WorkflowButtons.vue'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const store = useTasksStore()
const projectsStore = useProjectsStore()
const epicsStore = useEpicsStore()

const project = ref(null)
const status = ref(null)
const priority = ref(null)
const type = ref(null)
const assignee = ref('')
const epic = ref(null)
const inboxItems = ref([])
const taskAttachments = ref([])
const expanded = ref(false)

const epicOptions = computed(() =>
  epicsStore.epics.map((e) => ({ label: e.name, value: e.id }))
)

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
  { label: 'Бэклог', value: 'backlog' },
  { label: 'В работе', value: 'in_progress' },
  { label: 'Выполнена', value: 'completed' },
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
  await loadEpics()
  store.markAsRead(route.params.id)
  inboxItems.value = await store.fetchTaskInbox(route.params.id)
  taskAttachments.value = await fetchTaskAttachments(route.params.id)
})

watch(() => store.currentTask, () => {
  syncFields()
  loadEpics()
})

// Эпики проекта задачи (схема: задача → проект по ИМЕНИ, API эпиков по ID проекта)
async function loadEpics() {
  const task = store.currentTask
  if (!task || !task.project) {
    epicsStore.epics = []
    return
  }
  let project = projectsStore.projectByName(task.project)
  if (!project) {
    await projectsStore.fetchProjects({ archived: 'false' })
    project = projectsStore.projectByName(task.project)
    if (!project) {
      epicsStore.epics = []
      return
    }
  }
  await epicsStore.fetchEpics(project.id)
  epic.value = task.epic_id || null
}

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

// Смена модуля. null — сброс (задача без модуля).
async function onEpicChange(epicId) {
  epic.value = epicId || null
  await store.updateTask(route.params.id, { epic_id: epicId || null })
}

async function onWorkflowTransition(newStatus) {
  status.value = newStatus
  await updateField('status', newStatus)
}

async function fetchTaskAttachments(taskId) {
  try {
    const { data } = await apiClient.get(`/tasks/${taskId}/attachments`)
    return data
  } catch {
    return []
  }
}

async function unlinkAttachment(attId) {
  try {
    await apiClient.delete(`/tasks/${route.params.id}/attachments/${attId}`)
    taskAttachments.value = taskAttachments.value.filter((a) => a.id !== attId)
    toast.add({ severity: 'success', summary: 'Вложение откреплено', life: 2000 })
  } catch {
    toast.add({ severity: 'error', summary: 'Ошибка', life: 3000 })
  }
}

function previewHtml(item) {
  const html = item.body_html || escapeHtml(item.body_text)
  if (html.length > 5000) {
    return html.slice(0, 5000) + '...'
  }
  return html
}

function needExpand(item) {
  const html = item.body_html || escapeHtml(item.body_text)
  return html.length > 2000
}

function handleImageClick(event) {
  if (event.target.tagName === 'IMG' && event.target.src) {
    window.open(event.target.src, '_blank')
  }
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
  return (
    d.toLocaleDateString('ru-RU') +
    ' ' +
    d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  )
}

function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function escapeHtml(text) {
  return text?.replace(/\n/g, '<br>') || ''
}

defineExpose({ epic, epicOptions, onEpicChange, loadEpics })
</script>

<style scoped>
.task-detail {
  overflow-x: hidden;
  min-height: 100vh;
  /* background: var(--mb-surface-hover); */
}

.task-body,
.item-body,
.item-body-preview {
  overflow-wrap: break-word;
  word-wrap: break-word;
  word-break: break-word;
  max-width: 100%;
}

/*
.item-body :deep(pre),
.item-body :deep(code),
.task-body :deep(pre),
.task-body :deep(code) {
  white-space: pre-wrap;
  word-break: break-all;
  max-width: 100%;
  overflow-x: auto;
}

.item-body :deep(table),
.task-body :deep(table) {
  max-width: 100%;
  display: block;
  overflow-x: auto;
}
*/
.item-body :deep(img),
.task-body :deep(img),
.item-body-preview :deep(img) {
  max-width: 100%;
  /* max-height: 400px; */
  height: auto;
  object-fit: contain;
  cursor: pointer;
}

.task-detail {
  min-height: 100vh;
  /* background: var(--mb-surface-hover); */
}

.task-header {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 2rem;
  background: var(--mb-surface);
  border-bottom: 1px solid var(--mb-border);
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
  color: var(--mb-text-muted);
  font-size: 1rem;
}

.task-body {
  line-height: 1.5;
  max-height: 500px;
  overflow-y: auto;
  padding: 0.5rem;
  border: 1px solid var(--mb-border);
  border-radius: 0.5rem;
  background: var(--mb-surface);
}

.task-body :deep(p) {
  margin: 0 0 0.5rem 0;
}

.task-body :deep(br) {
  display: block;
  content: '';
  margin-top: 0.25rem;
}

.attachments {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--mb-border);
}

.attachment-item {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  padding: 0.25rem 0;
}

.size {
  color: var(--mb-text-muted);
  font-size: 0.85rem;
}

.reply-section {
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid var(--mb-border);
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

.inbox-context {
  border-left: 3px solid var(--mb-primary);
}

.inbox-context-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.inbox-context-subject {
  font-weight: 600;
}

.inbox-context-meta {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.inbox-context-item {
  margin-bottom: 1rem;
}

.inbox-context-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  user-select: none;
}

.inbox-collapsed {
  color: var(--mb-text-muted);
  font-size: 0.85rem;
}

.item-body-preview {
  max-height: 400px;
  overflow: hidden;
  position: relative;
  line-height: 1.5;
}

.item-body-preview::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 50px;
  background: linear-gradient(transparent, var(--mb-surface));
}

.expand-btn {
  margin-top: 0.5rem;
}

.workflow-buttons {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  justify-content: flex-end;
}
</style>
