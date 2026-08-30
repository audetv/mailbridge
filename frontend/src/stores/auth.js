import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import apiClient from '@/api/client'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user = ref(null)

  const isAuthenticated = computed(() => !!token.value)

  async function login(username, password) {
    const { data } = await apiClient.post('/auth/login', { username, password })
    token.value = data.token
    user.value = data.user
    localStorage.setItem('token', data.token)
    return data
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('token')
  }

  // Восстановление user после полной перезагрузки (F5 / прямой URL).
  // Token живёт в localStorage, user — только в памяти: без этого при
  // перезагрузке «кто я» неизвестно (кнопка approve, RBAC — ломаются).
  async function restore() {
    if (user.value || !token.value) return
    try {
      const { data } = await apiClient.get('/auth/me')
      if (data && data.username) user.value = { username: data.username }
    } catch (err) {
      // 401 — токен мёртв: чистим состояние.
      if (err?.response?.status === 401) logout()
    }
  }

  return { token, user, isAuthenticated, login, logout, restore }
})
