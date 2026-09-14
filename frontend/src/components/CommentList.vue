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
        <span class="author" :title="isLegacyAIUser(comment) ? 'AI-саммари письма' : undefined">
          {{ isLegacyAIUser(comment) ? legacyAIAuthor(comment) : comment.author }}
        </span>
        <span class="comment-badges">
          <span v-if="isLegacyAIUser(comment)" class="kind-badge ai-summary-badge">AI-саммари</span>
          <span v-if="kindLabel(comment)" class="kind-badge">{{ kindLabel(comment) }}</span>
          <span v-if="isApproved(comment)" class="approved-badge">Утверждён</span>
        </span>
        <span class="date">
          {{ formatDate(comment.created_at) }}
          <button type="button" class="copy-btn"
                  title="Копировать (чистый текст, без символов Markdown)"
                  :aria-label="'Копировать чистый текст комментария ' + comment.id"
                  @click="copyBody(comment, 'text')">
            <i v-if="copyState(comment.id) === 'text'" class="pi pi-check"></i>
            <i v-else class="pi pi-clipboard"></i>
          </button>
          <button type="button" class="copy-btn"
                  title="Копировать как Markdown (дословно: символы ** ## - сохраняются)"
                  :aria-label="'Копировать Markdown комментария ' + comment.id"
                  @click="copyBody(comment, 'md')">
            <span v-if="copyState(comment.id) === 'md'" class="copy-done">✓ MD</span>
            <template v-else>MD</template>
          </button>
        </span>
      </div>

      <!-- Шаг 3b: MD-комментарии рендерим как Markdown (v-html через sanitize),
           plain-комментарии — прежний текстовый путь с linkify. -->
      <div v-if="isMd(comment)" class="comment-body md" v-html="renderMarkdown(comment)"></div>
      <div v-else class="comment-body">
        <template v-for="(seg, i) in linkify(comment.body)" :key="i">
          <a v-if="seg.href" :href="seg.href" target="_blank" rel="noopener">{{ seg.text }}</a>
          <template v-else>{{ seg.text }}</template>
        </template>
      </div>

      <!-- v0.23 шаг 5 (решение владельца 14.09, задача 378):
           detail «Оригинал письма» — ТОЛЬКО в комментарии-саммари (author='user',
           legacy AI-саммари письма) — там, где письмо смотрят.
           Из AI-ответа (ai_verdict) detail УБРАН: вердикт = только саммари,
           письмо и так стоит выше в потоке. -->
      <details
        v-if="showOriginal(comment)"
        class="comment-original"
      >
        <summary>
          Оригинал письма&ensp;
          <span class="comment-original-meta">
            {{ senderOf(comment) }} · {{ subjectOf(comment) }}
          </span>
        </summary>
        <!-- Спойлер = ТОЛЬКО новая часть письма (diff с предыдущим письмом треда
              + отсечение подписи/«Просьба при ответе»). Вся история — в ленте Inbox.
              (решение владельца, 15.09, задача 378: «простыня» у Маргариты = её
              клиент приписывает историю; спойлер показывает только ответ). -->
        <div class="comment-original-body">
          {{ originalBody(comment) }}
          <div class="comment-original-foot">
            Вся переписка — в ленте:
            <router-link :to="'/inbox/' + (inboxItemOf(comment) && inboxItemOf(comment).id)">Открыть в Inbox</router-link>
          </div>
        </div>
      </details>

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

      <!-- Вложения комментария.
           AI-вердикт (author='ai', kind='ai_verdict') — ВСЕГДА только саммари:
           вложения из письма туда не выносим (решение владельца, 15.09).
           У legacy AI-саммари (author='user') вложения ОСТАЮТcя. -->
      <div v-if="showAttachments(comment) && commentAttachments[comment.id]?.length > 0" class="comment-attachments">
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
import { copyComment } from '@/utils/copy-comment'
import { linkify } from '@/utils/linkify'
import { renderMd, looksLikeMd } from '@/utils/render-md'
import { newMessagePartOf } from '@/utils/new-message-part'

// Шаг 3b: MD-комментарии рендерим как Markdown (v-html через sanitize),
// plain-комментарии — прежний текстовый путь с linkify.
// Кэш по id+body: body меняется редких (новый коммент) — не пересчитываем.
const mdRenderCache = new Map()
function isMd(comment) {
  return looksLikeMd(comment?.body ?? '')
}
function renderMarkdown(comment) {
  const body = comment?.body ?? ''
  const key = comment?.id ?? -1
  const cached = mdRenderCache.get(key)
  if (cached && cached.body === body) return cached.html
  const html = renderMd(body)
  mdRenderCache.set(key, { body, html })
  // простой bound: не копим кэш бесконечно
  if (mdRenderCache.size > 200) {
    const first = mdRenderCache.keys().next().value
    mdRenderCache.delete(first)
  }
  return html
}

const props = defineProps({
  comments: { type: Array, default: () => [] },
  // Шаг 5 v0.23: inbox-элементы задачи — источник «Оригинал письма»
  // (отправитель + текст) по comment.inbox_item_id. Пусто → секции нет.
  inboxItems: { type: Array, default: () => [] }
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

// Шаг 5 v0.23 (решение #1): legacy-комментарии — AI саммари письма под именем
// «user» (83 строки до фикса в verdicts.go, не мигрируем). Сигны: author
// «user» (+ direction in — все реальные user-комменты direction=out, проверено
// по БД). Показываем реального отправителя (из inboxItems по inbox_item_id)
// + бейдж «AI-саммари».
function isLegacyAIUser(comment) {
  return comment?.author === 'user'
}
function legacyAIAuthor(comment) {
  const s = senderOf(comment)
  return s || 'автор письма'
}

// «Оригинал письма»: источник по comment.inbox_item_id (используется legacy-автором).
function inboxItemOf(comment) {
  const id = comment?.inbox_item_id
  if (!id) return null
  return props.inboxItems.find((i) => i?.id === id) || null
}
// Подпись оригинала: реальный отправитель письма (не «user»).
function senderOf(comment) {
  const item = inboxItemOf(comment)
  if (!item) return ''
  const name = (item.from_name || '').trim()
  const addr = (item.from_contact || item.from_email || '').trim()
  if (name && addr) return `${name} (${addr})`
  return name || addr || ''
}
function subjectOf(comment) {
  const item = inboxItemOf(comment)
  return item?.subject ? `«${item.subject}»` : ''
}
// Detail письма показываем ТОЛЬКО в саммари-комментарии (author='user',
// есть inbox_item_id) — там, где смотрят письмо. AI-вердикт (ai_verdict)
// спойлера НЕ имеет (решение 14.09: там только саммари).
function showOriginal(comment) {
  return comment?.author === 'user' && Boolean(inboxItemOf(comment))
}
// AI-вердикт = только саммари: вложения в нём не показываем.
// «AI» = новый ai_verdict (author='ai') — legacy-саммари (author='user') НЕ в счёт:
// там вложения остаются (это комментарий с письмом, не чистый вердикт).
function isAiVerdictComment(comment) {
  return comment?.author === 'ai' || comment?.kind === 'ai_verdict'
}
function showAttachments(comment) {
  return !isAiVerdictComment(comment)
}
function originalBody(comment) {
  const item = inboxItemOf(comment)
  if (!item) return ''
  const body = item.body_text || item.body_html || ''
  if (!body) return ''
  // предыдущее письмо треда (по порядку inboxItems) — для diff
  const list = props.inboxItems
  const idx = list.indexOf(item)
  const prev = idx > 0 ? list[idx - 1]?.body_text || '' : ''
  return newMessagePartOf(body, prev)
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

// Копирование тела комментария (step 3 v0.23):
// 'text' = чистый текст без MD-символов (по умолчанию), 'md' = дословно.
const copyStates = ref({}) // { [comment.id]: 'text'|'md'|undefined }
const copyTimers = {}
function copyState(commentId) {
  return copyStates.value[commentId]
}
async function copyBody(comment, mode) {
  const body = comment?.body ?? ''
  if (!body.trim()) return
  const { ok } = await copyComment(body, mode)
  if (!ok) return
  copyStates.value = { ...copyStates.value, [comment.id]: mode }
  clearTimeout(copyTimers[comment.id])
  copyTimers[comment.id] = setTimeout(() => {
    copyStates.value = { ...copyStates.value, [comment.id]: undefined }
  }, 2000)
}

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
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.copy-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.2rem;
  margin-left: 0.35rem;
  padding: 0.15rem 0.45rem;
  border: 1px solid var(--mb-border);
  border-radius: 0.4rem;
  background: transparent;
  color: var(--mb-text-muted);
  font-size: 0.72rem;
  font-weight: 600;
  cursor: pointer;
}
.copy-btn:hover {
  border-color: var(--mb-primary);
  color: var(--mb-primary);
}
.copy-btn .pi {
  font-size: 0.8rem;
}
.copy-done {
  color: #1a7f37;
  font-weight: 600;
}

.comment-body {
  font-size: 1rem;
  white-space: pre-wrap;

  a {
    color: var(--mb-link, #2563eb);
    text-decoration: underline;
    word-break: break-all;
  }
}

/* Шаг 3b: MD-рендер (v-html, sanitize'нутый). Тег-специфичные отступы
   заменяют ручной pre-wrap; ссылки — те же стили, что в plain. */
.comment-body.md {
  white-space: normal;

  > :first-child { margin-top: 0; }
  > :last-child { margin-bottom: 0; }

  h1, h2, h3, h4, h5, h6 {
    font-weight: 700;
    line-height: 1.3;
    margin: 0.9em 0 0.4em;
    color: var(--mb-text);
  }
  h1 { font-size: 1.35rem; }
  h2 { font-size: 1.2rem; border-bottom: 1px solid var(--mb-border); padding-bottom: 0.25em; }
  h3 { font-size: 1.08rem; }
  h4, h5, h6 { font-size: 1rem; }

  p { margin: 0.45em 0; }

  ul, ol {
    margin: 0.45em 0;
    padding-left: 1.4em;
  }
  li { margin: 0.2em 0; }
  li > ul, li > ol { margin: 0.2em 0; }

  /* GFM task-lists: checkbox в li — прижаты к тексту */
  li > input[type='checkbox'] {
    margin: 0 0.4em 0 0;
    vertical-align: -0.1em;
    accent-color: var(--mb-primary);
  }
  li.task-list-item { list-style: none; margin-left: -1.2em; }

  code {
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 0.88em;
    background: var(--mb-surface-hover, rgba(0, 0, 0, 0.06));
    border-radius: 0.3em;
    padding: 0.1em 0.35em;
  }
  pre {
    background: var(--mb-surface-hover, rgba(0, 0, 0, 0.06));
    border: 1px solid var(--mb-border);
    border-radius: 0.5em;
    padding: 0.7em 0.9em;
    overflow-x: auto;
    margin: 0.5em 0;
  }
  pre code {
    background: transparent;
    padding: 0;
    font-size: 0.85em;
    white-space: pre;
    display: block;
  }

  blockquote {
    border-left: 3px solid var(--mb-border);
    margin: 0.5em 0;
    padding: 0.1em 0 0.1em 0.9em;
    color: var(--mb-text-muted);
  }

  hr {
    border: none;
    border-top: 1px solid var(--mb-border);
    margin: 1em 0;
  }

  table {
    border-collapse: collapse;
    margin: 0.5em 0;
    max-width: 100%;
    display: block;
    overflow-x: auto;
  }
  th, td {
    border: 1px solid var(--mb-border);
    padding: 0.3em 0.6em;
    text-align: left;
    vertical-align: top;
  }
  th { background: var(--mb-surface-hover, rgba(0,0,0,0.06)); font-weight: 600; }

  a {
    color: var(--mb-link, #2563eb);
    text-decoration: underline;
    word-break: break-all;
  }

  img {
    max-width: 100%;
    height: auto;
    border-radius: 0.4em;
  }
}
.comment-original {
  margin-top: 0.5rem;
  border: 1px solid var(--mb-border, rgba(0,0,0,0.12));
  border-radius: 0.4em;
}
.comment-original summary {
  cursor: pointer;
  padding: 0.4rem 0.6rem;
  font-size: 0.9rem;
  color: var(--mb-muted, #555);
}
.comment-original-meta {
  color: var(--mb-muted, #777);
}
.comment-original-body {
  padding: 0.4rem 0.6rem;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 0.95rem;
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
