/**
 * useFunctionParamInitialization - 统一数据初始化 Composable
 * 
 * ============================================
 * 📋 需求说明
 * ============================================
 * 
 * 1. **数据初始化来源**：
 *    - URL 参数：从 URL 查询参数初始化表单字段
 *    - 默认值：从字段配置的默认值初始化
 *    - 组件自治初始化：组件自己决定是否需要初始化（如 SelectWidget 加载选项）
 * 
 * 2. **初始化顺序**：
 *    - URL 参数优先（priority: 2）
 *    - 默认值其次（priority: 1）
 *    - 组件自治初始化最后（在所有初始化源之后）
 * 
 * 3. **组件自治初始化**：
 *    - 某些组件需要动态初始化（如 SelectWidget 从 API 加载选项）
 *    - 组件实现 `IWidgetInitializer` 接口
 *    - 初始化器可以决定是否需要初始化（返回 null 表示不需要）
 * 
 * ============================================
 * 🎯 设计思路
 * ============================================
 * 
 * 1. **初始化源模式**：
 *    - 使用 `InitSource` 接口定义初始化源
 *    - 每个初始化源有优先级（priority）
 *    - 按优先级顺序执行初始化
 * 
 * 2. **依赖倒置原则**：
 *    - 框架只依赖抽象接口（IWidgetInitializer）
 *    - 不依赖具体组件实现
 *    - 组件可以决定是否需要初始化
 * 
 * 3. **初始化流程**：
 *    - 按优先级执行初始化源
 *    - 每个初始化源可以修改表单数据
 *    - 最后调用组件自治初始化
 * 
 * ============================================
 * 📝 关键功能
 * ============================================
 * 
 * 1. **URLParamsInitSource**：
 *    - 从 URL 查询参数初始化表单字段
 *    - 优先级：2（最高）
 *    - 支持复杂类型的 JSON 反序列化
 * 
 * 2. **DefaultInitSource**：
 *    - 从字段默认值初始化表单字段
 *    - 优先级：1（较低）
 *    - 只在字段没有值时设置默认值
 * 
 * 3. **triggerWidgetInitialization**：
 *    - 调用组件自治初始化
 *    - 遍历所有字段，调用对应的初始化器
 *    - 初始化器可以返回新的 FieldValue 或 null
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **初始化顺序**：
 *    - URL 参数优先，会覆盖默认值
 *    - 组件自治初始化最后，可以覆盖 URL 参数和默认值
 * 
 * 2. **字段类型检查**：
 *    - 确保 `functionDetail.request` 是数组
 *    - 使用 `Array.isArray` 检查，避免类型错误
 * 
 * 3. **组件自治初始化**：
 *    - 组件可以决定是否需要初始化（返回 null）
 *    - 初始化器可以访问所有表单数据（用于依赖字段）
 * 
 * ============================================
 * 📚 相关文档
 * ============================================
 * 
 * - Widget 初始化接口：`web/src/architecture/presentation/widgets/interfaces/IWidgetInitializer.ts`
 * - Widget 初始化器注册表：`web/src/architecture/presentation/widgets/initializers/WidgetInitializerRegistry.ts`
 */

import { ref, computed, type ComputedRef } from 'vue'
import { useRoute } from 'vue-router'
import type { FunctionDetail, FieldConfig } from '../../domain/types'
import type { FieldValue } from '../../domain/types'
import { widgetInitializerRegistry } from '../../presentation/widgets/initializers/WidgetInitializerRegistry'
import type { WidgetInitContext } from '../../presentation/widgets/interfaces/IWidgetInitializer'
import { eventBus, FormEvent } from '../../infrastructure/eventBus'
import { Logger } from '@/core/utils/logger'
import { getWidgetDefaultValue } from '../../presentation/widgets/composables/useWidgetDefaultValue'
import { useAuthStore } from '@/stores/auth'
import { FieldValueMeta, FieldCallback } from '@/core/constants/field'
import { DataType } from '@/core/constants/widget'
import { convertValueByFieldType } from '../../presentation/widgets/utils/typeConverter'

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
      requestFieldsCount: (Array.isArray(functionDetail.request) ? functionDetail.request : []).length
    })
    
    // 从 URL 解析参数
    const formData: Record<string, FieldValue> = {}
    // 🔥 确保 requestFields 是数组，防止类型错误
    const requestFields = Array.isArray(functionDetail.request) ? functionDetail.request : []
    
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
        
        // 🔥 根据字段类型进行转换（URL 参数都是字符串，需要转换为正确的类型）
        const convertedValue = convertValueByFieldType(value, field)
        
        console.log(`🔍 [URLParamsInitSource] 字段 ${field.code} 类型转换`, {
          originalValue: value,
          convertedValue,
          fieldType: field.data?.type || 'string',
          originalType: typeof value,
          convertedType: typeof convertedValue
        })
        
        // 🔥 将转换后的值保存为 FieldValue
        formData[field.code] = {
          raw: convertedValue,  // 使用转换后的值（可能是数字、布尔值等）
          display: String(convertedValue),  // 显示值始终是字符串
          meta: {
            [FieldValueMeta.FROM_URL]: true,  // 标记来自 URL
            [FieldValueMeta.ORIGINAL_VALUE]: value  // 保存原始值（用于调试）
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
      requestFieldsCount: (Array.isArray(functionDetail.request) ? functionDetail.request : []).length,
      currentFormDataKeys: Object.keys(currentFormData),
      currentFormDataCount: Object.keys(currentFormData).length
    })
    
    const formData: Record<string, FieldValue> = {}
    // 🔥 确保 requestFields 是数组，防止类型错误
    const requestFields = Array.isArray(functionDetail.request) ? functionDetail.request : []
    
    // 遍历所有字段，对于没有初始值的字段，使用默认值
    requestFields.forEach(field => {
      // 如果已经有初始值（来自 URL 或快链），跳过
      if (currentFormData.hasOwnProperty(field.code)) {
        console.log(`🔍 [DefaultInitSource] 字段 ${field.code} 已有初始值，跳过默认值初始化`)
        return
      }
      
      // 使用 getWidgetDefaultValue 获取默认值
      const defaultValue = getWidgetDefaultValue(field, undefined, () => useAuthStore())
      
      // 🔥 对于 table 和 form 类型字段，即使默认值是空数组/空对象，也需要设置
      // 因为它们是容器组件，需要初始化为空数组/空对象才能正常工作
      const isContainerWidget = field.widget?.type === 'table' || field.widget?.type === 'form'
      
      // 只有当默认值不是空值时才设置（但容器组件例外）
      if (isContainerWidget || (defaultValue.raw !== null && defaultValue.raw !== undefined && defaultValue.raw !== '')) {
        formData[field.code] = defaultValue
        console.log(`🔍 [DefaultInitSource] 字段 ${field.code} 使用默认值`, {
          raw: defaultValue.raw,
          display: defaultValue.display,
          widgetType: field.widget?.type,
          isContainerWidget,
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
        requestFieldsCount: (Array.isArray(detail.request) ? detail.request : []).length,
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
    
    // 🔥 确保 fields 是数组，防止类型错误
    const fields = Array.isArray(detail.request) ? detail.request : []
    
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
      // 🔥 每次循环都从 formDataStore 获取最新值，确保获取到之前字段初始化后的最新值
      const currentValue = options.formDataStore.getValue(field.code)
      if (!currentValue || currentValue.raw === null || currentValue.raw === undefined) {
        console.log(`🔍 [triggerWidgetInitialization] 跳过字段 ${field.code}（没有值）`)
        continue  // 没有值，跳过
      }
      
      // 🔥 每次循环都从 formDataStore 获取所有字段的最新值，确保 allFormData 包含之前字段初始化后的最新值
      const allFormData = options.formDataStore.getAllValues()
      
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
        allFormData: allFormData,  // 🔥 使用实时获取的最新值
        functionDetail: detail,  // 🔥 使用解包后的 detail
        initSource: 'url',
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

