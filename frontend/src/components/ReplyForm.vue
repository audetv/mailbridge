<template>
  <div class="reply-form">
    <div class="kind-row">
      <Select
        v-model="kind"
        :options="KIND_OPTIONS"
        optionLabel="label"
        optionValue="value"
        class="kind-select"
        :disabled="sending"
        aria-label="Тип сообщения"
      />
    </div>
    <div class="row">
      <Textarea v-model="body" rows="3" placeholder="Текст ответа..." :disabled="sending" />
      <div class="buttons">
        <Button label="Отправить" icon="pi pi-send" @click="send" :loading="sending" />
      </div>
    </div>
    <div v-if="error" class="form-error" role="alert">{{ error }}</div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import Textarea from 'primevue/textarea'
import Button from 'primevue/button'
import Select from 'primevue/select'
import { useTasksStore } from '@/stores/tasks'

const props = defineProps({
  taskId: { type: Number, required: true }
})

const emit = defineEmits(['sent'])

const store = useTasksStore()
const body = ref('')
const kind = ref('user_comment')
const sending = ref(false)
const error = ref('')

const KIND_OPTIONS = [
  { value: 'user_comment', label: 'Комментарий' },
  { value: 'report', label: 'Отчёт' },
  { value: 'reply', label: 'Ответ пользователю' }
]

// Утверждение конкретного ответа — только под комментарием в CommentList
// (кнопка «Утвердить ответ»): контекст важен, из формы approve «последнего»
// — мисклик-зона (админ подтвердил 2026-08-30: дубль кнопку не нужен).
async function send() {
  if (!body.value.trim()) return
  sending.value = true
  error.value = ''
  try {
    await store.replyTask(props.taskId, body.value, kind.value)
    body.value = ''
    kind.value = 'user_comment'
    emit('sent')
  } catch (err) {
    // Бэкенд: {"error": "..."} — показать, а не молчать (раньше ошибка
    // терялась, и админ думал, что «отчёт» не работает).
    error.value = err?.response?.data?.error || 'Не удалось отправить сообщение'
  } finally {
    sending.value = false
  }
}
</script>

<style scoped>
.reply-form {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.kind-row {
  max-width: 20rem;
}

.row {
  display: flex;
  gap: 0.5rem;
  align-items: flex-start;
}

.reply-form .p-textarea {
  flex: 1;
}

.buttons {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  flex-shrink: 0;
}

.form-error {
  color: #b3261e;
  font-size: 0.85rem;
}
</style>
