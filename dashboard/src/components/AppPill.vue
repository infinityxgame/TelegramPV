<script setup>
defineProps({
  count: {
    type: Number,
    default: 0
  }
})
</script>

<template>
  <span class="count-pill">
    {{ count }}
  </span>
</template>
