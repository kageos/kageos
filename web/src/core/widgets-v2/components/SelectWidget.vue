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
          <span>{{ option.label }}</span>
          <span v-if="option.displayInfo" class="display-info">{{ option.displayInfo }}</span>
        </div>
      </el-option>
    </el-select>
    
    <!-- 响应模式（只读） -->
    <span v-else-if="mode === 'response'" class="response-value">
      {{ displayValue }}
    </span>
    
    <!-- 表格单元格模式 -->
    <span v-else-if="mode === 'table-cell'" class="table-cell-value">
      {{ displayValue }}
    </span>
    
    <!-- 详情模式 -->
    <div v-else-if="mode === 'detail'" class="detail-value">
      <div class="detail-label">{{ field.name }}</div>
      <div class="detail-content">{{ displayValue }}</div>
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
      />
    </el-select>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElSelect, ElOption, ElMessage } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'
import { selectFuzzy } from '@/api/function'

const props = defineProps<WidgetComponentProps>()
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 选项列表
const options = ref<Array<{ label: string; value: any; disabled?: boolean; displayInfo?: string }>>([])

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

// 处理值变化
function handleChange(value: any): void {
  // 值变化时，保存 displayInfo
  const selectedOption = options.value.find(opt => opt.value === value)
  if (selectedOption) {
    const newFieldValue = {
      raw: value,
      display: selectedOption.label,
      meta: {
        displayInfo: selectedOption.displayInfo
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
  color: var(--el-text-color-regular);
}

.detail-value {
  margin-bottom: 16px;
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

