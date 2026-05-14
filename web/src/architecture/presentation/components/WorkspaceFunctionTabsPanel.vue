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
              :function-detail="currentFunctionDetail"
              :auto-load="activeTab === 'scheduledTask'"
              @total-change="onScheduledTaskTotalChange"
              @open-function-operate-log="onOpenFunctionOperateLog"
              @apply-execution="onApplyFormOperateLog"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="showScheduledAgentTaskTab" name="scheduledAgentTask" label="定时会话">
          <div class="tab-content">
            <ScheduledAgentTaskList
              :resource-path="currentFunction?.full_code_path"
              :auto-load="activeTab === 'scheduledAgentTask'"
              @total-change="onScheduledAgentTaskTotalChange"
              @open-session="onOpenWorkspaceSession"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { FunctionDetail } from '@/architecture/domain/types'
import type { ServiceTree as ServiceTreeType } from '@/types'
import FormOperateLogSection from './FormOperateLogSection.vue'
import ScheduledAgentTaskList from './ScheduledAgentTaskList.vue'
import ScheduledTaskList from './ScheduledTaskList.vue'
import WorkspaceFunctionRenderer from './WorkspaceFunctionRenderer.vue'
import FunctionInfoPanel from './FunctionInfoPanel.vue'
import type { WorkspaceSessionItem } from '@/api/workspace'

type FunctionTabName = 'content' | 'detail' | 'operateLog' | 'scheduledTask' | 'scheduledAgentTask'

const props = withDefaults(defineProps<{
  activeTab: FunctionTabName
  currentFunction: ServiceTreeType | null
  currentFunctionDetail: FunctionDetail | null
  showFormOperateLogTab?: boolean
  showScheduledTaskTab?: boolean
  showScheduledAgentTaskTab?: boolean
  permissionTab?: string
  functionFormViewRef?: (instance: any | null) => void
  formOperateLogSectionRef?: (instance: any | null) => void
  onFunctionTabChange: (tab: string) => void
  onApplyFormOperateLog: (payload: {
    requestBody?: Record<string, any> | null
    responseBody?: Record<string, any> | null
    responseMetadata?: Record<string, any> | null
    replayContext?: {
      source: 'scheduled_task' | 'operate_log'
      title?: string
      taskId?: number
      executionId?: number
      traceId?: string
      executedAt?: string
    } | null
  }) => void
  onScheduledTaskTotalChange: (total: number) => void
  onScheduledAgentTaskTotalChange: (total: number) => void
  onOpenWorkspaceSession: (session: WorkspaceSessionItem) => void
  onOpenFunctionOperateLog: (filters?: {
    requestUser?: string
    traceId?: string
    keyword?: string
    status?: string
    source?: string
  }) => void
}>(), {
  showFormOperateLogTab: false,
  showScheduledTaskTab: false,
  showScheduledAgentTaskTab: false,
  permissionTab: undefined
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
