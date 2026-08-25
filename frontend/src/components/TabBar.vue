<template>
  <div class="tab-bar">
    <button
      v-for="tab in tabs"
      :key="tab.key"
      :class="{ active: activeTab === tab.key }"
      @click="$emit('select', tab.key)"
    >
      {{ tab.label }}
      <Badge v-if="tab.count > 0" :value="tab.count" severity="info" size="small" />
    </button>
  </div>
</template>

<script setup>
import Badge from 'primevue/badge'

defineProps({
  tabs: { type: Array, required: true },
  activeTab: { type: String, required: true }
})

defineEmits(['select'])
</script>

<style scoped>
.tab-bar {
  display: flex;
  gap: 0;
  border-bottom: 2px solid var(--mb-border);
  margin-bottom: 1rem;
}

.tab-bar button {
  padding: 0.75rem 1.5rem;
  border: none;
  background: none;
  cursor: pointer;
  font-size: 0.95rem;
  color: var(--mb-text-muted);
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  transition: all 0.2s;
}

.tab-bar button:hover {
  color: var(--mb-text);
}

.tab-bar button.active {
  color: var(--mb-primary);
  border-bottom-color: var(--mb-primary);
  font-weight: 600;
}
</style>
