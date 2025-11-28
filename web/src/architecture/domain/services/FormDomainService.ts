/**
 * FormDomainService - 表单领域服务
 * 
 * 职责：表单相关的业务逻辑
 * - 初始化表单
 * - 更新字段值
 * - 处理字段依赖（depend_on）
 * - 验证表单
 * 
 * 特点：
 * - 依赖接口，不依赖具体实现
 * - 通过事件总线通信
 * - 通过状态管理器管理状态
 */

import type { IStateManager } from '../interfaces/IStateManager'
import type { IEventBus } from '../interfaces/IEventBus'
import { FormEvent } from '../interfaces/IEventBus'
import type { FieldConfig, FieldValue } from '../types'

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
 * 表单领域服务
 */
export class FormDomainService {
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
    // 更新字段配置
    this.fields = fields

    const state = this.stateManager.getState()
    const newData = new Map<string, FieldValue>()

    fields.forEach(field => {
      const fieldCode = field.code
      
      // 🔥 优先级：initialData > 已有值 > 默认值
      // 这样可以保留用户已输入的值，同时支持 URL 参数覆盖
      if (initialData && initialData.hasOwnProperty(fieldCode)) {
        // 优先使用 initialData（URL 参数）
        const rawValue = initialData[fieldCode]
        newData.set(fieldCode, {
          raw: rawValue,
          display: typeof rawValue === 'object' ? JSON.stringify(rawValue) : String(rawValue),
          meta: {}
        })
      } else if (state.data && state.data.has(fieldCode)) {
        // 保留已有值（如果 initialData 中没有该字段）
        const existingValue = state.data.get(fieldCode)
        if (existingValue) {
          newData.set(fieldCode, existingValue)
        } else {
          // 使用默认值
          const defaultValue = this.getDefaultValue(field)
          newData.set(fieldCode, defaultValue)
        }
      } else {
        // 使用默认值
        const defaultValue = this.getDefaultValue(field)
        newData.set(fieldCode, defaultValue)
      }
    })

    // 更新状态
    this.stateManager.setState({
      data: newData,
      errors: new Map(),
      submitting: false
    })

    // 触发事件
    this.eventBus.emit(FormEvent.initialized, { fields, data: newData })
  }

  /**
   * 更新字段值
   */
  updateFieldValue(fieldCode: string, value: FieldValue): void {
    const state = this.stateManager.getState()
    const newData = new Map(state.data)
    newData.set(fieldCode, value)

    // 更新状态
    this.stateManager.setState({ 
      ...state,
      data: newData 
    })

    // 处理字段依赖
    this.handleDependency(fieldCode, newData)

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
   */
  private getDefaultValue(field: FieldConfig): FieldValue {
    // 检查是否有配置的默认值
    const configDefault = field.widget?.config?.default
    if (configDefault !== undefined) {
      return {
        raw: configDefault,
        display: typeof configDefault === 'object' ? JSON.stringify(configDefault) : String(configDefault),
        meta: {}
      }
    }

    // 返回空值
    return { raw: null, display: '', meta: {} }
  }

  /**
   * 验证表单
   */
  validateForm(fields: FieldConfig[]): boolean {
    const state = this.stateManager.getState()
    const errors = new Map<string, ValidationResult[]>()

    // 验证所有字段
    fields.forEach(field => {
      const value = state.data.get(field.code)
      if (value && field.validation) {
        // TODO: 这里应该调用验证引擎
        // 为了简化，暂时不实现具体验证逻辑
        // const fieldErrors = validationEngine.validateField(field, value, fields)
        // if (fieldErrors.length > 0) {
        //   errors.set(field.code, fieldErrors)
        // }
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
   * 设置提交状态
   */
  setSubmitting(submitting: boolean): void {
    const state = this.stateManager.getState()
    this.stateManager.setState({
      ...state,
      submitting
    })
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

