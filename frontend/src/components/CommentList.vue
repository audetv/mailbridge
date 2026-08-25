<template>
  <div class="comment-list">
    <div v-if="comments.length === 0" class="empty">Нет комментариев</div>
    <div
      v-for="comment in comments"
      :key="comment.id"
      class="comment"
      :class="[comment.direction, comment.kind]"
    >
      <div class="comment-header">
        <span class="author">{{ comment.author }}</span>
        <span class="date">{{ formatDate(comment.created_at) }}</span>
      </div>
      <div class="comment-body">{{ comment.body }}</div>

      <!-- Вложения комментария -->
      <div v-if="commentAttachments[comment.id]?.length > 0" class="comment-attachments">
        <div v-for="att in commentAttachments[comment.id]" :key="att.id" class="comment-attachment-item">
          <i class="pi pi-paperclip" />
          <a
            :href="`/api/attachments/${att.storage_path}/${encodeURIComponent(att.filename)}`"
            target="_blank"
          >
            {{ att.filename }}
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import apiClient from '@/api/client'

const props = defineProps({
  comments: { type: Array, default: () => [] }
})

const route = useRoute()
const commentAttachments = ref({})

async function loadAttachments(commentId) {
  try {
    const { data } = await apiClient.get(`/tasks/${route.params.id}/comments/${commentId}/attachments`)
    commentAttachments.value[commentId] = data
  } catch {
    commentAttachments.value[commentId] = []
  }
}

watch(
  () => props.comments,
  (newComments) => {
    for (const comment of newComments) {
      if (!commentAttachments.value[comment.id]) {
        loadAttachments(comment.id)
      }
    }
  },
  { immediate: true }
)

function formatDate(dateStr) {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return (
    d.toLocaleDateString('ru-RU') +
    ' ' +
    d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  )
}
</script>

<style scoped>
.comment-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.empty {
  color: var(--mb-text-muted);
  text-align: center;
  padding: 2rem;
}

.comment {
  padding: 0.75rem;
  border-radius: 0.5rem;
}

.comment.in {
  /* background: var(--mb-surface-hover); */
}

.comment.out {
  background: var(--mb-primary-soft);
  margin-left: 1rem;
}

.comment.ai_verdict {
  /* background: var(--mb-primary-softer); */
  border-left: 3px solid var(--mb-primary);
}

.comment-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 0.25rem;
}

.author {
  font-weight: 600;
  font-size: 1rem;
}

.date {
  font-size: 0.9rem;
  color: var(--mb-text-muted);
}

.comment-body {
  font-size: 1rem;
  white-space: pre-wrap;
}

.comment-attachments {
  margin-top: 0.5rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.comment-attachment-item {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.9rem;
}

.comment-attachment-item a {
  text-decoration: none;
  color: var(--mb-primary);
}
</style>
