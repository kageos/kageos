/**
 * useFormParamURLSync - Form 参数 URL 同步 Composable
 *
 * ============================================
 * 📋 需求说明
 * ============================================
 *
 * 1. **URL 同步范围**：
 *    - 仅在新增模式下同步表单数据到 URL（`_tab=OnTableAddRow`）
 *    - 编辑模式和详情模式不同步 URL（`_tab=detail` 或不设置 `_tab`）
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
 *    - 通过 `route.query._tab` 判断是否需要同步
 *    - 只有 `_tab=OnTableAddRow` 时才同步
 *    - 其他模式（`_tab=detail` 或不设置 `_tab`）不同步
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
 *    - 检查是否需要同步（`_tab=OnTableAddRow`）
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
 *    - 只在新增模式下同步（`_tab=OnTableAddRow`）
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
 * - 路由管理器：`web/src/architecture/infrastructure/routeManager/RouteManager.ts`
 */

import { watch, computed, type Ref, type ComputedRef } from 'vue'
import { useRoute } from 'vue-router'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { RouteSource } from '@/utils/routeSource'
import type { FunctionDetail, FieldConfig, FieldValue } from '../../domain/types'
import { Logger } from '@/core/utils/logger'
import { isEmptyValue, shouldSkipURLSync, convertFieldValueToURLParam, mergeURLQueryParams } from './utils/urlSyncUtils'
import { isLinkNavigation } from '@/utils/linkNavigation'

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
    if (isEmptyValue(fieldValue)) {
      return
    }

    // 黑名单检查：排除复杂类型和密码字段
    if (shouldSkipURLSync(field, '[useFormParamURLSync]')) {
      return
    }

    // 🔥 默认支持所有其他类型：转换为 URL 参数
    // 支持的类型包括：input, text, text_area, number, float, switch, select, multiselect,
    // radio, checkbox, timestamp, ID, rate, user, slider, color, richtext, link, progress 等
    query[field.code] = convertFieldValueToURLParam(fieldValue)
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

    // 🔥 双重检查：确保只有在 _tab=OnTableAddRow 时才同步
    // 防止编辑模式（_tab=detail 或不设置 _tab）时误同步
    if (route.query._tab !== 'OnTableAddRow') {
      Logger.debug('[useFormParamURLSync]', '检测到非新增模式标识，跳过 URL 同步', {
        currentTab: route.query._tab
      })
      return
    }

    const detail = functionDetail.value
    if (!detail) {
      return
    }

    // 🔥 支持所有 template_type（form、table、chart 等），不再限制
    // 如果某个场景不需要 URL 同步，可以通过 enabled 参数控制

    // 构建表单查询参数
    // 🔥 确保 requestFields 是数组，防止类型错误
    const requestFields = Array.isArray(detail.request) ? detail.request : []
    const query = buildFormQueryParams(requestFields, options.formDataStore)

    // 获取当前 URL 的查询参数并合并
    const currentQuery = route.query
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

