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
import type { FieldValue } from '@/core/types/field'

/**
 * 表单状态类型
 */
export interface FormState {
  data: Map<string, FieldValue>
  errors: Map<string, any[]>
  submitting: boolean
}

/**
 * 表单状态管理实现
 */
export class FormStateManager extends StateManagerImpl<FormState> implements IStateManager<FormState> {
  private formStore = useFormDataStore()
  private errors = reactive<Map<string, any[]>>(new Map())
  private submitting = reactive({ value: false })

  constructor() {
    super({
      data: this.formStore.data, // 直接使用 Pinia Store 的响应式 Map
      errors: this.errors,
      submitting: this.submitting.value
    })

    // 设置 watch，监听 Pinia Store 的变化
    this.setWatch(() => {
      return {
        data: this.formStore.data,
        errors: this.errors,
        submitting: this.submitting.value
      }
    })

    // 监听 Pinia Store 的变化，同步到 StateManager
    watch(() => this.formStore.data, () => {
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
      submitting: this.submitting.value
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

}

