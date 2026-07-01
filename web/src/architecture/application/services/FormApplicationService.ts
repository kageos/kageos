/**
 * FormApplicationService - 表单应用服务
 * 
 * ============================================
 * 📋 需求说明
 * ============================================
 * 
 * 1. **表单初始化**：
 *    - 监听函数加载完成事件，初始化表单
 *    - 协调 Domain Service 初始化表单数据
 *    - 处理初始数据回显（编辑模式）
 * 
 * 2. **表单提交**：
 *    - 验证表单数据
 *    - 提取提交数据（使用 FieldExtractorRegistry）
 *    - 调用 API 提交数据
 *    - 处理提交结果（成功/失败）
 * 
 * 3. **事件协调**：
 *    - 监听 WorkspaceEvent.functionLoaded 事件
 *    - 触发 FormEvent.initialized、FormEvent.submitted 等事件
 *    - 协调 Domain Service 和 Infrastructure Layer
 * 
 * ============================================
 * 🎯 设计思路
 * ============================================
 * 
 * 1. **应用层职责**：
 *    - 不包含业务逻辑，只负责流程编排
 *    - 协调 Domain Services 完成业务流程
 *    - 通过事件总线监听和触发事件
 * 
 * 2. **依赖关系**：
 *    - 依赖 FormDomainService（业务逻辑）
 *    - 依赖 IFormGateway（表单提交）
 *    - 依赖 IEventBus（事件通信）
 * 
 * 3. **数据流**：
 *    - 初始化：事件 → FormApplicationService → FormDomainService → StateManager
 *    - 提交：FormApplicationService → 验证 → 提取数据 → API → 处理结果
 * 
 * ============================================
 * 📝 关键功能
 * ============================================
 * 
 * 1. **handleFunctionLoaded**：
 *    - 监听函数加载完成事件
 *    - 调用 FormDomainService.initializeForm 初始化表单
 *    - 触发 FormEvent.initialized 事件
 * 
 * 2. **submitForm**：
 *    - 验证表单数据（FormDomainService.validateForm）
 *    - 提取提交数据（使用 FieldExtractorRegistry）
 *    - 调用 API 提交数据
 *    - 处理提交结果，触发 FormEvent.submitted 事件
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **数据提取**：
 *    - 使用 FieldExtractorRegistry 提取字段值
 *    - 只提取 `raw` 值，不提取 `display` 值
 *    - 支持嵌套结构（form、table）的递归提取
 * 
 * 2. **验证时机**：
 *    - 提交前验证表单
 *    - 验证失败时抛出错误，不提交数据
 *    - 验证错误使用字段的中文名称
 * 
 * 3. **错误处理**：
 *    - API 错误通过 request.ts 拦截器处理
 *    - 其他错误通过事件或异常抛出
 */

import { unwrapApiResponseData } from '@/architecture/shared/apiError'
import { FormDomainService } from '../../domain/services/FormDomainService'
import type { IEventBus } from '../../domain/interfaces/IEventBus'
import { FormEvent } from '../../domain/interfaces/IEventBus'
import type { FieldConfig, FieldValue, FunctionDetail } from '../../domain/types'
import type { IFormGateway } from '../../domain/interfaces/IFormGateway'
import { isFormStateManager } from '@/architecture/domain/interfaces/IFormStateManager'
import { getFormRequestFields } from '@/architecture/domain/utils/functionSchemaSelectors'

type ResponseWithMetadata = {
  _metadata?: Record<string, unknown>
}

/**
 * 表单应用服务
 */
export class FormApplicationService {
  private unsubscribeFieldValueUpdated?: () => void

  constructor(
    private domainService: FormDomainService,
    private eventBus: IEventBus,
    private formGateway: IFormGateway
  ) {
    this.setupEventHandlers()
  }

  /**
   * 设置事件处理器
   */
  private setupEventHandlers(): void {
    // 监听字段值更新事件（可以在这里添加额外的业务逻辑）
    this.unsubscribeFieldValueUpdated = this.eventBus.on<{ fieldCode: string, value: unknown }>(FormEvent.fieldValueUpdated, () => {
      // 可以在这里添加额外的业务逻辑
      // 例如：自动保存、自动验证等
    })
  }

  dispose(): void {
    if (this.unsubscribeFieldValueUpdated) {
      this.unsubscribeFieldValueUpdated()
      this.unsubscribeFieldValueUpdated = undefined
    }
  }

  /**
   * 处理函数加载完成
   */
  async handleFunctionLoaded(detail: FunctionDetail): Promise<void> {
    // 初始化表单
    const fields = getFormRequestFields(detail) as FieldConfig[]
    const initialData = {} // 从 URL 或其他地方获取初始数据
    
    this.domainService.setFields(fields)
    this.domainService.initializeForm(fields, initialData)
  }

  /**
   * 提交表单
   */
  async submitForm(functionDetail: FunctionDetail): Promise<unknown> {
    const fields = getFormRequestFields(functionDetail) as FieldConfig[]
    // 主提交链路也要先跑前端校验，保证顶层 form 与弹窗/抽屉场景行为一致。
    const isValid = this.domainService.validateForm(fields)
    if (!isValid) {
      throw new Error('请先修正表单校验错误')
    }

    // 设置提交状态
    this.domainService.setSubmitting(true)

    try {
      const submitData = this.getSubmitData(fields)
      let response = await this.formGateway.submitForm({ functionDetail, data: submitData })

      response = unwrapApiResponseData(response, '提交失败，请稍后重试')

      // 保存响应数据到状态管理器
      const stateManager = this.domainService.getStateManager()
      if (isFormStateManager(stateManager)) {
        // 处理响应数据：如果 response 不是对象，包装成对象
        const responseData: Record<string, unknown> = response && typeof response === 'object'
          ? response as Record<string, unknown>
          : { result: response }
        
        // 提取 metadata（从 response._metadata，由 request.ts 响应拦截器附加）
        const metadata = response && typeof response === 'object'
          ? (response as ResponseWithMetadata)._metadata
          : undefined
        if (metadata) {
          stateManager.setMetadata(metadata)
        }
        
        stateManager.setResponse(responseData)
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
  private getSubmitData(fields: FieldConfig[]): Record<string, unknown> {
    // 使用 Domain Service 的方法获取提交数据（遵循依赖倒置原则）
    return this.domainService.getSubmitData(fields)
  }

  /**
   * 初始化表单（供外部调用）
   * @param fields 字段配置列表
   * @param initialData 初始数据（编辑模式）
   * @param isUpdateMode 是否为更新模式（true=更新模式，false=新增模式）
   */
  initializeForm(fields: FieldConfig[], initialData?: Record<string, unknown>, isUpdateMode: boolean = false): void {
    this.domainService.setFields(fields)
    this.domainService.initializeForm(fields, initialData, isUpdateMode)
  }

  /**
   * 更新字段值（供外部调用）
   */
  updateFieldValue(fieldCode: string, value: FieldValue): void {
    this.domainService.updateFieldValue(fieldCode, value)
  }

  /**
   * 清空表单（供外部调用）
   */
  clearForm(): void {
    this.domainService.clearForm()
  }
}
