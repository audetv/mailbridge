import { defineStore } from 'pinia'
import { ref } from 'vue'
import apiClient from '@/api/client'

export const useInboxStore = defineStore('inbox', () => {
  const items = ref([])
  const total = ref(0)
  const loading = ref(false)
  const unreadCount = ref(0)
  const filters = ref({
    status: '',
    page: 1,
    perPage: 20
  })

  async function fetchItems() {
    loading.value = true
    try {
      const { data } = await apiClient.get('/inbox', { params: filters.value })
      items.value = data.items
      total.value = data.total
    } finally {
      loading.value = false
    }
  }

  async function fetchUnreadCount() {
    try {
      const { data } = await apiClient.get('/inbox', {
        params: { status: 'unread', page: 1, per_page: 1 }
      })
      unreadCount.value = data.total || 0
    } catch {
      unreadCount.value = 0
    }
  }

  function setFilter(key, value) {
    filters.value[key] = value
    filters.value.page = 1
    fetchItems()
  }

  return { items, total, loading, unreadCount, filters, fetchItems, fetchUnreadCount, setFilter }
})