/**
 * useTableStatistics - TableWidget 聚合计算组合式函数
 * 🔥 完全新增，不依赖旧代码
 */

import { computed, ref } from 'vue'
import { ExpressionParser } from '../../utils/ExpressionParser'
import type { WidgetComponentProps } from '../types'

export function useTableStatistics(
  props: WidgetComponentProps,
  getAllRowsData: () => any[]
) {
  // 聚合统计配置（从回调获取）
  const statisticsConfig = ref<Record<string, string>>({})
  
  // 🔥 聚合统计结果（使用 computed 自动计算）
  const statisticsResult = computed(() => {
    if (!statisticsConfig.value || Object.keys(statisticsConfig.value).length === 0) {
      return {}
    }
    
    try {
      const allRows = getAllRowsData()
      const result: Record<string, any> = {}
      
      for (const [label, expression] of Object.entries(statisticsConfig.value)) {
        try {
          const value = ExpressionParser.evaluate(expression, allRows)
          result[label] = value
        } catch (error) {
          console.error(`[TableWidget] 计算失败: ${label} = ${expression}`, error)
          result[label] = 0
        }
      }
      
      return result
    } catch (error) {
      console.error('[TableWidget] 聚合计算失败', error)
      return {}
    }
  })
  
  // 设置聚合配置
  function setStatisticsConfig(config: Record<string, string>): void {
    statisticsConfig.value = config
  }
  
  return {
    statisticsConfig,
    statisticsResult,
    setStatisticsConfig
  }
}

