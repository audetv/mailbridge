<script setup>
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Select from 'primevue/select'
import { useTasksStore } from '@/stores/tasks'
import { useProjectsStore } from '@/stores/projects'
import { useEpicsStore } from '@/stores/epics'

// Диалог ручного создания задачи (v0.22 step 11.3).
// Обязательно — только заголовок. Проект: props.projects + изначальный
// (ProjectsView передаёт закреплённый проект). Модуль — опциональный Select.
// Статус всегда «новая» — задаётся сервером. После создания — переход
// на задачу в режиме редактирования (TaskDetailView).
const props = defineProps({
  visible: { type: Boolean, default: false },
  projects: { type: Array, default: () => [] },
  initialProject: { type: String, default: '' },
  // true — проект закреплён (ProjectsView): Select не показываем, только контекст.
  lockProject: { type: Boolean, default: false },
  epicOptions: { type: Array, default: () => [] }
})

const emit = defineEmits(['cancel', 'success'])

// Крестик в шапке диалога: PrimeVue шлёт update:visible (false) — пробрасываем
// в родител, чтобы он выключил флаг (v-model:visible на обоих уровнях).
const innerVisible = computed({
  get: () => props.visible,
  set: (v) => {
    if (!v) emit('cancel')
  },
})

const router = useRouter()
const toast = useToast()
const store = useTasksStore()
const projectsStore = useProjectsStore()
const epicsStore = useEpicsStore()

const title = ref('')
const project = ref(props.initialProject || '')
const epic = ref(null)
const description = ref('')
const loading = ref(false)
const localEpics = ref([])

// Эффективные опции «Модуль»: pre-loaded из вида (epicOptions) или
// подгруженные самим диалогом по выбранному проекту.
function effectiveEpics() {
  return props.epicOptions.length ? props.epicOptions : localEpics.value
}

async function loadProjectEpics(projectName) {
  localEpics.value = []
  if (!projectName || props.epicOptions.length) return
  const p = projectsStore.projectByName(projectName)
  if (!p) return
  try {
    const epics = await epicsStore.fetchEpics(p.id)
    localEpics.value = epics.map((e) => ({ value: e.id, label: e.name }))
  } catch {
    localEpics.value = []
  }
}

watch(
  () => props.visible,
  (open) => {
    if (open) {
      title.value = ''
      project.value = props.initialProject || (props.projects[0]?.name || '')
      epic.value = null
      description.value = ''
      localEpics.value = []
      loadProjectEpics(project.value)
    }
  }
)

watch(project, (name) => {
  // Старый модуль невалиден для нового проекта — сбрасываем.
  epic.value = null
  loadProjectEpics(name)
})

const canSubmit = computed(
  () => title.value.trim().length > 0 && !!project.value && !loading.value
)

const showProjectSelect = computed(
  () => !props.lockProject && props.projects.length > 1
)

// Ставим модуль (value из effectiveEpics). Вынесено в метод, чтобы тест мог
// задать его без обхода ref-прокси (shallow setupState: присваивание w.vm.epic
// затирало бы сам ref). В реальном UI это делает Select через v-model.
function setEpic(id) {
  epic.value = id
}

async function buildPayload() {

  const payload = { title: title.value.trim(), project: project.value }
  if (description.value.trim()) payload.description = description.value.trim()
  if (epic.value) payload.epic_id = epic.value
  return payload
}

async function submit() {
  if (!canSubmit.value) return
  loading.value = true
  try {
    const task = await store.createTask(await buildPayload())
    const id = task?.id ?? task?.data?.id
    emit('success', task)
    router.push({ name: 'task-detail', params: { id } })
  } catch (err) {
    const resp = err?.response
    toast.add({
      severity: 'error',
      summary: 'Не удалось создать задачу',
      detail: resp?.data?.error || err?.message || 'Попробуйте позже',
      life: 4000
    })
  } finally {
    loading.value = false
  }
}

function cancel() {
  emit('cancel')
}

defineExpose({ title, project, epic, description, submit, buildPayload, canSubmit, showProjectSelect, effectiveEpics, loadProjectEpics, localEpics, setEpic })
</script>

<template>
  <Dialog
    v-model:visible="innerVisible"
    modal
    :style="{ width: '480px' }"
    header="Новая задача"
    :dismissableMask="false"
    @close="cancel"
  >
    <div class="create-task-form">
      <label class="field">
        <span>Заголовок <span class="req">*</span></span>
        <InputText
          v-model="title"
          maxlength="500"
          placeholder="Что нужно сделать?"
          autofocus
          @keyup.enter="submit"
        />
      </label>

      <label v-if="showProjectSelect" class="field">
        <span>Проект</span>
        <Select v-model="project" :options="projects" optionLabel="name" placeholder="Выберите проект" />
      </label>
      <div v-else-if="project" class="ctx-project">Проект: <strong>{{ project }}</strong></div>

      <label v-if="effectiveEpics().length > 0" class="field">
        <span>Модуль (необязательно)</span>
        <Select
          v-model="epic"
          :options="effectiveEpics()"
          optionLabel="label"
          allowEmpty
          placeholder="без модуля"
        />
      </label>

      <label class="field">
        <span>Описание (необязательно)</span>
        <Textarea v-model="description" rows="3" :autoResize="true" @keyup.enter.exact="submit" />
      </label>

      <p class="hint">Статус будет «новая». После создания откроется задача в режиме редактирования.</p>
    </div>

    <template #footer>
      <div class="footer-btns">
        <Button label="Отмена" severity="secondary" outlined @click="cancel" />
        <Button label="Создать" icon="pi pi-plus" :loading="loading" :disabled="!canSubmit" @click="submit" />
      </div>
    </template>
  </Dialog>
</template>

<style scoped>
.create-task-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
  color: var(--p-color-700, #444);
}
.req {
  color: #d9534f;
}
.ctx-project {
  font-size: 13px;
}
.hint {
  margin: 0;
  font-size: 12px;
  color: var(--p-color-500, #888);
}
.footer-btns {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
</style>
