import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import Aura from '@primeuix/themes/aura'
import App from './App.vue'
import router from './router'
import 'primeicons/primeicons.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(PrimeVue, {
    license: "eyJpZCI6IjdlOTA1NDRiLTBmYjktNDE0Yy1iZDM0LTFmM2FkNWE2MjQ1NSIsInByb2R1Y3QiOiJwcmltZXVpIiwidGllciI6ImNvbW11bml0eSIsInR5cGUiOiJkZXYiLCJpYXQiOjE3ODU2MDkyMzMsImV4cCI6MTgxNzE0NTIzM30.SWjZfdOgybwbisIsg0rCbY-rpyg_yBTWbJqLMTnivZEn5-_VEUPSivWe91PD8t-ucZh2ZnnSHn9SEEAnotgAAQ",
    theme: {
        preset: Aura
    }
})

app.mount('#app')