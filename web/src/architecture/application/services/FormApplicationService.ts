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

import { Logger } from '@/core/utils/logger'
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
    // 🔥 确保 fields 是数组，防止类型错误
    const fields = (Array.isArray(detail.request) ? detail.request : []) as FieldConfig[]
    const initialData = {} // 从 URL 或其他地方获取初始数据
    
    this.domainService.setFields(fields)
    this.domainService.initializeForm(fields, initialData)
  }

  /**
   * 提交表单
   */
  async submitForm(functionDetail: FunctionDetail): Promise<any> {
    // 🔥 不进行前端验证，由后端验证

    // 设置提交状态
    this.domainService.setSubmitting(true)

    try {
      // 获取提交数据（从 StateManager）
      // 注意：这里需要访问 FormStateManager 的 getSubmitData 方法
      // 为了保持依赖倒置，我们通过 Domain Service 获取
      const submitData = this.getSubmitData(fields)

      // ⭐ 使用标准 API：/form/submit/{full-code-path}
      const fullCodePath = functionDetail.router?.startsWith('/') 
        ? functionDetail.router 
        : `/${functionDetail.router || ''}`
      const url = `/workspace/api/v1/form/submit${fullCodePath}`
      const method = functionDetail.method?.toUpperCase() || 'POST'
      
      let response: any
      if (method === 'GET') {
        response = await this.apiClient.get(url, submitData)
      } else {
        response = await this.apiClient.post(url, submitData)
      }
      
      // ⭐ 旧版本（已注释，保留用于参考）
      // const url = `/workspace/api/v1/run${functionDetail.router}`
      // const method = functionDetail.method?.toUpperCase() || 'POST'
      // let response: any
      // if (method === 'GET') {
      //   response = await this.apiClient.get(url, submitData)
      // } else {
      //   response = await this.apiClient.post(url, submitData)
      // }

      // 🔥 保存响应数据到状态管理器
      const stateManager = this.domainService.getStateManager()
      if (stateManager && typeof (stateManager as any).setResponse === 'function') {
        // 处理响应数据：如果 response 不是对象，包装成对象
        const responseData = response && typeof response === 'object' 
          ? response 
          : { result: response }
        
        // 🔥 提取 metadata（从 response._metadata，由 request.ts 响应拦截器附加）
        const metadata = (response as any)?._metadata
        if (metadata && typeof (stateManager as any).setMetadata === 'function') {
          ;(stateManager as any).setMetadata(metadata)
        }
        
        ;(stateManager as any).setResponse(responseData)
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
   * 遵循依赖倒置原则：通过 Domain Service 获取提交数据，而不是直接访问 StateManager
   */
  private getSubmitData(fields: FieldConfig[]): Record<string, any> {
    // 使用 Domain Service 的方法获取提交数据（遵循依赖倒置原则）
    return this.domainService.getSubmitData(fields)
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

