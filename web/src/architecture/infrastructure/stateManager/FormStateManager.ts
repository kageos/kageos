/**
 * FormStateManager - 表单状态管理实现
 * 
 * 职责：基于 Pinia Store 实现表单状态管理
 * 
 * 特点：
 * - 使用现有的 useFormDataStore
 * - 适配 IStateManager 接口
 * - 同步 Pinia Store 和 StateManager 的状态
 */

import { reactive, watch } from 'vue'
import { useFormDataStore } from '@/core/stores-v2/formData'
import { StateManagerImpl } from './StateManagerImpl'
import type { IStateManager } from '../../domain/interfaces/IStateManager'
import type { FieldValue } from '@/architecture/domain/types'

/**
 * 表单状态类型
 */
export interface FormState {
  data: Map<string, FieldValue>
  errors: Map<string, any[]>
  submitting: boolean
  response: Record<string, any> | null // 🔥 新增：响应数据
  metadata: Record<string, any> | null // 🔥 新增：元数据（如 total_cost_mill、trace_id 等）
}

/**
 * 表单状态管理实现
 */
export class FormStateManager extends StateManagerImpl<FormState> implements IStateManager<FormState> {
  private formStore: ReturnType<typeof useFormDataStore>
  private errors = reactive<Map<string, any[]>>(new Map())
  private submitting = reactive({ value: false })

  private response = reactive<{ value: Record<string, any> | null }>({ value: null })
  private metadata = reactive<{ value: Record<string, any> | null }>({ value: null })
  
  // 🔥 防止循环更新的标志
  private isUpdatingFromStore = false

  constructor() {
    // 1. 先调用 super 传递初始空状态
    super({
      data: new Map(),
      errors: new Map(),
      submitting: false,
      response: null,
      metadata: null
    })

    // 2. 初始化 store 和其他属性
    this.formStore = useFormDataStore()
    
    // 3. 立即同步真实状态
    this.updateState()

    // 设置 watch，监听 Pinia Store 的变化
    this.setWatch(() => {
      return {
        data: this.formStore.data,
        errors: this.errors,
        submitting: this.submitting.value,
        response: this.response.value,
        metadata: this.metadata.value
      }
    })

    // 监听 Pinia Store 的变化，同步到 StateManager
    watch(() => this.formStore.data, () => {
      // 🔥 如果正在从 setState 更新 store，跳过 watch，避免循环
      if (this.isUpdatingFromStore) {
        return
      }
      this.updateState()
    }, { deep: true })
  }

  /**
   * 更新状态并通知订阅者
   */
  private updateState(): void {
    const newState: FormState = {
      data: this.formStore.data,
      errors: this.errors,
      submitting: this.submitting.value,
      response: this.response.value,
      metadata: this.metadata.value
    }
    this.setState(newState)
  }

  /**
   * 设置字段值
   */
  setValue(fieldPath: string, value: FieldValue): void {
    this.formStore.setValue(fieldPath, value)
  }

  /**
   * 获取字段值
   */
  getValue(fieldPath: string): FieldValue {
    return this.formStore.getValue(fieldPath)
  }

  /**
   * 设置错误
   */
  setError(fieldCode: string, errors: any[]): void {
    this.errors.set(fieldCode, errors)
    this.updateState()
  }

  /**
   * 清除错误
   */
  clearError(fieldCode: string): void {
    this.errors.delete(fieldCode)
    this.updateState()
  }

  /**
   * 设置提交状态
   */
  setSubmitting(submitting: boolean): void {
    this.submitting.value = submitting
    this.updateState()
  }

  /**
   * 获取提交数据（使用 FieldExtractorRegistry）
   */
  getSubmitData(fields: any[]): Record<string, any> {
    return this.formStore.getSubmitData(fields)
  }

  /**
   * 设置响应数据
   */
  setResponse(response: Record<string, any> | null): void {
    this.response.value = response
    this.updateState()
  }

  /**
   * 获取响应数据
   */
  getResponse(): Record<string, any> | null {
    return this.response.value
  }

  /**
   * 设置元数据
   */
  setMetadata(metadata: Record<string, any> | null): void {
    this.metadata.value = metadata
    this.updateState()
  }

  /**
   * 获取元数据
   */
  getMetadata(): Record<string, any> | null {
    return this.metadata.value
  }

  /**
   * 🔥 重写 setState，确保同步更新 formStore.data
   * 当调用 initializeForm 时，需要将数据同步到 formStore
   */
  setState(newState: FormState): void {
    // 🔥 设置标志，防止 watch 触发循环更新
    this.isUpdatingFromStore = true
    
    try {
      // 🔥 如果 newState.data 存在，同步到 formStore.data
      if (newState.data && newState.data instanceof Map) {
        // 清空 formStore 并设置新值
        this.formStore.clear()
        newState.data.forEach((value, key) => {
          this.formStore.setValue(key, value)
        })
      }
      
      // 同步其他状态
      if (newState.errors) {
        this.errors.clear()
        newState.errors.forEach((errors, key) => {
          this.errors.set(key, errors)
        })
      }
      
      if (newState.submitting !== undefined) {
        this.submitting.value = newState.submitting
      }
      
      if (newState.response !== undefined) {
        this.response.value = newState.response
      }
      
      if (newState.metadata !== undefined) {
        this.metadata.value = newState.metadata
      }
      
      // 调用父类的 setState，触发响应式更新
      super.setState(newState)
    } finally {
      // 🔥 使用 nextTick 确保所有更新完成后再重置标志
      // 这样可以避免在同一个 tick 内触发 watch
      setTimeout(() => {
        this.isUpdatingFromStore = false
      }, 0)
    }
  }

}

