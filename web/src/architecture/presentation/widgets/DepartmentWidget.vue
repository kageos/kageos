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
      <!-- 搜索模式下支持多选时的显示 -->
      <div
        v-if="mode === 'search' && supportsMultipleSelection && selectedDepartmentsForDisplay.length > 0"
        class="departments-select-display"
        @click="handleOpenDialog()"
      >
        <div class="selected-departments-list">
          <div
            v-for="(dept, index) in selectedDepartmentsForDisplay"
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
      <!-- 单选模式或未选中时的显示 -->
      <div
        v-else-if="selectedDepartmentForDisplay"
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
      <!-- 搜索模式下，如果支持 IN 查询，使用多选对话框 -->
      <DepartmentsSearchDialog
        v-if="mode === 'search' && supportsMultipleSelection"
        v-model="dialogVisible"
        :title="`选择${field.name || '组织架构'}`"
        :placeholder="field.desc || '搜索部门名称或路径...'"
        :initial-paths="value?.raw"
        @confirm="handleDepartmentsSelected"
      />
      <!-- 其他情况使用单选对话框 -->
      <DepartmentSelectorDialog
        v-else
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
    
    <!-- 表格单元格模式（使用 DepartmentDisplay 组件，显示最后一段） -->
    <DepartmentDisplay
      v-else-if="mode === 'table-cell'"
      :department-info="departmentInfoForDisplay"
      :full-code-path="value?.raw"
      :display-name="departmentDisplayName"
      mode="card"
      layout="horizontal"
      size="small"
      :show-full-path="false"
    />
    
    <!-- 详情模式（使用 DepartmentDisplay 组件） -->
    <div v-else-if="mode === 'detail'" class="department-detail">
      <DepartmentDisplay
        :department-info="departmentInfoForDisplay"
        :full-code-path="value?.raw"
        :display-name="departmentDisplayName"
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
import DepartmentsSearchDialog from './DepartmentsSearchDialog.vue'
import { ElButton, ElIcon } from 'element-plus'
import { OfficeBuilding, Edit, Close } from '@element-plus/icons-vue'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { useDepartmentInfoStore } from '@/stores/departmentInfo'
import type { Department } from '@/api/department'
import { getDepartmentTree, getDepartmentByPath } from '@/api/department'
import { Logger } from '@/core/utils/logger'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import { SearchType, hasSearchType } from '@/core/constants/search'

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

// 是否支持多选（搜索模式下且支持 IN 查询）
const supportsMultipleSelection = computed(() => {
  if (props.mode !== 'search') {
    return false
  }
  const searchType = props.field.search || ''
  return hasSearchType(searchType, SearchType.IN)
})

// 处理组织架构选择（单选）
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

// 处理组织架构选择（多选，搜索模式下使用）
function handleDepartmentsSelected(departments: Department[]): void {
  // 将组织架构列表转换为逗号分隔的字符串
  const paths = departments.map(d => d.full_code_path).join(',')
  const displayNames = departments.map(d => d.full_name_path || d.name).join(', ')
  
  // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
  const newFieldValue = createFieldValue(
    props.field,
    paths, // 提交时使用逗号分隔的字符串
    displayNames,
    {
      departmentInfoList: departments // 保存组织架构信息列表到 meta，用于显示
    }
  )
  
  formDataStore.setValue(props.fieldPath, newFieldValue)
  emit('update:modelValue', newFieldValue)
  
  // 更新 departmentInfo 用于显示（多选模式下，取第一个）
  if (departments.length > 0) {
    departmentInfo.value = departments[0]
  } else {
    departmentInfo.value = null
  }
}

// 格式化组织架构显示名称
function formatDepartmentDisplayName(dept: Department): string {
  if (dept.full_name_path && dept.full_name_path !== dept.name) {
    return `${dept.name} (${dept.full_name_path})`
  }
  return dept.name
}

// 选中的部门列表（用于多选模式显示）
const selectedDepartmentsForDisplay = computed(() => {
  if (props.mode === 'search' && supportsMultipleSelection.value) {
    // 优先从 meta 中获取
    if (props.value?.meta?.departmentInfoList && Array.isArray(props.value.meta.departmentInfoList)) {
      return props.value.meta.departmentInfoList
    }
    // 如果 value.raw 是逗号分隔的字符串，尝试加载
    if (props.value?.raw) {
      const paths = String(props.value.raw).split(',').map(p => p.trim()).filter(p => p)
      if (paths.length > 0) {
        // 异步加载部门信息（这里只返回空数组，实际显示会通过 loadDepartmentsInfo 更新）
        loadDepartmentsInfoForMultiSelect(paths)
      }
    }
  }
  return []
})

// 选中组织架构（用于单选模式显示）
const selectedDepartmentForDisplay = computed(() => {
  if (props.mode === 'edit' || (props.mode === 'search' && !supportsMultipleSelection.value)) {
    const currentValue = props.value?.raw
    if (currentValue) {
      // 单选模式：从 meta 中获取（优先）
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

// 多选模式下加载部门信息列表
const departmentInfoListForMultiSelect = ref<Department[]>([])

async function loadDepartmentsInfoForMultiSelect(paths: string[]): Promise<void> {
  if (paths.length === 0) {
    departmentInfoListForMultiSelect.value = []
    return
  }
  
  try {
    const departments: Department[] = []
    
    // 🔥 先从缓存中获取已有的部门信息
    const cachedDepartments: Department[] = []
    const uncachedPaths: string[] = []
    
    paths.forEach(path => {
      const cachedDept = departmentInfoStore.departmentInfoCache.get(path)
      if (cachedDept) {
        cachedDepartments.push(cachedDept)
      } else {
        uncachedPaths.push(path)
      }
    })
    
    // 🔥 如果有未缓存的路径，批量获取（会自动处理缓存和降级策略）
    if (uncachedPaths.length > 0) {
      const fetchedDepartments = await departmentInfoStore.batchGetDepartmentInfo(uncachedPaths)
      departments.push(...fetchedDepartments)
    }
    
    // 合并缓存和获取的部门信息，按原始顺序排列
    paths.forEach(path => {
      const dept = [...cachedDepartments, ...departments].find(d => d.full_code_path === path)
      if (dept && !departmentInfoListForMultiSelect.value.find(d => d.full_code_path === path)) {
        departmentInfoListForMultiSelect.value.push(dept)
      }
    })
  } catch (error) {
    Logger.error(COMPONENT_NAME, '加载组织架构信息列表失败', { paths, error })
    departmentInfoListForMultiSelect.value = []
  }
}

// 移除单个组织架构（多选模式）
async function handleRemoveDepartment(dept: Department): Promise<void> {
  const currentPaths = props.value?.raw ? String(props.value.raw).split(',').map(p => p.trim()).filter(p => p) : []
  const newPaths = currentPaths.filter(p => p !== dept.full_code_path)
  
  if (newPaths.length > 0) {
    await loadDepartmentsInfoForMultiSelect(newPaths)
    const displayNames = departmentInfoListForMultiSelect.value.map(d => d.full_name_path || d.name).join(', ')
    
    const newFieldValue = createFieldValue(
      props.field,
      newPaths.join(','),
      displayNames,
      {
        departmentInfoList: departmentInfoListForMultiSelect.value
      }
    )
    formDataStore.setValue(props.fieldPath, newFieldValue)
    emit('update:modelValue', newFieldValue)
  } else {
    const newFieldValue = createFieldValue(
      props.field,
      '',
      '',
      {}
    )
    formDataStore.setValue(props.fieldPath, newFieldValue)
    emit('update:modelValue', newFieldValue)
    departmentInfoListForMultiSelect.value = []
  }
}

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

// 用于显示的部门信息（所有模式都使用）
const departmentInfoForDisplay = computed(() => {
  // 优先使用 departmentInfo（已加载的）
  if (departmentInfo.value) {
    return departmentInfo.value
  }
  // 如果 meta 中有部门信息，使用它
  if (props.value?.meta?.departmentInfo) {
    return props.value.meta.departmentInfo
  }
  // 如果 store 缓存中有，使用它
  if (props.value?.raw) {
    const cachedDept = departmentInfoStore.departmentInfoCache.get(props.value.raw)
    if (cachedDept) {
      return cachedDept
    }
  }
  return null
})

// 用于显示的部门名称
// 注意：table-cell 模式下，DepartmentDisplay 会根据 showFullPath 配置来决定显示什么
// 这里返回 null，让 DepartmentDisplay 自己根据 showFullPath 处理
const departmentDisplayName = computed(() => {
  // 如果 value.display 有值且不是 full-code-path，使用它
  if (props.value?.display && props.value.display !== props.value?.raw) {
    return props.value.display
  }
  // 返回 null，让 DepartmentDisplay 自己根据 showFullPath 处理
  return null
})

// 监听值变化，加载组织架构信息
watch(() => props.value?.raw, (newValue: any) => {
  // 搜索模式下支持多选时，加载多个部门信息
  if (props.mode === 'search' && supportsMultipleSelection.value) {
    if (newValue) {
      const paths = String(newValue).split(',').map(p => p.trim()).filter(p => p)
      loadDepartmentsInfoForMultiSelect(paths)
    } else {
      departmentInfoListForMultiSelect.value = []
    }
  } else {
    // 其他模式：加载单个部门信息
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

/* 多选模式下的显示样式 */
.departments-select-display {
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

.departments-select-display:hover {
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
  padding: 4px 8px;
  background-color: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  font-size: 12px;
}

.selected-department-tag .department-icon-small {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.selected-department-tag .department-display-text {
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

.selected-department-tag .remove-icon {
  cursor: pointer;
  color: var(--el-text-color-secondary);
  font-size: 14px;
  margin-left: 4px;
  transition: color 0.2s;
}

.selected-department-tag .remove-icon:hover {
  color: var(--el-color-primary);
}

.departments-select-display .edit-icon {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
  font-size: 16px;
  transition: color 0.2s;
}

.departments-select-display:hover .edit-icon {
  color: var(--el-color-primary);
}

/* 详情模式样式 */
.department-detail {
  display: flex;
  align-items: center;
  gap: 16px;
}
</style>
