<!--
  DepartmentsWidget - 多组织架构组件
  功能：
  - 输入场景（edit/search）：多组织架构选择器，支持搜索
  - 输出场景（response/table-cell/detail）：显示多个组织架构信息
  - 值使用逗号分隔的字符串存储（如 "/dept1,/dept2"），便于存储到数据库
-->
<template>
  <div class="departments-widget">
    <!-- 编辑模式：多组织架构选择器（使用弹窗搜索） -->
    <div v-if="mode === 'edit' || mode === 'search'" class="departments-select-wrapper">
      <!-- 选中后的显示 -->
      <div
        v-if="selectedDepartmentsForDisplay.length > 0"
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
              {{ formatDepartmentDisplayName(dept) }}
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
      <!-- 未选中时显示按钮 -->
      <el-button
        v-else
        :placeholder="field.desc || `请选择${field.name}`"
        @click="handleOpenDialog()"
      >
        <el-icon><OfficeBuilding /></el-icon>
        {{ field.desc || `请选择${field.name}` }}
      </el-button>
      
      <!-- 多组织架构搜索弹窗 -->
      <DepartmentsSearchDialog
        v-model="dialogVisible"
        :title="`选择${field.name || '组织架构'}`"
        :placeholder="field.desc || '搜索部门名称或路径...'"
        :initial-paths="value?.raw"
        :max-count="maxCount"
        @confirm="handleDepartmentsSelected"
      />
    </div>
    
    <!-- 响应模式：显示多个组织架构 -->
    <div v-else-if="mode === 'response'" class="departments-response">
      <div v-if="displayDepartments.length > 0" class="departments-list departments-list-horizontal">
        <DepartmentDisplay
          v-for="(dept, index) in displayDepartments"
          :key="dept.full_code_path || index"
          :department-info="dept"
          :full-code-path="dept.full_code_path"
          :display-name="dept.full_name_path || dept.name"
          mode="card"
          layout="horizontal"
          size="small"
          class="department-item"
        />
      </div>
      <span v-else class="empty-text">-</span>
    </div>
    
    <!-- 表格单元格模式：显示组织架构名称 -->
    <div v-else-if="mode === 'table-cell'" class="departments-table-cell">
      <div v-if="displayDepartments.length > 0" class="departments-tags-list">
        <el-tag
          v-for="(dept, index) in displayDepartments"
          :key="dept.full_code_path || index"
          size="small"
          class="department-tag"
        >
          {{ dept.name }}
        </el-tag>
      </div>
      <span v-else class="empty-text">-</span>
    </div>
    
    <!-- 详情模式：显示组织架构列表 -->
    <div v-else-if="mode === 'detail'" class="departments-detail">
      <div v-if="displayDepartments.length > 0" class="departments-detail-list">
        <DepartmentDisplay
          v-for="(dept, index) in displayDepartments"
          :key="dept.full_code_path || index"
          :department-info="dept"
          :full-code-path="dept.full_code_path"
          :display-name="dept.full_name_path || dept.name"
          mode="card"
          layout="horizontal"
          size="medium"
          class="department-detail-item"
        />
      </div>
      <span v-else class="empty-text">-</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import DepartmentDisplay from './DepartmentDisplay.vue'
import DepartmentsSearchDialog from './DepartmentsSearchDialog.vue'
import { ElButton, ElIcon, ElTag } from 'element-plus'
import { OfficeBuilding, Edit, Close } from '@element-plus/icons-vue'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import type { Department } from '@/api/department'
import { getDepartmentTree, getDepartmentByPath } from '@/api/department'
import { Logger } from '@/core/utils/logger'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'

const COMPONENT_NAME = 'DepartmentsWidget'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 弹窗显示状态
const dialogVisible = ref(false)

// 当前组织架构信息列表（用于显示）
const departmentInfoList = ref<Department[]>([])

// 获取配置
const config = computed(() => {
  return (props.field.widget?.config || {}) as DepartmentsWidgetConfig
})

// 最大选择数量
const maxCount = computed(() => {
  return config.value?.max_count || 0
})

interface DepartmentsWidgetConfig {
  default?: string
  max_count?: number
}

// 处理打开弹窗
function handleOpenDialog(): void {
  dialogVisible.value = true
}

// 处理组织架构选择（多个）
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
  
  // 更新 departmentInfoList 用于显示
  departmentInfoList.value = departments
}

// 移除单个组织架构
function handleRemoveDepartment(dept: Department): void {
  const currentPaths = props.value?.raw ? String(props.value.raw).split(',').map(p => p.trim()).filter(p => p) : []
  const newPaths = currentPaths.filter(p => p !== dept.full_code_path)
  
  // 重新加载组织架构信息
  if (newPaths.length > 0) {
    loadDepartmentsInfo(newPaths.join(','))
  } else {
    // 清空
    const newFieldValue = createFieldValue(
      props.field,
      '',
      '',
      {}
    )
    formDataStore.setValue(props.fieldPath, newFieldValue)
    emit('update:modelValue', newFieldValue)
    departmentInfoList.value = []
  }
}

// 格式化组织架构显示名称
function formatDepartmentDisplayName(dept: Department): string {
  if (dept.full_name_path && dept.full_name_path !== dept.name) {
    return `${dept.name} (${dept.full_name_path})`
  }
  return dept.name
}

// 选中组织架构列表（用于编辑模式显示）
const selectedDepartmentsForDisplay = computed(() => {
  if (props.mode === 'edit' || props.mode === 'search') {
    // 优先从 meta 中获取
    if (props.value?.meta?.departmentInfoList && Array.isArray(props.value.meta.departmentInfoList)) {
      return props.value.meta.departmentInfoList
    }
    // 从 departmentInfoList 中获取
    if (departmentInfoList.value.length > 0) {
      return departmentInfoList.value
    }
  }
  return []
})

// 显示组织架构列表（用于响应模式）
const displayDepartments = computed(() => {
  // 优先从 meta 中获取
  if (props.value?.meta?.departmentInfoList && Array.isArray(props.value.meta.departmentInfoList)) {
    return props.value.meta.departmentInfoList
  }
  // 从 departmentInfoList 中获取
  if (departmentInfoList.value.length > 0) {
    return departmentInfoList.value
  }
  return []
})

// 加载组织架构信息列表（用于显示）
async function loadDepartmentsInfo(paths: string): Promise<void> {
  if (!paths || paths.trim() === '') {
    departmentInfoList.value = []
    return
  }
  
  const pathList = paths.split(',').map(p => p.trim()).filter(p => p)
  if (pathList.length === 0) {
    departmentInfoList.value = []
    return
  }
  
  // 并行加载所有组织架构信息
  try {
    const departments: Department[] = []
    
    await Promise.all(
      pathList.map(async (path) => {
        try {
          const res = await getDepartmentByPath(path)
          if (res.department) {
            departments.push(res.department)
          }
        } catch (error) {
          Logger.error(COMPONENT_NAME, '加载组织架构信息失败', { path, error })
        }
      })
    )
    
    departmentInfoList.value = departments
  } catch (error) {
    Logger.error(COMPONENT_NAME, '加载组织架构信息列表失败', { paths, error })
    departmentInfoList.value = []
  }
}

// 监听值变化，加载组织架构信息
watch(() => props.value?.raw, (newValue: any) => {
  if (newValue) {
    loadDepartmentsInfo(String(newValue))
  } else {
    departmentInfoList.value = []
  }
}, { immediate: true })

// 监听 mode 变化，如果切换到显示模式，加载组织架构信息
watch(() => props.mode, (newMode: string) => {
  if (newMode !== 'edit' && newMode !== 'search' && props.value?.raw) {
    loadDepartmentsInfo(String(props.value.raw))
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
    const config = props.field.widget?.config
    const defaultValue = config && typeof config === 'object' && 'default' in config 
      ? (config as Record<string, any>).default 
      : undefined
    
    // 🔥 检查是否需要解析 MyDepartment() 函数调用
    // 情况1：value.raw 是 "MyDepartment()" 字符串（FormDomainService 还没有解析）
    // 情况2：value.raw 包含 "MyDepartment()"（如 "MyDepartment(),/dept2"）
    // 情况3：value.raw 是 null/undefined/空字符串，且配置中有 "MyDepartment()" 默认值
    const needsResolveMyDepartment = (typeof currentRaw === 'string' && currentRaw.includes('MyDepartment()')) ||
      ((!currentRaw || currentRaw === '') && 
       typeof defaultValue === 'string' && defaultValue.includes('MyDepartment()'))
    
    if (needsResolveMyDepartment) {
      // ⚠️ 检查是否是编辑模式：
      // 1. 如果 meta.fromInitialData 为 true，说明字段来自 initialData（编辑模式）
      // 2. 如果 existingValue 存在且 raw 不包含 "MyDepartment()"，说明是编辑模式
      // 编辑模式下，existingValue.raw 应该是实际的 full_code_path，不应该是 "MyDepartment()"
      const isEditMode = props.value?.meta?.fromInitialData === true ||
                        (existingValue && 
                         existingValue.raw !== null && 
                         existingValue.raw !== undefined && 
                         existingValue.raw !== '' && 
                         (typeof existingValue.raw !== 'string' || !existingValue.raw.includes('MyDepartment()')))
      
      // 只有在新增模式下才解析 MyDepartment()
      if (!isEditMode) {
        const { useAuthStore } = await import('@/stores/auth')
        const authStore = useAuthStore()
        const currentUserDepartmentPath = authStore.user?.department_full_path
        if (currentUserDepartmentPath) {
          // 加载组织架构信息
          const dept = await getDepartmentByPath(currentUserDepartmentPath).then(res => res.department).catch(() => null)
          if (dept) {
            let processedValue: string
            
            // 处理 MyDepartment() 格式
            if (typeof defaultValue === 'string' && defaultValue === 'MyDepartment()') {
              // 单个 MyDepartment()
              processedValue = currentUserDepartmentPath
            } else if (typeof defaultValue === 'string' && defaultValue.includes(',')) {
              // 多个默认值，用逗号分隔（如 "MyDepartment(),/dept2"）
              processedValue = defaultValue.replace(/MyDepartment\(\)/g, currentUserDepartmentPath)
            } else if (typeof currentRaw === 'string' && currentRaw === 'MyDepartment()') {
              // value.raw 是 "MyDepartment()"，直接替换
              processedValue = currentUserDepartmentPath
            } else if (typeof currentRaw === 'string' && currentRaw.includes('MyDepartment()')) {
              // value.raw 包含 "MyDepartment()"（如 "MyDepartment(),/dept2"）
              processedValue = currentRaw.replace(/MyDepartment\(\)/g, currentUserDepartmentPath)
            } else {
              processedValue = currentUserDepartmentPath
            }
            
            if (processedValue && processedValue.trim()) {
              // 加载所有组织架构信息
              await loadDepartmentsInfo(processedValue)
              const displayNames = departmentInfoList.value.map(d => d.full_name_path || d.name).join(', ')
              
              // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
              const newFieldValue = createFieldValue(
                props.field,
                processedValue,
                displayNames,
                {
                  departmentInfoList: departmentInfoList.value
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
  }

  if (props.value?.raw) {
    // 加载组织架构信息用于显示
    loadDepartmentsInfo(String(props.value.raw))
  }
})
</script>

<style scoped>
.departments-widget {
  width: 100%;
}

/* 选择器包装器 */
.departments-select-wrapper {
  position: relative;
  width: 100%;
}

/* 选中后的显示（可点击） */
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
  gap: 8px;
  flex: 1;
}

.selected-department-tag {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background-color: transparent;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  font-size: 12px;
}

.department-icon-small {
  width: 16px;
  height: 16px;
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
  transition: color 0.2s;
}

.remove-icon:hover {
  color: var(--el-color-danger);
}

.edit-icon {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
  font-size: 16px;
  transition: color 0.2s;
}

.departments-select-display:hover .edit-icon {
  color: var(--el-color-primary);
}

/* 响应模式样式 */
.departments-response {
  width: 100%;
}

.departments-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.departments-list-horizontal {
  flex-direction: row;
  flex-wrap: wrap;
}

.department-item {
  flex-shrink: 0;
}

/* 表格单元格模式样式 */
.departments-table-cell {
  width: 100%;
}

.departments-tags-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.department-tag {
  margin: 0;
}

/* 详情模式样式 */
.departments-detail {
  width: 100%;
}

.departments-detail-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.department-detail-item {
  width: 100%;
}

.empty-text {
  color: var(--el-text-color-placeholder);
  font-size: 14px;
}
</style>
