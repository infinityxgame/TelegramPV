<script setup>
import { computed, ref } from 'vue'
import { ChevronRight } from '../icons'

const props = defineProps({
  title: { type: String, required: true },
  description: { type: String, default: '' },
  modelValue: { type: Boolean, default: undefined }
})

const emit = defineEmits(['update:modelValue', 'update:model-value'])

const isOpenLocal = ref(false)

const isOpen = computed(() => {
  if (props.modelValue !== undefined) {
    return props.modelValue
  }
  return isOpenLocal.value
})

const toggle = () => {
  const nextState = !isOpen.value
  isOpenLocal.value = nextState
  emit('update:modelValue', nextState)
  emit('update:model-value', nextState)
}
</script>

<template>
  <section class="accordion-block" :class="{ open: isOpen }">
    <button
      type="button"
      class="accordion-header"
      :aria-expanded="isOpen"
      @click="toggle"
    >
      <span class="accordion-icon">
        <slot name="icon"></slot>
      </span>

      <span class="accordion-title">
        <p>
          <strong>{{ title }}</strong>
        </p>
        <p v-if="description">
          <small>{{ description }}</small>
        </p>
      </span>

      <ChevronRight class="accordion-chevron" :size="16" />
    </button>

    <div v-show="isOpen" class="accordion-body">
      <slot></slot>
    </div>
  </section>
</template>

<style scoped>
.accordion-title > * {
  margin: 0;
}

.accordion-title strong {
  font:
    600 13px 'Space Grotesk',
    sans-serif;
  color: #f1f7ff;
}

.accordion-title small {
  font-size: 11px;
  color: var(--user-text-dim, #728ba2);
}

.accordion-block {
  border: 1px solid var(--user-border, #1b344b);
  border-radius: 14px;
  background: var(--user-bg-base, #07111f);
  overflow: hidden;
  transition: border-color 0.2s;
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
}

.accordion-block + .accordion-block {
  margin-top: 12px;
}

.accordion-block.open {
  border-color: var(--user-border-light, #234765);
}

.accordion-header {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 14px 16px;
  background: transparent;
  border: 0;
  cursor: pointer;
  text-align: left;
  color: #dbe7f5;
  transition: background 0.2s;
  box-sizing: border-box;
}

.accordion-header:hover {
  background: var(--user-surface-light, #0e2032);
}

.accordion-header:focus-visible {
  outline: 2px solid var(--user-accent, #38a7ff);
  outline-offset: -2px;
}

.accordion-icon {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border-radius: 9px;
  background: var(--user-icon-bg, #11385b);
  color: var(--user-primary, #38a7ff);
  flex: none;
}

.accordion-chevron {
  color: var(--user-text-dim, #728ba2);
  flex: none;
  transition:
    transform 0.2s ease,
    color 0.2s;
  margin-left: auto;
}

.accordion-block.open .accordion-chevron {
  transform: rotate(90deg);
  color: var(--user-accent, #38a7ff);
}

.accordion-body {
  padding: 16px 16px 20px;
  border-top: 1px solid var(--user-border, #1b344b);
  animation: accordionOpen 0.18s ease both;
  box-sizing: border-box;
  width: 100%;
  min-width: 0;
  overflow: hidden;
}

.accordion-body :deep(.settings-columns) {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(260px, 100%), 1fr));
  gap: 34px;
  align-items: start;
  min-width: 0;
}

@keyframes accordionOpen {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

@media (max-width: 720px) {
  .accordion-header {
    padding: 12px;
    gap: 10px;
  }
  .accordion-body {
    padding: 14px 12px 16px;
  }
  .accordion-body :deep(.settings-columns) {
    gap: 20px;
  }
}
</style>
