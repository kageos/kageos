<!--
  ColorWidget - 颜色选择器组件
  🔥 完全新增，不依赖旧代码
  
  功能：
  - 编辑模式：显示为颜色选择器（el-color-picker）
  - 响应模式：显示颜色值和颜色块
  - 表格单元格模式：显示颜色块和值
  - 详情模式：显示颜色块和值
  - 搜索模式：颜色值输入（文本输入框）
-->

<template>
  <div class="color-widget">
    <!-- 编辑模式：颜色选择器 -->
    <el-color-picker
      v-if="mode === 'edit'"
      v-model="internalValue"
      :color-format="colorFormat"
      :show-alpha="showAlpha"
      :disabled="field.widget?.config?.disabled"
      @change="handleChange"
    />
    
    <!-- 响应模式（只读） -->
    <div v-else-if="mode === 'response'" class="response-value">
      <span class="color-block" :style="{ backgroundColor: colorValue }"></span>
      <span class="color-text">{{ displayValue }}</span>
    </div>
    
    <!-- 表格单元格模式：显示颜色块和值 -->
    <div v-else-if="mode === 'table-cell'" class="table-cell-value">
      <span class="color-block" :style="{ backgroundColor: colorValue }"></span>
      <span class="color-text">{{ displayValue }}</span>
    </div>
    
    <!-- 详情模式：显示颜色块和值 -->
    <div v-else-if="mode === 'detail'" class="detail-value">
      <span class="color-block" :style="{ backgroundColor: colorValue }"></span>
      <span class="color-text">{{ displayValue }}</span>
    </div>
    
    <!-- 搜索模式：颜色值输入 -->
    <el-input
      v-else-if="mode === 'search'"
      v-model="searchValue"
      :placeholder="`搜索${field.name}`"
      :clearable="true"
      @input="handleSearchChange"
      @clear="handleSearchClear"
    >
      <template #prefix>
        <el-color-picker
          v-model="searchValue"
          :color-format="colorFormat"
          :show-alpha="showAlpha"
          size="small"
          @change="handleSearchChange"
        />
      </template>
    </el-input>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElColorPicker, ElInput } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { ColorWidgetConfig } from '@/core/types/widget-configs'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

// 获取配置（带类型）
const config = computed(() => {
  return (props.field.widget?.config || {}) as ColorWidgetConfig
})

// 颜色格式、是否显示透明度
const colorFormat = computed(() => {
  const format = config.value.format
  if (format === 'hex' || format === 'rgb' || format === 'rgba') {
    return format
  }
  return 'hex' // 默认hex格式
})

const showAlpha = computed(() => {
  return config.value.show_alpha === true || config.value.show_alpha === 'true'
})

// 默认值
const defaultValue = computed(() => {
  const def = config.value.default
  if (def && typeof def === 'string') {
    return def
  }
  return undefined
})

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit') {
      const value = props.value?.raw
      if (value !== null && value !== undefined && value !== '') {
        return String(value)
      }
      // 如果没有值且有默认值，返回默认值
      if (defaultValue.value !== undefined) {
        return defaultValue.value
      }
      return '#409EFF' // 默认颜色
    }
    return undefined
  },
  set: (newValue: string | null) => {
    if (props.mode === 'edit') {
      const value = newValue ?? null
      const display = value !== null ? String(value) : ''
      // 🔥 使用工具函数创建 FieldValue，确保包含 dataType 和 widgetType
      const newFieldValue = createFieldValue(
        props.field,
        value,
        display
      )
      
      formDataStore.setValue(props.fieldPath, newFieldValue)
      emit('update:modelValue', newFieldValue)
    }
  }
})

// 颜色值（用于显示）
const colorValue = computed(() => {
  const value = props.value?.raw
  if (value === null || value === undefined || value === '') {
    return 'transparent'
  }
  const strValue = String(value)
  // 验证是否为有效的颜色值
  if (isValidColor(strValue)) {
    return strValue
  }
  return 'transparent'
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

/**
 * 验证颜色值是否有效
 */
function isValidColor(color: string): boolean {
  if (!color) return false
  
  // 验证 hex 格式 (#RRGGBB 或 #RRGGBBAA)
  if (/^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3}|[A-Fa-f0-9]{8})$/.test(color)) {
    return true
  }
  
  // 验证 rgb/rgba 格式
  if (/^rgba?\(/.test(color)) {
    return true
  }
  
  // 验证颜色名称（如 red, blue 等）
  const colorNames = ['red', 'blue', 'green', 'yellow', 'orange', 'purple', 'pink', 'black', 'white', 'gray', 'grey']
  if (colorNames.includes(color.toLowerCase())) {
    return true
  }
  
  return false
}

/**
 * 处理编辑模式的值变化
 */
function handleChange(value: string | null): void {
  // 值变化已在 internalValue 的 setter 中处理
}

/**
 * 搜索模式：搜索值
 */
const searchValue = ref<string>('')

/**
 * 处理搜索模式的值变化
 */
function handleSearchChange(): void {
  const value = searchValue.value?.trim() || null
  const newFieldValue = value ? {
    raw: value,
    display: value,
    meta: {}
  } : null
  
  formDataStore.setValue(props.fieldPath, newFieldValue)
  emit('update:modelValue', newFieldValue)
}

/**
 * 处理搜索模式清空
 */
function handleSearchClear(): void {
  searchValue.value = ''
  handleSearchChange()
}

/**
 * 监听值变化，处理初始化和值恢复
 * 
 * ⚠️ 关键逻辑：
 * 1. 编辑模式：如果字段没有值，使用默认值
 * 2. 搜索模式：从 value.raw 中恢复搜索值（用于 URL 恢复）
 * 
 * 注意：使用 deep: true 确保能监听到对象内部的变化
 */
watch(
  () => props.value,
  (newValue: any) => {
    if (props.mode === 'edit') {
      if (!newValue || newValue.raw === null || newValue.raw === undefined || newValue.raw === '') {
        // 编辑模式：如果字段没有值，使用默认值
        if (defaultValue.value !== undefined) {
          internalValue.value = defaultValue.value
        }
      } else {
        // 🔥 关键：如果值存在，确保它能正确显示
        const strValue = String(newValue.raw)
        if (internalValue.value !== strValue) {
          internalValue.value = strValue
        }
      }
    } else if (props.mode === 'search') {
      // 搜索模式：从 value.raw 中恢复搜索值
      if (newValue?.raw) {
        searchValue.value = String(newValue.raw)
      } else {
        searchValue.value = ''
      }
    }
  },
  { immediate: true, deep: true }
)
</script>

<style scoped>
.color-widget {
  width: 100%;
}

.response-value {
  display: flex;
  align-items: center;
  gap: 8px;
}

.table-cell-value {
  display: flex;
  align-items: center;
  gap: 8px;
}

.detail-value {
  display: flex;
  align-items: center;
  gap: 8px;
}

.color-block {
  display: inline-block;
  width: 20px;
  height: 20px;
  border-radius: 4px;
  border: 1px solid var(--el-border-color);
  flex-shrink: 0;
}

.color-text {
  color: var(--el-text-color-regular);
  font-size: 14px;
}
</style>

