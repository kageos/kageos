<template>
  <div class="search-input">
    <!-- 🔥 精确搜索 / 模糊搜索 -->
    <el-input
      v-if="inputConfig.component === 'ElInput'"
      v-model="localValue"
      :placeholder="inputConfig.props?.placeholder"
      :clearable="inputConfig.props?.clearable"
      :disabled="inputConfig.props?.disabled"
      :style="inputConfig.props?.style"
      @input="handleInput"
    />

    <!-- 🔥 下拉选择 -->
    <el-select
      v-else-if="inputConfig.component === 'ElSelect'"
      v-model="localValue"
      :placeholder="inputConfig.props?.placeholder"
      :clearable="inputConfig.props?.clearable"
      :style="inputConfig.props?.style"
      @change="handleInput"
    >
      <el-option
        v-for="option in selectOptions"
        :key="typeof option === 'object' ? option.value : option"
        :label="typeof option === 'object' ? option.label : option"
        :value="typeof option === 'object' ? option.value : option"
      />
    </el-select>

    <!-- 🔥 数字范围输入 -->
    <div v-else-if="inputConfig.component === 'NumberRangeInput'" class="number-range">
      <el-input-number
        v-model="rangeValue.min"
        :placeholder="inputConfig.props?.minPlaceholder"
        :precision="inputConfig.props?.precision"
        :step="inputConfig.props?.step"
        :min="inputConfig.props?.min"
        :max="inputConfig.props?.max"
        :clearable="true"
        :controls-position="'right'"
        :style="{ width: '160px' }"
        @change="handleRangeChange"
      />
      <span class="range-separator">至</span>
      <el-input-number
        v-model="rangeValue.max"
        :placeholder="inputConfig.props?.maxPlaceholder"
        :precision="inputConfig.props?.precision"
        :step="inputConfig.props?.step"
        :min="inputConfig.props?.min"
        :max="inputConfig.props?.max"
        :clearable="true"
        :controls-position="'right'"
        :style="{ width: '160px' }"
        @change="handleRangeChange"
      />
    </div>

    <!-- 🔥 日期范围选择 -->
    <el-date-picker
      v-else-if="inputConfig.component === 'ElDatePicker'"
      v-model="localValue"
      :type="inputConfig.props?.type"
      :range-separator="inputConfig.props?.rangeSeparator"
      :start-placeholder="inputConfig.props?.startPlaceholder"
      :end-placeholder="inputConfig.props?.endPlaceholder"
      :format="inputConfig.props?.format"
      :value-format="inputConfig.props?.valueFormat"
      :shortcuts="inputConfig.props?.shortcuts"
      :clearable="inputConfig.props?.clearable"
      :style="inputConfig.props?.style"
      @change="handleInput"
    />

    <!-- 🔥 文本范围输入（默认降级） -->
    <div v-else-if="inputConfig.component === 'RangeInput'" class="text-range">
      <el-input
        v-model="rangeValue.min"
        :placeholder="inputConfig.props?.minPlaceholder"
        clearable
        style="width: 160px"
        @input="handleRangeChange"
      />
      <span class="range-separator">至</span>
      <el-input
        v-model="rangeValue.max"
        :placeholder="inputConfig.props?.maxPlaceholder"
        clearable
        style="width: 160px"
        @input="handleRangeChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { WidgetBuilder } from '@/core/factories/WidgetBuilder'
import { ErrorHandler } from '@/core/utils/ErrorHandler'
import type { FieldConfig } from '@/types'

interface Props {
  field: FieldConfig
  searchType: string
  modelValue: any
}

interface Emits {
  (e: 'update:modelValue', value: any): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 本地值（单值）
const localValue = ref(props.modelValue)

// 范围值（最小值、最大值）
const rangeValue = ref({
  min: undefined as any,
  max: undefined as any
})

// 初始化范围值
if (props.searchType?.includes('gte') && props.searchType?.includes('lte')) {
  if (props.modelValue) {
    rangeValue.value = props.modelValue
  }
}

// 🔥 提取下拉选项
const selectOptions = computed(() => {
  if (inputConfig.value.component !== 'ElSelect') {
    return []
  }
  return inputConfig.value.props?.options || []
})

// 🔥 通过 Widget 获取搜索输入配置
const inputConfig = computed(() => {
  try {
    // ✅ 使用 WidgetBuilder 创建临时 Widget（formManager 为 null）
    const tempWidget = WidgetBuilder.createTemporary({
      field: props.field
    })
    
    // 🔥 调用 Widget 的 renderSearchInput 方法
    return (tempWidget as any).renderSearchInput(props.searchType)
  } catch (error) {
    // ✅ 使用 ErrorHandler 统一处理错误
    return ErrorHandler.handleWidgetError('SearchInput.inputConfig', error, {
      showMessage: false,
      fallbackValue: {
        component: 'ElInput',
        props: {
          placeholder: `请输入${props.field.name}`,
          clearable: true,
          style: { width: '200px' }
        }
      }
    })
  }
})

// 处理单值输入
const handleInput = (value: any) => {
  localValue.value = value
  emit('update:modelValue', value)
}

// 处理范围输入
const handleRangeChange = () => {
  emit('update:modelValue', {
    min: rangeValue.value.min,
    max: rangeValue.value.max
  })
}

// 监听外部值变化
watch(() => props.modelValue, (newValue) => {
  if (props.searchType?.includes('gte') && props.searchType?.includes('lte')) {
    rangeValue.value = newValue || { min: undefined, max: undefined }
  } else {
    localValue.value = newValue
  }
})
</script>

<style scoped>
.search-input {
  display: inline-flex;
  align-items: center;
}

.number-range,
.text-range {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.range-separator {
  color: var(--el-text-color-secondary);
  font-size: 14px;
}
</style>

