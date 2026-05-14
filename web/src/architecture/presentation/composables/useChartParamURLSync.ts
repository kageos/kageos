/**
 * useChartParamURLSync - Chart 参数 URL 同步 Composable
 * 
 * 功能：
 * - 监听图表筛选条件变化，自动更新 URL 参数
 * - 支持复杂字段的序列化（JSON）
 * - 支持从 URL 参数回显筛选条件（通过 useFunctionParamInitialization 的 URLParamsInitSource）
 * 
 * 🔥 设计原则（黑名单模式）：
 * - 默认支持所有组件类型的 URL 同步
 * - 黑名单：复杂类型（form、table、files）+ 密码字段（安全性考虑）
 * - 空值不添加到 URL（保持 URL 简洁）
 * - 支持所有 template_type（form、table、chart 等），通过 enabled 参数控制是否启用
 */

import { watch, computed, type Ref, type ComputedRef } from 'vue'
import { useRoute } from 'vue-router'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { RouteSource } from '@/architecture/runtime/utils/routeSource'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import { Logger } from '@/architecture/shared/logger'
import { isEmptyValue, shouldSkipURLSync, convertFieldValueToURLParam, mergeURLQueryParams } from './utils/urlSyncUtils'
import { isLinkNavigation } from '@/architecture/runtime/utils/linkNavigation'
import { getChartRequestFields } from '@/architecture/domain/utils/functionSchemaSelectors'
import { deleteFieldQueryKey } from '@/architecture/runtime/utils/queryParamKeys'

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

    if (!fieldValue) {
      return
    }
    
    // 跳过空值
    if (isEmptyValue(fieldValue)) {
      return
    }
    
    // 黑名单检查：排除复杂类型和密码字段
    if (shouldSkipURLSync(field, '[useChartParamURLSync]')) {
      return
    }
    
    // 🔥 默认支持所有其他类型：转换为 URL 参数
    // 支持的类型包括：input, text, text_area, number, float, switch, select, multiselect, 
    // radio, checkbox, datetime, ID, rate, user, slider, color, richtext, link, progress 等
    // Chart request 参数是 sdk-app 入参，保持原始 field.code，
    // 让 URL 和 sdk-app json/form tag 对齐。
    query[field.code] = convertFieldValueToURLParam(fieldValue)
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
    
    // 🔥 黑名单模式：默认都支持 URL 同步，只有特定场景不支持
    const detail = functionDetail.value
    if (!detail) {
      return
    }
    
    // 🔥 支持所有 template_type（form、table、chart 等），不再限制
    // 如果某个场景不需要 URL 同步，可以通过 enabled 参数控制
    
    // 构建图表查询参数
    const requestFields = getChartRequestFields(detail)
    const query = buildChartQueryParams(requestFields, options.fieldValues.value)
    
    // 获取当前 URL 的查询参数并合并
    const currentQuery = { ...route.query }
    requestFields.forEach(field => {
      deleteFieldQueryKey(currentQuery, field.code)
    })
    const newQuery = mergeURLQueryParams(currentQuery, query, 'chart')
    
    // 判断是否是 link 跳转
    const isLinkNav = isLinkNavigation(currentQuery)
    
    Logger.debug('[useChartParamURLSync]', '发出路由更新请求', {
      queryKeys: Object.keys(newQuery),
      queryLength: Object.keys(newQuery).length,
      isLinkNavigation: isLinkNav
    })
    
    eventBus.emit(RouteEvent.updateRequested, {
      query: newQuery,
      preserveParams: {
        table: false,        // Chart 不需要保留 table 参数
        search: false,       // Chart 不需要保留搜索参数
        state: true,         // 保留状态参数（_ 开头）
        linkNavigation: isLinkNav  // 如果是 link 跳转，保留所有参数
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
