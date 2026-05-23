<template>
  <div class="admin-toast-wrap" aria-live="polite">
    <div v-for="toast in items" :key="toast.id" class="admin-toast" :class="toast.type">
      <span class="admin-toast-icon">✓</span>
      <span>{{ toast.message }}</span>
    </div>
  </div>
</template>

<script setup>
defineProps({
  items: {
    type: Array,
    default: () => []
  }
})
</script>
