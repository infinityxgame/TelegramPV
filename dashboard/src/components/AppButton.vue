<script setup>
import { computed } from 'vue'

const props = defineProps({
  variant: {
    type: String,
    default: 'primary'
  },
  size: {
    type: String,
    default: 'md'
  },
  disabled: {
    type: Boolean,
    default: false
  },
  loading: {
    type: Boolean,
    default: false
  }
})

const classes = computed(() => [
  'primary-button',
  props.variant !== 'primary' ? `btn-${props.variant}` : '',
  props.size !== 'md' ? `btn-${props.size}` : ''
])
</script>

<template>
  <button :class="classes" :disabled="disabled || loading">
    <slot />
  </button>
</template>
