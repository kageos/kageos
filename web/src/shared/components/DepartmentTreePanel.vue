<!--
  DepartmentTreePanel - 组织架构树形面板组件

  需求：
  - 使用树形结构展示组织架构
  - 支持点击节点查看部门用户
  - 支持右键菜单操作（编辑、删除、查看用户等）
-->
<template>
  <div class="department-tree-panel" v-loading="loading">
    <div class="tree-header">
      <h3>组织架构</h3>

      <div class="header-actions">
        <el-button type="primary" plain size="small" @click="$emit('create-department')">
          <el-icon><Plus /></el-icon>
          新增
        </el-button>
        <el-button text size="small" @click="$emit('refresh')">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <div v-if="treeData.length > 0" class="tree-search">
      <el-input
        v-model="searchKeyword"
        clearable
        :prefix-icon="Search"
        placeholder="搜索部门名称、编码或路径"
      />
    </div>

    <div class="tree-content">
      <el-tree
        v-if="treeData.length > 0"
        ref="treeRef"
        :data="treeData"
        :props="{ children: 'children', label: 'name' }"
        node-key="id"
        :current-node-key="currentNodeId ?? undefined"
        :default-expand-all="true"
        :expand-on-click-node="false"
        :highlight-current="true"
        :filter-node-method="filterTreeNode"
        empty-text="没有匹配的部门"
        @node-click="handleNodeClick"
      >
        <template #default="{ node, data }">
          <div class="tree-node">
            <div class="node-texts">
              <span class="node-label-row">
                <span class="node-label">{{ node.label }}</span>
                <span v-if="data.is_system_default" class="node-status">默认</span>
              </span>
              <span class="node-caption">{{ data.code }}</span>
            </div>

            <el-dropdown
              trigger="click"
              :teleported="true"
              popper-class="department-tree-dropdown-popper"
              @click.stop
              class="node-more-actions"
              @command="(command: string) => handleNodeAction(command, data)"
            >
              <el-icon class="more-icon" @click.stop>
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
          </div>
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
import { nextTick, ref, watch } from 'vue'
import type { TreeInstance } from 'element-plus'
import { Delete, Edit, MoreFilled, Plus, Refresh, Search, User } from '@element-plus/icons-vue'
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

const treeRef = ref<TreeInstance>()
const searchKeyword = ref('')

const applyTreeState = async () => {
  await nextTick()
  treeRef.value?.filter(searchKeyword.value.trim())
  treeRef.value?.setCurrentKey(props.currentNodeId ?? undefined)
}

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

const filterTreeNode = (keyword: string, data: Department) => {
  const normalizedKeyword = keyword.trim().toLowerCase()
  if (!normalizedKeyword) return true

  return [
    data.name,
    data.code,
    data.full_name_path,
    data.full_code_path
  ]
    .filter(Boolean)
    .some((value) => String(value).toLowerCase().includes(normalizedKeyword))
}

watch(searchKeyword, () => {
  treeRef.value?.filter(searchKeyword.value.trim())
})

watch(() => props.currentNodeId, applyTreeState, { immediate: true })
watch(() => props.treeData, applyTreeState)

defineExpose({
  treeRef
})
</script>

<style scoped lang="scss">
.department-tree-panel {
  --dept-tree-accent: var(--color-primary);
  --dept-tree-ink: var(--text-primary);
  --dept-tree-muted: var(--text-secondary);
  --dept-tree-line: color-mix(in srgb, var(--border-base) 82%, var(--color-primary) 18%);
  --dept-tree-surface: var(--bg-primary);
  --dept-tree-accent-soft: color-mix(in srgb, var(--color-primary) 10%, transparent);
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: transparent;
}

.tree-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.tree-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--dept-tree-ink);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.tree-search {
  :deep(.el-input__wrapper) {
    border-radius: 14px;
    box-shadow: none;
    background: var(--bg-primary);
    border: 1px solid var(--border-base);
  }
}

.tree-content {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 4px 2px 0;
}

.empty-state {
  min-height: 280px;
  display: flex;
  align-items: center;
  justify-content: center;
}

:deep(.el-tree) {
  background: transparent;
}

:deep(.el-tree-node__content) {
  height: 44px;
  margin-bottom: 4px;
  padding: 0 8px;
  border-radius: 10px;
  background: var(--dept-tree-surface);
  transition: background-color 0.18s ease;

  &:hover {
    background: color-mix(in srgb, var(--color-primary) 4%, var(--bg-primary) 96%);

    .node-more-actions {
      opacity: 1;
    }
  }
}

:deep(.el-tree-node.is-current > .el-tree-node__content) {
  background: color-mix(in srgb, var(--color-primary) 8%, var(--bg-primary) 92%) !important;
  box-shadow: inset 2px 0 0 var(--color-primary);

  .node-more-actions {
    opacity: 1;
  }

  .node-label {
    color: var(--dept-tree-ink);
  }
}

.tree-node {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.node-texts {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.node-label-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.node-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 500;
  color: var(--dept-tree-ink);
}

.node-status {
  display: inline-flex;
  align-items: center;
  padding: 1px 6px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-warning) 14%, var(--bg-primary) 86%);
  color: var(--color-warning);
  font-size: 11px;
  line-height: 1.4;
  flex-shrink: 0;
}

.node-caption {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--dept-tree-muted);
}

.node-more-actions {
  opacity: 0;
  transition: opacity 0.18s ease;
}

.more-icon {
  width: 26px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  cursor: pointer;
  color: var(--dept-tree-muted);
  transition: background-color 0.18s ease, color 0.18s ease;

  &:hover {
    background: var(--dept-tree-accent-soft);
    color: var(--dept-tree-accent);
  }
}

.disabled-hint {
  color: var(--text-disabled);
}

:deep(.el-dropdown-menu),
:global(.department-tree-dropdown-popper .el-dropdown-menu) {
  min-width: 176px;
}

:deep(.el-dropdown-menu__item),
:global(.department-tree-dropdown-popper .el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 14px;
  white-space: nowrap;
}

@media (max-width: 920px) {
  .tree-header {
    flex-direction: column;
  }

  .header-actions {
    width: 100%;
    justify-content: flex-start;
  }

  :deep(.el-tree-node__content) {
    height: 52px;
  }
}
</style>
