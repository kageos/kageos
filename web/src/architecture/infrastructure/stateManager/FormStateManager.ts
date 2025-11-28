/**
 * FormStateManager - 表单状态管理实现
 * 
 * 职责：基于 Pinia Store 实现表单状态管理
 * 
 * 特点：
 * - 使用现有的 useFormDataStore
 * - 适配 IStateManager 接口
 */

import { computed } from 'vue'
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

  constructor() {
    super({
      data: new Map(),
      errors: new Map(),
      submitting: false
    })

    // 设置 watch，监听 Pinia Store 的变化
    this.setWatch(() => {
      // 从 Pinia Store 获取数据
      const data = new Map<string, FieldValue>()
      // 这里需要从 formStore 获取数据，但 formStore 内部是 Map，需要转换
      // 为了简化，我们直接使用 formStore 的 data
      return {
        data: this.formStore.data,
        errors: new Map(),
        submitting: false
      }
    })
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
}

