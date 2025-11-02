/**
 * NumberWidget - 数字输入组件（整数）
 * 用于 data.type = "int" / "integer"
 */

import { h } from 'vue'
import { ElInput } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import type { FieldValue } from '../types/field'

/**
 * Number 配置
 */
interface NumberConfig {
  default?: number
  placeholder?: string
  min?: number
  max?: number
  step?: number
  disabled?: boolean
  prepend?: string
  append?: string
}

export class NumberWidget extends BaseWidget {
  private numberConfig: NumberConfig

  constructor(props: any) {
    super(props)
    this.numberConfig = (this.field.widget?.config as NumberConfig) || {}
  }

  render() {
    const currentValue = this.getValue()
    
    // 🔥 不渲染 label，由 FormRenderer 的 el-form-item 统一渲染
    return h(ElInput, {
      type: 'number',
      modelValue: currentValue?.raw,
      placeholder: this.numberConfig.placeholder || `请输入${this.field.name}`,
      disabled: this.numberConfig.disabled || false,
      min: this.numberConfig.min,
      max: this.numberConfig.max,
      step: this.numberConfig.step || 1,
      onInput: (value: string | number) => {
        // 🔥 整数处理：转为整数或 null
        const numValue = value === '' ? null : parseInt(String(value), 10)
        this.updateRawValue(numValue)
      }
    }, {
      prepend: this.numberConfig.prepend ? () => this.numberConfig.prepend : undefined,
      append: this.numberConfig.append ? () => this.numberConfig.append : undefined
    })
  }
}

