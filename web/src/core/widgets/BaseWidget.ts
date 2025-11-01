/**
 * BaseWidget - 所有 Widget 的基类
 */

import { ref, type Ref } from 'vue'
import type { FieldConfig, FieldValue } from '../types/field'
import type { WidgetRenderProps, WidgetSnapshot } from '../types/widget'
import type { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'

/**
 * Widget 快照接口
 */
export interface IWidgetSnapshot {
  captureSnapshot(): WidgetSnapshot
  restoreSnapshot(snapshot: WidgetSnapshot): void
}

/**
 * BaseWidget 基类
 */
export abstract class BaseWidget implements IWidgetSnapshot {
  protected field: FieldConfig
  protected fieldPath: string
  protected fieldCode: string
  protected value: Ref<FieldValue>
  protected formManager: ReactiveFormDataManager
  protected formRenderer: any
  protected depth: number
  protected onChange: (newValue: FieldValue) => void

  // 最大嵌套深度
  protected static readonly MAX_DEPTH = 10

  /**
   * 获取字段的默认值
   * 每个 Widget 子类可以重写此方法来提供自定义的默认值逻辑
   */
  static getDefaultValue(field: FieldConfig): FieldValue {
    // 1. 优先使用 widget.config.default
    if (field.widget?.config && typeof field.widget.config === 'object' && field.widget.config.default !== undefined && field.widget.config.default !== '') {
      return {
        raw: field.widget.config.default,
        display: String(field.widget.config.default),
        meta: {}
      }
    }

    // 2. 根据字段类型设置默认值
    const fieldType = field.data?.type || 'string'
    
    switch (fieldType.toLowerCase()) {
      case 'int':
      case 'float':
      case 'number':
        return {
          raw: undefined,
          display: '',
          meta: {}
        }
      case 'bool':
        return {
          raw: false,
          display: '否',
          meta: {}
        }
      case 'array':
      case '[]struct':
        return {
          raw: [],
          display: '[]',
          meta: {}
        }
      case 'struct':
        return {
          raw: {},
          display: '{}',
          meta: {}
        }
      default:
        return {
          raw: '',
          display: '',
          meta: {}
        }
    }
  }

  constructor(props: WidgetRenderProps) {
    this.field = props.field
    this.fieldPath = props.currentFieldPath
    this.fieldCode = props.field.code
    this.value = ref(props.value)
    this.formManager = props.formManager
    this.formRenderer = props.formRenderer
    this.depth = props.depth || 0
    this.onChange = props.onChange

    // 深度检查
    if (this.depth > BaseWidget.MAX_DEPTH) {
      console.error(`嵌套深度超过限制: ${this.depth}，字段: ${this.fieldPath}`)
      throw new Error(`最大嵌套深度为 ${BaseWidget.MAX_DEPTH}`)
    }

    console.log(`[BaseWidget] 创建 Widget: ${this.fieldPath}, depth: ${this.depth}`)
  }

  /**
   * 获取当前值
   */
  protected getValue(): FieldValue {
    const value = this.value.value
    // 如果值不存在，返回默认值
    if (!value) {
      return {
        raw: '',
        display: '',
        meta: {}
      }
    }
    return value
  }

  /**
   * 设置值
   */
  protected setValue(newValue: FieldValue): void {
    this.value.value = newValue
    this.onChange(newValue)
    console.log(`[BaseWidget] ${this.fieldPath} 值变更:`, newValue)
  }

  /**
   * 根据字段类型转换值
   */
  protected convertValueByType(value: any): any {
    if (value === null || value === undefined || value === '') {
      return value
    }
    
    // 🔥 获取字段类型：优先使用 data.type，如果为空则使用 widget.type
    let fieldType = this.field.data?.type || ''
    if (!fieldType || fieldType.trim() === '') {
      fieldType = this.field.widget?.type || 'string'
    }
    
    // 根据类型转换
    switch (fieldType.toLowerCase()) {
      case 'int':
      case 'integer':
      case 'number':  // 🔥 widget.type 可能是 'number'
        const intValue = Number(value)
        return isNaN(intValue) ? value : intValue
      
      case 'float':
      case 'double':
        const floatValue = Number(value)
        return isNaN(floatValue) ? value : floatValue
      
      case 'bool':
      case 'boolean':
      case 'switch':  // 🔥 widget.type 可能是 'switch'
        if (typeof value === 'boolean') return value
        if (typeof value === 'string') {
          const lower = value.toLowerCase()
          return lower === 'true' || lower === '1' || lower === 'yes'
        }
        return Boolean(value)
      
      case 'string':
      case 'input':  // 🔥 widget.type 可能是 'input'
      case 'text':
      case 'textarea':
      case 'text_area':
      default:
        return String(value)
    }
  }

  /**
   * 获取用于提交的原始值（已转换类型）
   */
  getRawValueForSubmit(): any {
    const raw = this.value.value.raw
    
    // 🔥 获取字段类型：优先使用 data.type，如果为空则使用 widget.type
    let fieldType = this.field.data?.type || ''
    if (!fieldType || fieldType.trim() === '') {
      fieldType = this.field.widget?.type || 'string'
    }
    
    // 对于嵌套结构（List/Struct），不做类型转换（由子组件处理）
    if (fieldType.includes('[]') || fieldType === 'struct' || 
        fieldType === 'table' || fieldType === 'list' || fieldType === 'form') {
      return raw
    }
    
    // 对于基础类型，转换类型
    return this.convertValueByType(raw)
  }

  /**
   * 更新原始值（保留 display 和 meta，自动类型转换）
   */
  protected updateRawValue(raw: any): void {
    // 转换类型（对于基础类型）
    const fieldType = this.field.data?.type || 'string'
    let convertedRaw = raw
    
    // 只有基础类型才转换，嵌套结构由子组件处理
    if (!fieldType.includes('[]') && fieldType !== 'struct') {
      convertedRaw = this.convertValueByType(raw)
    }
    
    this.setValue({
      ...this.value.value,
      raw: convertedRaw,
      display: String(raw)  // display 保持原样（用于显示）
    })
  }

  /**
   * 捕获快照（默认实现）
   */
  captureSnapshot(): WidgetSnapshot {
    console.log(`[BaseWidget] ${this.fieldPath} 捕获快照`)

    return {
      widget_type: this.field.widget.type,
      field_path: this.fieldPath,
      field_code: this.fieldCode,
      field_value: {
        raw: this.value.value.raw,
        display: this.value.value.display,
        meta: this.value.value.meta
      },
      component_data: this.captureComponentData()
    }
  }

  /**
   * 恢复快照（默认实现）
   */
  restoreSnapshot(snapshot: WidgetSnapshot): void {
    console.log(`[BaseWidget] ${this.fieldPath} 恢复快照`)

    // 恢复 FieldValue
    this.setValue({
      raw: snapshot.field_value.raw,
      display: snapshot.field_value.display,
      meta: snapshot.field_value.meta
    })

    // 恢复组件特定数据
    if (snapshot.component_data) {
      this.restoreComponentData(snapshot.component_data)
    }
  }

  /**
   * 捕获组件特定数据（子类覆盖）
   */
  protected captureComponentData(): any {
    return null
  }

  /**
   * 恢复组件特定数据（子类覆盖）
   */
  protected restoreComponentData(data: any): void {
    // 默认无操作
  }

  /**
   * 渲染方法（子类必须实现）
   */
  abstract render(): any
}

