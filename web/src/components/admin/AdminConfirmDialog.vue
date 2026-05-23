<template>
  <div v-if="open" class="modal-mask">
    <section class="confirm-panel" role="dialog" aria-modal="true" :aria-labelledby="titleId">
      <header class="confirm-header" :class="variant">
        <span class="confirm-icon">!</span>
        <div>
          <strong :id="titleId">{{ title }}</strong>
          <p v-if="subtitle">{{ subtitle }}</p>
        </div>
      </header>
      <p class="confirm-message">{{ message }}</p>
      <footer class="confirm-actions">
        <button type="button" class="secondary-button" :disabled="loading" @click="$emit('cancel')">{{ cancelText }}</button>
        <button type="button" :class="confirmButtonClass" :disabled="loading" @click="$emit('confirm')">
          {{ loading ? loadingText : confirmText }}
        </button>
      </footer>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: '确认操作' },
  subtitle: { type: String, default: '' },
  message: { type: String, default: '' },
  confirmText: { type: String, default: '确认' },
  cancelText: { type: String, default: '取消' },
  loadingText: { type: String, default: '处理中...' },
  loading: { type: Boolean, default: false },
  variant: { type: String, default: 'danger' }
})

defineEmits(['cancel', 'confirm'])

const titleId = computed(() => `confirm-title-${props.title.replace(/\s+/g, '-')}`)
const confirmButtonClass = computed(() => (props.variant === 'danger' ? 'danger-button' : ''))
</script>
