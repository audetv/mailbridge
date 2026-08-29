import PrimeVue from 'primevue/config'
import Aura from '@primeuix/themes/aura'

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
