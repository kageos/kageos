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
 *    - 从 `functionDetail.schema.form.request` 读取表单字段
 *    - selector 统一兜底为空数组，避免类型错误
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
import type { FunctionDetail } from '../../domain/types'
import type { FieldValue } from '../../domain/types'
import { widgetInitializerRegistry } from '../../presentation/widgets/initializers/WidgetInitializerRegistry'
import type { WidgetInitContext } from '../../presentation/widgets/interfaces/IWidgetInitializer'
import { eventBus, FormEvent } from '../../infrastructure/eventBus'
import { Logger } from '@/core/utils/logger'
import { getWidgetDefaultValue } from '../../presentation/widgets/composables/useWidgetDefaultValue'
import { useAuthStore } from '@/stores/auth'
import { FieldValueMeta } from '@/core/constants/field'
import { convertValueByFieldType } from '../../presentation/widgets/utils/typeConverter'
import { getScopedFieldQueryValue, shouldAllowLegacyFormDraftFallback } from '@/utils/queryFieldNamespace'
import { getFormRequestFields } from '@/utils/functionSchemaSelectors'

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

    // 从 URL 解析参数
    const formData: Record<string, FieldValue> = {}
    const requestFields = getFormRequestFields(functionDetail)
    const fallbackToLegacyRaw = shouldAllowLegacyFormDraftFallback(query as Record<string, any>)
    
    requestFields.forEach(field => {
      const queryValue = getScopedFieldQueryValue(query, field.code, 'form', {
        fallbackToLegacyRaw
      })
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
            }
          } catch {
            // URL 解码失败，使用原始值
          }
        }

        // 🔥 根据字段类型进行转换（URL 参数都是字符串，需要转换为正确的类型）
        const convertedValue = convertValueByFieldType(value, field)

        // 🔥 将转换后的值保存为 FieldValue
        formData[field.code] = {
          raw: convertedValue,  // 使用转换后的值（可能是数字、布尔值等）
          display: String(convertedValue),  // 显示值始终是字符串
          meta: {
            [FieldValueMeta.FROM_URL]: true,  // 标记来自 URL
            [FieldValueMeta.ORIGINAL_VALUE]: value  // 保存原始值（用于调试）
          }
        }
      }
    })

    return { formData }
  }
  
}

/**
 * 默认值初始化源
 * 
 * 职责：
 * - 处理 widget.config.render_default 渲染默认值
 * - 对于没有 URL 参数和初始值的字段，使用默认值
 */
class DefaultInitSource implements InitSource {
  priority = InitSourcePriority.DEFAULT
  name = 'Default'
  
  async initialize(context: InitContext): Promise<InitResult> {
    const { functionDetail, currentFormData } = context

    const formData: Record<string, FieldValue> = {}
    const requestFields = getFormRequestFields(functionDetail)
    
    // 遍历所有字段，对于没有初始值的字段，使用默认值
    requestFields.forEach(field => {
      // 如果已经有初始值（来自 URL 或其他初始化源），跳过
      if (currentFormData.hasOwnProperty(field.code)) {
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
      }
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

  const hydrateCurrentWidgetDisplays = async (initSource: 'url' | 'default' | 'initialData' = 'url'): Promise<void> => {
    const detail = functionDetail.value
    if (!detail) {
      return
    }

    const fields = getFormRequestFields(detail)

    for (const field of fields) {
      const currentValue = options.formDataStore.getValue(field.code)
      if (!currentValue || currentValue.raw === null || currentValue.raw === undefined) {
        continue
      }

      const allFormData = options.formDataStore.getAllValues()
      const initContext: WidgetInitContext = {
        field,
        currentValue,
        allFormData,
        formDataStore: options.formDataStore,
        functionDetail: detail,
        initSource,
        fieldPath: field.code
      }

      try {
        const initializedValue = await widgetInitializerRegistry.initialize(initContext)
        const needsUpdate = initializedValue !== currentValue ||
          initializedValue.display !== currentValue.display ||
          JSON.stringify(initializedValue.meta) !== JSON.stringify(currentValue.meta)

        if (needsUpdate) {
          options.formDataStore.setValue(field.code, initializedValue)
        }
      } catch (error: any) {
        Logger.warn('[useFunctionParamInitialization]', '组件初始化失败', {
          fieldCode: field.code,
          widgetType: field.widget?.type,
          initSource,
          error: error?.message || error
        })
      }
    }
  }
  
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
   * 1. 通用初始化（框架负责）：URL/默认值加载、类型转换、构建基础 FieldValue
   * 2. 组件自治初始化（组件负责）：调用组件的初始化接口
   * 3. 应用字段元数据（保留扩展点）
   * 4. 完成初始化，触发 FormEvent.initialized 事件
   * 
   * @returns metadata 元数据（包含 responseParams 等）
   */
  const initialize = async (): Promise<Record<string, any>> => {
    if (isInitializing.value) {
      return {}
    }
    
    // 🔥 检查 functionDetail 是否有效（使用 computed 的值）
    const detail = functionDetail.value
    if (!detail || detail.id === undefined || detail.id === null) {
      return {}
    }
    
    isInitializing.value = true
    
    try {
      // 步骤 1：通用初始化（框架负责）
      let currentFormData: Record<string, FieldValue> = {}
      let fieldMetadata: Record<string, any> = {}
      let metadata: Record<string, any> = {}
      
      // 按优先级执行初始化源
      const sortedSources = initSources.sort((a, b) => a.priority - b.priority)
      
      for (const source of sortedSources) {
        const result = await source.initialize({
          functionDetail: detail,  // 🔥 使用解包后的 detail
          currentFormData,
          route
        })

        // 合并数据（后面的优先级更高，会覆盖前面的）
        currentFormData = { ...currentFormData, ...result.formData }
        fieldMetadata = { ...fieldMetadata, ...(result.fieldMetadata || {}) }
        metadata = { ...metadata, ...(result.metadata || {}) }
      }

      // 步骤 2：应用数据到 formDataStore
      Object.keys(currentFormData).forEach(fieldCode => {
        const fieldValue = currentFormData[fieldCode]
        if (fieldValue) {
          options.formDataStore.setValue(fieldCode, fieldValue)
        }
      })

      // 步骤 3：组件自治初始化（组件负责）
      await triggerWidgetInitialization(currentFormData, fieldMetadata)
      
      // 步骤 4：应用字段元数据（保留扩展点，未来实现）
      // applyFieldMetadata(fieldMetadata)
      
      // 步骤 5：触发 FormEvent.initialized 事件
      eventBus.emit(FormEvent.initialized)

      // 🔥 返回 metadata（包含 responseParams 等）
      return metadata
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
    _formData: Record<string, FieldValue>,
    _fieldMetadata: Record<string, any>
  ): Promise<void> => {
    await hydrateCurrentWidgetDisplays('url')
  }
  
  return {
    initialize,
    hydrateCurrentWidgetDisplays,
    isInitializing
  }
}
