<!--
  DepartmentSelector - 组织架构选择器组件
  功能：
  - 显示当前选中的组织架构（使用 DepartmentDisplay）
  - 点击后弹出对话框，显示组织架构树供选择
  - 支持清空选择
-->
<template>
  <div class="department-selector">
    <div v-if="selectedDepartmentPath" class="selected-department">
      <div class="selected-department-main" @click="dialogVisible = true">
        <img src="/组织架构.svg" alt="组织架构" class="selected-department-icon" width="18" height="18" />
        <span class="selected-department-text">{{ selectedDepartmentLabel }}</span>
      </div>
      <div class="selected-department-actions">
        <el-button link size="small" @click="dialogVisible = true">重新选择</el-button>
        <el-button
          type="danger"
          link
          size="small"
          @click="handleClear"
        >
          清空
        </el-button>
      </div>
    </div>
    
    <!-- 未选择时显示按钮 -->
    <el-button
      v-else
      :icon="OfficeBuilding"
      @click="dialogVisible = true"
    >
      {{ placeholder }}
    </el-button>

    <DepartmentPickerDialog
      v-model="dialogVisible"
      :initial-paths="selectedDepartmentPath"
      :department-tree="departmentTree"
      @confirm="handleDepartmentsConfirm"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElButton } from 'element-plus'
import { OfficeBuilding } from '@element-plus/icons-vue'
import type { Department } from '@/architecture/infrastructure/api/department'
import DepartmentPickerDialog from '@/shared/components/DepartmentPickerDialog.vue'

interface Props {
  modelValue: string | null
  departmentTree?: Department[]
  placeholder?: string
}

const props = withDefaults(defineProps<Props>(), {
  departmentTree: () => [],
  placeholder: '选择组织架构'
})

const emit = defineEmits<{
  'update:modelValue': [value: string | null]
}>()

const dialogVisible = ref(false)

// 部门树数据
const departmentTree = computed(() => {
  return props.departmentTree || []
})

// 当前选中的组织架构路径
const selectedDepartmentPath = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

function findDepartmentByPath(departments: Department[], path: string): Department | null {
  for (const department of departments) {
    if (department.full_code_path === path) {
      return department
    }
    if (department.children?.length) {
      const match = findDepartmentByPath(department.children, path)
      if (match) {
        return match
      }
    }
  }
  return null
}

const selectedDepartmentLabel = computed(() => {
  if (!selectedDepartmentPath.value) {
    return ''
  }

  const department = findDepartmentByPath(departmentTree.value, selectedDepartmentPath.value)
  if (!department) {
    return selectedDepartmentPath.value
  }

  return department.full_name_path || department.name
})

function handleDepartmentsConfirm(departments: Department[]) {
  const department = departments[0]
  selectedDepartmentPath.value = department ? department.full_code_path : null
}

// 清空选择
function handleClear() {
  selectedDepartmentPath.value = null
}
</script>

<style scoped>
.department-selector {
  width: 100%;
}

.selected-department {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background-color: var(--el-bg-color);
}

.selected-department-main {
  min-width: 0;
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.selected-department-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.selected-department-icon {
  width: 18px;
  height: 18px;
  min-width: 18px;
  min-height: 18px;
  max-width: 18px;
  max-height: 18px;
  flex-shrink: 0;
  object-fit: contain;
  display: block;
}

.selected-department-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--el-text-color-primary);
  font-size: 14px;
}
</style>
