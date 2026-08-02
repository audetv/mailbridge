<template>
    <div class="reply-form">
        <Textarea v-model="body" rows="3" placeholder="Текст ответа..." class="reply-input" />
        <Button label="Отправить" icon="pi pi-send" @click="send" :loading="sending" :disabled="!body.trim()" />
    </div>
</template>

<script setup>
import { ref } from 'vue'
import Textarea from 'primevue/textarea'
import Button from 'primevue/button'

const emit = defineEmits(['reply'])

const body = ref('')
const sending = ref(false)

async function send() {
    if (!body.value.trim()) return
    sending.value = true
    try {
        emit('reply', body.value)
        body.value = ''
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

.reply-input {
    width: 100%;
}
</style>