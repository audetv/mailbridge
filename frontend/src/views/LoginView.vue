<template>
  <div class="login-container">
    <Card class="login-card">
      <template #title>
        <h2>Mailbridge</h2>
      </template>
      <template #content>
        <div class="flex flex-col gap-3">
          <InputText v-model="username" placeholder="Логин" />
          <InputText v-model="password" placeholder="Пароль" type="password" />
          <Message v-if="error" severity="error" :closable="false">{{ error }}</Message>
        </div>
      </template>
      <template #footer>
        <Button label="Войти" @click="handleLogin" :loading="loading" />
      </template>
    </Card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Card from 'primevue/card'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import Message from 'primevue/message'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    await authStore.login(username.value, password.value)
    router.push('/')
  } catch (e) {
    error.value = e.response?.data?.error || 'Ошибка входа'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  /* background: var(--p-surface-100); */
}

.login-card {
  width: 400px;
}
</style>