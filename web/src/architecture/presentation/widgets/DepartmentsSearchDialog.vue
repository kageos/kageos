<!--
  DepartmentsSearchDialog - 多组织架构搜索弹窗组件
  功能：
  - 弹窗式组织架构搜索和选择
  - 支持多选模式
  - 搜索、选择、确认
-->
<template>
  <el-dialog
    v-model="dialogVisible"
    :title="title"
    width="600px"
    :close-on-click-modal="false"
    @close="handleClose"
    @opened="handleDialogOpened"
  >
    <!-- 搜索框 -->
    <div class="departments-search-dialog-search">
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
    </div>

    <!-- 已选组织架构列表 -->
    <div v-if="selectedDepartments.length > 0" class="departments-search-dialog-selected">
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
          <span class="department-path">{{ dept.full_code_path }}</span>
          <el-icon class="remove-icon" @click="handleRemoveDepartment(dept)">
            <Close />
          </el-icon>
        </div>
      </div>
    </div>

    <!-- 组织架构列表 -->
    <div class="departments-search-dialog-list" v-loading="loading">
      <div
        v-if="filteredDepartments.length === 0 && !loading && searchKeyword"
        class="departments-search-dialog-empty"
      >
        <el-empty description="未找到组织架构" :image-size="80" />
      </div>
      <div
        v-else-if="filteredDepartments.length === 0 && !loading && !searchKeyword"
        class="departments-search-dialog-empty"
      >
        <el-empty description="请输入关键词搜索组织架构" :image-size="80" />
      </div>
      <div
        v-else
        class="departments-search-dialog-items"
      >
        <div
          v-for="dept in filteredDepartments"
          :key="dept.full_code_path"
          class="departments-search-dialog-item"
          :class="{ 'is-selected': isDepartmentSelected(dept) }"
          @click="handleToggleDepartment(dept)"
        >
          <el-checkbox
            :model-value="isDepartmentSelected(dept)"
            @change="handleToggleDepartment(dept)"
            @click.stop
          />
          <img src="/组织架构.svg" alt="组织架构" class="department-icon" />
          <div class="department-info">
            <div class="department-name">{{ dept.name }}</div>
            <div class="department-meta">
              <span class="department-path">
                {{ dept.full_code_path }}
              </span>
              <span v-if="dept.full_name_path && dept.full_name_path !== dept.name" class="department-full-name">
                {{ dept.full_name_path }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="departments-search-dialog-footer">
        <el-button @click="handleClose">取消</el-button>
        <el-button type="primary" @click="handleConfirm">确认</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { ElDialog, ElInput, ElButton, ElIcon, ElEmpty, ElCheckbox } from 'element-plus'
import { Search, Close } from '@element-plus/icons-vue'
import { getDepartmentTree } from '@/api/department'
import type { Department } from '@/api/department'
import { Logger } from '@/core/utils/logger'

interface Props {
  modelValue: boolean
  title?: string
  placeholder?: string
  initialPaths?: string | null // 逗号分隔的 full_code_path 列表
  maxCount?: number // 最大选择数量，0表示不限制
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'confirm', departments: Department[]): void
}

const props = withDefaults(defineProps<Props>(), {
  title: '选择组织架构',
  placeholder: '搜索部门名称或路径...',
  initialPaths: null,
  maxCount: 0
})

const emit = defineEmits<Emits>()

const dialogVisible = ref(false)
const searchKeyword = ref('')
const loading = ref(false)
const departmentTree = ref<Department[]>([])
const selectedDepartments = ref<Department[]>([])
const inputRef = ref<InstanceType<typeof ElInput> | null>(null)

// 扁平化部门列表（用于搜索和显示）
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

// 过滤后的部门列表
const filteredDepartments = computed(() => {
  const allDepartments = flattenDepartments(departmentTree.value)
  
  if (!searchKeyword.value || searchKeyword.value.trim().length === 0) {
    return allDepartments
  }
  
  const keyword = searchKeyword.value.trim().toLowerCase()
  return allDepartments.filter(dept => {
    return (
      dept.name.toLowerCase().includes(keyword) ||
      dept.full_code_path.toLowerCase().includes(keyword) ||
      (dept.full_name_path && dept.full_name_path.toLowerCase().includes(keyword)) ||
      (dept.code && dept.code.toLowerCase().includes(keyword))
    )
  })
})

// 加载部门树
const loadDepartmentTree = async () => {
  loading.value = true
  try {
    const response = await getDepartmentTree()
    departmentTree.value = response.departments || []
  } catch (error) {
    Logger.error('DepartmentsSearchDialog', '加载部门树失败', error)
    departmentTree.value = []
  } finally {
    loading.value = false
  }
}

// 根据路径查找部门
const findDepartmentByPath = (depts: Department[], path: string): Department | null => {
  for (const dept of depts) {
    if (dept.full_code_path === path) {
      return dept
    }
    if (dept.children && dept.children.length > 0) {
      const found = findDepartmentByPath(dept.children, path)
      if (found) return found
    }
  }
  return null
}

// 初始化已选部门
const initializeSelectedDepartments = async () => {
  if (!props.initialPaths) {
    selectedDepartments.value = []
    return
  }
  
  const paths = props.initialPaths.split(',').map(p => p.trim()).filter(p => p)
  if (paths.length === 0) {
    selectedDepartments.value = []
    return
  }
  
  // 确保部门树已加载
  if (departmentTree.value.length === 0) {
    await loadDepartmentTree()
  }
  
  const selected: Department[] = []
  for (const path of paths) {
    const dept = findDepartmentByPath(departmentTree.value, path)
    if (dept) {
      selected.push(dept)
    }
  }
  selectedDepartments.value = selected
}

// 监听 modelValue 变化，控制弹窗显示
watch(() => props.modelValue, async (newValue) => {
  dialogVisible.value = newValue
  if (newValue) {
    // 加载部门树
    await loadDepartmentTree()
    // 初始化已选部门
    await initializeSelectedDepartments()
    // 聚焦搜索框
    await nextTick()
    inputRef.value?.focus()
  }
})

// 监听 dialogVisible 变化，同步到 modelValue
watch(dialogVisible, (newValue) => {
  emit('update:modelValue', newValue)
})

// 处理搜索输入
const handleSearchInput = (value: string) => {
  searchKeyword.value = value
  // 不需要防抖，因为过滤是实时的
}

// 处理清空搜索
const handleClearSearch = () => {
  searchKeyword.value = ''
}

// 处理弹窗打开
const handleDialogOpened = async () => {
  // 聚焦搜索框
  await nextTick()
  inputRef.value?.focus()
}

// 判断组织架构是否已选中
const isDepartmentSelected = (dept: Department): boolean => {
  return selectedDepartments.value.some(d => d.full_code_path === dept.full_code_path)
}

// 切换组织架构选择状态
const handleToggleDepartment = (dept: Department) => {
  if (isDepartmentSelected(dept)) {
    // 取消选择
    selectedDepartments.value = selectedDepartments.value.filter(d => d.full_code_path !== dept.full_code_path)
  } else {
    // 检查是否超过最大数量
    if (props.maxCount > 0 && selectedDepartments.value.length >= props.maxCount) {
      return
    }
    // 添加选择
    selectedDepartments.value.push(dept)
  }
}

// 移除已选组织架构
const handleRemoveDepartment = (dept: Department) => {
  selectedDepartments.value = selectedDepartments.value.filter(d => d.full_code_path !== dept.full_code_path)
}

// 清空所有已选组织架构
const handleClearAll = () => {
  selectedDepartments.value = []
}

// 确认选择
const handleConfirm = () => {
  emit('confirm', selectedDepartments.value)
  handleClose()
}

// 关闭弹窗
const handleClose = () => {
  dialogVisible.value = false
  searchKeyword.value = ''
}

// ⚠️ 注意：部门路径（full_code_path）不是服务目录路径，不能跳转到服务目录
// 如果需要跳转功能，需要先根据部门路径查找对应的服务目录节点
// 暂时移除跳转功能，避免错误
</script>

<style lang="scss" scoped>
.departments-search-dialog-search {
  margin-bottom: 16px;
}

.departments-search-dialog-selected {
  margin-bottom: 16px;
  padding: 12px;
  background-color: var(--el-fill-color-lighter);
  border-radius: 4px;
}

.selected-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.selected-departments {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.selected-department-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  background-color: var(--el-bg-color);
  border-radius: 4px;
  border: 1px solid var(--el-border-color);
}

.department-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.department-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
  font-weight: 500;
}

.department-path {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  flex: 1;
}

.remove-icon {
  cursor: pointer;
  color: var(--el-text-color-secondary);
  transition: color 0.2s;
  
  &:hover {
    color: var(--el-color-danger);
  }
}

.departments-search-dialog-list {
  max-height: 400px;
  overflow-y: auto;
}

.departments-search-dialog-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.departments-search-dialog-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
  
  &:hover {
    border-color: var(--el-color-primary);
    background-color: var(--el-fill-color-light);
  }
  
  &.is-selected {
    border-color: var(--el-color-primary);
    background-color: var(--el-color-primary-light-9);
  }
}

.department-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.department-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}


.departments-search-dialog-empty {
  padding: 40px 0;
  text-align: center;
}

.departments-search-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
