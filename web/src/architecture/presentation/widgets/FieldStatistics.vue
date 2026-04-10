<template>
  <div class="field-statistics" v-if="hasStatistics">
    <el-row :gutter="0">
      <el-col 
        v-for="(stat, index) in statisticsData" 
        :key="index"
        :span="getStatisticSpan(statisticsData.length)"
      >
        <!-- 数值型：上下展示（标题在上，数值在下） -->
        <div v-if="typeof stat.value === 'number'" class="field-statistic number-statistic">
          <div class="statistic-title">{{ stat.label }}</div>
          <div class="statistic-value">
            {{ formatNumber(stat.value, stat.precision) }}<span v-if="stat.suffix" class="statistic-suffix">{{ stat.suffix }}</span>
          </div>
        </div>
        <!-- 纯展示型数据（字符串） -->
        <div v-else class="field-statistic display-statistic">
          <div class="statistic-title">{{ stat.label }}</div>
          <div class="statistic-value">{{ stat.value }}</div>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FieldConfig } from '@/architecture/presentation/widgets/types'
import { ExpressionParserAdapter } from '@/core/utils/ExpressionParserAdapter'
import { Logger } from '@/core/utils/logger'

interface Props {
  field: FieldConfig
  value: any
  statistics?: Record<string, any>
}

const props = defineProps<Props>()

// 检查是否有统计数据
const hasStatistics = computed(() => {
  return props.statistics && typeof props.statistics === 'object' && Object.keys(props.statistics).length > 0
})

// 计算统计数据
const statisticsData = computed(() => {
  if (!props.statistics) return []
  
  const results: Array<{ label: string; value: number | string; precision?: number; suffix?: string }> = []
  
  // 🔥 判断数据源类型：
  // 1. 如果 value 是数组，说明是表格场景，直接使用数组作为数据源
  // 2. 如果 value 是对象，说明是单个字段场景，需要提取 displayInfo
  
  let dataSource: any[] = []
  let selectedItem: any = null
  
  if (Array.isArray(props.value)) {
    // 🔥 表格场景：value 直接是数组（所有行的数据）
    dataSource = props.value
    // 对于 value() 函数，使用第一行数据作为 selectedItem
    selectedItem = dataSource.length > 0 ? dataSource[0] : null
  } else if (props.value && typeof props.value === 'object') {
    // 🔥 单个字段场景：需要从 value 中提取 displayInfo
    let displayInfo: any = null
    
    // 如果是 FieldValue 对象，从 meta.displayInfo 获取
    if ('meta' in props.value && props.value.meta?.displayInfo) {
      displayInfo = props.value.meta.displayInfo
    } else if ('displayInfo' in props.value) {
      displayInfo = props.value.displayInfo
    } else if ('display_info' in props.value) {
      displayInfo = props.value.display_info
    } else if (props.field.widget?.type === 'multiselect' && Array.isArray(props.value)) {
      // 多选：使用第一个选中项的 DisplayInfo
      if (props.value.length > 0) {
        const firstItem = props.value[0]
        if (firstItem && typeof firstItem === 'object') {
          displayInfo = firstItem.displayInfo || firstItem.display_info || firstItem
        }
      }
    } else {
      // 单选：直接使用 value 作为 DisplayInfo
      displayInfo = props.value
    }
    
    if (!displayInfo || typeof displayInfo !== 'object') {
      return []
    }
    
    // 将 displayInfo 转换为数组格式（ExpressionParser 需要数组）
    dataSource = Array.isArray(displayInfo) ? displayInfo : [displayInfo]
    selectedItem = displayInfo
  } else {
    return []
  }
  
  if (dataSource.length === 0) {
    return []
  }
  
  try {
    for (const [label, expression] of Object.entries(props.statistics)) {
      try {
        // 🔥 使用适配器计算表达式，自动支持新旧两种语法
        // 对于 value() 函数，传递 selectedItem 参数
        const value = ExpressionParserAdapter.evaluate(expression as string, dataSource, selectedItem)
        
        // 判断是数值还是字符串
        if (typeof value === 'number') {
          results.push({
            label,
            value,
            precision: 2 // 默认保留2位小数
          })
        } else {
          results.push({
            label,
            value: value || '暂无信息',
            precision: undefined
          })
        }
      } catch (error: any) {
        Logger.error('[FieldStatistics]', '统计表达式计算失败', {
          label,
          expression,
          error
        })
      }
    }
  } catch (error: any) {
    Logger.error('[FieldStatistics]', '统计计算失败', { error })
  }
  
  return results
})

// 数字格式化
const formatNumber = (value: number, precision?: number) => {
  const p = typeof precision === 'number' ? precision : 0
  if (p > 0) return value.toFixed(p)
  return String(value)
}

// 计算统计组件的span值
const getStatisticSpan = (count: number) => {
  if (count <= 2) return 12
  if (count <= 4) return 6
  if (count <= 6) return 4
  return 3
}
</script>

<style scoped>
.field-statistics {
  margin-top: 12px;
  padding: 16px;
  background-color: var(--el-fill-color-light);
  border-radius: 8px;
  border: 1px solid var(--el-border-color);
  /* 确保宽度与父容器一致，避免右侧边距过大 */
  width: 100%;
  box-sizing: border-box;
}

.field-statistic {
  text-align: center;
}

/* 数值型统计：上下布局（标题在上，数值在下） */
.number-statistic {
  text-align: center;
}

.number-statistic .statistic-title {
  font-size: 13px;
  color: var(--el-text-color-regular);
  margin-bottom: 8px;
}

.number-statistic .statistic-value {
  font-size: 24px;
  font-weight: 600;
  color: var(--el-color-primary);
}

.number-statistic .statistic-suffix {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  margin-left: 4px;
}

.display-statistic {
  text-align: center;
}

.display-statistic .statistic-title {
  font-size: 13px;
  color: var(--el-text-color-regular);
  margin-bottom: 8px;
}

.display-statistic .statistic-value {
  font-size: 24px;
  font-weight: 600;
  color: var(--el-color-primary);
}

/* 确保栅格系统不会产生额外的边距 */
:deep(.el-row) {
  margin: 0 !important;
}

:deep(.el-col) {
  padding: 0 !important;
}
</style>
