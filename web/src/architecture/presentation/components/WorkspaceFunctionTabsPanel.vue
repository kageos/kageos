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
              :has-permission-error="hasPermissionError"
              :form-view-ref-target="functionFormViewRef"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="showPermissionTabs" name="permissionRequest">
          <template #label>
            <el-badge
              :value="currentFunction?.pending_count || 0"
              :hidden="!currentFunction?.pending_count || currentFunction.pending_count === 0"
              :max="99"
            >
              <span>权限审批</span>
            </el-badge>
          </template>
          <div class="tab-content">
            <div class="permission-tab-panel">
              <PermissionRequestList
                :ref="functionPermissionRequestListRef || undefined"
                :resource-path="currentFunction?.full_code_path"
                resource-type="function"
                :template-type="currentFunctionDetail?.template_type"
                :auto-load="activeTab === 'permissionRequest'"
              />
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="showPermissionTabs" name="permissionManage" label="授权记录">
          <div class="tab-content">
            <div class="permission-tab-panel">
              <PermissionManageList
                :ref="functionPermissionManageListRef || undefined"
                :resource-path="currentFunction?.full_code_path"
                resource-type="function"
                :template-type="currentFunctionDetail?.template_type"
                :auto-load="activeTab === 'permissionManage'"
              />
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="showFormOperateLogTab" name="operateLog" label="执行记录">
          <div class="tab-content">
            <FormOperateLogSection
              :ref="formOperateLogSectionRef || undefined"
              :full-code-path="currentFunction?.full_code_path || ''"
              :function-detail="currentFunctionDetail"
              :auto-load="activeTab === 'operateLog'"
              @apply-log="onApplyFormOperateLog"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="showScheduledTaskTab" name="scheduledTask" label="定时任务">
          <div class="tab-content">
            <ScheduledTaskList
              :resource-path="currentFunction?.full_code_path"
              :auto-load="activeTab === 'scheduledTask'"
              @total-change="onScheduledTaskTotalChange"
              @open-function-operate-log="onOpenFunctionOperateLog"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import PermissionRequestList from '@/shared/components/permission/PermissionRequestList.vue'
import PermissionManageList from '@/shared/components/permission/PermissionManageList.vue'
import type { FunctionDetail } from '@/architecture/domain/types'
import type { ServiceTree as ServiceTreeType } from '@/types'
import FormOperateLogSection from './FormOperateLogSection.vue'
import ScheduledTaskList from './ScheduledTaskList.vue'
import WorkspaceFunctionRenderer from './WorkspaceFunctionRenderer.vue'

type FunctionTabName = 'content' | 'permissionRequest' | 'permissionManage' | 'operateLog' | 'scheduledTask'

const props = withDefaults(defineProps<{
  activeTab: FunctionTabName
  currentFunction: ServiceTreeType | null
  currentFunctionDetail: FunctionDetail | null
  hasPermissionError: boolean
  showFunctionPermissionTabs?: boolean
  showFunctionPermissionRequestTab?: boolean
  showFormOperateLogTab?: boolean
  showScheduledTaskTab?: boolean
  permissionTab?: string
  functionFormViewRef?: (instance: any | null) => void
  functionPermissionRequestListRef?: (instance: any | null) => void
  functionPermissionManageListRef?: (instance: any | null) => void
  formOperateLogSectionRef?: (instance: any | null) => void
  onFunctionTabChange: (tab: string) => void
  onApplyFormOperateLog: (payload: {
    requestBody?: Record<string, any> | null
    responseBody?: Record<string, any> | null
    responseMetadata?: Record<string, any> | null
  }) => void
  onScheduledTaskTotalChange: (total: number) => void
  onOpenFunctionOperateLog: (filters?: {
    requestUser?: string
    traceId?: string
    keyword?: string
    status?: string
    source?: string
  }) => void
}>(), {
  showFunctionPermissionTabs: false,
  showFunctionPermissionRequestTab: false,
  showFormOperateLogTab: false,
  showScheduledTaskTab: false,
  permissionTab: undefined
})

const showPermissionTabs = computed(() => {
  return props.showFunctionPermissionTabs || props.showFunctionPermissionRequestTab
})

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

.permission-tab-panel {
  flex: 1;
  min-height: 0;
  overflow: auto;
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
