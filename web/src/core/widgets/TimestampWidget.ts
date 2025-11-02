/**
 * TimestampWidget - 时间戳组件
 * 用于 data.type = "timestamp" 或 widget.type = "timestamp"
 */

import { h } from 'vue'
import { ElDatePicker } from 'element-plus'
import { BaseWidget } from './BaseWidget'

interface TimestampConfig {
  disabled?: boolean
  placeholder?: string
  type?: 'date' | 'datetime' | 'datetimerange' | 'daterange'
  format?: string
  valueFormat?: string
  clearable?: boolean
}

export class TimestampWidget extends BaseWidget {
  private timestampConfig: TimestampConfig

  constructor(props: any) {
    super(props)
    this.timestampConfig = (this.field.widget?.config as TimestampConfig) || {}
  }

  render() {
    const currentValue = this.getValue()
    
    return h(ElDatePicker, {
      modelValue: currentValue?.raw,
      type: this.timestampConfig.type || 'datetime',
      placeholder: this.timestampConfig.placeholder || `请选择${this.field.name}`,
      disabled: this.timestampConfig.disabled || false,
      format: this.timestampConfig.format || 'YYYY-MM-DD HH:mm:ss',
      valueFormat: this.timestampConfig.valueFormat || 'x',  // 默认返回时间戳（毫秒）
      clearable: this.timestampConfig.clearable !== false,
      style: { width: '100%' },
      onChange: (value: number | string | null) => {
        // 转换为时间戳（整数）
        if (value === null || value === undefined) {
          this.updateRawValue(null)
        } else {
          const timestamp = typeof value === 'string' ? parseInt(value, 10) : value
          this.updateRawValue(timestamp)
        }
      }
    })
  }
}

