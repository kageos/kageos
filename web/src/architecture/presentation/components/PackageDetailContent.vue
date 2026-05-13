<template>
  <div class="detail-content">
    <div v-if="hasNoDirectoryPermissions" class="permission-error-wrapper">
      <el-card class="permission-error-card" shadow="hover">
        <template #header>
          <div class="permission-error-header">
            <el-icon class="permission-error-icon"><Lock /></el-icon>
            <span class="permission-error-title">权限不足</span>
          </div>
        </template>
        <div class="permission-error-content">
          <div class="permission-error-message">
            <p class="error-message-text">
              您没有 <strong>访问该目录</strong> 的权限
            </p>
          </div>
          <div v-if="packageNode?.full_code_path" class="permission-error-info">
            <el-icon><Document /></el-icon>
            <span class="info-label">资源路径：</span>
            <span class="info-value">{{ packageNode.full_code_path }}</span>
          </div>
          <div class="permission-error-actions">
            <el-button type="primary" size="default" :icon="Lock" @click="$emit('apply-permission')">
              立即申请权限
            </el-button>
          </div>
        </div>
      </el-card>
    </div>

    <div v-else-if="showDirectoryTabs" class="permission-request-section">
      <el-tabs v-model="currentActiveTab" class="detail-tabs">
        <el-tab-pane name="info">
          <template #label>
            <span>目录信息</span>
          </template>
          <div class="tab-content">
            <PackageDetailOverviewCard
              :package-node="packageNode || null"
              :total-run-count="totalRunCount"
            />
            <details v-if="directoryMarkdown" class="directory-markdown-detail">
              <summary class="directory-markdown-summary">
                <span>目录详情</span>
                <span class="directory-markdown-summary-hint">展开</span>
              </summary>
              <div class="directory-markdown-body" v-html="renderMarkdown(directoryMarkdown)" />
            </details>
            <PackageDetailChildrenGrid
              :children="packageNode?.children || []"
              @select-child="$emit('select-child', $event)"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="showScheduledAgentTaskTab" name="scheduledAgentTask">
          <template #label>
            <span>定时会话</span>
          </template>
          <div class="tab-content scheduled-agent-tab-content">
            <ScheduledAgentTaskList
              :resource-path="packageNode?.full_code_path"
              :auto-load="currentActiveTab === 'scheduledAgentTask'"
              @open-session="$emit('open-session', $event)"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="showPermissionRequestTab" name="permissionRequest">
          <template #label>
            <el-badge
              :value="packageNode?.pending_count || 0"
              :hidden="!packageNode?.pending_count || packageNode.pending_count === 0"
              :max="99"
            >
              <span>权限审批</span>
            </el-badge>
          </template>
          <div class="tab-content">
            <PermissionRequestList
              ref="permissionRequestListRef"
              :resource-path="packageNode?.full_code_path"
              :resource-type="resourceType"
              :auto-load="currentActiveTab === 'permissionRequest'"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="showPermissionRequestTab" name="permissionManage">
          <template #label>
            <span>授权记录</span>
          </template>
          <div class="tab-content">
            <PermissionManageList
              ref="permissionManageListRef"
              :resource-path="packageNode?.full_code_path"
              :resource-type="resourceType"
              :auto-load="currentActiveTab === 'permissionManage'"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <div v-else-if="packageNode">
      <PackageDetailOverviewCard
        :package-node="packageNode"
        :total-run-count="totalRunCount"
      />
      <details v-if="directoryMarkdown" class="directory-markdown-detail">
        <summary class="directory-markdown-summary">
          <span>目录详情</span>
          <span class="directory-markdown-summary-hint">展开</span>
        </summary>
        <div class="directory-markdown-body" v-html="renderMarkdown(directoryMarkdown)" />
      </details>
      <PackageDetailChildrenGrid
        :children="packageNode.children || []"
        @select-child="$emit('select-child', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { Document, Lock } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/types'
import PermissionRequestList from '@/shared/components/permission/PermissionRequestList.vue'
import PermissionManageList from '@/shared/components/permission/PermissionManageList.vue'
import PackageDetailOverviewCard from './PackageDetailOverviewCard.vue'
import PackageDetailChildrenGrid from './PackageDetailChildrenGrid.vue'
import ScheduledAgentTaskList from './ScheduledAgentTaskList.vue'
import type { WorkspaceSessionItem } from '@/api/workspace'
import { useLazyMarkdownRenderer } from '@/composables/useLazyMarkdownRenderer'
import { featureFlags } from '@/config/features'

type DetailTabName = 'info' | 'permissionRequest' | 'permissionManage' | 'scheduledAgentTask'

const props = defineProps<{
  packageNode: ServiceTree | null
  totalRunCount: number
  hasNoDirectoryPermissions: boolean
  showPermissionRequestTab: boolean
  activeTab: DetailTabName
  resourceType: 'directory'
}>()

const emit = defineEmits<{
  (e: 'update:activeTab', value: DetailTabName): void
  (e: 'apply-permission'): void
  (e: 'select-child', child: ServiceTree): void
  (e: 'open-session', session: WorkspaceSessionItem): void
}>()

const permissionRequestListRef = ref<InstanceType<typeof PermissionRequestList> | null>(null)
const permissionManageListRef = ref<InstanceType<typeof PermissionManageList> | null>(null)
const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()

const currentActiveTab = computed({
  get: () => props.activeTab,
  set: (value: DetailTabName) => emit('update:activeTab', value)
})

const showScheduledAgentTaskTab = computed(() => {
  return featureFlags.scheduledTasks && props.packageNode?.type === 'package' && !!props.packageNode.full_code_path
})

const showDirectoryTabs = computed(() => {
  return props.showPermissionRequestTab || showScheduledAgentTaskTab.value
})

const directoryMarkdown = computed(() => {
  return props.packageNode?.description?.trim() || ''
})

watch(
  () => currentActiveTab.value,
  (tabName) => {
    if (tabName === 'permissionRequest') {
      nextTick(() => {
        permissionRequestListRef.value?.loadRequests()
      })
      return
    }

    if (tabName === 'permissionManage') {
      nextTick(() => {
        permissionManageListRef.value?.loadPermissions()
      })
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

.permission-error-wrapper {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 400px;
  padding: 40px 20px;
}

.permission-error-card {
  max-width: 600px;
  width: 100%;
  border-radius: 16px;
  border: none;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;

  &:hover {
    box-shadow: 0 6px 24px rgba(0, 0, 0, 0.12);
    transform: translateY(-2px);
  }
}

.permission-error-header {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 18px;
  font-weight: 600;
  color: var(--el-color-warning);
}

.permission-error-icon {
  font-size: 24px;
}

.permission-error-title {
  font-size: 18px;
}

.permission-error-content {
  padding: 8px 0;
}

.permission-error-message {
  margin-bottom: 24px;
  padding: 16px;
  background: rgba(245, 158, 11, 0.08);
  border-radius: 12px;
  border-left: 4px solid var(--el-color-warning);
}

.error-message-text {
  margin: 0;
  font-size: 15px;
  line-height: 1.6;
  color: var(--el-text-color-primary);

  strong {
    color: var(--el-color-warning);
    font-weight: 600;
  }
}

.permission-error-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding: 12px 16px;
  background: var(--el-bg-color-page);
  border-radius: 10px;
  font-size: 14px;
  transition: all 0.2s ease;

  &:hover {
    background: var(--el-fill-color-light);
  }

  .el-icon {
    color: var(--el-color-info);
    font-size: 18px;
  }

  .info-label {
    color: var(--el-text-color-regular);
    font-weight: 500;
  }

  .info-value {
    color: var(--el-text-color-primary);
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 13px;
    word-break: break-all;
  }
}

.permission-error-actions {
  margin-top: 24px;
  display: flex;
  justify-content: center;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.detail-tabs {
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

.directory-markdown-detail {
  margin: -8px 0 24px;
  border: 1px solid var(--app-shell-panel-border);
  border-radius: 8px;
  background: var(--app-shell-panel-bg-strong);
  box-shadow: var(--app-shell-panel-shadow-soft);
  overflow: hidden;
}

.directory-markdown-summary {
  min-height: 44px;
  padding: 0 14px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  list-style: none;
  user-select: none;
}

.directory-markdown-summary::-webkit-details-marker {
  display: none;
}

.directory-markdown-summary::before {
  content: '';
  width: 0;
  height: 0;
  border-top: 5px solid transparent;
  border-bottom: 5px solid transparent;
  border-left: 6px solid var(--el-color-primary);
  transition: transform 0.18s ease;
}

.directory-markdown-detail[open] .directory-markdown-summary::before {
  transform: rotate(90deg);
}

.directory-markdown-summary-hint {
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.directory-markdown-detail[open] .directory-markdown-summary-hint {
  font-size: 0;
}

.directory-markdown-detail[open] .directory-markdown-summary-hint::after {
  content: '收起';
  font-size: 12px;
}

.directory-markdown-body {
  padding: 0 18px 18px 36px;
  color: var(--el-text-color-regular);
  font-size: 14px;
  line-height: 1.72;
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
