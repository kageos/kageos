/**
 * FloatWidget - 浮点数输入组件
 * 用于 data.type = "float" / "double"
 */

import { h } from 'vue'
import { ElInput } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import type { FieldValue } from '../types/field'

/**
 * Float 配置
 */
interface FloatConfig {
  default?: number
  placeholder?: string
  min?: number
  max?: number
  step?: number
  precision?: number  // 小数点精度
  disabled?: boolean
  prepend?: string
  append?: string
}

export class FloatWidget extends BaseWidget {
  private floatConfig: FloatConfig

  constructor(props: any) {
    super(props)
    this.floatConfig = (this.field.widget?.config as FloatConfig) || {}
  }

  render() {
    const currentValue = this.getValue()
    
    return h('div', { class: 'float-widget' }, [
      h('label', {
        style: {
          display: 'block',
          marginBottom: '8px',
          fontSize: '14px',
          color: '#606266'
        }
      }, this.field.name),
      h(ElInput, {
        type: 'number',
        modelValue: currentValue?.raw,
        placeholder: this.floatConfig.placeholder || `请输入${this.field.name}`,
        disabled: this.floatConfig.disabled || false,
        min: this.floatConfig.min,
        max: this.floatConfig.max,
        step: this.floatConfig.step || 0.01,  // 🔥 浮点数默认步长 0.01
        onInput: (value: string | number) => {
          // 🔥 浮点数处理：转为浮点数或 null
          if (value === '') {
            this.updateRawValue(null)
          } else {
            let numValue = parseFloat(String(value))
            
            // 如果配置了精度，进行四舍五入
            if (this.floatConfig.precision !== undefined && !isNaN(numValue)) {
              numValue = Number(numValue.toFixed(this.floatConfig.precision))
            }
            
            this.updateRawValue(isNaN(numValue) ? null : numValue)
          }
        }
      }, {
        prepend: this.floatConfig.prepend ? () => this.floatConfig.prepend : undefined,
        append: this.floatConfig.append ? () => this.floatConfig.append : undefined
      })
    ])
  }
}

