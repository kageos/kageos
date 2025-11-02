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
import { widgetFactory } from '../factories/WidgetFactory'
import type { FieldConfig, FieldValue } from '../types/field'
import type { WidgetRenderProps } from '../types/widget'

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

  constructor(props: WidgetRenderProps) {
    super(props)
    
    // 解析 Form 配置
    this.formConfig = (this.field.widget?.config as FormConfig) || {}
    
    // 解析子字段
    this.subFields = this.parseSubFields()
    
    // 创建子 Widget 实例
    this.subWidgets = new Map()
    this.createSubWidgets()
    
    console.log(`[FormWidget] ${this.fieldPath} 初始化，子字段数: ${this.subFields.length}`)
  }

  /**
   * 解析子字段配置
   */
  private parseSubFields(): FieldConfig[] {
    const children = this.field.children || []
    
    if (children.length === 0) {
      console.warn(`[FormWidget] ${this.fieldPath} 没有子字段定义`)
    }
    
    return children
  }

  /**
   * 创建子 Widget 实例
   */
  private createSubWidgets(): void {
    this.subFields.forEach(subField => {
      const subFieldPath = `${this.fieldPath}.${subField.code}`
      
      // 初始化子字段的值
      this.formManager.initializeField(
        subFieldPath,
        BaseWidget.getDefaultValue(subField)
      )
      
      // 创建子 Widget
      const childProps: WidgetRenderProps = {
        field: subField,
        currentFieldPath: subFieldPath,
        value: this.formManager.getValue(subFieldPath),
        onChange: (newValue: FieldValue) => {
          this.formManager.setValue(subFieldPath, newValue)
        },
        formManager: this.formManager,
        formRenderer: this.formRenderer,
        depth: this.depth + 1
      }
      
      const WidgetClass = widgetFactory.getWidgetClass(subField.widget.type)
      const widget = new WidgetClass(childProps)
      
      // 🔥 使用 markRaw 防止 Vue 响应式转换
      this.subWidgets.set(subField.code, markRaw(widget))
      
      // 🔥 注册到父级的 allWidgets（用于快照和提交）
      if (this.formRenderer?.registerWidget) {
        this.formRenderer.registerWidget(subFieldPath, widget)
      }
    })
    
    console.log(`[FormWidget] ${this.fieldPath} 创建了 ${this.subWidgets.size} 个子 Widget`)
  }

  /**
   * 🔥 重写：获取提交时的原始值（递归收集子组件的值）
   * 
   * FormWidget 不依赖自己的 raw 值，而是主动遍历子组件收集它们的值
   * 返回一个对象 { field1: value1, field2: value2, ... }
   */
  getRawValueForSubmit(): Record<string, any> {
    const result: Record<string, any> = {}
    
    console.log(`[FormWidget] ${this.fieldPath} 开始收集子组件值`)
    
    // 遍历每个子字段
    this.subWidgets.forEach((widget, fieldCode) => {
      // 🔥 递归调用：子组件可能是基础组件（直接返回值）或容器组件（继续递归）
      const rawWidget = widget as any  // markRaw 后需要转换
      result[fieldCode] = rawWidget.getRawValueForSubmit()
      
      console.log(`[FormWidget]   - ${fieldCode}:`, result[fieldCode])
    })
    
    console.log(`[FormWidget] ${this.fieldPath} 收集完成:`, result)
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
            color: '#303133'
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
                  (widget as any).render()
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
    console.log(`[FormWidget] 恢复组件数据:`, data)
  }
}

