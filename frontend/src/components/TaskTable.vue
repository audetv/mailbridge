<template>
  <div data-testid="task-table">
    <DataTable
      :value="store.tasks"
      :loading="store.loading"
      paginator
      :rows="50"
      :totalRecords="store.total"
      @page="onPage"
      lazy
      dataKey="id"
      selectionMode="multiple"
      v-model:selection="selectedTasks"
      stripedRows
      :rowClass="rowClass"
      @row-click="onRowClick"
    >
      <template #header>
        <div class="bulk-toolbar">
          <input
            type="checkbox"
            data-testid="bulk-select-all"
            :checked="allPageSelected"
            :indeterminate.prop="somePageSelected && !allPageSelected"
            :disabled="store.tasks.length === 0"
            @change="onToggleAllPage"
            aria-label="select all on page"
          />
          <label class="bulk-toolbar-label" for="bulk-select-all">
            Выбрать все на странице
          </label>
          <span v-if="selectedTasks.length" class="bulk-count" data-testid="bulk-count">
            Выбрано: {{ selectedTasks.length }}
          </span>
        </div>
      </template>

      <Column selectionMode="multiple" style="width: 44px" />
      <Column field="id" header="ID" style="width: 80px" />
      <Column field="created_at" header="Дата" style="width: 150px">
        <template #body="{ data }">
          {{ formatDate(data.created_at) }}
        </template>
      </Column>
      <Column field="from_email" header="От кого" style="width: 200px" />
      <Column field="subject" header="Тема">
        <template #body="{ data }">
          <div class="subject-cell">
            <Badge
              v-if="data.unread_comments > 0"
              :value="data.unread_comments"
              severity="info"
              size="small"
              class="unread-badge"
            />
            <span>{{ data.subject }}</span>
          </div>
        </template>
      </Column>
      <Column field="type" header="Тип" style="width: 100px">
        <template #body="{ data }">
          <Tag :value="data.type" v-if="data.type" />
        </template>
      </Column>
      <Column field="priority" header="Приоритет" style="width: 100px">
        <template #body="{ data }">
          <Tag :value="data.priority" :severity="prioritySeverity(data.priority)" />
        </template>
      </Column>
      <Column field="project" header="Проект" style="width: 150px">
        <template #body="{ data }">
          <a
            v-if="data.project"
            href="#"
            class="project-link"
            @click.prevent="goToProject(data.project)"
          >{{ data.project }}</a>
          <span v-else>—</span>
        </template>
      </Column>
      <Column field="epic_id" header="Модуль" style="width: 130px">
        <template #body="{ data }">
          <Tag v-if="epicName(data)" :value="epicName(data)" severity="secondary" />
          <span v-else class="epic-none">—</span>
        </template>
      </Column>
      <Column field="status" header="Статус" style="width: 120px">
        <template #body="{ data }">
          <Tag :value="statusLabel(data.status)" :severity="statusSeverity(data.status)" />
        </template>
      </Column>
      <Column field="assignee" header="Исполнитель" style="width: 120px" />
    </DataTable>

    <!-- Плавающая панель bulk-действий — появляется после >=1 выбранного (шаг 4). -->
    <div
      v-if="selectedTasks.length"
      class="bulk-panel"
      role="toolbar"
      aria-label="bulk actions"
      data-testid="bulk-panel"
    >
      <span class="bulk-panel-count">{{ selectedTasks.length }} выбрана</span>

      <select
        v-model="selectedStatus"
        class="bulk-select"
        data-testid="bulk-status"
        aria-label="status to apply"
      >
        <option value="" disabled>К статусу…</option>
        <option v-for="s in ALL_STATUSES" :key="s" :value="s">
          К {{ statusLabel(s) }}
        </option>
      </select>

      <button
        class="bulk-apply"
        :disabled="busy || !selectedStatus"
        @click="onApplyStatus"
        data-testid="bulk-apply-status"
      >
        Применить
      </button>

      <span class="bulk-sep" aria-hidden="true">|</span>

      <select
        v-model="selectedProject"
        class="bulk-select"
        data-testid="bulk-project"
        aria-label="project to apply"
      >
        <option value="" disabled>К проекту…</option>
        <option v-for="p in projectNames" :key="p" :value="p">
          {{ p }}
        </option>
      </select>

      <button
        class="bulk-apply"
        :disabled="busy || !selectedProject"
        @click="onApplyProject"
        data-testid="bulk-apply-project"
      >
        Применить
      </button>

      <button class="bulk-clear" @click="onClearSelection" :disabled="busy">
        Снять выбор
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTasksStore } from '@/stores/tasks'
import { useEpicsStore } from '@/stores/epics'
import { useProjectsStore } from '@/stores/projects'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Badge from 'primevue/badge'
import { useToast } from 'primevue/usetoast'

const store = useTasksStore()
const epics = useEpicsStore()
const projectsStore = useProjectsStore()
const router = useRouter()
const route = useRoute()
const toast = useToast()

// — Bulk actions (шаг 4 v0.23) —
const selectedTasks = ref([])
const busy = ref(false)
const ALL_STATUSES = ['backlog', 'new', 'in_progress', 'completed', 'closed']
const projectNames = computed(() =>
  (projectsStore.projects || []).map((p) => p.name).filter(Boolean)
)

// Ленивая подгрузка списка проектов — только при первом появлении панели.
let projectsLoaded = false
watch(
  () => selectedTasks.value.length,
  (n) => {
    if (n && !projectsLoaded) {
      projectsLoaded = true
      projectsStore.fetchProjects({ archived: 'false' }).catch(() => {})
    }
  },
  { immediate: true }
)

const allPageSelected = computed(
  () =>
    store.tasks.length > 0 &&
    store.tasks.every((t) => selectedTasks.value.some((s) => s.id === t.id))
)
const somePageSelected = computed(
  () => store.tasks.some((t) => selectedTasks.value.some((s) => s.id === t.id))
)

function onToggleAllPage() {
  if (allPageSelected.value) {
    const pageIds = new Set(store.tasks.map((t) => t.id))
    selectedTasks.value = selectedTasks.value.filter((t) => !pageIds.has(t.id))
  } else {
    const have = new Set(selectedTasks.value.map((t) => t.id))
    for (const t of store.tasks) {
      if (!have.has(t.id)) selectedTasks.value.push(t)
    }
  }
}

function onClearSelection() {
  selectedTasks.value = []
}

async function applyBulk(changes) {
  if (selectedTasks.value.length === 0 || busy.value) return
  busy.value = true
  try {
    const ids = selectedTasks.value.map((t) => t.id)
    const res = await store.bulkUpdate(ids, changes)
    toast.add({
      severity: 'success',
      summary: `Готово: ${res.count} ${res.count === 1 ? 'задача' : 'задач'}`,
      life: 3000,
    })
    selectedTasks.value = []
    selectedStatus.value = ''
    selectedProject.value = ''
    // остаёмся на листе — не перескакиваем на детальку (правило UX шага 4)
  } catch (e) {
    toast.add({
      severity: 'error',
      summary: 'Ошибка bulk',
      detail: e.response?.data?.error || e.message,
      life: 5000,
    })
  } finally {
    busy.value = false
  }
}

const selectedStatus = ref('')
const selectedProject = ref('')

function onApplyStatus() {
  if (!selectedStatus.value) return
  applyBulk({ status: selectedStatus.value })
}
function onApplyProject() {
  if (!selectedProject.value) return
  applyBulk({ project: selectedProject.value })
}

// Имя модуля по task.epic_id из epics-store (если загружен), иначе null.
function epicName(task) {
  if (!task.epic_id) return null
  const e = epics.epicById(task.epic_id)
  return e ? e.name : null
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

function statusLabel(status) {
  const labels = {
    new: 'Новая',
    backlog: 'Backlog',
    in_progress: 'В работе',
    completed: 'Готово',
    closed: 'Закрыта',
  }
  return labels[status] || status
}

function statusSeverity(status) {
  const map = {
    new: 'info',
    backlog: 'secondary',
    in_progress: 'warn',
    completed: 'success',
    closed: 'neutral',
  }
  return map[status] || 'secondary'
}

function prioritySeverity(priority) {
  const map = { urgent: 'danger', high: 'warn', medium: 'info', low: 'success' }
  return map[priority] || 'info'
}

function rowClass(task) {
  return task.unread_comments > 0 ? 'task-unread' : ''
}

function onRowClick(event) {
  router.push({ path: `/tasks/${event.data.id}`, query: { tab: route.query.tab } })
}

// Ссылка «Проект» → вкладка «Активные» + фильтр по проекту (таб — по URL,
// watch в DashboardView; epic от другого проекта сбрасываем).
function goToProject(projectName) {
  store.filters.epic_id = ''
  store.setFilter('project', projectName)
  router.replace({ query: { ...route.query, tab: 'active', project: projectName } })
}

function onPage(event) {
  store.filters.page = event.page + 1
  store.fetchTasks()
}

defineExpose({ epicName })
</script>

<style scoped>
.subject-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.unread-badge {
  flex-shrink: 0;
}

.epic-none {
  opacity: 0.4;
}

.project-link {
  color: var(--mb-primary, #2563eb);
  text-decoration: none;
}

.project-link:hover {
  text-decoration: underline;
}

:deep(.task-unread) {
  font-weight: 600;
  background: var(--mb-primary-soft) !important;
}

:deep(.task-unread td:first-child) {
  border-left: 3px solid var(--mb-primary);
}

/* — Bulk actions (шаг 4) — */
.bulk-toolbar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
}
.bulk-toolbar-label {
  font-size: 0.9rem;
  color: var(--mb-muted);
  cursor: pointer;
}
.bulk-count {
  margin-left: auto;
  padding: 0.15rem 0.6rem;
  background: var(--mb-primary);
  color: white;
  border-radius: 999px;
  font-size: 0.85rem;
}

.bulk-panel {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
  padding: 0.6rem 0.9rem;
  margin-top: 0.5rem;
  background: var(--mb-card);
  border: 1px solid var(--mb-border, rgba(0, 0, 0, 0.1));
  border-radius: 0.6rem;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
}
.bulk-panel-count {
  font-weight: 600;
  color: var(--mb-primary);
}
.bulk-sep {
  opacity: 0.3;
}
.bulk-select {
  padding: 0.35rem 0.5rem;
  border: 1px solid var(--mb-border, rgba(0, 0, 0, 0.15));
  border-radius: 0.35rem;
  background: white;
  color: var(--mb-text);
  min-width: 130px;
}
.bulk-apply {
  padding: 0.35rem 0.85rem;
  border: 0;
  border-radius: 0.35rem;
  background: var(--mb-primary);
  color: white;
  cursor: pointer;
}
.bulk-apply:disabled {
  background: rgba(0, 0, 0, 0.15);
  cursor: not-allowed;
}
.bulk-clear {
  padding: 0.35rem 0.85rem;
  border: 1px solid var(--mb-border, rgba(0, 0, 0, 0.15));
  border-radius: 0.35rem;
  background: transparent;
  color: var(--mb-muted);
  cursor: pointer;
  margin-left: auto;
}
</style>
