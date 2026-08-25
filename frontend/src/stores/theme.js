import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const isDark = ref(localStorage.getItem('mailbridge_theme') === 'dark')

  function applyTheme() {
    if (isDark.value) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }

  function toggleTheme() {
    isDark.value = !isDark.value
    localStorage.setItem('mailbridge_theme', isDark.value ? 'dark' : 'light')
    applyTheme()
  }

  // Применяем при инициализации
  applyTheme()

  return { isDark, toggleTheme }
})
