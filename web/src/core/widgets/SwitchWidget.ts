/**
 * SwitchWidget - 开关组件
 * 用于 data.type = "bool" 或 widget.type = "switch"
 */

import { h } from 'vue'
import { ElSwitch, ElTag } from 'element-plus'
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
    this.switchConfig = this.getConfig<SwitchConfig>()
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

  /**
   * 🔥 渲染响应参数（只读模式）
   * 在响应参数中显示 Tag 而不是开关组件
   */
  renderForResponse(): any {
    return this.renderForDetail()
  }

  /**
   * 🔥 渲染详情展示（用于 TableRenderer 详情抽屉）
   * 显示 Tag 而不是开关组件
   */
  renderForDetail(value?: FieldValue): any {
    const currentValue = value || this.getValue()
    const boolValue = currentValue?.raw === true || 
                      currentValue?.raw === 'true' || 
                      currentValue?.raw === 1 || 
                      currentValue?.raw === '1' ||
                      (this.switchConfig.activeValue !== undefined && currentValue?.raw === this.switchConfig.activeValue)
    
    const displayText = boolValue 
      ? (this.switchConfig.activeText || '是')
      : (this.switchConfig.inactiveText || '否')
    
    return h(ElTag, {
      type: boolValue ? 'success' : 'info',
      size: 'default'
    }, () => displayText)
  }

  /**
   * 🔥 获取复制文本
   * 复制显示文本（"是"/"否"）
   */
  getCopyText(): string {
    const currentValue = this.getValue()
    const boolValue = currentValue?.raw === true || 
                      currentValue?.raw === 'true' || 
                      currentValue?.raw === 1 || 
                      currentValue?.raw === '1' ||
                      (this.switchConfig.activeValue !== undefined && currentValue?.raw === this.switchConfig.activeValue)
    
    return boolValue 
      ? (this.switchConfig.activeText || '是')
      : (this.switchConfig.inactiveText || '否')
  }
}

