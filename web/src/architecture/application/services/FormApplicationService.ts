/**
 * FormApplicationService - 表单应用服务
 * 
 * 职责：表单业务流程编排
 * - 监听事件，调用 Domain Services
 * - 协调表单初始化和提交流程
 * - 不包含业务逻辑，只负责编排
 * 
 * 特点：
 * - 依赖 Domain Services
 * - 通过事件总线监听和触发事件
 * - 不包含业务逻辑，只负责流程编排
 */

import { FormDomainService } from '../../domain/services/FormDomainService'
import type { IEventBus } from '../../domain/interfaces/IEventBus'
import { WorkspaceEvent, FormEvent } from '../../domain/interfaces/IEventBus'
import type { FieldConfig, FunctionDetail } from '../../domain/types'
import type { IApiClient } from '../../domain/interfaces/IApiClient'

/**
 * 表单应用服务
 */
export class FormApplicationService {
  constructor(
    private domainService: FormDomainService,
    private eventBus: IEventBus,
    private apiClient: IApiClient
  ) {
    this.setupEventHandlers()
  }

  /**
   * 设置事件处理器
   */
  private setupEventHandlers(): void {
    // 监听函数加载完成事件
    this.eventBus.on(WorkspaceEvent.functionLoaded, async (payload: { detail: FunctionDetail }) => {
      if (payload.detail.template_type === 'form') {
        await this.handleFunctionLoaded(payload.detail)
      }
    })

    // 监听字段值更新事件（可以在这里添加额外的业务逻辑）
    this.eventBus.on(FormEvent.fieldValueUpdated, (payload: { fieldCode: string, value: any }) => {
      // 可以在这里添加额外的业务逻辑
      // 例如：自动保存、自动验证等
    })
  }

  /**
   * 处理函数加载完成
   */
  async handleFunctionLoaded(detail: FunctionDetail): Promise<void> {
    // 初始化表单
    const fields = (detail.request || []) as FieldConfig[]
    const initialData = {} // 从 URL 或其他地方获取初始数据
    
    this.domainService.setFields(fields)
    this.domainService.initializeForm(fields, initialData)
  }

  /**
   * 提交表单
   */
  async submitForm(functionDetail: FunctionDetail): Promise<any> {
    // 验证表单
    const fields = (functionDetail.request || []) as FieldConfig[]
    const isValid = this.domainService.validateForm(fields)
    
    if (!isValid) {
      throw new Error('表单验证失败')
    }

    // 设置提交状态
    this.domainService.setSubmitting(true)

    try {
      // 获取提交数据（从 StateManager）
      // 注意：这里需要访问 FormStateManager 的 getSubmitData 方法
      // 为了保持依赖倒置，我们通过 Domain Service 获取
      const submitData = this.getSubmitData(fields)

      // 调用 API
      const url = `/api/v1/run${functionDetail.router}`
      const method = functionDetail.method?.toUpperCase() || 'POST'
      
      let response: any
      if (method === 'GET') {
        response = await this.apiClient.get(url, submitData)
      } else {
        response = await this.apiClient.post(url, submitData)
      }

      // 触发事件
      this.eventBus.emit(FormEvent.submitted, { functionDetail, response })
      this.eventBus.emit(FormEvent.responseReceived, { response })

      return response
    } finally {
      // 重置提交状态
      this.domainService.setSubmitting(false)
    }
  }

  /**
   * 获取提交数据（内部方法）
   * 注意：这里需要访问 StateManager，但为了保持依赖倒置，
   * 我们通过 Domain Service 的状态管理器获取
   */
  private getSubmitData(fields: FieldConfig[]): Record<string, any> {
    // 从 Domain Service 获取状态管理器
    const stateManager = this.domainService.getStateManager()
    
    // 如果 StateManager 有 getSubmitData 方法（FormStateManager 特有），使用它
    if (stateManager && typeof (stateManager as any).getSubmitData === 'function') {
      return (stateManager as any).getSubmitData(fields)
    }
    
    // 否则，从状态中手动提取数据
    const state = stateManager.getState()
    const result: Record<string, any> = {}
    
    fields.forEach(field => {
      const value = state.data.get(field.code)
      if (value) {
        result[field.code] = value.raw
      }
    })
    
    return result
  }

  /**
   * 初始化表单（供外部调用）
   */
  initializeForm(fields: FieldConfig[], initialData?: Record<string, any>): void {
    this.domainService.setFields(fields)
    this.domainService.initializeForm(fields, initialData)
  }

  /**
   * 更新字段值（供外部调用）
   */
  updateFieldValue(fieldCode: string, value: any): void {
    this.domainService.updateFieldValue(fieldCode, value)
  }

  /**
   * 清空表单（供外部调用）
   */
  clearForm(): void {
    this.domainService.clearForm()
  }
}

