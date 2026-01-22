/**
 * FormDomainService - 表单领域服务
 * 
 * ============================================
 * 📋 需求说明
 * ============================================
 * 
 * 1. **表单初始化**：
 *    - 根据字段配置初始化表单数据
 *    - 支持初始数据回显（编辑模式）
 *    - 支持字段默认值设置
 * 
 * 2. **字段值更新**：
 *    - 更新单个字段的值
 *    - 处理字段依赖关系（`depend_on`）
 *    - 清除字段验证错误（提交时验证，不实时验证）
 * 
 * 3. **表单验证**：
 *    - 提交时验证所有字段
 *    - 支持多种验证规则（required、min、max、email 等）
 *    - 验证错误使用字段的中文名称（`field.name`）
 * 
 * ============================================
 * 🎯 设计思路
 * ============================================
 * 
 * 1. **依赖倒置原则**：
 *    - 依赖 `IStateManager` 接口，不依赖具体实现
 *    - 依赖 `IEventBus` 接口，通过事件总线通信
 *    - 可以轻松替换实现，提高可测试性
 * 
 * 2. **状态管理**：
 *    - 通过 StateManager 管理表单状态（字段值、验证错误）
 *    - 状态变化通过事件总线通知其他组件
 * 
 * 3. **验证引擎**：
 *    - 使用 ValidationEngine 统一管理验证规则
 *    - 支持多种验证器（RequiredValidator、MinValidator 等）
 *    - 验证错误使用字段的中文名称，提升用户体验
 * 
 * ============================================
 * 📝 关键功能
 * ============================================
 * 
 * 1. **initializeForm**：
 *    - 初始化表单字段和初始数据
 *    - 优先使用 `initialData`（编辑模式）
 *    - 如果没有初始数据，使用字段默认值
 * 
 * 2. **updateFieldValue**：
 *    - 更新字段值并清除该字段的验证错误
 *    - 不进行实时验证（只在提交时验证）
 *    - 触发 `FormEvent.fieldValueUpdated` 事件
 * 
 * 3. **validateForm**：
 *    - 验证所有字段
 *    - 返回验证结果和错误信息
 *    - 验证错误使用字段的中文名称
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **初始数据优先级**：
 *    - `initialData` > 字段默认值
 *    - 编辑模式下必须提供 `initialData`
 * 
 * 2. **验证时机**：
 *    - 只在提交时验证，不进行实时验证
 *    - 字段更新时只清除该字段的错误，不重新验证
 * 
 * 3. **字段依赖**：
 *    - 支持 `depend_on` 字段依赖关系
 *    - 依赖字段变化时，自动更新被依赖字段
 */

import type { IStateManager } from '../interfaces/IStateManager'
import type { IEventBus } from '../interfaces/IEventBus'
import { FormEvent } from '../interfaces/IEventBus'
import type { FieldConfig, FieldValue } from '../types'
import { ValidationEngine, createDefaultValidatorRegistry } from '@/core/validation'
import type { ReactiveFormDataManager } from '@/core/managers/ReactiveFormDataManager'
import { Logger } from '@/core/utils/logger'
import { getWidgetDefaultValue } from '@/architecture/presentation/widgets/composables/useWidgetDefaultValue'
import { useAuthStore } from '@/stores/auth'

/**
 * 验证结果类型（简化，实际应该从 validation 导入）
 */
export interface ValidationResult {
  message: string
  field: string
  [key: string]: any
}

/**
 * 表单状态
 */
export interface FormState {
  data: Map<string, FieldValue>
  errors: Map<string, ValidationResult[]>
  submitting: boolean
}

/**
 * FormStateManager 适配器（用于 ValidationEngine）
 * 将 IStateManager 适配为 ValidationEngine 需要的接口
 * 
 * ValidationEngine 只需要 formManager.getValue() 方法，用于条件验证
 */
class FormStateManagerAdapter {
  constructor(private stateManager: IStateManager<FormState>) {}
  
  /**
   * 获取字段值（ValidationEngine 主要使用此方法，用于条件验证）
   */
  getValue(fieldPath: string): FieldValue {
    const state = this.stateManager.getState()
    return state.data.get(fieldPath) || { raw: null, display: '', meta: {} }
  }
}

/**
 * 表单领域服务
 */
export class FormDomainService {
  private validationEngine: ValidationEngine | null = null
  
  constructor(
    private stateManager: IStateManager<FormState>,
    private eventBus: IEventBus,
    private fields: FieldConfig[] = [] // 字段配置（用于处理依赖）
  ) {}

  /**
   * 设置字段配置（用于处理依赖）
   */
  setFields(fields: FieldConfig[]): void {
    this.fields = fields
  }

  /**
   * 初始化表单
   */
  initializeForm(fields: FieldConfig[], initialData?: Record<string, any>): void {
    Logger.debug('FormDomainService', 'initializeForm 被调用', {
      fieldsCount: fields.length,
      fieldCodes: fields.map(f => f.code),
      initialDataKeys: initialData ? Object.keys(initialData) : []
    })
    
    // 更新字段配置
    this.fields = fields

    // 🔥 关键修复：从 formStore.data 获取当前数据，而不是从 state.data
    // 因为刷新后 state.data 可能是空的，但 formStore.data 可能有数据（从 URL 参数恢复或用户输入）
    const stateManager = this.stateManager as any
    let currentData: Map<string, FieldValue>
    
    if (stateManager && stateManager.formStore && stateManager.formStore.data) {
      // 从 formStore.data 获取当前数据（这是真实的数据源）
      currentData = new Map(stateManager.formStore.data)
    } else {
      // 如果 formStore 不可用，从 state 获取（向后兼容）
      const state = this.stateManager.getState()
      currentData = new Map(state.data || new Map())
    }

    const state = this.stateManager.getState()
    const newData = new Map<string, FieldValue>()

    fields.forEach(field => {
      const fieldCode = field.code
      // 🔥 优先从 currentData（formStore.data）获取，如果没有则从 state.data 获取
      const existingValue = currentData.get(fieldCode) || state.data?.get(fieldCode)
      const hasInitialData = initialData && initialData.hasOwnProperty(fieldCode)
      const initialRawValue = hasInitialData ? initialData[fieldCode] : undefined
      
      // 🔥 优先级：已有完整值（包含 display）> initialData > 已有值（只有 raw）> 默认值
      // 这样可以保留 SelectWidgetInitializer 更新后的完整 FieldValue（包含 display）
      
      // 1. 如果已有值且 display 存在且不等于 raw，说明已经通过 SelectWidgetInitializer 初始化过了
      // 此时应该保留这个完整值，即使 initialData 中有该字段
      if (existingValue && 
          existingValue.display && 
          String(existingValue.display) !== String(existingValue.raw) &&
          existingValue.display !== '') {
        newData.set(fieldCode, existingValue)
        return  // 保留完整值，跳过后续处理
      }
      
      // 2. 如果 initialData 中有该字段，使用 initialData（但保留已有的 display 和 meta）
      if (hasInitialData) {
        // 如果 raw 值相同，保留已有的 display 和 meta（可能已经通过 SelectWidgetInitializer 初始化）
        if (existingValue && existingValue.raw === initialRawValue) {
          // 🔥 标记该字段来自 initialData（编辑模式），确保默认值不会覆盖
          newData.set(fieldCode, {
            ...existingValue,
            meta: {
              ...existingValue.meta,
              fromInitialData: true
            }
          })
        } else {
          // 🔥 对于有 OnSelectFuzzy 回调的字段，display 暂时设置为空字符串
          // 让 SelectWidgetInitializer 通过 by_value 来获取 label
          const hasOnSelectFuzzy = field.callbacks?.includes('OnSelectFuzzy') || false
          newData.set(fieldCode, {
            raw: initialRawValue,
            display: hasOnSelectFuzzy ? '' : (typeof initialRawValue === 'object' ? JSON.stringify(initialRawValue) : String(initialRawValue)),
            meta: {
              fromInitialData: true  // 🔥 标记该字段来自 initialData（编辑模式）
            }
          })
        }
        return
      }
      
      // 3. 保留已有值（如果 initialData 中没有该字段）
      // 🔥 关键修复：如果 existingValue 是空值（raw 为 null/undefined/空字符串），且没有 initialData，应该使用默认值
      // 这样可以确保新增模式下默认值能生效，而更新模式下不会覆盖 initialData
      if (existingValue) {
        // 检查 existingValue 是否是空值
        const isEmptyValue = existingValue.raw === null || 
                            existingValue.raw === undefined || 
                            existingValue.raw === '' ||
                            (Array.isArray(existingValue.raw) && existingValue.raw.length === 0) ||
                            (typeof existingValue.raw === 'object' && Object.keys(existingValue.raw).length === 0)
        
        // 如果是空值且没有 initialData（新增模式），使用默认值
        // 如果有 initialData（编辑模式），保留 existingValue（可能是空值，但 initialData 会覆盖）
        if (isEmptyValue && !initialData) {
          // 新增模式且值为空，使用默认值
          const defaultValue = this.getDefaultValue(field)
          newData.set(fieldCode, defaultValue)
          return
        }
        
        // 非空值或编辑模式，保留已有值
        newData.set(fieldCode, existingValue)
        return
      }
      
      // 4. 使用默认值（没有 existingValue 且没有 initialData）
      const defaultValue = this.getDefaultValue(field)
      newData.set(fieldCode, defaultValue)
    })

    // 更新状态
    this.stateManager.setState({
      data: newData,
      errors: new Map(),
      submitting: false
    })

    Logger.debug('FormDomainService', 'initializeForm 完成', {
      fieldsCount: fields.length,
      newDataSize: newData.size,
      newDataKeys: Array.from(newData.keys())
    })

    // 触发事件
    this.eventBus.emit(FormEvent.initialized, { fields, data: newData })
  }

  /**
   * 更新字段值
   * 🔥 移除实时验证，只在提交时验证
   * 🔥 更新字段值时，立即清除该字段的所有错误，避免显示过时的错误消息
   */
  updateFieldValue(fieldCode: string, value: FieldValue): void {
    // 🔥 关键修复：直接从 formStore 获取当前数据，而不是从 state.data
    // 因为刷新后 state.data 可能是空的，但 formStore.data 可能有数据（从 URL 参数恢复）
    const stateManager = this.stateManager as any
    let currentData: Map<string, FieldValue>
    
    if (stateManager && stateManager.formStore && stateManager.formStore.data) {
      // 从 formStore.data 获取当前数据（这是真实的数据源）
      // 🔥 创建新的 Map，确保不会修改原始 Map
      currentData = new Map(stateManager.formStore.data)
    } else {
      // 如果 formStore 不可用，从 state 获取（向后兼容）
      const state = this.stateManager.getState()
      currentData = new Map(state.data || new Map())
    }
    
    // 🔥 调试日志：检查更新前的数据
    Logger.debug('FormDomainService', 'updateFieldValue 开始', {
      fieldCode,
      valueRaw: value?.raw,
      currentDataSize: currentData.size,
      currentDataKeys: Array.from(currentData.keys())
    })
    
    // 更新字段值
    currentData.set(fieldCode, value)

    // 🔥 更新字段值时，立即清除该字段的所有错误（不进行实时验证）
    // 验证只在提交时进行，避免在输入/选择时显示错误
    const state = this.stateManager.getState()
    const newErrors = new Map(state.errors)
    newErrors.delete(fieldCode)  // 清除该字段的所有错误

    // 🔥 更新状态：只传递 data 和 errors，不传递其他字段，避免覆盖
    // setState 会合并更新，不会清空 formStore.data
    this.stateManager.setState({ 
      data: currentData,
      errors: newErrors
    } as any)

    // 🔥 调试日志：检查更新后的数据
    Logger.debug('FormDomainService', 'updateFieldValue 完成', {
      fieldCode,
      valueRaw: value?.raw,
      formStoreDataSize: stateManager?.formStore?.data?.size || 0,
      formStoreDataKeys: stateManager?.formStore?.data ? Array.from(stateManager.formStore.data.keys()) : []
    })

    // 处理字段依赖
    this.handleDependency(fieldCode, currentData)

    // 触发事件
    this.eventBus.emit(FormEvent.fieldValueUpdated, { fieldCode, value })
  }

  /**
   * 处理字段依赖（depend_on）
   */
  private handleDependency(fieldCode: string, data: Map<string, FieldValue>): void {
    // 查找依赖该字段的其他字段
    this.fields.forEach(field => {
      if (field.depend_on === fieldCode) {
        // 清空依赖字段的值
        const clearedValue: FieldValue = {
          raw: null,
          display: '',
          meta: {}
        }
        
        const newData = new Map(data)
        newData.set(field.code, clearedValue)
        
        // 更新状态
        const state = this.stateManager.getState()
        this.stateManager.setState({
          ...state,
          data: newData
        })

        // 清除错误
        const newErrors = new Map(state.errors)
        newErrors.delete(field.code)
        this.stateManager.setState({
          ...state,
          errors: newErrors
        })
      }
    })
  }

  /**
   * 获取默认值
   * 🔥 使用 getWidgetDefaultValue 来解析动态默认值（如 Me(), MyDepartment() 等）
   */
  private getDefaultValue(field: FieldConfig): FieldValue {
    // 🔥 使用 getWidgetDefaultValue 来获取默认值，它会自动解析动态默认值（如 Me(), MyDepartment()）
    return getWidgetDefaultValue(field, undefined, () => useAuthStore())
  }

  /**
   * 验证表单
   */
  validateForm(fields: FieldConfig[]): boolean {
    const state = this.stateManager.getState()
    const errors = new Map<string, ValidationResult[]>()

    // 初始化验证引擎（如果还没有初始化或字段配置变化）
    if (!this.validationEngine || this.fields !== fields) {
      const registry = createDefaultValidatorRegistry()
      const formManagerAdapter = new FormStateManagerAdapter(this.stateManager)
      // 类型断言：适配器实现了 ValidationEngine 需要的接口
      this.validationEngine = new ValidationEngine(
        registry,
        formManagerAdapter as any as ReactiveFormDataManager,
        fields
      )
      this.fields = fields
    }

    // 验证所有字段
    fields.forEach(field => {
      const value = state.data.get(field.code) || { raw: null, display: '', meta: {} }
      if (field.validation) {
        const fieldErrors = this.validationEngine!.validateField(field, value, fields)
        if (fieldErrors.length > 0) {
          errors.set(field.code, fieldErrors)
        }
      }
    })

    // 更新状态
    this.stateManager.setState({ 
      ...state,
      errors 
    })

    // 触发事件
    this.eventBus.emit(FormEvent.validated, { errors })

    return errors.size === 0
  }

  /**
   * 获取字段值
   */
  getFieldValue(fieldCode: string): FieldValue {
    const state = this.stateManager.getState()
    return state.data.get(fieldCode) || { raw: null, display: '', meta: {} }
  }

  /**
   * 获取字段错误
   */
  getFieldError(fieldCode: string): ValidationResult[] {
    const state = this.stateManager.getState()
    return state.errors.get(fieldCode) || []
  }

  /**
   * 获取提交数据（供 Application Layer 使用，遵循依赖倒置原则）
   * 🔥 委托给 StateManager，使用 FieldExtractorRegistry 进行递归提取
   */
  getSubmitData(fields: FieldConfig[]): Record<string, any> {
    // 🔥 委托给 FormStateManager.getSubmitData()，它会使用 FieldExtractorRegistry
    const stateManager = this.stateManager as any
    if (stateManager && typeof stateManager.getSubmitData === 'function') {
      return stateManager.getSubmitData(fields)
    }
    
    Logger.warn('FormDomainService', 'stateManager.getSubmitData 方法不存在，返回空对象')
    return {}
  }

  /**
   * 设置提交状态
   * 🔥 只更新 submitting 字段，不更新 data，避免清空表单数据
   */
  setSubmitting(submitting: boolean): void {
    // 🔥 直接调用 StateManager 的 setSubmitting 方法，而不是 setState
    // 这样可以避免传递整个 state 对象，防止意外清空数据
    const stateManager = this.stateManager as any
    if (stateManager && typeof stateManager.setSubmitting === 'function') {
      stateManager.setSubmitting(submitting)
    } else {
      // 如果 StateManager 没有 setSubmitting 方法，使用 setState 但只传递 submitting
      // ⚠️ 注意：不传递 data 字段，这样 setState 不会清空数据
      const state = this.stateManager.getState()
      this.stateManager.setState({
        ...state,
        submitting,
        // 🔥 不传递 data 字段，保持原有数据不变
        data: undefined as any
      } as any)
    }
  }

  /**
   * 清空表单
   */
  clearForm(): void {
    const stateManager = this.stateManager as any
    // 清空响应数据
    if (stateManager && typeof stateManager.setResponse === 'function') {
      stateManager.setResponse(null)
    }
    
    this.stateManager.setState({
      data: new Map(),
      errors: new Map(),
      submitting: false,
      response: null
    })
  }

  /**
   * 获取状态管理器（供 Application Service 使用）
   */
  getStateManager(): IStateManager<FormState> {
    return this.stateManager
  }
}

