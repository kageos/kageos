<!--
  DepartmentSearchWidget - 部门搜索专用组件（用于搜索表单）
  功能：
  - 专门为搜索场景设计，直接处理原始值格式（string | string[] | null）
  - 使用弹窗搜索，体验更好，可以展示更多信息
  - 支持单选和多选（根据 search 类型自动判断）
-->
<template>
  <div class="department-search-widget">
    <!-- 选中后的显示（多选模式） -->
    <div
      v-if="supportsMultiple && selectedDepartments.length > 0"
      class="department-search-display"
      @click="handleOpenDialog()"
    >
      <div class="selected-departments-list">
        <div
          v-for="dept in selectedDepartments"
          :key="dept.full_code_path"
          class="selected-department-tag"
        >
          <img src="/组织架构.svg" alt="组织架构" class="department-icon-small" />
          <span class="department-display-text">
            {{ dept.name }}
          </span>
          <el-icon class="remove-icon" @click.stop="handleRemoveDepartment(dept)">
            <Close />
          </el-icon>
        </div>
      </div>
      <el-icon class="edit-icon">
        <Edit />
      </el-icon>
    </div>
    
    <!-- 选中后的显示（单选模式） -->
    <div
      v-else-if="!supportsMultiple && selectedDepartment"
      class="department-search-display"
      @click="handleOpenDialog()"
    >
      <img src="/组织架构.svg" alt="组织架构" class="department-icon-small" />
      <span class="department-display-text">
        {{ selectedDepartment.name }}
      </span>
      <el-icon class="edit-icon">
        <Edit />
      </el-icon>
    </div>
    <!-- 未选中时显示按钮 -->
    <el-button
      v-else
      class="search-trigger-button"
      @click="handleOpenDialog()"
    >
      <el-icon><OfficeBuilding /></el-icon>
      <span class="search-trigger-text">{{ field.desc || `请选择${field.name}` }}</span>
    </el-button>
    
    <!-- 多组织架构搜索弹窗（支持 IN 查询时使用） -->
    <DepartmentsSearchDialog
      v-if="supportsMultiple"
      v-model="dialogVisible"
      :title="`选择${field.name || '组织架构'}`"
      :placeholder="field.desc || '搜索部门名称或路径...'"
      :initial-paths="normalizedModelValue"
      @confirm="handleDepartmentsSelected"
    />
    
    <!-- 单组织架构搜索弹窗（EQ 查询时使用） -->
    <DepartmentSelectorDialog
      v-else
      v-model="dialogVisible"
      :selected-department="selectedDepartment"
      @select="handleDepartmentSelected"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElButton, ElIcon } from 'element-plus'
import { OfficeBuilding, Edit, Close } from '@element-plus/icons-vue'
import DepartmentSelectorDialog from '@/shared/components/DepartmentSelectorDialog.vue'
import DepartmentsSearchDialog from '@/shared/components/DepartmentsSearchDialog.vue'
import { useDepartmentInfoStore } from '@/stores/departmentInfo'
import type { Department } from '@/api/department'
import type { FieldConfig } from '@/core/types/field'
import { Logger } from '@/core/utils/logger'
import { SearchType, hasSearchType } from '@/core/constants/search'

interface Props {
  field: FieldConfig
  modelValue: string | string[] | null  // 搜索表单的原始值格式（逗号分隔的字符串或数组）
  searchType?: string  // 搜索类型，用于判断单选还是多选
}

interface Emits {
  (e: 'update:modelValue', value: string | string[] | null): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const departmentInfoStore = useDepartmentInfoStore()
const dialogVisible = ref(false)
const selectedDepartments = ref<Department[]>([])
const selectedDepartment = ref<Department | null>(null)

// 是否支持多选（根据 search 类型判断）
const supportsMultiple = computed(() => {
  const searchType = props.searchType || ''
  return hasSearchType(searchType, SearchType.IN)
})

// 规范化 modelValue 为字符串格式（用于传递给对话框）
const normalizedModelValue = computed(() => {
  if (!props.modelValue) return null
  if (Array.isArray(props.modelValue)) {
    return props.modelValue.map(v => String(v).trim()).filter(v => v).join(',') || null
  }
  return String(props.modelValue).trim() || null
})

// 处理打开弹窗
function handleOpenDialog(): void {
  dialogVisible.value = true
}

// 处理部门选择（多选）
function handleDepartmentsSelected(departments: Department[]): void {
  // 直接返回原始值格式（逗号分隔的字符串）
  const paths = departments.map(d => d.full_code_path).join(',')
  selectedDepartments.value = departments
  emit('update:modelValue', paths || null)
}

// 处理部门选择（单选）
function handleDepartmentSelected(department: Department | null): void {
  selectedDepartment.value = department
  emit('update:modelValue', department ? department.full_code_path : null)
}

// 移除单个部门
function handleRemoveDepartment(dept: Department): void {
  const currentPaths = getPathsFromValue(props.modelValue)
  const newPaths = currentPaths.filter(p => p !== dept.full_code_path)
  selectedDepartments.value = selectedDepartments.value.filter(d => d.full_code_path !== dept.full_code_path)
  emit('update:modelValue', newPaths.length > 0 ? newPaths.join(',') : null)
}

// 从值中提取路径列表
function getPathsFromValue(value: string | string[] | null): string[] {
  if (!value) return []
  if (Array.isArray(value)) {
    return value.map(v => String(v).trim()).filter(v => v)
  }
  return String(value).split(',').map(p => p.trim()).filter(p => p)
}

// 加载已选部门信息
async function loadSelectedDepartments(): Promise<void> {
  if (supportsMultiple.value) {
    // 多选模式
    const paths = getPathsFromValue(props.modelValue)
    if (paths.length === 0) {
      selectedDepartments.value = []
      return
    }
    
    try {
      // 从 store 批量获取部门信息
      const departments = await departmentInfoStore.batchGetDepartmentInfo(paths)
      selectedDepartments.value = departments
    } catch (error) {
      Logger.error('DepartmentSearchWidget', '加载已选部门信息失败', { error })
      selectedDepartments.value = []
    }
  } else {
    // 单选模式
    const path = props.modelValue ? String(props.modelValue).trim() : null
    if (!path) {
      selectedDepartment.value = null
      return
    }
    
    try {
      const department = await departmentInfoStore.getDepartmentInfo(path)
      selectedDepartment.value = department || null
    } catch (error) {
      Logger.error('DepartmentSearchWidget', '加载部门信息失败', { path, error })
      selectedDepartment.value = null
    }
  }
}

// 监听值变化，加载部门信息
watch(() => [props.modelValue, supportsMultiple], () => {
  loadSelectedDepartments()
}, { immediate: true })
</script>

<style scoped>
.department-search-widget {
  width: 100%;
}

.search-trigger-button {
  width: 100%;
  min-height: 32px;
  justify-content: flex-start;
  padding: 0 12px;
}

.search-trigger-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.department-search-display {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 10px;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  background-color: var(--el-bg-color);
  cursor: pointer;
  transition: all 0.2s;
  min-height: 32px;
}

.department-search-display .department-icon-small {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.department-search-display .department-display-text {
  flex: 1;
  font-size: 13px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.department-search-display:hover {
  border-color: var(--el-color-primary);
  background-color: var(--el-fill-color-light);
}

.selected-departments-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  flex: 1;
  align-items: center;
}

.selected-department-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  background-color: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  font-size: 12px;
}

.department-icon-small {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.department-display-text {
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

.remove-icon {
  cursor: pointer;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  margin-left: 4px;
  transition: color 0.2s;
}

.remove-icon:hover {
  color: var(--el-color-primary);
}

.edit-icon {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
  font-size: 16px;
  transition: color 0.2s;
}

.department-search-display:hover .edit-icon {
  color: var(--el-color-primary);
}
</style>
