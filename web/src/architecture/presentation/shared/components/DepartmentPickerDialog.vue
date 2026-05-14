<template>
  <el-dialog
    v-model="dialogVisible"
    class="entity-search-dialog-shell"
    :title="title"
    width="600px"
    :close-on-click-modal="false"
    @close="handleClose"
    @opened="handleDialogOpened"
  >
    <div class="department-picker-dialog-search">
      <el-input
        ref="inputRef"
        v-model="searchKeyword"
        :placeholder="placeholder"
        :clearable="true"
        :loading="loading"
        @input="handleSearchInput"
        @clear="handleClearSearch"
        size="large"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <div class="dialog-status">
        <span class="status-chip">
          {{ loading ? '加载中...' : `共 ${totalDepartments} 个部门${searchKeyword ? `，匹配“${searchKeyword}”` : ''}` }}
        </span>
        <span v-if="selectedDepartments.length > 0" class="status-chip status-chip-active">
          已选 {{ selectedDepartments.length }}{{ multiple && maxCount > 0 ? `/${maxCount}` : '' }} 项
        </span>
      </div>
    </div>

    <div v-if="multiple && selectedDepartments.length > 0" class="department-picker-dialog-selected">
      <div class="selected-header">
        <span>已选择 ({{ selectedDepartments.length }}{{ maxCount > 0 ? `/${maxCount}` : '' }})</span>
        <el-button type="text" size="small" @click="handleClearAll">清空</el-button>
      </div>
      <div class="selected-departments">
        <div
          v-for="dept in selectedDepartments"
          :key="dept.full_code_path"
          class="selected-department-item"
        >
          <img src="/组织架构.svg" alt="组织架构" class="selected-department-icon" width="16" height="16" />
          <span class="department-name">{{ dept.name }}</span>
          <el-icon class="remove-icon" @click="handleRemoveDepartment(dept)">
            <Close />
          </el-icon>
        </div>
      </div>
    </div>

    <div class="department-picker-dialog-list" v-loading="loading">
      <div
        v-if="departmentTreeData.length === 0 && !loading"
        class="department-picker-dialog-empty"
      >
        <el-empty description="暂无组织架构" :image-size="80" />
      </div>
      <div
        v-else
        class="department-picker-dialog-tree-shell"
      >
        <el-tree
          ref="treeRef"
          :data="departmentTreeData"
          node-key="full_code_path"
          :props="{ children: 'children', label: 'name' }"
          :default-expand-all="true"
          :expand-on-click-node="false"
          :highlight-current="!multiple"
          :show-checkbox="multiple"
          :check-strictly="multiple"
          :check-on-click-node="multiple"
          :filter-node-method="filterTreeNode"
          empty-text="没有匹配的组织架构"
          @node-click="handleTreeNodeClick"
          @check="handleTreeCheck"
        >
          <template #default="{ data }">
            <div class="department-tree-node">
              <div class="department-tree-main">
                <span class="department-tree-name">{{ data.name }}</span>
                <span v-if="data.is_system_default" class="department-tree-badge">默认</span>
              </div>
              <div class="department-tree-meta">
                <span class="department-tree-code">{{ data.code }}</span>
                <span class="department-tree-path">{{ data.full_name_path || data.full_code_path }}</span>
              </div>
            </div>
          </template>
        </el-tree>
      </div>
    </div>

    <template #footer>
      <div class="department-picker-dialog-footer">
        <el-button @click="handleClose">{{ multiple || !autoConfirmSingle ? '取消' : '关闭' }}</el-button>
        <el-button
          v-if="multiple || !autoConfirmSingle"
          type="primary"
          :disabled="selectedDepartments.length === 0"
          @click="handleConfirm"
        >
          确认
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import type { TreeInstance } from 'element-plus'
import { ElButton, ElDialog, ElEmpty, ElIcon, ElInput } from 'element-plus'
import { Close, Search } from '@element-plus/icons-vue'
import { getDepartmentTree } from '@/architecture/infrastructure/api/department'
import type { Department } from '@/architecture/infrastructure/api/department'
import { Logger } from '@/architecture/shared/logger'

interface Props {
  modelValue: boolean
  title?: string
  placeholder?: string
  initialPaths?: string | null
  multiple?: boolean
  maxCount?: number
  autoConfirmSingle?: boolean
  departmentTree?: Department[]
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'confirm', departments: Department[]): void
}

const props = withDefaults(defineProps<Props>(), {
  title: '选择组织架构',
  placeholder: '搜索部门名称或路径...',
  initialPaths: null,
  multiple: false,
  maxCount: 0,
  autoConfirmSingle: true,
  departmentTree: () => []
})

const emit = defineEmits<Emits>()

const dialogVisible = ref(false)
const searchKeyword = ref('')
const loading = ref(false)
const departmentTreeData = ref<Department[]>([])
const selectedDepartments = ref<Department[]>([])
const inputRef = ref<InstanceType<typeof ElInput> | null>(null)
const treeRef = ref<TreeInstance>()

const multiple = computed(() => props.multiple)
const maxCount = computed(() => props.maxCount)
const autoConfirmSingle = computed(() => props.autoConfirmSingle)

const flattenDepartments = (depts: Department[]): Department[] =>
  depts.flatMap((dept) => [dept, ...(dept.children ? flattenDepartments(dept.children) : [])])

const totalDepartments = computed(() => flattenDepartments(departmentTreeData.value).length)

const normalizePaths = (value: string | null): string[] => {
  if (!value) {
    return []
  }
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

const findDepartmentByPath = (depts: Department[], path: string): Department | null => {
  for (const dept of depts) {
    if (dept.full_code_path === path) {
      return dept
    }
    if (dept.children && dept.children.length > 0) {
      const found = findDepartmentByPath(dept.children, path)
      if (found) {
        return found
      }
    }
  }
  return null
}

const loadDepartmentTreeData = async () => {
  if (props.departmentTree && props.departmentTree.length > 0) {
    departmentTreeData.value = props.departmentTree
    return
  }

  loading.value = true
  try {
    const response = await getDepartmentTree()
    departmentTreeData.value = response.departments || []
  } catch (error) {
    Logger.error('DepartmentPickerDialog', '加载部门树失败', error)
    departmentTreeData.value = []
  } finally {
    loading.value = false
  }
}

const initializeSelectedDepartments = async () => {
  const paths = normalizePaths(props.initialPaths)
  if (paths.length === 0) {
    selectedDepartments.value = []
    return
  }

  if (departmentTreeData.value.length === 0) {
    await loadDepartmentTreeData()
  }

  const loadedDepartments = paths
    .map((path) => findDepartmentByPath(departmentTreeData.value, path))
    .filter((dept): dept is Department => !!dept)

  selectedDepartments.value = multiple.value ? loadedDepartments : loadedDepartments.slice(0, 1)
}

const applyTreeState = async () => {
  await nextTick()

  if (!treeRef.value) {
    return
  }

  treeRef.value.filter(searchKeyword.value.trim())

  if (multiple.value) {
    treeRef.value.setCheckedKeys(selectedDepartments.value.map((dept) => dept.full_code_path), false)
    return
  }

  treeRef.value.setCurrentKey(selectedDepartments.value[0]?.full_code_path ?? undefined)
}

watch(
  () => props.modelValue,
  async (newValue) => {
    dialogVisible.value = newValue
    if (!newValue) {
      resetSearchState()
      return
    }
    await loadDepartmentTreeData()
    await initializeSelectedDepartments()
    resetSearchState()
    await applyTreeState()
  }
)

watch(dialogVisible, (newValue) => {
  emit('update:modelValue', newValue)
})

watch(
  () => props.departmentTree,
  async (newTree) => {
    if (newTree && newTree.length > 0) {
      departmentTreeData.value = newTree
      await applyTreeState()
    }
  },
  { deep: true }
)

const handleDialogOpened = async () => {
  await nextTick()
  await nextTick()
  inputRef.value?.focus()
  await applyTreeState()
}

const isDepartmentSelected = (dept: Department): boolean => {
  return selectedDepartments.value.some((item) => item.full_code_path === dept.full_code_path)
}

const handleTreeNodeClick = (dept: Department) => {
  if (multiple.value) {
    return
  }

  selectedDepartments.value = [dept]
  if (autoConfirmSingle.value) {
    emit('confirm', [dept])
    handleClose()
  }
}

const handleRemoveDepartment = (dept: Department) => {
  selectedDepartments.value = selectedDepartments.value.filter((item) => item.full_code_path !== dept.full_code_path)
}

const handleClearAll = () => {
  selectedDepartments.value = []
  if (multiple.value) {
    treeRef.value?.setCheckedKeys([], false)
  } else {
    treeRef.value?.setCurrentKey(undefined)
  }
}

const handleConfirm = () => {
  emit('confirm', [...selectedDepartments.value])
  handleClose()
}

const filterTreeNode = (keyword: string, data: Department): boolean => {
  const normalizedKeyword = keyword.trim().toLowerCase()
  if (!normalizedKeyword) {
    return true
  }

  const selfMatched = [
    data.name,
    data.code,
    data.full_code_path,
    data.full_name_path
  ]
    .filter(Boolean)
    .some((value) => String(value).toLowerCase().includes(normalizedKeyword))

  if (selfMatched) {
    return true
  }

  return (data.children || []).some((child) => filterTreeNode(keyword, child))
}

const handleTreeCheck = () => {
  if (!multiple.value || !treeRef.value) {
    return
  }

  const checkedDepartments = treeRef.value.getCheckedNodes(false, false) as Department[]

  if (maxCount.value > 0 && checkedDepartments.length > maxCount.value) {
    treeRef.value.setCheckedKeys(selectedDepartments.value.map((dept) => dept.full_code_path), false)
    return
  }

  selectedDepartments.value = checkedDepartments
}

const handleSearchInput = (value: string) => {
  searchKeyword.value = value
}

const handleClearSearch = () => {
  resetSearchState()
}

const resetSearchState = () => {
  searchKeyword.value = ''
  treeRef.value?.filter('')
}

const handleClose = () => {
  dialogVisible.value = false
  resetSearchState()
}

watch(searchKeyword, async () => {
  await nextTick()
  treeRef.value?.filter(searchKeyword.value.trim())
})
</script>

<style scoped>
:deep(.entity-search-dialog-shell) {
  border-radius: 18px;
  overflow: hidden;
}

:deep(.entity-search-dialog-shell .el-dialog__header) {
  padding: 18px 22px 0;
}

:deep(.entity-search-dialog-shell .el-dialog__body) {
  padding: 18px 22px 12px;
}

:deep(.entity-search-dialog-shell .el-dialog__footer) {
  padding: 0 22px 20px;
}

.department-picker-dialog-search {
  margin-bottom: 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dialog-status {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.status-chip {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
}

.status-chip-active {
  background: rgba(24, 144, 255, 0.12);
  color: var(--el-color-primary);
}

.department-picker-dialog-selected {
  margin-bottom: 18px;
  padding: 14px;
  background:
    linear-gradient(180deg, rgba(24, 144, 255, 0.06), rgba(24, 144, 255, 0)),
    var(--el-fill-color-lighter);
  border-radius: 14px;
  border: 1px solid rgba(24, 144, 255, 0.12);
}

.selected-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.selected-departments {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.selected-department-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 9px;
  background-color: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 999px;
}

.selected-department-icon {
  width: 16px;
  height: 16px;
  min-width: 16px;
  min-height: 16px;
  max-width: 16px;
  max-height: 16px;
  flex-shrink: 0;
  object-fit: contain;
  display: block;
}

.department-picker-dialog-list {
  min-height: 300px;
  max-height: 400px;
}

.department-picker-dialog-tree-shell {
  height: 100%;
  min-height: 300px;
  max-height: 400px;
  overflow: auto;
  padding-right: 4px;
}

.department-picker-dialog-empty {
  padding: 40px 0;
}

.department-tree-node {
  width: 100%;
  display: flex;
  min-width: 0;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  padding: 6px 0;
}

.department-tree-main,
.department-tree-meta {
  width: 100%;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.department-tree-main {
  justify-content: flex-start;
}

.department-tree-meta {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.department-tree-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.department-tree-badge {
  display: inline-flex;
  align-items: center;
  padding: 1px 6px;
  border-radius: 999px;
  background: rgba(24, 144, 255, 0.1);
  color: var(--el-color-primary);
  font-size: 11px;
}

.department-tree-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  flex-shrink: 0;
}

.department-tree-path {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.remove-icon {
  cursor: pointer;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  transition: color 0.2s;
}

.remove-icon:hover {
  color: var(--el-color-danger);
}

:deep(.department-picker-dialog-tree-shell .el-tree) {
  background: transparent;
}

:deep(.department-picker-dialog-tree-shell .el-tree-node__content) {
  height: auto;
  min-height: 52px;
  padding: 0 8px;
  border-radius: 10px;
  transition: background-color 0.18s ease;
}

:deep(.department-picker-dialog-tree-shell .el-tree-node__content:hover) {
  background: var(--el-fill-color-light);
}

:deep(.department-picker-dialog-tree-shell .el-tree-node.is-current > .el-tree-node__content) {
  background: rgba(24, 144, 255, 0.08);
}

:deep(.department-picker-dialog-tree-shell .el-checkbox) {
  margin-right: 8px;
}

.department-picker-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
