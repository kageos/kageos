<template>
  <div class="knowledge-tree">
    <div
      v-for="node in data"
      :key="node.id"
      class="tree-node-wrapper"
    >
      <KnowledgeTreeNode
        :node="node"
        :level="0"
        :selected-id="selectedId"
        @select="handleSelect"
        @toggle="handleToggle"
        @action="handleAction"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import KnowledgeTreeNode from './KnowledgeTreeNode.vue'

interface TreeNodeData {
  id: string | number
  label: string
  title?: string
  icon?: string
  type?: string
  children?: TreeNodeData[]
  expanded?: boolean
  [key: string]: any
}

interface Props {
  data: TreeNodeData[]
  modelValue?: string | number | null
}

interface Emits {
  (e: 'update:modelValue', value: string | number | null): void
  (e: 'node-click', node: TreeNodeData): void
  (e: 'node-action', action: string, node: TreeNodeData): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const selectedId = ref<string | number | null>(props.modelValue || null)

watch(() => props.modelValue, (newVal) => {
  selectedId.value = newVal
})

const handleSelect = (node: TreeNodeData) => {
  selectedId.value = node.id
  emit('update:modelValue', node.id)
  emit('node-click', node)
}

const handleToggle = (node: TreeNodeData) => {
  node.expanded = !node.expanded
}

const handleAction = (action: string, node: TreeNodeData) => {
  emit('node-action', action, node)
}
</script>

<style lang="scss" scoped>
.knowledge-tree {
  width: 100%;
}

.tree-node-wrapper {
  margin-bottom: 2px;
}
</style>

