/**
 * NumberWidget - 数字输入组件（整数）
 * 用于 data.type = "int" / "integer"
 */

import { h } from 'vue'
import { ElInput } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import { Logger } from '../utils/logger'
import type { NumberLikeConfig } from './types/widget-config'
import { createInputSlots, getDisabledState, getPlaceholder } from './utils/render-helpers'

export class NumberWidget extends BaseWidget {
  private numberConfig: NumberLikeConfig

  constructor(props: WidgetRenderProps) {
    super(props)
    this.numberConfig = (this.field.widget?.config as NumberLikeConfig) || {}
  }

  render() {
    const currentValue = this.getValue()
    
    // 🔥 不渲染 label，由 FormRenderer 的 el-form-item 统一渲染
    return h(ElInput, {
      type: 'number',
      modelValue: currentValue?.raw,
      placeholder: getPlaceholder(this.numberConfig.placeholder, this.field.name),
      disabled: getDisabledState(this.numberConfig.disabled, this.field.table_permission),
      min: this.numberConfig.min,
      max: this.numberConfig.max,
      step: this.numberConfig.step || 1,
      clearable: this.numberConfig.clearable !== false,
      // 🔥 禁用 Element Plus 的原生验证（使用我们的自定义验证系统）
      validateEvent: false,
      onInput: (value: string | number) => {
        // 🔥 整数处理：转为整数或 null
        const numValue = value === '' ? null : parseInt(String(value), 10)
        this.updateRawValue(numValue)
      }
    }, createInputSlots(this.numberConfig.prepend, this.numberConfig.append))
  }

  /**
   * 🔥 渲染整数范围搜索（覆盖父类）
   */
  protected renderRangeSearch(): any {
    return {
      component: 'NumberRangeInput',
      props: {
        minPlaceholder: `最小${this.field.name}`,
        maxPlaceholder: `最大${this.field.name}`,
        precision: 0,  // 整数，无小数
        step: this.numberConfig.step || 1,
        min: this.numberConfig.min,
        max: this.numberConfig.max
      }
    }
  }
}

