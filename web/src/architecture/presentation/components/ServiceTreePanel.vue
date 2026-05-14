<template>
  <div class="service-tree-panel" data-testid="service-tree-panel" v-loading="loading">
    <div class="tree-header">
      <div class="tree-primary-row">
        <el-input
          v-model="searchKeyword"
          class="tree-search-input"
          size="small"
          placeholder="搜索目录或名称…"
          clearable
          :prefix-icon="Search"
          data-testid="service-tree-search"
        />
        <el-tooltip v-if="!multiSelectMode" content="多选" placement="bottom">
          <el-button
            class="tree-select-button"
            size="small"
            :icon="Select"
            aria-label="多选"
            @click="enterMultiSelectMode"
          />
        </el-tooltip>
      </div>
      <div v-if="multiSelectMode" class="tree-bulk-toolbar">
        <span class="bulk-selected-count">已选 {{ selectedNodeCount }}</span>
        <el-button
          v-if="featureFlags.capabilityBundle"
          size="small"
          :icon="Download"
          :loading="bulkExporting"
          :disabled="exportableSelectedNodes.length === 0"
          @click="handleBulkExport"
        >
          导出
        </el-button>
        <el-button
          size="small"
          type="danger"
          plain
          :icon="Delete"
          :disabled="deletableSelectedNodes.length === 0"
          @click="handleBulkDelete"
        >
          删除
        </el-button>
        <el-button size="small" text :icon="Close" @click="exitMultiSelectMode">
          取消
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
            <span
              class="tree-node"
              :data-testid="`service-tree-node-${data.id}`"
              :data-node-id="String(data.id)"
              :data-node-type="data.type"
              :data-root-node="isRootNode(data) ? 'true' : 'false'"
              :class="{ 'tree-node-draggable': !multiSelectMode && (data.type === 'function' || data.type === 'package') }"
              :draggable="!multiSelectMode && (data.type === 'function' || data.type === 'package')"
              @dragstart="onTreeNodeDragStart($event, data)"
              @contextmenu.prevent
              :title="multiSelectMode ? '点击选择' : '右键显示菜单'"
            >
            <!-- 根节点：使用工作空间图标（package 类型且为根节点） -->
            <img 
              v-if="data.type === 'package' && isRootNode(data)" 
              src="/service-tree/custom-folder.svg" 
              alt="工作空间" 
              class="node-icon app-icon-img"
              :class="getNodeIconClass(data)"
            />
            <!-- package 类型：统一使用目录图标 -->
            <img 
              v-else-if="data.type === 'package'" 
              src="/service-tree/custom-folder.svg" 
              alt="目录" 
              class="node-icon package-icon-img"
              :class="getNodeIconClass(data)"
            />
            <!-- function 类型：根据 template_type 显示不同图标 -->
            <template v-else-if="data.type === 'function'">
              <!-- 表单类型：使用编辑图标 -->
              <img 
                v-if="data.template_type === TEMPLATE_TYPE.FORM"
                src="/service-tree/编辑.svg" 
                alt="表单" 
                class="node-icon form-icon-img"
                :class="getNodeIconClass(data)"
              />
              <!-- 其他类型：使用组件图标 -->
              <el-icon v-else 
                       class="node-icon" 
                       :class="getNodeIconClass(data)">
                <component :is="getFunctionIcon(data)" />
              </el-icon>
            </template>
            <!-- docs 类型：使用文档图标 -->
            <img 
              v-else-if="data.type === 'docs'" 
              src="/文档.svg" 
              alt="文档" 
              class="node-icon docs-icon-img"
              :class="getNodeIconClass(data)"
            />
            <!-- board 类型：讨论区图标 -->
            <img 
              v-else-if="data.type === 'board'" 
              src="/讨论区.svg" 
              alt="讨论区" 
              class="node-icon board-icon-img"
              :class="getNodeIconClass(data)"
            />
            <!-- 其他类型：显示 fx 文本 -->
            <span v-else class="node-icon fx-icon" :class="getNodeIconClass(data)">fx</span>
            <span class="node-label">{{ node.label }}</span>

            <!-- 运行态 badge：来自 agent-server state 接口，表示当前目录及子目录正在运行的会话数 -->
            <el-badge
              v-if="hasRuntimeBadge(data)"
              :value="getRuntimeBadgeText(data)"
              :max="99"
              :class="getRuntimeBadgeClass(data)"
              :title="getRuntimeSummaryTitle(data)"
            />

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
              <el-icon class="more-icon" :data-testid="`service-tree-more-${data.id}`" @click.stop><MoreFilled /></el-icon>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item
                    v-for="action in getNodeActions(data)"
                    :key="action.command"
                    :data-testid="buildServiceTreeNodeActionTestId(action.command, data)"
                    :command="action.command"
                  >
                    <el-icon><component :is="action.icon" /></el-icon>
                    {{ action.label }}
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  v-for="action in getNodeActions(data)"
                  :key="`context-${action.command}`"
                  :command="action.command"
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
    <input
      ref="capabilityImportInputRef"
      type="file"
      accept=".json,application/json"
      class="capability-import-input"
      @change="handleCapabilityImportFileChange"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { MoreFilled, Document, Download, Delete, Search, Select, Close } from '@element-plus/icons-vue'
import ChartIcon from '@/architecture/presentation/shared/components/icons/ChartIcon.vue'
import TableIcon from '@/architecture/presentation/shared/components/icons/TableIcon.vue'
import FormIcon from '@/architecture/presentation/shared/components/icons/FormIcon.vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import type { ServiceTree } from '@/architecture/domain/types'
import { isRootNode } from '@/architecture/runtime/utils/tree-utils'
import { TEMPLATE_TYPE } from '@/architecture/runtime/utils/functionTypes'
import { exportCapabilityBundle, installCapabilityBundle, updatePackage, updateServiceTreeFunction, updateDocs } from '@/architecture/infrastructure/api/service-tree'
import { getRuntimeStateSummary, type RuntimeStateSummary } from '@/architecture/infrastructure/api/state'
import { downloadCapabilityBundleFile, parseCapabilityBundleJson } from '@/architecture/runtime/utils/directoryBundleFile'
import { eventBus, WorkspaceEvent } from '@/architecture/infrastructure/eventBus'
import { useServiceTreeClipboard } from '../composables/useServiceTreeClipboard'
import { useServiceTreeSearchExpand } from '../composables/useServiceTreeSearchExpand'
import {
  buildServiceTreeNodeActionTestId,
  getServiceTreeNodeActions,
  type ServiceTreeNodeActionCommand
} from '../utils/serviceTreeNodeActions'
import { featureFlags } from '@/architecture/infrastructure/config/features'

interface Props {
  treeData: ServiceTree[]
  loading?: boolean
  currentNodeId?: number | string | null
  currentFunction?: ServiceTree | null  // 当前选中的节点（用于判断是否可以克隆）
  expandedKeys?: number[] // ⭐ 需要自动展开的节点ID列表（从后端返回）
}

interface Emits {
  (e: 'node-click', node: ServiceTree): void
  (e: 'create-directory', parentNode?: ServiceTree): void
  (e: 'create-docs', parentNode?: ServiceTree): void
  (e: 'create-board', parentNode?: ServiceTree): void
  (e: 'delete-doc', node: ServiceTree): void
  (e: 'delete-board', node: ServiceTree): void
  (e: 'delete-function', node: ServiceTree): void  // 删除函数
  (e: 'delete-directory', node: ServiceTree): void  // 删除目录（非根 package）
  (e: 'bulk-delete', nodes: ServiceTree[]): void
  (e: 'refresh-tree'): void  // 刷新树（复制粘贴后需要刷新）
  (e: 'update-history', node?: ServiceTree): void  // 显示变更记录（工作空间或目录）
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const runtimeSummaries = ref<Record<string, RuntimeStateSummary>>({})
let runtimeSummaryTimer: ReturnType<typeof setInterval> | null = null

const {
  treeRef,
  groupedTreeData,
  searchKeyword,
  filterNodeMethod,
  defaultExpandedKeysWithWorkspace,
  expandedKeysState,
  treeKey,
  expandPaths
} = useServiceTreeSearchExpand({
  treeData: computed(() => props.treeData),
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

const rootFullCodePath = computed(() => props.treeData[0]?.full_code_path || '')
const multiSelectMode = ref(false)
const selectedNodes = ref<ServiceTree[]>([])
const bulkExporting = ref(false)
const capabilityImportInputRef = ref<HTMLInputElement | null>(null)
const capabilityImportTargetNode = ref<ServiceTree | null>(null)
let unsubscribeRuntimeRefresh: (() => void) | null = null

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

const startRuntimeSummaryPolling = () => {
  stopRuntimeSummaryPolling()
  if (!rootFullCodePath.value) return
  runtimeSummaryTimer = setInterval(refreshRuntimeSummary, 3000)
  unsubscribeRuntimeRefresh = eventBus.on(WorkspaceEvent.scheduledAgentTaskCreated, refreshRuntimeSummary)
  window.addEventListener('focus', refreshRuntimeSummary)
}

watch(rootFullCodePath, () => {
  runtimeSummaries.value = {}
  exitMultiSelectMode()
  refreshRuntimeSummary()
  startRuntimeSummaryPolling()
}, { immediate: true })

onBeforeUnmount(stopRuntimeSummaryPolling)

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
  const parts = [`运行中 ${summary.running_count}`]
  if (summary.thinking_count > 0) parts.push(`思考中 ${summary.thinking_count}`)
  if (summary.tool_running_count > 0) parts.push(`工具执行 ${summary.tool_running_count}`)
  if (summary.scheduled_running_count > 0) parts.push(`定时会话 ${summary.scheduled_running_count}`)
  if (summary.failed_recent_count > 0) parts.push(`最近失败 ${summary.failed_recent_count}`)
  return parts.join('，')
}

// 重命名目录
const handleRename = async (node: ServiceTree) => {
  if (node.type !== 'package') {
    ElMessage.warning('只能重命名目录')
    return
  }
  
  try {
    const { value: newName } = await ElMessageBox.prompt(
      `请输入新的名称（当前：${node.name}）`,
      '重命名目录',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputPattern: /^.+$/,
        inputErrorMessage: '名称不能为空',
        inputValue: node.name
      }
    )
    
    if (!newName || newName.trim() === '') {
      ElMessage.warning('名称不能为空')
      return
    }
    
    const trimmedName = newName.trim()
    
    // 如果名称没有变化，直接返回
    if (trimmedName === node.name) {
      return
    }
    
    try {
      // ⭐ 根据节点类型调用对应的更新接口
      if (node.type === 'package') {
        await updatePackage(node.id, { name: trimmedName })
      } else if (node.type === 'function') {
        await updateServiceTreeFunction(node.id, { name: trimmedName })
      } else if (node.type === 'docs') {
        await updateDocs(node.id, { name: trimmedName })
      } else {
        ElMessage.warning('不支持的节点类型')
        return
      }
      ElMessage.success('重命名成功')
      
      // 刷新树
      emit('refresh-tree')
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || '重命名失败'
      ElMessage.error(errorMessage)
    }
  } catch (error) {
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
  return node.type === 'package' || node.type === 'function'
}

function canDeleteNode(node: ServiceTree): boolean {
  if (!node.full_code_path) return false
  if (node.type === 'package') {
    return !isRootNode(node)
  }
  return node.type === 'function' || node.type === 'docs' || node.type === 'board'
}

function buildBulkExportName(nodes: ServiceTree[]): string {
  const firstNode = nodes[0]
  if (nodes.length === 1 && firstNode) {
    return firstNode.name || firstNode.code || firstNode.full_code_path?.split('/').filter(Boolean).pop() || 'capability'
  }
  return props.treeData[0]?.name || 'capability'
}

async function handleBulkExport() {
  const nodes = exportableSelectedNodes.value
  if (nodes.length === 0) {
    ElMessage.warning('请选择可导出的目录或函数')
    return
  }

  const skippedCount = selectedNodes.value.length - nodes.length
  bulkExporting.value = true
  try {
    const bundle = await exportCapabilityBundle({
      source_root_path: rootFullCodePath.value,
      source_directory_paths: nodes.map((node) => node.full_code_path as string),
      name: buildBulkExportName(nodes)
    })
    downloadCapabilityBundleFile(bundle, rootFullCodePath.value)
	    ElMessage.success(skippedCount > 0 ? `已开始下载能力包 JSON 文件，跳过 ${skippedCount} 个不可导出节点` : '已开始下载能力包 JSON 文件')
    exitMultiSelectMode()
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '导出失败'
    ElMessage.error(message)
  } finally {
    bulkExporting.value = false
  }
}

function handleBulkDelete() {
  const nodes = deletableSelectedNodes.value
  if (nodes.length === 0) {
    ElMessage.warning('请选择可删除的节点')
    return
  }

  emit('bulk-delete', nodes)
  exitMultiSelectMode()
}

const handleExportJson = async (data: ServiceTree) => {
  if (data.type !== 'package') {
    ElMessage.warning('只能导出目录')
    return
  }

  if (!data.full_code_path) {
    ElMessage.warning('无法获取目录路径，请刷新后重试')
    return
  }

  try {
    const bundle = await exportCapabilityBundle({
      source_directory_path: data.full_code_path,
      name: data.name || data.code
    })
    downloadCapabilityBundleFile(bundle, data.full_code_path)
	    ElMessage.success('已开始下载能力包 JSON 文件')
  } catch (error: any) {
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '导出失败'
    ElMessage.error(message)
  }
}

function requestCapabilityJsonImport(data: ServiceTree) {
  if (data.type !== 'package') {
    ElMessage.warning('只能导入到目录')
    return
  }
  if (!data.full_code_path) {
    ElMessage.warning('无法获取目录路径，请刷新后重试')
    return
  }
  capabilityImportTargetNode.value = data
  if (capabilityImportInputRef.value) {
    capabilityImportInputRef.value.value = ''
    capabilityImportInputRef.value.click()
  }
}

async function handleCapabilityImportFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  const targetNode = capabilityImportTargetNode.value
  capabilityImportTargetNode.value = null

  if (!file || !targetNode?.full_code_path) {
    return
  }

  try {
    const bundle = parseCapabilityBundleJson(await file.text())
    await ElMessageBox.confirm(
      `将能力包「${bundle.name || file.name}」导入到 ${targetNode.full_code_path}，同名文件会被覆盖。`,
      '导入能力包',
      {
        confirmButtonText: '覆盖导入',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    const resp = await installCapabilityBundle({
      target_directory_path: targetNode.full_code_path,
      overwrite: true,
      force_diff: true,
      bundle
    })
    ElMessage.success(resp.message || '导入成功')
    emit('refresh-tree')
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') {
      return
    }
    const message = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '导入失败'
    ElMessage.error(message)
  }
}

const handleNodeAction = (command: ServiceTreeNodeActionCommand, data: ServiceTree) => {
  switch (command) {
    case 'create-directory':
      emit('create-directory', data)
      return
    case 'create-docs':
      emit('create-docs', data)
      return
    case 'create-board':
      emit('create-board', data)
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
    case 'import-json':
      requestCapabilityJsonImport(data)
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
    case 'delete-board':
      emit('delete-board', data)
      return
    case 'delete-directory':
      emit('delete-directory', data)
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

// 获取函数图标组件（根据 template_type）
const getFunctionIcon = (data: ServiceTree) => {
  if (data.template_type === TEMPLATE_TYPE.TABLE) {
    return TableIcon
  } else if (data.template_type === TEMPLATE_TYPE.FORM) {
    return FormIcon
  } else if (data.template_type === TEMPLATE_TYPE.CHART) {
    return ChartIcon
  }
  // 默认使用 Document 图标（如果没有 template_type 或不是已知类型）
  return Document
}

// 获取节点图标样式类
const getNodeIconClass = (data: ServiceTree) => {
  if (data.type === 'package') {
    return 'package-icon'
  } else if (data.type === 'function') {
    // 根据 template_type 返回不同的样式类
    if (data.template_type === TEMPLATE_TYPE.TABLE) {
      return 'table-icon'
    } else if (data.template_type === TEMPLATE_TYPE.FORM) {
      return 'form-icon'
    } else if (data.template_type === TEMPLATE_TYPE.CHART) {
      return 'chart-icon'
    }
    return 'function-icon'
  } else if (data.type === 'docs') {
    return 'docs-icon'
  } else if (data.type === 'board') {
    return 'board-icon'
  }
  return 'function-icon'
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

  .tree-search-input :deep(.el-input__wrapper) {
    min-height: 34px;
    border: 1px solid rgba(var(--el-color-primary-rgb), 0.14);
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.045);
    box-shadow: none;
    backdrop-filter: blur(14px) saturate(1.15);
    -webkit-backdrop-filter: blur(14px) saturate(1.15);
    transition: border-color 0.18s ease, background 0.18s ease, box-shadow 0.18s ease;
  }

  .tree-search-input :deep(.el-input__wrapper:hover) {
    border-color: rgba(var(--el-color-primary-rgb), 0.24);
    background: rgba(255, 255, 255, 0.065);
  }

  .tree-search-input :deep(.el-input__wrapper.is-focus) {
    border-color: rgba(var(--el-color-primary-rgb), 0.38);
    background: rgba(255, 255, 255, 0.075);
    box-shadow: 0 0 0 3px rgba(var(--el-color-primary-rgb), 0.08);
  }

  .tree-search-input :deep(.el-input__inner) {
    color: var(--el-text-color-primary);
    background: transparent;
  }

  .tree-search-input :deep(.el-input__inner::placeholder) {
    color: var(--el-text-color-placeholder);
  }

  .tree-search-input :deep(.el-input__prefix),
  .tree-search-input :deep(.el-input__suffix) {
    color: var(--el-text-color-secondary);
  }

  .tree-select-button {
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

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  width: 100%;
  min-width: 0; /* ⭐ 允许 flexbox 子元素正确收缩 */
  
  &.tree-node-draggable {
    cursor: grab;
    &:active {
      cursor: grabbing;
    }
  }
  
  .node-icon {
    width: 16px;
    height: 16px;
    margin-right: 8px;
    color: #6366f1;  /* ✅ 统一主题色（indigo-500） */
    opacity: 0.8;
    flex-shrink: 0;
    transition: color 0.2s ease;
    
    &.package-icon {
      color: #6366f1;
      opacity: 0.8;
    }
    
    &.package-icon-img {
      width: 16px;
      height: 16px;
      object-fit: contain;
      opacity: 0.9;
    }
    
    &.table-icon {
      color: #10b981; /* green-500 - 表格用绿色 */
      opacity: 0.9;
    }
    
    &.form-icon {
      color: #3b82f6; /* blue-500 - 表单用蓝色 */
      opacity: 0.9;
    }
    
    &.form-icon-img {
      width: 16px;
      height: 16px;
      object-fit: contain;
      opacity: 0.9;
    }
    
    &.board-icon-img {
      width: 16px;
      height: 16px;
      object-fit: contain;
      opacity: 0.9;
    }
    
    &.function-icon {
      color: #6366f1; /* indigo-500 - 默认函数图标 */
      opacity: 0.8;
    }
    
    &.fx-icon {
      font-size: 12px;
      font-weight: 600;
      font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
      font-style: italic;
      color: #6366f1;
      opacity: 0.8;
    }
    
    &.group-icon {
      color: #909399;
      opacity: 0.9;
    }
    
    &.group-icon-img {
      width: 16px;
      height: 16px;
      object-fit: contain;
      opacity: 0.9;
    }
  }
  
  .group-label {
    font-weight: 500;
    color: var(--el-text-color-regular);
  }
  
  .group-tag {
    margin-left: 8px;
    font-size: 11px;
  }
  
  .node-label {
    font-size: 14px;
    color: var(--el-text-color-primary);
    flex: 1;
    min-width: 0; /* ⭐ 允许文本正确收缩并显示省略号 */
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    
  }
  
  /* ⭐ 待审批数量 badge - 防止被挤压 */
  .pending-count-badge {
    flex-shrink: 0;
    margin-left: 6px;
    cursor: pointer;
  }

  .runtime-state-badge {
    flex-shrink: 0;
    margin-left: 6px;
  }

  .runtime-state-badge :deep(.el-badge__content) {
    border: none;
    background: #0ea5e9;
    box-shadow: 0 0 0 2px rgba(14, 165, 233, 0.12);
  }

  .runtime-state-badge-thinking :deep(.el-badge__content) {
    background: #38bdf8;
    box-shadow: 0 0 12px rgba(56, 189, 248, 0.45);
  }

  .runtime-state-badge-tool :deep(.el-badge__content) {
    background: #f59e0b;
    box-shadow: 0 0 12px rgba(245, 158, 11, 0.42);
  }

  .runtime-state-badge-approval :deep(.el-badge__content) {
    background: #a855f7;
    box-shadow: 0 0 12px rgba(168, 85, 247, 0.42);
  }

  .runtime-state-badge-failed :deep(.el-badge__content) {
    background: #ef4444;
    box-shadow: 0 0 12px rgba(239, 68, 68, 0.42);
  }

  .node-more-actions {
    flex-shrink: 0;
    margin-left: auto; /* 靠右对齐 */
    opacity: 0;
    transition: opacity 0.2s;
    .more-icon {
      font-size: 14px;
      color: var(--el-text-color-secondary);
      cursor: pointer;
      padding: 4px;
      &:hover { color: var(--el-color-primary); }
    }
  }
  &:hover .node-more-actions { opacity: 1; }
}

:deep(.el-tree-node__content) {
  height: 32px;
  padding: 0 8px;
  display: flex;
  align-items: center;
  position: relative; /* 确保下拉菜单定位正确 */
  overflow: visible; /* 确保下拉菜单不被裁剪 */
  
  &:hover {
    background-color: var(--el-fill-color-light);
    .tree-node .node-more-actions { opacity: 1; }
  }
}

:deep(.service-tree--bulk-selecting .el-tree-node__content) {
  cursor: pointer;
}

:deep(.service-tree--bulk-selecting .el-checkbox) {
  margin-right: 6px;
}

:deep(.el-tree-node.is-current > .el-tree-node__content) {
  background-color: rgba(99, 102, 241, 0.15) !important;
  border-left: 2px solid #6366f1;
  
  .tree-node {
    .node-label {
      color: var(--el-text-color-primary);
      font-weight: 500;
    }
    
    .node-icon {
      color: #6366f1;
      opacity: 0.8;
    }
    .node-more-actions { opacity: 1 !important; }
  }
}

/* 确保子节点不受父节点选中状态影响 */
:deep(.el-tree-node.is-current .el-tree-node__children .el-tree-node__content) {
  background-color: transparent;
  border-left: none;
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
  --ctx-bg: #0d1321;
  --ctx-bg-hover: rgba(0, 212, 255, 0.12);
  --ctx-border: rgba(0, 212, 255, 0.4);
  --ctx-glow: #00d4ff;
  --ctx-glow-rgb: 0, 212, 255;
  --ctx-text: #e2e8f0;
  --ctx-text-muted: #94a3b8;

  &.el-popper {
    padding: 0 !important;
    background: var(--ctx-bg) !important;
    border: 1px solid var(--ctx-border) !important;
    box-shadow: 0 0 24px rgba(var(--ctx-glow-rgb), 0.2), 0 4px 20px rgba(0, 0, 0, 0.4) !important;
    border-radius: 8px !important;
    overflow: hidden;
    position: relative;

    /* 顶部流光条 */
    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 2px;
      background: linear-gradient(90deg, transparent, var(--ctx-glow), transparent);
      opacity: 0.7;
    }

    /* 淡网格背景 */
    &::after {
      content: '';
      position: absolute;
      inset: 0;
      background-image: linear-gradient(rgba(var(--ctx-glow-rgb), 0.03) 1px, transparent 1px),
        linear-gradient(90deg, rgba(var(--ctx-glow-rgb), 0.03) 1px, transparent 1px);
      background-size: 12px 12px;
      pointer-events: none;
    }
  }

  .el-dropdown-menu {
    background: transparent !important;
    border: none !important;
    padding: 8px 0 !important;
    min-width: 180px !important;
    position: relative;
    z-index: 1;
  }

  .el-dropdown-menu__item {
    color: var(--ctx-text) !important;
    padding: 10px 16px !important;
    margin: 0 4px !important;
    border-radius: 6px !important;
    transition: all 0.2s ease !important;

    .el-icon {
      color: var(--ctx-glow) !important;
      opacity: 0.9;
    }

    &:not(.is-disabled):hover {
      background: var(--ctx-bg-hover) !important;
      color: var(--ctx-glow) !important;
      box-shadow: inset 0 0 12px rgba(var(--ctx-glow-rgb), 0.08) !important;

      .el-icon {
        color: var(--ctx-glow) !important;
        opacity: 1;
      }
    }

    &.is-disabled {
      color: var(--ctx-text-muted) !important;
      opacity: 0.5;
    }
  }

  /* 分隔线 */
  .el-dropdown-menu__item--divided {
    margin-top: 4px;
    padding-top: 4px;
    border-top: 1px solid rgba(var(--ctx-glow-rgb), 0.2);
  }
}

.capability-import-input {
  display: none;
}
</style>
