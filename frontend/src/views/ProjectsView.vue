<template>
  <div class="projects">
    <!-- Создание -->
    <Card class="create-card">
      <template #title>Новый проект</template>
      <template #subtitle>Внутренний проект — ссылка для задач и модулей</template>
      <template #content>
        <div class="create-form">
          <div class="create-fields">
            <label class="field">
              <span>Название</span>
              <InputText
                v-model="newName"
                maxlength="128"
                placeholder="Например: ТРК"
                @keyup.enter="create"
              />
            </label>
            <label class="field grow">
              <span>Описание (необязательно)</span>
              <Textarea v-model="newDesc" rows="1" :autoResize="true" @keyup.enter="create" />
            </label>
          </div>
          <Button label="Создать" icon="pi pi-plus" :loading="store.loading" :disabled="!newName.trim()" @click="create" />
          <Message
            v-if="createError"
            severity="error"
            :closable="true"
            @close="createError = ''"
            class="create-error"
          >
            {{ createError }}
          </Message>
        </div>
      </template>
    </Card>

    <!-- Список -->
    <DataTable :value="store.projects" :loading="store.loading" stripedRows>
      <Column field="name" header="Проект" style="min-width: 200px">
        <template #body="{ data }">
          <div class="name-cell">
            <span class="name">{{ data.name }}</span>
            <Tag v-if="data.archived" value="архив" severity="secondary" />
          </div>
        </template>
      </Column>
      <Column field="description" header="Описание" style="min-width: 240px">
        <template #body="{ data }">
          <span v-if="data.description">{{ data.description }}</span>
          <span v-else class="muted">—</span>
        </template>
      </Column>
      <Column field="created_at" header="Создан" style="width: 150px">
        <template #body="{ data }">{{ formatDate(data.created_at) }}</template>
      </Column>
      <Column header="Модули" style="width: 130px">
        <template #body="{ data }">
          <Button
            label="Показать"
            severity="secondary"
            text
            size="small"
            icon="pi pi-box"
            @click="epicProject = epicProject && epicProject.id === data.id ? null : data"
          />
        </template>
      </Column>
      <Column header="Задача" style="width: 110px">
        <template #body="{ data }">
          <Button
            label="Создать"
            severity="secondary"
            text
            size="small"
            icon="pi pi-plus"
            @click="openCreateTask(data)"
          />
        </template>
      </Column>
      <Column header="Задачи" style="width: 110px">
        <template #body="{ data }">
          <Button
            label="К задачам"
            severity="secondary"
            text
            size="small"
            icon="pi pi-list"
            @click="goToTasks(data)"
          />
        </template>
      </Column>
      <Column header="Действия" style="width: 320px">
        <template #body="{ data }">
          <div class="row-actions">
            <Button
              v-if="editId !== data.id"
              label="Переименовать"
              severity="secondary"
              text
              size="small"
              icon="pi pi-pencil"
              @click="startEdit(data)"
            />
            <template v-else>
              <InputText v-model="editName" size="small" @keyup.enter="saveRename(data)" />
              <Button label="Сохранить" size="small" icon="pi pi-check" @click="saveRename(data)" />
              <Button label="Отмена" size="small" text @click="cancelEdit" />
            </template>
            <Button
              v-if="!data.archived"
              label="В архив"
              severity="secondary"
              text
              size="small"
              icon="pi pi-ban"
              @click="archive(data)"
            />
            <Button
              v-else
              label="Восстановить"
              severity="secondary"
              text
              size="small"
              icon="pi pi-refresh"
              @click="unarchive(data)"
            />
          </div>
        </template>
      </Column>
      <template #empty>
        <div class="empty">
          <div v-if="store.projects.length === 0">Пока нет активных проектов. Создайте первый выше.</div>
        </div>
      </template>
    </DataTable>

    <!-- Панель модулей выбранного проекта -->
    <EpicPanel v-if="epicProject" :project="epicProject" />

    <!-- Диалог создания задачи (проект закреплён) -->
    <CreateTaskDialog
      :visible="createTaskProject !== null"
      :projects="store.projects"
      :initial-project="createTaskProject?.name || ''"
      :lock-project="true"
      :epic-options="createTaskEpics"
      @cancel="createTaskProject = null"
    />

    <Message v-if="store.error" severity="error" :closable="true" @close="store.error = ''">
      {{ store.error }}
    </Message>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import Card from 'primevue/card'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Message from 'primevue/message'
import { useProjectsStore } from '@/stores/projects'
import { useTasksStore } from '@/stores/tasks'
import { useEpicsStore } from '@/stores/epics'
import EpicPanel from '@/components/EpicPanel.vue'
import CreateTaskDialog from '@/components/CreateTaskDialog.vue'

const store = useProjectsStore()
const epicsStore = useEpicsStore()
const tasksStore = useTasksStore()
const toast = useToast()
const router = useRouter()
const route = useRoute()
const epicProject = ref(null)


// Диалог создания задачи: проект закреплён, модуль опциональный
const createTaskProject = ref(null)
const createTaskEpics = ref([])

async function openCreateTask(p) {
  createTaskProject.value = { id: p.id, name: p.name }
  createTaskEpics.value = []
  try {
    const epics = await epicsStore.fetchEpics(p.id)
    createTaskEpics.value = epics.map((e) => ({ value: e.id, label: e.name }))
  } catch { /* без модулей — поле в диалоге пустое */ }
}

// «К задачам» — вкладка «Активные» с фильтром по этому проекту (шаг 6: ссылка
// проекта→задачи). Вкладка переключается самим URL (watch в DashboardView);
// фильтр — в store (до смены таба, чтобы первый fetch уже был отфильтрован).
function goToTasks(p) {
  // модуль от другого проекта — сбрасываем (иначе чужой epic обрезит список)
  tasksStore.filters.epic_id = ''
  tasksStore.setFilter('project', p.name)
  router.replace({ query: { ...route.query, tab: 'active', project: p.name } })
}

const newName = ref('')
const newDesc = ref('')
const createError = ref('')

const editId = ref(null)
const editName = ref('')

onMounted(() => {
  store.fetchProjects({ archived: 'false' })
})

async function create() {
  createError.value = ''
  const name = newName.value.trim()
  if (!name) return
  try {
    const p = await store.createProject(name, newDesc.value.trim())
    newName.value = ''
    newDesc.value = ''
    toast.add({ severity: 'success', summary: `Проект создан: ${p.name}`, life: 3000 })
  } catch (e) {
    createError.value = e.response?.data?.error || 'Не удалось создать проект'
  }
}

function startEdit(p) {
  editId.value = p.id
  editName.value = p.name
}

function cancelEdit() {
  editId.value = null
  editName.value = ''
}

async function saveRename(p) {
  const name = editName.value.trim()
  if (!name) return
  try {
    const up = await store.renameProject(p.id, name, p.description)
    toast.add({ severity: 'success', summary: `Проект переименован: ${up.name}`, life: 3000 })
  } catch (e) {
    toast.add({ severity: 'error', summary: e.response?.data?.error || 'Ошибка переименования', life: 4000 })
    return
  }
  cancelEdit()
}

async function archive(p) {
  try {
    await store.archiveProject(p.id)
    toast.add({ severity: 'info', summary: `Проект в архив: ${p.name}`, life: 3000 })
  } catch (e) {
    toast.add({ severity: 'error', summary: e.response?.data?.error || 'Ошибка', life: 4000 })
  }
}

async function unarchive(p) {
  try {
    await store.unarchiveProject(p.id)
    toast.add({ severity: 'success', summary: `Восстановлен: ${p.name}`, life: 3000 })
  } catch (e) {
    toast.add({ severity: 'error', summary: e.response?.data?.error || 'Ошибка', life: 4000 })
  }
}

function formatDate(v) {
  if (!v) return '—'
  return new Date(v).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
}
</script>

<style scoped>
.projects {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.create-card {
  width: 100%;
}

.create-form {
  display: flex;
  gap: 1rem;
  align-items: flex-end;
}

.create-fields {
  display: flex;
  gap: 1rem;
  flex: 1;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  min-width: 200px;
}

.field.grow {
  flex: 1;
}

.field span {
  font-size: 0.85rem;
  color: var(--mb-text-muted);
}

.create-error {
  margin-top: 1rem;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.name {
  font-weight: 600;
}

.muted {
  color: var(--mb-text-muted);
}

.row-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.empty {
  padding: 2rem;
  text-align: center;
  color: var(--mb-text-muted);
}
</style>
