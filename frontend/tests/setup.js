import PrimeVue from 'primevue/config'
import Aura from '@primeuix/themes/aura'

// happy-dom не реализует Web Storage — подставляем minimal-меморизационную реализацию
// (достаточно для DashboardView: localStorage.getItem/setItem('mailbridge_active_tab')).
function makeStorage() {
  const m = new Map()
  return {
    getItem: (k) => (m.has(k) ? m.get(k) : null),
    setItem: (k, v) => m.set(k, String(v)),
    removeItem: (k) => m.delete(k),
    clear: () => m.clear()
  }
}
if (typeof globalThis.localStorage === 'undefined') {
  globalThis.localStorage = makeStorage()
}

// PrimeVue 5 (Aura) — компоненты (в т.ч. cx()/тема) требуют глобальный PrimeVue,
// как в src/main.js: app.use(PrimeVue, { theme: { preset: Aura } }).
// Регистрируем в контексте Vue-конфига vitest (наследуется каждым mount()).
import { config } from '@vue/test-utils'

config.global.plugins = [
  [
    PrimeVue,
    {
      theme: {
        preset: Aura,
        options: { darkModeSelector: '.dark' }
      }
    }
  ]
]
