/**
 * MockFormManager - 用于创建临时 Widget 时的模拟 FormManager
 * 
 * 使用场景：
 * 1. SearchInput 渲染搜索框配置
 * 2. ListWidget 渲染表格单元格
 * 3. 其他不需要实际数据管理的临时 Widget
 */

import type { ReactiveFormDataManager } from './ReactiveFormDataManager'
import type { FieldValue } from '../types/field'
import { ref, type Ref } from 'vue'

export class MockFormManager {
  /**
   * 创建一个最小化的 mock FormManager
   * 提供所有必需的接口，但不做实际操作
   */
  static create(): ReactiveFormDataManager {
    const mockFormManager = {
      // 基础数据操作
      getValue: (fieldPath: string): FieldValue => {
        return { raw: null, display: '', meta: {} }
      },
      
      setValue: (fieldPath: string, value: FieldValue): void => {
        // 空实现
      },
      
      getRawValue: (fieldPath: string): any => {
        return null
      },
      
      // 表单数据操作
      setFormData: (data: Record<string, any>): void => {
        // 空实现
      },
      
      getAllValues: (): Record<string, FieldValue> => {
        return {}
      },
      
      getAllRawValues: (): Record<string, any> => {
        return {}
      },
      
      // 字段初始化
      initializeField: (fieldPath: string, defaultValue: FieldValue): void => {
        // 空实现
      },
      
      // 事件系统
      emit: (eventName: string, payload?: any): void => {
        // 空实现
      },
      
      on: (pattern: string, handler: (event: any) => void): (() => void) => {
        // 返回空的取消订阅函数
        return () => {}
      },
      
      // 清理
      clear: (): void => {
        // 空实现
      },
      
      // 内部状态（只读）
      formData: ref({}),
      fieldValues: ref({})
    } as ReactiveFormDataManager
    
    return mockFormManager
  }
  
  /**
   * 创建一个带初始值的 mock FormManager
   * 用于某些需要预设值的场景
   */
  static createWithValue(initialValue: FieldValue): ReactiveFormDataManager {
    const mockManager = this.create()
    
    // 覆盖 getValue 方法返回预设值
    const originalGetValue = mockManager.getValue.bind(mockManager)
    mockManager.getValue = (fieldPath: string) => {
      return initialValue
    }
    
    return mockManager
  }
}

