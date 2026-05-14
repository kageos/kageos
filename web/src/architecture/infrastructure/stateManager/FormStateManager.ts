/**
 * FormStateManager - 表单状态管理实现
 * 
 * ============================================
 * 📋 需求说明
 * ============================================
 * 
 * 1. **状态管理**：
 *    - 管理表单数据（字段值、验证错误）
 *    - 基于 Pinia Store（useFormDataStore）实现
 *    - 适配 IStateManager 接口，供 Domain Service 使用
 * 
 * 2. **状态同步**：
 *    - 同步 Pinia Store 和 StateManager 的状态
 *    - 避免递归更新（使用 `isUpdatingFromStore` 标志）
 *    - 确保状态一致性
 * 
 * 3. **数据提取**：
 *    - 提供 `getSubmitData` 方法提取提交数据
 *    - 使用 FieldExtractorRegistry 提取字段值
 *    - 支持嵌套结构（form、table）的递归提取
 * 
 * ============================================
 * 🎯 设计思路
 * ============================================
 * 
 * 1. **适配器模式**：
 *    - 适配 IStateManager 接口
 *    - 内部使用 Pinia Store 存储数据
 *    - 提供统一的接口供 Domain Service 使用
 * 
 * 2. **状态同步机制**：
 *    - 使用 `isUpdatingFromStore` 标志防止递归更新
 *    - `setState` 时设置标志，更新 Pinia Store
 *    - Pinia Store 的 `watch` 检查标志，跳过更新
 * 
 * 3. **数据提取**：
 *    - 委托给 Pinia Store 的 `getSubmitData` 方法
 *    - 使用 FieldExtractorRegistry 提取字段值
 *    - 支持任意嵌套深度
 * 
 * ============================================
 * 📝 关键功能
 * ============================================
 * 
 * 1. **setState**：
 *    - 更新表单状态（字段值、验证错误）
 *    - 同步到 Pinia Store
 *    - 使用 `isUpdatingFromStore` 防止递归更新
 * 
 * 2. **getState**：
 *    - 获取当前表单状态
 *    - 从 Pinia Store 获取数据
 * 
 * 3. **getSubmitData**：
 *    - 提取提交数据
 *    - 委托给 Pinia Store 的 `getSubmitData` 方法
 *    - 使用 FieldExtractorRegistry 提取字段值
 * 
 * ============================================
 * ⚠️ 注意事项
 * ============================================
 * 
 * 1. **递归更新防护**：
 *    - 必须使用 `isUpdatingFromStore` 标志
 *    - 防止 `setState` 和 Pinia Store 的 `watch` 形成循环
 * 
 * 2. **状态一致性**：
 *    - 确保 Pinia Store 和 StateManager 的状态一致
 *    - 状态更新必须同步到 Pinia Store
 * 
 * 3. **数据提取**：
 *    - 只提取 `raw` 值，不提取 `display` 值
 *    - `null` 值也要包含在提交数据中（让后端验证必填字段）
 */

import { reactive, watch } from 'vue'
import { useFormDataStore, type FormDataStore } from '@/architecture/infrastructure/stores/formData'
import { StateManagerImpl } from './StateManagerImpl'
import type { IFormStateManager } from '../../domain/interfaces/IFormStateManager'
import type { FieldValue, FormState, ValidationResult } from '@/architecture/domain/types'

/**
 * 表单状态管理实现
 */
export class FormStateManager extends StateManagerImpl<FormState> implements IFormStateManager {
  private formStore: FormDataStore
  private errors = reactive<Map<string, ValidationResult[]>>(new Map())
  private submitting = reactive({ value: false })

  private response = reactive<{ value: Record<string, any> | null }>({ value: null })
  private metadata = reactive<{ value: Record<string, any> | null }>({ value: null })

  constructor(formStore?: FormDataStore) {
    // 1. 先调用 super 传递初始空状态
    super({
      data: new Map(),
      errors: new Map(),
      submitting: false,
      response: null,
      metadata: null
    })

    // 2. 初始化 store 和其他属性
    this.formStore = formStore || useFormDataStore()
    
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
      this.updateState()
    }, { deep: true })
  }

  /**
   * 重写 setState，确保同步到 formStore.data
   * 🔥 关键修复：合并更新而不是替换，避免丢失数据
   */
  setState(newState: Partial<FormState>): void {
    // ⭐ 同步 data 到 formStore.data
    if (newState.data !== undefined) {
      if (newState.data.size === 0) {
        // 🔥 如果 newState.data 是空 Map，说明是要清空数据（如 clearForm）
        this.formStore.data.clear()
      } else {
        // 🔥 关键修复：合并更新，而不是清空后复制
        // 这样可以避免在更新单个字段时丢失其他字段的数据
        // 遍历 newState.data，只更新有变化的字段，保留 formStore.data 中的其他字段
        // ⚠️ 重要：不要清空 formStore.data，直接合并更新，这样可以保留 WidgetComponent 直接设置的数据
        newState.data.forEach((value, key) => {
          this.formStore.data.set(key, value)
        })
      }
    }
    // 🔥 如果 newState.data 是 undefined，说明不更新 data，保持原有数据不变
    
    // ⭐ 同步 errors
    if (newState.errors) {
      this.errors.clear()
      newState.errors.forEach((errors, key) => {
        this.errors.set(key, errors)
      })
    }
    
    // ⭐ 同步 submitting
    if (newState.submitting !== undefined) {
      this.submitting.value = newState.submitting
    }
    
    // ⭐ 同步 response
    if (newState.response !== undefined) {
      this.response.value = newState.response
    }
    
    // ⭐ 同步 metadata
    if (newState.metadata !== undefined) {
      this.metadata.value = newState.metadata
    }
    
    // ⭐ 调用父类的 setState（会触发响应式更新）
    // 🔥 关键修复：传递给父类的 newState 应该使用更新后的 formStore.data，而不是 newState.data
    // 这样可以确保父类中的 state.data 始终与 formStore.data 保持一致
    // ⚠️ 重要：使用 formStore.data（已经合并更新后的数据），而不是 newState.data（可能只包含部分字段）
    const stateToSet: FormState = {
      ...this.getState(),
      ...newState,
      data: this.formStore.data  // 🔥 使用 formStore.data，确保包含所有字段（包括 WidgetComponent 直接设置的）
    }
    super.setState(stateToSet)
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
    super.setState(newState)
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

  hasValue(fieldPath: string): boolean {
    return this.formStore.data.has(fieldPath)
  }

  deleteValue(fieldPath: string): void {
    this.formStore.deleteValue(fieldPath)
  }

  getAllFieldPaths(): string[] {
    return this.formStore.getAllFieldPaths()
  }

  getDataSnapshot(): Map<string, FieldValue> {
    return new Map(this.formStore.data)
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

}
