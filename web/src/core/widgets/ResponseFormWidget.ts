/**
 * ResponseFormWidget - 返回值表单组件
 * 用于渲染返回值中的 form/struct 类型字段（只读展示）
 */

import { h } from 'vue'
import { ElForm, ElFormItem, ElInput, ElInputNumber } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import type { FieldConfig } from '../types/field'

export class ResponseFormWidget extends BaseWidget {
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
    
    // 如果没有数据，显示空状态
    if (Object.keys(formData).length === 0) {
      return h('div', {
        style: {
          padding: '20px',
          textAlign: 'center',
          color: 'var(--el-text-color-placeholder)'
        }
      }, '暂无数据')
    }
    
    // 渲染表单
    return h(ElForm, {
      labelWidth: '120px'
    }, {
      default: () => fields.map(field => 
        h(ElFormItem, {
          key: field.code,
          label: field.name
        }, {
          default: () => this.renderField(field, formData[field.code])
        })
      )
    })
  }
}

