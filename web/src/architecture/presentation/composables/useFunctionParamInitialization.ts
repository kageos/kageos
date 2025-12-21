/**
 * useFunctionParamInitialization - 统一数据初始化 Composable
 * 
 * 🔥 依赖倒置原则：框架只依赖抽象接口，不依赖具体组件
 * 
 * 功能：
 * - 统一管理所有初始化源（URL、快链、默认值）
 * - 控制初始化顺序
 * - 调用组件自治初始化
 * - 提供统一的初始化接口
 */

import { ref, computed, type ComputedRef } from 'vue'
import { useRoute } from 'vue-router'
import type { FunctionDetail, FieldConfig } from '../../../core/types/field'
import type { FieldValue } from '../../../core/types/field'
import { widgetInitializerRegistry } from '../../../core/widgets-v2/initializers/WidgetInitializerRegistry'
import type { WidgetInitContext } from '../../../core/widgets-v2/interfaces/IWidgetInitializer'
import { eventBus, FormEvent } from '../../infrastructure/eventBus'
import { Logger } from '../../../core/utils/logger'
import { getWidgetDefaultValue } from '../../../core/widgets-v2/composables/useWidgetDefaultValue'
import { useAuthStore } from '@/stores/auth'
import { FieldValueMeta, FieldCallback } from '../../../core/constants/field'
import { DataType } from '../../../core/constants/widget'

/**
 * 初始化源接口
 */
interface InitSource {
  priority: number
  name: string
  initialize: (context: InitContext) => Promise<InitResult>
}

/**
 * 初始化上下文
 */
interface InitContext {
  functionDetail: FunctionDetail
  currentFormData: Record<string, FieldValue>
  route: ReturnType<typeof useRoute>
}

/**
 * 初始化结果
 */
interface InitResult {
  formData: Record<string, FieldValue>
  fieldMetadata?: Record<string, any>
  metadata?: Record<string, any>
}

/**
 * 初始化源优先级
 */
enum InitSourcePriority {
  // 🔥 OnPageLoad 暂时不做，保留扩展接口
  // ON_PAGE_LOAD = 0,  // 未来：OnPageLoad 回调（最高优先级）
  
  QUICK_LINK = 1,      // 快链（包含完整的 FieldValue 和扩展信息）
  URL_PARAMS = 2,      // URL 参数（简单值，需要转换为 FieldValue）
  DEFAULT = 3          // 默认值
}

/**
 * URL 参数初始化源
 */
class URLParamsInitSource implements InitSource {
  priority = InitSourcePriority.URL_PARAMS
  name = 'URLParams'
  
  async initialize(context: InitContext): Promise<InitResult> {
    const { route, functionDetail } = context
    const query = route.query
    
    console.log('🔍 [URLParamsInitSource] 开始初始化', {
      queryKeys: Object.keys(query),
      queryCount: Object.keys(query).length,
      requestFieldsCount: (functionDetail.request || []).length
    })
    
    // 从 URL 解析参数
    const formData: Record<string, FieldValue> = {}
    const requestFields = functionDetail.request || []
    
    requestFields.forEach(field => {
      const queryValue = query[field.code]
      if (queryValue !== undefined && queryValue !== null) {
        let value = Array.isArray(queryValue) ? queryValue[0] : queryValue
        
        // 🔥 URL 解码：如果值是 URL 编码的 JSON 字符串，先解码
        if (typeof value === 'string') {
          try {
            // 尝试 URL 解码
            const decoded = decodeURIComponent(value)
            // 检查是否是 JSON 字符串（以 [ 或 { 开头）
            if ((decoded.startsWith('[') || decoded.startsWith('{')) && decoded !== value) {
              value = decoded
              console.log(`🔍 [URLParamsInitSource] 字段 ${field.code} URL 解码成功`, {
                original: value,
                decoded
              })
            }
          } catch (e) {
            // URL 解码失败，使用原始值
            console.log(`🔍 [URLParamsInitSource] 字段 ${field.code} URL 解码失败，使用原始值`, {
              value,
              error: e
            })
          }
        }
        
        console.log(`🔍 [URLParamsInitSource] 解析字段 ${field.code}`, {
          queryValue,
          value,
          fieldType: field.data?.type || 'string',
          widgetType: (field.widget && 'type' in field.widget) ? field.widget.type : 'unknown'
        })
        
        // 🔥 框架层只负责获取原始值，不进行类型转换
        // 类型转换交给组件初始化器处理（符合依赖倒置原则）
        formData[field.code] = {
          raw: String(value),  // 保持为字符串，让组件自己转换
          display: String(value),
          meta: {
            [FieldValueMeta.FROM_URL]: true,  // 标记来自 URL，需要类型转换
            [FieldValueMeta.ORIGINAL_VALUE]: value  // 保存原始值（可能是字符串、数字、JSON 字符串等）
          }
        }
        const savedFieldValue = formData[field.code]
        console.log(`✅ [URLParamsInitSource] 字段 ${field.code} 原始值已保存`, {
          originalValue: value,
          raw: savedFieldValue?.raw,
          hasFromURLFlag: !!savedFieldValue?.meta?.[FieldValueMeta.FROM_URL]
        })
      }
    })
    
    console.log('✅ [URLParamsInitSource] 初始化完成', {
      formDataKeys: Object.keys(formData),
      formDataCount: Object.keys(formData).length
    })
    
    return { formData }
  }
  
}

/**
 * 快链初始化源
 * 
 * 🔥 从后端加载快链数据，使用完整的 FieldValue 结构
 */
class QuickLinkInitSource implements InitSource {
  priority = InitSourcePriority.QUICK_LINK
  name = 'QuickLink'
  
  async initialize(context: InitContext): Promise<InitResult> {
    const { route, functionDetail } = context
    const quickLinkId = route.query._quicklink_id || route.query._quick_link_id
    
    if (!quickLinkId) {
      return { formData: {} }
    }
    
    try {
      // 1. 调用后端 API 加载快链数据
      const { getQuickLink } = await import('@/api/quicklink')
      const quickLink = await getQuickLink(Number(quickLinkId))
      
      Logger.debug('[QuickLinkInitSource]', '加载快链数据', {
        quickLinkId,
        functionRouter: quickLink.function_router,
        currentRouter: functionDetail?.router || 'undefined'
      })
      
      // 2. 验证快链是否匹配当前函数
      if (functionDetail) {
        if (quickLink.function_router !== functionDetail.router ||
            quickLink.function_method !== functionDetail.method) {
          Logger.warn('[QuickLinkInitSource]', '快链函数不匹配', {
            quickLinkRouter: quickLink.function_router,
            quickLinkMethod: quickLink.function_method,
            currentRouter: functionDetail.router,
            currentMethod: functionDetail.method
          })
          return { formData: {} }
        }
      }
      
      // 3. 恢复 FieldValue 到 formData
      const formData: Record<string, FieldValue> = {}
      Object.keys(quickLink.request_params || {}).forEach(fieldCode => {
        const fieldValue = quickLink.request_params[fieldCode]
        if (fieldValue) {
          // 🔥 确保 FieldValue 结构完整
          formData[fieldCode] = {
            raw: fieldValue.raw,
            display: fieldValue.display || String(fieldValue.raw || ''),
            meta: {
              ...(fieldValue.meta || {}),
              _fromQuickLink: true,  // 标记来自快链
              _quickLinkId: quickLink.id
            }
          }
        }
      })
      
      Logger.debug('[QuickLinkInitSource]', '快链数据恢复完成', {
        formDataKeys: Object.keys(formData),
        formDataCount: Object.keys(formData).length
      })
      
      // 4. 返回初始化结果
      return {
        formData,
        fieldMetadata: quickLink.field_metadata || {},
        metadata: {
          responseParams: quickLink.response_params || null,
          tableState: quickLink.metadata?.table_state,
          chartFilters: quickLink.metadata?.chart_filters,
          ...quickLink.metadata
        }
      }
    } catch (error: any) {
      Logger.error('[QuickLinkInitSource]', '加载快链失败', error)
      return { formData: {} }
    }
  }
}

/**
 * 默认值初始化源
 * 
 * 职责：
 * - 处理 widget.config.default 默认值
 * - 对于没有 URL 参数和快链的字段，使用默认值
 */
class DefaultInitSource implements InitSource {
  priority = InitSourcePriority.DEFAULT
  name = 'Default'
  
  async initialize(context: InitContext): Promise<InitResult> {
    const { functionDetail, currentFormData } = context
    
    console.log('🔍 [DefaultInitSource] 开始初始化', {
      requestFieldsCount: (functionDetail.request || []).length,
      currentFormDataKeys: Object.keys(currentFormData),
      currentFormDataCount: Object.keys(currentFormData).length
    })
    
    const formData: Record<string, FieldValue> = {}
    const requestFields = functionDetail.request || []
    
    // 遍历所有字段，对于没有初始值的字段，使用默认值
    requestFields.forEach(field => {
      // 如果已经有初始值（来自 URL 或快链），跳过
      if (currentFormData.hasOwnProperty(field.code)) {
        console.log(`🔍 [DefaultInitSource] 字段 ${field.code} 已有初始值，跳过默认值初始化`)
        return
      }
      
      // 使用 getWidgetDefaultValue 获取默认值
      const defaultValue = getWidgetDefaultValue(field, undefined, () => useAuthStore())
      
      // 只有当默认值不是空值时才设置
      if (defaultValue.raw !== null && defaultValue.raw !== undefined && defaultValue.raw !== '') {
        formData[field.code] = defaultValue
        console.log(`🔍 [DefaultInitSource] 字段 ${field.code} 使用默认值`, {
          raw: defaultValue.raw,
          display: defaultValue.display,
          widgetType: field.widget?.type,
          hasConfigDefault: !!(field.widget?.config as any)?.default
        })
      } else {
        console.log(`🔍 [DefaultInitSource] 字段 ${field.code} 没有默认值，跳过`)
      }
    })
    
    console.log('✅ [DefaultInitSource] 初始化完成', {
      formDataKeys: Object.keys(formData),
      formDataCount: Object.keys(formData).length
    })
    
    return { formData }
  }
}

/**
 * useFunctionParamInitialization 选项
 */
export interface UseFunctionParamInitializationOptions {
  functionDetail: FunctionDetail | ComputedRef<FunctionDetail | null>  // 🔥 支持直接传入 FunctionDetail 或 ComputedRef
  formDataStore: {
    getValue: (fieldCode: string) => FieldValue | undefined
    setValue: (fieldCode: string, value: FieldValue) => void
    getAllValues: () => Record<string, FieldValue>
    clear: () => void
  }
}

/**
 * 统一数据初始化 Composable
 */
export function useFunctionParamInitialization(
  options: UseFunctionParamInitializationOptions
) {
  const route = useRoute()
  const isInitializing = ref(false)
  
  // 🔥 将 functionDetail 统一转换为 computed，方便后续使用
  const functionDetail = computed(() => {
    const detail = options.functionDetail
    // 如果是 ComputedRef，获取其 value；否则直接使用
    return detail && typeof detail === 'object' && 'value' in detail ? detail.value : detail
  })
  
  // 注册初始化源
  const initSources: InitSource[] = [
    new QuickLinkInitSource(),
    new URLParamsInitSource(),
    new DefaultInitSource()
    // 🔥 OnPageLoad 暂时不做，保留扩展接口
    // new OnPageLoadInitSource()
  ]
  
  /**
   * 初始化函数参数
   * 
   * 流程：
   * 1. 通用初始化（框架负责）：URL/快链加载、类型转换、构建基础 FieldValue
   * 2. 组件自治初始化（组件负责）：调用组件的初始化接口
   * 3. 应用字段元数据（快链特有）
   * 4. 完成初始化，触发 FormEvent.initialized 事件
   * 
   * @returns metadata 元数据（包含 responseParams 等）
   */
  const initialize = async (): Promise<Record<string, any>> => {
    if (isInitializing.value) {
      console.log('🔍 [useFunctionParamInitialization] 正在初始化中，跳过')
      return {}
    }
    
    // 🔥 检查 functionDetail 是否有效（使用 computed 的值）
    const detail = functionDetail.value
    if (!detail || !detail.id) {
      console.log('🔍 [useFunctionParamInitialization] functionDetail 无效，跳过初始化', {
        functionDetail: detail,
        isComputedRef: options.functionDetail && typeof options.functionDetail === 'object' && 'value' in options.functionDetail
      })
      return {}
    }
    
    isInitializing.value = true
    
    try {
      console.log('🔍 [useFunctionParamInitialization] 开始初始化', {
        functionId: detail.id,
        router: detail.router,
        functionName: detail.name,
        requestFieldsCount: (detail.request || []).length,
        currentQuery: route.query,
        currentQueryKeys: Object.keys(route.query)
      })
      
      // 步骤 1：通用初始化（框架负责）
      let currentFormData: Record<string, FieldValue> = {}
      let fieldMetadata: Record<string, any> = {}
      let metadata: Record<string, any> = {}
      
      // 按优先级执行初始化源
      const sortedSources = initSources.sort((a, b) => a.priority - b.priority)
      console.log('🔍 [useFunctionParamInitialization] 初始化源列表', {
        sources: sortedSources.map(s => ({ name: s.name, priority: s.priority })),
        count: sortedSources.length
      })
      
      for (const source of sortedSources) {
        console.log(`🔍 [useFunctionParamInitialization] 执行初始化源: ${source.name}`, {
          priority: source.priority,
          currentFormDataKeys: Object.keys(currentFormData),
          currentFormDataCount: Object.keys(currentFormData).length
        })
        
        const result = await source.initialize({
          functionDetail: detail,  // 🔥 使用解包后的 detail
          currentFormData,
          route
        })
        
        console.log(`🔍 [useFunctionParamInitialization] 初始化源 ${source.name} 完成`, {
          resultFormDataKeys: Object.keys(result.formData),
          resultFormDataCount: Object.keys(result.formData).length,
          hasFieldMetadata: !!result.fieldMetadata,
          fieldMetadataKeys: result.fieldMetadata ? Object.keys(result.fieldMetadata) : [],
          hasMetadata: !!result.metadata,
          metadataKeys: result.metadata ? Object.keys(result.metadata) : []
        })
        
        // 合并数据（后面的优先级更高，会覆盖前面的）
        currentFormData = { ...currentFormData, ...result.formData }
        fieldMetadata = { ...fieldMetadata, ...(result.fieldMetadata || {}) }
        metadata = { ...metadata, ...(result.metadata || {}) }
      }
      
      console.log('🔍 [useFunctionParamInitialization] 通用初始化完成', {
        finalFormDataKeys: Object.keys(currentFormData),
        finalFormDataCount: Object.keys(currentFormData).length,
        finalFormData: currentFormData
      })
      
      // 步骤 2：应用数据到 formDataStore
      Object.keys(currentFormData).forEach(fieldCode => {
        const fieldValue = currentFormData[fieldCode]
        if (fieldValue) {
          options.formDataStore.setValue(fieldCode, fieldValue)
        }
      })
      console.log('🔍 [useFunctionParamInitialization] 数据已应用到 formDataStore', {
        appliedFields: Object.keys(currentFormData)
      })
      
      // 步骤 3：组件自治初始化（组件负责）
      console.log('🔍 [useFunctionParamInitialization] 开始组件自治初始化')
      await triggerWidgetInitialization(currentFormData, fieldMetadata)
      console.log('🔍 [useFunctionParamInitialization] 组件自治初始化完成')
      
      // 步骤 4：应用字段元数据（快链特有，未来实现）
      // applyFieldMetadata(fieldMetadata)
      
      // 步骤 5：触发 FormEvent.initialized 事件
      console.log('🔍 [useFunctionParamInitialization] 触发 FormEvent.initialized 事件')
      eventBus.emit(FormEvent.initialized)
      
      console.log('✅ [useFunctionParamInitialization] 初始化完成', {
        functionId: detail.id,
        router: detail.router,
        initializedFields: Object.keys(currentFormData),
        initializedFieldsCount: Object.keys(currentFormData).length
      })
      
      // 🔥 返回 metadata（包含 responseParams 等）
      return metadata
    } catch (error: any) {
      console.error('❌ [useFunctionParamInitialization] 初始化失败', error)
      Logger.error('[useFunctionParamInitialization]', '初始化失败', error)
      throw error
    } finally {
      isInitializing.value = false
    }
  }
  
  /**
   * 触发组件自治初始化
   * 
   * 🔥 依赖倒置原则：只调用抽象接口，不关心具体组件
   * 
   * @param formData 表单数据
   * @param fieldMetadata 字段元数据
   */
  const triggerWidgetInitialization = async (
    formData: Record<string, FieldValue>,
    fieldMetadata: Record<string, any>
  ): Promise<void> => {
    const detail = functionDetail.value
    if (!detail) {
      console.log('🔍 [triggerWidgetInitialization] functionDetail 无效，跳过组件自治初始化')
      return
    }
    
    const fields = detail.request || []
    
    console.log('🔍 [triggerWidgetInitialization] 开始组件自治初始化', {
      fieldsCount: fields.length,
      fields: fields.map((f: FieldConfig) => ({ 
        code: f.code, 
        widgetType: f.widget?.type, 
        hasCallback: f.callbacks?.includes(FieldCallback.ON_SELECT_FUZZY) 
      }))
    })
    
    // 遍历所有字段，调用组件的初始化接口
    for (const field of fields) {
      const currentValue = options.formDataStore.getValue(field.code)
      if (!currentValue || currentValue.raw === null || currentValue.raw === undefined) {
        console.log(`🔍 [triggerWidgetInitialization] 跳过字段 ${field.code}（没有值）`)
        continue  // 没有值，跳过
      }
      
      console.log(`🔍 [triggerWidgetInitialization] 初始化字段 ${field.code}`, {
        widgetType: field.widget?.type,
        hasCallback: field.callbacks?.includes(FieldCallback.ON_SELECT_FUZZY),
        currentValue: {
          raw: currentValue.raw,
          display: currentValue.display,
          hasDisplayInfo: !!currentValue.meta?.displayInfo
        }
      })
      
      // 🔥 调用抽象接口，组件自己决定是否需要初始化
      const initContext: WidgetInitContext = {
        field,
        currentValue,
        allFormData: formData,
        functionDetail: detail,  // 🔥 使用解包后的 detail
        initSource: route.query._quicklink_id ? 'quicklink' : 'url',
        fieldPath: field.code  // 🔥 顶层字段的路径就是 field.code
      }
      
      try {
        const initializedValue = await widgetInitializerRegistry.initialize(initContext)
        
        // 🔥 判断是否需要更新：即使 raw 相同，如果 display 或 meta 不同，也需要更新
        const needsUpdate = initializedValue !== currentValue || 
                            initializedValue.display !== currentValue.display ||
                            JSON.stringify(initializedValue.meta) !== JSON.stringify(currentValue.meta)
        
        if (needsUpdate) {
          console.log(`✅ [triggerWidgetInitialization] 字段 ${field.code} 初始化完成`, {
            widgetType: field.widget?.type,
            oldValue: {
              raw: currentValue.raw,
              display: currentValue.display,
              hasDisplayInfo: !!currentValue.meta?.displayInfo
            },
            newValue: {
              raw: initializedValue.raw,
              display: initializedValue.display,
              hasDisplayInfo: !!initializedValue.meta?.displayInfo
            }
          })
          options.formDataStore.setValue(field.code, initializedValue)
        } else {
          console.log(`🔍 [triggerWidgetInitialization] 字段 ${field.code} 不需要初始化（组件返回 null 或原始值）`)
        }
      } catch (error: any) {
        console.warn(`⚠️ [triggerWidgetInitialization] 字段 ${field.code} 初始化失败`, {
          widgetType: field.widget?.type,
          error: error?.message || error
        })
        Logger.warn('[useFunctionParamInitialization]', '组件初始化失败', {
          fieldCode: field.code,
          widgetType: field.widget?.type,
          error: error?.message || error
        })
        // 初始化失败不影响其他字段，继续处理下一个字段
      }
    }
    
    console.log('✅ [triggerWidgetInitialization] 组件自治初始化完成', {
      processedFieldsCount: fields.length
    })
  }
  
  return {
    initialize,
    isInitializing
  }
}

