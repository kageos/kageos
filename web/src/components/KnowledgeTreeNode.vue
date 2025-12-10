<template>
  <div class="tree-node">
    <!-- 节点内容 -->
    <div 
      class="node-content"
      :class="{ 
        'is-selected': isSelected,
        'has-children': hasChildren 
      }"
      :style="{ paddingLeft: `${level * 20 + 12}px` }"
      @click="handleClick"
    >
      <!-- 展开/收起图标 -->
      <div class="expand-icon" @click.stop="handleToggle">
        <el-icon v-if="hasChildren" :class="{ 'is-expanded': node.expanded }">
          <CaretRight />
        </el-icon>
        <div v-else class="icon-placeholder"></div>
      </div>

      <!-- 节点图标 -->
      <div class="node-icon">
        <el-icon>
          <component :is="getNodeIcon()" />
        </el-icon>
      </div>

      <!-- 节点标签 -->
      <span class="node-label">{{ node.label || node.title }}</span>

      <!-- 文件类型标签 -->
      <el-tag 
        v-if="node.file_type"
        size="small"
        class="file-type-tag"
      >
        {{ node.file_type.toUpperCase() }}
      </el-tag>

      <!-- 状态标签 -->
      <el-tag 
        v-if="node.status"
        :type="getStatusType(node.status)"
        size="small"
        class="status-tag"
      >
        {{ getStatusText(node.status) }}
      </el-tag>

      <!-- 操作按钮 -->
      <div class="node-actions" v-if="isSelected || isHovered" @click.stop>
        <el-dropdown @command="handleAction" trigger="click">
          <el-button 
            type="text" 
            size="small" 
            class="action-btn"
            @mouseenter="isHovered = true"
            @mouseleave="isHovered = false"
          >
            <el-icon><MoreFilled /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item 
                command="edit"
              >
                <el-icon><Edit /></el-icon>
                编辑
              </el-dropdown-item>
              <el-dropdown-item 
                command="view"
              >
                <el-icon><View /></el-icon>
                查看
              </el-dropdown-item>
              <el-dropdown-item 
                command="delete"
                class="delete-item"
                divided
              >
                <el-icon><Delete /></el-icon>
                删除
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <!-- 子节点 -->
    <div v-if="hasChildren && node.expanded" class="children">
      <KnowledgeTreeNode
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :level="level + 1"
        :selected-id="selectedId"
        @select="$emit('select', $event)"
        @toggle="$emit('toggle', $event)"
        @action="$emit('action', $event, child)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { 
  CaretRight, 
  Folder, 
  Document, 
  MoreFilled, 
  Plus, 
  Edit, 
  Delete,
  View
} from '@element-plus/icons-vue'

interface TreeNodeData {
  id: string | number
  label?: string
  title?: string
  icon?: string
  type?: string
  file_type?: string
  status?: string
  children?: TreeNodeData[]
  expanded?: boolean
  [key: string]: any
}

interface Props {
  node: TreeNodeData
  level: number
  selectedId: string | number | null
}

interface Emits {
  (e: 'select', node: TreeNodeData): void
  (e: 'toggle', node: TreeNodeData): void
  (e: 'action', action: string, node: TreeNodeData): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const isHovered = ref(false)

const isSelected = computed(() => props.selectedId === props.node.id)
const hasChildren = computed(() => props.node.children && props.node.children.length > 0)

const getNodeIcon = () => {
  if (props.node.icon) {
    return props.node.icon
  }
  // 如果有子节点，显示文件夹图标，否则显示文档图标
  return hasChildren.value ? Folder : Document
}

const getStatusType = (status: string) => {
  const statusMap: Record<string, string> = {
    completed: 'success',
    failed: 'danger',
    processing: 'warning',
    active: 'success',
    inactive: 'info'
  }
  return statusMap[status] || 'info'
}

const getStatusText = (status: string) => {
  const statusMap: Record<string, string> = {
    processing: '处理中',
    completed: '已完成',
    failed: '失败',
    active: '活跃',
    inactive: '非活跃'
  }
  return statusMap[status] || status
}

const handleClick = () => {
  emit('select', props.node)
}

const handleToggle = () => {
  if (hasChildren.value) {
    emit('toggle', props.node)
  }
}

const handleAction = (action: string) => {
  emit('action', action, props.node)
}
</script>

<style lang="scss" scoped>
.tree-node {
  .node-content {
    display: flex;
    align-items: center;
    height: 36px;
    padding: 4px 8px 4px 12px;
    cursor: pointer;
    border-radius: 6px;
    margin: 1px 0;
    transition: all 0.2s ease;
    position: relative;

    &:hover {
      background-color: var(--el-fill-color-light);
    }

    &.is-selected {
      background-color: var(--el-fill-color) !important;
      border: 1px solid var(--el-border-color);
      color: var(--el-text-color-primary);
      font-weight: 500;
      
      // 添加一个左侧的彩色条
      &::before {
        content: '';
        position: absolute;
        left: 0;
        top: 0;
        bottom: 0;
        width: 3px;
        background-color: var(--el-color-primary);
        border-radius: 0 2px 2px 0;
      }
    }

    .expand-icon {
      width: 16px;
      height: 16px;
      display: flex;
      align-items: center;
      justify-content: center;
      margin-right: 4px;
      cursor: pointer;
      border-radius: 2px;
      transition: all 0.2s ease;

      &:hover {
        background-color: var(--el-fill-color);
      }

      .el-icon {
        transition: transform 0.2s ease;
        color: var(--el-text-color-secondary);

        &.is-expanded {
          transform: rotate(90deg);
        }
      }

      .icon-placeholder {
        width: 16px;
        height: 16px;
      }
    }

    .node-icon {
      width: 16px;
      height: 16px;
      margin-right: 8px;
      display: flex;
      align-items: center;
      justify-content: center;

      .el-icon {
        color: var(--el-color-primary);
        font-size: 16px;
      }
    }

    .node-label {
      flex: 1;
      font-size: 14px;
      color: var(--el-text-color-primary);
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      min-width: 0;
    }

    .file-type-tag {
      margin-left: 8px;
      flex-shrink: 0;
    }

    .status-tag {
      margin-left: 4px;
      flex-shrink: 0;
    }

    .node-actions {
      opacity: 0;
      transition: opacity 0.2s ease;
      margin-left: 8px;

      .action-btn {
        padding: 4px;
        min-height: auto;
        color: var(--el-text-color-secondary);

        &:hover {
          color: var(--el-color-primary);
          background-color: var(--el-fill-color);
        }
      }
    }

    &:hover .node-actions,
    &.is-selected .node-actions {
      opacity: 1;
    }
  }

  .children {
    margin-left: 0;
  }
}

:deep(.el-dropdown-menu__item.delete-item) {
  color: var(--el-color-danger);

  &:hover {
    background-color: var(--el-color-danger-light-9);
    color: var(--el-color-danger);
  }
}
</style>

