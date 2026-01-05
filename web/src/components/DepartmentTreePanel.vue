<!--
  DepartmentTreePanel - 组织架构树形面板组件
  
  需求：
  - 使用类似服务目录的树形结构展示组织架构
  - 支持点击节点查看部门用户
  - 支持右键菜单操作（编辑、删除、查看用户等）
  
  设计思路：
  - 参考 ServiceTreePanel 的实现
  - 使用 el-tree 展示部门层级
  - 支持节点点击、右键菜单等交互
-->
<template>
  <div class="department-tree-panel" v-loading="loading">
    <div class="tree-header">
      <div class="header-title">
        <img src="/组织架构.svg" alt="组织架构" class="header-icon" />
        <h3>组织架构</h3>
      </div>
      <div class="header-actions">
        <el-link
          v-if="!loading"
          type="primary"
          :underline="false"
          @click="$emit('create-department')"
          class="header-link"
        >
          <el-icon><Plus /></el-icon>
          新增部门
        </el-link>
        <el-link
          v-if="!loading"
          type="primary"
          :underline="false"
          @click="$emit('refresh')"
          class="header-link"
        >
          <el-icon><Refresh /></el-icon>
          刷新
        </el-link>
      </div>
    </div>
    
    <div class="tree-content">
      <el-tree
        v-if="treeData.length > 0"
        ref="treeRef"
        :data="treeData"
        :props="{ children: 'children', label: 'name' }"
        node-key="id"
        :default-expand-all="true"
        :expand-on-click-node="false"
        :highlight-current="true"
        @node-click="handleNodeClick"
      >
        <template #default="{ node, data }">
          <span class="tree-node">
            <!-- 部门图标 -->
            <img 
              src="/组织架构.svg" 
              alt="部门" 
              class="node-icon department-icon-img"
            />
            <span class="node-label">{{ node.label }}</span>
            <span class="node-code">({{ data.code }})</span>
            
            <!-- 更多操作按钮 - 鼠标悬停时显示 -->
            <el-dropdown
              trigger="click"
              :teleported="true"
              popper-class="department-tree-dropdown-popper"
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
                  <el-dropdown-item command="view-users">
                    <el-icon><User /></el-icon>
                    查看用户
                  </el-dropdown-item>
                  <el-dropdown-item command="create-child" divided>
                    <el-icon><Plus /></el-icon>
                    添加子部门
                  </el-dropdown-item>
                  <el-dropdown-item command="edit" divided>
                    <el-icon><Edit /></el-icon>
                    编辑
                  </el-dropdown-item>
                  <el-dropdown-item 
                    command="delete" 
                    divided
                    :disabled="data.is_system_default"
                  >
                    <el-icon><Delete /></el-icon>
                    删除
                    <span v-if="data.is_system_default" class="disabled-hint">（系统默认组织不可删除）</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </span>
        </template>
      </el-tree>
      
      <div v-else class="empty-state">
        <el-empty description="暂无组织架构" :image-size="80">
          <el-button type="primary" @click="$emit('create-department')">
            <el-icon><Plus /></el-icon>
            创建部门
          </el-button>
        </el-empty>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Plus, MoreFilled, Refresh, User, Edit, Delete } from '@element-plus/icons-vue'
import type { Department } from '@/api/department'

interface Props {
  treeData: Department[]
  loading?: boolean
  currentNodeId?: number | null
}

interface Emits {
  (e: 'node-click', node: Department): void
  (e: 'create-department', parentNode?: Department): void
  (e: 'view-users', node: Department): void
  (e: 'edit', node: Department): void
  (e: 'delete', node: Department): void
  (e: 'refresh'): void
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  currentNodeId: null
})

const emit = defineEmits<Emits>()

// el-tree 的引用
const treeRef = ref()

const handleNodeClick = (data: Department) => {
  emit('node-click', data)
}

const handleNodeAction = (command: string, data: Department) => {
  if (command === 'view-users') {
    emit('view-users', data)
  } else if (command === 'create-child') {
    emit('create-department', data)
  } else if (command === 'edit') {
    emit('edit', data)
  } else if (command === 'delete') {
    emit('delete', data)
  }
}

// 暴露方法给父组件
defineExpose({
  treeRef
})
</script>

<style scoped>
.department-tree-panel {
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
  
  .header-title {
    display: flex;
    align-items: center;
    gap: 8px;
    
    .header-icon {
      width: 20px;
      height: 20px;
      flex-shrink: 0;
    }
    
    h3 {
      margin: 0;
      font-size: 16px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
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
    color: #6366f1 !important;
    
    &:hover {
      color: #4f46e5 !important;
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
  overflow-x: visible;
  padding: 8px;
  display: flex;
  flex-direction: column;
  position: relative;
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
    color: #6366f1;
    opacity: 0.8;
    flex-shrink: 0;
    transition: color 0.2s ease;
    
    &.department-icon-img {
      width: 16px;
      height: 16px;
      object-fit: contain;
      opacity: 0.9;
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
  
  .node-code {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    flex-shrink: 0;
  }
  
  .node-more-actions {
    flex-shrink: 0;
    opacity: 0;
    transition: opacity 0.2s;
    position: relative;
    z-index: 10;
    pointer-events: auto;
    
    .more-icon {
      font-size: 14px;
      color: var(--el-text-color-secondary);
      cursor: pointer;
      padding: 4px;
      pointer-events: auto;
      
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
  position: relative;
  overflow: visible;
  
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
      opacity: 0.8;
      filter: brightness(0.9);
    }
    
    .node-more-actions {
      opacity: 1 !important;
      z-index: 100;
      pointer-events: auto !important;
      
      .more-icon {
        pointer-events: auto !important;
      }
    }
  }
}

:deep(.el-tree-node.is-current .el-tree-node__children .el-tree-node__content) {
  background-color: transparent;
  border-left: none;
}

:deep(.el-dropdown-menu),
:global(.department-tree-dropdown-popper .el-dropdown-menu) {
  min-width: 160px;
  z-index: 9999 !important;
}

:deep(.el-dropdown-menu__item),
:global(.department-tree-dropdown-popper .el-dropdown-menu__item) {
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

