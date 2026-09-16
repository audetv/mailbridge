<template>
  <div class="persons">
    <!-- Список -->
    <Card>
      <template #title>Персоны</template>
      <template #subtitle>Справочник людей и компаний — заказчики и исполнители (v0.24, шаг 6)</template>
      <template #content>
        <!-- Фильтры -->
        <div class="persons-toolbar">
          <InputText
            v-model="store.search"
            placeholder="Поиск по имени или организации…"
            class="persons-search"
            @input="debouncedRefetch"
          />
          <Checkbox v-model="showUnconfirmed" label="Не подтверждённые" @change="onFilterChange" />
          <Checkbox v-model="store.isInternal" label="Свои" @change="onFilterChange" />
          <Button label="Создать" icon="pi pi-plus" severity="secondary" @click="createDialog = true" />
        </div>

        <DataTable :value="visiblePersons" :loading="store.loading" stripedRows dataKey="id">
          <Column field="name" header="Персона" style="min-width: 220px">
            <template #body="{ data }">
              <span v-if="data.name" class="name">{{ data.name }}</span>
              <span v-else class="muted name">({{ primaryEmail(data) || 'в процессе узнавания' }})</span>
              <Tag v-if="data.org && data.org !== 'машина'" :value="data.org" severity="secondary" class="org-tag" />
              <!-- 6-F: эвристика «машина» (no-reply/bot) — org='машина' → бейдж ⚙️, человек НЕ предпологается -->
              <Tag v-if="data.org === 'машина'" value="⚙️ машина" severity="help" class="org-tag" />
              <Tag v-if="data.isInternal" value="своя" severity="info" />
              <Tag v-if="!data.confirmed" value="не подтверждена" severity="warn" />
              <Tag v-if="data.archived" value="архив" severity="secondary" />
            </template>
          </Column>
          <Column field="email" header="Email" style="min-width: 200px">
            <template #body="{ data }">
              <span v-if="primaryEmail(data)">{{ primaryEmail(data) }}</span>
              <span v-else class="muted">—</span>
            </template>
          </Column>
          <Column field="created_at" header="Создана" style="width: 150px">
            <template #body="{ data }">{{ formatDate(data.created_at) }}</template>
          </Column>
          <Column header="Действия" style="width: 380px">
            <template #body="{ data }">
              <div class="row-actions">
                <Button
                  label="Профиль"
                  severity="secondary"
                  text
                  size="small"
                  icon="pi pi-eye"
                  @click="openProfile(data)"
                />
                <!-- 6-G: ручное подтверждение (не архивным, пока не подтверждена) -->
                <Button
                  v-if="!data.confirmed && !data.archived"
                  label="Подтвердить"
                  severity="secondary"
                  text
                  size="small"
                  icon="pi pi-check"
                  :loading="confirmBusyId === data.id"
                  @click="confirm(data)"
                />
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
                <Button
                  label="К задачам"
                  severity="secondary"
                  text
                  size="small"
                  icon="pi pi-list"
                  @click="goToTasks(data)"
                />
              </div>
            </template>
          </Column>
          <template #empty>
            <div class="empty">Пока нет персон. Они появляются автоматически при первом контакте или создаются кнопкой выше.</div>
          </template>
        </DataTable>
      </template>
    </Card>

    <!-- Диалог создания -->
    <Dialog v-model:visible="createDialog" header="Новая персона" :modal="true" :style="{ width: '480px' }">
      <div class="create-fields">
        <label class="field">
          <span>Имя</span>
          <InputText v-model="newName" placeholder="ФИО или псевдоним" />
        </label>
        <label class="field">
          <span>Организация</span>
          <InputText v-model="newOrg" placeholder="Например: ТРК" />
        </label>
        <label class="field">
          <span>Email</span>
          <InputText v-model="newEmail" placeholder="name@example.com" />
        </label>
        <label class="field checkbox-field">
          <Checkbox v-model="newInternal" label="Своя команда" />
        </label>
        <Message v-if="createError" severity="error" :closable="true" @close="createError = ''">
          {{ createError }}
        </Message>
      </div>
      <template #footer>
        <div class="dialog-actions">
          <Button label="Создать" icon="pi pi-plus" :loading="creating" :disabled="!newName.trim() && !newEmail.trim()" @click="create" />
          <Button label="Отмена" severity="secondary" text @click="createDialog = false" />
        </div>
      </template>
    </Dialog>

    <!-- Профиль персоны: идентичности + merge -->
    <Dialog v-if="profile" v-model:visible="profileDialog" :header="profileHeader" :modal="true" :style="{ width: '640px' }">
      <div class="profile-body">
        <div class="profile-meta">
          <span v-if="profile.org" class="muted">{{ profile.org }}</span>
          <Tag v-if="profile.isInternal" value="своя" severity="info" />
          <Tag :value="profile.confirmed ? 'подтверждена' : 'не подтверждена'" :severity="profile.confirmed ? 'success' : 'warn'" />
          <Tag v-if="profile.archived" value="архив" severity="secondary" />
        </div>

        <h4 class="profile-section">Идентичности</h4>
        <div class="identity-row" v-for="i in store.identities" :key="i.id">
          <Tag :value="i.kind" severity="secondary" />
          <span>{{ i.value }}</span>
          <span v-if="i.is_primary" class="muted">основная</span>
          <span v-if="i.provenance" class="muted">({{ i.provenance }})</span>
        </div>
        <div v-if="!store.identities.length" class="muted">Идентичностей нет</div>

        <div class="identity-add">
          <InputText v-model="newIdentityValue" placeholder="Email / телефон / ник" />
          <Button label="Привязать" size="small" severity="secondary" icon="pi pi-link" :loading="identityBusy" @click="addIdentity" />
          <Message v-if="identityError" severity="error" :closable="true" @close="identityError = ''" class="identity-error">
            {{ identityError }}
          </Message>
        </div>

        <!-- 6-G: ручное подтверждение персоны -->
        <h4 class="profile-section">Подтверждение</h4>
        <p v-if="!profile.confirmed" class="muted">
          Персона создана автоматически при первом контакте (heuristic identity).
          Подтвердите, что это корректный человек / организация.
        </p>
        <p v-else class="muted">Персона подтверждена (6-G).</p>
        <Message v-if="confirmError" severity="error" :closable="true" @close="confirmError = ''" class="merge-error">
          {{ confirmError }}
        </Message>

        <h4 class="profile-section">Слияние дубля</h4>
        <div class="merge-row">
          <InputText v-model="mergeValue" placeholder="Email или имя дубля (source)" />
          <Button label="Найти дубли" size="small" severity="secondary" icon="pi pi-search" :loading="mergeBusy" @click="suggest" />
        </div>
        <div v-if="mergeLinked" class="merge-result">
          <p>Электроника уже привязана к персоне: <strong>{{ mergeLinked.name || mergeLinked.id }}</strong></p>
          <Button label="Слить в эту" size="small" :loading="mergeApplying" @click="applyMerge(mergeLinked.id)" />
        </div>
        <div v-for="c in mergeCandidates" :key="c.id" class="merge-result">
          <p><strong>{{ c.name || c.id }}</strong> <span v-if="c.org" class="muted">({{ c.org }})</span></p>
          <Button label="Слить в эту (дубль уйдёт в архив)" size="small" :loading="mergeApplying" @click="applyMerge(c.id)" />
        </div>
        <p v-if="mergeBusy || mergeApplying" class="muted">Обработка…</p>
        <Message v-if="mergeError" severity="error" :closable="true" @close="mergeError = ''" class="merge-error">
          {{ mergeError }}
        </Message>
      </div>
      <template #footer>
        <div class="dialog-actions">
          <!-- 6-G: ручное подтверждение персоны -->
          <Button
            v-if="profile && !profile.confirmed"
            label="Подтвердить"
            icon="pi pi-check"
            :loading="confirmBusyId === (profile && profile.id)"
            @click="confirm(profile)"
          />
          <Button label="К задачам" icon="pi pi-list" severity="secondary" @click="goToTasks(profile)" />
          <Button label="Закрыть" severity="secondary" text @click="profileDialog = false" />
        </div>
      </template>
    </Dialog>
    <Message v-if="store.error" severity="error" :closable="true" @close="store.error = ''">
      {{ store.error }}
    </Message>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import Card from 'primevue/card'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Checkbox from 'primevue/checkbox'
import Message from 'primevue/message'
import Dialog from 'primevue/dialog'
import { usePersonsStore } from '@/stores/persons'
import { useTasksStore } from '@/stores/tasks'

function formatDate(iso) {
  if (!iso) return '—'
  const d = new Date(iso)
  return d.toLocaleDateString('ru-RU')
}

const store = usePersonsStore()
const tasksStore = useTasksStore()
const toast = useToast()
const router = useRouter()
const route = useRoute()

// «К задачам» — вкладка «Активные» с фильтром заказчика = персона (шаг 6:
// персона → её задачи). Ссылка persona→tasks, как project→tasks в ProjectsView.
function goToTasks(p) {
  tasksStore.filters.requestor_id = p.id
  router.replace({ query: { ...route.query, tab: 'active', requestor_id: p.id } })
}

// Фильтры =================================================================
const showUnconfirmed = ref(false)
const searchTimer = ref(null)

// «Не подтверждённые» — режим (confirmed=false); «Свои» — is_internal=true.
// archived всегда показываем отдельно (тег «архив»), т.к. archived —
// «вместо удаления» (решение: закрытые задачи не теряют ссылку).
const visiblePersons = computed(() => {
  if (!showUnconfirmed.value) return store.list
  return store.list.filter((p) => !p.confirmed)
})

function primaryEmail(p) {
  // Список несёт primary_email (основная email-идентичность, v0.24 шаг 6);
  // персона без имени «в процессе узнавания» — email единственная деталь.
  return p.primary_email || ''
}

function onFilterChange() {
  refetch()
}

function debouncedRefetch() {
  clearTimeout(searchTimer.value)
  searchTimer.value = setTimeout(refetch, 300)
}

function refetch() {
  store.page = 1
  store.fetchPersons()
}

// Создание =================================================================
const createDialog = ref(false)
const newName = ref('')
const newOrg = ref('')
const newEmail = ref('')
const newInternal = ref(false)
const createError = ref('')
const creating = ref(false)

async function create() {
  createError.value = ''
  creating.value = true
  try {
    const payload = {
      name: newName.value.trim(),
      org: newOrg.value.trim(),
      is_internal: newInternal.value
    }
    if (newEmail.value.trim()) payload.email = newEmail.value.trim()
    const p = await store.createPerson(payload)
    toast.add({ severity: 'success', summary: `Персона создана: ${p.name || p.id}`, life: 3000 })
    resetCreate()
    store.fetchPersons()
  } catch (e) {
    createError.value = e.response?.data?.error || 'Не удалось создать персону'
  } finally {
    creating.value = false
  }
}

function resetCreate() {
  createDialog.value = false
  newName.value = ''
  newOrg.value = ''
  newEmail.value = ''
  newInternal.value = false
}

// Профиль и идентичности ===================================================
const profile = ref(null)
const profileDialog = ref(false)
const newIdentityValue = ref('')
const identityBusy = ref(false)
const identityError = ref('')

const profileHeader = computed(() => {
  if (!profile.value) return 'Профиль'
  return profile.value.name || profile.value.id
})

async function openProfile(p) {
  profile.value = p
  store.identities = []
  newIdentityValue.value = ''
  identityError.value = ''
  mergeValue.value = ''
  mergeCandidates.value = []
  mergeLinked.value = null
  mergeError.value = ''
  profileDialog.value = true
  await store.fetchIdentities(p.id)
}

async function addIdentity() {
  const v = newIdentityValue.value.trim()
  if (!v || !profile.value) return
  identityBusy.value = true
  identityError.value = ''
  try {
    // Email vs телефон — по виду (контракт: kind+value; kind email/phone — v0.24)
    const kind = v.includes('@') ? 'email' : 'phone'
    await store.addIdentity(profile.value.id, { kind, value: v })
    newIdentityValue.value = ''
    await store.fetchIdentities(profile.value.id)
  } catch (e) {
    identityError.value = e.response?.data?.error || 'Не удалось привязать идентичность'
  } finally {
    identityBusy.value = false
  }
}

// 6-G: ручное подтверждение персоны ===================================
// Кнопка «Подтвердить» (строка списка + футер профиля) → store.confirmPerson:
// PUT /api/persons/{id} ПОЛНЫМ payload (UpdatePerson — full-overwrite, см. store).
const confirmError = ref('')
const confirmBusyId = ref(null)

async function confirm(p) {
  if (!p) return
  confirmError.value = ''
  confirmBusyId.value = p.id
  try {
    const updated = await store.confirmPerson(p)
    toast.add({
      severity: 'success',
      summary: `Подтверждена: ${updated.name || p.name || p.id}`,
      life: 3000
    })
    if (profile.value && profile.value.id === p.id) {
      profile.value = { ...profile.value, ...updated }
    }
  } catch (e) {
    confirmError.value = e.response?.data?.error || 'Не удалось подтвердить персону'
  } finally {
    confirmBusyId.value = null
  }
}

// Слияние ==================================================================
const mergeValue = ref('')
const mergeBusy = ref(false)
const mergeApplying = ref(false)
const mergeError = ref('')
const mergeCandidates = ref([])
const mergeLinked = ref(null)

async function suggest() {
  const v = mergeValue.value.trim()
  if (!v) return
  mergeBusy.value = true
  mergeError.value = ''
  mergeCandidates.value = []
  mergeLinked.value = null
  try {
    const kind = v.includes('@') ? 'email' : undefined
    const res = kind
      ? await store.suggestMatches({ kind, value: v })
      : await store.suggestMatches({ kind: 'name', value: v })
    if (res.linked) mergeLinked.value = res.person
    else mergeCandidates.value = res.candidates
  } catch (e) {
    mergeError.value = e.response?.data?.error || 'Не найдено'
  } finally {
    mergeBusy.value = false
  }
}

async function applyMerge(sourceId) {
  if (!profile.value) return
  mergeApplying.value = true
  mergeError.value = ''
  try {
    await store.mergePersons({ targetId: profile.value.id, sourceId })
    toast.add({ severity: 'success', summary: 'Дубль объединён', life: 3000 })
    mergeCandidates.value = []
    mergeLinked.value = null
    await store.fetchIdentities(profile.value.id)
    store.fetchPersons()
  } catch (e) {
    mergeError.value = e.response?.data?.error || 'Не удалось объединить'
  } finally {
    mergeApplying.value = false
  }
}

// Архив ====================================================================
async function archive(p) {
  try {
    await store.setArchived(p.id, true)
    toast.add({ severity: 'success', summary: 'Персона в архиве', life: 3000 })
    store.fetchPersons()
  } catch (e) {
    toast.add({ severity: 'error', summary: e.response?.data?.error || 'Ошибка архивации', life: 4000 })
  }
}

async function unarchive(p) {
  try {
    await store.setArchived(p.id, false)
    toast.add({ severity: 'success', summary: 'Персона восстановлена', life: 3000 })
    store.fetchPersons()
  } catch (e) {
    toast.add({ severity: 'error', summary: e.response?.data?.error || 'Ошибка восстановления', life: 4000 })
  }
}

onMounted(() => {
  store.fetchPersons()
})

onBeforeUnmount(() => {
  clearTimeout(searchTimer.value)
})
</script>

<style scoped>
.persons-toolbar {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}
.persons-search {
  min-width: 280px;
}
.row-actions {
  display: flex;
  gap: 0.25rem;
  flex-wrap: wrap;
  align-items: center;
}
.create-fields {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
.checkbox-field {
  flex-direction: row;
  align-items: center;
}
.profile-meta {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  flex-wrap: wrap;
  margin-bottom: 0.5rem;
}
.profile-section {
  margin: 1rem 0 0.5rem;
}
.identity-row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  padding: 0.25rem 0;
  flex-wrap: wrap;
}
.identity-add {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-top: 0.5rem;
  flex-wrap: wrap;
}
.identity-error,
.merge-error {
  width: 100%;
}
.merge-row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  flex-wrap: wrap;
}
.merge-result {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-top: 0.5rem;
  flex-wrap: wrap;
}
.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}
.org-tag {
  margin-left: 0.5rem;
}
.empty {
  padding: 1rem 0;
  color: var(--mb-text-muted);
}
.muted {
  color: var(--mb-text-muted);
  font-size: 0.85rem;
}
.name {
  font-weight: 500;
}
</style>
