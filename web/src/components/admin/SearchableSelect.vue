<template>
  <div ref="root" class="searchable-select" :class="{ open }">
    <button type="button" class="searchable-select-trigger" @click="toggle">
      <span class="searchable-select-label">{{ selectedLabel }}</span>
      <span class="searchable-select-actions">
        <span v-if="hasValue" class="searchable-select-clear" title="清空" @click.stop="clear">×</span>
        <span class="searchable-select-arrow" aria-hidden="true"></span>
      </span>
    </button>
    <div v-if="open" class="searchable-select-panel">
      <input v-model.trim="keyword" class="searchable-select-input" placeholder="搜索选项" @keydown.stop />
      <button
        v-for="option in filteredOptions"
        :key="String(option.value)"
        type="button"
        class="searchable-select-option"
        :class="{ selected: isSelected(option.value) }"
        @click="select(option.value)"
      >
        {{ option.label }}
      </button>
      <p v-if="filteredOptions.length === 0" class="searchable-select-empty">没有匹配项</p>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const props = defineProps({
  modelValue: {
    type: [String, Number, Boolean],
    default: ''
  },
  options: {
    type: Array,
    default: () => []
  },
  placeholder: {
    type: String,
    default: '请选择'
  }
})

const emit = defineEmits(['update:modelValue'])
const root = ref(null)
const open = ref(false)
const keyword = ref('')

const hasValue = computed(() => props.modelValue !== '' && props.modelValue !== undefined && props.modelValue !== null)
const selectedOption = computed(() => props.options.find((option) => sameValue(option.value, props.modelValue)))
const selectedLabel = computed(() => selectedOption.value?.label || props.placeholder)
const filteredOptions = computed(() => {
  if (!keyword.value) return props.options
  const key = keyword.value.toLowerCase()
  return props.options.filter((option) => String(option.label || '').toLowerCase().includes(key))
})

function toggle() {
  open.value = !open.value
  if (open.value) keyword.value = ''
}

function select(value) {
  emit('update:modelValue', value)
  open.value = false
}

function clear() {
  emit('update:modelValue', '')
  keyword.value = ''
  open.value = false
}

function isSelected(value) {
  return sameValue(value, props.modelValue)
}

function sameValue(a, b) {
  return String(a) === String(b)
}

function onDocumentClick(event) {
  if (!root.value?.contains(event.target)) {
    open.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', onDocumentClick)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', onDocumentClick)
})
</script>
