/**
 * useChartParamURLSync - Chart 参数 URL 同步 Composable
 * 
 * 功能：
 * - 监听图表筛选条件变化，自动更新 URL 参数
 * - 支持复杂字段的序列化（JSON）
 * - 支持从 URL 参数回显筛选条件（通过 useFunctionParamInitialization 的 URLParamsInitSource）
 * 
 * 🔥 设计原则：
 * - 只同步简单字段到 URL（字符串、数字、布尔值）
 * - 复杂字段（form、table、files）使用 JSON 序列化
 * - 空值不添加到 URL（保持 URL 简洁）
 */

import { watch, computed, type Ref, type ComputedRef } from 'vue'
import { useRoute } from 'vue-router'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { RouteSource } from '@/utils/routeSource'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import { Logger } from '@/core/utils/logger'

export interface UseChartParamURLSyncOptions {
  functionDetail: Ref<FunctionDetail | null> | ComputedRef<FunctionDetail | null>
  fieldValues: Ref<Record<string, FieldValue>>  // Chart 的字段值对象
  enabled?: boolean  // 是否启用 URL 同步（默认 true）
  debounceMs?: number  // 防抖延迟（默认 300ms）
}

/**
 * 构建 Chart 查询参数
 * 
 * @param requestFields 请求字段配置
 * @param fieldValues 字段值对象
 * @returns URL 查询参数对象
 */
function buildChartQueryParams(
  requestFields: FieldConfig[],
  fieldValues: Record<string, FieldValue>
): Record<string, string> {
  const query: Record<string, string> = {}
  
  requestFields.forEach(field => {
    const fieldValue = fieldValues[field.code]
    
    // 跳过空值
    if (!fieldValue || fieldValue.raw === null || fieldValue.raw === undefined) {
      return
    }
    
    // 跳过空字符串
    if (typeof fieldValue.raw === 'string' && fieldValue.raw.trim() === '') {
      return
    }
    
    // 跳过空数组
    if (Array.isArray(fieldValue.raw) && fieldValue.raw.length === 0) {
      return
    }
    
    // 跳过空对象
    if (typeof fieldValue.raw === 'object' && !Array.isArray(fieldValue.raw) && Object.keys(fieldValue.raw).length === 0) {
      return
    }
    
    // 🔥 暂不支持复杂类型（form、table、files）的 URL 同步（太复杂，后续通过快链支持）
    const widgetType = field.widget?.type
    if (widgetType === WidgetType.FORM || widgetType === WidgetType.TABLE || widgetType === WidgetType.FILES) {
      Logger.debug('[useChartParamURLSync]', `字段 ${field.code} 是复杂类型（${widgetType}），跳过 URL 同步，后续通过快链支持`)
      return
    }
    
    // 处理简单类型（字符串、数字、布尔值）
    if (widgetType === WidgetType.INPUT || widgetType === WidgetType.TEXT || widgetType === WidgetType.TEXT_AREA || 
        widgetType === WidgetType.NUMBER || widgetType === WidgetType.FLOAT || widgetType === WidgetType.SWITCH ||
        widgetType === WidgetType.SELECT || widgetType === WidgetType.RADIO || widgetType === WidgetType.CHECKBOX ||
        widgetType === WidgetType.TIMESTAMP || widgetType === WidgetType.ID) {
      // 简单类型直接转换为字符串
      if (Array.isArray(fieldValue.raw)) {
        // 多选：使用逗号分隔
        query[field.code] = fieldValue.raw.map(v => String(v)).join(',')
      } else {
        query[field.code] = String(fieldValue.raw)
      }
    } else if (widgetType === WidgetType.MULTI_SELECT) {
      // 多选：使用逗号分隔
      if (Array.isArray(fieldValue.raw)) {
        query[field.code] = fieldValue.raw.map(v => String(v)).join(',')
      } else {
        query[field.code] = String(fieldValue.raw)
      }
    } else {
      // 其他类型：暂不支持 URL 同步
      Logger.debug('[useChartParamURLSync]', `字段 ${field.code} 类型 ${widgetType} 暂不支持 URL 同步，后续通过快链支持`)
      return
    }
  })
  
  return query
}

/**
 * 同步图表参数到 URL
 */
export function useChartParamURLSync(options: UseChartParamURLSyncOptions) {
  const route = useRoute()
  const enabled = options.enabled !== false  // 默认启用
  const debounceMs = options.debounceMs || 300
  
  // 计算 functionDetail（支持 Ref 和 ComputedRef）
  const functionDetail = computed(() => {
    const detail = options.functionDetail
    return detail && typeof detail === 'object' && 'value' in detail ? detail.value : detail
  })
  
  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  
  /**
   * 同步到 URL
   */
  const syncToURL = (): void => {
    if (!enabled) {
      return
    }
    
    // 检查当前函数类型，如果是非 chart 函数，不应该调用 syncToURL
    const detail = functionDetail.value
    if (!detail || detail.template_type !== 'chart') {
      return
    }
    
    // 构建图表查询参数
    const requestFields = detail.request || []
    const query = buildChartQueryParams(requestFields, options.fieldValues.value)
    
    // 获取当前 URL 的查询参数
    const currentQuery = route.query
    const hasQueryParams = Object.keys(currentQuery).length > 0
    const isLinkNavigation = currentQuery._link_type === 'chart'
    
    console.log('🔍 [useChartParamURLSync] 开始同步到 URL', {
      hasQueryParams,
      currentQueryKeys: Object.keys(currentQuery),
      isLinkNavigation,
      newQuery: query,
      newQueryKeys: Object.keys(query)
    })
    
    // 🔥 如果 URL 没有查询参数（刚切换函数），直接使用新的查询参数，不保留任何旧参数
    let newQuery: Record<string, string | string[]>
    if (!hasQueryParams && !isLinkNavigation) {
      // 刚切换函数，URL 是空的，直接使用新的查询参数
      console.log('🔍 [useChartParamURLSync] URL 没有查询参数，不保留旧参数，直接使用新参数')
      newQuery = { ...query }
    } else {
      // URL 有查询参数，保留现有参数（如 _link_type）并合并新的 chart 参数
      newQuery = { ...currentQuery }
      
      // 保留以 _ 开头的参数（前端状态参数），但清除 _link_type（临时参数）
      Object.keys(newQuery).forEach(key => {
        if (key.startsWith('_') && key !== '_link_type') {
          // 保留状态参数
        } else if (key.startsWith('_') && key === '_link_type') {
          // 清除临时参数
          delete newQuery[key]
        }
      })
      
      // 合并新的 chart 参数（覆盖旧的同名参数）
      Object.assign(newQuery, query)
      
      console.log('🔍 [useChartParamURLSync] URL 有查询参数，保留现有参数', {
        preservedQuery: newQuery,
        preservedQueryKeys: Object.keys(newQuery)
      })
    }
    
    // 🔥 发出路由更新请求
    console.log('🔍 [useChartParamURLSync] 发出路由更新请求', {
      query: newQuery,
      queryKeys: Object.keys(newQuery),
      queryLength: Object.keys(newQuery).length
    })
    
    eventBus.emit(RouteEvent.updateRequested, {
      query: newQuery,
      preserveParams: {
        table: false,        // Chart 不需要保留 table 参数
        search: false,       // Chart 不需要保留搜索参数
        state: true,         // 保留状态参数（_ 开头）
        linkNavigation: isLinkNavigation  // 如果是 link 跳转，保留所有参数
      },
      source: RouteSource.CHART_SYNC
    })
  }
  
  /**
   * 防抖版本的 syncToURL
   */
  const debouncedSyncToURL = (): void => {
    if (debounceTimer) {
      clearTimeout(debounceTimer)
    }
    debounceTimer = setTimeout(() => {
      syncToURL()
      debounceTimer = null
    }, debounceMs)
  }
  
  /**
   * 监听图表筛选条件变化，自动同步到 URL
   */
  const watchChartData = (): void => {
    if (!enabled) {
      return
    }
    
    // 监听字段值的变化
    watch(
      () => options.fieldValues.value,
      () => {
        // 字段值变化时，防抖同步到 URL
        debouncedSyncToURL()
      },
      { deep: true }
    )
  }
  
  return {
    syncToURL,
    debouncedSyncToURL,
    watchChartData
  }
}

