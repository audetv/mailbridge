<template>
  <div class="reply-form">
    <div class="kind-row">
      <Select
        v-model="kind"
        :options="KIND_OPTIONS"
        optionLabel="label"
        class="kind-select"
        :disabled="sending"
        aria-label="Тип сообщения"
      />
    </div>
    <div class="row">
      <Textarea v-model="body" rows="3" placeholder="Текст ответа..." :disabled="sending" />
      <div class="buttons">
        <Button
          v-if="isAdmin"
          label="Утвердил ответ"
          icon="pi pi-check"
          severity="success"
          :disabled="!latestReply || approving"
          @click="approveLastReply"
        />
        <Button label="Отправить" icon="pi pi-send" @click="send" :loading="sending" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import Textarea from 'primevue/textarea'
import Button from 'primevue/button'
import Select from 'primevue/select'
import { useTasksStore } from '@/stores/tasks'
import { useAuthStore } from '@/stores/auth'

const props = defineProps({
  taskId: { type: Number, required: true },
  // Комментарии задачи (для «Утвердил ответ» на последнем kind=reply)
  comments: { type: Array, default: () => [] }
})

const emit = defineEmits(['sent'])

const store = useTasksStore()
const authStore = useAuthStore()
const body = ref('')
const sending = ref(false)
const approving = ref(false)
const kind = ref('user_comment')

const KIND_OPTIONS = [
  { value: 'user_comment', label: 'Комментарий' },
  { value: 'report', label: 'Отчёт' },
  { value: 'reply', label: 'Ответ пользователю' }
]

const isAdmin = computed(() => authStore.user?.username === 'admin')

// Последний «Ответ пользователю» (kind=reply, direction=out) — кандидат на утверждение.
const latestReply = computed(() => {
  const replies = props.comments.filter(c => c.direction === 'out' && c.kind === 'reply')
  return replies.length ? replies[replies.length - 1] : null
})

async function approveLastReply() {
  const target = latestReply.value
  if (!target) {
    window.alert('Сначала отправьте «Ответ пользователю»')
    return
  }
  approving.value = true
  try {
    await store.approveComment(target.id)
    target.approved = 1
  } catch (err) {
    window.alert(err?.response?.data?.error || 'Не удалось утвердить ответ')
  } finally {
    approving.value = false
  }
}

async function send() {
  if (!body.value.trim()) return
  sending.value = true
  try {
    await store.replyTask(props.taskId, body.value, kind.value)
    body.value = ''
    kind.value = 'user_comment'
    emit('sent')
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
</style>
