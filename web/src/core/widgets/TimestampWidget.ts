/**
 * TimestampWidget - 时间戳组件
 * 用于 data.type = "timestamp" 或 widget.type = "timestamp"
 */

import { h } from 'vue'
import { ElDatePicker } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import { Logger } from '../utils/logger'
import { getDateTimeShortcuts } from './utils/date-shortcuts'
import { getElementPlusFormProps } from './utils/widgetHelpers'
import type { FieldConfig, FieldValue } from '../types/field'
import { formatTimestamp } from '@/utils/date'

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

  constructor(props: WidgetRenderProps) {
    super(props)
    this.timestampConfig = this.getConfig<TimestampConfig>()
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
      // 🔥 统一处理 Element Plus 表单组件的通用属性
      ...getElementPlusFormProps(this.formManager, this.formRenderer, this.fieldPath),
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

  /**
   * 🔥 渲染时间范围搜索（覆盖父类）
   */
  protected renderRangeSearch(): any {
    const showShortcuts = this.timestampConfig.shortcuts !== false
    
    return {
      component: 'ElDatePicker',
      props: {
        type: 'datetimerange',
        rangeSeparator: '至',
        startPlaceholder: '开始时间',
        endPlaceholder: '结束时间',
        format: this.timestampConfig.format || 'YYYY-MM-DD HH:mm:ss',
        valueFormat: 'x',  // 返回时间戳（毫秒）
        shortcuts: showShortcuts ? getDateTimeShortcuts('datetimerange') : undefined,
        style: { width: '360px' },
        clearable: true
      }
    }
  }

  /**
   * 🔥 重写：渲染表格单元格
   * 显示格式化后的时间，而不是时间戳
   */
  renderTableCell(value?: FieldValue): any {
    const fieldValue = value || this.safeGetValue(this.fieldPath)
    
    if (!fieldValue || fieldValue.raw === null || fieldValue.raw === undefined) {
      return '-'
    }
    
    // ✅ 优先使用 display（如果已格式化）
    if (fieldValue.display && fieldValue.display !== String(fieldValue.raw)) {
      return fieldValue.display
    }
    
    // ✅ 格式化时间戳
    const format = this.timestampConfig.format || 'YYYY-MM-DD HH:mm:ss'
    return formatTimestamp(fieldValue.raw, format)
  }

  /**
   * 🔥 渲染详情展示（用于 TableRenderer 详情抽屉）
   * 显示格式化后的时间
   */
  renderForDetail(value?: FieldValue): any {
    const fieldValue = value || this.safeGetValue(this.fieldPath)
    
    if (!fieldValue || fieldValue.raw === null || fieldValue.raw === undefined) {
      return '-'
    }
    
    // ✅ 优先使用 display（如果已格式化）
    if (fieldValue.display && fieldValue.display !== String(fieldValue.raw) && fieldValue.display !== '-') {
      return fieldValue.display
    }
    
    // ✅ 格式化时间戳
    const format = this.timestampConfig.format || 'YYYY-MM-DD HH:mm:ss'
    return formatTimestamp(fieldValue.raw, format)
  }

  /**
   * 🔥 获取复制文本
   * 复制格式化后的时间
   */
  getCopyText(): string {
    const fieldValue = this.safeGetValue(this.fieldPath)
    
    if (!fieldValue || fieldValue.raw === null || fieldValue.raw === undefined) {
      return ''
    }
    
    // ✅ 优先使用 display（如果已格式化）
    if (fieldValue.display && fieldValue.display !== String(fieldValue.raw) && fieldValue.display !== '-') {
      return fieldValue.display
    }
    
    // ✅ 格式化时间戳
    const format = this.timestampConfig.format || 'YYYY-MM-DD HH:mm:ss'
    return formatTimestamp(fieldValue.raw, format)
  }

  /**
   * 🔥 静态方法：从原始数据加载为 FieldValue 格式
   * 确保时间戳被正确格式化
   */
  static loadFromRawData(rawValue: any, field: FieldConfig): FieldValue {
    // 🔥 如果已经是 FieldValue 格式，直接返回
    if (rawValue && typeof rawValue === 'object' && 'raw' in rawValue && 'display' in rawValue) {
      return rawValue
    }
    
    // 🔥 空值处理
    if (rawValue === null || rawValue === undefined) {
      return {
        raw: null,
        display: '-',
        meta: {}
      }
    }

    // ✅ 解析配置
    const config = (field.widget?.config || {}) as TimestampConfig
    const format = config.format || 'YYYY-MM-DD HH:mm:ss'
    
    // ✅ 格式化时间戳
    const display = formatTimestamp(rawValue, format)
    
    return {
      raw: rawValue,
      display,
      meta: {}
    }
  }
}

