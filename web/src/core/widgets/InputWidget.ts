/**
 * InputWidget - 输入框组件
 */

import { h } from 'vue'
import { BaseWidget } from './BaseWidget'
import { ElInput } from 'element-plus'

export class InputWidget extends BaseWidget {
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

    return h(ElInput, {
      modelValue: currentValue?.raw || '',
      'onUpdate:modelValue': (value: string) => this.handleInput(value),
      placeholder: `请输入${this.field.name}`,
      clearable: true,
      disabled: this.field.table_permission === 'read'
    })
  }
}

