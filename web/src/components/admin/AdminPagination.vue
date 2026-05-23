<template>
  <div v-if="total > 0" class="admin-pagination">
    <div class="admin-pagination-summary">
      共 {{ total }} 条
      <span>第 {{ currentPage }} / {{ totalPages }} 页</span>
    </div>
    <div class="admin-pagination-controls">
      <button type="button" :disabled="currentPage <= 1" @click="changePage(1)">首页</button>
      <button type="button" :disabled="currentPage <= 1" @click="changePage(currentPage - 1)">上一页</button>
      <button
        v-for="item in pageItems"
        :key="item"
        type="button"
        :class="{ active: item === currentPage }"
        @click="changePage(item)"
      >
        {{ item }}
      </button>
      <button type="button" :disabled="currentPage >= totalPages" @click="changePage(currentPage + 1)">下一页</button>
      <button type="button" :disabled="currentPage >= totalPages" @click="changePage(totalPages)">尾页</button>
      <select :value="pageSize" @change="changePageSize">
        <option v-for="size in pageSizes" :key="size" :value="size">{{ size }} 条/页</option>
      </select>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  page: { type: Number, default: 1 },
  pageSize: { type: Number, default: 15 },
  total: { type: Number, default: 0 },
  pageSizes: { type: Array, default: () => [10, 15, 20, 30, 50] }
})

const emit = defineEmits(['change', 'update:pageSize'])

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const currentPage = computed(() => Math.min(Math.max(1, props.page), totalPages.value))
const pageItems = computed(() => {
  const max = 5
  let start = Math.max(1, currentPage.value - 2)
  let end = Math.min(totalPages.value, start + max - 1)
  start = Math.max(1, end - max + 1)
  return Array.from({ length: end - start + 1 }, (_, index) => start + index)
})

function changePage(page) {
  const next = Math.min(Math.max(1, Number(page)), totalPages.value)
  if (next === currentPage.value) return
  emit('change', next)
}

function changePageSize(event) {
  const nextSize = Number(event.target.value)
  if (!Number.isFinite(nextSize) || nextSize <= 0) return
  emit('update:pageSize', nextSize)
  emit('change', 1)
}
</script>
