<template>
  <div
    class="service-tree-panel"
    data-testid="service-tree-panel"
    v-loading="panelLoading"
    :element-loading-text="panelLoadingText"
    element-loading-background="rgba(2, 6, 23, 0.18)"
  >
    <div class="tree-header">
      <div class="tree-primary-row">
        <el-input
          v-model="searchKeyword"
          class="tree-search-input"
          :placeholder="t('serviceTree.searchPlaceholder')"
          clearable
          :prefix-icon="Search"
          data-testid="service-tree-search"
        />
        <el-popover
          v-if="!multiSelectMode"
          v-model:visible="treeToolsVisible"
          placement="bottom-end"
          trigger="click"
          :width="260"
          popper-class="service-tree-tools-popper"
        >
          <template #reference>
            <el-button
              class="tree-tools-button"
              size="small"
              text
              :icon="MoreFilled"
              :aria-label="t('serviceTree.directoryTools')"
              data-testid="service-tree-tools"
            />
          </template>
          <div class="tree-tools-menu">
            <button
              v-if="featureFlags.capabilityBundle"
              type="button"
              class="tree-tool-action"
              :disabled="!canAdmin(resolveHeaderImportTarget())"
              :title="canAdmin(resolveHeaderImportTarget()) ? t('serviceTree.importBundle') : t('access.requiresPermission', { permission: 'Admin' })"
              data-testid="service-tree-import-directory"
              @click="handleHeaderImport"
            >
              <el-icon><Upload /></el-icon>
              <span>{{ t('serviceTree.importBundle') }}</span>
            </button>
            <button type="button" class="tree-tool-action" @click="handleHeaderMultiSelect">
              <el-icon><Select /></el-icon>
              <span>{{ t('serviceTree.multiSelect') }}</span>
            </button>
            <div class="tree-tool-preference">
              <span>
                <strong>{{ t('serviceTree.authorizedOnly') }}</strong>
                <small>{{ t('serviceTree.authorizedOnlyDescription') }}</small>
              </span>
              <el-switch
                v-model="authorizedOnly"
                data-testid="service-tree-authorized-only"
                :aria-label="t('serviceTree.authorizedOnly')"
              />
            </div>
          </div>
        </el-popover>
      </div>
      <div v-if="multiSelectMode" class="tree-bulk-toolbar">
        <span class="bulk-selected-count">{{ t('common.selectedCount', { count: selectedNodeCount }) }}</span>
        <el-button
          v-if="featureFlags.capabilityBundle"
          size="small"
          :icon="Download"
          :loading="bulkExporting"
          :disabled="exportableSelectedNodes.length === 0"
          @click="handleBulkExport"
        >
          {{ t('common.export') }}
        </el-button>
        <el-button
          size="small"
          type="danger"
          plain
          :icon="Delete"
          :disabled="deletableSelectedNodes.length === 0"
          @click="handleBulkDelete"
        >
          {{ t('common.delete') }}
        </el-button>
        <el-button size="small" text :icon="Close" @click="exitMultiSelectMode">
          {{ t('common.cancel') }}
        </el-button>
      </div>
    </div>

    <div class="tree-content" data-testid="service-tree-content">
      <el-tree
        v-if="groupedTreeData.length > 0"
        :key="treeKey"
        ref="treeRef"
        :data="groupedTreeData"
        :props="treeProps"
        node-key="id"
        :show-checkbox="multiSelectMode"
        :check-strictly="true"
        :default-expand-all="false"
        :default-expanded-keys="defaultExpandedKeysWithWorkspace"
        :expanded-keys="expandedKeysState"
        :expand-on-click-node="false"
        :highlight-current="true"
        :filter-node-method="filterNodeMethod"
        :class="{ 'service-tree--bulk-selecting': multiSelectMode }"
        @node-click="handleNodeClick"
        @check="handleBulkCheck"
      >
        <template #default="{ node, data }">
          <el-dropdown
            trigger="contextmenu"
            :disabled="multiSelectMode"
            :teleported="true"
            popper-class="service-tree-contextmenu-popper"
            @command="(command: ServiceTreeNodeActionCommand) => handleNodeAction(command, data)"
          >
            <ServiceTreeNodeContent
              :data-testid="`service-tree-node-${data.id}`"
              :data-node-id="String(data.id)"
              :data-node-type="data.type"
              :data-root-node="isRootNode(data) ? 'true' : 'false'"
              :draggable="!multiSelectMode && canRead(data) && (data.type === 'function' || data.type === 'package')"
              :node="data"
              :label="node.label"
              :active="String(currentNodeId || '') === String(data.id || '')"
              :show-runtime-badge="hasRuntimeBadge(data)"
              :runtime-badge-value="getRuntimeBadgeText(data)"
              :runtime-badge-class="getRuntimeBadgeClass(data)"
              :runtime-badge-title="getRuntimeSummaryTitle(data)"
              :show-notification-badge="hasNotificationBadge(data)"
              :notification-badge-value="getNotificationBadgeText(data)"
              :notification-badge-title="getNotificationSummaryTitle(data)"
              :show-notification-route-badge="hasNotificationRouteBadge(data)"
              :notification-route-badge-title="getNotificationRouteSummaryTitle(data)"
              :notification-route-badge-tone="getNotificationRouteBadgeTone(data)"
              :notification-route-channels="getNotificationRouteChannels(data)"
              :show-scheduled-agent-badge="hasScheduledAgentBadge(data)"
              :scheduled-agent-badge-title="getScheduledAgentBadgeTitle(data)"
              :scheduled-agent-state="getScheduledAgentState(data)"
              :show-access-lock="!canRead(data)"
              :access-request-pending="Boolean(getOwnPendingRequestSource(data.full_code_path))"
              :access-lock-title="getAccessLockTitle(data)"
              :show-permission-request-badge="getPermissionRequestSummary(data).totalCount > 0"
              :permission-request-badge-value="getPermissionRequestSummary(data).totalCount"
              :permission-request-badge-class="getPermissionRequestSummary(data).reviewPendingCount > 0 ? 'needs-review' : 'is-mine'"
              :permission-request-badge-title="getPermissionRequestBadgeTitle(data)"
              @scheduled-agent-click="handleNodeClick(data)"
              @access-request-click="openAccessRequestPage(data)"
              @permission-request-click="openNodePermissionRecords(data.full_code_path)"
              @dragstart="onTreeNodeDragStart($event, data)"
              @contextmenu.prevent
              @notification-click="openNodeNotifications(data)"
              @notification-route-click="openNodeNotificationRoutes(data)"
              :title="multiSelectMode ? t('serviceTree.clickSelect') : t('serviceTree.rightClickMenu')"
            >
              <template #actions>
                <!-- 更多操作按钮 - 鼠标悬停时显示（与右键菜单并存，点击也可打开） -->
                <el-dropdown
                  v-if="!multiSelectMode"
                  trigger="click"
                  :teleported="true"
                  popper-class="service-tree-contextmenu-popper"
                  @click.stop
                  class="node-more-actions"
                  @command="(command: ServiceTreeNodeActionCommand) => handleNodeAction(command, data)"
                >
                  <el-button
                    class="node-more-button"
                    text
                    :icon="MoreFilled"
                    :aria-label="t('serviceTree.moreActions')"
                    :data-testid="`service-tree-more-${data.id}`"
                    @click.stop
                  />
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item
                        v-for="action in getNodeActions(data)"
                        :key="action.command"
                        :data-testid="buildServiceTreeNodeActionTestId(action.command, data)"
                        :command="action.command"
                        :disabled="action.disabled"
                        :title="action.disabledReason || action.label"
                      >
                        <el-icon><component :is="action.icon" /></el-icon>
                        {{ action.label }}
                      </el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </template>
            </ServiceTreeNodeContent>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  v-for="action in getNodeActions(data)"
                  :key="`context-${action.command}`"
                  :command="action.command"
                  :disabled="action.disabled"
                  :title="action.disabledReason || action.label"
                >
                  <el-icon><component :is="action.icon" /></el-icon>
                  {{ action.label }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-tree>
    </div>

    <WorkspaceImportDirectoryDialog
      v-model:visible="importDirectoryDialogVisible"
      :target-node="importDirectoryTargetNode"
      @imported="handleDirectoryImported"
    />

    <el-dialog
      v-model="exportDialogVisible"
      width="680px"
      title="导出自动执行配置"
      destroy-on-close
      :close-on-click-modal="false"
    >
      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="服务目录内置任务会强制随服务目录导出；自定义任务可按需选择，选中后会作为新服务目录的内置模板。"
      />
      <el-alert
        v-if="exportPreviewWarnings.length"
        class="export-preview-warning"
        type="warning"
        :closable="false"
        show-icon
        :title="exportPreviewWarnings.join('；')"
      />
      <section class="export-task-section">
        <div class="export-task-section-head">
          <strong>服务目录内置</strong>
          <span>{{ exportBuiltinTasks.length }} 项 · 必选且不可修改</span>
        </div>
        <div v-if="exportBuiltinTasks.length" class="export-task-list">
          <label v-for="item in exportBuiltinTasks" :key="`${item.kind}:${item.task.id}`" class="export-task-row is-locked">
            <el-checkbox :model-value="true" disabled />
            <span class="export-task-main">
              <span>{{ item.task.title || '未命名任务' }}</span>
              <small>{{ automationKindLabel(item.kind) }} · {{ item.resource_path }}</small>
            </span>
            <el-tag size="small" type="info">服务目录内置</el-tag>
          </label>
        </div>
        <el-empty v-else :image-size="52" description="没有服务目录内置任务" />
      </section>
      <section class="export-task-section">
        <div class="export-task-section-head">
          <strong>自定义</strong>
          <span>{{ exportUserTasks.length }} 项 · 默认不导出</span>
        </div>
        <el-checkbox-group v-if="exportUserTasks.length" v-model="selectedExportUserTaskIDs" class="export-task-list">
          <label v-for="item in exportUserTasks" :key="`${item.kind}:${item.task.id}`" class="export-task-row">
            <el-checkbox :value="item.task.id" />
            <span class="export-task-main">
              <span>{{ item.task.title || '未命名任务' }}</span>
              <small>{{ automationKindLabel(item.kind) }} · {{ item.resource_path }}</small>
            </span>
            <el-tag size="small" type="success">自定义</el-tag>
          </label>
        </el-checkbox-group>
        <el-empty v-else :image-size="52" description="没有可选的自定义任务" />
      </section>
      <template #footer>
        <el-button @click="exportDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="exportConfirming" @click="confirmCapabilityExport">
          确认导出
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { MoreFilled, Download, Delete, Search, Select, Close, Upload } from '@element-plus/icons-vue'
import { ElLoading, ElMessageBox, ElMessage } from 'element-plus'
import type { ServiceTree } from '@/architecture/domain/types'
import { isRootNode } from '@/architecture/domain/utils/tree-utils'
import {
  exportCapabilityBundle,
  getDirectoryOverview,
  updatePackage,
  updateServiceTreeFunction,
  updateDocs,
  type DirectoryOverviewScheduledTask,
} from '@/architecture/presentation/context/api/service-tree'
import { getRuntimeStateSummary, type RuntimeStateSummary } from '@/architecture/presentation/context/api/state'
import {
  listMessageNotificationRouteSummary,
  type MessageNotificationRouteInfo,
  type MessageNotificationRoutePathSummary
} from '@/architecture/presentation/context/api/message'
import { downloadCapabilityBundleFile } from '@/architecture/presentation/utils/directoryBundleFile'
import { eventBus } from '@/architecture/presentation/context/eventBusContext'
import { resolveWorkspaceUrl } from '@/architecture/shared/routing/route'
import { useServiceTreeClipboard } from '../composables/useServiceTreeClipboard'
import { useServiceTreeSearchExpand } from '../composables/useServiceTreeSearchExpand'
import {
  buildServiceTreeNodeActionTestId,
  getServiceTreeNodeActions,
  type ServiceTreeNodeActionCommand
} from '../utils/serviceTreeNodeActions'
import { featureFlags } from '@/architecture/shared/config/features'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import { canAdmin, canDelete, canRead } from '@/architecture/presentation/composables/useAccessControl'
import { findNearestPermissionRequestAncestor } from '@/architecture/presentation/features/access/utils/permissionRequestSelection'
import { getPermissionRequestWorkspaceRoot } from '@/architecture/presentation/features/access/utils/permissionRequestSummary'
import {
  getPermissionRequestSummaryState,
  loadPermissionRequestSummary,
  ownPendingPermissionRequestPaths,
  seedPermissionRequestSummaryFromTree,
} from '@/architecture/presentation/features/access/utils/permissionRequestSummaryStore'
import ServiceTreeNodeContent from './ServiceTreeNodeContent.vue'
import WorkspaceImportDirectoryDialog from './WorkspaceImportDirectoryDialog.vue'
import {
  aggregatePermissionRequestSummaries,
  collectPermissionRequestExpandedDirectoryIds,
  filterServiceTreeByReadAccess,
} from '@/architecture/presentation/utils/serviceTreePermissionView'

interface Props {
  treeData: ServiceTree[]
  loading?: boolean
  currentNodeId?: number | string | null
  currentFunction?: ServiceTree | null  // 当前选中的节点（用于判断是否可以克隆）
  expandedKeys?: number[] // ⭐ 需要自动展开的节点ID列表（从后端返回）
  messageCounts?: Record<string, ServiceTreeNotificationCount>
}

interface Emits {
  (e: 'node-click', node: ServiceTree): void
  (e: 'open-notifications', node: ServiceTree): void
  (e: 'create-directory', parentNode?: ServiceTree): void
  (e: 'create-docs', parentNode?: ServiceTree): void
  (e: 'delete-doc', node: ServiceTree): void
  (e: 'delete-function', node: ServiceTree): void  // 删除函数
  (e: 'delete-directory', node: ServiceTree): void  // 删除服务目录（非根 package）
  (e: 'bulk-delete', nodes: ServiceTree[]): void
  (e: 'refresh-tree'): void  // 刷新树（复制粘贴后需要刷新）
  (e: 'update-history', node?: ServiceTree): void  // 显示变更记录（工作空间或服务目录）
}

interface ServiceTreeNotificationCount {
  unread_count?: number
  message_count?: number
  latest_at?: string
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()

const runtimeSummaries = ref<Record<string, RuntimeStateSummary>>({})
const notificationRouteSummaries = ref<Record<string, MessageNotificationRoutePathSummary>>({})
let runtimeSummaryTimer: ReturnType<typeof setInterval> | null = null

const rootFullCodePath = computed(() => props.treeData[0]?.full_code_path || '')
const authorizedOnly = ref(false)
const treeToolsVisible = ref(false)
const treePreferenceKey = computed(() => {
  const username = authStore.user?.username || 'anonymous'
  const root = rootFullCodePath.value
  return root ? `kageos:service-tree:authorized-only:${username}:${root}` : ''
})
const visibleTreeData = computed(() => (
  authorizedOnly.value ? filterServiceTreeByReadAccess(props.treeData) : props.treeData
))

watch(treePreferenceKey, (key) => {
  authorizedOnly.value = key ? localStorage.getItem(key) === 'true' : false
}, { immediate: true })

watch(authorizedOnly, (value) => {
  if (treePreferenceKey.value) localStorage.setItem(treePreferenceKey.value, String(value))
})

const {
  treeRef,
  groupedTreeData,
  searchKeyword,
  filterNodeMethod,
  defaultExpandedKeysWithWorkspace,
  expandedKeysState,
  treeKey,
  expandPaths,
  expandNodeIds,
} = useServiceTreeSearchExpand({
  treeData: visibleTreeData,
  expandedKeys: computed(() => props.expandedKeys)
})

const treeProps = {
  children: 'children',
  label: 'name'
}

const {
  copiedDirectory,
  handleCopy,
  handlePaste
} = useServiceTreeClipboard({
  treeData: groupedTreeData,
  currentFunction: computed(() => props.currentFunction),
  currentNodeId: computed(() => props.currentNodeId),
  onRefreshTree: () => emit('refresh-tree')
})

const permissionRequestWorkspaceRoot = computed(() => getPermissionRequestWorkspaceRoot(rootFullCodePath.value))
const permissionRequestSummaryState = computed(() => getPermissionRequestSummaryState(permissionRequestWorkspaceRoot.value))
const aggregatedPermissionRequestSummaries = computed(() => aggregatePermissionRequestSummaries(
  props.treeData,
  permissionRequestSummaryState.value.paths,
))
const permissionRequestExpandedDirectoryIds = computed(() => collectPermissionRequestExpandedDirectoryIds(
  visibleTreeData.value,
  aggregatedPermissionRequestSummaries.value,
))
const pendingAccessRequestPaths = computed(() => ownPendingPermissionRequestPaths(permissionRequestSummaryState.value))
const hasAnyAdminNode = computed(() => {
  const walk = (nodes: ServiceTree[]): boolean => nodes.some(node => canAdmin(node) || walk(node.children || []))
  return walk(props.treeData || [])
})
const multiSelectMode = ref(false)
const selectedNodes = ref<ServiceTree[]>([])
const bulkExporting = ref(false)
const renamingNode = ref(false)
const importDirectoryDialogVisible = ref(false)
const importDirectoryTargetNode = ref<ServiceTree | null>(null)
const exportDialogVisible = ref(false)
const exportConfirming = ref(false)
const exportBuiltinTasks = ref<DirectoryOverviewScheduledTask[]>([])
const exportUserTasks = ref<DirectoryOverviewScheduledTask[]>([])
const selectedExportUserTaskIDs = ref<number[]>([])
const exportPreviewWarnings = ref<string[]>([])
const pendingCapabilityExport = ref<{
  request: {
    source_directory_path?: string
    source_directory_paths?: string[]
    source_root_path?: string
    name?: string
  }
  downloadPath: string
  skippedCount: number
  exitBulkAfterSuccess: boolean
} | null>(null)
const pendingExpandPath = ref('')
let unsubscribeRuntimeRefresh: (() => void) | null = null
let unsubscribeNotificationRouteRefresh: (() => void) | null = null
let unsubscribePermissionRequestRefresh: (() => void) | null = null
let lastPermissionExpansionSignature = ''

watch(permissionRequestExpandedDirectoryIds, async (ids) => {
  const signature = ids.join(',')
  if (!signature || signature === lastPermissionExpansionSignature) {
    if (!signature) lastPermissionExpansionSignature = ''
    return
  }
  lastPermissionExpansionSignature = signature
  await nextTick()
  await expandNodeIds(ids)
}, { immediate: true, flush: 'post' })

const panelLoading = computed(() => Boolean(props.loading) || bulkExporting.value || renamingNode.value)
const panelLoadingText = computed(() => {
  if (renamingNode.value) return '正在更新服务目录...'
  if (bulkExporting.value) return '正在导出服务目录...'
  return '正在刷新服务目录...'
})

function showBlockingLoading(text: string) {
  return ElLoading.service({
    lock: true,
    text,
    background: 'rgba(15, 23, 42, 0.36)'
  })
}

function refreshTreeAndExpand(path?: string) {
  pendingExpandPath.value = path || ''
  emit('refresh-tree')
}

watch(
  () => props.treeData,
  async () => {
    const path = pendingExpandPath.value
    if (!path) return
    pendingExpandPath.value = ''
    await nextTick()
    expandPaths([path])
  },
  { flush: 'post' }
)

const stopRuntimeSummaryPolling = () => {
  if (runtimeSummaryTimer) {
    clearInterval(runtimeSummaryTimer)
    runtimeSummaryTimer = null
  }
  if (unsubscribeRuntimeRefresh) {
    unsubscribeRuntimeRefresh()
    unsubscribeRuntimeRefresh = null
  }
  window.removeEventListener('focus', refreshRuntimeSummary)
}

const refreshRuntimeSummary = async () => {
  const root = rootFullCodePath.value
  if (!root) {
    runtimeSummaries.value = {}
    return
  }
  try {
    const resp = await getRuntimeStateSummary({ root_full_code_path: root })
    runtimeSummaries.value = resp.summaries || {}
  } catch {
    // 运行态仅用于提示，不影响服务树主流程。
  }
}

const refreshNotificationRouteSummary = async () => {
  const root = rootFullCodePath.value
  if (!root || !hasAnyAdminNode.value) {
    notificationRouteSummaries.value = {}
    return
  }
  try {
    const resp = await listMessageNotificationRouteSummary(root)
    notificationRouteSummaries.value = resp.routes || {}
  } catch {
    // 通知路由标识仅用于辅助提示，不影响服务树主流程。
  }
}

const startRuntimeSummaryPolling = () => {
  stopRuntimeSummaryPolling()
  if (!rootFullCodePath.value) return
  runtimeSummaryTimer = setInterval(refreshRuntimeSummary, 15000)
  window.addEventListener('focus', refreshRuntimeSummary)
}

watch([rootFullCodePath, hasAnyAdminNode], () => {
  runtimeSummaries.value = {}
  notificationRouteSummaries.value = {}
  exitMultiSelectMode()
  refreshRuntimeSummary()
  refreshNotificationRouteSummary()
  startRuntimeSummaryPolling()
}, { immediate: true })

watch(
  () => props.treeData,
  (nodes) => {
    const root = permissionRequestWorkspaceRoot.value
    if (root) seedPermissionRequestSummaryFromTree(root, nodes)
  },
  { immediate: true },
)

async function refreshPendingAccessRequests() {
  const root = permissionRequestWorkspaceRoot.value
  if (!root) return
  try {
    await loadPermissionRequestSummary(root, { force: true })
  } catch {
    // 申请状态只用于树上提示，不阻断服务目录加载。
  }
}

unsubscribeNotificationRouteRefresh = eventBus.on('notification-route:changed', () => {
  void refreshNotificationRouteSummary()
})

unsubscribePermissionRequestRefresh = eventBus.on('permission-request:changed', () => {
  void refreshPendingAccessRequests()
})

onBeforeUnmount(() => {
  stopRuntimeSummaryPolling()
  if (unsubscribeNotificationRouteRefresh) {
    unsubscribeNotificationRouteRefresh()
    unsubscribeNotificationRouteRefresh = null
  }
  if (unsubscribePermissionRequestRefresh) {
    unsubscribePermissionRequestRefresh()
    unsubscribePermissionRequestRefresh = null
  }
})

const getRuntimeSummary = (node: ServiceTree): RuntimeStateSummary | undefined => {
  if (!node.full_code_path) return undefined
  return runtimeSummaries.value[node.full_code_path]
}

const hasRuntimeBadge = (node: ServiceTree): boolean => {
  const summary = getRuntimeSummary(node)
  return !!summary?.badge_text || !!summary?.running_count || !!summary?.failed_recent_count
}

const getRuntimeBadgeText = (node: ServiceTree): string | number => {
  const summary = getRuntimeSummary(node)
  return summary?.badge_text || summary?.running_count || ''
}

const getRuntimeBadgeClass = (node: ServiceTree): string => {
  const tone = getRuntimeSummary(node)?.badge_tone || 'running'
  return `runtime-state-badge runtime-state-badge-${tone}`
}

const getRuntimeSummaryTitle = (node: ServiceTree): string => {
  const summary = getRuntimeSummary(node)
  if (!summary) return ''
  if (summary.tooltip) return summary.tooltip
  const parts = [t('serviceTree.running', { count: summary.running_count })]
  if (summary.thinking_count > 0) parts.push(t('serviceTree.thinking', { count: summary.thinking_count }))
  if (summary.tool_running_count > 0) parts.push(t('serviceTree.toolRunning', { count: summary.tool_running_count }))
  if (summary.failed_recent_count > 0) parts.push(t('serviceTree.recentFailed', { count: summary.failed_recent_count }))
  return parts.join('，')
}

function normalizeTreePath(path?: string): string {
  const normalized = (path || '').trim().replace(/\/+$/g, '')
  return normalized && !normalized.startsWith('/') ? `/${normalized}` : normalized
}

function getNotificationScopeAncestors(path: string): string[] {
  const normalized = normalizeTreePath(path)
  if (!normalized) return []
  const parts = normalized.split('/').filter(Boolean)
  if (parts.length === 0) return []
  const minParts = parts.length >= 2 ? 2 : 1
  const paths: string[] = []
  for (let size = parts.length; size >= minParts; size -= 1) {
    paths.push(`/${parts.slice(0, size).join('/')}`)
  }
  return paths
}

const notificationSummaries = computed<Record<string, ServiceTreeNotificationCount>>(() => {
  const summaries: Record<string, ServiceTreeNotificationCount> = {}
  const walk = (node: ServiceTree) => {
    const path = normalizeTreePath(node.full_code_path)
    if (path) {
      summaries[path] = props.messageCounts?.[path] || {}
    }
    for (const child of node.children || []) {
      walk(child)
    }
  }
  for (const node of props.treeData || []) {
    walk(node)
  }
  return summaries
})

const getNotificationSummary = (node: ServiceTree): ServiceTreeNotificationCount | undefined => {
  const path = normalizeTreePath(node.full_code_path)
  if (!path) return undefined
  return notificationSummaries.value[path]
}

const hasNotificationBadge = (node: ServiceTree): boolean => {
  return Number(getNotificationSummary(node)?.unread_count || 0) > 0
}

const getNotificationBadgeText = (node: ServiceTree): string | number => {
  return getNotificationSummary(node)?.unread_count || ''
}

const getNotificationSummaryTitle = (node: ServiceTree): string => {
  const summary = getNotificationSummary(node)
  const unread = Number(summary?.unread_count || 0)
  const total = Number(summary?.message_count || 0)
  if (unread > 0) {
    return `${unread} 条未读通知，${total} 条通知`
  }
  return `${total} 条通知`
}

const getScheduledAgentTaskCount = (node: ServiceTree): number => {
  return Number(node.scheduled_agent_tasks || 0)
}

const hasScheduledAgentBadge = (node: ServiceTree): boolean => {
  return node.type === 'package' && getScheduledAgentTaskCount(node) > 0
}

const getScheduledAgentBadgeTitle = (node: ServiceTree): string => {
  const total = getScheduledAgentTaskCount(node)
  const enabled = Number(node.enabled_agent_tasks || 0)
  const running = Number(node.running_agent_tasks || 0)
  const failed = Number(node.failed_agent_tasks || 0)
  if (running > 0 && failed > 0) return `${running} 名数字员工正在处理，${failed} 名需要关注`
  if (running > 0) return `${running} 名数字员工正在处理，${total} 名员工在值守`
  if (failed > 0) return `${failed} 名数字员工需要关注`
  if (enabled > 0) return `${enabled} 名数字员工已启动，${total} 名员工在值守`
  return `${total} 名数字员工已配置，目前全部暂停`
}

const getScheduledAgentState = (node: ServiceTree): 'running' | 'enabled' | 'paused' | 'failed' => {
  if (Number(node.failed_agent_tasks || 0) > 0) return 'failed'
  if (Number(node.running_agent_tasks || 0) > 0) return 'running'
  if (Number(node.enabled_agent_tasks || 0) > 0) return 'enabled'
  return 'paused'
}

function openNodeNotifications(node: ServiceTree) {
  if (!node.full_code_path) return
  emit('open-notifications', node)
}

interface NotificationRouteEffectiveSummary {
  route?: MessageNotificationRouteInfo
  routes: MessageNotificationRouteInfo[]
  scopePath: string
  inherited: boolean
  disabled: boolean
}

function isNotificationRouteNode(node: ServiceTree): boolean {
  return node.type === 'package' || node.type === 'function'
}

function getUsableNotificationRoutes(summary?: MessageNotificationRoutePathSummary): MessageNotificationRouteInfo[] {
  return (summary?.routes || []).filter((route) => Boolean(route.enabled) && Boolean(route.has_webhook_url))
}

function getConfiguredNotificationRoutes(summary?: MessageNotificationRoutePathSummary): MessageNotificationRouteInfo[] {
  return (summary?.routes || []).filter((route) => {
    return Boolean(route.id || route.display_name || route.enabled || route.has_webhook_url || route.last_error)
  })
}

function getEffectiveNotificationRoute(node: ServiceTree): NotificationRouteEffectiveSummary | undefined {
  const path = normalizeTreePath(node.full_code_path)
  if (!path || !isNotificationRouteNode(node)) return undefined

  for (const scopePath of getNotificationScopeAncestors(path)) {
    const routes = getUsableNotificationRoutes(notificationRouteSummaries.value[scopePath])
    if (routes.length > 0) {
      return {
        route: routes[0],
        routes,
        scopePath,
        inherited: scopePath !== path,
        disabled: false
      }
    }
  }

  const directRoutes = getConfiguredNotificationRoutes(notificationRouteSummaries.value[path])
  if (directRoutes.length === 0) return undefined
  return {
    route: directRoutes[0],
    routes: directRoutes,
    scopePath: path,
    inherited: false,
    disabled: true
  }
}

function hasNotificationRouteBadge(node: ServiceTree): boolean {
  return canAdmin(node) && Boolean(getEffectiveNotificationRoute(node))
}

function getNotificationRouteChannels(node: ServiceTree): string[] {
  const summary = getEffectiveNotificationRoute(node)
  if (!summary) return []
  const channelOrder = ['feishu', 'wecom', 'dingtalk']
  return [...new Set(summary.routes.map((route) => String(route.channel || '').trim()).filter(Boolean))]
    .sort((left, right) => {
      const leftIndex = channelOrder.indexOf(left)
      const rightIndex = channelOrder.indexOf(right)
      return (leftIndex < 0 ? channelOrder.length : leftIndex) - (rightIndex < 0 ? channelOrder.length : rightIndex)
    })
}

function notificationChannelLabel(channel?: string): string {
  if (channel === 'feishu') return t('userSettings.channelFeishu')
  if (channel === 'wecom') return t('userSettings.channelWecom')
  if (channel === 'dingtalk') return t('userSettings.channelDingtalk')
  return channel || t('notificationRoute.unknownChannel')
}

function formatNotificationRouteName(route?: MessageNotificationRouteInfo): string {
  if (!route) return t('notificationRoute.unnamedRoute')
  const displayName = (route.display_name || '').trim() || t('notificationRoute.unnamedRoute')
  return `${displayName} · ${notificationChannelLabel(String(route.channel || ''))}`
}

function getNotificationRouteBadgeTone(node: ServiceTree): string {
  const summary = getEffectiveNotificationRoute(node)
  if (!summary) return 'direct'
  if (summary.disabled) return 'disabled'
  if (summary.routes.some((route) => route.last_error || Number(route.fail_count || 0) > 0)) return 'failed'
  if (summary.inherited) return 'inherited'
  return 'direct'
}

function getNotificationRouteSummaryTitle(node: ServiceTree): string {
  const summary = getEffectiveNotificationRoute(node)
  if (!summary) return ''
  const name = summary.routes.map(formatNotificationRouteName).join('、')
  let title = summary.disabled
    ? t('serviceTree.notificationRouteDisabledTitle', { name })
    : summary.inherited
      ? t('serviceTree.notificationRouteInheritedTitle', { name, path: summary.scopePath })
      : t('serviceTree.notificationRouteDirectTitle', { name })
  const lastError = summary.routes.find((route) => route.last_error)?.last_error
  if (lastError) {
    title += `；${t('serviceTree.notificationRouteLastError', { error: lastError })}`
  }
  return title
}

function openNodeNotificationRoutes(node: ServiceTree) {
  const path = normalizeTreePath(node.full_code_path)
  if (!path || !isNotificationRouteNode(node)) return
  if (!canAdmin(node)) {
    ElMessage.warning(t('access.requiresPermission', { permission: 'Admin' }))
    return
  }
  void router.push({
    path: resolveWorkspaceUrl(path),
    query: { _panel: 'notification' }
  })
}

function findPackageNodeById(nodes: ServiceTree[], id: number | string): ServiceTree | null {
  for (const node of nodes) {
    if (Number(node.id) === Number(id) && node.type === 'package') {
      return node
    }
    if (node.children?.length) {
      const found = findPackageNodeById(node.children, id)
      if (found) return found
    }
  }
  return null
}

function resolveHeaderImportTarget(): ServiceTree | null {
  if (props.currentFunction?.type === 'package') {
    return props.currentFunction
  }
  if (props.currentNodeId != null) {
    const node = findPackageNodeById(props.treeData, props.currentNodeId)
    if (node) return node
  }
  const rootNode = props.treeData[0]
  return rootNode?.type === 'package' ? rootNode : null
}

function openDirectoryImportDialog(node: ServiceTree | null | undefined) {
  if (!featureFlags.capabilityBundle) return
  if (!node || node.type !== 'package') {
    ElMessage.warning(t('serviceTree.importOnlyDirectory'))
    return
  }
  if (!node.full_code_path) {
    ElMessage.warning(t('serviceTree.pathMissingRefresh'))
    return
  }
  if (!canAdmin(node)) {
    ElMessage.warning(t('access.requiresPermission', { permission: 'Admin' }))
    return
  }
  importDirectoryTargetNode.value = node
  importDirectoryDialogVisible.value = true
}

function openCurrentDirectoryImportDialog() {
  openDirectoryImportDialog(resolveHeaderImportTarget())
}

function handleHeaderImport() {
  treeToolsVisible.value = false
  openCurrentDirectoryImportDialog()
}

function handleHeaderMultiSelect() {
  treeToolsVisible.value = false
  enterMultiSelectMode()
}

function handleDirectoryImported(path?: string) {
  refreshTreeAndExpand(path || importDirectoryTargetNode.value?.full_code_path || '')
}

// 重命名服务目录
const handleRename = async (node: ServiceTree) => {
  if (node.type !== 'package') {
    ElMessage.warning(t('serviceTree.renameOnlyDirectory'))
    return
  }
  
  try {
    const { value: newName } = await ElMessageBox.prompt(
      t('serviceTree.renamePrompt', { name: node.name }),
      t('serviceTree.renameTitle'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        inputPattern: /^.+$/,
        inputErrorMessage: t('serviceTree.nameRequired'),
        inputValue: node.name
      }
    )
    
    if (!newName || newName.trim() === '') {
      ElMessage.warning(t('serviceTree.nameRequired'))
      return
    }
    
    const trimmedName = newName.trim()
    
    // 如果名称没有变化，直接返回
    if (trimmedName === node.name) {
      return
    }
    
    const loadingInstance = showBlockingLoading('正在更新服务目录，请稍候...')
    renamingNode.value = true
    try {
      // ⭐ 根据节点类型调用对应的更新接口
      if (node.type === 'package') {
        await updatePackage(node.id, { name: trimmedName })
      } else if (node.type === 'function') {
        await updateServiceTreeFunction(node.id, { name: trimmedName })
      } else if (node.type === 'docs') {
        await updateDocs(node.id, { name: trimmedName })
      } else {
        ElMessage.warning(t('serviceTree.unsupportedNodeType'))
        return
      }
      ElMessage.success(t('serviceTree.renameSuccess'))
      
      // 刷新树
      refreshTreeAndExpand(node.full_code_path)
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('serviceTree.renameFailed')
      ElMessage.error(errorMessage)
    } finally {
      renamingNode.value = false
      loadingInstance.close()
    }
  } catch (_error) {
    // 用户取消了输入
  }
}

function onTreeNodeDragStart(e: DragEvent, data: ServiceTree) {
  if (!e.dataTransfer || !data.full_code_path) return
  e.dataTransfer.setData('application/x-workspace-node', JSON.stringify({
    type: data.type,
    full_code_path: data.full_code_path,
    name: data.name || data.full_code_path?.split('/').pop() || '',
    id: data.id,
  }))
  e.dataTransfer.effectAllowed = 'copy'
}

const handleNodeClick = (data: ServiceTree) => {
  if (multiSelectMode.value) {
    toggleNodeSelection(data)
    return
  }

  // 直接触发 node-click 事件，让父组件处理路由跳转
  // ⭐ 下拉菜单的点击已经通过 @click.stop.prevent 阻止了事件冒泡，所以这里不需要额外检查
  emit('node-click', data)
}

function getNodeActions(data: ServiceTree) {
  return getServiceTreeNodeActions(data, {
    hasCopiedDirectory: Boolean(copiedDirectory.value)
  })
}

function openAccessPage(data: ServiceTree) {
  if (!data.full_code_path) {
    ElMessage.warning(t('serviceTree.pathMissingRefresh'))
    return
  }
  void router.push({
    path: '/permissions',
    query: {
      resource: data.full_code_path
    }
  })
}

function openAccessRequestPage(data: ServiceTree) {
  if (!data.full_code_path) {
    ElMessage.warning(t('serviceTree.pathMissingRefresh'))
    return
  }
  const pendingSource = getOwnPendingRequestSource(data.full_code_path)
  if (pendingSource) {
    openNodePermissionRecords(pendingSource)
    return
  }
  void router.push({
    path: '/permissions',
    query: {
      resource: data.full_code_path,
      mode: 'request'
    }
  })
}

function getAccessLockTitle(data: ServiceTree): string {
  const pendingSource = getOwnPendingRequestSource(data.full_code_path)
  if (!pendingSource) return t('serviceTree.accessRequestAction')
  return pendingSource === data.full_code_path
    ? t('serviceTree.accessRequestPending')
    : t('serviceTree.accessRequestInheritedPending', { source: pendingSource })
}

function getOwnPendingRequestSource(resourcePath?: string): string {
  if (!resourcePath) return ''
  if (pendingAccessRequestPaths.value.has(resourcePath)) return resourcePath
  return findNearestPermissionRequestAncestor(resourcePath, pendingAccessRequestPaths.value) || ''
}

function getPermissionRequestSummary(data: ServiceTree) {
  return aggregatedPermissionRequestSummaries.value[normalizeTreePath(data.full_code_path)] || {
    ownPendingCount: 0,
    reviewPendingCount: 0,
    totalCount: 0,
  }
}

function getPermissionRequestBadgeTitle(data: ServiceTree): string {
  const summary = getPermissionRequestSummary(data)
  return t('serviceTree.permissionRequestBadgeTitle', {
    review: summary.reviewPendingCount,
    mine: summary.ownPendingCount,
  })
}

function openNodePermissionRecords(resourcePath?: string) {
  if (!resourcePath) return
  void router.push({
    path: resolveWorkspaceUrl(resourcePath),
    query: { _panel: 'permission' },
  })
}

const selectedNodeCount = computed(() => selectedNodes.value.length)

const exportableSelectedNodes = computed(() => {
  return compactSelectedTreeNodes(selectedNodes.value.filter(canExportNode))
})

const deletableSelectedNodes = computed(() => {
  return compactSelectedTreeNodes(selectedNodes.value.filter(canDeleteNode))
    .sort((left, right) => getNodePathDepth(right) - getNodePathDepth(left))
})

function enterMultiSelectMode() {
  multiSelectMode.value = true
  selectedNodes.value = []
  nextTick(() => {
    treeRef.value?.setCheckedKeys?.([])
  })
}

function exitMultiSelectMode() {
  multiSelectMode.value = false
  selectedNodes.value = []
  treeRef.value?.setCheckedKeys?.([])
}

function syncBulkSelection() {
  const checkedNodes = treeRef.value?.getCheckedNodes?.(false, false) as ServiceTree[] | undefined
  selectedNodes.value = (checkedNodes || []).filter((node) => Boolean(node.full_code_path))
}

function collectNodeAndDescendantIds(node: ServiceTree): Array<number | string> {
  const ids: Array<number | string> = []
  const walk = (current: ServiceTree) => {
    if (current.id) {
      ids.push(current.id)
    }
    for (const child of current.children || []) {
      walk(child)
    }
  }
  walk(node)
  return ids
}

function isNodeChecked(node: ServiceTree): boolean {
  if (!node.id) return false
  const checked = treeRef.value?.getCheckedKeys?.() as Array<number | string> | undefined
  const checkedKeys = new Set(checked || [])
  return checkedKeys.has(node.id)
}

function applyNodeSelectionCascade(node: ServiceTree, checked: boolean) {
  const currentKeys = treeRef.value?.getCheckedKeys?.() as Array<number | string> | undefined
  const checkedKeys = new Set(currentKeys || [])
  for (const id of collectNodeAndDescendantIds(node)) {
    if (checked) {
      checkedKeys.add(id)
    } else {
      checkedKeys.delete(id)
    }
  }
  treeRef.value?.setCheckedKeys?.([...checkedKeys])
  syncBulkSelection()
}

function handleBulkCheck(node: ServiceTree) {
  applyNodeSelectionCascade(node, isNodeChecked(node))
}

function toggleNodeSelection(node: ServiceTree) {
  if (!node.id) return
  applyNodeSelectionCascade(node, !isNodeChecked(node))
}

function getNodePathDepth(node: ServiceTree): number {
  return (node.full_code_path || '').split('/').filter(Boolean).length
}

function hasSelectedPackageAncestor(node: ServiceTree, packagePaths: string[]): boolean {
  const nodePath = node.full_code_path || ''
  if (!nodePath) return false
  return packagePaths.some((packagePath) => {
    return packagePath !== nodePath && nodePath.startsWith(`${packagePath}/`)
  })
}

function compactSelectedTreeNodes(nodes: ServiceTree[]): ServiceTree[] {
  const seen = new Set<number | string>()
  const uniqueNodes = nodes.filter((node) => {
    if (!node.id || seen.has(node.id)) return false
    seen.add(node.id)
    return Boolean(node.full_code_path)
  })
  const selectedPackagePaths = uniqueNodes
    .filter((node) => node.type === 'package' && node.full_code_path)
    .map((node) => node.full_code_path as string)

  return uniqueNodes.filter((node) => !hasSelectedPackageAncestor(node, selectedPackagePaths))
}

function canExportNode(node: ServiceTree): boolean {
  if (!node.full_code_path) return false
  return canRead(node) && (node.type === 'package' || node.type === 'function')
}

function canDeleteNode(node: ServiceTree): boolean {
  if (!node.full_code_path || !canDelete(node)) return false
  if (node.type === 'package') {
    return !isRootNode(node)
  }
  return node.type === 'function' || node.type === 'docs'
}

function buildBulkExportName(nodes: ServiceTree[]): string {
  const firstNode = nodes[0]
  if (nodes.length === 1 && firstNode) {
    return firstNode.name || firstNode.code || firstNode.full_code_path?.split('/').filter(Boolean).pop() || 'capability'
  }
  return props.treeData[0]?.name || 'capability'
}

function automationKindLabel(kind: DirectoryOverviewScheduledTask['kind']): string {
  return kind === 'agent' ? '数字员工' : '函数定时'
}

function isTaskInsideExportNodes(item: DirectoryOverviewScheduledTask, nodes: ServiceTree[]): boolean {
  const resourcePath = String(item.resource_path || item.task.resource_key || '').replace(/\/+$/, '')
  if (!resourcePath) return false
  return nodes.some((node) => {
    const nodePath = String(node.full_code_path || '').replace(/\/+$/, '')
    if (!nodePath) return false
    if (node.type === 'function') return resourcePath === nodePath
    return resourcePath === nodePath || resourcePath.startsWith(`${nodePath}/`)
  })
}

async function prepareCapabilityExport(
  request: {
    source_directory_path?: string
    source_directory_paths?: string[]
    source_root_path?: string
    name?: string
  },
  nodes: ServiceTree[],
  downloadPath: string,
  skippedCount = 0,
  exitBulkAfterSuccess = false,
) {
  const overviewPath = request.source_root_path || request.source_directory_path
  if (!overviewPath) throw new Error('缺少导出服务目录路径')
  const overview = await getDirectoryOverview(overviewPath)
  const tasks = [
    ...(overview.scheduled_function_tasks || []),
    ...(overview.scheduled_agent_tasks || []),
  ].filter(item => isTaskInsideExportNodes(item, nodes))
  exportBuiltinTasks.value = tasks.filter(item => item.builtin || item.origin === 'manifest')
  exportUserTasks.value = tasks.filter(item => !item.builtin && item.origin !== 'manifest')
  exportPreviewWarnings.value = overview.warnings || []
  selectedExportUserTaskIDs.value = []
  pendingCapabilityExport.value = {
    request,
    downloadPath,
    skippedCount,
    exitBulkAfterSuccess,
  }
  exportDialogVisible.value = true
}

async function confirmCapabilityExport() {
  const pending = pendingCapabilityExport.value
  if (!pending) return
  exportConfirming.value = true
  try {
    const bundle = await exportCapabilityBundle({
      ...pending.request,
      include_user_task_ids: selectedExportUserTaskIDs.value,
    })
    downloadCapabilityBundleFile(bundle, pending.downloadPath)
    ElMessage.success(
      pending.skippedCount > 0
        ? t('serviceTree.exportStartedSkipped', { count: pending.skippedCount })
        : t('serviceTree.exportStarted')
    )
    exportDialogVisible.value = false
    if (pending.exitBulkAfterSuccess) exitMultiSelectMode()
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('serviceTree.exportFailed')
    ElMessage.error(message)
  } finally {
    exportConfirming.value = false
  }
}

async function handleBulkExport() {
  const nodes = exportableSelectedNodes.value
  if (nodes.length === 0) {
    ElMessage.warning(t('serviceTree.exportSelectableWarning'))
    return
  }

  const skippedCount = selectedNodes.value.length - nodes.length
  bulkExporting.value = true
  try {
    await prepareCapabilityExport({
      source_root_path: rootFullCodePath.value,
      source_directory_paths: nodes.map((node) => node.full_code_path as string),
      name: buildBulkExportName(nodes)
    }, nodes, rootFullCodePath.value, skippedCount, true)
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('serviceTree.exportFailed')
    ElMessage.error(message)
  } finally {
    bulkExporting.value = false
  }
}

function handleBulkDelete() {
  const nodes = deletableSelectedNodes.value
  if (nodes.length === 0) {
    ElMessage.warning(t('serviceTree.deleteSelectableWarning'))
    return
  }

  emit('bulk-delete', nodes)
  exitMultiSelectMode()
}

const handleExportJson = async (data: ServiceTree) => {
  if (data.type !== 'package') {
    ElMessage.warning(t('serviceTree.exportOnlyDirectory'))
    return
  }

  if (!data.full_code_path) {
    ElMessage.warning(t('serviceTree.pathMissingRefresh'))
    return
  }

  try {
    await prepareCapabilityExport({
      source_directory_path: data.full_code_path,
      name: data.name || data.code
    }, [data], data.full_code_path)
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || t('serviceTree.exportFailed')
    ElMessage.error(message)
  }
}

const handleNodeAction = (command: ServiceTreeNodeActionCommand, data: ServiceTree) => {
  const action = getNodeActions(data).find(item => item.command === command)
  if (!action || action.disabled) {
    if (action?.disabledReason) ElMessage.warning(action.disabledReason)
    return
  }
  switch (command) {
    case 'create-directory':
      emit('create-directory', data)
      return
    case 'create-docs':
      emit('create-docs', data)
      return
    case 'rename':
      handleRename(data)
      return
    case 'copy':
      handleCopy(data)
      return
    case 'export-json':
      void handleExportJson(data)
      return
    case 'import-directory':
      openDirectoryImportDialog(data)
      return
    case 'paste':
      if (data.type === 'package') {
        handlePaste(data)
      } else {
        handlePaste()
      }
      return
    case 'delete-function':
      emit('delete-function', data)
      return
    case 'delete-doc':
      emit('delete-doc', data)
      return
    case 'delete-directory':
      emit('delete-directory', data)
      return
    case 'manage-access':
      openAccessPage(data)
      return
    case 'update-history':
      emit('update-history', data)
      return
    case 'open-workstation':
      eventBus.emit('workspace:open-workstation', { full_code_path: data.full_code_path || '' })
      return
    default: {
      const exhaustive: never = command
      return exhaustive
    }
  }
}

// 暴露方法给父组件
defineExpose({
  treeRef,
  expandPaths
})
</script>

<style scoped>
.service-tree-panel {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
}

.export-task-section {
  margin-top: 16px;
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
}

.export-preview-warning {
  margin-top: 12px;
}

.export-task-section-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.export-task-section-head span {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.export-task-list {
  display: grid;
  max-height: 190px;
  gap: 6px;
  overflow: auto;
}

.export-task-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
  padding: 9px 10px;
  border-radius: 9px;
  background: var(--el-fill-color-lighter);
  cursor: pointer;
}

.export-task-row.is-locked {
  cursor: default;
}

.export-task-main {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 3px;
  overflow: hidden;
}

.export-task-main > span,
.export-task-main small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.export-task-main small {
  color: var(--el-text-color-secondary);
}

.tree-header {
  padding: 10px 12px;
  border-bottom: 1px solid var(--el-border-color-light);
  display: flex;
  flex-direction: column;
  gap: 8px;

  .tree-primary-row {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .tree-search-input {
    flex: 1 1 auto;
    min-width: 0;
  }

  /* 填充式搜索框：柔和底色、无生硬边框、圆角与图标按钮(8px)一致、高度对齐 32px */
  .tree-search-input :deep(.el-input__wrapper) {
    height: 32px;
    padding: 0 10px;
    border-radius: 8px;
    background-color: var(--el-fill-color-light);
    border: 1px solid transparent;
    box-shadow: none;
    transition: background-color 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
  }

  .tree-search-input :deep(.el-input__wrapper:hover) {
    background-color: var(--el-fill-color);
    border-color: var(--border-light);
  }

  .tree-search-input :deep(.el-input__wrapper.is-focus) {
    background-color: var(--el-bg-color);
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px rgba(var(--color-primary-rgb), 0.12);
  }

  .tree-search-input :deep(.el-input__inner) {
    height: 30px;
    line-height: 30px;
    font-size: 13px;
    color: var(--text-primary);
  }

  .tree-search-input :deep(.el-input__inner::placeholder) {
    color: var(--text-disabled);
  }

  .tree-search-input :deep(.el-input__prefix) {
    color: var(--text-disabled);
    margin-right: 2px;
  }

  .tree-search-input :deep(.el-input__prefix .el-icon) {
    font-size: 15px;
  }

  .tree-search-input :deep(.el-input__suffix) {
    color: var(--text-secondary);
  }

  /* 聚焦时搜索图标点亮为主色，强化“正在搜索”的反馈 */
  .tree-search-input :deep(.el-input__wrapper.is-focus .el-input__prefix) {
    color: var(--color-primary);
  }

  .tree-tools-button {
    flex: 0 0 32px;
    width: 32px;
    min-width: 32px;
    height: 32px;
    padding: 0;
    border-radius: 8px;
  }

  .tree-bulk-toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
    min-height: 28px;
    min-width: 0;
  }

  .bulk-selected-count {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    margin-right: auto;
  }
  
  .header-actions {
    display: flex;
    align-items: center;
    gap: 16px;
  }
  
  .header-link {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s;
    color: #6366f1 !important; /* ✅ 与服务目录 fx 图标颜色一致（indigo-500） */
    
    &:hover {
      color: #1677ff !important; /* indigo-600，更深的紫色 */
      opacity: 1;
    }
    
    .el-icon {
      font-size: 14px;
      color: inherit;
    }
  }
}

.tree-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: visible; /* 确保下拉菜单不被裁剪 */
  padding: 8px;
  padding-bottom: 16px;
  display: flex;
  flex-direction: column;
  position: relative; /* 确保下拉菜单定位正确 */
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.node-more-actions {
  .node-more-button {
    width: 30px;
    min-width: 30px;
    height: 30px;
    padding: 0;
    border-radius: 8px;
    color: var(--el-text-color-secondary);
    font-size: 18px;

    &:hover {
      background: var(--el-fill-color-light);
      color: var(--el-color-primary);
    }
  }
}

:deep(.el-tree-node__content) {
  height: 32px;
  padding: 0 8px;
  margin: 2px 8px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  position: relative; /* 确保下拉菜单定位正确 */
  overflow: visible; /* 确保下拉菜单不被裁剪 */
  transition: background-color 0.15s ease;
  
  &:hover {
    background-color: var(--bg-tertiary);
  }
}

:deep(.service-tree--bulk-selecting .el-tree-node__content) {
  cursor: pointer;
}

:deep(.service-tree--bulk-selecting .el-checkbox) {
  margin-right: 6px;
}

:deep(.el-tree-node.is-current > .el-tree-node__content) {
  background-color: var(--el-fill-color) !important;
  color: var(--color-primary);
  font-weight: 500;
}

/* 确保子节点不受父节点选中状态影响 */
:deep(.el-tree-node.is-current .el-tree-node__children .el-tree-node__content) {
  background-color: transparent;
  color: var(--text-primary);
  font-weight: 400;
  
  &:hover {
    background-color: var(--bg-tertiary);
  }
}

/* 右键/三点菜单样式 */
:deep(.el-dropdown-menu),
:global(.service-tree-contextmenu-popper .el-dropdown-menu),
:global(.service-tree-dropdown-popper .el-dropdown-menu) {
  min-width: 160px;
  z-index: var(--aos-z-floating-popper) !important;
}

:deep(.el-dropdown-menu__item),
:global(.service-tree-contextmenu-popper .el-dropdown-menu__item),
:global(.service-tree-dropdown-popper .el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  white-space: nowrap;
  .el-icon { font-size: 14px; }
}
</style>

<!-- 科幻风格右键菜单（全局样式，popper teleported 到 body） -->
<style lang="scss">
.service-tree-contextmenu-popper {
  &.el-popper {
    padding: 4px !important;
    background: var(--app-shell-panel-bg-strong, var(--bg-secondary)) !important;
    border: 1px solid var(--app-shell-panel-border, var(--border-base)) !important;
    box-shadow: var(--app-shell-panel-shadow-soft, var(--box-shadow-lg)) !important;
    border-radius: 10px !important;
    overflow: hidden;
    position: relative;
    backdrop-filter: blur(8px);
  }

  .el-dropdown-menu {
    background: transparent !important;
    border: none !important;
    padding: 4px 0 !important;
    min-width: 180px !important;
    position: relative;
    z-index: 1;
  }

  .el-dropdown-menu__item {
    color: var(--text-primary) !important;
    padding: 8px 12px !important;
    margin: 2px 4px !important;
    border-radius: 4px !important;
    transition: all 0.15s ease !important;

    .el-icon {
      color: var(--text-secondary) !important;
      opacity: 0.9;
    }

    &:not(.is-disabled):hover {
      background: color-mix(in srgb, var(--el-color-primary) 10%, transparent) !important;
      color: var(--color-primary) !important;

      .el-icon {
        color: var(--color-primary) !important;
        opacity: 1;
      }
    }

    &.is-disabled {
      color: var(--text-disabled) !important;
      opacity: 0.5;
    }
  }

  /* 分隔线 */
  .el-dropdown-menu__item--divided {
    margin-top: 4px !important;
    border-top: 1px solid var(--border-light) !important;
    &::before {
      display: none !important;
    }
  }
}

.service-tree-tools-popper.el-popper {
  padding: 6px !important;
  background: var(--app-shell-panel-bg-strong, var(--el-bg-color)) !important;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-light)) !important;
  border-radius: 10px !important;
  box-shadow: var(--app-shell-panel-shadow-soft, var(--el-box-shadow-light)) !important;

  .tree-tools-menu {
    display: grid;
    gap: 3px;
  }

  .tree-tool-action {
    display: flex;
    width: 100%;
    align-items: center;
    gap: 9px;
    padding: 9px 10px;
    border: 0;
    border-radius: 7px;
    background: transparent;
    color: var(--el-text-color-primary);
    font: inherit;
    text-align: left;
    cursor: pointer;

    &:hover:not(:disabled) {
      background: var(--el-fill-color-light);
      color: var(--el-color-primary);
    }

    &:disabled {
      color: var(--el-text-color-disabled);
      cursor: not-allowed;
    }
  }

  .tree-tool-preference {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-top: 3px;
    padding: 10px;
    border-top: 1px solid var(--el-border-color-lighter);

    > span {
      display: grid;
      min-width: 0;
      gap: 3px;
    }

    strong {
      color: var(--el-text-color-primary);
      font-size: 13px;
      font-weight: 600;
    }

    small {
      color: var(--el-text-color-secondary);
      font-size: 11px;
      line-height: 1.35;
    }
  }
}

</style>
