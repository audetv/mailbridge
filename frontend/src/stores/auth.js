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

    return { token, user, isAuthenticated, login, logout }
})