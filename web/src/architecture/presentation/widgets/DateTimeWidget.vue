<!--
  DateTimeWidget - 真实日期时间组件
  raw value 使用 "YYYY-MM-DD HH:mm:ss" 字符串，配合后端 datetime/time 类型字段。
-->

<template>
  <div class="datetime-widget">
    <el-time-picker
      v-if="mode === 'edit' && isTimeOnly"
      v-model="singlePickerValue"
      :disabled="widgetConfig.disabled"
      :placeholder="field.desc || `请选择${field.name}`"
      :format="format"
      :value-format="format"
      :clearable="true"
      @change="handleChange"
    />
    <el-date-picker
      v-else-if="mode === 'edit'"
      v-model="singlePickerValue"
      :disabled="widgetConfig.disabled"
      :placeholder="field.desc || `请选择${field.name}`"
      type="datetime"
      :format="format"
      :value-format="format"
      :clearable="true"
      :shortcuts="shortcuts"
      @change="handleChange"
    />

    <span v-else-if="mode === 'response'" class="response-value">
      {{ displayValue }}
    </span>

    <span v-else-if="mode === 'table-cell'" class="table-cell-value">
      {{ displayValue }}
    </span>

    <div v-else-if="mode === 'detail'" class="detail-value">
      <div class="detail-content">{{ displayValue }}</div>
    </div>

    <div v-else-if="mode === 'search' && isRangeSearch" class="datetime-range-search">
      <el-time-picker
        v-if="isTimeOnly"
        v-model="rangeStartValue"
        class="datetime-range-field"
        :format="format"
        :value-format="format"
        :clearable="true"
        :placeholder="`开始${field.name}`"
      />
      <el-date-picker
        v-else
        v-model="rangeStartValue"
        class="datetime-range-field"
        type="datetime"
        :format="format"
        :value-format="format"
        :clearable="true"
        :shortcuts="shortcuts"
        :placeholder="`开始${field.name}`"
      />
      <span class="range-separator">至</span>
      <el-time-picker
        v-if="isTimeOnly"
        v-model="rangeEndValue"
        class="datetime-range-field"
        :format="format"
        :value-format="format"
        :clearable="true"
        :placeholder="`结束${field.name}`"
      />
      <el-date-picker
        v-else
        v-model="rangeEndValue"
        class="datetime-range-field"
        type="datetime"
        :format="format"
        :value-format="format"
        :clearable="true"
        :shortcuts="shortcuts"
        :placeholder="`结束${field.name}`"
      />
    </div>
    <el-time-picker
      v-else-if="mode === 'search' && isTimeOnly"
      v-model="singlePickerValue"
      :format="format"
      :value-format="format"
      :clearable="true"
    />
    <el-date-picker
      v-else-if="mode === 'search'"
      v-model="singlePickerValue"
      type="datetime"
      :format="format"
      :value-format="format"
      :clearable="true"
      :shortcuts="shortcuts"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { ElDatePicker, ElTimePicker } from 'element-plus'
import type { WidgetComponentProps, WidgetComponentEmits } from '@/architecture/presentation/widgets/types'
import { useFormDataStore } from '@/architecture/runtime/stores/formData'
import { createFieldValue } from '@/architecture/presentation/widgets/utils/createFieldValue'
import { formatDateTimeValue } from '@/architecture/shared/date'
import type { DateTimeWidgetConfig } from '@/architecture/domain/types/widget-configs'
import { resolveWidgetSearchType } from '@/architecture/presentation/widgets/utils/searchType'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})
const emit = defineEmits<WidgetComponentEmits>()

const formDataStore = useFormDataStore()

type DateTimeSingleInput = Date | number | string | null
type DateTimeRangeInput = [DateTimeSingleInput, DateTimeSingleInput]
type DateTimeModelInput = DateTimeSingleInput | DateTimeRangeInput
type DateTimeRawRange = [string | null, string | null]
type DateTimeRawValue = string | DateTimeRawRange | null
type DateTimePickerValue = string | DateTimeRawRange | null

const widgetConfig = computed(() => {
  return (props.field.widget?.config || {}) as DateTimeWidgetConfig
})

const format = computed(() => {
  return widgetConfig.value.format || 'YYYY-MM-DD HH:mm:ss'
})

const isTimeOnly = computed(() => {
  const fmt = format.value
  return fmt === 'HH:mm' || fmt === 'HH:mm:ss'
})

const shortcuts = computed(() => {
  if (isTimeOnly.value) {
    return undefined
  }

  const now = new Date()
  return [
    { text: '现在', value: () => new Date(now) },
    {
      text: '今天',
      value: () => {
        const date = new Date(now)
        date.setHours(0, 0, 0, 0)
        return date
      }
    },
    {
      text: '明天',
      value: () => {
        const date = new Date(now)
        date.setDate(date.getDate() + 1)
        date.setHours(0, 0, 0, 0)
        return date
      }
    },
    {
      text: '昨天',
      value: () => {
        const date = new Date(now)
        date.setDate(date.getDate() - 1)
        date.setHours(0, 0, 0, 0)
        return date
      }
    }
  ]
})

const isRangeSearch = computed(() => {
  const currentSearchType = resolveWidgetSearchType(props.searchType)
  return currentSearchType.includes('gte') && currentSearchType.includes('lte')
})

function isDateTimeEmptyLike(value: unknown): boolean {
  if (value === null || value === undefined) {
    return true
  }
  if (value instanceof Date) {
    return Number.isNaN(value.getTime())
  }
  if (typeof value === 'number') {
    return Number.isNaN(value) || value === 0
  }
  if (typeof value === 'string') {
    const trimmed = value.trim()
    return trimmed === '' || trimmed === '0'
  }
  return false
}

function normalizeDateTimeItem(value: Date | number | string): string {
  const formatted = formatDateTimeValue(value, format.value)
  if (formatted === '-') {
    throw new Error(`[DateTimeWidget] 无法转换值: ${value}`)
  }
  return formatted
}

function normalizeDateTimeValue(value: DateTimeModelInput): DateTimeRawValue {
  if (value === null || value === undefined) {
    return null
  }

  if (Array.isArray(value)) {
    const normalizedRange = value.map((item): string | null => {
      if (isDateTimeEmptyLike(item)) {
        return null
      }
      return normalizeDateTimeItem(item as Date | number | string)
    }) as DateTimeRawRange

    if (normalizedRange[0] === null && normalizedRange[1] === null) {
      return null
    }

    return normalizedRange
  }

  if (isDateTimeEmptyLike(value)) {
    return null
  }

  return normalizeDateTimeItem(value as Date | number | string)
}

function isSameDateTimeRawValue(left: unknown, right: DateTimeRawValue): boolean {
  return JSON.stringify(left ?? null) === JSON.stringify(right ?? null)
}

function toPickerValue(value: DateTimeSingleInput): string | null {
  if (isDateTimeEmptyLike(value)) {
    return null
  }
  return normalizeDateTimeItem(value as Date | number | string)
}

const internalValue = computed<DateTimePickerValue>({
  get: () => {
    if (props.mode !== 'edit' && props.mode !== 'search') {
      return null
    }

    const value = props.value?.raw
    if (isDateTimeEmptyLike(value)) {
      return null
    }

    if (Array.isArray(value)) {
      const [startValue, endValue] = value
      return [
        toPickerValue(startValue as DateTimeSingleInput),
        toPickerValue(endValue as DateTimeSingleInput)
      ] as DateTimeRawRange
    }

    return toPickerValue(value as DateTimeSingleInput)
  },
  set: (newValue: DateTimeModelInput) => {
    commitDateTimeValue(newValue)
  }
})

const singlePickerValue = computed<DateTimeSingleInput>({
  get: () => {
    const currentValue = internalValue.value
    return Array.isArray(currentValue) ? (currentValue[0] ?? null) : currentValue
  },
  set: (newValue) => {
    commitDateTimeValue(newValue)
  }
})

const rangeStartValue = computed<DateTimeSingleInput>({
  get: () => {
    const currentValue = internalValue.value
    return Array.isArray(currentValue) ? (currentValue[0] ?? null) : null
  },
  set: (newValue) => {
    const currentValue = internalValue.value
    const endValue = Array.isArray(currentValue) ? (currentValue[1] ?? null) : null
    commitDateTimeValue(buildRangeValue(newValue, endValue))
  }
})

const rangeEndValue = computed<DateTimeSingleInput>({
  get: () => {
    const currentValue = internalValue.value
    return Array.isArray(currentValue) ? (currentValue[1] ?? null) : null
  },
  set: (newValue) => {
    const currentValue = internalValue.value
    const startValue = Array.isArray(currentValue) ? (currentValue[0] ?? null) : null
    commitDateTimeValue(buildRangeValue(startValue, newValue))
  }
})

const displayValue = computed(() => {
  const raw = props.value?.raw
  if (isDateTimeEmptyLike(raw)) {
    return '-'
  }

  return formatDateTimeDisplay(raw as DateTimeRawValue)
})

function handleChange(_value: DateTimeModelInput): void {
  // v-model setter already commits the value.
}

function formatDateTimeDisplay(value: DateTimeRawValue): string {
  if (value === null) {
    return ''
  }

  if (Array.isArray(value)) {
    const [startValue, endValue] = value
    const startDisplay = isDateTimeEmptyLike(startValue) ? '' : formatDateTimeValue(startValue, format.value)
    const endDisplay = isDateTimeEmptyLike(endValue) ? '' : formatDateTimeValue(endValue, format.value)

    if (startDisplay && endDisplay) {
      return `${startDisplay} 至 ${endDisplay}`
    }

    return startDisplay || endDisplay
  }

  if (isDateTimeEmptyLike(value)) {
    return ''
  }

  return formatDateTimeValue(value, format.value)
}

function commitDateTimeValue(newValue: DateTimeModelInput): void {
  if (props.mode !== 'edit' && props.mode !== 'search') {
    return
  }

  const rawValue = normalizeDateTimeValue(newValue)
  const newFieldValue = createFieldValue(
    props.field,
    rawValue,
    formatDateTimeDisplay(rawValue)
  )

  if (props.mode === 'edit') {
    formDataStore.setValue(props.fieldPath, newFieldValue)
  }

  emit('update:modelValue', newFieldValue)
}

function buildRangeValue(
  startValue: DateTimeSingleInput,
  endValue: DateTimeSingleInput
): DateTimeRangeInput | null {
  if (isDateTimeEmptyLike(startValue) && isDateTimeEmptyLike(endValue)) {
    return null
  }

  return [startValue ?? null, endValue ?? null]
}

watch(
  () => props.value?.raw,
  (rawValue) => {
    if (props.mode !== 'edit') {
      return
    }

    const normalizedRawValue = normalizeDateTimeValue(rawValue as DateTimeModelInput)
    if (isSameDateTimeRawValue(rawValue, normalizedRawValue)) {
      return
    }

    const normalizedFieldValue = createFieldValue(
      props.field,
      normalizedRawValue,
      formatDateTimeDisplay(normalizedRawValue),
      props.value?.meta
    )

    formDataStore.setValue(props.fieldPath, normalizedFieldValue)
    emit('update:modelValue', normalizedFieldValue)
  },
  { immediate: true }
)
</script>

<style scoped>
.datetime-widget {
  width: 100%;
}

.datetime-range-search {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-width: 0;
}

.datetime-range-field {
  flex: 1 1 0;
  min-width: 0;
}

.range-separator {
  color: var(--el-text-color-secondary);
  font-size: 14px;
  flex-shrink: 0;
}

.response-value,
.table-cell-value,
.detail-content {
  color: var(--el-text-color-regular);
}

.detail-value {
  margin-bottom: 16px;
}
</style>
