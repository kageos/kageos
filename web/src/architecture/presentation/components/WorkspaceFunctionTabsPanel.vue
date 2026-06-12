<template>
  <div class="function-tabs-wrapper" data-testid="workspace-function-tabs">
    <div class="function-tabs-shell">
      <FunctionConnectorBar
        :current-function="currentFunction"
        :function-detail="currentFunctionDetail"
      />
      <el-tabs
        :model-value="activeTab"
        class="function-detail-tabs"
        @update:model-value="$emit('update:activeTab', $event)"
        @tab-change="onFunctionTabChange"
      >
        <el-tab-pane name="content">
          <template #label>
            <span>{{ t('functionTabs.content') }}</span>
          </template>
          <div class="tab-content">
            <WorkspaceFunctionRenderer
              :current-function="currentFunction"
              :function-detail="currentFunctionDetail"
              :form-view-ref-target="functionFormViewRef"
              :show-connector-bar="false"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane name="permission" :label="t('functionTabs.permission')">
          <div class="tab-content">
            <TeamAccessPanel
              ref="accessPanelRef"
              :node="currentFunction"
              embedded
              @changed="$emit('accessChanged')"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="isFormFunction" name="publicShare" label="公开链接">
          <div class="tab-content">
            <PublicSharePanel
              :function-detail="currentFunctionDetail"
              :function-node="currentFunction"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane
          v-if="featureFlags.operateLogs"
          name="operateLog"
          :label="t('functionTabs.operateLog')"
        >
          <div class="tab-content">
            <OperateLogSection
              ref="operateLogSectionRef"
              :full-code-path="currentFunction?.full_code_path || currentFunctionDetail?.full_code_path || ''"
              :row-id="0"
              :function-detail="currentFunctionDetail"
              scope="function"
              embedded
              show-refresh
              :title="t('functionTabs.functionOperateLog')"
              :auto-load="false"
              :on-apply-form-log="handleApplyFormLog"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane
          v-if="featureFlags.scheduledTasks"
          name="scheduledTask"
          label="定时函数"
        >
          <div class="tab-content">
            <ScheduledTaskList
              :resource-path="currentFunction?.full_code_path || currentFunctionDetail?.full_code_path || ''"
              :function-detail="currentFunctionDetail"
              :auto-load="activeTab === 'scheduledTask'"
              :focus-task-id="scheduledFocusTaskID"
              :focus-execution-id="scheduledFocusExecutionID"
            />
          </div>
        </el-tab-pane>

      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import type { FunctionDetail } from '@/architecture/domain/types'
import type { ServiceTree as ServiceTreeType } from '@/architecture/domain/types'
import WorkspaceFunctionRenderer from './WorkspaceFunctionRenderer.vue'
import FunctionConnectorBar from './FunctionConnectorBar.vue'
import OperateLogSection from './OperateLogSection.vue'
import TeamAccessPanel from './TeamAccessPanel.vue'
import PublicSharePanel from './PublicSharePanel.vue'
import ScheduledTaskList from './ScheduledTaskList.vue'
import { featureFlags } from '@/architecture/shared/config/features'
import {
  isScheduledPanelQuery,
  PLATFORM_PANEL_QUERY_KEY,
  PLATFORM_SCHEDULED_EXECUTION_ID_QUERY_KEY,
  PLATFORM_SCHEDULED_TASK_ID_QUERY_KEY,
  readStringQuery,
} from '@/architecture/shared/routing/platformRouteParams'
import { ElMessage } from 'element-plus'

type FunctionTabName = 'content' | 'permission' | 'publicShare' | 'operateLog' | 'scheduledTask'

interface LoadableOperateLogSection {
  load: () => void
}

interface LoadableAccessPanel {
  loadMembers: () => void
}

interface ReplayableFormView {
  applyOperateLog?: (requestBody: Record<string, any>, responseBody?: Record<string, any> | null) => void
}

const props = withDefaults(defineProps<{
  activeTab: FunctionTabName
  currentFunction: ServiceTreeType | null
  currentFunctionDetail: FunctionDetail | null
  functionFormViewRef?: (instance: any | null) => void
  currentFormView?: ReplayableFormView | null
  onFunctionTabChange: (tab: string) => void
}>(), {})

const { t } = useI18n()
const route = useRoute()

const emit = defineEmits<{
  (e: 'update:activeTab', value: FunctionTabName): void
  (e: 'accessChanged'): void
}>()

const operateLogSectionRef = ref<LoadableOperateLogSection | null>(null)
const accessPanelRef = ref<LoadableAccessPanel | null>(null)
const isFormFunction = computed(() => props.currentFunctionDetail?.template_type === 'form' || props.currentFunction?.template_type === 'form')
const scheduledFocusTaskID = computed(() => readStringQuery(route.query, PLATFORM_SCHEDULED_TASK_ID_QUERY_KEY))
const scheduledFocusExecutionID = computed(() => readStringQuery(route.query, PLATFORM_SCHEDULED_EXECUTION_ID_QUERY_KEY))

function loadOperateLogTab(tabName: FunctionTabName) {
  if (tabName === 'operateLog' && featureFlags.operateLogs) {
    nextTick(() => operateLogSectionRef.value?.load())
  }
  if (tabName === 'permission') {
    nextTick(() => accessPanelRef.value?.loadMembers())
  }
}

async function waitForFormView(): Promise<ReplayableFormView | null> {
  for (let i = 0; i < 10; i += 1) {
    if (props.currentFormView?.applyOperateLog) {
      return props.currentFormView
    }
    await nextTick()
    await new Promise((resolve) => window.setTimeout(resolve, 16))
  }
  return null
}

async function handleApplyFormLog(requestBody: Record<string, any>, responseBody: Record<string, any> | null) {
  emit('update:activeTab', 'content')
  props.onFunctionTabChange('content')
  const formView = await waitForFormView()
  if (!formView?.applyOperateLog) {
    ElMessage.warning(t('operateLog.formReplayTargetMissing'))
    return
  }
  formView.applyOperateLog(requestBody, responseBody)
}

watch(
  () => props.activeTab,
  (tabName) => loadOperateLogTab(tabName),
  { immediate: true }
)

watch(
  () => [
    props.currentFunction?.full_code_path,
    props.currentFunctionDetail?.full_code_path,
  ],
  () => {
    if (props.activeTab === 'operateLog') {
      loadOperateLogTab('operateLog')
    }
  }
)

watch(
  () => [
    route.query._open,
    route.query._scheduled,
    route.query._scheduled_kind,
    route.query[PLATFORM_PANEL_QUERY_KEY],
    route.query[PLATFORM_SCHEDULED_TASK_ID_QUERY_KEY],
    props.currentFunction?.full_code_path,
    props.currentFunctionDetail?.full_code_path,
  ],
  () => {
    if (isScheduledPanelQuery(route.query, 'function') && scheduledFocusTaskID.value && featureFlags.scheduledTasks) {
      emit('update:activeTab', 'scheduledTask')
      props.onFunctionTabChange('scheduledTask')
    }
  },
  { immediate: true }
)
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
