import { defineStore } from 'pinia'
import apiClient from '../api/client'

// Персоны (v0.24, шаг 6) — каталог людей/компаний.
// Канон: docs/ontology.md §7.7 (Person), §7.7.1 (PersonIdentity).
// Роли (requestor/assignee) — на задаче, не на персоне (решение 2 плана).
export const usePersonsStore = defineStore('persons', {
  state: () => ({
    list: [],
    total: 0,
    loading: false,
    error: '',
    selected: null,
    identities: [],
    // Фильтры (контракт GET /api/persons)
    search: '',
    confirmed: false,
    isInternal: false,
    showArchived: false,
    page: 1,
    perPage: 50
  }),

  actions: {
    // GET /api/persons — каталог + фильтры (search по name/org).
    async fetchPersons() {
      this.loading = true
      try {
        const params = { page: this.page, per_page: this.perPage }
        if (this.search) params.search = this.search
        if (this.confirmed) params.confirmed = 'true'
        if (this.isInternal) params.is_internal = 'true'
        if (this.showArchived) params.archived = 'true'
        const { data } = await apiClient.get('/persons', { params })
        this.list = data.persons || []
        this.total = data.total || 0
        this.error = ''
      } catch (e) {
        this.error = e.response?.data?.error || 'Не удалось загрузить персон'
        this.list = []
      } finally {
        this.loading = false
      }
    },

    // GET /api/persons/{id}
    async fetchPerson(id) {
      const { data } = await apiClient.get(`/persons/${id}`)
      this.selected = data
      return data
    },

    // PUT /api/persons/{id} — name/org/is_internal.
    async updatePerson(id, payload) {
      const { data } = await apiClient.put(`/persons/${id}`, payload)
      return data
    },

    // POST /api/persons — создание (+опц. email → идентичность).
    async createPerson(payload) {
      const { data } = await apiClient.post('/persons', payload)
      return data
    },

    // POST /api/persons/{id}/archive | /unarchive
    async setArchived(id, archived) {
      const url = archived ? `/persons/${id}/archive` : `/persons/${id}/unarchive`
      const { data } = await apiClient.post(url)
      return data
    },

    // Идентичности §7.7.1 ===================================================

    // GET /api/persons/{id}/identities → {identities: []}
    async fetchIdentities(id) {
      const { data } = await apiClient.get(`/persons/${id}/identities`)
      this.identities = data.identities || []
      return this.identities
    },

    // POST /api/persons/{id}/identities — {kind,value,external_id}
    async addIdentity(personId, { kind = 'email', value, externalId }) {
      const body = { kind, value }
      if (externalId) body.external_id = externalId
      const { data } = await apiClient.post(`/persons/${personId}/identities`, body)
      return data
    },

    // Слияние дублей ========================================================

    // POST /api/persons-suggest {kind,value,name|email} — кандидаты.
    // Контракт: fast-path {linked:true, person} (электроника уже привязана)
    // или {linked:false, candidates:[...]}.
    async suggestMatches({ kind = 'email', value, name = '' }) {
      const { data } = await apiClient.post('/persons-suggest', { kind, value, name })
      if (data.linked) return { linked: true, person: data.person }
      return { linked: false, candidates: data.candidates || [] }
    },

    // POST /api/persons-merge {target_id, source_id} — перенос и архив дубля.
    async mergePersons({ targetId, sourceId }) {
      const { data } = await apiClient.post('/persons-merge', {
        target_id: targetId,
        source_id: sourceId
      })
      return data
    },

    // Роли персоны на задаче (шаг 6, решение 6) ============================

    // PUT /api/tasks/{taskId}/persons — {requestor_id?, assignee_id?}.
    // Опущенное поле не меняется; null — отвязать.
    async setTaskRoles(taskId, { requestorId = null, assigneeId = null } = {}) {
      const payload = {}
      if (requestorId !== undefined) payload.requestor_id = requestorId
      if (assigneeId !== undefined) payload.assignee_id = assigneeId
      const { data } = await apiClient.put(`/tasks/${taskId}/persons`, payload)
      return data
    }
  }
})
