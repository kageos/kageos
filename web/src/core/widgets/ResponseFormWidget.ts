/**
 * ResponseFormWidget - 返回值表单组件
 * 用于渲染返回值中的 form/struct 类型字段（只读展示）
 */

import { h } from 'vue'
import { ElForm, ElFormItem, ElInput, ElInputNumber, ElCard } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import type { FieldConfig } from '../types/field'

export class ResponseFormWidget extends BaseWidget {
  // 标记是否有实际返回数据（通过检查是否有非空值判断）
  private get hasData(): boolean {
    const currentValue = this.getValue()
    const formData = currentValue?.raw || {}
    const keys = Object.keys(formData)
    if (keys.length === 0) return false
    // 检查是否至少有一个字段有实际值（不为 undefined/null/空字符串）
    return keys.some(key => {
      const value = formData[key]
      return value !== undefined && value !== null && value !== ''
    })
  }
  /**
   * 格式化时间戳
   */
  private formatTimestamp(timestamp: number | string | null | undefined, format?: string): string {
    if (!timestamp) return '-'
    const date = new Date(typeof timestamp === 'string' ? parseInt(timestamp, 10) : timestamp)
    if (isNaN(date.getTime())) return '-'
    
    const formatStr = format || 'YYYY-MM-DD HH:mm:ss'
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    const hours = String(date.getHours()).padStart(2, '0')
    const minutes = String(date.getMinutes()).padStart(2, '0')
    const seconds = String(date.getSeconds()).padStart(2, '0')
    
    return formatStr
      .replace('YYYY', String(year))
      .replace('MM', month)
      .replace('DD', day)
      .replace('HH', hours)
      .replace('mm', minutes)
      .replace('ss', seconds)
  }

  /**
   * 格式化浮点数
   */
  private formatFloat(value: number | null | undefined): string {
    if (value === null || value === undefined) return '-'
    return Number(value).toLocaleString('zh-CN', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    })
  }

  /**
   * 根据字段类型渲染单个字段
   */
  private renderField(field: FieldConfig, value: any): any {
    const widgetType = field.widget?.type || 'input'
    
    // 时间戳
    if (widgetType === 'timestamp') {
      const formatted = this.formatTimestamp(value, field.widget.config?.format)
      return h(ElInput, {
        modelValue: formatted,
        disabled: true,
        style: { width: '100%' }
      })
    }
    
    // 浮点数
    if (widgetType === 'float' || field.data?.type === 'float') {
      const formatted = this.formatFloat(value)
      return h(ElInput, {
        modelValue: formatted,
        disabled: true,
        style: { width: '100%' }
      })
    }
    
    // 整数
    if (widgetType === 'number' || field.data?.type === 'int') {
      return h(ElInputNumber, {
        modelValue: value,
        disabled: true,
        style: { width: '100%' }
      })
    }
    
    // 文本域
    if (widgetType === 'textarea' || widgetType === 'text_area') {
      return h(ElInput, {
        modelValue: value !== undefined && value !== null ? String(value) : '',
        type: 'textarea',
        rows: 4,
        disabled: true,
        style: { width: '100%' }
      })
    }
    
    // 默认输入框
    return h(ElInput, {
      modelValue: value !== undefined && value !== null ? String(value) : '',
      disabled: true,
      style: { width: '100%' }
    })
  }

  /**
   * 渲染表单
   */
  render(): any {
    const currentValue = this.getValue()
    const formData = currentValue?.raw || {}
    
    // 获取子字段配置
    const fields: FieldConfig[] = this.field.children || []
    
    // 渲染表单（即使没有数据也显示框架结构）
    return h(ElCard, {
      shadow: 'never',
      bodyStyle: {
        padding: '20px'
      },
      style: {
        backgroundColor: 'var(--el-bg-color-page)',
        border: '1px solid var(--el-border-color-lighter)'
      }
    }, {
      default: () => h(ElForm, {
        labelWidth: '140px',  // 增加标签宽度，使布局更宽松
        labelPosition: 'right' as const
      }, {
        default: () => fields.map(field => 
          h(ElFormItem, {
            key: field.code,
            label: field.name,
            style: {
              marginBottom: '20px'  // 增加表单项间距
            }
          }, {
            default: () => {
              const value = formData[field.code]
              // 如果没有数据，显示占位符
              if (!this.hasData && (value === undefined || value === null)) {
                return h(ElInput, {
                  modelValue: '',
                  placeholder: '等待数据...',
                  disabled: true,
                  style: { width: '100%' }
                })
              }
              return this.renderField(field, value)
            }
          })
        )
      })
    })
  }
}

