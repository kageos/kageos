<!--
  MultiSelectWidget - 多选组件
  🔥 完全新增，不依赖旧代码
  
  功能：
  - 支持多选（返回数组）
  - 支持静态选项和远程搜索
  - 支持最大选择数量限制
  - 支持创建新选项（可选）
-->

<template>
  <div class="multiselect-widget">
    <!-- 编辑模式 -->
    <el-select
      v-if="mode === 'edit'"
      v-model="selectedValues"
      multiple
      filterable
      :remote="hasRemoteSearch"
      :remote-method="remoteMethod"
      :loading="loading"
      :placeholder="placeholder"
      :multiple-limit="maxCount"
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
      clearable
      @change="handleChange"
    >
      <el-option
        v-for="option in options"
        :key="option.value"
        :label="option.label"
        :value="option.value"
      />
    </el-select>
    
    <!-- 响应模式（只读） -->
    <div v-else-if="mode === 'response'" class="response-multiselect">
      <el-tag
        v-for="(value, index) in displayValues"
        :key="index"
        class="tag-item"
      >
        {{ getOptionLabel(value) }}
      </el-tag>
      <span v-if="displayValues.length === 0" class="empty-text">-</span>
    </div>
    
    <!-- 表格单元格模式 -->
    <div v-else-if="mode === 'table-cell'" class="table-cell-multiselect">
      <el-tag
        v-for="(value, index) in displayValues"
        :key="index"
        class="tag-item"
        size="small"
      >
        {{ getOptionLabel(value) }}
      </el-tag>
      <span v-if="displayValues.length === 0" class="empty-text">-</span>
    </div>
    
    <!-- 详情模式 -->
    <div v-else class="detail-multiselect">
      <el-tag
        v-for="(value, index) in displayValues"
        :key="index"
        class="tag-item"
      >
        {{ getOptionLabel(value) }}
      </el-tag>
      <span v-if="displayValues.length === 0" class="empty-text">-</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElSelect, ElOption, ElTag } from 'element-plus'
import type { WidgetComponentProps } from '../types'
import { selectFuzzy } from '@/api/function'
import { Logger } from '../../utils/logger'
import { useFormDataStore } from '../../stores-v2/formData'
import { withDefaults } from 'vue'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {}
  })
})

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const formDataStore = useFormDataStore()

// 配置
const config = computed(() => props.field.widget?.config || {})
const options = computed(() => {
  const staticOptions = config.value.options || []
  return staticOptions.map((opt: any) => {
    if (typeof opt === 'string') {
      return { label: opt, value: opt }
    }
    return opt
  })
})

const placeholder = computed(() => {
  return config.value.placeholder || `请选择${props.field.name}`
})

const maxCount = computed(() => {
  return config.value.max_count || 0  // 0 表示无限制
})

// 是否支持远程搜索
const hasRemoteSearch = computed(() => {
  return props.field.callbacks?.includes('OnSelectFuzzy') || false
})

// 加载状态
const loading = ref(false)

// 选中的值（数组）
const selectedValues = computed({
  get: () => {
    const raw = props.value?.raw
    if (Array.isArray(raw)) {
      return raw
    }
    // 如果值是字符串，尝试转换为数组（兼容旧数据）
    if (typeof raw === 'string' && raw) {
      return [raw]
    }
    return []
  },
  set: (newValues: any[]) => {
    // 验证数量限制
    let finalValues = newValues
    if (maxCount.value > 0 && finalValues.length > maxCount.value) {
      Logger.warn(`[MultiSelectWidget] ${props.field.code} 超出数量限制! 限制: ${maxCount.value}, 实际: ${finalValues.length}`)
      finalValues = finalValues.slice(0, maxCount.value)
    }
    
    const fieldValue = {
      raw: finalValues,
      display: finalValues.length > 0 ? finalValues.join(', ') : '',
      meta: {}
    }
    
    formDataStore.setValue(props.fieldPath, fieldValue)
    emit('update:modelValue', fieldValue)
  }
})

// 显示值（用于只读模式）
const displayValues = computed(() => {
  const raw = props.value?.raw
  if (Array.isArray(raw)) {
    return raw
  }
  // 兼容旧数据：如果是字符串，转换为数组
  if (typeof raw === 'string' && raw) {
    return [raw]
  }
  return []
})

// 获取选项标签
function getOptionLabel(value: any): string {
  const option = options.value.find(opt => opt.value === value)
  return option ? option.label : String(value)
}

// 远程搜索方法
async function remoteMethod(query: string): Promise<void> {
  if (!hasRemoteSearch.value || !query) {
    return
  }
  
  try {
    loading.value = true
    const result = await selectFuzzy(
      props.formRenderer?.getFunctionMethod() || 'GET',
      props.formRenderer?.getFunctionRouter() || '',
      {
        code: props.field.code,
        type: 'by_keyword',
        value: query,
        request: {},
        value_type: props.field.data?.type || 'string'
      }
    )
    
    // 更新选项（这里需要根据实际返回格式调整）
    // TODO: 根据实际 API 返回格式处理
    Logger.debug('[MultiSelectWidget]', '远程搜索结果:', result)
  } catch (error) {
    Logger.error('[MultiSelectWidget]', '远程搜索失败:', error)
  } finally {
    loading.value = false
  }
}

// 处理值变化
function handleChange(values: any[]): void {
  selectedValues.value = values
}

// 初始化：如果字段没有值，使用默认值
watch(
  () => props.value,
  (newValue) => {
    if (!newValue || !newValue.raw) {
      const defaultValue = config.value.default
      if (Array.isArray(defaultValue) && defaultValue.length > 0) {
        selectedValues.value = defaultValue
      }
    }
  },
  { immediate: true }
)
</script>

<style scoped>
.multiselect-widget {
  width: 100%;
}

.response-multiselect,
.table-cell-multiselect,
.detail-multiselect {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.tag-item {
  margin-right: 4px;
}

.empty-text {
  color: #999;
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

