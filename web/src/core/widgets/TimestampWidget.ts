/**
 * TimestampWidget - 时间戳组件
 * 用于 data.type = "timestamp" 或 widget.type = "timestamp"
 */

import { h } from 'vue'
import { ElDatePicker } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import { getDateTimeShortcuts } from './utils/date-shortcuts'

interface TimestampConfig {
  disabled?: boolean
  placeholder?: string
  type?: 'date' | 'datetime' | 'datetimerange' | 'daterange'
  format?: string
  valueFormat?: string
  clearable?: boolean
  shortcuts?: boolean  // 是否显示快捷选择（默认true）
}

export class TimestampWidget extends BaseWidget {
  private timestampConfig: TimestampConfig

  constructor(props: any) {
    super(props)
    this.timestampConfig = (this.field.widget?.config as TimestampConfig) || {}
  }

  render() {
    const currentValue = this.getValue()
    const pickerType = this.timestampConfig.type || 'datetime'
    const showShortcuts = this.timestampConfig.shortcuts !== false  // 默认显示快捷选择
    
    return h(ElDatePicker, {
      modelValue: currentValue?.raw,
      type: pickerType,
      placeholder: this.timestampConfig.placeholder || `请选择${this.field.name}`,
      disabled: this.timestampConfig.disabled || false,
      format: this.timestampConfig.format || 'YYYY-MM-DD HH:mm:ss',
      valueFormat: this.timestampConfig.valueFormat || 'x',  // 默认返回时间戳（毫秒）
      clearable: this.timestampConfig.clearable !== false,
      shortcuts: showShortcuts ? getDateTimeShortcuts(pickerType) : undefined,  // 添加快捷选择
      style: { width: '100%' },
      onChange: (value: number | string | [number, number] | [string, string] | null) => {
        // 转换为时间戳（整数）
        if (value === null || value === undefined) {
          this.updateRawValue(null)
        } else if (Array.isArray(value)) {
          // 范围选择：转换为时间戳数组
          const timestamps = value.map(v => typeof v === 'string' ? parseInt(v, 10) : v)
          this.updateRawValue(timestamps)
        } else {
          const timestamp = typeof value === 'string' ? parseInt(value, 10) : value
          this.updateRawValue(timestamp)
        }
      }
    })
  }
}

