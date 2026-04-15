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

        <el-tab-pane v-if="showFunctionPermissionRequestTab" name="permission">
          <template #label>
            <el-badge
              :value="currentFunction?.pending_count || 0"
              :hidden="!currentFunction?.pending_count || currentFunction.pending_count === 0"
              :max="99"
            >
              <span>权限</span>
            </el-badge>
          </template>
          <div class="tab-content">
            <el-tabs
              :model-value="permissionTab"
              class="permission-detail-tabs"
              @update:model-value="$emit('update:permissionTab', $event)"
              @tab-change="onFunctionPermissionTabChange"
            >
              <el-tab-pane name="request">
                <template #label>
                  <el-badge
                    :value="currentFunction?.pending_count || 0"
                    :hidden="!currentFunction?.pending_count || currentFunction.pending_count === 0"
                    :max="99"
                  >
                    <span>审批流</span>
                  </el-badge>
                </template>
                <div class="permission-tab-panel">
                  <PermissionRequestList
                    :ref="functionPermissionRequestListRef || undefined"
                    :resource-path="currentFunction?.full_code_path"
                    resource-type="function"
                    :template-type="currentFunctionDetail?.template_type"
                    :auto-load="activeTab === 'permission' && permissionTab === 'request'"
                  />
                </div>
              </el-tab-pane>

              <el-tab-pane name="manage" label="权限管理">
                <div class="permission-tab-panel">
                  <PermissionManageList
                    :ref="functionPermissionManageListRef || undefined"
                    :resource-path="currentFunction?.full_code_path"
                    resource-type="function"
                    :template-type="currentFunctionDetail?.template_type"
                    :auto-load="activeTab === 'permission' && permissionTab === 'manage'"
                  />
                </div>
              </el-tab-pane>
            </el-tabs>
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
import PermissionRequestList from '@/shared/components/permission/PermissionRequestList.vue'
import PermissionManageList from '@/shared/components/permission/PermissionManageList.vue'
import type { FunctionDetail } from '@/architecture/domain/types'
import type { ServiceTree as ServiceTreeType } from '@/types'
import FormOperateLogSection from './FormOperateLogSection.vue'
import ScheduledTaskList from './ScheduledTaskList.vue'
import WorkspaceFunctionRenderer from './WorkspaceFunctionRenderer.vue'

type FunctionTabName = 'content' | 'permission' | 'operateLog' | 'scheduledTask'
type FunctionPermissionTabName = 'request' | 'manage'

defineProps<{
  activeTab: FunctionTabName
  permissionTab: FunctionPermissionTabName
  currentFunction: ServiceTreeType | null
  currentFunctionDetail: FunctionDetail | null
  hasPermissionError: boolean
  showFunctionPermissionRequestTab: boolean
  showFormOperateLogTab: boolean
  showScheduledTaskTab: boolean
  functionFormViewRef?: (instance: any | null) => void
  functionPermissionRequestListRef?: (instance: any | null) => void
  functionPermissionManageListRef?: (instance: any | null) => void
  formOperateLogSectionRef?: (instance: any | null) => void
  onFunctionTabChange: (tab: string) => void
  onFunctionPermissionTabChange: (tab: string) => void
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
}>()

defineEmits<{
  (e: 'update:activeTab', value: FunctionTabName): void
  (e: 'update:permissionTab', value: FunctionPermissionTabName): void
}>()
</script>

<style scoped lang="scss">
.function-tabs-wrapper {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0 16px 16px;
}

.function-tabs-shell {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0 16px 16px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 16px;
  background: var(--el-bg-color);
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.05);
}

.function-detail-tabs,
.permission-detail-tabs {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.function-detail-tabs :deep(.el-tabs__header) {
  margin: 14px 0 12px;
  flex-shrink: 0;
}

.permission-detail-tabs :deep(.el-tabs__header) {
  margin: 2px 0 14px;
  flex-shrink: 0;
}

.function-detail-tabs :deep(.el-tabs__nav-wrap::after),
.permission-detail-tabs :deep(.el-tabs__nav-wrap::after) {
  background-color: var(--el-border-color-extra-light);
}

.function-detail-tabs :deep(.el-tabs__item.is-active) {
  font-weight: 600;
}

.function-detail-tabs :deep(.el-tabs__content) {
  background: transparent;
}

.function-detail-tabs :deep(.el-tabs__item),
.permission-detail-tabs :deep(.el-tabs__item) {
  font-size: 14px;
}

.permission-detail-tabs :deep(.el-tabs__item) {
  font-size: 13px;
}

.function-detail-tabs :deep(.el-tabs__content),
.permission-detail-tabs :deep(.el-tabs__content) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.function-detail-tabs :deep(.el-tab-pane),
.permission-detail-tabs :deep(.el-tab-pane) {
  height: 100%;
}

.function-detail-tabs :deep(.el-badge),
.permission-detail-tabs :deep(.el-badge) {
  position: relative;
  display: inline-block;
}

.function-detail-tabs :deep(.el-badge__content),
.permission-detail-tabs :deep(.el-badge__content) {
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
