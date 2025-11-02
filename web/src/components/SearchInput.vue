<template>
  <div class="search-input">
    <!-- 🔥 精确搜索 / 模糊搜索 -->
    <el-input
      v-if="inputConfig.component === 'ElInput'"
      v-model="localValue"
      v-bind="inputConfig.props"
      @input="handleInput"
    />

    <!-- 🔥 下拉选择 -->
    <el-select
      v-else-if="inputConfig.component === 'ElSelect'"
      v-model="localValue"
      v-bind="inputConfig.props"
      @change="handleInput"
    >
      <el-option
        v-for="option in inputConfig.props?.options || []"
        :key="option"
        :label="option"
        :value="option"
      />
    </el-select>

    <!-- 🔥 数字范围输入 -->
    <div v-else-if="inputConfig.component === 'NumberRangeInput'" class="number-range">
      <el-input-number
        v-model="rangeValue.min"
        v-bind="getRangeProps('min')"
        @change="handleRangeChange"
        controls-position="right"
      />
      <span class="range-separator">至</span>
      <el-input-number
        v-model="rangeValue.max"
        v-bind="getRangeProps('max')"
        @change="handleRangeChange"
        controls-position="right"
      />
    </div>

    <!-- 🔥 日期范围选择 -->
    <el-date-picker
      v-else-if="inputConfig.component === 'ElDatePicker'"
      v-model="localValue"
      v-bind="inputConfig.props"
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
import { widgetFactory } from '@/core/factories/WidgetFactory'
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

// 🔥 通过 Widget 获取搜索输入配置
const inputConfig = computed(() => {
  try {
    // 创建临时 Widget 实例
    const WidgetClass = widgetFactory.getWidgetClass(props.field.widget?.type || 'input')
    
    // 注意：这里不需要完整的 formManager 等，只是为了调用 renderSearchInput
    const tempWidget = new WidgetClass({
      field: props.field,
      fieldPath: `_search_.${props.field.code}`,
      initialValue: { raw: null, display: '', meta: {} },
      formManager: null as any,  // 搜索不需要 formManager
      formRenderer: null,
      depth: 0,
      onChange: () => {}
    })
    
    // 🔥 调用 Widget 的 renderSearchInput 方法
    return tempWidget.renderSearchInput(props.searchType)
  } catch (error) {
    console.error('[SearchInput] 获取配置失败:', error)
    // 降级：返回默认输入框
    return {
      component: 'ElInput',
      props: {
        placeholder: `请输入${props.field.name}`,
        clearable: true,
        style: { width: '200px' }
      }
    }
  }
})

// 获取范围输入的 props
const getRangeProps = (type: 'min' | 'max') => {
  const baseProps = {
    placeholder: type === 'min' ? inputConfig.value.props?.minPlaceholder : inputConfig.value.props?.maxPlaceholder,
    clearable: true,
    style: { width: '160px' }
  }
  
  if (inputConfig.value.component === 'NumberRangeInput') {
    return {
      ...baseProps,
      precision: inputConfig.value.props?.precision,
      step: inputConfig.value.props?.step,
      min: inputConfig.value.props?.min,
      max: inputConfig.value.props?.max
    }
  }
  
  return baseProps
}

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

