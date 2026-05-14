/**
 * useFormParamURLSync - Form 参数 URL 同步 Composable
 *
 * ============================================
 * 📋 需求说明
 * ============================================
 *
 * 1. **URL 同步范围**：
 *    - 独立 Form 函数页面同步表单数据到 URL
 *    - Table 新增模式同步表单数据到 URL（`_tab=OnTableAddRow`）
 *    - 编辑模式和详情模式不同步 URL（`_tab=detail` 或合成表单）
 *    - 同步时保留表格参数、搜索参数等其他参数
 *
 * 2. **字段过滤**：
 *    - 黑名单模式：默认支持所有字段类型
 *    - 排除复杂类型：form、table、files（序列化复杂，URL 过长）
 *    - 排除密码字段：安全性考虑
 *    - 空值不添加到 URL（保持 URL 简洁）
 *
 * 3. **参数保留**：
 *    - 保留表格参数（page、page_size、sorts）
 *    - 保留搜索参数（like、eq、in 等）
 *    - 保留其他状态参数（linkNavigation 等）
 *
 * ============================================
 * 🎯 设计思路
 * ============================================
 *
 * 1. **模式判断**：
 *    - 真实 Form 函数页面默认同步
 *    - `_tab=OnTableAddRow` 的 Table 新增表单同步
 *    - `_tab=detail` 或 id=0 的合成编辑表单不同步
 *
 * 2. **事件驱动**：
 *    - 通过 `RouteEvent.updateRequested` 事件更新 URL
 *    - 不直接操作路由，通过 RouteManager 统一处理
 *    - 支持防抖，避免频繁更新 URL
 *
 * 3. **参数构建**：
 *    - 提取表单字段的 `raw` 值
 *    - 复杂类型（对象、数组）序列化为 JSON
 *    - 合并现有参数，保留非表单参数
 *
 * ============================================
 * 📝 关键功能
 * ============================================
 *
 * 1. **syncToURL**：
 *    - 检查当前场景是否需要同步
 *    - 提取表单字段值并构建查询参数
 *    - 发出路由更新请求事件
 *
 * 2. **buildFormQueryParams**：
 *    - 遍历表单字段，提取值
 *    - 过滤黑名单字段（form、table、files、password）
 *    - 序列化复杂类型为 JSON
 *
 * 3. **watchFormData**：
 *    - 监听表单数据变化
 *    - 防抖处理，避免频繁更新 URL
 *    - 只在启用时监听（`enabled` 参数）
 *
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 *
 * 1. **同步时机**：
 *    - 独立 Form 函数页面和 Table 新增模式同步
 *    - 编辑模式和详情模式不同步，避免 URL 污染
 *
 * 2. **参数保留**：
 *    - 必须保留表格参数、搜索参数等其他参数
 *    - 使用 `preserveParams` 配置保留哪些参数
 *
 * 3. **字段过滤**：
 *    - 复杂类型（form、table）不同步到 URL
 *    - 密码字段不同步到 URL（安全性）
 *    - 空值不添加到 URL
 *
 * ============================================
 * 📚 相关文档
 * ============================================
 *
 * - URL 同步工具：`web/src/architecture/presentation/composables/utils/urlSyncUtils.ts`
 * - 路由管理器：`web/src/architecture/presentation/router/routeManager/RouteManager.ts`
 */

import { watch, computed, type Ref, type ComputedRef } from 'vue'
import { useRoute } from 'vue-router'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { RouteSource } from '@/architecture/shared/routing/routeSource'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import { Logger } from '@/architecture/shared/logger'
import { isEmptyValue, shouldSkipURLSync, convertFieldValueToURLParam, mergeURLQueryParams } from './utils/urlSyncUtils'
import { isLinkNavigation } from '@/architecture/shared/routing/linkNavigation'
import { deleteFieldQueryKey } from '@/architecture/shared/routing/queryParamKeys'
import { getFormRequestFields } from '@/architecture/domain/utils/functionSchemaSelectors'

export interface UseFormParamURLSyncOptions {
  functionDetail: Ref<FunctionDetail | null> | ComputedRef<FunctionDetail | null>
  formDataStore: {
    getValue: (fieldCode: string) => FieldValue
    getAllValues: () => Record<string, FieldValue>
  }
  enabled?: boolean | Ref<boolean> | ComputedRef<boolean>  // 是否启用 URL 同步（默认 true）
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
    if (isEmptyValue(fieldValue)) {
      return
    }

    // 黑名单检查：排除复杂类型和密码字段
    if (shouldSkipURLSync(field, '[useFormParamURLSync]')) {
      return
    }

    // 🔥 默认支持所有其他类型：转换为 URL 参数
    // 支持的类型包括：input, text, text_area, number, float, switch, select, multiselect,
    // radio, checkbox, datetime, ID, rate, user, slider, color, richtext, link, progress 等
    // Form request 参数是 sdk-app 入参，query key 必须保持 schema 原始
    // field.code，并和 sdk-app json/form tag 对齐。
    query[field.code] = convertFieldValueToURLParam(fieldValue)
  })

  return query
}

function getQueryStringValue(value: unknown): string | undefined {
  if (Array.isArray(value)) {
    return value[0] === undefined || value[0] === null ? undefined : String(value[0])
  }

  return value === undefined || value === null ? undefined : String(value)
}

function isRealFormFunction(detail: FunctionDetail): boolean {
  return detail.template_type === 'form' && detail.id !== undefined && detail.id !== null && detail.id !== 0
}

function shouldSyncFormParams(detail: FunctionDetail, query: Record<string, any>): boolean {
  const currentTab = getQueryStringValue(query._tab)

  if (currentTab === 'OnTableAddRow') {
    return true
  }

  if (currentTab) {
    return false
  }

  return isRealFormFunction(detail)
}

/**
 * 同步表单参数到 URL
 */
export function useFormParamURLSync(options: UseFormParamURLSyncOptions) {
  const route = useRoute()
  const enabled = computed(() => {
    const option = options.enabled
    if (option && typeof option === 'object' && 'value' in option) {
      return option.value !== false
    }
    return option !== false
  })
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
    if (!enabled.value) {
      return
    }

    const detail = functionDetail.value
    if (!detail) {
      return
    }

    if (!shouldSyncFormParams(detail, route.query as Record<string, any>)) {
      Logger.debug('[useFormParamURLSync]', '当前场景不需要 Form URL 同步，跳过', {
        currentTab: route.query._tab,
        functionId: detail.id,
        templateType: detail.template_type
      })
      return
    }

    // 🔥 支持所有 template_type（form、table、chart 等），不再限制
    // 如果某个场景不需要 URL 同步，可以通过 enabled 参数控制

    // 构建表单查询参数
    const requestFields = getFormRequestFields(detail)
    const query = buildFormQueryParams(requestFields, options.formDataStore)

    // 获取当前 URL 的查询参数并合并
    const currentQuery = { ...route.query }
    requestFields.forEach(field => {
      // 重写前只删除 request 字段的原始 key；平台状态统一放在 `_` key 下，
      // 由 RouteManager 的 preserve 规则处理。
      deleteFieldQueryKey(currentQuery, field.code)
    })
    const newQuery = mergeURLQueryParams(currentQuery, query, 'form')

    // 判断是否是 link 跳转
    const isLinkNav = isLinkNavigation(currentQuery)

    Logger.debug('[useFormParamURLSync]', '发出路由更新请求', {
      queryKeys: Object.keys(newQuery),
      queryLength: Object.keys(newQuery).length,
      isLinkNavigation: isLinkNav
    })

    eventBus.emit(RouteEvent.updateRequested, {
      query: newQuery,
      preserveParams: {
        table: false,        // Form 不需要保留 table 参数
        search: false,       // Form 不需要保留搜索参数
        state: true,         // 保留状态参数（_ 开头，如 _tab=OnTableAddRow）
        linkNavigation: isLinkNav  // 如果是 link 跳转，保留所有参数
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
        if (enabled.value) {
          // 字段值变化时，防抖同步到 URL
          debouncedSyncToURL()
        }
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
