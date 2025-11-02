/**
 * SwitchWidget - 开关组件
 * 用于 data.type = "bool" 或 widget.type = "switch"
 */

import { h } from 'vue'
import { ElSwitch } from 'element-plus'
import { BaseWidget } from './BaseWidget'

interface SwitchConfig {
  disabled?: boolean
  activeText?: string
  inactiveText?: string
  activeValue?: boolean | string | number
  inactiveValue?: boolean | string | number
}

export class SwitchWidget extends BaseWidget {
  private switchConfig: SwitchConfig

  constructor(props: WidgetRenderProps) {
    super(props)
    this.switchConfig = (this.field.widget?.config as SwitchConfig) || {}
  }

  render() {
    const currentValue = this.getValue()
    
    return h(ElSwitch, {
      modelValue: currentValue?.raw,
      disabled: this.switchConfig.disabled || false,
      activeText: this.switchConfig.activeText,
      inactiveText: this.switchConfig.inactiveText,
      activeValue: this.switchConfig.activeValue !== undefined ? this.switchConfig.activeValue : true,
      inactiveValue: this.switchConfig.inactiveValue !== undefined ? this.switchConfig.inactiveValue : false,
      onChange: (value: boolean | string | number) => {
        this.updateRawValue(value)
      }
    })
  }
}

