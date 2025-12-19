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
        :expand-on-click-node="false"
        :highlight-current="true"
        @node-click="handleNodeClick"
      >
        <template #default="{ node, data }">
          <span class="tree-node">
            <!-- package 类型：显示自定义文件夹图标 -->
            <img 
              v-if="data.type === 'package'" 
              src="/service-tree/custom-folder.svg" 
              alt="目录" 
              class="node-icon package-icon-img"
              :class="getNodeIconClass(data)"
            />
            <!-- function 类型：根据 template_type 显示不同图标 -->
            <template v-else-if="data.type === 'function'">
              <!-- 表单类型：使用自定义 SVG -->
              <img 
                v-if="data.template_type === TEMPLATE_TYPE.FORM"
                src="/service-tree/表单 (3).svg" 
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
            <!-- 其他类型：显示 fx 文本 -->
            <span v-else class="node-icon fx-icon" :class="getNodeIconClass(data)">fx</span>
            <span class="node-label">{{ node.label }}</span>
            
            <!-- Hub 标记 - 已发布到 Hub 的目录显示 -->
            <span
              v-if="data.type === 'package' && data.hub_directory_id && data.hub_directory_id > 0"
              class="hub-badge"
              @click.stop="handleHubBadgeClick(data)"
              :title="data.hub_version ? `已发布到应用中心 ${data.hub_version}` : '已发布到应用中心'"
            >
              <el-icon class="hub-icon"><Link /></el-icon>
              <span v-if="data.hub_version" class="hub-version">{{ data.hub_version }}</span>
            </span>
            
            <!-- 更多操作按钮 - 鼠标悬停时显示 -->
            <el-dropdown
              trigger="click"
              @click.stop
              class="node-more-actions"
              @command="(command: string) => handleNodeAction(command, data)"
            >
              <el-icon class="more-icon" @click.stop>
                <MoreFilled />
              </el-icon>
              <template #dropdown>
                <el-dropdown-menu>
                  <!-- 仅对package类型显示创建子目录选项 -->
                  <el-dropdown-item v-if="data.type === 'package'" command="create-directory">
                    <el-icon><Plus /></el-icon>
                    添加服务目录
                  </el-dropdown-item>
                  <!-- 仅对package类型显示复制选项 -->
                  <el-dropdown-item v-if="data.type === 'package'" command="copy" divided>
                    <el-icon><CopyDocument /></el-icon>
                    复制
                  </el-dropdown-item>
                  <!-- 粘贴选项（当有复制的内容且当前节点是目录时显示） -->
                  <el-dropdown-item v-if="data.type === 'package' && copiedDirectory" command="paste">
                    <el-icon><Document /></el-icon>
                    粘贴到此处
                  </el-dropdown-item>
                  <el-dropdown-item command="copy-link">
                    <el-icon><Link /></el-icon>
                    复制链接
                  </el-dropdown-item>
                  <!-- 仅对package类型显示发布到Hub选项（未发布时） -->
                  <el-dropdown-item v-if="data.type === 'package' && (!data.hub_directory_id || data.hub_directory_id === 0)" command="publish-to-hub" divided>
                    <el-icon><Upload /></el-icon>
                    发布到应用中心
                  </el-dropdown-item>
                  <!-- 仅对package类型显示推送到Hub选项（已发布时） -->
                  <el-dropdown-item v-if="data.type === 'package' && data.hub_directory_id && data.hub_directory_id > 0" command="push-to-hub" divided>
                    <el-icon><Upload /></el-icon>
                    推送到应用中心
                  </el-dropdown-item>
                  <!-- 仅对package类型显示变更记录选项 -->
                  <el-dropdown-item v-if="data.type === 'package'" command="update-history" divided>
                    <el-icon><Clock /></el-icon>
                    变更记录
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
import { Plus, MoreFilled, Link, CopyDocument, Document, Clock, Upload, Download } from '@element-plus/icons-vue'
import ChartIcon from './icons/ChartIcon.vue'
import TableIcon from './icons/TableIcon.vue'
import FormIcon from './icons/FormIcon.vue'
import { ElTag, ElLink, ElMessageBox, ElMessage } from 'element-plus'
import type { ServiceTree } from '@/types'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { copyDirectory } from '@/api/service-tree'
import {
  findPathToNode,
  expandParentNodes,
  findNodeByPath,
  expandPathAndSelect
} from '@/utils/serviceTreeUtils'
import { navigateToHubDirectoryDetail } from '@/utils/hub-navigation'

interface Props {
  treeData: ServiceTree[]
  loading?: boolean
  currentNodeId?: number | string | null
  currentFunction?: ServiceTree | null  // 当前选中的节点（用于判断是否可以克隆）
}

interface Emits {
  (e: 'node-click', node: ServiceTree): void
  (e: 'create-directory', parentNode?: ServiceTree): void
  (e: 'copy-link', node: ServiceTree): void
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

// el-tree 的引用
const treeRef = ref()

// 复制粘贴相关状态
const copiedDirectory = ref<ServiceTree | null>(null)  // 复制的目录信息
const isPasting = ref(false)  // 是否正在粘贴


// 复制目录
const handleCopy = (node: ServiceTree) => {
  if (node.type !== 'package') {
    ElMessage.warning('只能复制目录（package类型）')
    return
  }
  
  copiedDirectory.value = node
  ElMessage.success(`已复制目录：${node.name}`)
}

  // 粘贴目录（使用当前选中的目录作为目标）
  const handlePaste = async (targetNode?: ServiceTree) => {
    if (!copiedDirectory.value) {
      ElMessage.warning('没有可粘贴的目录')
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


// 直接使用原始树数据，不再进行分组处理
const groupedTreeData = computed(() => props.treeData)

const handleNodeClick = (data: ServiceTree) => {
  // 直接触发 node-click 事件，让父组件处理路由跳转
  emit('node-click', data)
}

const handleNodeAction = (command: string, data: ServiceTree) => {
  if (command === 'create-directory') {
    emit('create-directory', data)
  } else if (command === 'copy') {
    handleCopy(data)
  } else if (command === 'paste') {
    // 粘贴时，如果右键的节点是目录，使用该节点；否则使用当前选中的目录
    if (data.type === 'package') {
      handlePaste(data)
    } else {
      handlePaste() // 使用当前选中的目录
    }
  } else if (command === 'copy-link') {
    emit('copy-link', data)
  } else if (command === 'publish-to-hub') {
    emit('publish-to-hub', data)
  } else if (command === 'push-to-hub') {
    emit('push-to-hub', data)
  } else if (command === 'update-history') {
    emit('update-history', data)
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

// 注册和注销键盘事件监听
onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
})

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
  }
  return 'function-icon'
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

// 🔥 监听服务树数据变化，如果 currentNodeId 存在但还没展开，重新尝试
watch(() => groupedTreeData.value, async (newTreeData) => {
  if (newTreeData.length > 0 && props.currentNodeId && treeRef.value) {
    await nextTick()
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
  padding: 8px;
  padding-bottom: 100px; /* ✅ 为左下角 AppSwitcher 留出空间，避免底部内容被遮挡 */
  display: flex;
  flex-direction: column;
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
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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
  
  .node-more-actions {
    flex-shrink: 0;
    opacity: 0;
    transition: opacity 0.2s;
    
    .more-icon {
      font-size: 14px;
      color: var(--el-text-color-secondary);
      cursor: pointer;
      padding: 4px;
      
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
  }
}

/* 确保子节点不受父节点选中状态影响 */
:deep(.el-tree-node.is-current .el-tree-node__children .el-tree-node__content) {
  background-color: transparent;
  border-left: none;
}

:deep(.el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
