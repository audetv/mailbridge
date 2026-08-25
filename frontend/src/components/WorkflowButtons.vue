<template>
  <div class="workflow-buttons">
    <Button
      v-for="action in availableActions"
      :key="action.status"
      :label="action.label"
      :severity="action.severity"
      :icon="action.icon"
      size="small"
      @click="$emit('transition', action.status)"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import Button from 'primevue/button'

const props = defineProps({
  currentStatus: { type: String, required: true }
})

defineEmits(['transition'])

const transitions = {
  new: [
    { status: 'in_progress', label: 'В работу', severity: 'primary', icon: 'pi pi-play' },
    { status: 'backlog', label: 'В бэклог', severity: 'secondary', icon: 'pi pi-inbox' },
    { status: 'closed', label: 'Закрыть', severity: 'danger', icon: 'pi pi-times' }
  ],
  in_progress: [
    { status: 'backlog', label: 'В бэклог', severity: 'secondary', icon: 'pi pi-inbox' },
    { status: 'completed', label: 'Выполнено', severity: 'success', icon: 'pi pi-check' }
  ],
  backlog: [
    { status: 'in_progress', label: 'В работу', severity: 'primary', icon: 'pi pi-play' },
    { status: 'closed', label: 'Закрыть', severity: 'danger', icon: 'pi pi-times' }
  ],
  completed: [
    { status: 'closed', label: 'Закрыть', severity: 'danger', icon: 'pi pi-times' },
    { status: 'in_progress', label: 'Вернуть в работу', severity: 'warn', icon: 'pi pi-replay' }
  ],
  closed: [{ status: 'in_progress', label: 'Вернуть в работу', severity: 'warn', icon: 'pi pi-replay' }]
}

const availableActions = computed(() => transitions[props.currentStatus] || [])
</script>

<style scoped>
.workflow-buttons {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  margin-bottom: 1rem;
}
</style>
