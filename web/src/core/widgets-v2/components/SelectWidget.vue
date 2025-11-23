<!--
  SelectWidget - 下拉选择组件
  🔥 完全新增，不依赖旧代码
  
  功能：
  - 支持静态选项
  - 支持回调接口（OnSelectFuzzy）
  - 支持 displayInfo 显示
  - 支持聚合统计
-->

<template>
  <div class="select-widget">
    <!-- 编辑模式 -->
    <el-select
      v-if="mode === 'edit'"
      v-model="internalValue"
      :disabled="field.widget?.config?.disabled"
      :placeholder="field.desc || `请选择${field.name}`"
      :clearable="true"
      :filterable="isFilterable"
      :loading="loading"
      :remote="hasCallback"
      :remote-method="handleRemoteSearch"
      popper-class="select-dropdown-popper"
      :popper-options="{
        strategy: 'fixed',
        modifiers: [
          {
            name: 'computeStyles',
            options: {
              adaptive: false,
              roundOffsets: false
            }
          },
          {
            name: 'offset',
            options: {
              offset: [0, 4]
            }
          }
        ]
      }"
      @change="handleChange"
      @focus="handleFocus"
    >
      <el-option
        v-for="option in options"
        :key="option.value"
        :label="option.label"
        :value="option.value"
        :disabled="option.disabled"
      >
        <div class="select-option">
          <!-- 🔥 显示颜色指示器（如果有颜色配置，放在左侧） -->
          <span
            v-if="getOptionColor(option.value)"
            class="option-color-indicator"
            :style="getOptionColorStyle(option.value)"
          />
          <span class="option-label">{{ option.label }}</span>
          <span v-if="option.displayInfo" class="display-info">{{ option.display-info }}</span>
        </div>
      </el-option>
    </el-select>
    
    <!-- 响应模式（只读） -->
    <span v-else-if="mode === 'response'" class="response-value">
      {{ displayValue }}
    </span>
    
    <!-- 表格单元格模式 -->
    <div v-else-if="mode === 'table-cell'" class="table-cell-value">
      <el-tag
        v-if="currentOptionColor"
        :type="isStandardColor(currentOptionColor) ? currentOptionColor : undefined"
        :color="!isStandardColor(currentOptionColor) ? currentOptionColor : undefined"
        size="small"
        class="select-tag select-tag-outline"
      >
        {{ displayValue }}
      </el-tag>
      <span v-else>{{ displayValue }}</span>
    </div>
    
    <!-- 详情模式 -->
    <div v-else-if="mode === 'detail'" class="detail-value">
      <el-tag
        v-if="currentOptionColor"
        :type="isStandardColor(currentOptionColor) ? currentOptionColor : undefined"
        :color="!isStandardColor(currentOptionColor) ? currentOptionColor : undefined"
        class="select-tag select-tag-outline"
      >
        {{ displayValue }}
      </el-tag>
      <span v-else class="detail-content">{{ displayValue }}</span>
    </div>
    
    <!-- 搜索模式 -->
    <el-select
      v-else-if="mode === 'search'"
      v-model="internalValue"
      :placeholder="`搜索${field.name}`"
      :clearable="true"
      :filterable="isFilterable"
      :loading="loading"
      :remote="hasCallback"
      :remote-method="handleRemoteSearch"
    >
      <el-option
        v-for="option in options"
        :key="option.value"
        :label="option.label"
        :value="option.value"
      >
        <!-- 🔥 显示颜色指示器（如果有颜色配置，放在左侧） -->
        <div class="select-option">
          <span
            v-if="getOptionColor(option.value)"
            class="option-color-indicator"
            :style="getOptionColorStyle(option.value)"
          />
          <span class="option-label">{{ option.label }}</span>
        </div>
      </el-option>
    </el-select>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElSelect, ElOption, ElMessage, ElTag } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'
import { selectFuzzy } from '@/api/function'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 选项列表
const options = ref<Array<{ label: string; value: any; disabled?: boolean; displayInfo?: string }>>([])

/**
 * 🔥 选项颜色配置
 * 
 * 支持两种颜色格式：
 * 1. Element Plus 标准颜色类型：success, warning, danger, info, primary
 *    使用 el-tag 的 type 属性
 * 2. 自定义颜色（hex 格式）：如 #FF5722, #4CAF50
 *    使用 el-tag 的 color 属性
 * 
 * options_colors 数组与 options 数组的索引对齐，通过索引获取对应选项的颜色
 */
const optionColors = computed(() => {
  return props.field.widget?.config?.options_colors || []
})

/**
 * 判断是否是 Element Plus 标准颜色类型
 * 标准颜色类型：success, warning, danger, info, primary
 * 这些颜色使用 el-tag 的 type 属性
 */
function isStandardColor(color: string): boolean {
  return ['success', 'warning', 'danger', 'info', 'primary'].includes(color)
}

/**
 * 获取当前选中值的颜色
 * 通过查找当前值在 options 中的索引，从 optionColors 数组中获取对应颜色
 * options_colors 数组与 options 数组的索引对齐
 */
const currentOptionColor = computed(() => {
  const rawValue = props.value?.raw
  if (rawValue === null || rawValue === undefined || rawValue === '') {
    return null
  }
  
  // 查找当前值在 options 中的索引
  const optionIndex = options.value.findIndex(opt => opt.value === rawValue)
  if (optionIndex >= 0 && optionIndex < optionColors.value.length) {
    return optionColors.value[optionIndex]
  }
  
  return null
})

/**
 * 🔥 获取选项的颜色（用于下拉选项显示）
 */
function getOptionColor(value: any): string | null {
  const optionIndex = options.value.findIndex(opt => opt.value === value)
  if (optionIndex >= 0 && optionIndex < optionColors.value.length) {
    return optionColors.value[optionIndex]
  }
  return null
}

/**
 * 🔥 获取选项的颜色样式对象（用于 span 的 style 绑定）
 */
function getOptionColorStyle(value: any): Record<string, string> {
  const color = getOptionColor(value)
  if (!color) return {}
  
  const isStandard = isStandardColor(color)
  const backgroundColor = isStandard ? undefined : color
  
  return {
    backgroundColor: backgroundColor || '',
    marginRight: '8px',
    display: 'inline-block',
    width: '12px',
    height: '12px',
    minWidth: '12px',
    minHeight: '12px',
    borderRadius: '2px',
    flexShrink: '0',
    border: 'none',
    verticalAlign: 'middle',
    /* 🔥 降低亮度：使用 filter 降低饱和度和亮度 */
    filter: 'brightness(0.95) saturate(0.9)',
    opacity: '0.9'
  }
}

// 加载状态
const loading = ref(false)

// 是否有回调接口
const hasCallback = computed(() => {
  return props.field.callbacks?.includes('OnSelectFuzzy') || false
})

// 是否可搜索
const isFilterable = computed(() => {
  return props.field.widget?.config?.filterable !== false
})

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit' || props.mode === 'search') {
      return props.value?.raw ?? null
    }
    return null
  },
  set: (newValue: any) => {
    if (props.mode === 'edit') {
      const selectedOption = options.value.find(opt => opt.value === newValue)
      const newFieldValue = {
        raw: newValue,
        display: selectedOption?.label || String(newValue),
        meta: {
          displayInfo: selectedOption?.displayInfo
        }
      }
      
      formDataStore.setValue(props.fieldPath, newFieldValue)
      emit('update:modelValue', newFieldValue)
    }
  }
})

// 显示值
const displayValue = computed(() => {
  const value = props.value
  if (!value) {
    return '-'
  }
  
  if (value.display) {
    return value.display
  }
  
  const raw = value.raw
  if (raw === null || raw === undefined || raw === '') {
    return '-'
  }
  
  return String(raw)
})

// 初始化选项
function initOptions(): void {
  const configOptions = props.field.widget?.config?.options
  if (configOptions && Array.isArray(configOptions)) {
    if (typeof configOptions[0] === 'string') {
      // 字符串数组
      options.value = configOptions.map(opt => ({
        label: opt,
        value: opt
      }))
    } else {
      // 对象数组
      options.value = configOptions
    }
  }
  
  // 如果有回调接口且有初始值，触发一次搜索
  if (hasCallback.value && props.value?.raw) {
    handleSearch('', true) // by_value
  }
}

// 处理远程搜索
async function handleRemoteSearch(query: string): Promise<void> {
  if (!hasCallback.value) {
    return
  }
  
  await handleSearch(query, false) // by_keyword
}

// 处理搜索
async function handleSearch(query: string, isByValue: boolean): Promise<void> {
  if (!hasCallback.value || !props.formRenderer) {
    return
  }
  
  const method = props.formRenderer.getFunctionMethod()
  const router = props.formRenderer.getFunctionRouter()
  
  if (!router) {
    return
  }
  
  loading.value = true
  
  try {
    const requestBody = {
      code: props.field.code,
      type: isByValue ? 'by_value' : 'by_keyword',
      value: query,
      request: props.formRenderer.getSubmitData(),
      value_type: props.field.data?.type || 'string'
    }
    
    const response = await selectFuzzy(method, router, requestBody)
    
    if (response.error_msg) {
      ElMessage.error(response.error_msg)
      options.value = []
      return
    }
    
    // 🔥 保存 statistics 配置（用于聚合计算）
    if (response.statistics && typeof response.statistics === 'object') {
      currentStatistics.value = response.statistics
      // 如果当前已有选中值，立即更新 meta.statistics
      if (props.value?.raw) {
        const newFieldValue = {
          ...props.value,
          meta: {
            ...props.value.meta,
            statistics: currentStatistics.value
          }
        }
        formDataStore.setValue(props.fieldPath, newFieldValue)
      }
    }
    
    if (response.items && Array.isArray(response.items)) {
      options.value = response.items.map((item: any) => ({
        label: item.label || String(item.value),
        value: item.value,
        disabled: false,
        displayInfo: item.display_info
      }))
    } else {
      options.value = []
    }
  } catch (error: any) {
    console.error('[SelectWidget] 回调失败', error)
    ElMessage.error(error?.message || '查询失败')
    options.value = []
  } finally {
    loading.value = false
  }
}

// 当前统计信息（从回调接口获取）
const currentStatistics = ref<Record<string, string>>({})

// 处理值变化
function handleChange(value: any): void {
  // 值变化时，保存 displayInfo 和 statistics
  const selectedOption = options.value.find(opt => opt.value === value)
  if (selectedOption) {
    const newFieldValue = {
      raw: value,
      display: selectedOption.label,
      meta: {
        displayInfo: selectedOption.displayInfo,
        statistics: currentStatistics.value  // 🔥 保存 statistics 配置
      }
    }
    
    formDataStore.setValue(props.fieldPath, newFieldValue)
    emit('update:modelValue', newFieldValue)
  }
}

// 处理聚焦
function handleFocus(): void {
  // 如果还没有选项且有回调接口，触发一次搜索
  if (options.value.length === 0 && hasCallback.value) {
    handleSearch('', false)
  }
}

// 初始化
onMounted(() => {
  initOptions()
})
</script>

<style scoped>
.select-widget {
  width: 100%;
}

.select-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.display-info {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: 8px;
}

.response-value {
  color: var(--el-text-color-regular);
}

.table-cell-value {
  display: inline-flex;
  align-items: center;
}

/* 🔥 单选组件的标签样式：使用空心样式（outline） */
.select-tag {
  font-weight: 500;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
  opacity: 0.9;
  transition: opacity 0.2s;
}

.select-tag:hover {
  opacity: 1;
}

/* 🔥 空心样式：使用边框和透明背景 */
.select-tag-outline {
  background-color: transparent !important;
  border: 2px solid currentColor !important;
}

/* 标准颜色的空心标签 */
.select-tag-outline.el-tag--success {
  color: var(--el-color-success) !important;
  border-color: var(--el-color-success) !important;
}

.select-tag-outline.el-tag--warning {
  color: var(--el-color-warning) !important;
  border-color: var(--el-color-warning) !important;
}

.select-tag-outline.el-tag--danger {
  color: var(--el-color-danger) !important;
  border-color: var(--el-color-danger) !important;
}

.select-tag-outline.el-tag--info {
  color: var(--el-color-info) !important;
  border-color: var(--el-color-info) !important;
}

.select-tag-outline.el-tag--primary {
  color: var(--el-color-primary) !important;
  border-color: var(--el-color-primary) !important;
}

/* 自定义颜色的空心标签：使用边框颜色 */
.select-tag-outline[style*="color"] {
  border-color: currentColor !important;
}

.table-cell-value .el-tag {
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

/* 自定义颜色的 tag，确保文字清晰 */
.table-cell-value .el-tag[style*="background-color"] {
  color: #fff !important;
  font-weight: 500;
}

.detail-value {
  margin-bottom: 16px;
  display: inline-flex;
  align-items: center;
}

.detail-value .el-tag {
  font-weight: 500;
  border: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

/* 自定义颜色的 tag，确保文字清晰 */
.detail-value .el-tag[style*="background-color"] {
  color: #fff !important;
  font-weight: 500;
}

/* 🔥 下拉选项中的颜色指示器样式 */
.option-color-indicator {
  display: inline-block !important;
  width: 12px !important;
  height: 12px !important;
  min-width: 12px !important;
  min-height: 12px !important;
  border-radius: 2px !important;
  flex-shrink: 0 !important;
  border: none !important;
  vertical-align: middle !important;
  /* 🔥 降低亮度：使用 filter 降低饱和度和亮度 */
  filter: brightness(0.95) saturate(0.9);
  opacity: 0.9;
}

.detail-label {
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
}

.detail-content {
  color: var(--el-text-color-regular);
}
</style>

<style>
/* 全局样式：确保下拉菜单在抽屉中正常显示 */
.select-dropdown-popper {
  z-index: 3001 !important;
}

.select-dropdown-popper .el-select-dropdown {
  z-index: 3001 !important;
}
</style>

