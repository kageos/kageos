/**
 * useFormParamURLSync - Form 参数 URL 同步 Composable
 * 
 * 功能：
 * - 监听表单字段变化，自动更新 URL 参数
 * - 支持复杂字段的序列化（JSON）
 * - 支持从 URL 参数回显表单（通过 useFunctionParamInitialization 的 URLParamsInitSource）
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
import { RouteSource } from '@/utils/routeSource'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import { Logger } from '@/core/utils/logger'
import { WidgetType } from '@/core/constants/widget'

export interface UseFormParamURLSyncOptions {
  functionDetail: Ref<FunctionDetail | null> | ComputedRef<FunctionDetail | null>
  formDataStore: {
    getValue: (fieldCode: string) => FieldValue
    getAllValues: () => Record<string, FieldValue>
  }
  enabled?: boolean  // 是否启用 URL 同步（默认 true）
  debounceMs?: number  // 防抖延迟（默认 300ms）
}

/**
 * 构建 Form 查询参数
 * 
 * @param requestFields 请求字段配置
 * @param formDataStore 表单数据 store
 * @returns URL 查询参数对象
 */
function buildFormQueryParams(
  requestFields: FieldConfig[],
  formDataStore: UseFormParamURLSyncOptions['formDataStore']
): Record<string, string> {
  const query: Record<string, string> = {}
  
  requestFields.forEach(field => {
    const fieldValue = formDataStore.getValue(field.code)
    
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
    
    // 🔥 黑名单模式：默认都支持 URL 同步，只有复杂类型和密码字段不支持
    const widgetType = field.widget?.type
    const widgetConfig = field.widget?.config as any
    
    // 排除复杂类型
    const unsupportedTypes = [WidgetType.FORM, WidgetType.TABLE, WidgetType.FILES]
    if (widgetType && unsupportedTypes.includes(widgetType)) {
      Logger.debug('[useFormParamURLSync]', `字段 ${field.code} 是复杂类型（${widgetType}），跳过 URL 同步`)
      return
    }
    
    // 🔥 排除密码字段（安全性考虑：密码不应出现在 URL 中）
    if (widgetType === WidgetType.INPUT && widgetConfig?.password === true) {
      Logger.debug('[useFormParamURLSync]', `字段 ${field.code} 是密码字段，跳过 URL 同步（安全性考虑）`)
      return
    }
    
    // 🔥 默认支持所有其他类型：直接转换为字符串
    // 支持的类型包括：input, text, text_area, number, float, switch, select, multiselect, 
    // radio, checkbox, timestamp, ID, rate, user, slider, color, richtext, link, progress 等
    if (Array.isArray(fieldValue.raw)) {
      // 数组类型（如 multiselect）：使用逗号分隔
      query[field.code] = fieldValue.raw.map(v => String(v)).join(',')
    } else {
      // 其他类型：直接转换为字符串
      query[field.code] = String(fieldValue.raw)
    }
  })
  
  return query
}

/**
 * 同步表单参数到 URL
 */
export function useFormParamURLSync(options: UseFormParamURLSyncOptions) {
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
    
    // 构建表单查询参数
    const requestFields = detail.request || []
    const query = buildFormQueryParams(requestFields, options.formDataStore)
    
    // 获取当前 URL 的查询参数
    const currentQuery = route.query
    const hasQueryParams = Object.keys(currentQuery).length > 0
    const isLinkNavigation = currentQuery._link_type === 'form'
    
    console.log('🔍 [useFormParamURLSync] 开始同步到 URL', {
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
      console.log('🔍 [useFormParamURLSync] URL 没有查询参数，不保留旧参数，直接使用新参数')
      newQuery = { ...query }
    } else {
      // URL 有查询参数，保留现有参数（如 _link_type、_tab）并合并新的 form 参数
      newQuery = { ...currentQuery }
      
      // 保留以 _ 开头的参数（前端状态参数，如 _tab=OnTableAddRow），但清除 _link_type（临时参数）
      Object.keys(newQuery).forEach(key => {
        if (key.startsWith('_') && key !== '_link_type') {
          // 保留状态参数（如 _tab）
        } else if (key.startsWith('_') && key === '_link_type') {
          // 清除临时参数
          delete newQuery[key]
        }
      })
      
      // 合并新的 form 参数（覆盖旧的同名参数）
      Object.assign(newQuery, query)
      
      console.log('🔍 [useFormParamURLSync] URL 有查询参数，保留现有参数', {
        preservedQuery: newQuery,
        preservedQueryKeys: Object.keys(newQuery)
      })
    }
    
    // 🔥 发出路由更新请求
    console.log('🔍 [useFormParamURLSync] 发出路由更新请求', {
      query: newQuery,
      queryKeys: Object.keys(newQuery),
      queryLength: Object.keys(newQuery).length
    })
    
    eventBus.emit(RouteEvent.updateRequested, {
      query: newQuery,
      preserveParams: {
        table: false,        // Form 不需要保留 table 参数
        search: false,       // Form 不需要保留搜索参数
        state: true,         // 保留状态参数（_ 开头，如 _tab=OnTableAddRow）
        linkNavigation: isLinkNavigation  // 如果是 link 跳转，保留所有参数
      },
      source: RouteSource.FORM_SYNC
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
   * 监听表单数据变化，自动同步到 URL
   */
  const watchFormData = (): void => {
    if (!enabled) {
      return
    }
    
    // 监听所有字段值的变化
    watch(
      () => {
        // 获取所有字段的值，用于触发 watch
        const allValues = options.formDataStore.getAllValues()
        return Object.keys(allValues).map(key => ({
          key,
          raw: allValues[key]?.raw,
          display: allValues[key]?.display
        }))
      },
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
    watchFormData
  }
}

