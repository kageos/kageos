<!--
  RateWidget - 评分组件
  🔥 统一架构组件
  
  功能：
  - 编辑模式：显示为星级评分（el-rate）
  - 响应模式：显示评分值
  - 表格单元格模式：显示评分值
  - 详情模式：显示评分值
  - 搜索模式：范围搜索（最小值、最大值）
-->

<template>
  <div class="rate-widget">
    <!-- 编辑模式：星级评分 -->
    <el-rate
      v-if="mode === 'edit'"
      class="rate-widget-control rate-widget-control--edit"
      v-model="internalValue"
      :max="max"
      :allow-half="allowHalf"
      :show-text="showText"
      :texts="texts"
      :colors="rateColors"
      :void-color="rateVoidColor"
      :disabled-void-color="rateDisabledVoidColor"
      :text-color="rateTextColor"
      size="large"
      :disabled="field.widget?.config?.disabled"
      @change="handleChange"
    />
    
    <!-- 响应模式（只读） -->
    <div v-else-if="mode === 'response'" class="response-value">
      <!-- 缩放样式：使用圆点 -->
      <div v-if="useScaledStyle" class="rate-scaled">
        <div class="rate-dots">
          <span
            v-for="i in max"
            :key="i"
            class="rate-dot"
            :class="{ 'rate-dot-filled': i <= rateValue, 'rate-dot-half': allowHalf && i - 0.5 <= rateValue && rateValue < i }"
          />
        </div>
        <span class="rate-score-text">{{ rateValue }}/{{ max }}</span>
        <span v-if="texts && texts.length > 0" class="rate-text-label">{{ getTextLabel(rateValue) }}</span>
      </div>
      <!-- 正常样式：使用星星 -->
      <el-rate
        v-else
        class="rate-widget-control rate-widget-control--response"
        :model-value="rateValue"
        :max="max"
        :allow-half="allowHalf"
        disabled
        :show-score="true"
        :score-template="scoreTemplate"
        :texts="texts"
        :colors="rateColors"
        :void-color="rateVoidColor"
        :disabled-void-color="rateDisabledVoidColor"
        :text-color="rateTextColor"
        size="large"
      />
    </div>
    
    <!-- 表格单元格模式：显示评分 -->
    <div v-else-if="mode === 'table-cell'" class="table-cell-value">
      <!-- 缩放样式：使用圆点 -->
      <div v-if="useScaledStyle" class="rate-scaled">
        <div class="rate-dots">
          <span
            v-for="i in max"
            :key="i"
            class="rate-dot"
            :class="{ 'rate-dot-filled': i <= rateValue, 'rate-dot-half': allowHalf && i - 0.5 <= rateValue && rateValue < i }"
          />
        </div>
        <span class="rate-score-text">{{ rateValue }}/{{ max }}</span>
      </div>
      <!-- 正常样式：使用星星 -->
      <el-rate
        v-else
        class="rate-widget-control rate-widget-control--table"
        :model-value="rateValue"
        :max="max"
        :allow-half="allowHalf"
        disabled
        :show-score="true"
        :score-template="scoreTemplate"
        :colors="rateColors"
        :void-color="rateVoidColor"
        :disabled-void-color="rateDisabledVoidColor"
        :text-color="rateTextColor"
      />
    </div>
    
    <!-- 详情模式：显示评分 -->
    <div v-else-if="mode === 'detail'" class="detail-value">
      <!-- 缩放样式：使用圆点 -->
      <div v-if="useScaledStyle" class="rate-scaled">
        <div class="rate-dots">
          <span
            v-for="i in max"
            :key="i"
            class="rate-dot"
            :class="{ 'rate-dot-filled': i <= rateValue, 'rate-dot-half': allowHalf && i - 0.5 <= rateValue && rateValue < i }"
          />
        </div>
        <span class="rate-score-text">{{ rateValue }}/{{ max }}</span>
        <span v-if="texts && texts.length > 0" class="rate-text-label">{{ getTextLabel(rateValue) }}</span>
      </div>
      <!-- 正常样式：使用星星 -->
      <el-rate
        v-else
        class="rate-widget-control rate-widget-control--detail"
        :model-value="rateValue"
        :max="max"
        :allow-half="allowHalf"
        disabled
        :show-score="true"
        :score-template="scoreTemplate"
        :texts="texts"
        :colors="rateColors"
        :void-color="rateVoidColor"
        :disabled-void-color="rateDisabledVoidColor"
        :text-color="rateTextColor"
        size="large"
      />
    </div>
    
    <!-- 搜索模式：范围输入 -->
    <div v-else-if="mode === 'search'" class="rate-search">
      <el-input-number
        v-model="minValue"
        :min="0"
        :max="max"
        :step="allowHalf ? 0.5 : 1"
        :precision="allowHalf ? 1 : 0"
        :placeholder="`最小${field.name}`"
        @change="handleSearchChange"
      />
      <span class="separator">-</span>
      <el-input-number
        v-model="maxValue"
        :min="0"
        :max="max"
        :step="allowHalf ? 0.5 : 1"
        :precision="allowHalf ? 1 : 0"
        :placeholder="`最大${field.name}`"
        @change="handleSearchChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElRate, ElInputNumber } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/architecture/runtime/stores/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import type { RateWidgetConfig } from '@/architecture/domain/types/widget-configs'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()
const rateColors = ['#f7ba2a', '#f7ba2a', '#f7ba2a']
const rateVoidColor = '#d8dee8'
const rateDisabledVoidColor = '#e5e7eb'
const rateTextColor = '#995c00'

// 获取配置（带类型）
const config = computed(() => {
  return (props.field.widget?.config || {}) as RateWidgetConfig
})

// 最大星级、是否允许半星、是否显示文字、自定义文字
const max = computed(() => {
  const maxValue = config.value.max
  if (maxValue !== undefined && maxValue !== null) {
    const num = Number(maxValue)
    if (isNaN(num) || num <= 0) {
      return 5 // 默认5星
    }
    return Math.floor(num)
  }
  return 5 // 默认5星
})

// 判断是否需要使用缩放样式（圆点/方块）
// max > 10 时使用圆点样式，否则使用星星样式
// 注意：max=10 时仍然使用星星样式，因为 10 个星星还是可以接受的
const useScaledStyle = computed(() => {
  return max.value > 10
})

const allowHalf = computed(() => {
  return config.value.allow_half === true || config.value.allow_half === 'true'
})

const texts = computed(() => {
  const textsValue = config.value.texts
  if (Array.isArray(textsValue) && textsValue.length > 0) {
    return textsValue
  }
  return undefined
})

// 🔥 简化逻辑：如果配置了 texts，就显示文字；否则不显示
const showText = computed(() => {
  return texts.value !== undefined && texts.value.length > 0
})

// 默认值
const defaultValue = computed(() => {
  const def = config.value.render_default
  if (def !== undefined && def !== null) {
    const num = Number(def)
    return isNaN(num) ? undefined : num
  }
  return undefined
})

// 内部值（用于 v-model）
const internalValue = computed({
  get: () => {
    if (props.mode === 'edit') {
      const value = props.value?.raw
      if (value !== null && value !== undefined && value !== '') {
        const numValue = Number(value)
        // 🔥 关键：如果转换失败，使用默认值或0
        if (!isNaN(numValue) && isFinite(numValue)) {
          // 确保值在 0 和 max 范围内
          const clampedValue = Math.max(0, Math.min(max.value, numValue))
          return clampedValue
        }
      }
      // 如果没有值且有默认值，返回默认值
      if (defaultValue.value !== undefined) {
        return defaultValue.value
      }
      return 0 // 默认返回0
    }
    return undefined
  },
  set: (newValue: number | undefined) => {
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

// 评分值（用于只读显示）
const rateValue = computed(() => {
  const value = props.value?.raw
  if (value === null || value === undefined || value === '') {
    return 0
  }
  const numValue = Number(value)
  if (isNaN(numValue) || !isFinite(numValue)) {
    return 0
  }
  return Math.max(0, Math.min(max.value, numValue))
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
  
  const numValue = Number(raw)
  if (isNaN(numValue)) {
    return String(raw)
  }
  
  return String(numValue)
})

// 评分模板（用于 show-score）
const scoreTemplate = computed(() => {
  return '{value} 分'
})

// 获取文字标签（用于 texts 配置）
function getTextLabel(value: number): string {
  if (!texts.value || texts.value.length === 0) {
    return ''
  }
  const index = Math.floor(value) - 1
  if (index >= 0 && index < texts.value.length) {
    return texts.value[index] ?? ''
  }
  return ''
}

/**
 * 处理编辑模式的值变化
 */
function handleChange(value: number): void {
  // 值变化已在 internalValue 的 setter 中处理
}

/**
 * 搜索模式：最小值、最大值
 */
const minValue = ref<number | undefined>(undefined)
const maxValue = ref<number | undefined>(undefined)

/**
 * 处理搜索模式的值变化
 * 
 * ⚠️ 关键：将 min/max 值转换为 { min, max } 对象格式
 * 这个对象会被传递给父组件，最终转换为 URL 参数：gte=field:min&lte=field:max
 * 
 * 如果 min 和 max 都为空，传递 null（表示清空搜索条件）
 */
function handleSearchChange(): void {
  const searchValue: any = {}
  if (minValue.value !== undefined && minValue.value !== null) {
    searchValue.min = minValue.value
  }
  if (maxValue.value !== undefined && maxValue.value !== null) {
    searchValue.max = maxValue.value
  }
  
  const hasValue = Object.keys(searchValue).length > 0
  const newFieldValue = hasValue ? {
    raw: searchValue,
    display: '',
    meta: {}
  } : {
    raw: null,
    display: '',
    meta: {}
  }
  
  formDataStore.setValue(props.fieldPath, newFieldValue)
  emit('update:modelValue', newFieldValue)
}

/**
 * 监听值变化，处理初始化和值恢复
 * 
 * ⚠️ 关键逻辑：
 * 1. 编辑模式：如果字段没有值，使用默认值
 * 2. 搜索模式：从 value.raw 中恢复 min/max（用于 URL 恢复）
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
        const numValue = Number(newValue.raw)
        if (!isNaN(numValue) && isFinite(numValue)) {
          // 确保值在范围内
          const clampedValue = Math.max(0, Math.min(max.value, numValue))
          // 只有当值发生变化时才更新，避免无限循环
          if (internalValue.value !== clampedValue) {
            internalValue.value = clampedValue
          }
        } else {
          // 如果值转换失败，使用默认值或0
          if (defaultValue.value !== undefined) {
            internalValue.value = defaultValue.value
          } else {
            internalValue.value = 0
          }
        }
      }
    } else if (props.mode === 'search') {
      // 搜索模式：从 value.raw 中恢复 min/max
      // ⚠️ 重要：只有当 newValue.raw 是对象时才恢复，避免其他类型的数据影响
      if (newValue?.raw && typeof newValue.raw === 'object' && !Array.isArray(newValue.raw)) {
        minValue.value = newValue.raw.min
        maxValue.value = newValue.raw.max
      } else {
        minValue.value = undefined
        maxValue.value = undefined
      }
    }
  },
  { immediate: true, deep: true }
)
</script>

<style scoped>
.rate-widget {
  width: 100%;
  --rate-active-color: #f7ba2a;
  --rate-void-color: #d8dee8;
  --rate-text-color: #995c00;
}

.rate-widget :deep(.rate-widget-control) {
  --el-rate-fill-color: var(--rate-active-color);
  --el-rate-void-color: var(--rate-void-color);
  --el-rate-disabled-void-color: #e5e7eb;
}

.rate-widget :deep(.rate-widget-control--edit) {
  min-height: 36px;
  display: inline-flex;
  align-items: center;
}

.rate-widget :deep(.rate-widget-control--edit .el-rate__item) {
  width: 30px;
  height: 30px;
  margin-right: 2px;
}

.rate-widget :deep(.rate-widget-control--edit .el-rate__icon) {
  font-size: 26px;
  color: var(--rate-active-color);
}

.rate-widget :deep(.rate-widget-control--edit .el-rate__text) {
  margin-left: 10px;
  font-size: 14px;
  color: var(--rate-text-color);
  font-weight: 600;
}

.response-value {
  color: var(--el-text-color-regular);
}

.table-cell-value {
  display: flex;
  align-items: center;
  gap: 2px;
  min-width: 0;
  overflow: visible;
  width: 100%;
}

.table-cell-value :deep(.el-rate) {
  font-size: 12px;
  line-height: 1;
  flex-shrink: 0;
}

.table-cell-value :deep(.el-rate__item) {
  margin-right: 0;
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.table-cell-value :deep(.el-rate__icon) {
  font-size: 12px;
  color: var(--rate-active-color);
}

.table-cell-value :deep(.el-rate__text) {
  font-size: 11px;
  margin-left: 4px;
  line-height: 1;
  flex-shrink: 0;
  white-space: nowrap;
}

.rate-text {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-left: 4px;
}

.detail-value {
  display: flex;
  align-items: center;
}

.response-value {
  display: flex;
  align-items: center;
}

.response-value :deep(.el-rate),
.detail-value :deep(.el-rate) {
  font-size: 20px;
  min-height: 32px;
  display: inline-flex;
  align-items: center;
}

.response-value :deep(.el-rate__item),
.detail-value :deep(.el-rate__item) {
  width: 24px;
  height: 24px;
  margin-right: 2px;
}

.response-value :deep(.el-rate__icon),
.detail-value :deep(.el-rate__icon) {
  font-size: 22px;
  color: var(--rate-active-color);
}

.response-value :deep(.el-rate__text),
.detail-value :deep(.el-rate__text) {
  font-size: 14px;
  margin-left: 10px;
  color: var(--rate-text-color);
  font-weight: 600;
}

/* 缩放样式：圆点显示 */
.rate-scaled {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  flex-shrink: 0;
}

.rate-dots {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-wrap: nowrap;
  min-width: 0;
}

.rate-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: var(--el-border-color-lighter);
  border: 1px solid var(--el-border-color);
  transition: all 0.2s;
}

.rate-dot-filled {
  background-color: var(--rate-active-color);
  border-color: var(--rate-active-color);
}

.rate-dot-half {
  background: linear-gradient(to right, var(--rate-active-color) 50%, var(--el-border-color-lighter) 50%);
  border-color: var(--rate-active-color);
}

.rate-score-text {
  font-size: 12px;
  color: var(--el-text-color-regular);
  font-weight: 500;
  white-space: nowrap;
}

.rate-text-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-left: 4px;
}

/* 表格单元格模式下的缩放样式 */
.table-cell-value .rate-scaled {
  gap: 4px;
  flex-wrap: nowrap;
}

.table-cell-value .rate-dots {
  flex-shrink: 0;
  gap: 1px;
}

.table-cell-value .rate-dot {
  width: 4px;
  height: 4px;
  flex-shrink: 0;
}

.table-cell-value .rate-score-text {
  font-size: 11px;
  flex-shrink: 0;
  white-space: nowrap;
}

.table-cell-value .rate-text-label {
  font-size: 11px;
  flex-shrink: 0;
}

.rate-search {
  display: flex;
  align-items: center;
  gap: 8px;
}

.separator {
  color: var(--el-text-color-secondary);
  font-size: 14px;
}
</style>
