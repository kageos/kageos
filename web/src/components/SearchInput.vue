<template>
  <div class="search-input">
    <!-- 🔥 用户搜索组件（自定义组件） -->
    <UserSearchInput
      v-if="inputConfig.component === 'UserSearchInput'"
      v-model="localValue"
      :placeholder="inputConfig.props?.placeholder"
      :multiple="inputConfig.props?.multiple"
      @update:modelValue="handleInput"
    />

    <!-- 🔥 精确搜索 / 模糊搜索 -->
    <el-input
      v-else-if="inputConfig.component === 'ElInput'"
      v-model="localValue"
      :placeholder="inputConfig.props?.placeholder"
      :clearable="inputConfig.props?.clearable"
      :disabled="inputConfig.props?.disabled"
      :style="inputConfig.props?.style"
      @input="handleInput"
      @clear="handleClear"
    />

    <!-- 🔥 下拉选择 -->
    <el-select
      v-else-if="inputConfig.component === 'ElSelect'"
      v-model="localValue"
      :placeholder="inputConfig.props?.placeholder"
      :clearable="inputConfig.props?.clearable"
      :filterable="inputConfig.props?.filterable"
      :remote="inputConfig.props?.remote"
      :remote-method="handleRemoteMethod"
      :multiple="inputConfig.props?.multiple"
      :loading="selectLoading || inputConfig.props?.loading"
      :popper-class="inputConfig.props?.popperClass"
      :style="inputConfig.props?.style"
      :collapse-tags="inputConfig.props?.multiple"
      :max-collapse-tags="3"
      :reserve-keyword="inputConfig.props?.remote && inputConfig.props?.multiple"
      class="user-select-search"
      @change="handleInput"
      @clear="handleClear"
    >
      <!-- 🔥 自定义标签显示（multiple 模式，使用 user-cell 样式） -->
      <template v-if="inputConfig.props?.multiple && inputConfig.props?.popperClass === 'user-select-dropdown-popper'" #tag="{ item, close }">
        <div
          v-if="item"
          class="user-cell user-cell-tag"
        >
          <el-avatar 
            v-if="item.value && getUserInfoByValue(item.value)"
            :src="getUserInfoByValue(item.value)?.avatar" 
            :size="24" 
            class="user-avatar"
          >
            {{ getUserInfoByValue(item.value)?.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <el-avatar 
            v-else
            :size="24" 
            class="user-avatar"
          >
            {{ (item?.label || '')?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <span class="user-name">{{ item?.label || '' }}</span>
          <el-icon class="user-tag-close" @click.stop="close">
            <Close />
          </el-icon>
        </div>
      </template>
      
      <el-option
        v-for="option in selectOptionsComputed"
        :key="typeof option === 'object' ? option.value : option"
        :label="typeof option === 'object' ? option.label : option"
        :value="typeof option === 'object' ? option.value : option"
      >
        <!-- 🔥 如果是用户选择器，显示头像和用户信息 -->
        <div v-if="option.userInfo" class="user-option">
          <el-avatar :src="option.userInfo.avatar" :size="24" class="user-avatar">
            {{ option.userInfo.username?.[0]?.toUpperCase() || 'U' }}
          </el-avatar>
          <span class="user-name">{{ option.userInfo.username }}</span>
          <span v-if="option.userInfo.nickname" class="user-nickname">({{ option.userInfo.nickname }})</span>
        </div>
        <!-- 普通选项 -->
        <span v-else>{{ typeof option === 'object' ? option.label : option }}</span>
      </el-option>
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
      v-model="dateRangeValue"
      :type="inputConfig.props?.type"
      :range-separator="inputConfig.props?.rangeSeparator"
      :start-placeholder="inputConfig.props?.startPlaceholder"
      :end-placeholder="inputConfig.props?.endPlaceholder"
      :format="inputConfig.props?.format"
      :value-format="inputConfig.props?.valueFormat"
      :shortcuts="inputConfig.props?.shortcuts"
      :clearable="inputConfig.props?.clearable"
      :style="inputConfig.props?.style"
      @change="handleDateRangeChange"
      @clear="handleClear"
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
import { ref, computed, watch, nextTick } from 'vue'
import { ElAvatar, ElIcon } from 'element-plus'
import { Close } from '@element-plus/icons-vue'
import UserSearchInput from './UserSearchInput.vue'
import { WidgetBuilder } from '@/core/factories/WidgetBuilder'
import { ErrorHandler } from '@/core/utils/ErrorHandler'
import type { FieldConfig } from '@/types'

// 防抖函数
function debounce<T extends (...args: any[]) => any>(func: T, wait: number): T {
  let timeout: ReturnType<typeof setTimeout> | null = null
  return ((...args: any[]) => {
    if (timeout) clearTimeout(timeout)
    timeout = setTimeout(() => {
      func(...args)
    }, wait)
  }) as T
}

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

// 日期范围值（用于 ElDatePicker，数组格式 [start, end]）
const dateRangeValue = ref<[number | string | null, number | string | null] | null>(null)

// 范围值（最小值、最大值，用于 NumberRangeInput 和 RangeInput）
// 🔥 对于时间戳类型，可能是数组 [start, end]，对于数字类型，可能是对象 { min, max }
const rangeValue = ref<any>({
  min: undefined,
  max: undefined
})

// 初始化范围值（在 watch 中处理，这里不需要初始化）

// 下拉选项列表（用于 remote 模式）
const selectOptions = ref<Array<{ label: string; value: any }>>([])

// 下拉加载状态
const selectLoading = ref(false)

// 🔥 根据值获取用户信息（用于标签显示）
const getUserInfoByValue = (value: any): any => {
  if (!value) return null
  if (!selectOptions.value || !Array.isArray(selectOptions.value)) return null
  const option = selectOptions.value.find((opt: any) => {
    if (!opt) return false
    const optValue = typeof opt === 'object' ? opt.value : opt
    return String(optValue) === String(value)
  })
  return option?.userInfo || null
}

// 🔥 提取下拉选项（兼容静态 options 和 remote 模式）
const selectOptionsComputed = computed(() => {
  if (inputConfig.value.component !== 'ElSelect') {
    return []
  }
  // 如果有静态 options，使用静态 options
  if (inputConfig.value.props?.options && inputConfig.value.props.options.length > 0) {
    return inputConfig.value.props.options
  }
  // 否则使用 remote 模式下的动态选项
  return selectOptions.value
})

// 🔥 处理 remote-method（如果有）
const handleRemoteMethod = async (query: string) => {
  if (inputConfig.value.component !== 'ElSelect' || !inputConfig.value.onRemoteMethod) {
    return
  }
  
  selectLoading.value = true
  try {
    const options = await inputConfig.value.onRemoteMethod(query)
    selectOptions.value = options || []
  } catch (error) {
    console.error('[SearchInput] Remote method error:', error)
    selectOptions.value = []
  } finally {
    selectLoading.value = false
  }
}

// 🔥 初始化已选中的值对应的选项（用于 remote 模式回显）
const initSelectedOptions = async () => {
  if (inputConfig.value.component !== 'ElSelect') {
    return
  }
  
  // 获取当前已选中的值
  const currentValue = localValue.value
  if (!currentValue) {
    return
  }
  
  // 🔥 优先使用 onInitOptions（如果存在），用于批量查询已选中值
  if (inputConfig.value.onInitOptions) {
    selectLoading.value = true
    try {
      const options = await inputConfig.value.onInitOptions(currentValue)
      selectOptions.value = options || []
    } catch (error) {
      console.error('[SearchInput] Init selected options error:', error)
      selectOptions.value = []
    } finally {
      selectLoading.value = false
    }
    return
  }
  
  // 🔥 如果没有 onInitOptions，回退到使用 onRemoteMethod（逐个查询）
  if (!inputConfig.value.onRemoteMethod) {
    return
  }
  
  // 如果是数组（multiple 模式），需要为每个值查询选项
  if (Array.isArray(currentValue) && currentValue.length > 0) {
    selectLoading.value = true
    try {
      // 为每个已选中的值查询对应的选项
      const optionPromises = currentValue.map(async (val: any) => {
        if (!val) return null
        const options = await inputConfig.value.onRemoteMethod(String(val))
        return options?.find((opt: any) => {
          const optValue = typeof opt === 'object' ? opt.value : opt
          return String(optValue) === String(val)
        })
      })
      
      const options = (await Promise.all(optionPromises)).filter(Boolean)
      selectOptions.value = options
    } catch (error) {
      console.error('[SearchInput] Init selected options error:', error)
    } finally {
      selectLoading.value = false
    }
  } else if (typeof currentValue === 'string' && currentValue.trim()) {
    // 单个值，直接查询
    selectLoading.value = true
    try {
      const options = await inputConfig.value.onRemoteMethod(String(currentValue))
      // 确保当前值在选项中
      const currentOption = options?.find((opt: any) => {
        const optValue = typeof opt === 'object' ? opt.value : opt
        return String(optValue) === String(currentValue)
      })
      if (currentOption) {
        selectOptions.value = [currentOption]
      } else if (options && options.length > 0) {
        selectOptions.value = options
      }
    } catch (error) {
      console.error('[SearchInput] Init selected options error:', error)
    } finally {
      selectLoading.value = false
    }
  }
}

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

// 处理单值输入（带防抖，实时同步URL）
const handleInputDebounced = debounce((value: any) => {
  // 🔥 清空时 value 可能是 null、undefined 或空字符串，统一转换为 null
  const normalizedValue = (value === '' || value === null || value === undefined) ? null : value
  emit('update:modelValue', normalizedValue)
}, 300)

const handleInput = (value: any) => {
  localValue.value = value
  // 🔥 使用防抖，避免频繁更新URL
  handleInputDebounced(value)
}

// 处理清空事件（ElInput、ElSelect、ElDatePicker 等组件的 clearable）
const handleClear = () => {
  localValue.value = null
  dateRangeValue.value = null
  rangeValue.value = { min: undefined, max: undefined }
  // 🔥 清空时立即触发更新，不使用防抖
  emit('update:modelValue', null)
}

// 处理范围输入（NumberRangeInput 和 RangeInput）
const handleRangeChange = () => {
  const min = rangeValue.value.min
  const max = rangeValue.value.max
  // 🔥 如果 min 和 max 都为空，传递 null 而不是空对象
  if ((min === undefined || min === null || min === '') && 
      (max === undefined || max === null || max === '')) {
    emit('update:modelValue', null)
  } else {
    emit('update:modelValue', {
      min: min === '' ? undefined : min,
      max: max === '' ? undefined : max
    })
  }
}

// 处理日期范围输入（ElDatePicker）
const handleDateRangeChange = (value: [number | string | null, number | string | null] | null) => {
  dateRangeValue.value = value
  // 🔥 ElDatePicker 返回数组格式 [start, end]，直接传递给父组件
  emit('update:modelValue', value)
}

// 监听外部值变化
watch(() => props.modelValue, (newValue: any) => {
  console.log(`[SearchInput] ${props.field.code} modelValue 变化:`, newValue, 'searchType:', props.searchType)
  
  if (props.searchType?.includes('gte') && props.searchType?.includes('lte')) {
    // 🔥 如果是数组格式（时间戳范围），用于 ElDatePicker
    if (Array.isArray(newValue)) {
      dateRangeValue.value = [
        newValue[0] || null,
        newValue[1] || null
      ]
      // 同时设置 rangeValue 用于其他范围输入组件
      rangeValue.value = {
        min: newValue[0] || undefined,
        max: newValue[1] || undefined
      }
      console.log(`[SearchInput] ${props.field.code} 设置日期范围值:`, dateRangeValue.value)
    } else if (newValue && typeof newValue === 'object') {
      // 已经是对象格式（数字范围）
      rangeValue.value = newValue
      dateRangeValue.value = null
    } else {
      rangeValue.value = { min: undefined, max: undefined }
      dateRangeValue.value = null
    }
  } else {
    localValue.value = newValue
    // 🔥 当值变化时，如果是 remote 模式的 ElSelect，初始化已选中值的选项
    if (inputConfig.value.component === 'ElSelect' && 
        inputConfig.value.props?.remote && 
        newValue && 
        (Array.isArray(newValue) ? newValue.length > 0 : true)) {
      // 延迟执行，确保 inputConfig 已更新
      nextTick(() => {
        initSelectedOptions()
      })
    }
  }
}, { immediate: true })

// 🔥 监听 inputConfig 变化，初始化已选中值的选项
watch(() => inputConfig.value, () => {
  if (inputConfig.value.component === 'ElSelect' && inputConfig.value.props?.remote && localValue.value) {
    initSelectedOptions()
  }
})
</script>

<style scoped>
/* 🔥 用户选择器选项样式（与 UserWidget 保持一致） */
.user-option {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-avatar {
  flex-shrink: 0;
}

.user-name {
  flex: 1;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.user-nickname {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

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

/* 🔥 用户选择器选中后的标签样式（multiple 模式，使用 user-cell 样式） */
.user-select-search :deep(.el-select__tags) {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.user-cell-tag {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  position: relative;
  padding-right: 20px;
}

.user-cell-tag .user-avatar {
  flex-shrink: 0;
  width: 24px !important;
  height: 24px !important;
}

.user-cell-tag .user-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

.user-tag-close {
  position: absolute;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  cursor: pointer;
  color: var(--el-text-color-secondary);
  transition: color 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.user-tag-close:hover {
  color: var(--el-text-color-primary);
}
</style>

<style>
/* 🔥 用户选择器下拉框样式（全局样式，与 UserWidget 保持一致） */
.user-select-dropdown-popper .el-select-dropdown__item {
  padding: 8px 12px;
}

.user-select-dropdown-popper .el-select-dropdown__item:hover {
  background-color: var(--el-fill-color-light);
}
</style>

