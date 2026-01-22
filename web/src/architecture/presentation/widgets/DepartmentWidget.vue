<!--
  DepartmentWidget - 组织架构组件
  功能：
  - 输入场景（edit/search）：组织架构选择器，支持搜索
  - 输出场景（response/table-cell/detail）：显示组织架构信息
-->
<template>
  <div class="department-widget">
    <!-- 编辑模式：组织架构选择器（使用弹窗搜索） -->
    <div v-if="mode === 'edit' || mode === 'search'" class="department-select-wrapper">
      <!-- 选中后的显示 -->
      <div
        v-if="selectedDepartmentForDisplay"
        class="department-select-display"
        :class="{ 'is-disabled': false }"
        @click="handleOpenDialog()"
      >
        <img src="/组织架构.svg" alt="组织架构" class="department-icon-small" />
        <span class="department-display-text">
          {{ formatDepartmentDisplayName(selectedDepartmentForDisplay) }}
        </span>
        <el-icon class="edit-icon">
          <Edit />
        </el-icon>
      </div>
      <!-- 未选中时显示按钮 -->
      <el-button
        v-else
        :disabled="false"
        :placeholder="field.desc || `请选择${field.name}`"
        @click="handleOpenDialog()"
      >
        <el-icon><OfficeBuilding /></el-icon>
        {{ field.desc || `请选择${field.name}` }}
      </el-button>
      
      <!-- 组织架构选择弹窗 -->
      <DepartmentSelectorDialog
        v-model="dialogVisible"
        :selected-department="selectedDepartmentForDisplay"
        @select="handleDepartmentSelected"
      />
    </div>
    
    <!-- 响应模式（使用 DepartmentDisplay 组件） -->
    <DepartmentDisplay
      v-else-if="mode === 'response'"
      :department-info="departmentInfo"
      :full-code-path="value?.raw"
      :display-name="value?.display"
      mode="card"
      layout="horizontal"
      size="small"
    />
    
    <!-- 表格单元格模式（使用 DepartmentDisplay 组件） -->
    <DepartmentDisplay
      v-else-if="mode === 'table-cell'"
      :department-info="departmentInfo"
      :full-code-path="value?.raw"
      :display-name="value?.display"
      mode="card"
      layout="horizontal"
      size="small"
    />
    
    <!-- 详情模式（使用 DepartmentDisplay 组件） -->
    <div v-else-if="mode === 'detail'" class="department-detail">
      <DepartmentDisplay
        :department-info="departmentInfo"
        :full-code-path="value?.raw"
        :display-name="value?.display"
        mode="card"
        layout="horizontal"
        size="large"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import DepartmentDisplay from './DepartmentDisplay.vue'
import DepartmentSelectorDialog from '@/components/DepartmentSelectorDialog.vue'
import { ElButton, ElIcon } from 'element-plus'
import { OfficeBuilding, Edit } from '@element-plus/icons-vue'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { useDepartmentInfoStore } from '@/stores/departmentInfo'
import type { Department } from '@/api/department'
import { getDepartmentTree, getDepartmentByPath } from '@/api/department'
import { Logger } from '@/core/utils/logger'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'

const COMPONENT_NAME = 'DepartmentWidget'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()
const departmentInfoStore = useDepartmentInfoStore()

// 弹窗显示状态
const dialogVisible = ref(false)

// 当前组织架构信息（用于显示）
const departmentInfo = ref<Department | null>(null)

// 处理打开弹窗
function handleOpenDialog(): void {
  dialogVisible.value = true
}

// 处理组织架构选择
function handleDepartmentSelected(department: Department): void {
  // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
  const newFieldValue = createFieldValue(
    props.field,
    department.full_code_path, // 提交时只提交 full_code_path
    department.full_name_path || department.name, // 显示名称
    {
      departmentInfo: department
    }
  )
  
  formDataStore.setValue(props.fieldPath, newFieldValue)
  emit('update:modelValue', newFieldValue)
  
  // 更新 departmentInfo 用于显示
  departmentInfo.value = department
}

// 格式化组织架构显示名称
function formatDepartmentDisplayName(dept: Department): string {
  if (dept.full_name_path && dept.full_name_path !== dept.name) {
    return `${dept.name} (${dept.full_name_path})`
  }
  return dept.name
}

// 选中组织架构（用于显示）
const selectedDepartmentForDisplay = computed(() => {
  if (props.mode === 'edit' || props.mode === 'search') {
    const currentValue = props.value?.raw
    if (currentValue) {
      // 从 meta 中获取（优先）
      if (props.value?.meta?.departmentInfo && props.value.meta.departmentInfo.full_code_path === currentValue) {
        departmentInfo.value = props.value.meta.departmentInfo
        return props.value.meta.departmentInfo
      }
      
      // 从 departmentInfo 中获取（可能是刚加载的）
      if (departmentInfo.value && departmentInfo.value.full_code_path === currentValue) {
        return departmentInfo.value
      }
      
      // 🔥 如果都没有，loadDepartmentInfo 会从 API 加载
    }
  }
  return null
})

// 加载组织架构信息（用于显示）
async function loadDepartmentInfo(fullCodePath: string | null): Promise<Department | null> {
  if (!fullCodePath) {
    departmentInfo.value = null
    return null
  }
  
  // 如果 meta 中已有组织架构信息，直接使用
  if (props.value?.meta?.departmentInfo && props.value.meta.departmentInfo.full_code_path === fullCodePath) {
    departmentInfo.value = props.value.meta.departmentInfo
    return props.value.meta.departmentInfo
  }
  
  // 🔥 优先从 store 缓存中获取（如果预加载已完成，这里会命中缓存）
  const cachedDept = departmentInfoStore.departmentInfoCache.get(fullCodePath)
  if (cachedDept) {
    departmentInfo.value = cachedDept
    return cachedDept
  }
  
  // 🔥 如果缓存中没有，使用 getDepartmentInfo（会自动处理缓存和降级策略）
  try {
    const dept = await departmentInfoStore.getDepartmentInfo(fullCodePath)
    if (dept) {
      departmentInfo.value = dept
      return dept
    } else {
      departmentInfo.value = null
      return null
    }
  } catch (error) {
    // 查询组织架构信息失败，静默处理
    Logger.error(COMPONENT_NAME, '查询组织架构信息失败', { fullCodePath, error })
    departmentInfo.value = null
    return null
  }
}

// 监听值变化，加载组织架构信息
watch(() => props.value?.raw, (newValue: any) => {
  if (props.mode === 'edit' || props.mode === 'search') {
    // 编辑模式：如果有值，加载组织架构信息用于显示
    if (newValue) {
      loadDepartmentInfo(String(newValue))
    } else {
      departmentInfo.value = null
    }
  } else {
    // 显示模式：加载组织架构信息用于显示
    if (newValue) {
      loadDepartmentInfo(String(newValue))
    } else {
      departmentInfo.value = null
    }
  }
}, { immediate: true })

// 监听 mode 变化，如果切换到显示模式，加载组织架构信息
watch(() => props.mode, (newMode: string) => {
  if (newMode !== 'edit' && newMode !== 'search' && props.value?.raw) {
    loadDepartmentInfo(String(props.value.raw))
  }
})

// 组件挂载时，如果有初始值，加载组织架构信息
// 🔥 同时检查是否有动态默认值（如 MyDepartment()）
onMounted(async () => {
  // 🔥 检查是否有动态默认值需要设置（MyDepartment()）
  // ⚠️ 重要：只有在新增模式下才使用默认值，编辑模式下不应该使用默认值
  if (props.mode === 'edit') {
    // ⚠️ 使用 nextTick 等待一下，确保 initializeForm 已经完成
    // 这样可以避免在编辑模式下错误地使用默认值
    await nextTick()
    
    const currentRaw = props.value?.raw
    const existingValue = formDataStore.getValue(props.fieldPath)
    
    // 🔥 检查是否需要解析 MyDepartment() 函数调用
    // 情况1：value.raw 是 "MyDepartment()" 字符串（FormDomainService 还没有解析）
    // 情况2：value.raw 是 null/undefined/空字符串，且配置中有 "MyDepartment()" 默认值
    const needsResolveMyDepartment = currentRaw === 'MyDepartment()' || 
      ((!currentRaw || currentRaw === '') && 
       props.field.widget?.config?.default === 'MyDepartment()')
    
    if (needsResolveMyDepartment) {
      // ⚠️ 检查是否是编辑模式：
      // 1. 如果 meta.fromInitialData 为 true，说明字段来自 initialData（编辑模式）
      // 2. 如果 existingValue 存在且 raw 不是 "MyDepartment()"，说明是编辑模式
      // 编辑模式下，existingValue.raw 应该是实际的 full_code_path，不应该是 "MyDepartment()"
      const isEditMode = props.value?.meta?.fromInitialData === true ||
                        (existingValue && 
                         existingValue.raw !== null && 
                         existingValue.raw !== undefined && 
                         existingValue.raw !== '' && 
                         existingValue.raw !== 'MyDepartment()')
      
      // 只有在新增模式下才解析 MyDepartment()
      if (!isEditMode) {
        const { useAuthStore } = await import('@/stores/auth')
        const authStore = useAuthStore()
        const currentUserDepartmentPath = authStore.user?.department_full_path
        if (currentUserDepartmentPath) {
          // 加载组织架构信息
          const dept = await loadDepartmentInfo(currentUserDepartmentPath)
          if (dept) {
            // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
            const newFieldValue = createFieldValue(
              props.field,
              currentUserDepartmentPath,
              dept.full_name_path || dept.name,
              {
                departmentInfo: dept
              }
            )
            formDataStore.setValue(props.fieldPath, newFieldValue)
            emit('update:modelValue', newFieldValue)
            return
          }
        }
      }
    }
  }

  if (props.value?.raw) {
    // 加载组织架构信息用于显示
    loadDepartmentInfo(String(props.value.raw))
  }
})
</script>

<style scoped>
.department-widget {
  width: 100%;
}

/* 选择器包装器 */
.department-select-wrapper {
  position: relative;
  width: 100%;
}

/* 选中后的显示（可点击） */
.department-select-display {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background-color: var(--el-bg-color);
  cursor: pointer;
  transition: all 0.2s;
}

.department-select-display:hover:not(.is-disabled) {
  border-color: var(--el-color-primary);
  background-color: var(--el-fill-color-light);
}

.department-select-display.is-disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.department-icon-small {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.department-select-display .department-display-text {
  flex: 1;
  font-size: 14px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.department-select-display .edit-icon {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
  font-size: 16px;
  transition: color 0.2s;
}

.department-select-display:hover:not(.is-disabled) .edit-icon {
  color: var(--el-color-primary);
}

/* 详情模式样式 */
.department-detail {
  display: flex;
  align-items: center;
  gap: 16px;
}
</style>
