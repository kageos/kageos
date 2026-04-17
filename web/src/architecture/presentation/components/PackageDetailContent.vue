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

    <div v-else-if="showPermissionRequestTab" class="permission-request-section">
      <el-tabs v-model="currentActiveTab" type="card" class="detail-tabs">
        <el-tab-pane name="info">
          <template #label>
            <span>目录信息</span>
          </template>
          <div class="tab-content">
            <PackageDetailOverviewCard
              :package-node="packageNode || null"
              :total-run-count="totalRunCount"
            />
            <PackageDetailChildrenGrid
              :children="packageNode?.children || []"
              @select-child="$emit('select-child', $event)"
            />
          </div>
        </el-tab-pane>

        <el-tab-pane v-if="canEdit && packageNode?.full_code_path" name="import">
          <template #label>
            <span>导入 Go 文件</span>
          </template>
          <div class="tab-content import-tab-content">
            <div
              class="import-go-drop-zone"
              :class="{ 'import-go-drop-zone--dragover': isImportGoDragging }"
              @dragover.prevent="$emit('set-import-go-dragging', true)"
              @dragleave.prevent="$emit('set-import-go-dragging', false)"
              @drop.prevent="$emit('import-go-drop', $event)"
            >
              <span>将 .go 文件拖到此处导入到「{{ packageNode?.name }}」</span>
            </div>
            <p class="import-tab-hint">支持多个 .go 文件，导入后可在工作台执行编译。</p>
          </div>
        </el-tab-pane>

        <el-tab-pane name="permissionRequest">
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

        <el-tab-pane name="permissionManage">
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

type DetailTabName = 'info' | 'import' | 'permissionRequest' | 'permissionManage'

const props = defineProps<{
  packageNode: ServiceTree | null
  totalRunCount: number
  hasNoDirectoryPermissions: boolean
  showPermissionRequestTab: boolean
  canEdit: boolean
  activeTab: DetailTabName
  isImportGoDragging: boolean
  resourceType: 'directory'
}>()

const emit = defineEmits<{
  (e: 'update:activeTab', value: DetailTabName): void
  (e: 'apply-permission'): void
  (e: 'select-child', child: ServiceTree): void
  (e: 'import-go-drop', event: DragEvent): void
  (e: 'set-import-go-dragging', value: boolean): void
}>()

const permissionRequestListRef = ref<InstanceType<typeof PermissionRequestList> | null>(null)
const permissionManageListRef = ref<InstanceType<typeof PermissionManageList> | null>(null)

const currentActiveTab = computed({
  get: () => props.activeTab,
  set: (value: DetailTabName) => emit('update:activeTab', value)
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
    margin-bottom: 24px;
    overflow: visible;
  }

  :deep(.el-tabs__nav-wrap) {
    overflow: visible !important;
  }

  :deep(.el-tabs__nav-wrap::after) {
    display: none;
  }

  :deep(.el-tabs__nav-scroll) {
    overflow: visible !important;
  }

  :deep(.el-tabs__nav) {
    padding: 6px;
    border: 1px solid var(--app-shell-panel-border);
    background: var(--app-shell-panel-muted-bg);
    border-radius: 20px;
    overflow: visible;
    box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
  }

  :deep(.el-tabs__item) {
    height: 42px;
    line-height: 42px;
    font-size: 14px;
    color: var(--el-text-color-regular);
    border: none;
    background: transparent;
    margin-right: 0;
    border-radius: 14px;
    transition: all 0.2s ease;
    padding: 0 20px;
    overflow: visible;
    font-weight: 500;

    &:hover {
      color: var(--el-color-primary);
      background: rgba(var(--el-color-primary-rgb), 0.06);
    }

    &.is-active {
      color: var(--el-color-primary);
      background: var(--app-shell-panel-bg-strong);
      border: 1px solid rgba(var(--el-color-primary-rgb), 0.14);
      font-weight: 600;
      opacity: 1;
      box-shadow: 0 14px 28px rgba(15, 23, 42, 0.08);
    }
  }

  :deep(.el-tabs__item + .el-tabs__item) {
    margin-left: 6px;
  }

  :deep(.el-tabs__active-bar) {
    display: none;
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

.import-tab-content {
  padding: 24px 0;
}

.import-go-drop-zone {
  padding: 28px 18px;
  border: 1px dashed rgba(var(--el-color-primary-rgb), 0.28);
  border-radius: 20px;
  font-size: 14px;
  color: var(--el-color-primary);
  text-align: center;
  transition: border-color 0.2s, background 0.2s;
  background: var(--app-shell-panel-muted-bg);
  box-shadow: inset 0 1px 0 var(--app-shell-panel-highlight);
}

.import-go-drop-zone--dragover {
  border-color: var(--el-color-primary);
  background: rgba(var(--el-color-primary-rgb), 0.08);
}

.import-tab-hint {
  margin: 12px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

@media (max-width: 768px) {
  .detail-content {
    padding: 24px 20px;
  }
}
</style>
