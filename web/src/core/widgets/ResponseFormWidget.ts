/**
 * ResponseFormWidget - 返回值表单组件
 * 用于渲染返回值中的 form/struct 类型字段（只读展示）
 */

import { h } from 'vue'
import { ElForm, ElFormItem, ElInput, ElInputNumber, ElCard } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import { WidgetBuilder } from '../factories/WidgetBuilder'
import { widgetFactory } from '../factories/WidgetFactory'
import { convertToFieldValue } from '../../utils/field'
import { Logger } from '../utils/logger'
import type { FieldConfig, FieldValue } from '../types/field'
import type { WidgetRenderProps } from '../types/widget'

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
   * 🔥 重构：遵循依赖倒置原则，完全移除硬编码组件类型判断
   * 
   * 设计原则：
   * - 统一使用 WidgetBuilder 创建 Widget
   * - 通过 hasChildren() 判断是否有子节点
   * - 通过 WidgetFactory.getResponseWidgetClass() 检查是否有 Response Widget
   * - 所有组件都使用 renderForResponse() 方法渲染
   * - 新增组件时无需修改此方法，只需在 WidgetFactory 中注册 Response Widget 即可
   */
  private renderField(field: FieldConfig, value: any): any {
    try {
      // 🔥 将原始值转换为 FieldValue 格式
      const fieldValue = convertToFieldValue(value, field)
      
      // 🔥 创建只读的 field 配置（禁用编辑）
      const readonlyField: FieldConfig = {
        ...field,
        widget: {
          ...field.widget,
          config: {
            ...field.widget?.config,
            disabled: true
          }
        }
      }
      
      // 🔥 检查字段是否有子节点（通过创建临时 Widget 来判断）
      const tempWidget = WidgetBuilder.createTemporary({
        field: readonlyField,
        value: fieldValue
      })
      
      // 🔥 如果有子节点，检查是否有对应的 Response Widget（通过工厂模式）
      if (tempWidget.hasChildren()) {
        const widgetType = field.widget?.type || 'input'
        const ResponseWidgetClass = widgetFactory.getResponseWidgetClass(widgetType)
        
        // 🔥 如果有 Response Widget，使用它（如 Form、Table）
        // Response Widget 会在构造函数中自己处理 FieldValue 的转换
        if (ResponseWidgetClass) {
          const widget = new ResponseWidgetClass({
            field: field,
            currentFieldPath: `${this.fieldPath}.${field.code}`,
            value: fieldValue,  // 🔥 直接传递 fieldValue，让 Response Widget 自己处理
            onChange: () => {},
            formManager: this.formManager,
            formRenderer: this.formRenderer,
            depth: this.depth + 1
          })
          return widget.render()
        }
      }
      
      // 🔥 对于所有其他类型（包括有子节点但没有 Response Widget 的），统一使用 WidgetBuilder 创建 Widget
      // 然后调用 renderForResponse() 方法，让组件自己决定如何渲染
      // 这样新增组件时，只需要实现 renderForResponse() 方法即可，无需修改此方法
      const widget = WidgetBuilder.create({
        field: readonlyField,
        fieldPath: `${this.fieldPath}.${field.code}`,
        formManager: this.formManager,
        formRenderer: this.formRenderer,
        depth: this.depth + 1,
        initialValue: fieldValue,
        onChange: () => {}
      })
      
      // 🔥 调用 Widget 的 renderForResponse() 方法（组件自治）
      return widget.renderForResponse()
    } catch (error) {
      Logger.error('[ResponseFormWidget]', `渲染字段失败: ${field.code}`, error)
      // 降级到字符串显示
      return h(ElInput, {
        modelValue: value !== undefined && value !== null ? String(value) : '',
        disabled: true,
        placeholder: '渲染失败',
        style: { width: '100%' }
      })
    }
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
        width: '100%',  // 确保卡片占满宽度
        backgroundColor: 'var(--el-bg-color-page)',
        border: '1px solid var(--el-border-color-lighter)'
      }
    }, {
      default: () => h(ElForm, {
        labelWidth: '140px',  // 增加标签宽度，使布局更宽松
        labelPosition: 'right' as const,
        style: {
          width: '100%'  // 确保表单占满宽度
        }
      }, {
        default: () => fields.map(field => 
          h(ElFormItem, {
            key: field.code,
            label: field.name,
            style: {
              marginBottom: '20px',  // 增加表单项间距
              width: '100%'  // 确保表单项占满宽度
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

