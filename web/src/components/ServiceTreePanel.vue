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

// 查找从根节点到目标节点的路径
const findPathToNode = (nodes: ServiceTree[], targetId: number | string): number[] => {
  const path: number[] = []
  // 确保 targetId 转换为数字进行比较
  const targetIdNum = Number(targetId)
  
  const findNode = (nodes: ServiceTree[], targetId: number): boolean => {
    for (const node of nodes) {
      // 🔥 跳过分组节点（分组节点是虚拟节点）
      if ((node as any).isGroup) {
        // 在分组节点的子节点中查找
        if (node.children && node.children.length > 0) {
          if (findNode(node.children, targetId)) {
            path.push(Number(node.id)) // 包含分组节点到路径中
            return true
          }
        }
        continue
      }
      
      const nodeIdNum = Number(node.id)
      path.push(nodeIdNum)
      
      if (nodeIdNum === targetId) {
        return true
      }
      
      if (node.children && node.children.length > 0) {
        if (findNode(node.children, targetId)) {
          return true
        }
      }
      
      path.pop()
    }
    return false
  }
  
  findNode(nodes, targetIdNum)
  return path
}

// 🔥 展开所有父节点（递归展开）
const expandParentNodes = (path: number[]) => {
  if (path.length === 0 || !treeRef.value) return
  
  // 展开所有父节点
  const expandKeys = path.slice(0, -1) // 最后一个节点不需要展开，只需选中
  expandKeys.forEach((key: number) => {
    const node = treeRef.value.store.nodesMap[key]
    if (node && !node.expanded) {
      node.expand()
    }
  })
}

// 根据 full_code_path 查找节点并展开
const findAndExpandByPath = (targetPath: string): ServiceTree | null => {
  if (!treeRef.value || !groupedTreeData.value.length) {
    return null
  }
  
  // 规范化路径（移除开头的斜杠，确保格式一致）
  const normalizedPath = targetPath.replace(/^\/+/, '')
  
  const findNode = (nodes: ServiceTree[], path: string, depth = 0): ServiceTree | null => {
    for (const node of nodes) {
      // 规范化节点的 full_code_path（移除开头的斜杠和 __group__ 部分）
      let nodePath = node.full_code_path.replace(/^\/+/, '')
      const isGroup = (node as any).isGroup
      
      // 如果是分组节点，移除 __group__ 部分来匹配目录路径
      if (isGroup) {
        nodePath = nodePath.replace(/\/__group__[^/]+$/, '')
      }
      
      // 检查当前节点是否匹配（精确匹配或目录匹配）
      if (nodePath === path || path.startsWith(nodePath + '/')) {
        // 展开当前节点
        const nodeKey = Number(node.id)
        const treeNode = treeRef.value.store.nodesMap[nodeKey]
        if (treeNode) {
          if (!treeNode.expanded) {
            treeNode.expand()
          }
        }
        
        // 如果是精确匹配，返回该节点
        if (nodePath === path) {
          return node
        }
        
        // 如果是目录匹配，继续在子节点中查找
        if (node.children && node.children.length > 0) {
          const found = findNode(node.children, path, depth + 1)
          if (found) return found
        }
      }
    }
    return null
  }
  
  return findNode(groupedTreeData.value, normalizedPath)
}

// 根据 full_group_code 查找函数组节点
const findGroupByFullGroupCode = (fullGroupCode: string): ServiceTree | null => {
  if (!groupedTreeData.value.length) {
    return null
  }
  
  const findNode = (nodes: ServiceTree[]): ServiceTree | null => {
    for (const node of nodes) {
      // 检查是否是函数组节点且 full_group_code 匹配
      if ((node as any).isGroup && (node as any).full_group_code === fullGroupCode) {
        return node
      }
      // 递归查找子节点
      if (node.children && node.children.length > 0) {
        const found = findNode(node.children)
        if (found) return found
      }
    }
    return null
  }
  
  return findNode(groupedTreeData.value)
}

// 查找函数组的父节点（package）
const findParentPackage = (groupNode: ServiceTree): ServiceTree | null => {
  if (!groupedTreeData.value.length) {
    return null
  }
  
  const findParent = (nodes: ServiceTree[], targetId: number): ServiceTree | null => {
    for (const node of nodes) {
      // 检查当前节点的子节点中是否包含目标节点
      if (node.children && node.children.length > 0) {
        const hasTarget = node.children.some(child => Number(child.id) === targetId)
        if (hasTarget) {
          return node
        }
        // 递归查找
        const found = findParent(node.children, targetId)
        if (found) return found
      }
    }
    return null
  }
  
  return findParent(groupedTreeData.value, Number(groupNode.id))
}

// 展开多个路径
const expandPaths = (paths: string[]) => {
  if (!treeRef.value || !groupedTreeData.value.length) {
    return
  }
  
  paths.forEach((path) => {
    // 先尝试根据 full_group_code 查找函数组
    const groupNode = findGroupByFullGroupCode(path)
    if (groupNode) {
      // 找到函数组节点，需要展开其父节点（package）
      const parentPackage = findParentPackage(groupNode)
      if (parentPackage) {
        // 先展开父节点（package）
        const parentPath = findPathToNode(groupedTreeData.value, Number(parentPackage.id))
        if (parentPath.length > 0) {
          expandParentNodes(parentPath)
          // 等待父节点展开后，再展开并选中函数组
          setTimeout(() => {
            const groupNodeId = Number(groupNode.id)
            // 确保函数组节点也被展开（如果它是可展开的）
            const treeNode = treeRef.value.store.nodesMap[groupNodeId]
            if (treeNode && !treeNode.expanded && treeNode.childNodes && treeNode.childNodes.length > 0) {
              treeNode.expand()
            }
            // 选中函数组节点
            treeRef.value.setCurrentKey(groupNodeId)
          }, 200)
        }
      }
      return
    }
    
    // 如果不是函数组，尝试根据 full_code_path 查找
    const node = findAndExpandByPath(path)
    if (node) {
      // 找到节点后，展开到该节点的所有父节点
      const nodeId = Number(node.id)
      const pathToNode = findPathToNode(groupedTreeData.value, nodeId)
      if (pathToNode.length > 0) {
        expandParentNodes(pathToNode)
        // 高亮显示该节点
        setTimeout(() => {
          treeRef.value.setCurrentKey(nodeId)
        }, 100)
      }
    }
  })
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
    // 例如：/workspace/luobei/demo/crm/crm_order -> /luobei/demo/crm/crm_order
    if (path && path.startsWith('/workspace/')) {
      const fullGroupCode = path.replace('/workspace', '')
      if (fullGroupCode) {
        nextTick(() => {
          expandPaths([fullGroupCode])
        })
      }
    }
  }
}, { immediate: true })

// 监听 currentNodeId 变化，自动展开并选中节点
watch(() => props.currentNodeId, (nodeId) => {
  if (nodeId && treeRef.value && groupedTreeData.value.length > 0) {
    // 🔥 使用 nextTick 确保 DOM 已渲染
    nextTick(() => {
      // 查找路径（使用分组后的数据）
      const path = findPathToNode(groupedTreeData.value, nodeId)
      
      if (path.length > 0) {
        // 🔥 展开所有父节点
        expandParentNodes(path)
        
        // 🔥 延迟选中，确保展开动画完成
        setTimeout(() => {
          // 再次确保所有父节点已展开
          expandParentNodes(path)
          
          // 选中当前节点
          treeRef.value.setCurrentKey(nodeId)
          
          // 🔥 滚动到选中节点（可见）
          nextTick(() => {
            const selectedNode = treeRef.value.store.nodesMap[nodeId]
            if (selectedNode) {
              selectedNode.visible = true
            }
          })
        }, 100)
      }
    })
  }
}, { immediate: true })

// 🔥 监听服务树数据变化，如果 currentNodeId 存在但还没展开，重新尝试
watch(() => groupedTreeData.value, (newTreeData) => {
  if (newTreeData.length > 0 && props.currentNodeId) {
    nextTick(() => {
      const path = findPathToNode(newTreeData, props.currentNodeId)
      if (path.length > 0) {
        expandParentNodes(path)
        setTimeout(() => {
          expandParentNodes(path)
          if (treeRef.value) {
            treeRef.value.setCurrentKey(props.currentNodeId)
          }
        }, 100)
      }
    })
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
