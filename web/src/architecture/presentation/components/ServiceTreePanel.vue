<template>
  <div class="service-tree-panel" data-testid="service-tree-panel" v-loading="loading">
    <div class="tree-header">
      <el-input
        v-model="searchKeyword"
        class="tree-search-input"
        placeholder="搜索目录或名称…"
        clearable
        :prefix-icon="Search"
        data-testid="service-tree-search"
      />
    </div>

    <div class="tree-content" data-testid="service-tree-content">
      <el-tree
        v-if="groupedTreeData.length > 0"
        :key="treeKey"
        ref="treeRef"
        :data="groupedTreeData"
        :props="{ children: 'children', label: 'name' }"
        node-key="id"
        :default-expand-all="false"
        :default-expanded-keys="defaultExpandedKeysWithWorkspace"
        :expanded-keys="expandedKeysState"
        :expand-on-click-node="false"
        :highlight-current="true"
        :filter-node-method="filterNodeMethod"
        @node-click="handleNodeClick"
      >
        <template #default="{ node, data }">
          <el-dropdown
            trigger="contextmenu"
            :teleported="true"
            popper-class="service-tree-contextmenu-popper"
            @command="(command: string) => handleNodeAction(command, data)"
          >
            <span
              class="tree-node"
              :data-testid="`service-tree-node-${data.id}`"
              :data-node-id="String(data.id)"
              :data-node-type="data.type"
              :data-root-node="isRootNode(data) ? 'true' : 'false'"
              :class="{ 'tree-node-draggable': data.type === 'function' || data.type === 'package' }"
              :draggable="data.type === 'function' || data.type === 'package'"
              @dragstart="onTreeNodeDragStart($event, data)"
              @contextmenu.prevent
              :title="'右键显示菜单'"
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
            <span class="node-label" :class="{ 'no-permission': !hasAnyPermissionForNode(data) }">{{ node.label }}</span>
            
            <!-- 无权限标识 - 没有权限的节点显示 -->
            <img 
              v-if="!hasAnyPermissionForNode(data)" 
              src="/锁定.svg" 
              alt="无权限" 
              class="no-permission-icon" 
              :title="'该节点没有权限，点击申请权限'"
              @click.stop="handleNoPermissionClick(data)"
            />
            
            <!-- ⭐ 待审批数量 badge - 仅管理员可见（package 和 function 类型都显示） -->
            <el-badge
              v-if="(data.type === 'package' || data.type === 'function') && isAdmin(data) && data.pending_count && data.pending_count > 0"
              :value="data.pending_count"
              :max="99"
              class="pending-count-badge"
              @click.stop="handlePendingCountClick(data)"
              :title="`有 ${data.pending_count} 个待审批的权限申请`"
            />
            
            <!-- 更多操作按钮 - 鼠标悬停时显示（与右键菜单并存，点击也可打开） -->
            <el-dropdown
              trigger="click"
              :teleported="true"
              popper-class="service-tree-contextmenu-popper"
              @click.stop
              class="node-more-actions"
              @command="(command: string) => handleNodeAction(command, data)"
            >
              <el-icon class="more-icon" :data-testid="`service-tree-more-${data.id}`" @click.stop><MoreFilled /></el-icon>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item :data-testid="`service-tree-action-apply-permission-${data.id}`" command="apply-permission"><el-icon><Key /></el-icon>申请权限</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'package' && hasPermission(data, DirectoryPermission.write)" :data-testid="`service-tree-action-create-directory-${data.id}`" command="create-directory"><el-icon><Plus /></el-icon>添加服务目录</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'package' && hasPermission(data, DirectoryPermission.write)" :data-testid="`service-tree-action-create-docs-${data.id}`" command="create-docs"><el-icon><Document /></el-icon>创建文档</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'package' && hasPermission(data, DirectoryPermission.write)" :data-testid="`service-tree-action-create-board-${data.id}`" command="create-board"><el-icon><ChatDotSquare /></el-icon>新增讨论区</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'package'" :data-testid="`service-tree-action-open-workstation-${data.id}`" command="open-workstation"><el-icon><ChatDotRound /></el-icon>打开工作台</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'package' && !isRootNode(data) && hasPermission(data, DirectoryPermission.delete)" :data-testid="`service-tree-action-delete-directory-${data.id}`" command="delete-directory"><el-icon><Delete /></el-icon>删除目录</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'package' && hasPermission(data, DirectoryPermission.update)" :data-testid="`service-tree-action-rename-${data.id}`" command="rename"><el-icon><Edit /></el-icon>重命名</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'package' && hasPermission(data, DirectoryPermission.read)" :data-testid="`service-tree-action-copy-${data.id}`" command="copy"><el-icon><CopyDocument /></el-icon>复制</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'package' && (copiedDirectory || copiedHubLink) && hasPermission(data, DirectoryPermission.write)" :data-testid="`service-tree-action-paste-${data.id}`" command="paste"><el-icon><DocumentChecked /></el-icon>粘贴</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'function' && hasPermission(data, TablePermission.delete)" :data-testid="`service-tree-action-delete-function-${data.id}`" command="delete-function"><el-icon><Delete /></el-icon>删除函数</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'docs' && hasPermission(data, DirectoryPermission.delete)" :data-testid="`service-tree-action-delete-doc-${data.id}`" command="delete-doc"><el-icon><Delete /></el-icon>删除文档</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'board' && hasPermission(data, DirectoryPermission.delete)" :data-testid="`service-tree-action-delete-board-${data.id}`" command="delete-board"><el-icon><Delete /></el-icon>删除讨论区</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'package' && hasPermission(data, DirectoryPermission.write)" :data-testid="`service-tree-action-import-go-files-${data.id}`" command="import-go-files"><el-icon><Download /></el-icon>导入 Go 文件</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'package' && !data.hub_full_code_path && hasPermission(data, DirectoryPermission.read)" :data-testid="`service-tree-action-publish-to-hub-${data.id}`" command="publish-to-hub"><el-icon><Upload /></el-icon>发布到 Hub</el-dropdown-item>
                  <el-dropdown-item v-if="data.type === 'package' && data.hub_full_code_path && hasPermission(data, DirectoryPermission.write)" :data-testid="`service-tree-action-push-to-hub-${data.id}`" command="push-to-hub"><el-icon><Upload /></el-icon>推送到 Hub</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                  <!-- 申请权限选项（对所有节点都显示） -->
                  <el-dropdown-item 
                    command="apply-permission"
                  >
                    <el-icon><Key /></el-icon>
                    申请权限
                  </el-dropdown-item>
                  
                  <!-- 对 package 类型显示创建子目录选项（包括根目录和普通目录，需要 directory:write 权限） -->
                  <el-dropdown-item 
                    v-if="data.type === 'package' && hasPermission(data, DirectoryPermission.write)" 
                    command="create-directory"
                  >
                    <el-icon><Plus /></el-icon>
                    添加服务目录
                  </el-dropdown-item>
                  
                  <!-- 创建文档选项（需要 directory:write 权限） -->
                  <el-dropdown-item 
                    v-if="data.type === 'package' && hasPermission(data, DirectoryPermission.write)" 
                    command="create-docs"
                  >
                    <el-icon><Document /></el-icon>
                    创建文档
                  </el-dropdown-item>
                  
                  <!-- 创建讨论区选项（需要 directory:write 权限） -->
                  <el-dropdown-item 
                    v-if="data.type === 'package' && hasPermission(data, DirectoryPermission.write)" 
                    command="create-board"
                  >
                    <el-icon><ChatDotSquare /></el-icon>
                    新增讨论区
                  </el-dropdown-item>
                  
                  <!-- 打开工作台（package 类型，含根目录） -->
                  <el-dropdown-item 
                    v-if="data.type === 'package'" 
                    command="open-workstation"
                  >
                    <el-icon><ChatDotRound /></el-icon>
                    打开工作台
                  </el-dropdown-item>
                  
                  <!-- 删除目录选项（仅对非根 package 类型，需要 directory:delete 权限） -->
                  <el-dropdown-item 
                    v-if="data.type === 'package' && !isRootNode(data) && hasPermission(data, DirectoryPermission.delete)" 
                    command="delete-directory"
                  >
                    <el-icon><Delete /></el-icon>
                    删除目录
                  </el-dropdown-item>
                  
                  <!-- 重命名选项（仅对 package 类型） -->
                  <el-dropdown-item 
                    v-if="data.type === 'package' && hasPermission(data, DirectoryPermission.update)" 
                    command="rename"
                  >
                    <el-icon><Edit /></el-icon>
                    重命名
                  </el-dropdown-item>
                  
                  <!-- 复制选项（仅对 package 类型） -->
                  <el-dropdown-item 
                    v-if="data.type === 'package' && hasPermission(data, DirectoryPermission.read)" 
                    command="copy"
                  >
                    <el-icon><CopyDocument /></el-icon>
                    复制
                  </el-dropdown-item>
                  
                  <!-- 粘贴选项（需要目标目录有 write 权限，且有已复制的内容） -->
                  <el-dropdown-item 
                    v-if="data.type === 'package' && (copiedDirectory || copiedHubLink) && hasPermission(data, DirectoryPermission.write)" 
                    command="paste"
                  >
                    <el-icon><DocumentChecked /></el-icon>
                    粘贴
                  </el-dropdown-item>
                  
                  <!-- 删除函数选项（仅对 function 类型） -->
                  <el-dropdown-item 
                    v-if="data.type === 'function' && hasPermission(data, TablePermission.delete)" 
                    command="delete-function"
                  >
                    <el-icon><Delete /></el-icon>
                    删除函数
                  </el-dropdown-item>
                  
                  <!-- 删除文档选项（仅对 docs 类型） -->
                  <el-dropdown-item 
                    v-if="data.type === 'docs' && hasPermission(data, DirectoryPermission.delete)" 
                    command="delete-doc"
                  >
                    <el-icon><Delete /></el-icon>
                    删除文档
                  </el-dropdown-item>
                  
                  <!-- 删除讨论区选项（仅对 board 类型） -->
                  <el-dropdown-item 
                    v-if="data.type === 'board' && hasPermission(data, DirectoryPermission.delete)" 
                    command="delete-board"
                  >
                    <el-icon><Delete /></el-icon>
                    删除讨论区
                  </el-dropdown-item>
                  
                  <!-- Hub 相关操作 -->
                  <!-- 导入 Go 文件：选择本地 .go 文件写入当前目录（与 write_go_file 一致） -->
                  <el-dropdown-item 
                    v-if="data.type === 'package' && hasPermission(data, DirectoryPermission.write)" 
                    command="import-go-files"
                  >
                    <el-icon><Download /></el-icon>
                    导入 Go 文件
                  </el-dropdown-item>
                  
                  <el-dropdown-item 
                    v-if="data.type === 'package' && !data.hub_full_code_path && hasPermission(data, DirectoryPermission.read)" 
                    command="publish-to-hub"
                  >
                    <el-icon><Upload /></el-icon>
                    发布到 Hub
                  </el-dropdown-item>
                  
                  <el-dropdown-item 
                    v-if="data.type === 'package' && data.hub_full_code_path && hasPermission(data, DirectoryPermission.write)" 
                    command="push-to-hub"
                  >
                    <el-icon><Upload /></el-icon>
                    推送到 Hub
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
          </el-dropdown>
        </template>
      </el-tree>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { Plus, MoreFilled, CopyDocument, Document, Upload, Download, Delete, Key, DocumentChecked, Edit, ChatDotRound, ChatDotSquare, Search } from '@element-plus/icons-vue'
import ChartIcon from '@/shared/components/icons/ChartIcon.vue'
import TableIcon from '@/shared/components/icons/TableIcon.vue'
import FormIcon from '@/shared/components/icons/FormIcon.vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import type { ServiceTree } from '@/types'
import { isRootNode } from '@/utils/tree-utils'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { updatePackage, updateServiceTreeFunction, updateDocs } from '@/api/service-tree'
import { 
  hasPermission, 
  hasAnyPermissionForNode, 
  DirectoryPermission,
  FunctionPermission,
  TablePermission, 
  buildPermissionApplyURL 
} from '@/utils/permission'
import { useAuthStore } from '@/stores/auth'
import { eventBus, RouteEvent } from '@/architecture/infrastructure/eventBus'
import { isServiceTreeNodeAdmin } from '@/utils/permissionActors'
import { useServiceTreeClipboard } from '../composables/useServiceTreeClipboard'
import { useServiceTreeSearchExpand } from '../composables/useServiceTreeSearchExpand'

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
  (e: 'refresh-tree'): void  // 刷新树（复制粘贴后需要刷新）
  (e: 'update-history', node?: ServiceTree): void  // 显示变更记录（工作空间或目录）
  (e: 'import-go-files', node: ServiceTree): void  // 导入 Go 文件到目录
  (e: 'publish-to-hub', node: ServiceTree): void  // 发布到 Hub
  (e: 'push-to-hub', node: ServiceTree): void  // 推送到 Hub
  (e: 'pull-from-hub', initialLink?: string, targetFullCodePath?: string, targetName?: string): void  // 从 Hub 拉取，可选预填链接与目标目录（路径+名称）
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const router = useRouter()

// 获取当前用户信息
const authStore = useAuthStore()

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

const {
  copiedDirectory,
  copiedHubLink,
  handleCopy,
  handlePaste
} = useServiceTreeClipboard({
  treeData: groupedTreeData,
  currentFunction: computed(() => props.currentFunction),
  currentNodeId: computed(() => props.currentNodeId),
  onRefreshTree: () => emit('refresh-tree'),
  onPullFromHub: (initialLink, targetFullCodePath, targetName) => {
    emit('pull-from-hub', initialLink, targetFullCodePath, targetName)
  }
})

// 重命名目录
const handleRename = async (node: ServiceTree) => {
  if (node.type !== 'package') {
    ElMessage.warning('只能重命名目录（package类型）')
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

// 处理无权限节点点击
const handleNoPermissionClick = (data: ServiceTree) => {
  // 跳转到权限申请页面
  const resourcePath = data.full_code_path
  // ⭐ 根据节点类型确定资源类型（package 统一为 directory）
  const resourceType = data.type === 'package' ? 'directory' : 'function'
  const templateType = data.template_type
  
  // 构建权限申请 URL
  const defaultAction = resourceType === 'directory' ? DirectoryPermission.read : FunctionPermission.read
  const url = `/permissions/apply?resource=${encodeURIComponent(resourcePath)}&action=${encodeURIComponent(defaultAction)}`
  const finalUrl = templateType ? `${url}&templateType=${encodeURIComponent(templateType)}` : url
  
  router.push(finalUrl)
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
  // 直接触发 node-click 事件，让父组件处理路由跳转
  // ⭐ 下拉菜单的点击已经通过 @click.stop.prevent 阻止了事件冒泡，所以这里不需要额外检查
  emit('node-click', data)
}

// 缓存当前用户名，避免在模板中重复访问响应式对象
const currentUsername = computed(() => authStore.user?.username || '')

// 判断是否是管理员（使用缓存的用户名）
const isAdmin = (node: ServiceTree): boolean => {
  return isServiceTreeNodeAdmin(node, currentUsername.value)
}

// 处理申请权限
const handleApplyPermission = (data: ServiceTree) => {
  const resourcePath = data.full_code_path
  // ⭐ 根据节点类型确定资源类型（package 统一为 directory）
  const resourceType = data.type === 'package' ? 'directory' : 'function'
  const defaultAction = resourceType === 'directory' ? DirectoryPermission.read : FunctionPermission.read
  const url = buildPermissionApplyURL(resourcePath, defaultAction, data.template_type)
  router.push(url)
}

// 处理待审批数量点击
const handlePendingCountClick = (data: ServiceTree) => {
  handleApprovePermission(data)
}

// 处理审批权限申请
const handleApprovePermission = (data: ServiceTree) => {
  // 先触发节点点击，确保节点详情已加载
  emit('node-click', data)
  
  // 然后通过事件总线更新路由，添加 tab 参数
  // 使用 nextTick 确保节点点击事件已处理
  nextTick(() => {
    const targetPath = `/workspace${data.full_code_path}`
    eventBus.emit(RouteEvent.updateRequested, {
      path: targetPath,
      query: {
        _panel: 'permissionRequest'
      },
      replace: true,
      preserveParams: {
        table: false,
        search: false,
        state: false,
        linkNavigation: false
      },
      source: 'approve-permission-click'
    })
  })
}

// 处理权限管理
const handleManagePermission = (data: ServiceTree) => {
  const resourcePath = data.full_code_path
  // ⭐ 根据节点类型确定资源类型（package 统一为 directory）
  const resourceType = data.type === 'package' ? 'directory' : 'function'
  const defaultAction = resourceType === 'directory' ? DirectoryPermission.read : FunctionPermission.read
  // 权限管理页面，默认显示授权模式
  const url = buildPermissionApplyURL(resourcePath, defaultAction, data.template_type) + '&mode=grant'
  router.push(url)
}

const handleNodeAction = (command: string, data: ServiceTree) => {
  if (command === 'create-directory') {
    emit('create-directory', data)
  } else if (command === 'create-docs') {
    emit('create-docs', data)
  } else if (command === 'create-board') {
    emit('create-board', data)
  } else if (command === 'rename') {
    handleRename(data)
  } else if (command === 'copy') {
    handleCopy(data)
  } else if (command === 'paste') {
    // 粘贴时,如果右键的节点是 package，使用该节点；否则使用当前选中的目录
    if (data.type === 'package') {
      handlePaste(data)
    } else {
      handlePaste() // 使用当前选中的目录
    }
  } else if (command === 'delete-function') {
    emit('delete-function', data)
  } else if (command === 'delete-doc') {
    emit('delete-doc', data)
  } else if (command === 'delete-board') {
    emit('delete-board', data)
  } else if (command === 'delete-directory') {
    emit('delete-directory', data)
  } else if (command === 'import-go-files') {
    emit('import-go-files', data)
  } else if (command === 'publish-to-hub') {
    emit('publish-to-hub', data)
  } else if (command === 'push-to-hub') {
    emit('push-to-hub', data)
  } else if (command === 'update-history') {
    emit('update-history', data)
  } else if (command === 'apply-permission') {
    handleApplyPermission(data)
  } else if (command === 'approve-permission') {
    handleApprovePermission(data)
  } else if (command === 'manage-permission') {
    handleManagePermission(data)
  } else if (command === 'open-workstation') {
    // 在本页打开 Mini 工作台，不新开标签；WorkspaceView 监听此事件
    eventBus.emit('workspace:open-workstation', { full_code_path: data.full_code_path || '' })
  }
}

// 处理从应用中心安装按钮点击
const handlePullFromHubClick = () => {
  emit('pull-from-hub')
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
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-light);
  display: flex;
  flex-direction: column;
  gap: 10px;

  .tree-search-input {
    flex-shrink: 0;
  }

  .tree-search-input :deep(.el-input__wrapper) {
    border-radius: 6px;
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
      color: #4f46e5 !important; /* indigo-600，更深的紫色 */
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
    color: #6366f1;  /* ✅ 旧版本紫色主题色（indigo-500） */
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
    
    &.no-permission {
      color: var(--el-text-color-disabled);
      opacity: 0.6;
    }
  }
  
  .no-permission-icon {
    width: 16px;
    height: 16px;
    margin-left: 4px;
    cursor: pointer;
    opacity: 0.7;
    flex-shrink: 0;
    transition: opacity 0.2s ease;
    
    &:hover {
      opacity: 1;
    }
  }
  
  /* ⭐ 待审批数量 badge - 防止被挤压 */
  .pending-count-badge {
    flex-shrink: 0;
    margin-left: 6px;
    cursor: pointer;
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
  z-index: 9999 !important;
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
</style>
