<template>
  <div class="tab-bar" ref="root">
    <template v-for="tab in tabs" :key="tab.key">
      <!-- Простая вкладка (Лента / Проекты / Персоны). -->
      <button
        v-if="!tab.options"
        :class="{ active: activeTab === tab.key }"
        :data-testid="`tab-${tab.key}`"
        @click="$emit('select', tab.key)"
      >
        {{ tab.label }}
        <Badge v-if="tab.count > 0" :value="tab.count" severity="info" size="small" />
      </button>

      <!-- Вкладка-дропдаун (Bootstrap «Tabs with dropdowns»): кнопка открывает
           список пунктов; выбранный пункт ПОКАЗЫВАЕТСЯ НА КНОПКЕ; выбор
           переключает фильтр вкладки. v0.28/28e, решение владельца 2026-09-20. -->
      <div v-else class="tab-dropdown">
        <button
          :class="{ active: activeTab === tab.key, open: openKey === tab.key }"
          :data-testid="`tab-${tab.key}`"
          aria-haspopup="true"
          :aria-expanded="String(openKey === tab.key)"
          @click="toggle(tab.key)"
        >
          <span class="tab-dd-label">{{ labelOf(tab) }}</span>
          <svg class="tab-dd-caret" width="10" height="10" viewBox="0 0 10 10" aria-hidden="true">
            <path d="M1 3.5 L5 7.5 L9 3.5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
          <Badge v-if="tab.count > 0" :value="tab.count" severity="info" size="small" />
        </button>
        <ul v-show="openKey === tab.key" class="tab-dd-menu" role="menu">
          <li v-for="opt in tab.options" :key="opt.value">
            <button
              role="menuitemradio"
              :class="{ selected: tab.selected === opt.value }"
              :aria-checked="String(tab.selected === opt.value)"
              @click="choose(tab, opt)"
            >
              <span class="tab-dd-check" :class="{ on: tab.selected === opt.value }">✓</span>
              {{ opt.label }}
            </button>
          </li>
        </ul>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import Badge from 'primevue/badge'

defineProps({
  tabs: { type: Array, required: true },
  activeTab: { type: String, required: true }
})

const emit = defineEmits(['select', 'select-option'])

// Открытый дропдаун (key вкладки; null — все закрыты).
const openKey = ref(null)
const root = ref(null)
function toggle(key) {
  openKey.value = openKey.value === key ? null : key
}
function labelOf(tab) {
  const opt = tab.options.find((o) => o.value === tab.selected)
  return opt ? opt.label : tab.label
}
function choose(tab, opt) {
  openKey.value = null
  emit('select-option', tab.key, opt.value)
}

// Клик вне вкладки закрывает открытое меню (Bootstrap-поведение).
function onDocClick(e) {
  if (openKey.value && root.value && !root.value.contains(e.target)) {
    openKey.value = null
  }
}
onMounted(() => document.addEventListener('click', onDocClick))
onUnmounted(() => document.removeEventListener('click', onDocClick))
</script>

<style scoped>
.tab-bar {
  display: flex;
  gap: 0;
  border-bottom: 2px solid var(--mb-border);
  margin-bottom: 1rem;
  align-items: center;
}

.tab-bar > button,
.tab-dropdown > button {
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

.tab-dropdown {
  position: relative;
}

.tab-bar > button:hover,
.tab-dropdown > button:hover {
  color: var(--mb-text);
}

.tab-bar > button.active,
.tab-dropdown > button.active {
  color: var(--mb-primary);
  border-bottom-color: var(--mb-primary);
  font-weight: 600;
}

.tab-dd-caret {
  flex: none;
  opacity: 0.7;
}

.tab-dropdown > button.open .tab-dd-caret {
  transform: rotate(180deg);
}

.tab-dd-menu {
  position: absolute;
  top: calc(100% + 2px);
  left: 0;
  z-index: 40;
  min-width: 160px;
  margin: 0;
  padding: 0.35rem 0;
  list-style: none;
  background: var(--mb-surface, var(--card, #fff));
  border: 1px solid var(--mb-border);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.14);
}

.tab-dd-menu button {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  width: 100%;
  padding: 0.45rem 1rem;
  border: none;
  background: none;
  text-align: left;
  cursor: pointer;
  font-size: 0.9rem;
  color: var(--mb-text, inherit);
  white-space: nowrap;
}

.tab-dd-menu button:hover {
  background: rgba(127, 127, 127, 0.12);
}

.tab-dd-check {
  width: 1em;
  flex: none;
  opacity: 0;
}
.tab-dd-check.on {
  opacity: 1;
}
</style>
