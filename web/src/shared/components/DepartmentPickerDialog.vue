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
          {{ loading ? '加载中...' : (searchKeyword ? `${filteredDepartments.length} 个结果` : '输入关键词开始搜索') }}
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
          <img src="/组织架构.svg" alt="组织架构" class="department-icon" />
          <span class="department-name">{{ dept.name }}</span>
          <el-icon class="remove-icon" @click="handleRemoveDepartment(dept)">
            <Close />
          </el-icon>
        </div>
      </div>
    </div>

    <div class="department-picker-dialog-list" v-loading="loading">
      <div
        v-if="filteredDepartments.length === 0 && !loading && searchKeyword"
        class="department-picker-dialog-empty"
      >
        <el-empty description="未找到组织架构" :image-size="80" />
      </div>
      <div
        v-else-if="filteredDepartments.length === 0 && !loading && !searchKeyword"
        class="department-picker-dialog-empty"
      >
        <el-empty description="请输入关键词搜索组织架构" :image-size="80" />
      </div>
      <div
        v-else
        class="department-picker-dialog-items"
      >
        <div
          v-for="dept in filteredDepartments"
          :key="dept.full_code_path"
          class="department-picker-dialog-item"
          :class="{ 'is-selected': isDepartmentSelected(dept) }"
          @click="handlePickDepartment(dept)"
        >
          <el-checkbox
            v-if="multiple"
            :model-value="isDepartmentSelected(dept)"
            @change="handleToggleDepartment(dept)"
            @click.stop
          />
          <img src="/组织架构.svg" alt="组织架构" class="department-icon" />
          <div class="department-info">
            <div class="department-name">{{ dept.name }}</div>
            <div class="department-meta">
              <span
                class="department-path clickable-path"
                @click.stop="handlePathClick(dept.full_code_path)"
                :title="`点击跳转到: ${dept.full_code_path}`"
              >
                {{ dept.full_code_path }}
              </span>
              <span v-if="dept.full_name_path && dept.full_name_path !== dept.name" class="department-full-name">
                {{ dept.full_name_path }}
              </span>
            </div>
          </div>
          <el-icon
            v-if="!multiple && isDepartmentSelected(dept)"
            class="selected-icon"
          >
            <Check />
          </el-icon>
        </div>
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
import { useRouter } from 'vue-router'
import { ElButton, ElCheckbox, ElDialog, ElEmpty, ElIcon, ElInput } from 'element-plus'
import { Check, Close, Search } from '@element-plus/icons-vue'
import { getDepartmentTree } from '@/api/department'
import type { Department } from '@/api/department'
import { Logger } from '@/core/utils/logger'

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
const router = useRouter()

const dialogVisible = ref(false)
const searchKeyword = ref('')
const loading = ref(false)
const departmentTreeData = ref<Department[]>([])
const selectedDepartments = ref<Department[]>([])
const inputRef = ref<InstanceType<typeof ElInput> | null>(null)

const multiple = computed(() => props.multiple)
const maxCount = computed(() => props.maxCount)
const autoConfirmSingle = computed(() => props.autoConfirmSingle)

const flattenDepartments = (depts: Department[]): Department[] => {
  const result: Department[] = []
  const traverse = (list: Department[]) => {
    for (const dept of list) {
      result.push(dept)
      if (dept.children && dept.children.length > 0) {
        traverse(dept.children)
      }
    }
  }
  traverse(depts)
  return result
}

const filteredDepartments = computed(() => {
  const allDepartments = flattenDepartments(departmentTreeData.value)
  if (!searchKeyword.value || searchKeyword.value.trim().length === 0) {
    return allDepartments
  }

  const keyword = searchKeyword.value.trim().toLowerCase()
  return allDepartments.filter((dept) => {
    return (
      dept.name.toLowerCase().includes(keyword) ||
      dept.full_code_path.toLowerCase().includes(keyword) ||
      (dept.full_name_path && dept.full_name_path.toLowerCase().includes(keyword)) ||
      (dept.code && dept.code.toLowerCase().includes(keyword))
    )
  })
})

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
  }
)

watch(dialogVisible, (newValue) => {
  emit('update:modelValue', newValue)
})

const handleDialogOpened = async () => {
  await nextTick()
  await nextTick()
  inputRef.value?.focus()
}

const isDepartmentSelected = (dept: Department): boolean => {
  return selectedDepartments.value.some((item) => item.full_code_path === dept.full_code_path)
}

const handleToggleDepartment = (dept: Department) => {
  if (!multiple.value) {
    selectedDepartments.value = [dept]
    return
  }

  if (isDepartmentSelected(dept)) {
    selectedDepartments.value = selectedDepartments.value.filter((item) => item.full_code_path !== dept.full_code_path)
    return
  }

  if (maxCount.value > 0 && selectedDepartments.value.length >= maxCount.value) {
    return
  }

  selectedDepartments.value = [...selectedDepartments.value, dept]
}

const handlePickDepartment = (dept: Department) => {
  if (multiple.value) {
    handleToggleDepartment(dept)
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
}

const handleConfirm = () => {
  emit('confirm', [...selectedDepartments.value])
  handleClose()
}

const handleSearchInput = (value: string) => {
  searchKeyword.value = value
}

const handleClearSearch = () => {
  resetSearchState()
}

const resetSearchState = () => {
  searchKeyword.value = ''
}

const handleClose = () => {
  dialogVisible.value = false
  resetSearchState()
}

const handlePathClick = (fullCodePath: string) => {
  if (!fullCodePath) {
    return
  }
  const targetPath = `/workspace${fullCodePath.startsWith('/') ? fullCodePath : `/${fullCodePath}`}`
  router.push(targetPath)
  handleClose()
}
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

.department-picker-dialog-list {
  min-height: 300px;
  max-height: 400px;
  overflow-y: auto;
  padding-right: 2px;
}

.department-picker-dialog-empty {
  padding: 40px 0;
}

.department-picker-dialog-items {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.department-picker-dialog-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  background:
    linear-gradient(180deg, rgba(24, 144, 255, 0.02), rgba(24, 144, 255, 0)),
    var(--el-bg-color);
  cursor: pointer;
  transition: all 0.2s;
}

.department-picker-dialog-item:hover {
  border-color: rgba(24, 144, 255, 0.24);
  background-color: var(--el-fill-color-light);
  transform: translateY(-1px);
}

.department-picker-dialog-item.is-selected {
  border-color: rgba(24, 144, 255, 0.34);
  background: linear-gradient(135deg, rgba(24, 144, 255, 0.12), rgba(24, 144, 255, 0.04));
  box-shadow: 0 8px 20px rgba(24, 144, 255, 0.08);
}

.department-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.department-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.department-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.department-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.department-path,
.department-full-name {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.4;
}

.clickable-path {
  cursor: pointer;
}

.clickable-path:hover {
  color: var(--el-color-primary);
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

.selected-icon {
  flex-shrink: 0;
  color: var(--el-color-primary);
  font-size: 20px;
}

.department-picker-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
