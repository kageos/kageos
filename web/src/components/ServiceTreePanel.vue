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
            <el-icon class="node-icon">
              <FolderOpened v-if="data.type === 'package'" />
              <Document v-else />
            </el-icon>
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
import { FolderOpened, Document, Plus, MoreFilled, Link } from '@element-plus/icons-vue'
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

// 监听 currentNodeId 变化，自动展开并选中节点
watch(() => props.currentNodeId, (nodeId) => {
  if (nodeId && treeRef.value && props.treeData.length > 0) {
    nextTick(() => {
      console.log('[ServiceTreePanel] 定位到节点:', nodeId)
      // 查找路径
      const path = findPathToNode(props.treeData, nodeId)
      console.log('[ServiceTreePanel] 节点路径:', path)
      
      if (path.length > 0) {
        // 展开所有父节点（除了当前节点本身）
        const expandKeys = path.slice(0, -1)
        console.log('[ServiceTreePanel] 展开节点:', expandKeys)
        if (expandKeys.length > 0) {
          treeRef.value.store.nodesMap[expandKeys[0]]?.expand()
          expandKeys.forEach((key: number) => {
            const node = treeRef.value.store.nodesMap[key]
            if (node) {
              node.expand()
            }
          })
        }
        
        // 选中当前节点
        console.log('[ServiceTreePanel] 选中节点:', nodeId)
        treeRef.value.setCurrentKey(nodeId)
      }
    })
  }
}, { immediate: true })
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
    font-size: 16px;
    color: var(--el-color-primary);  // ✅ 使用深色主题色（现代风格）
    flex-shrink: 0;
    transition: color 0.2s ease;
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
  background-color: var(--el-color-primary-light-9);
  
  .tree-node {
    .node-label {
      color: var(--el-color-primary);
      font-weight: 500;
    }
    
    .node-icon {
      color: var(--el-color-primary);
    }
  }
}

// ✅ 为图标添加更深的颜色（现代风格）
.tree-node .node-icon {
  // 深色模式下的颜色
  @media (prefers-color-scheme: dark) {
    color: #409eff;  // 深蓝色，更现代
  }
  
  // 浅色模式下的颜色
  @media (prefers-color-scheme: light) {
    color: #409eff;  // 保持一致的深蓝色
  }
}

// ✅ 选中状态下的图标颜色更深
:deep(.el-tree-node.is-current > .el-tree-node__content) {
  .tree-node .node-icon {
    color: #2b85e4;  // 更深的蓝色
  }
}

:deep(.el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
