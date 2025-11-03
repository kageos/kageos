<template>
  <div class="service-tree-panel" v-loading="loading">
    <div class="tree-header">
      <h3>服务目录</h3>
      <el-button
        v-if="!loading"
        type="primary"
        size="small"
        @click="$emit('create-directory')"
        class="create-btn"
      >
        <el-icon><Plus /></el-icon>
        创建目录
      </el-button>
    </div>
    
    <div class="tree-content">
      <el-tree
        v-if="treeData.length > 0"
        ref="treeRef"
        :data="treeData"
        :props="{ children: 'children', label: 'name' }"
        node-key="id"
        :default-expand-all="false"
        :expand-on-click-node="false"
        :highlight-current="true"
        @node-click="handleNodeClick"
      >
        <template #default="{ node, data }">
          <span class="tree-node">
            <el-icon v-if="data.type === 'package'" class="node-icon" :class="getNodeIconClass(data)">
              <Folder />
            </el-icon>
            <span v-else class="node-icon fx-icon" :class="getNodeIconClass(data)">fx</span>
            <span class="node-label">{{ node.label }}</span>
            
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
import { ref, watch, nextTick } from 'vue'
import { Folder, Plus, MoreFilled, Link } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/types'

interface Props {
  treeData: ServiceTree[]
  loading?: boolean
  currentNodeId?: number | string | null
}

interface Emits {
  (e: 'node-click', node: ServiceTree): void
  (e: 'create-directory', parentNode?: ServiceTree): void
  (e: 'copy-link', node: ServiceTree): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// el-tree 的引用
const treeRef = ref()

const handleNodeClick = (data: ServiceTree) => {
  emit('node-click', data)
}

const handleNodeAction = (command: string, data: ServiceTree) => {
  if (command === 'create-directory') {
    emit('create-directory', data)
  } else if (command === 'copy-link') {
    emit('copy-link', data)
  }
}

// 获取节点图标样式类
const getNodeIconClass = (data: ServiceTree) => {
  if (data.type === 'package') {
    return 'package-icon'
  } else {
    return 'function-icon fx-icon'
  }
}

// 查找从根节点到目标节点的路径
const findPathToNode = (nodes: ServiceTree[], targetId: number | string): number[] => {
  const path: number[] = []
  // 确保 targetId 转换为数字进行比较
  const targetIdNum = Number(targetId)
  
  const findNode = (nodes: ServiceTree[], targetId: number): boolean => {
    for (const node of nodes) {
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

// 监听 currentNodeId 变化，自动展开并选中节点
watch(() => props.currentNodeId, (nodeId) => {
  if (nodeId && treeRef.value && props.treeData.length > 0) {
    // 🔥 使用 nextTick 确保 DOM 已渲染
    nextTick(() => {
      console.log('[ServiceTreePanel] 定位到节点:', nodeId)
      // 查找路径
      const path = findPathToNode(props.treeData, nodeId)
      console.log('[ServiceTreePanel] 节点路径:', path)
      
      if (path.length > 0) {
        // 🔥 展开所有父节点
        expandParentNodes(path)
        
        // 🔥 延迟选中，确保展开动画完成
        setTimeout(() => {
          // 再次确保所有父节点已展开
          expandParentNodes(path)
          
          // 选中当前节点
          console.log('[ServiceTreePanel] 选中节点:', nodeId)
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
watch(() => props.treeData, (newTreeData) => {
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
  
  .create-btn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }
}

.tree-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
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
      color: #6366f1;
      opacity: 0.8;
    }
    
    &.form-icon {
      color: #6366f1;
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
