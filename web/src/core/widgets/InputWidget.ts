/**
 * InputWidget - 输入框组件
 * 支持前缀、后缀、密码框等功能
 */

import { h } from 'vue'
import { BaseWidget } from './BaseWidget'
import { ElInput } from 'element-plus'

interface InputConfig {
  placeholder?: string
  disabled?: boolean
  maxlength?: number
  minlength?: number
  showWordLimit?: boolean
  clearable?: boolean
  prepend?: string  // 前缀
  append?: string   // 后缀
  password?: boolean  // 是否为密码框
  showPassword?: boolean  // 是否显示密码切换按钮
  default?: string
}

export class InputWidget extends BaseWidget {
  private inputConfig: InputConfig

  constructor(props: any) {
    super(props)
    this.inputConfig = (this.field.widget?.config as InputConfig) || {}
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
      placeholder: this.inputConfig.placeholder || `请输入${this.field.name}`,
      clearable: this.inputConfig.clearable !== false,
      disabled: this.inputConfig.disabled || this.field.table_permission === 'read',
      maxlength: this.inputConfig.maxlength,
      minlength: this.inputConfig.minlength,
      showWordLimit: this.inputConfig.showWordLimit || false
    }

    // 密码框配置
    if (this.inputConfig.password) {
      inputProps.type = 'password'
      inputProps.showPassword = this.inputConfig.showPassword !== false  // 默认显示密码切换按钮
    }

    // Slots 配置
    const slots: any = {}
    
    // 前缀
    if (this.inputConfig.prepend) {
      slots.prepend = () => this.inputConfig.prepend
    }
    
    // 后缀
    if (this.inputConfig.append) {
      slots.append = () => this.inputConfig.append
    }

    return h(ElInput, inputProps, slots)
  }
}

