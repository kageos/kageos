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
          @click="handleForkButtonClick"
          class="header-link"
        >
          <el-icon><CopyDocument /></el-icon>
          闪电克隆
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
            <!-- 分组节点：显示分组图标和组名 -->
            <template v-if="(data as any).isGroup">
              <el-icon class="node-icon group-icon">
                <FolderOpened />
              </el-icon>
              <span class="node-label group-label">{{ node.label }}</span>
              <el-tag type="info" size="small" class="group-tag">业务系统</el-tag>
            </template>
            <!-- 普通节点 -->
            <template v-else>
              <!-- package 类型：显示文件夹图标 -->
              <el-icon v-if="data.type === 'package'" class="node-icon" :class="getNodeIconClass(data)">
                <Folder />
              </el-icon>
              <!-- function 类型：根据 template_type 显示不同图标 -->
              <el-icon v-else-if="data.type === 'function'" 
                       class="node-icon" 
                       :class="getNodeIconClass(data)">
                <component :is="getFunctionIcon(data)" />
              </el-icon>
              <!-- 其他类型：显示 fx 文本 -->
              <span v-else class="node-icon fx-icon" :class="getNodeIconClass(data)">fx</span>
              <span class="node-label">{{ node.label }}</span>
            </template>
            
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
                  <el-dropdown-item v-if="!(data as any).isGroup && data.type === 'package'" command="create-directory">
                    <el-icon><Plus /></el-icon>
                    添加服务目录
                  </el-dropdown-item>
                  <!-- 仅对函数组（业务系统）显示发布到应用中心选项 -->
                  <el-dropdown-item v-if="(data as any).isGroup && (data as any).full_group_code" command="publish-to-hub" divided>
                    <el-icon><Upload /></el-icon>
                    发布到应用中心
                  </el-dropdown-item>
                  <el-dropdown-item command="copy-link">
                    <el-icon><Link /></el-icon>
                    复制链接
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
import { ref, watch, nextTick, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Folder, FolderOpened, Plus, MoreFilled, Link, CopyDocument, Upload, Grid, Postcard, Document } from '@element-plus/icons-vue'
import { ElTag, ElLink } from 'element-plus'
import { generateGroupId, createGroupNode, groupFunctionsByCode, getGroupName, type ExtendedServiceTree } from '@/utils/tree-utils'
import type { ServiceTree } from '@/types'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import {
  findPathToNode,
  expandParentNodes,
  findNodeByPath,
  findGroupByFullGroupCode,
  findParentNode,
  expandPathAndSelect
} from '@/utils/serviceTreeUtils'
import { extractFullGroupCodeFromRoute } from '@/utils/route'

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
  (e: 'fork-group', node: ServiceTree | null): void  // Fork 业务系统（可以为 null，表示打开对话框让用户选择）
  (e: 'publish-to-hub', node: ServiceTree): void   // 发布到应用中心
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const router = useRouter()
const route = useRoute()

// el-tree 的引用
const treeRef = ref()

/**
 * 🔥 按组分组处理服务树数据
 * 将相同 full_group_code 的函数分组显示，组名使用 group_name
 */
const groupedTreeData = computed(() => {
  const processNode = (node: ServiceTree): ServiceTree => {
    // 如果是 package 且有子节点，需要分组处理
    if (node.type === 'package' && node.children && node.children.length > 0) {
      // 分离函数和包
      const functions = node.children.filter(child => child.type === 'function')
      const packages = node.children.filter(child => child.type === 'package')
      
      // 按 full_group_code 分组函数
      const groupedFunctions = new Map<string, ServiceTree[]>()
      const ungroupedFunctions: ServiceTree[] = []
      
      functions.forEach(func => {
        if (func.full_group_code && func.full_group_code.trim() !== '') {
          if (!groupedFunctions.has(func.full_group_code)) {
            groupedFunctions.set(func.full_group_code, [])
          }
          groupedFunctions.get(func.full_group_code)!.push(func)
        } else {
          ungroupedFunctions.push(func)
        }
      })
      
      // 构建新的 children 数组
      const newChildren: ServiceTree[] = []
      
      // 1. 先添加包（保持原有顺序）
      packages.forEach(pkg => {
        newChildren.push(processNode(pkg))
      })
      
      // 2. 添加分组后的函数
      groupedFunctions.forEach((funcs, groupCode) => {
        const groupName = getGroupName(funcs, groupCode)
        const groupNode = createGroupNode(groupCode, groupName, node, true)
        // 业务系统下包含函数节点
        groupNode.children = funcs.map(func => processNode(func))
        newChildren.push(groupNode)
      })
      
      // 3. 添加未分组的函数
      ungroupedFunctions.forEach(func => {
        newChildren.push(processNode(func))
      })
      
      return {
        ...node,
        children: newChildren
      }
    }
    
    // 如果是函数或没有子节点，直接返回
    return node
  }
  
  return props.treeData.map(node => processNode(node))
})

const handleNodeClick = (data: ServiceTree) => {
  // 如果是函数组（isGroup && full_group_code），更新路由
  if ((data as any).isGroup && (data as any).full_group_code) {
    const fullGroupCode = (data as any).full_group_code
    // 使用 full_group_code 作为路径，例如：/luobei/demo/crm/crm_ticket -> /workspace/luobei/demo/crm/crm_ticket
    const targetPath = `/workspace${fullGroupCode}`
    // 更新路由，只保留 _node_type=function_group 参数，清除其他所有参数
    router.push({
      path: targetPath,
      query: {
        _node_type: 'function_group'
      }
    })
    // 定位并展开函数组
    nextTick(() => {
      expandPaths([fullGroupCode])
    })
    return // 不继续触发 node-click 事件
  }
  
  // 如果是 package 类型，直接触发 node-click 事件，让父组件处理路由跳转
  // 其他节点类型，也触发 node-click 事件
  emit('node-click', data)
}

const handleNodeAction = (command: string, data: ServiceTree) => {
  if (command === 'create-directory') {
    emit('create-directory', data)
  } else if (command === 'copy-link') {
    emit('copy-link', data)
  } else if (command === 'fork') {
    emit('fork-group', data)
  } else if (command === 'publish-to-hub') {
    emit('publish-to-hub', data)
  }
}

// 处理克隆按钮点击（直接打开克隆对话框，不需要选中节点）
const handleForkButtonClick = () => {
  // 如果有选中的函数组节点，使用它；否则传递 null，让对话框自己处理
  if (props.currentFunction) {
    const data = props.currentFunction as any
    // 如果当前选中的是业务系统节点，直接使用它
    if (data.isGroup && data.full_group_code) {
      emit('fork-group', props.currentFunction)
      return
    }
  }
  // 否则传递 null，打开对话框让用户选择要克隆的业务系统
  emit('fork-group', null)
}

// 获取函数图标组件（根据 template_type）
const getFunctionIcon = (data: ServiceTree) => {
  if (data.template_type === TEMPLATE_TYPE.TABLE) {
    return Grid
  } else if (data.template_type === TEMPLATE_TYPE.FORM) {
    return Postcard
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
    }
    return 'function-icon'
  }
  return 'function-icon'
  }
  
// 使用工具函数：findPathToNode, expandParentNodes, findNodeByPath, findGroupByFullGroupCode, findParentNode
// 这些函数已从 @/utils/serviceTreeUtils 导入

// 展开多个路径
const expandPaths = async (paths: string[]) => {
  if (!treeRef.value || !groupedTreeData.value.length) {
    return
  }
  
  for (const path of paths) {
    // 先尝试根据 full_group_code 查找函数组
    const groupNode = findGroupByFullGroupCode(groupedTreeData.value, path)
    if (groupNode) {
      // 找到函数组节点，需要展开其父节点（package）
      const parentPackage = findParentNode(groupedTreeData.value, Number(groupNode.id))
      if (parentPackage) {
        // 先展开父节点（package）
        const parentPath = findPathToNode(groupedTreeData.value, Number(parentPackage.id))
        if (parentPath.length > 0) {
          expandParentNodes(treeRef.value, parentPath)
          // 等待父节点展开后，再展开并选中函数组
          await expandPathAndSelect(
            treeRef.value,
            groupedTreeData.value,
            [Number(parentPackage.id)],
            Number(groupNode.id)
          )
        }
      }
      continue
    }
    
    // 如果不是函数组，尝试根据 full_code_path 查找
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

// 监听路由查询参数中的 full_group_code，自动定位并展开函数组
watch(() => route.query.full_group_code, (fullGroupCode) => {
  if (fullGroupCode && typeof fullGroupCode === 'string' && groupedTreeData.value.length > 0) {
    nextTick(() => {
      expandPaths([fullGroupCode])
    })
  }
}, { immediate: true })

// 🔥 监听 _node_type=function_group，从路由路径中提取 full_group_code 并展开
watch(() => [route.query._node_type, route.path, groupedTreeData.value.length], ([nodeType, path, treeLength]) => {
  if (nodeType === 'function_group' && treeLength > 0) {
    // 从路由路径中提取 full_group_code
    const fullGroupCode = extractFullGroupCodeFromRoute(path as string)
    if (fullGroupCode) {
      nextTick(() => {
        expandPaths([fullGroupCode])
      })
      }
    }
}, { immediate: true })

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
    
    &.table-icon {
      color: #10b981; /* green-500 - 表格用绿色 */
      opacity: 0.9;
    }
    
    &.form-icon {
      color: #3b82f6; /* blue-500 - 表单用蓝色 */
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
