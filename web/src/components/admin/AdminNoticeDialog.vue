<template>
  <div v-if="open" class="modal-mask">
    <section class="confirm-panel" role="alertdialog" aria-modal="true" :aria-labelledby="titleId">
      <header class="confirm-header" :class="variant">
        <span class="confirm-icon">!</span>
        <div>
          <strong :id="titleId">{{ title }}</strong>
          <p v-if="subtitle">{{ subtitle }}</p>
        </div>
      </header>
      <p class="confirm-message">{{ message }}</p>
      <footer class="confirm-actions">
        <button type="button" @click="$emit('close')">{{ closeText }}</button>
      </footer>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: '操作提示' },
  subtitle: { type: String, default: '' },
  message: { type: String, default: '' },
  closeText: { type: String, default: '知道了' },
  variant: { type: String, default: 'warning' }
})

defineEmits(['close'])

const titleId = computed(() => `notice-title-${props.title.replace(/\s+/g, '-')}`)
</script>
