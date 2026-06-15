<template>
  <div class="detail-content">
    <div v-if="packageNode">
      <el-tabs v-model="activeTab" class="detail-tabs">
        <el-tab-pane :label="t('packageDetail.permission')" name="permission">
          <div class="tab-content access-tab-content">
            <TeamAccessPanel
              ref="accessPanelRef"
              :node="packageNode"
              embedded
              @changed="$emit('access-changed')"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane
          v-if="featureFlags.operateLogs"
          :label="t('packageDetail.operateLog')"
          name="operateLog"
        >
          <div class="tab-content operate-log-tab-content">
            <OperateLogSection
              ref="operateLogSectionRef"
              :full-code-path="packageNode.full_code_path || ''"
              :row-id="0"
              :function-detail="null"
              scope="directory"
              embedded
              show-refresh
              :title="t('packageDetail.directoryOperateLog')"
              :auto-load="activeTab === 'operateLog'"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane
          v-if="featureFlags.scheduledTasks"
          :label="t('packageDetail.scheduledAgentTask')"
          name="scheduledAgentTask"
        >
          <div class="tab-content scheduled-agent-tab-content">
            <ScheduledAgentTaskList
              :resource-path="packageNode.full_code_path || ''"
              :auto-load="activeTab === 'scheduledAgentTask'"
              :focus-task-id="scheduledFocusTaskID"
              :focus-execution-id="scheduledFocusExecutionID"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane :label="t('packageDetail.detail')" name="detail">
          <div class="tab-content directory-detail-tab-content">
            <div
              v-if="directoryMarkdown"
              class="directory-markdown-body"
              v-html="renderMarkdown(directoryMarkdown)"
            />
            <PackageDetailChildrenGrid
              :children="packageNode.children || []"
              @select-child="$emit('select-child', $event)"
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
import type { ServiceTree } from '@/architecture/domain/types'
import PackageDetailChildrenGrid from './PackageDetailChildrenGrid.vue'
import OperateLogSection from './OperateLogSection.vue'
import ScheduledAgentTaskList from './ScheduledAgentTaskList.vue'
import TeamAccessPanel from './TeamAccessPanel.vue'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'
import { featureFlags } from '@/architecture/shared/config/features'
import {
  isOperateLogPanelQuery,
  isScheduledPanelQuery,
  PLATFORM_PANEL_QUERY_KEY,
  PLATFORM_SCHEDULED_EXECUTION_ID_QUERY_KEY,
  PLATFORM_SCHEDULED_TASK_ID_QUERY_KEY,
  readStringQuery,
} from '@/architecture/shared/routing/platformRouteParams'

type PackageTabName = 'permission' | 'operateLog' | 'scheduledAgentTask' | 'detail'

interface LoadableOperateLogSection {
  load: () => void
}

interface LoadableAccessPanel {
  loadMembers: () => void
}

const props = defineProps<{
  packageNode: ServiceTree | null
}>()

defineEmits<{
  (e: 'select-child', child: ServiceTree): void
  (e: 'access-changed'): void
}>()

const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()
const { t } = useI18n()
const route = useRoute()

const activeTab = ref<PackageTabName>(featureFlags.operateLogs ? 'operateLog' : 'detail')
const operateLogSectionRef = ref<LoadableOperateLogSection | null>(null)
const accessPanelRef = ref<LoadableAccessPanel | null>(null)

const directoryMarkdown = computed(() => {
  return props.packageNode?.description?.trim() || ''
})
const scheduledFocusTaskID = computed(() => readStringQuery(route.query, PLATFORM_SCHEDULED_TASK_ID_QUERY_KEY))
const scheduledFocusExecutionID = computed(() => readStringQuery(route.query, PLATFORM_SCHEDULED_EXECUTION_ID_QUERY_KEY))

function loadOperateLogTab(tabName: PackageTabName) {
  if (tabName === 'operateLog' && featureFlags.operateLogs) {
    nextTick(() => operateLogSectionRef.value?.load())
  }
  if (tabName === 'permission') {
    nextTick(() => accessPanelRef.value?.loadMembers())
  }
}

watch(
  activeTab,
  (tabName) => loadOperateLogTab(tabName),
  { immediate: true }
)

watch(
  () => props.packageNode?.full_code_path,
  () => {
    if (activeTab.value === 'operateLog') {
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
    props.packageNode?.full_code_path,
  ],
  () => {
    if (isScheduledPanelQuery(route.query, 'agent') && scheduledFocusTaskID.value && featureFlags.scheduledTasks) {
      activeTab.value = 'scheduledAgentTask'
    }
  },
  { immediate: true }
)

watch(
  () => [route.query._open, route.query[PLATFORM_PANEL_QUERY_KEY], props.packageNode?.full_code_path],
  () => {
    if (isOperateLogPanelQuery(route.query) && featureFlags.operateLogs) {
      activeTab.value = 'operateLog'
    }
  },
  { immediate: true }
)

</script>

<style scoped lang="scss">
.detail-content {
  flex: 1;
  overflow-y: auto;
  padding: 32px 40px;
  min-width: 0;
  width: 100%;
}

.detail-tabs {
  min-height: 0;

  :deep(.el-tabs__header) {
    margin: 10px 0 18px;
    overflow: visible;
  }

  :deep(.el-tabs__nav-wrap) {
    overflow: visible !important;
  }

  :deep(.el-tabs__nav-wrap::after) {
    display: none;
    height: 0;
    background: transparent;
  }

  :deep(.el-tabs__nav-scroll) {
    overflow: visible !important;
  }

  :deep(.el-tabs__nav) {
    padding: 0;
    border: none;
    background: transparent;
    border-radius: 0;
    overflow: visible;
    box-shadow: none;
  }

  :deep(.el-tabs__item) {
    height: 38px;
    line-height: 38px;
    font-size: 14px;
    color: var(--el-text-color-secondary);
    border: none;
    background: transparent;
    margin-right: 0;
    border-radius: 0;
    transition: all 0.2s ease;
    padding: 0 6px 2px;
    overflow: visible;
    font-weight: 500;

    &:hover {
      color: var(--el-color-primary);
      background: transparent;
    }

    &.is-active {
      color: var(--el-color-primary);
      background: transparent;
      border: none;
      font-weight: 600;
      opacity: 1;
      box-shadow: none;
    }
  }

  :deep(.el-tabs__item + .el-tabs__item) {
    margin-left: 22px;
  }

  :deep(.el-tabs__active-bar) {
    height: 2px;
    border-radius: 999px;
    background: var(--el-color-primary);
  }

  :deep(.el-badge) {
    position: relative;
    display: inline-block;

    .el-badge__content {
      font-size: 11px;
      height: 18px;
      line-height: 18px;
      padding: 0 6px;
      min-width: 18px;
      border-radius: 9px;
      z-index: 10;
      box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    }
  }
}

.tab-content {
  padding: 0;
}

.directory-detail-tab-content,
.operate-log-tab-content,
.scheduled-agent-tab-content,
.access-tab-content {
  min-height: 360px;
}

.directory-markdown-body {
  margin-bottom: 18px;
  color: var(--el-text-color-regular);
  font-size: 14px;
  line-height: 1.72;
}

.directory-detail-tab-content :deep(.children-section) {
  margin-top: 0;
}

.directory-detail-tab-content :deep(.empty-state) {
  margin-top: 18px;
}

.directory-markdown-body :deep(h1),
.directory-markdown-body :deep(h2),
.directory-markdown-body :deep(h3),
.directory-markdown-body :deep(h4),
.directory-markdown-body :deep(h5),
.directory-markdown-body :deep(h6) {
  margin: 18px 0 10px;
  color: var(--el-text-color-primary);
  line-height: 1.35;
}

.directory-markdown-body :deep(p) {
  margin: 8px 0;
}

.directory-markdown-body :deep(ul),
.directory-markdown-body :deep(ol) {
  margin: 8px 0;
  padding-left: 22px;
}

.directory-markdown-body :deep(blockquote) {
  margin: 12px 0;
  padding: 8px 12px;
  border-left: 3px solid var(--el-color-primary);
  background: var(--app-shell-panel-muted-bg);
  border-radius: 6px;
}

.directory-markdown-body :deep(code) {
  padding: 2px 5px;
  border-radius: 5px;
  background: var(--app-shell-panel-muted-bg);
  color: var(--el-color-primary);
}

.directory-markdown-body :deep(pre) {
  padding: 12px;
  border-radius: 8px;
  background: #0f172a;
  color: #e2e8f0;
  overflow-x: auto;
}

.directory-markdown-body :deep(pre code) {
  padding: 0;
  background: transparent;
  color: inherit;
}

.directory-markdown-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
}

.directory-markdown-body :deep(th),
.directory-markdown-body :deep(td) {
  padding: 8px 10px;
  border: 1px solid var(--app-shell-panel-border);
}

.directory-markdown-body :deep(th) {
  background: var(--app-shell-panel-muted-bg);
  font-weight: 700;
}

@media (max-width: 768px) {
  .detail-content {
    padding: 24px 20px;
  }
}
</style>
