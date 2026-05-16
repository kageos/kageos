<template>
  <div class="function-tabs-wrapper" data-testid="workspace-function-tabs">
    <div class="function-tabs-shell">
      <el-tabs
        :model-value="activeTab"
        class="function-detail-tabs"
        @update:model-value="$emit('update:activeTab', $event)"
        @tab-change="onFunctionTabChange"
      >
        <el-tab-pane name="content">
          <template #label>
            <span>函数内容</span>
          </template>
          <div class="tab-content">
            <WorkspaceFunctionRenderer
              :current-function="currentFunction"
              :function-detail="currentFunctionDetail"
              :form-view-ref-target="functionFormViewRef"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane name="detail" label="详情">
          <div class="tab-content">
            <FunctionInfoPanel
              :function-data="currentFunctionDetail"
              :function-node="currentFunction"
            />
          </div>
        </el-tab-pane>

      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { FunctionDetail } from '@/architecture/domain/types'
import type { ServiceTree as ServiceTreeType } from '@/architecture/domain/types'
import WorkspaceFunctionRenderer from './WorkspaceFunctionRenderer.vue'
import FunctionInfoPanel from './FunctionInfoPanel.vue'

type FunctionTabName = 'content' | 'detail'

const props = withDefaults(defineProps<{
  activeTab: FunctionTabName
  currentFunction: ServiceTreeType | null
  currentFunctionDetail: FunctionDetail | null
  functionFormViewRef?: (instance: any | null) => void
  onFunctionTabChange: (tab: string) => void
}>(), {})

defineEmits<{
  (e: 'update:activeTab', value: FunctionTabName): void
}>()
</script>

<style scoped lang="scss">
.function-tabs-wrapper {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 14px 14px 16px;
}

.function-tabs-shell {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0 14px 14px;
  border: none;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  backdrop-filter: none;
}

.function-detail-tabs {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.function-detail-tabs :deep(.el-tabs__header) {
  margin: 12px 0 10px;
  flex-shrink: 0;
}

.function-detail-tabs :deep(.el-tabs__nav-wrap::after) {
  background: rgba(148, 163, 184, 0.18);
}

.function-detail-tabs :deep(.el-tabs__item) {
  font-size: 14px;
  min-height: 38px;
  padding: 0 6px 2px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}

.function-detail-tabs :deep(.el-tabs__item + .el-tabs__item) {
  margin-left: 22px;
}

.function-detail-tabs :deep(.el-tabs__item.is-active) {
  font-weight: 600;
  background: transparent;
  border: none;
  box-shadow: none;
  color: var(--el-color-primary);
}

.function-detail-tabs :deep(.el-tabs__active-bar) {
  height: 2px;
  border-radius: 999px;
}

.function-detail-tabs :deep(.el-tabs__content) {
  background: transparent;
}

.function-detail-tabs :deep(.el-tabs__content) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.function-detail-tabs :deep(.el-tab-pane) {
  height: 100%;
}

.function-detail-tabs :deep(.el-badge) {
  position: relative;
  display: inline-block;
}

.function-detail-tabs :deep(.el-badge__content) {
  font-size: 11px;
  height: 16px;
  line-height: 16px;
  min-width: 16px;
  padding: 0 5px;
  border-radius: 8px;
}

.tab-content {
  flex: 1;
  overflow-y: auto !important;
  overflow-x: hidden;
  min-height: 0;
  height: 100%;
  -webkit-overflow-scrolling: touch;
}
</style>
