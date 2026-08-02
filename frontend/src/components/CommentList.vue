<template>
    <div class="comment-list">
        <div v-if="!comments.length" class="no-comments">Нет комментариев</div>
        <div v-for="c in comments" :key="c.id" class="comment" :class="{ 'comment-out': c.direction === 'out' }">
            <div class="comment-header">
                <span class="comment-author">{{ c.author }}</span>
                <span class="comment-time">{{ formatDate(c.created_at) }}</span>
                <Tag v-if="c.direction === 'out'" value="Ответ" severity="info" class="comment-tag" />
            </div>
            <div class="comment-body" v-html="escapeHtml(c.body)"></div>
        </div>
    </div>
</template>

<script setup>
import Tag from 'primevue/tag'

defineProps({
    comments: { type: Array, default: () => [] }
})

function formatDate(dateStr) {
    if (!dateStr) return ''
    const d = new Date(dateStr)
    return d.toLocaleDateString('ru-RU') + ' ' + d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

function escapeHtml(text) {
    return text?.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\n/g, '<br>') || ''
}
</script>

<style scoped>
.comment-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
}

.no-comments {
    color: var(--p-text-muted-color);
}

.comment {
    padding: 0.75rem;
    background: var(--p-surface-50);
    border-radius: 0.5rem;
    border-left: 3px solid var(--p-primary-color);
}

.comment-out {
    border-left-color: var(--p-green-500);
    background: var(--p-green-50);
}

.comment-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.25rem;
}

.comment-author {
    font-weight: 600;
}

.comment-time {
    font-size: 0.85rem;
    color: var(--p-text-muted-color);
}

.comment-tag {
    margin-left: auto;
}

.comment-body {
    white-space: pre-wrap;
}
</style>