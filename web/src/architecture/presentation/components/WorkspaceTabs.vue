<!--
  WorkspaceTabs - 工作空间 Tab 标签页组件
  
  职责：
  - Tab 列表展示
  - Tab 点击处理
  - Tab 编辑处理（添加/删除）
-->

<template>
  <div v-if="tabs.length > 0" class="workspace-tabs-container">
    <el-tabs
      v-model="activeTabId"
      type="card"
      editable
      class="workspace-tabs"
      @tab-click="handleTabClick"
      @edit="handleTabsEdit"
    >
      <el-tab-pane
        v-for="tab in tabs"
        :key="tab.id"
        :label="tab.title"
        :name="tab.id"
        :closable="tabs.length > 1"
      />
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WorkspaceTab } from '../../domain/services/WorkspaceDomainService'

interface Props {
  tabs: WorkspaceTab[]
  activeTabId: string
}

interface Emits {
  (e: 'update:activeTabId', value: string): void
  (e: 'tab-click', tab: any): void
  (e: 'tab-edit', targetName: string | undefined, action: 'remove' | 'add'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const activeTabId = computed({
  get: () => props.activeTabId,
  set: (val) => emit('update:activeTabId', val)
})

const handleTabClick = (tab: any) => {
  emit('tab-click', tab)
}

const handleTabsEdit = (targetName: string | undefined, action: 'remove' | 'add') => {
  emit('tab-edit', targetName, action)
}
</script>

<style scoped lang="scss">
.workspace-tabs-container {
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
  padding: 0 16px;
}

.workspace-tabs {
  :deep(.el-tabs__header) {
    margin: 0;
    border-bottom: none;
  }

  :deep(.el-tabs__nav-wrap) {
    &::after {
      display: none;
    }
  }

  :deep(.el-tabs__item) {
    height: 40px;
    line-height: 40px;
    padding: 0 16px;
    border: 1px solid var(--el-border-color-lighter);
    border-bottom: none;
    border-radius: 4px 4px 0 0;
    margin-right: 4px;
    background: var(--el-fill-color-lighter);
    transition: all 0.2s;

    &:hover {
      background: var(--el-fill-color-light);
    }

    &.is-active {
      background: var(--el-bg-color);
      border-color: var(--el-border-color);
      border-bottom-color: var(--el-bg-color);
      z-index: 1;
    }
  }
}
</style>


