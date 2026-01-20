<template>
  <div class="service-tree-panel" v-loading="loading">
    <div class="tree-header">
      <h3>服务目录</h3>
      <div class="header-actions">
        <el-link
          v-if="!loading"
          type="primary"
          :underline="false"
          @click="$emit('create-directory')"
          class="header-link"
        >
          <el-icon><Plus /></el-icon>
          创建目录
        </el-link>
        <el-link
          v-if="!loading"
          type="primary"
          :underline="false"
          @click="handleUpdateHistoryClick"
          class="header-link"
        >
          <el-icon><Clock /></el-icon>
          变更记录
        </el-link>
      </div>
    </div>
    
    <div class="tree-content">
      <el-tree
        v-if="groupedTreeData.length > 0"
        ref="treeRef"
        :data="groupedTreeData"
        :props="{ children: 'children', label: 'name' }"
        node-key="id"
        :default-expand-all="false"
        :default-expanded-keys="defaultExpandedKeysWithWorkspace"
        :expand-on-click-node="false"
        :highlight-current="true"
        @node-click="handleNodeClick"
      >
        <template #default="{ node, data }">
          <span class="tree-node">
            <!-- app 类型：使用目录图标（与 package 保持一致） -->
            <img 
              v-if="data.type === 'app'" 
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
            
            <!-- Hub 标记 - 已发布到 Hub 的 app 或目录显示 -->
            <span
              v-if="(data.type === 'app' || data.type === 'package') && data.hub_directory_id && data.hub_directory_id > 0"
              class="hub-badge"
              @click.stop="handleHubBadgeClick(data)"
              :title="data.hub_version ? `已发布到应用中心 ${data.hub_version}` : '已发布到应用中心'"
            >
              <el-icon class="hub-icon"><Link /></el-icon>
              <span v-if="data.hub_version" class="hub-version">{{ data.hub_version }}</span>
            </span>
            
            <!-- ⭐ 待审批数量 badge - 仅管理员可见（app、package 和 function 类型都显示） -->
            <el-badge
              v-if="(data.type === 'app' || data.type === 'package' || data.type === 'function') && isAdmin(data) && data.pending_count && data.pending_count > 0"
              :value="data.pending_count"
              :max="99"
              class="pending-count-badge"
              @click.stop="handlePendingCountClick(data)"
              :title="`有 ${data.pending_count} 个待审批的权限申请`"
            />
            
            <!-- 更多操作按钮 - 鼠标悬停时显示 -->
            <el-dropdown
              trigger="click"
              :teleported="true"
              popper-class="service-tree-dropdown-popper"
              @click.stop
              class="node-more-actions"
              @command="(command: string) => handleNodeAction(command, data)"
            >
              <el-icon 
                class="more-icon" 
                @click.stop
              >
                <MoreFilled />
              </el-icon>
              <template #dropdown>
                <el-dropdown-menu>
                  <!-- 对 app 和 package 类型显示创建子目录选项（需要 directory:write 或 app:write 权限） -->
                  <el-dropdown-item 
                    v-if="(data.type === 'app' || data.type === 'package') && (hasPermission(data, DirectoryPermissions.write) || hasPermission(data, AppPermissions.write))" 
                    command="create-directory"
                  >
                    <el-icon><Plus /></el-icon>
                    添加服务目录
                  </el-dropdown-item>
                  <!-- 对 app 和 package 类型显示创建文档选项（需要 directory:write 或 app:write 权限） -->
                  <el-dropdown-item 
                    v-if="(data.type === 'app' || data.type === 'package') && (hasPermission(data, DirectoryPermissions.write) || hasPermission(data, AppPermissions.write))" 
                    command="create-docs"
                  >
                    <el-icon><Document /></el-icon>
                    添加文档
                  </el-dropdown-item>
                  <!-- 对 app 和 package 类型显示复制选项（需要 directory:read 或 app:read 权限） -->
                  <el-dropdown-item 
                    v-if="(data.type === 'app' || data.type === 'package') && (hasPermission(data, DirectoryPermissions.read) || hasPermission(data, AppPermissions.read))" 
                    command="copy" 
                    divided
                  >
                    <el-icon><CopyDocument /></el-icon>
                    复制
                  </el-dropdown-item>
                  <!-- 粘贴选项（当有复制的内容或 Hub 链接时显示，粘贴到当前选中的目录，需要 directory:write 或 app:write 权限） -->
                  <el-dropdown-item 
                    v-if="(copiedDirectory || copiedHubLink) && (data.type === 'app' || data.type === 'package') && (hasPermission(data, DirectoryPermissions.write) || hasPermission(data, AppPermissions.write))" 
                    command="paste" 
                    divided
                  >
                    <el-icon><Document /></el-icon>
                    粘贴
                  </el-dropdown-item>
                  <!-- 对 app 和 package 类型显示重命名选项（需要 directory:write 或 app:write 权限） -->
                  <el-dropdown-item 
                    v-if="(data.type === 'app' || data.type === 'package') && (hasPermission(data, DirectoryPermissions.write) || hasPermission(data, AppPermissions.write))"
                    command="rename"
                  >
                    <el-icon><Edit /></el-icon>
                    重命名
                  </el-dropdown-item>
                  <!-- 复制链接（需要 directory:read、app:read 或 function:read 权限） -->
                  <el-dropdown-item 
                    v-if="hasPermission(data, data.type === 'app' ? AppPermissions.read : (data.type === 'package' ? DirectoryPermissions.read : 'function:read'))"
                    command="copy-link"
                  >
                    <el-icon><Link /></el-icon>
                    复制链接
                  </el-dropdown-item>
                  <!-- 仅对function类型显示删除选项（需要 function:delete 权限） -->
                  <el-dropdown-item 
                    v-if="data.type === 'function' && hasPermission(data, 'function:delete')"
                    command="delete-function"
                    divided
                  >
                    <el-icon><Delete /></el-icon>
                    删除
                  </el-dropdown-item>
                  <!-- 仅对docs类型显示删除文档选项（需要 directory:write 权限） -->
                  <el-dropdown-item 
                    v-if="data.type === 'docs' && hasPermission(data, DirectoryPermissions.write)"
                    command="delete-doc"
                    divided
                  >
                    <el-icon><Delete /></el-icon>
                    删除文档
                  </el-dropdown-item>
                  <!-- 对 app 和 package 类型显示发布到Hub选项（未发布时，需要 directory:manage 或 app:admin 权限） -->
                  <el-dropdown-item 
                    v-if="(data.type === 'app' || data.type === 'package') && (!data.hub_directory_id || data.hub_directory_id === 0) && (hasPermission(data, DirectoryPermissions.manage) || hasPermission(data, AppPermissions.admin))" 
                    command="publish-to-hub" 
                    divided
                  >
                    <el-icon><Upload /></el-icon>
                    发布到应用中心
                  </el-dropdown-item>
                  <!-- 对 app 和 package 类型显示推送到Hub选项（已发布时，需要 directory:manage 或 app:admin 权限） -->
                  <el-dropdown-item 
                    v-if="(data.type === 'app' || data.type === 'package') && data.hub_directory_id && data.hub_directory_id > 0 && (hasPermission(data, DirectoryPermissions.manage) || hasPermission(data, AppPermissions.admin))" 
                    command="push-to-hub" 
                    divided
                  >
                    <el-icon><Upload /></el-icon>
                    推送到应用中心
                  </el-dropdown-item>
                  <!-- 对 app 和 package 类型显示变更记录选项（需要 directory:read 或 app:read 权限） -->
                  <el-dropdown-item 
                    v-if="(data.type === 'app' || data.type === 'package') && (hasPermission(data, DirectoryPermissions.read) || hasPermission(data, AppPermissions.read))" 
                    command="update-history" 
                    divided
                  >
                    <el-icon><Clock /></el-icon>
                    变更记录
                  </el-dropdown-item>
                  <!-- 申请权限选项（对所有节点都显示） -->
                  <el-dropdown-item 
                    command="apply-permission" 
                    divided
                  >
                    <el-icon><Key /></el-icon>
                    申请权限
                  </el-dropdown-item>
                  <!-- ⭐ 审批权限申请选项（仅管理员可见，且有待审批申请时显示） -->
                  <el-dropdown-item 
                    v-if="(data.type === 'app' || data.type === 'package' || data.type === 'function') && isAdmin(data) && data.pending_count && data.pending_count > 0" 
                    command="approve-permission" 
                  >
                    <el-icon><DocumentChecked /></el-icon>
                    审批权限申请
                    <el-badge 
                      :value="data.pending_count" 
                      :max="99" 
                      class="dropdown-badge"
                    />
                  </el-dropdown-item>
                  <!-- 权限管理选项（对 app 和 package 类型显示，且仅管理员可见） -->
                  <el-dropdown-item 
                    v-if="(data.type === 'app' || data.type === 'package') && isAdmin(data)" 
                    command="manage-permission" 
                  >
                    <el-icon><User /></el-icon>
                    权限管理
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </span>
        </template>
      </el-tree>
      
      <div v-else class="empty-state">
        <el-empty description="暂无服务目录" :image-size="80">
          <el-button type="primary" @click="$emit('create-directory')">
            <el-icon><Plus /></el-icon>
            创建服务目录
          </el-button>
        </el-empty>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Plus, MoreFilled, Link, CopyDocument, Document, Clock, Upload, Download, Delete, Key, User, DocumentChecked, Edit } from '@element-plus/icons-vue'
import ChartIcon from './icons/ChartIcon.vue'
import TableIcon from './icons/TableIcon.vue'
import FormIcon from './icons/FormIcon.vue'
import { ElTag, ElLink, ElMessageBox, ElMessage } from 'element-plus'
import type { ServiceTree } from '@/types'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { copyDirectory, updateServiceTree } from '@/api/service-tree'
import {
  findPathToNode,
  expandParentNodes,
  findNodeByPath,
  expandPathAndSelect,
  expandPathOnly
} from '@/utils/serviceTreeUtils'
import { navigateToHubDirectoryDetail } from '@/utils/hub-navigation'
import { hasPermission, hasAnyPermissionForNode, DirectoryPermissions, AppPermissions, TablePermissions, buildPermissionApplyURL } from '@/utils/permission'
import { useAuthStore } from '@/stores/auth'
import { eventBus, RouteEvent } from '@/architecture/infrastructure/eventBus'

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
  (e: 'delete-doc', node: ServiceTree): void
  (e: 'copy-link', node: ServiceTree): void
  (e: 'delete-function', node: ServiceTree): void  // 删除函数
  (e: 'refresh-tree'): void  // 刷新树（复制粘贴后需要刷新）
  (e: 'update-history', node?: ServiceTree): void  // 显示变更记录（工作空间或目录）
  (e: 'publish-to-hub', node: ServiceTree): void  // 发布到 Hub
  (e: 'push-to-hub', node: ServiceTree): void  // 推送到 Hub
  (e: 'pull-from-hub'): void  // 从 Hub 拉取
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const router = useRouter()
const route = useRoute()

// 获取当前用户信息
const authStore = useAuthStore()

// el-tree 的引用
const treeRef = ref()

// 复制粘贴相关状态
const copiedDirectory = ref<ServiceTree | null>(null)  // 复制的目录信息（本地目录）
const copiedHubLink = ref<string | null>(null)  // 复制的 Hub 链接
const isPasting = ref(false)  // 是否正在粘贴

// localStorage 键名
const COPIED_DIRECTORY_KEY = 'copied_directory'
const COPIED_HUB_LINK_KEY = 'copied_hub_link'

// 从 localStorage 恢复复制的目录或 Hub 链接
const restoreCopiedDirectory = () => {
  try {
    // 恢复本地目录
    const saved = localStorage.getItem(COPIED_DIRECTORY_KEY)
    if (saved) {
      const parsed = JSON.parse(saved)
      // 验证数据格式
      if (parsed && parsed.full_code_path && parsed.name) {
        copiedDirectory.value = parsed as ServiceTree
      } else {
        localStorage.removeItem(COPIED_DIRECTORY_KEY)
      }
    }
    
    // 恢复 Hub 链接
    const savedHubLink = localStorage.getItem(COPIED_HUB_LINK_KEY)
    if (savedHubLink && savedHubLink.startsWith('hub://')) {
      copiedHubLink.value = savedHubLink
    } else if (savedHubLink) {
      localStorage.removeItem(COPIED_HUB_LINK_KEY)
    }
  } catch (error) {
    console.error('恢复复制的目录失败:', error)
    localStorage.removeItem(COPIED_DIRECTORY_KEY)
    localStorage.removeItem(COPIED_HUB_LINK_KEY)
  }
}

// 保存复制的目录到 localStorage
const saveCopiedDirectory = (node: ServiceTree) => {
  try {
    // 只保存必要的字段，避免存储过多数据
    const dataToSave = {
      id: node.id,
      name: node.name,
      full_code_path: node.full_code_path,
      app_id: node.app_id,
      type: node.type
    }
    localStorage.setItem(COPIED_DIRECTORY_KEY, JSON.stringify(dataToSave))
    // 清除 Hub 链接（如果存在）
    copiedHubLink.value = null
    localStorage.removeItem(COPIED_HUB_LINK_KEY)
  } catch (error) {
    console.error('保存复制的目录失败:', error)
  }
}

// 保存复制的 Hub 链接到 localStorage
const saveCopiedHubLink = (hubLink: string) => {
  try {
    localStorage.setItem(COPIED_HUB_LINK_KEY, hubLink)
    // 清除本地目录（如果存在）
    copiedDirectory.value = null
    localStorage.removeItem(COPIED_DIRECTORY_KEY)
  } catch (error) {
    console.error('保存复制的 Hub 链接失败:', error)
  }
}

// 组件挂载时恢复复制的目录
onMounted(() => {
  restoreCopiedDirectory()
  window.addEventListener('keydown', handleKeyDown)
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
      await updateServiceTree(node.id, { name: trimmedName })
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

// 复制目录
const handleCopy = (node: ServiceTree) => {
  if (node.type !== 'package') {
    ElMessage.warning('只能复制目录（package类型）')
    return
  }
  
  copiedDirectory.value = node
  saveCopiedDirectory(node)  // 保存到 localStorage
  ElMessage.success(`已复制目录：${node.name}`)
}

  // 粘贴目录（使用当前选中的目录作为目标）
  // 支持两种模式：
  // 1. 粘贴本地复制的目录
  // 2. 粘贴 Hub 链接（从剪贴板检测或已保存的 Hub 链接）
  const handlePaste = async (targetNode?: ServiceTree) => {
    // 首先检查剪贴板是否有 Hub 链接
    let hubLinkToPaste: string | null = null
    try {
      const clipboardText = await navigator.clipboard.readText()
      if (clipboardText && clipboardText.trim().startsWith('hub://')) {
        hubLinkToPaste = clipboardText.trim()
        // 保存到 localStorage
        saveCopiedHubLink(hubLinkToPaste)
        copiedHubLink.value = hubLinkToPaste
      }
    } catch (error) {
      // 剪贴板访问失败，忽略（可能是权限问题）
      console.debug('无法读取剪贴板:', error)
    }
    
    // 如果剪贴板没有 Hub 链接，检查已保存的 Hub 链接
    if (!hubLinkToPaste && copiedHubLink.value) {
      hubLinkToPaste = copiedHubLink.value
    }
    
    // 如果有 Hub 链接，使用 Hub 链接粘贴
    if (hubLinkToPaste) {
      await handlePasteHubLink(hubLinkToPaste, targetNode)
      return
    }
    
    // 否则使用本地复制的目录
    if (!copiedDirectory.value) {
      ElMessage.warning('没有可粘贴的目录或 Hub 链接')
      return
    }
    
    // 如果没有传入 targetNode，使用当前选中的目录
    let finalTargetNode = targetNode
    if (!finalTargetNode && props.currentFunction && props.currentFunction.type === 'package') {
      finalTargetNode = props.currentFunction
    }
    
    // 如果还是没有目标节点，尝试从树数据中查找当前选中的节点
    if (!finalTargetNode && props.currentNodeId) {
      const findNodeById = (nodes: ServiceTree[], id: number | string): ServiceTree | null => {
        for (const node of nodes) {
          if (Number(node.id) === Number(id)) {
            return node
          }
          if (node.children && node.children.length > 0) {
            const found = findNodeById(node.children, id)
            if (found) return found
          }
        }
        return null
      }
      finalTargetNode = findNodeById(groupedTreeData.value, props.currentNodeId)
    }
    
    if (!finalTargetNode) {
      ElMessage.warning('请先选择一个目录作为粘贴目标')
      return
    }
    
    if (finalTargetNode.type !== 'package') {
      ElMessage.warning('只能粘贴到目录（package类型）')
      return
    }
    
    // 检查是否粘贴到自己或子目录
    if (copiedDirectory.value.full_code_path === finalTargetNode.full_code_path) {
      ElMessage.warning('不能粘贴到自己')
      return
    }
    
    // 检查是否粘贴到自己的子目录
    if (finalTargetNode.full_code_path.startsWith(copiedDirectory.value.full_code_path + '/')) {
      ElMessage.warning('不能粘贴到自己的子目录')
      return
    }
    
    // 检查是否是跨应用复制
    const sourcePathParts = copiedDirectory.value.full_code_path.split('/').filter(Boolean)
    const targetPathParts = finalTargetNode.full_code_path.split('/').filter(Boolean)
    const isCrossApp = sourcePathParts.length >= 2 && targetPathParts.length >= 2 && 
                       (sourcePathParts[0] !== targetPathParts[0] || sourcePathParts[1] !== targetPathParts[1])
    
    // 构建确认消息
    let confirmMessage = `确定要将目录 "${copiedDirectory.value.name}" 复制到 "${finalTargetNode.name}" 吗？\n\n`
    confirmMessage += `源目录：${copiedDirectory.value.full_code_path}\n`
    confirmMessage += `目标目录：${finalTargetNode.full_code_path}`
    if (isCrossApp) {
      confirmMessage += `\n\n⚠️ 注意：这是跨应用复制操作`
    }
    
    // 弹窗确认
    try {
      await ElMessageBox.confirm(
        confirmMessage,
        '确认粘贴',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'info'
        }
      )
      
      // 执行粘贴
      isPasting.value = true
      try {
        // 解析目标应用信息（从 finalTargetNode.full_code_path 中提取）
        const targetPathParts = finalTargetNode.full_code_path.split('/').filter(Boolean)
        if (targetPathParts.length < 2) {
          throw new Error('目标路径格式错误')
        }
        
        // 获取目标应用ID
        if (!finalTargetNode.app_id) {
          throw new Error('无法获取目标应用ID，请确保目标目录有效')
        }
        
        const targetAppId = finalTargetNode.app_id
        
        await copyDirectory({
          source_directory_path: copiedDirectory.value.full_code_path,
          target_directory_path: finalTargetNode.full_code_path,
          target_app_id: targetAppId
        })
      
      ElMessage.success('目录复制成功')
      
      // 触发刷新树事件
      emit('refresh-tree')
      
      // 清空复制状态（可选，也可以保留以便多次粘贴）
      // copiedDirectory.value = null
    } catch (error: any) {
      // 用户取消操作不显示错误
      if (error !== 'cancel' && error !== 'close') {
        const errorMessage = error?.response?.data?.message || error?.message || '复制失败'
        ElMessage.error(errorMessage)
      }
    } finally {
      isPasting.value = false
    }
  } catch (error) {
    // 用户取消
  }
}

// 粘贴 Hub 链接
const handlePasteHubLink = async (hubLink: string, targetNode?: ServiceTree) => {
  // 如果没有传入 targetNode，使用当前选中的目录
  let finalTargetNode = targetNode
  if (!finalTargetNode && props.currentFunction && props.currentFunction.type === 'package') {
    finalTargetNode = props.currentFunction
  }
  
  // 如果还是没有目标节点，尝试从树数据中查找当前选中的节点
  if (!finalTargetNode && props.currentNodeId) {
    const findNodeById = (nodes: ServiceTree[], id: number | string): ServiceTree | null => {
      for (const node of nodes) {
        if (Number(node.id) === Number(id)) {
          return node
        }
        if (node.children && node.children.length > 0) {
          const found = findNodeById(node.children, id)
          if (found) return found
        }
      }
      return null
    }
    finalTargetNode = findNodeById(groupedTreeData.value, props.currentNodeId)
  }
  
  if (!finalTargetNode) {
    ElMessage.warning('请先选择一个目录作为粘贴目标')
    return
  }
  
  if (finalTargetNode.type !== 'package') {
    ElMessage.warning('只能粘贴到目录（package类型）')
    return
  }
  
  // 构建确认消息
  let confirmMessage = `确定要从 Hub 链接复制目录到 "${finalTargetNode.name}" 吗？\n\n`
  confirmMessage += `Hub 链接：${hubLink}\n`
  confirmMessage += `目标目录：${finalTargetNode.full_code_path}`
  
  // 弹窗确认
  try {
    await ElMessageBox.confirm(
      confirmMessage,
      '确认粘贴',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info'
      }
    )
    
    // 执行粘贴
    isPasting.value = true
    try {
      // 获取目标应用ID
      if (!finalTargetNode.app_id) {
        throw new Error('无法获取目标应用ID，请确保目标目录有效')
      }
      
      const targetAppId = finalTargetNode.app_id
      
      // 调用复制 API（后端会自动检测 hub:// 前缀）
      await copyDirectory({
        source_directory_path: hubLink,  // Hub 链接
        target_directory_path: finalTargetNode.full_code_path,
        target_app_id: targetAppId
      })
    
      ElMessage.success('目录复制成功')
      
      // 触发刷新树事件
      emit('refresh-tree')
      
      // 保留 Hub 链接以便多次粘贴
    } catch (error: any) {
      // 用户取消操作不显示错误
      if (error !== 'cancel' && error !== 'close') {
        const errorMessage = error?.response?.data?.message || error?.message || '复制失败'
        ElMessage.error(errorMessage)
      }
    } finally {
      isPasting.value = false
    }
  } catch (error) {
    // 用户取消
  }
}


// ⭐ 直接使用后端返回的树数据（已包含 app 根节点）
const groupedTreeData = computed(() => {
  return props.treeData
})

// ⭐ 默认展开的节点（后端返回的 expandedKeys 中已包含 app 根节点）
const defaultExpandedKeysWithWorkspace = computed(() => {
  // 直接使用后端返回的 expandedKeys
  return props.expandedKeys || []
})

// 处理无权限节点点击
const handleNoPermissionClick = (data: ServiceTree) => {
  // 跳转到权限申请页面
  const resourcePath = data.full_code_path
  const resourceType = data.type === 'app' ? 'app' : (data.type === 'package' ? 'directory' : 'function')
  const templateType = data.template_type
  
  // 构建权限申请 URL
  const defaultAction = resourceType === 'app' ? 'app:read' : (resourceType === 'directory' ? 'directory:read' : 'function:read')
  const url = `/permissions/apply?resource=${encodeURIComponent(resourcePath)}&action=${encodeURIComponent(defaultAction)}`
  const finalUrl = templateType ? `${url}&templateType=${encodeURIComponent(templateType)}` : url
  
  router.push(finalUrl)
}

const handleNodeClick = (data: ServiceTree) => {
  // 直接触发 node-click 事件，让父组件处理路由跳转
  // ⭐ 下拉菜单的点击已经通过 @click.stop.prevent 阻止了事件冒泡，所以这里不需要额外检查
  emit('node-click', data)
}

// 判断是否是管理员
const isAdmin = (node: ServiceTree): boolean => {
  if (!node.admins || !authStore.user?.username) {
    return false
  }
  const admins = node.admins.split(',').map(a => a.trim()).filter(Boolean)
  return admins.includes(authStore.user.username)
}

// 处理申请权限
const handleApplyPermission = (data: ServiceTree) => {
  const resourcePath = data.full_code_path
  const resourceType = data.type === 'app' ? 'app' : (data.type === 'package' ? 'directory' : 'function')
  const defaultAction = resourceType === 'app' ? 'app:read' : (resourceType === 'directory' ? 'directory:read' : 'function:read')
  const url = buildPermissionApplyURL(resourcePath, defaultAction, data.template_type)
  router.push(url)
}

// 处理待审批数量点击
const handlePendingCountClick = (data: ServiceTree) => {
  // 点击待审批数量时，跳转到节点详情页面的权限申请 tab
  // 这里先触发 node-click 事件，让父组件处理路由跳转
  // 后续可以在详情页面添加权限申请 tab
  emit('node-click', data)
  // TODO: 在详情页面添加权限申请 tab，显示待审批的申请列表
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
        tab: 'permissionRequest'  // 指定要打开的 tab
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
  const resourceType = data.type === 'app' ? 'app' : (data.type === 'package' ? 'directory' : 'function')
  const defaultAction = resourceType === 'app' ? 'app:read' : (resourceType === 'directory' ? 'directory:read' : 'function:read')
  // 权限管理页面，默认显示授权模式
  const url = buildPermissionApplyURL(resourcePath, defaultAction, data.template_type) + '&mode=grant'
  router.push(url)
}

const handleNodeAction = (command: string, data: ServiceTree) => {
  if (command === 'create-directory') {
    emit('create-directory', data)
  } else if (command === 'create-docs') {
    emit('create-docs', data)
  } else if (command === 'rename') {
    handleRename(data)
  } else if (command === 'copy') {
    handleCopy(data)
  } else if (command === 'paste') {
    // 粘贴时,如果右键的节点是目录或 app，使用该节点；否则使用当前选中的目录
    if (data.type === 'app' || data.type === 'package') {
      handlePaste(data)
    } else {
      handlePaste() // 使用当前选中的目录
    }
  } else if (command === 'copy-link') {
    emit('copy-link', data)
  } else if (command === 'delete-function') {
    emit('delete-function', data)
  } else if (command === 'delete-doc') {
    emit('delete-doc', data)
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
  }
}

// 处理 Ctrl+V 快捷键
const handleKeyDown = (event: KeyboardEvent) => {
  // 检查是否是 Ctrl+V 或 Cmd+V（Mac）
  if ((event.ctrlKey || event.metaKey) && event.key === 'v') {
    // 检查是否在输入框中（避免与输入框的粘贴冲突）
    const target = event.target as HTMLElement
    if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
      return // 在输入框中，不处理
    }
    
    // 阻止默认行为
    event.preventDefault()
    
    // 执行粘贴
    handlePaste()
  }
}

// 注销键盘事件监听
onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})

// 处理变更记录按钮点击
const handleUpdateHistoryClick = () => {
  // 显示工作空间变更记录
  emit('update-history')
}

// 处理 Hub 标记点击 - 跳转到 Hub 目录详情页
const handleHubBadgeClick = (data: ServiceTree) => {
  if (data.hub_directory_id && data.hub_directory_id > 0) {
    navigateToHubDirectoryDetail(data.hub_directory_id)
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
  if (data.type === 'app' || data.type === 'package') {
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
  }
  return 'function-icon'
  }
  
// ⭐ 递归查找所有 pending_count > 0 的节点
const findAllNodesWithPendingCount = (nodes: ServiceTree[]): ServiceTree[] => {
  const result: ServiceTree[] = []
  
  const traverse = (nodeList: ServiceTree[]) => {
    for (const node of nodeList) {
      // 检查当前节点是否有 pending_count > 0
      if (node.pending_count && node.pending_count > 0) {
        result.push(node)
      }
      
      // 递归处理子节点
      if (node.children && node.children.length > 0) {
        traverse(node.children)
      }
    }
  }
  
  traverse(nodes)
  return result
}

// ⭐ 自动展开所有 pending_count > 0 的节点及其父节点
const expandNodesWithPendingCount = async (treeData: ServiceTree[]) => {
  if (!treeRef.value || !treeData.length) {
    return
  }
  
  // 查找所有 pending_count > 0 的节点
  const nodesWithPending = findAllNodesWithPendingCount(treeData)
  
  if (nodesWithPending.length === 0) {
    return
  }
  
  console.log(`[ServiceTreePanel] 找到 ${nodesWithPending.length} 个待审批节点，自动展开`)
  
  // 收集所有需要展开的节点 ID（包括节点本身及其所有父节点）
  const expandNodeIds = new Set<number>()
  
  for (const node of nodesWithPending) {
    const nodeId = Number(node.id)
    // 找到从根到该节点的路径
    const path = findPathToNode(treeData, nodeId)
    // 将路径中的所有节点 ID 添加到展开集合中
    path.forEach(id => expandNodeIds.add(id))
  }
  
  // 展开所有收集到的节点
  if (expandNodeIds.size > 0) {
    const expandKeys = Array.from(expandNodeIds)
    console.log(`[ServiceTreePanel] 展开 ${expandKeys.length} 个节点:`, expandKeys)
    
    // 使用 Element Plus Tree 的 setExpandedKeys 方法批量展开
    await nextTick()
    if (treeRef.value && treeRef.value.setExpandedKeys) {
      treeRef.value.setExpandedKeys(expandKeys, false) // false 表示不触发 expand 事件
    } else {
      // 如果 setExpandedKeys 不可用，使用 expandPathAndSelect 逐个展开
      for (const nodeId of expandKeys) {
        const path = findPathToNode(treeData, nodeId)
        if (path.length > 0) {
          await expandPathAndSelect(
            treeRef.value,
            treeData,
            path,
            nodeId
          )
        }
      }
    }
  }
}

// 展开多个路径
const expandPaths = async (paths: string[]) => {
  if (!treeRef.value || !groupedTreeData.value.length) {
    return
  }
  
  for (const path of paths) {
    // 根据 full_code_path 查找节点
    const node = findNodeByPath(groupedTreeData.value, path)
    if (node) {
      // 找到节点后，展开到该节点的所有父节点
      const nodeId = Number(node.id)
      const pathToNode = findPathToNode(groupedTreeData.value, nodeId)
      if (pathToNode.length > 0) {
        await expandPathAndSelect(
          treeRef.value,
          groupedTreeData.value,
          pathToNode,
          nodeId
        )
      }
    }
  }
}

// 监听 currentNodeId 变化，自动展开并选中节点
watch(() => props.currentNodeId, async (nodeId) => {
  if (nodeId && treeRef.value && groupedTreeData.value.length > 0) {
    // 🔥 使用 nextTick 确保 DOM 已渲染
    await nextTick()
      // 查找路径（使用分组后的数据）
      const path = findPathToNode(groupedTreeData.value, nodeId)
      
      if (path.length > 0) {
      // 展开路径并选中节点
      await expandPathAndSelect(
        treeRef.value,
        groupedTreeData.value,
        path,
        Number(nodeId)
      )
          
          // 🔥 滚动到选中节点（可见）
      await nextTick()
            const selectedNode = treeRef.value.store.nodesMap[nodeId]
            if (selectedNode) {
              selectedNode.visible = true
            }
      }
  }
}, { immediate: true })

// ⭐ 防重复展开标志
let isExpanding = false
let lastExpandedKeys: number[] = []

// ⭐ 展开节点的辅助函数
const expandKeysNow = async (keys: number[]) => {
  if (keys.length === 0) {
    return
  }
  
  // ⭐ 防重复展开：如果正在展开或 keys 相同，跳过
  const keysStr = JSON.stringify(keys.sort())
  const lastKeysStr = JSON.stringify(lastExpandedKeys.sort())
  if (isExpanding || keysStr === lastKeysStr) {
    console.log('[ServiceTreePanel] 跳过重复展开:', {
      isExpanding,
      keysStr,
      lastKeysStr,
      isSame: keysStr === lastKeysStr
    })
    return
  }
  
  isExpanding = true
  lastExpandedKeys = [...keys]
  
  try {
    if (!treeRef.value) {
      console.warn('[ServiceTreePanel] treeRef.value 未初始化，等待...')
      // 等待 treeRef 初始化
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 100))
      if (!treeRef.value) {
        console.error('[ServiceTreePanel] treeRef.value 仍未初始化，无法展开节点')
        return
      }
    }
    
    if (!groupedTreeData.value.length) {
      console.warn('[ServiceTreePanel] groupedTreeData 为空，等待数据加载...')
      // 等待数据加载
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 100))
      if (!groupedTreeData.value.length) {
        console.error('[ServiceTreePanel] groupedTreeData 仍为空，无法展开节点')
        return
      }
    }
    
    console.log(`[ServiceTreePanel] 准备展开 ${keys.length} 个节点:`, keys)
    console.log('[ServiceTreePanel] treeRef.value 状态:', {
      exists: !!treeRef.value,
      hasSetExpandedKeys: !!(treeRef.value && treeRef.value.setExpandedKeys),
      dataLength: groupedTreeData.value.length
    })
    
    // 等待 DOM 渲染完成
    await nextTick()
    await new Promise(resolve => setTimeout(resolve, 200)) // 给树组件一些时间渲染
    
    if (treeRef.value && treeRef.value.setExpandedKeys) {
      try {
        treeRef.value.setExpandedKeys(keys, false) // false 表示不触发 expand 事件
        console.log(`[ServiceTreePanel] ✅ 已调用 setExpandedKeys，展开节点数:`, keys.length)
      } catch (error) {
        console.error('[ServiceTreePanel] setExpandedKeys 调用失败:', error)
        // 回退方案：使用 expandPathOnly 批量展开（不选中节点，避免节点切换）
        console.warn('[ServiceTreePanel] 回退到 expandPathOnly 方式')
        // ⭐ 批量展开所有路径，而不是逐个展开，减少节点切换
        const paths: number[][] = []
        for (const nodeId of keys) {
          const path = findPathToNode(groupedTreeData.value, nodeId)
          if (path.length > 0) {
            paths.push(path)
          }
        }
        // 一次性展开所有路径（不选中节点）
        for (const path of paths) {
          await expandPathOnly(treeRef.value, path)
        }
      }
    } else {
      console.warn('[ServiceTreePanel] treeRef.value.setExpandedKeys 不可用，尝试使用 expandPathOnly')
      // 回退方案：使用 expandPathOnly 批量展开（不选中节点，避免节点切换）
      const paths: number[][] = []
      for (const nodeId of keys) {
        const path = findPathToNode(groupedTreeData.value, nodeId)
        if (path.length > 0) {
          paths.push(path)
        }
      }
      // 一次性展开所有路径（不选中节点）
      for (const path of paths) {
        await expandPathOnly(treeRef.value, path)
      }
    }
  } finally {
    isExpanding = false
  }
}

// 🔥 监听 expandedKeys 变化，自动展开节点
watch(() => props.expandedKeys, async (keys: number[] | undefined, oldKeys: number[] | undefined) => {
  if (keys && keys.length > 0) {
    // ⭐ 防重复：如果 keys 和 oldKeys 相同，跳过
    const keysStr = JSON.stringify(keys.sort())
    const oldKeysStr = oldKeys ? JSON.stringify(oldKeys.sort()) : ''
    if (keysStr === oldKeysStr) {
      console.log('[ServiceTreePanel] expandedKeys 未变化，跳过展开')
      return
    }
    
    console.log(`[ServiceTreePanel] expandedKeys 变化:`, {
      oldKeys: oldKeys?.length || 0,
      newKeys: keys.length,
      keys: keys
    })
    // 无论树数据是否已加载，都尝试展开（expandKeysNow 内部会等待）
    await expandKeysNow(keys)
  }
}, { immediate: true })

// 🔥 监听服务树数据变化，如果 currentNodeId 存在但还没展开，重新尝试
watch(() => groupedTreeData.value, async (newTreeData: ServiceTree[]) => {
  if (newTreeData.length > 0 && treeRef.value) {
    await nextTick()
    
    // ⭐ 优先使用后端返回的 expanded_keys（如果存在）
    if (props.expandedKeys && props.expandedKeys.length > 0) {
      console.log(`[ServiceTreePanel] 服务树数据变化，使用后端返回的 expanded_keys，展开 ${props.expandedKeys.length} 个节点`)
      await expandKeysNow(props.expandedKeys)
    } else {
      // ⭐ 如果没有后端返回的 expanded_keys，使用前端计算的方式（兼容旧逻辑）
      await expandNodesWithPendingCount(newTreeData)
    }
    
    // 如果 currentNodeId 存在，展开并选中当前节点
    if (props.currentNodeId) {
      const path = findPathToNode(newTreeData, props.currentNodeId)
      if (path.length > 0) {
        await expandPathAndSelect(
          treeRef.value,
          newTreeData,
          path,
          Number(props.currentNodeId)
        )
      }
    }
  }
})

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
  padding: 16px;
  border-bottom: 1px solid var(--el-border-color-light);
  display: flex;
  align-items: center;
  justify-content: space-between;
  
  h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
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
  padding-bottom: 100px; /* ✅ 为左下角 AppSwitcher 留出空间，避免底部内容被遮挡 */
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
  
  .hub-badge {
    margin-left: 6px;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 3px;
    transition: all 0.2s;
    flex-shrink: 0;
    padding: 2px 4px;
    border-radius: 3px;
    color: var(--el-color-primary);
    
    &:hover {
      background-color: var(--el-color-primary-light-9);
      color: var(--el-color-primary);
    }
    
    .hub-icon {
      font-size: 13px;
      color: var(--el-color-primary);
    }
    
    .hub-version {
      font-size: 10px;
      color: var(--el-text-color-secondary);
      margin-left: 2px;
      font-weight: 500;
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
    opacity: 0;
    transition: opacity 0.2s;
    position: relative; /* 确保下拉菜单定位正确 */
    z-index: 10; /* 确保下拉菜单在最上层 */
    pointer-events: auto; /* 确保可以点击 */
    
    .more-icon {
      font-size: 14px;
      color: var(--el-text-color-secondary);
      cursor: pointer;
      padding: 4px;
      pointer-events: auto; /* 确保可以点击 */
      
      &:hover {
        color: var(--el-color-primary);
      }
    }
  }
  
  &:hover .node-more-actions {
    opacity: 1;
  }
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
    
    .tree-node .node-more-actions {
      opacity: 1;
    }
  }
}

:deep(.el-tree-node.is-current > .el-tree-node__content) {
  background-color: var(--el-fill-color-lighter);
  border-left: 2px solid #6366f1;
  
  .tree-node {
    .node-label {
      color: var(--el-text-color-primary);
      font-weight: 500;
    }
    
    .node-icon {
      color: #6366f1;  /* ✅ 旧版本紫色主题色 */
      opacity: 0.8;
    }
    
    /* 确保高亮节点时下拉菜单也能正常显示 */
    .node-more-actions {
      opacity: 1 !important; /* 高亮节点时始终显示下拉按钮 */
      z-index: 100; /* 确保下拉菜单在最上层 */
      pointer-events: auto !important; /* 确保可以点击 */
      
      .more-icon {
        pointer-events: auto !important; /* 确保图标可以点击 */
      }
    }
  }
}

/* 确保子节点不受父节点选中状态影响 */
:deep(.el-tree-node.is-current .el-tree-node__children .el-tree-node__content) {
  background-color: transparent;
  border-left: none;
}

/* 下拉菜单样式修复 */
:deep(.el-dropdown-menu),
:global(.service-tree-dropdown-popper .el-dropdown-menu) {
  min-width: 160px;
  z-index: 9999 !important; /* 确保下拉菜单在最上层 */
}

:deep(.el-dropdown-menu__item),
:global(.service-tree-dropdown-popper .el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  white-space: nowrap;
  
  .el-icon {
    font-size: 14px;
  }
}
</style>
