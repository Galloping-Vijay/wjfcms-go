<template>
  <div class="permission-node">
    <label class="permission-check">
      <input type="checkbox" :checked="checked" @change="emit('toggle', node, $event.target.checked)" />
      <span>{{ node.name }}</span>
    </label>
    <div v-if="node.children?.length" class="permission-children">
      <PermissionNode
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :selected="selected"
        @toggle="forwardToggle"
      />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  node: {
    type: Object,
    required: true
  },
  selected: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['toggle'])
const checked = computed(() => props.selected.includes(props.node.id))

function forwardToggle(node, checked) {
  emit('toggle', node, checked)
}
</script>
