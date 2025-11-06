/**
 * FormWidget - 表单组件
 * 用于渲染对象类型字段（data.type = "struct"），包含多个子字段
 * 
 * 数据结构：
 * {
 *   detail: {
 *     address: "北京市朝阳区",
 *     phone: "13800138000",
 *     note: "请在工作日配送"
 *   }
 * }
 * 
 * 重要：
 * - data.type = "struct" → 数据类型（对象）
 * - widget.type = "form" → 组件类型（表单）
 */

import { h, markRaw } from 'vue'
import { ElCard, ElForm, ElFormItem } from 'element-plus'
import { BaseWidget } from './BaseWidget'
import { Logger } from '../utils/logger'
import { WidgetBuilder } from '../factories/WidgetBuilder'
import { widgetFactory } from '../factories/WidgetFactory'
import { ErrorHandler } from '../utils/ErrorHandler'
import type { FieldConfig, FieldValue } from '../types/field'
import type { WidgetRenderProps, MarkRawWidget } from '../types/widget'

/**
 * Form 配置（目前为空，未来可能扩展）
 */
interface FormConfig {
  // 未来可能的配置项
  collapsible?: boolean  // 是否可折叠
  defaultExpanded?: boolean  // 默认是否展开
}

/**
 * Form 组件数据（用于快照）
 */
interface FormComponentData {
  // 暂时为空，未来可能需要保存折叠状态等
}

export class FormWidget extends BaseWidget {
  // Form 配置
  private formConfig: FormConfig
  
  // 子字段配置
  private subFields: FieldConfig[]
  
  // 子 Widget 实例 [field_code -> Widget]
  private subWidgets: Map<string, BaseWidget>

  /**
   * FormWidget 的默认值是空对象
   */
  static getDefaultValue(field: FieldConfig): FieldValue {
    return {
      raw: {},
      display: '{}',
      meta: {}
    }
  }

  /**
   * 🔥 从原始数据加载为 FieldValue 格式（重写父类方法）
   * 
   * FormWidget 的特殊逻辑：
   * 1. rawValue 应该是对象
   * 2. 递归调用子组件的 loadFromRawData() 处理每个字段
   * 3. 返回的 raw 是 { field_code: FieldValue } 格式
   */
  static loadFromRawData(rawValue: any, field: FieldConfig): FieldValue {
    // 🔥 如果已经是 FieldValue 格式，直接返回
    if (rawValue && typeof rawValue === 'object' && 'raw' in rawValue && 'display' in rawValue) {
      return rawValue
    }
    
    // 🔥 空值或非对象：返回空对象
    if (!rawValue || typeof rawValue !== 'object' || Array.isArray(rawValue)) {
      return this.getDefaultValue(field)
    }
    
    // 🔥 缺少 children 配置：无法递归，返回原始数据
    const subFields = field.children || []
    if (subFields.length === 0) {
      Logger.warn(`[FormWidget] ${field.code} 缺少 children 配置，无法递归解析`)
      return {
        raw: rawValue,
        display: JSON.stringify(rawValue),
        meta: {}
      }
    }
    
    // 🔥 递归转换每个字段
    const convertedData: Record<string, FieldValue> = {}
    
    for (const subField of subFields) {
      const subRawValue = rawValue[subField.code]
      
      // 🔥 通过工厂获取子组件类，调用其 loadFromRawData()（多态）
      try {
        const WidgetClass = widgetFactory.getWidgetClass(subField.widget?.type || 'input')
        convertedData[subField.code] = WidgetClass.loadFromRawData(subRawValue, subField)
      } catch (error) {
        Logger.error('[FormWidget]', `loadFromRawData 失败: 字段${subField.code}`, error)
        // 失败时使用基类默认实现
        convertedData[subField.code] = BaseWidget.loadFromRawData(subRawValue, subField)
      }
    }
    
    return {
      raw: convertedData,
      display: JSON.stringify(convertedData),
      meta: {}
    }
  }

  constructor(props: WidgetRenderProps) {
    super(props)
    
    // 解析 Form 配置
    this.formConfig = this.getConfig<FormConfig>()
    
    // 解析子字段
    this.subFields = this.parseSubFields()
    
    // 🔥 从父组件加载已有数据（如果有）
    this.loadInitialData()
    
    // 创建子 Widget 实例
    this.subWidgets = new Map()
    this.createSubWidgets()
    
  }

  /**
   * 解析子字段配置
   */
  private parseSubFields(): FieldConfig[] {
    const children = this.field.children || []
    
    if (children.length === 0) {
      Logger.warn(`[FormWidget] ${this.fieldPath} 没有子字段定义`)
    }
    
    return children
  }

  /**
   * 🔥 从父组件加载已有数据
   * 
   * 使用静态方法 loadFromRawData() 进行数据转换
   * 符合开闭原则：FormWidget 不需要知道子组件的具体实现
   */
  private loadInitialData(): void {
    const currentValue = this.getValue()
    
    // 🔥 使用静态方法加载数据（多态递归）
    const converted = FormWidget.loadFromRawData(currentValue?.raw, this.field)
    
    // 🔥 converted.raw 已经是 { field_code: FieldValue } 格式
    // 将转换后的数据写回 FormDataManager
    if (converted.raw && typeof converted.raw === 'object' && !Array.isArray(converted.raw)) {
      for (const [fieldCode, fieldValue] of Object.entries(converted.raw)) {
        const subFieldPath = `${this.fieldPath}.${fieldCode}`
        this.formManager?.setValue(subFieldPath, fieldValue as FieldValue)
      }
    }
  }

  /**
   * 创建子 Widget 实例
   */
  private createSubWidgets(): void {
    this.subFields.forEach(subField => {
      const subFieldPath = `${this.fieldPath}.${subField.code}`
      
      try {
        // ✅ 使用 WidgetBuilder 创建子 Widget
        const widget = WidgetBuilder.create({
          field: subField,
          fieldPath: subFieldPath,
          formManager: this.formManager,
          formRenderer: this.formRenderer,
          depth: this.depth + 1
        })
        
        // 🔥 使用 markRaw 防止 Vue 响应式转换
        this.subWidgets.set(subField.code, markRaw(widget))
        
        // 🔥 注册到父级的 allWidgets（用于快照和提交）
        if (this.formRenderer?.registerWidget) {
          this.formRenderer.registerWidget(subFieldPath, widget)
        }
      } catch (error) {
        ErrorHandler.handleWidgetError(`FormWidget.createSubWidgets[${subField.code}]`, error, {
          showMessage: false
        })
      }
    })
    
  }

  /**
   * 🔥 重写：获取提交时的原始值（递归收集子组件的值）
   * 
   * FormWidget 不依赖自己的 raw 值，而是主动遍历子组件收集它们的值
   * 返回一个对象 { field1: value1, field2: value2, ... }
   */
  getRawValueForSubmit(): Record<string, any> {
    const result: Record<string, any> = {}
    
    
    // 遍历每个子字段
    this.subWidgets.forEach((widget, fieldCode) => {
      // 🔥 递归调用：子组件可能是基础组件（直接返回值）或容器组件（继续递归）
      // 🔥 类型安全地访问 markRaw 后的 Widget
      const rawWidget = widget as MarkRawWidget
      result[fieldCode] = rawWidget.getRawValueForSubmit()
      
    })
    
    return result
  }

  /**
   * 渲染 Form 组件
   */
  render() {
    // 渲染成一个卡片，包含所有子字段
    return h('div', { 
      class: 'form-widget',
      style: {
        marginBottom: '20px',
        width: '100%'  // 🔥 确保占满宽度
      }
    }, [
      h(ElCard, {
        shadow: 'hover',
        bodyStyle: { padding: '20px', width: '100%' },  // 🔥 卡片内容占满宽度
        style: { width: '100%' }  // 🔥 卡片本身占满宽度
      }, {
        header: () => h('div', {
          style: {
            fontSize: '14px',
            fontWeight: 'bold',
            color: 'var(--el-text-color-primary)'  // 🔥 使用 CSS 变量，适配深色模式
          }
        }, this.field.name),
        default: () => [
          // 🔥 使用 ElForm 包裹子字段，提供统一的表单布局
          h(ElForm, {
            labelWidth: '100px',
            style: { width: '100%' }  // 🔥 表单占满宽度
          }, () => [
            // 遍历子字段，渲染每个 Widget（包含标签）
          ...Array.from(this.subWidgets.entries()).map(([fieldCode, widget]) => {
              const subField = this.subFields.find(f => f.code === fieldCode)
              if (!subField) return null
              
              return h(ElFormItem, {
              key: fieldCode,
                label: subField.name,  // 🔥 显示字段标签
                prop: fieldCode,
              style: { 
                  width: '100%',
                  marginBottom: '18px'  // 🔥 增加表单项之间的间距
              } 
              }, {
                default: () => [
              // 渲染子 Widget
              (widget as MarkRawWidget).render()
                ]
              })
            })
            ])
        ]
      })
    ])
  }

  /**
   * 捕获组件数据（用于快照）
   */
  protected captureComponentData(): FormComponentData {
    return {
      // 暂时为空，未来可能保存折叠状态等
    }
  }

  /**
   * 恢复组件数据（从快照）
   */
  protected restoreComponentData(data: FormComponentData): void {
    // TODO: 恢复 Form 状态
  }
}

