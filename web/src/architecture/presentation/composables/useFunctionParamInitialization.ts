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

import { ref } from 'vue'
import { useRoute } from 'vue-router'
import type { FunctionDetail } from '../../../core/types/field'
import type { FieldValue } from '../../../core/types/field'
import { widgetInitializerRegistry } from '../../../core/widgets-v2/initializers/WidgetInitializerRegistry'
import type { WidgetInitContext } from '../../../core/widgets-v2/interfaces/IWidgetInitializer'
import { eventBus, FormEvent } from '../../infrastructure/eventBus'
import { Logger } from '../../../core/utils/logger'

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
    
    // 从 URL 解析参数
    const formData: Record<string, FieldValue> = {}
    const requestFields = functionDetail.request || []
    
    requestFields.forEach(field => {
      const queryValue = query[field.code]
      if (queryValue !== undefined && queryValue !== null) {
        const value = Array.isArray(queryValue) ? queryValue[0] : queryValue
        formData[field.code] = this.convertToFieldValue(value, field)
      }
    })
    
    return { formData }
  }
  
  /**
   * 将简单值转换为 FieldValue 结构
   */
  private convertToFieldValue(value: any, field: any): FieldValue {
    // 类型转换
    let rawValue: any = value
    if (field.data?.type === 'int' || field.data?.type === 'integer') {
      rawValue = parseInt(String(value), 10)
    } else if (field.data?.type === 'float' || field.data?.type === 'number') {
      rawValue = parseFloat(String(value))
    } else if (field.data?.type === 'bool' || field.data?.type === 'boolean') {
      rawValue = String(value) === 'true' || String(value) === '1'
    }
    
    return {
      raw: rawValue,
      display: String(value),  // URL 参数只有简单值，display 暂时等于 raw
      dataType: field.data?.type,
      widgetType: field.widget?.type,
      meta: {}  // URL 参数没有 meta 信息，后续由组件初始化补充
    }
  }
}

/**
 * 快链初始化源
 * 
 * 🔥 暂时不做，保留扩展接口
 * 未来实现：从后端加载快链数据，使用完整的 FieldValue 结构
 */
class QuickLinkInitSource implements InitSource {
  priority = InitSourcePriority.QUICK_LINK
  name = 'QuickLink'
  
  async initialize(context: InitContext): Promise<InitResult> {
    const { route } = context
    const quickLinkId = route.query._quicklink_id || route.query._quick_link_id
    
    if (!quickLinkId) {
      return { formData: {} }
    }
    
    // 🔥 TODO: 未来实现快链加载
    // const quickLink = await loadQuickLink(String(quickLinkId))
    // return {
    //   formData: quickLink.request_params || {},
    //   fieldMetadata: quickLink.field_metadata || {},
    //   metadata: {
    //     responseParams: quickLink.response_params,
    //     tableState: quickLink.table_state,
    //     chartFilters: quickLink.chart_filters,
    //     ...quickLink.metadata
    //   }
    // }
    
    Logger.debug('[QuickLinkInitSource]', '快链功能暂未实现', { quickLinkId })
    return { formData: {} }
  }
}

/**
 * 默认值初始化源
 */
class DefaultInitSource implements InitSource {
  priority = InitSourcePriority.DEFAULT
  name = 'Default'
  
  async initialize(context: InitContext): Promise<InitResult> {
    // 默认值初始化：返回空数据，由组件自己处理默认值
    return { formData: {} }
  }
}

/**
 * useFunctionParamInitialization 选项
 */
export interface UseFunctionParamInitializationOptions {
  functionDetail: FunctionDetail
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
   */
  const initialize = async (): Promise<void> => {
    if (isInitializing.value) {
      Logger.debug('[useFunctionParamInitialization]', '正在初始化中，跳过')
      return
    }
    
    isInitializing.value = true
    
    try {
      Logger.debug('[useFunctionParamInitialization]', '开始初始化', {
        functionId: options.functionDetail.id,
        router: options.functionDetail.router
      })
      
      // 步骤 1：通用初始化（框架负责）
      let currentFormData: Record<string, FieldValue> = {}
      let fieldMetadata: Record<string, any> = {}
      
      // 按优先级执行初始化源
      const sortedSources = initSources.sort((a, b) => a.priority - b.priority)
      for (const source of sortedSources) {
        const result = await source.initialize({
          functionDetail: options.functionDetail,
          currentFormData,
          route
        })
        
        // 合并数据（后面的优先级更高，会覆盖前面的）
        currentFormData = { ...currentFormData, ...result.formData }
        fieldMetadata = { ...fieldMetadata, ...(result.fieldMetadata || {}) }
      }
      
      // 步骤 2：应用数据到 formDataStore
      Object.keys(currentFormData).forEach(fieldCode => {
        options.formDataStore.setValue(fieldCode, currentFormData[fieldCode])
      })
      
      // 步骤 3：组件自治初始化（组件负责）
      await triggerWidgetInitialization(currentFormData, fieldMetadata)
      
      // 步骤 4：应用字段元数据（快链特有，未来实现）
      // applyFieldMetadata(fieldMetadata)
      
      // 步骤 5：触发 FormEvent.initialized 事件
      eventBus.emit(FormEvent.initialized)
      
      Logger.debug('[useFunctionParamInitialization]', '初始化完成', {
        functionId: options.functionDetail.id,
        initializedFields: Object.keys(currentFormData)
      })
    } catch (error: any) {
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
    const fields = options.functionDetail.request || []
    
    // 遍历所有字段，调用组件的初始化接口
    for (const field of fields) {
      const currentValue = options.formDataStore.getValue(field.code)
      if (!currentValue || currentValue.raw === null || currentValue.raw === undefined) {
        continue  // 没有值，跳过
      }
      
      // 🔥 调用抽象接口，组件自己决定是否需要初始化
      const initContext: WidgetInitContext = {
        field,
        currentValue,
        allFormData: formData,
        functionDetail: options.functionDetail,
        initSource: route.query._quicklink_id ? 'quicklink' : 'url'
      }
      
      try {
        const initializedValue = await widgetInitializerRegistry.initialize(initContext)
        
        // 如果组件返回了新的值，更新 formDataStore
        if (initializedValue !== currentValue) {
          options.formDataStore.setValue(field.code, initializedValue)
          Logger.debug('[useFunctionParamInitialization]', '组件初始化完成', {
            fieldCode: field.code,
            widgetType: field.widget?.type,
            hasDisplay: !!initializedValue.display,
            hasDisplayInfo: !!initializedValue.meta?.displayInfo
          })
        }
      } catch (error: any) {
        Logger.warn('[useFunctionParamInitialization]', '组件初始化失败', {
          fieldCode: field.code,
          widgetType: field.widget?.type,
          error: error?.message || error
        })
        // 初始化失败不影响其他字段，继续处理下一个字段
      }
    }
  }
  
  return {
    initialize,
    isInitializing
  }
}

