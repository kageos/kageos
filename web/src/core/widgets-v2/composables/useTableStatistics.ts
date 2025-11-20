/**
 * useTableStatistics - TableWidget 聚合计算组合式函数
 * 🔥 完全新增，不依赖旧代码
 * 
 * 功能：
 * - 自动从所有行的 meta.statistics 中收集聚合配置
 * - 使用 computed 自动计算聚合结果
 * - 支持行内聚合和 List 层聚合
 */

import { computed, ref, watch } from 'vue'
import { ExpressionParser } from '../../utils/ExpressionParser'
import type { WidgetComponentProps } from '../types'
import { useFormDataStore } from '../../stores-v2/formData'
import { Logger } from '../../utils/logger'

export function useTableStatistics(
  props: WidgetComponentProps,
  getAllRowsData: () => any[]
) {
  const formDataStore = useFormDataStore()
  
  // 聚合统计配置（从所有行的 meta.statistics 中自动收集）
  const statisticsConfig = ref<Record<string, string>>({})
  
  /**
   * 🔥 从所有行中收集聚合配置
   * 响应式自动收集：当行数据变化时，自动更新配置
   */
  function collectStatisticsConfig(): void {
    try {
      const allRows = getAllRowsData()
      const configMap = new Map<string, string>()
      
      Logger.debug('[useTableStatistics]', '开始收集聚合配置', { rowCount: allRows.length })
      
      // 遍历所有行，收集每个字段的 statistics 配置
      props.field.children?.forEach((itemField: any) => {
        const fieldCode = itemField.code
        
        // 遍历所有行，查找该字段的 statistics 配置
        for (let i = 0; i < allRows.length; i++) {
          const fieldPath = `${props.fieldPath}[${i}].${fieldCode}`
          const itemValue = formDataStore.getValue(fieldPath)
          
          // 如果该字段有 statistics 配置，收集它
          if (itemValue?.meta?.statistics && typeof itemValue.meta.statistics === 'object') {
            const statistics = itemValue.meta.statistics
            Logger.debug('[useTableStatistics]', `找到 statistics 配置`, { 
              fieldPath, 
              fieldCode, 
              statistics 
            })
            // 合并所有统计配置（使用第一个遇到的配置，通常所有行的配置应该一致）
            Object.entries(statistics).forEach(([label, expression]) => {
              if (typeof expression === 'string' && !configMap.has(label)) {
                configMap.set(label, expression)
                Logger.debug('[useTableStatistics]', `添加统计配置`, { label, expression })
              }
            })
          }
        }
      })
      
      // 更新配置
      const newConfig: Record<string, string> = {}
      configMap.forEach((expression, label) => {
        newConfig[label] = expression
      })
      
      // 只有当配置发生变化时才更新（避免无限循环）
      const configStr = JSON.stringify(newConfig)
      const oldConfigStr = JSON.stringify(statisticsConfig.value)
      if (configStr !== oldConfigStr) {
        statisticsConfig.value = newConfig
        Logger.debug('[useTableStatistics]', '聚合配置已更新', newConfig)
      } else {
        Logger.debug('[useTableStatistics]', '聚合配置无变化', newConfig)
      }
    } catch (error) {
      Logger.error('[useTableStatistics]', '收集聚合配置失败', error)
    }
  }
  
  // 🔥 响应式监听：当行数据变化时，自动收集配置
  // 使用 computed 来追踪所有行的数据变化
  const dataWatcher = computed(() => {
    // 访问 formDataStore.data 来触发响应式追踪
    const _ = formDataStore.data
    
    // 获取所有行数据
    const allRows = getAllRowsData()
    
    // 收集每行的关键信息用于变化检测（包括 raw 值和 statistics）
    const signatures: string[] = []
    
    for (let i = 0; i < allRows.length; i++) {
      const rowSignature: Record<string, any> = { index: i }
      
      props.field.children?.forEach((itemField: any) => {
        const fieldPath = `${props.fieldPath}[${i}].${itemField.code}`
        const itemValue = formDataStore.getValue(fieldPath)
        
        // 记录 raw 值（用于检测行数据变化）
        rowSignature[itemField.code] = itemValue?.raw
        
        // 记录 statistics 配置（用于检测配置变化）
        if (itemValue?.meta?.statistics && typeof itemValue.meta.statistics === 'object') {
          rowSignature[`${itemField.code}_statistics`] = JSON.stringify(itemValue.meta.statistics)
        }
      })
      
      signatures.push(JSON.stringify(rowSignature))
    }
    
    // 返回签名组合，用于 watch 变化检测
    return signatures.join('|') || 'empty'
  })
  
  // 监听数据变化（包括行数据和 statistics 配置）
  watch(
    dataWatcher,
    () => {
      // 数据变化时，重新收集配置
      collectStatisticsConfig()
    },
    { immediate: true }
  )
  
  // 🔥 聚合统计结果（使用 computed 自动计算）
  const statisticsResult = computed(() => {
    if (!statisticsConfig.value || Object.keys(statisticsConfig.value).length === 0) {
      Logger.debug('[useTableStatistics]', '无聚合配置，返回空结果')
      return {}
    }
    
    try {
      const allRows = getAllRowsData()
      Logger.debug('[useTableStatistics]', '开始计算聚合结果', { 
        config: statisticsConfig.value, 
        rowCount: allRows.length 
      })
      
      const result: Record<string, any> = {}
      
      for (const [label, expression] of Object.entries(statisticsConfig.value)) {
        try {
          const value = ExpressionParser.evaluate(expression, allRows)
          result[label] = value
          Logger.debug('[useTableStatistics]', `计算成功: ${label} = ${value}`, { expression })
        } catch (error) {
          Logger.error(`[useTableStatistics] 计算失败: ${label} = ${expression}`, error)
          result[label] = 0
        }
      }
      
      Logger.debug('[useTableStatistics]', '聚合计算结果', result)
      return result
    } catch (error) {
      Logger.error('[useTableStatistics] 聚合计算失败', error)
      return {}
    }
  })
  
  // 设置聚合配置（手动设置，用于外部调用）
  function setStatisticsConfig(config: Record<string, string>): void {
    statisticsConfig.value = config
  }
  
  return {
    statisticsConfig,
    statisticsResult,
    setStatisticsConfig,
    collectStatisticsConfig
  }
}

