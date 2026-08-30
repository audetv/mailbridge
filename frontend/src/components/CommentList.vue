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
        <span class="comment-badges">
          <span v-if="kindLabel(comment)" class="kind-badge">{{ kindLabel(comment) }}</span>
          <span v-if="isApproved(comment)" class="approved-badge">Утверждён</span>
        </span>
        <span class="date">{{ formatDate(comment.created_at) }}</span>
      </div>
      <div class="comment-body">{{ comment.body }}</div>

      <!-- Утверждение ответа (admin-only, ФАЗА 4) -->
      <div v-if="canApprove(comment)" class="comment-actions">
        <button
          v-if="!isApproved(comment)"
          class="approve-btn"
          :disabled="approving"
          @click="approve(comment)"
        >
          Утвердить ответ
        </button>
        <span v-else class="approved-note">Ответ утверждён</span>
      </div>

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
import { useAuthStore } from '@/stores/auth'

const props = defineProps({
  comments: { type: Array, default: () => [] }
})

const route = useRoute()
const authStore = useAuthStore()
const commentAttachments = ref({})
const approving = ref(false)

// Комментарий утверждён (approved=1, ФАЗА 4).
function isApproved(comment) {
  return comment?.approved === 1
}

// Подпись раздела по kind (Отчёт / Ответ пользователю, ФАЗА 4).
const KIND_LABELS = {
  user_comment: '',
  report: 'Отчёт',
  reply: 'Ответ пользователю',
  ai_verdict: ''
}
function kindLabel(comment) {
  return KIND_LABELS[comment?.kind] || ''
}

// Кнопка «Утвердить ответ»: admin + kind=reply.
function canApprove(comment) {
  const isAdmin = authStore.user?.username === 'admin'
  return isAdmin && comment?.kind === 'reply'
}

async function approve(comment) {
  approving.value = true
  try {
    await apiClient.patch(`/comments/${comment.id}/approve`)
    comment.approved = 1
  } catch (err) {
    // 403 (не admin) / 400 (не kind=reply) — показать текст бэкенда
    const msg = err?.response?.data?.error || 'Не удалось утвердить ответ'
    window.alert(msg)
  } finally {
    approving.value = false
  }
}

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
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.25rem;
}

.comment-badges {
  display: flex;
  gap: 0.35rem;
  margin-left: auto;
  margin-right: 0.5rem;
}

.kind-badge {
  font-size: 0.75rem;
  padding: 0.1rem 0.5rem;
  border-radius: 0.75rem;
  background: var(--mb-primary-soft);
  color: var(--mb-primary);
  font-weight: 600;
}

.approved-badge {
  font-size: 0.75rem;
  padding: 0.1rem 0.5rem;
  border-radius: 0.75rem;
  background: #d5f5dd;
  color: #1a7f37;
  border: 1px solid #a5d6b6;
  font-weight: 600;
}

.comment-actions {
  margin-top: 0.5rem;
}

.approve-btn {
  font-size: 0.85rem;
  padding: 0.3rem 0.75rem;
  border-radius: 0.4rem;
  border: 1px solid var(--mb-primary);
  background: transparent;
  color: var(--mb-primary);
  cursor: pointer;
}

.approve-btn:hover {
  background: var(--mb-primary-soft);
}

.approve-btn:disabled {
  opacity: 0.5;
  cursor: wait;
}

.approved-note {
  font-size: 0.85rem;
  color: #1a7f37;
  font-weight: 600;
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
