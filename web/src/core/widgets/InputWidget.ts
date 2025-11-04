/**
 * InputWidget - 输入框组件
 * 支持前缀、后缀、密码框等功能
 */

import { h } from 'vue'
import { BaseWidget } from './BaseWidget'
import { ElInput } from 'element-plus'
import type { InputConfig } from './types/widget-config'
import { createInputSlots, getDisabledState, getPlaceholder } from './utils/render-helpers'
import { getElementPlusFormProps } from './utils/widgetHelpers'

export class InputWidget extends BaseWidget {
  private inputConfig: InputConfig

  constructor(props: WidgetRenderProps) {
    super(props)
    this.inputConfig = this.getConfig<InputConfig>()
  }

  /**
   * 处理输入变更
   */
  private handleInput(value: string): void {
    this.updateRawValue(value)
  }

  /**
   * 渲染输入框
   */
  render(): any {
    const currentValue = this.getValue()

    // 基础配置
    const inputProps: any = {
      modelValue: currentValue?.raw || '',
      'onUpdate:modelValue': (value: string) => this.handleInput(value),
      placeholder: getPlaceholder(this.inputConfig.placeholder, this.field.name),
      clearable: this.inputConfig.clearable !== false,
      disabled: getDisabledState(this.inputConfig.disabled, this.field.table_permission),
      maxlength: this.inputConfig.maxlength,
      minlength: this.inputConfig.minlength,
      showWordLimit: this.inputConfig.showWordLimit || false,
      // 🔥 统一处理 Element Plus 表单组件的通用属性（validateEvent, onBlur）
      ...getElementPlusFormProps(this.formManager, this.formRenderer, this.fieldPath)
    }

    // 密码框配置
    if (this.inputConfig.password) {
      inputProps.type = 'password'
      inputProps.showPassword = this.inputConfig.showPassword !== false  // 默认显示密码切换按钮
    }

    return h(ElInput, inputProps, createInputSlots(this.inputConfig.prepend, this.inputConfig.append))
  }
}

