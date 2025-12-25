<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { ElTree, ElMessage } from 'element-plus'
import { Folder, Document, Plus, Edit, Delete } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/types'
import { hasPermission, DirectoryPermissions } from '@/utils/permission'

interface Props {
  data: ServiceTree[]
  loading?: boolean
  showActions?: boolean
  defaultExpandedKeys?: number[]
  selectable?: boolean
  draggable?: boolean
}

interface Emits {
  (e: 'node-click', data: ServiceTree): void
  (e: 'node-expand', data: ServiceTree): void
  (e: 'node-collapse', data: ServiceTree): void
  (e: 'node-create', parentId: number): void
  (e: 'node-edit', data: ServiceTree): void
  (e: 'node-delete', data: ServiceTree): void
  (e: 'node-drag', source: ServiceTree, target: ServiceTree): void
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  showActions: true,
  defaultExpandedKeys: () => [],
  selectable: true,
  draggable: false
})

const emit = defineEmits<Emits>()

// Tree组件ref
const treeRef = ref<InstanceType<typeof ElTree>>()

// 展开的节点
const expandedKeys = ref<number[]>([...props.defaultExpandedKeys])

// 当前选中的节点
const currentKey = ref<number>()

// 计算Tree组件的props
const treeProps = {
  children: 'children',
  label: 'name'
}

// 过滤文本
const filterText = ref('')

// 节点类型图标
const getNodeIcon = (type: string) => {
  return type === 'package' ? Folder : Document
}

// 节点类型标签
const getNodeTypeLabel = (type: string) => {
  return type === 'package' ? '目录' : '功能'
}

// 节点类型标签样式
const getNodeTypeTagType = (type: string) => {
  return type === 'package' ? 'primary' : 'success'
}

// 处理节点点击
const handleNodeClick = (data: ServiceTree) => {
  if (props.selectable) {
    currentKey.value = data.id
    emit('node-click', data)
  }
}

// 处理节点展开
const handleNodeExpand = (data: ServiceTree) => {
  if (!expandedKeys.value.includes(data.id)) {
    expandedKeys.value.push(data.id)
  }
  emit('node-expand', data)
}

// 处理节点折叠
const handleNodeCollapse = (data: ServiceTree) => {
  const index = expandedKeys.value.indexOf(data.id)
  if (index > -1) {
    expandedKeys.value.splice(index, 1)
  }
  emit('node-collapse', data)
}

// 处理创建子节点
const handleCreateNode = (data: ServiceTree, event: Event) => {
  event.stopPropagation()
  emit('node-create', data.id)
}

// 处理编辑节点
const handleEditNode = (data: ServiceTree, event: Event) => {
  event.stopPropagation()
  emit('node-edit', data)
}

// 处理删除节点
const handleDeleteNode = (data: ServiceTree, event: Event) => {
  event.stopPropagation()
  emit('node-delete', data)
}

// 处理拖拽
const handleNodeDrop = (draggingNode: any, dropNode: any, type: string) => {
  emit('node-drag', draggingNode.data, dropNode.data)
}

// 过滤节点
const filterNode = (value: string, data: ServiceTree) => {
  if (!value) return true
  return data.name.toLowerCase().includes(value.toLowerCase()) ||
         data.code?.toLowerCase().includes(value.toLowerCase()) ||
         data.description?.toLowerCase().includes(value.toLowerCase())
}

// 展开所有节点
const expandAll = () => {
  const allKeys: number[] = []
  const traverse = (nodes: ServiceTree[]) => {
    nodes.forEach(node => {
      allKeys.push(node.id)
      if (node.children && node.children.length > 0) {
        traverse(node.children)
      }
    })
  }
  traverse(props.data)
  expandedKeys.value = allKeys
}

// 折叠所有节点
const collapseAll = () => {
  expandedKeys.value = []
}

// 获取当前选中的节点数据
const getCurrentNode = (): ServiceTree | null => {
  if (!currentKey.value || !treeRef.value) return null
  return treeRef.value.getNode(currentKey.value)?.data || null
}

// 获取选中的节点数据
const getSelectedNodes = (): ServiceTree[] => {
  if (!treeRef.value) return []
  return treeRef.value.getCheckedNodes()
}

// 设置当前选中的节点
const setCurrentKey = (key: number) => {
  currentKey.value = key
  treeRef.value?.setCurrentKey(key)
}

// 监听数据变化，更新展开的节点
watch(() => props.data, (newData) => {
  if (newData.length > 0 && expandedKeys.value.length === 0) {
    // 默认展开第一层
    const firstLevelKeys = newData.map(item => item.id).filter(id => id !== undefined)
    expandedKeys.value = firstLevelKeys
  }
})

// 监听过滤文本变化
watch(filterText, (value) => {
  treeRef.value?.filter(value)
})

// 监听默认展开的节点
watch(() => props.defaultExpandedKeys, (keys) => {
  expandedKeys.value = [...keys]
}, { immediate: true })

// 暴露方法给父组件
defineExpose({
  expandAll,
  collapseAll,
  getCurrentNode,
  getSelectedNodes,
  setCurrentKey,
  treeRef
})

onMounted(() => {
  // 如果有默认展开的节点，设置它们
  if (props.defaultExpandedKeys.length > 0) {
    expandedKeys.value = [...props.defaultExpandedKeys]
  }
})
</script>

<template>
  <div class="service-tree-container">
    <!-- 工具栏 -->
    <div class="tree-toolbar" v-if="showActions || $slots.toolbar">
      <slot name="toolbar">
        <div class="toolbar-left">
          <el-input
            v-model="filterText"
            placeholder="搜索节点..."
            clearable
            size="small"
            class="search-input"
          >
            <template #prefix>
              <el-icon><Document /></el-icon>
            </template>
          </el-input>
        </div>
        <div class="toolbar-right">
          <el-button-group size="small">
            <el-button @click="expandAll" title="展开全部">
              <el-icon><Folder /></el-icon>
            </el-button>
            <el-button @click="collapseAll" title="折叠全部">
              <el-icon><Folder /></el-icon>
            </el-button>
          </el-button-group>
        </div>
      </slot>
    </div>

    <!-- Tree组件 -->
    <el-tree
      ref="treeRef"
      v-loading="loading"
      :data="data"
      :props="treeProps"
      :expand-on-click-node="false"
      :default-expanded-keys="expandedKeys"
      :current-node-key="currentKey"
      :draggable="draggable"
      :filter-node-method="filterNode"
      class="service-tree"
      @node-click="handleNodeClick"
      @node-expand="handleNodeExpand"
      @node-collapse="handleNodeCollapse"
      @node-drop="handleNodeDrop"
    >
      <template #default="{ node, data }">
        <div class="tree-node" :class="{ 'is-selected': currentKey === data.id }">
          <!-- 节点信息 -->
          <div class="node-info">
            <el-icon class="node-icon">
              <component :is="getNodeIcon(data.type)" />
            </el-icon>
            <div class="node-content">
              <div class="node-name">{{ node.label }}</div>
              <div class="node-code" v-if="data.code">{{ data.code }}</div>
              <div class="node-description" v-if="data.description">{{ data.description }}</div>
            </div>
          </div>

          <!-- 节点元信息 -->
          <div class="node-meta">
            <el-tag
              :type="getNodeTypeTagType(data.type)"
              size="small"
              class="node-type-tag"
            >
              {{ getNodeTypeLabel(data.type) }}
            </el-tag>

            <!-- 操作按钮 -->
            <div class="node-actions" v-if="showActions" @click.stop>
              <el-dropdown @command="(command) => handleCommand(command, data)">
                <el-button type="text" size="small" class="action-button">
                  <el-icon><Edit /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <!-- 创建子节点（需要 directory:create 权限） -->
                    <el-dropdown-item 
                      command="create" 
                      v-if="data.type === 'package' && hasPermission(data, DirectoryPermissions.create)"
                    >
                      <el-icon><Plus /></el-icon>
                      创建子节点
                    </el-dropdown-item>
                    <!-- 编辑节点（需要 directory:update 权限） -->
                    <el-dropdown-item 
                      command="edit"
                      v-if="hasPermission(data, data.type === 'package' ? DirectoryPermissions.update : 'function:read')"
                    >
                      <el-icon><Edit /></el-icon>
                      编辑节点
                    </el-dropdown-item>
                    <!-- 删除节点（需要 directory:delete 权限） -->
                    <el-dropdown-item 
                      command="delete" 
                      divided
                      v-if="hasPermission(data, data.type === 'package' ? DirectoryPermissions.delete : 'function:read')"
                    >
                      <el-icon><Delete /></el-icon>
                      删除节点
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </div>
        </div>
      </template>
    </el-tree>
  </div>
</template>

<style scoped>
.service-tree-container {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.tree-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}

.toolbar-left {
  flex: 1;
}

.search-input {
  width: 200px;
}

.toolbar-right {
  display: flex;
  gap: 8px;
}

.service-tree {
  flex: 1;
  padding: 8px;
  overflow: auto;
}

.tree-node {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  width: 100%;
  padding: 8px 12px;
  border-radius: 6px;
  transition: all 0.2s ease;  /* 照抄旧版本的过渡时间 */
  cursor: pointer;
  position: relative;
  margin: 1px 0;  /* 照抄旧版本的间距 */
}

.tree-node:hover {
  background-color: var(--el-fill-color-light);
}

.tree-node.is-selected {
  background-color: var(--el-fill-color) !important;
  border: 1px solid var(--el-border-color);
  color: var(--el-text-color-primary);
  font-weight: 500;
}

/* 选中状态的左侧彩色条 - 照抄旧版本 */
.tree-node.is-selected::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  background-color: var(--el-color-primary);
  border-radius: 0 2px 2px 0;
}

.node-info {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.node-icon {
  color: var(--el-color-primary);  /* 照抄旧版本 - 图标用主题色 */
  font-size: 16px;
  margin-top: 2px;
  flex-shrink: 0;
  transition: color 0.2s ease;  /* 添加过渡动画 */
}

.node-content {
  flex: 1;
  min-width: 0;
}

.node-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  line-height: 1.4;
  word-break: break-word;
}

.node-code {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.node-description {
  font-size: 12px;
  color: var(--el-text-color-regular);
  margin-top: 4px;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.node-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.node-type-tag {
  font-size: 10px;
}

.node-actions {
  opacity: 0;
  transition: opacity 0.2s ease;  /* 照抄旧版本的过渡时间 */
  margin-left: 8px;
}

.tree-node:hover .node-actions,
.tree-node.is-selected .node-actions {  /* 照抄旧版本 - 选中状态也显示操作按钮 */
  opacity: 1;
}

.action-button {
  padding: 4px;
  min-height: 24px;
  width: 24px;
  color: var(--el-text-color-secondary);
  transition: all 0.2s ease;  /* 添加过渡动画 */
  border-radius: 2px;  /* 照抄旧版本 */
}

.action-button:hover {
  color: var(--el-color-primary);
  background-color: var(--el-fill-color);  /* 照抄旧版本 */
}

/* Element Plus Tree样式覆盖 */
:deep(.el-tree-node__content) {
  height: auto;
  padding: 0;
  margin-bottom: 2px;
}

:deep(.el-tree-node__content:hover) {
  background-color: transparent;
}

:deep(.el-tree-node__expand-icon) {
  padding: 6px;
  transition: all 0.2s ease;  /* 照抄旧版本的过渡动画 */
  color: var(--el-text-color-secondary);
  border-radius: 2px;  /* 照抄旧版本 */
  cursor: pointer;
}

/* 展开图标悬停效果 - 照抄旧版本 */
:deep(.el-tree-node__expand-icon:hover) {
  background-color: var(--el-fill-color);
}

/* 展开状态的旋转效果 - 照抄旧版本 */
:deep(.el-tree-node.is-expanded > .el-tree-node__content .el-tree-node__expand-icon) {
  transform: rotate(90deg);
}

:deep(.el-tree-node__expand-icon.is-leaf) {
  color: transparent;
}

:deep(.el-tree-node.is-current > .el-tree-node__content) {
  background-color: transparent;
  font-weight: normal;
}

:deep(.el-input__inner) {
  font-size: 12px;
}

:deep(.el-button-group .el-button) {
  padding: 5px 8px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .tree-toolbar {
    flex-direction: column;
    gap: 8px;
    align-items: stretch;
  }

  .search-input {
    width: 100%;
  }

  .node-description {
    display: none;
  }

  .node-actions {
    opacity: 1;
  }
}
</style>