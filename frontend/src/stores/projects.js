import { defineStore } from 'pinia'
import { ref } from 'vue'
import apiClient from '@/api/client'

// Проекты (внутренние): список, создание, переименование, архив/разархив.
// WS-события project_* потребляет DashboardView (тосты + fetchProjects()).
export const useProjectsStore = defineStore('projects', () => {
  const projects = ref([])
  const loading = ref(false)
  const error = ref('')

  async function fetchProjects(params = { archived: 'false' }) {
    loading.value = true
    error.value = ''
    try {
      const { data } = await apiClient.get('/projects', { params })
      projects.value = data
    } catch (e) {
      error.value = e.response?.data?.error || e.message
      projects.value = []
    } finally {
      loading.value = false
    }
  }

  async function createProject(name, description) {
    const { data } = await apiClient.post('/projects', { name, description })
    await fetchProjects({ archived: 'false' })
    return data
  }

  async function renameProject(id, name, description) {
    const { data } = await apiClient.put(`/projects/${id}`, { name, description })
    await fetchProjects({ archived: 'false' })
    return data
  }

  async function archiveProject(id) {
    const { data } = await apiClient.delete(`/projects/${id}`)
    await fetchProjects({ archived: 'false' })
    return data
  }

  async function unarchiveProject(id) {
    const { data } = await apiClient.post(`/projects/${id}/unarchive`)
    await fetchProjects({ archived: 'false' })
    return data
  }

  function projectById(id) {
    return projects.value.find((p) => p.id === id) || null
  }

  function projectByName(name) {
    if (!name) return null
    return projects.value.find((p) => p.name === name) || null
  }

  return {
    projects,
    loading,
    error,
    fetchProjects,
    createProject,
    renameProject,
    archiveProject,
    unarchiveProject,
    projectById,
    projectByName
  }
})
