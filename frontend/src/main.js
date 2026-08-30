import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import Aura from '@primeuix/themes/aura'
import App from './App.vue'
import router from './router'
import { useThemeStore } from '@/stores/theme'
import { useAuthStore } from '@/stores/auth'
import 'primeicons/primeicons.css'
import './assets/global.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)

// Применяем тему сразу при старте: store читает localStorage и вешает
// .dark на <html>. Без этого класс теряется при полной перезагрузке страницы
// (раньше store создавался только внутри DashboardView).
useThemeStore()

// Восстановление «кто я» после F5/прямого URL: token в localStorage, а username
// живёт только в памяти. Без restore authStore.user = null, и admin-only UI
// (кнопка «Утвердил ответ») некорректно спрятан до первого логина.
useAuthStore().restore()
app.use(PrimeVue, {
  license:
    'eyJpZCI6IjdlOTA1NDRiLTBmYjktNDE0Yy1iZDM0LTFmM2FkNWE2MjQ1NSIsInByb2R1Y3QiOiJwcmltZXVpIiwidGllciI6ImNvbW11bml0eSIsInR5cGUiOiJkZXYiLCJpYXQiOjE3ODU2MDkyMzMsImV4cCI6MTgxNzE0NTIzM30.SWjZfdOgybwbisIsg0rCbY-rpyg_yBTWbJqLMTnivZEn5-_VEUPSivWe91PD8t-ucZh2ZnnSHn9SEEAnotgAAQ',
  theme: {
    preset: Aura,
    options: {
      darkModeSelector: '.dark'
    }
  }
})
app.use(ToastService)

app.mount('#app')
