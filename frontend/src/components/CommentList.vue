<template>
    <div class="comment-list">
        <div v-if="comments.length === 0" class="empty">Нет комментариев</div>
        <div v-for="comment in comments" :key="comment.id" class="comment" :class="comment.direction">
            <div class="comment-header">
                <span class="author">{{ comment.author }}</span>
                <span class="date">{{ formatDate(comment.created_at) }}</span>
            </div>
            <div class="comment-body">{{ comment.body }}</div>
        </div>
    </div>
</template>

<script setup>
defineProps({
    comments: { type: Array, default: () => [] }
})

function formatDate(dateStr) {
    if (!dateStr) return ''
    const d = new Date(dateStr)
    return d.toLocaleDateString('ru-RU') + ' ' + d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.comment-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
}

.empty {
    color: var(--p-text-muted-color);
    text-align: center;
    padding: 2rem;
}

.comment {
    padding: 0.75rem;
    border-radius: 0.5rem;
}

.comment.in {
    background: var(--p-surface-100);
}

.comment.out {
    background: var(--p-primary-50);
    margin-left: 1rem;
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
    font-size: 0.8rem;
    color: var(--p-text-muted-color);
}

.comment-body {
    font-size: 1rem;
    white-space: pre-wrap;
}
</style>