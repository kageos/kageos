<template>
  <div class="detail-content">
    <div v-if="showDirectoryTabs" class="directory-tabs-section">
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
import { computed } from 'vue'
import type { ServiceTree } from '@/types'
import PackageDetailOverviewCard from './PackageDetailOverviewCard.vue'
import PackageDetailChildrenGrid from './PackageDetailChildrenGrid.vue'
import ScheduledAgentTaskList from './ScheduledAgentTaskList.vue'
import type { WorkspaceSessionItem } from '@/architecture/infrastructure/api/workspace'
import { useLazyMarkdownRenderer } from '@/composables/useLazyMarkdownRenderer'
import { featureFlags } from '@/config/features'

type DetailTabName = 'info' | 'scheduledAgentTask'

const props = defineProps<{
  packageNode: ServiceTree | null
  totalRunCount: number
  activeTab: DetailTabName
}>()

const emit = defineEmits<{
  (e: 'update:activeTab', value: DetailTabName): void
  (e: 'select-child', child: ServiceTree): void
  (e: 'open-session', session: WorkspaceSessionItem): void
}>()

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
  return showScheduledAgentTaskTab.value
})

const directoryMarkdown = computed(() => {
  return props.packageNode?.description?.trim() || ''
})

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
