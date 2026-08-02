<template>
    <div class="reply-form">
        <Textarea v-model="body" rows="3" placeholder="Текст ответа..." />
        <Button label="Отправить" icon="pi pi-send" @click="send" :loading="sending" />
    </div>
</template>

<script setup>
import { ref } from 'vue'
import Textarea from 'primevue/textarea'
import Button from 'primevue/button'

const props = defineProps({
    taskId: { type: Number, required: true }
})

const emit = defineEmits(['sent'])

const body = ref('')
const sending = ref(false)

async function send() {
    if (!body.value.trim()) return
    sending.value = true
    try {
        const { useTasksStore } = await import('@/stores/tasks')
        const store = useTasksStore()
        await store.replyTask(props.taskId, body.value)
        body.value = ''
        emit('sent')
    } finally {
        sending.value = false
    }
}
</script>

<style scoped>
.reply-form {
    display: flex;
    gap: 0.5rem;
    align-items: flex-start;
}

.reply-form .p-textarea {
    flex: 1;
}
</style>