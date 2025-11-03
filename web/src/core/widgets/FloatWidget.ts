/**
 * FloatWidget - 浮点数输入组件
 * 用于 data.type = "float" / "double"
 */

import { h } from 'vue'
import { ElInput } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import { Logger } from '../utils/logger'
import type { NumberLikeConfig } from './types/widget-config'
import type { WidgetRenderProps } from '../types/widget'
import { createInputSlots, getDisabledState, getPlaceholder } from './utils/render-helpers'
import { getElementPlusFormProps } from './utils/widgetHelpers'

/**
 * Float 配置（继承数字配置，添加精度）
 */
interface FloatConfig extends NumberLikeConfig {
  precision?: number  // 小数点精度
}

export class FloatWidget extends BaseWidget {
  private floatConfig: FloatConfig

  constructor(props: WidgetRenderProps) {
    super(props)
    this.floatConfig = (this.field.widget?.config as FloatConfig) || {}
  }

  render() {
    const currentValue = this.getValue()
    
    // 🔥 不渲染 label，由 FormRenderer 的 el-form-item 统一渲染
    return h(ElInput, {
      type: 'number',
      modelValue: currentValue?.raw,
      placeholder: getPlaceholder(this.floatConfig.placeholder, this.field.name),
      disabled: getDisabledState(this.floatConfig.disabled, this.field.table_permission),
      min: this.floatConfig.min,
      max: this.floatConfig.max,
      step: this.floatConfig.step || 0.01,  // 🔥 浮点数默认步长 0.01
      clearable: this.floatConfig.clearable !== false,
      // 🔥 统一处理 Element Plus 表单组件的通用属性
      ...getElementPlusFormProps(this.formManager, this.formRenderer, this.fieldPath),
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
    }, createInputSlots(this.floatConfig.prepend, this.floatConfig.append))
  }

  /**
   * 🔥 渲染浮点数范围搜索（覆盖父类）
   */
  protected renderRangeSearch(): any {
    return {
      component: 'NumberRangeInput',
      props: {
        minPlaceholder: `最小${this.field.name}`,
        maxPlaceholder: `最大${this.field.name}`,
        precision: this.floatConfig.precision !== undefined ? this.floatConfig.precision : 2,
        step: this.floatConfig.step || 0.01,
        min: this.floatConfig.min,
        max: this.floatConfig.max
      }
    }
  }
}

