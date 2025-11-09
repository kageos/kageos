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

import { h, markRaw, ref } from 'vue'
import { ElCard, ElForm, ElFormItem, ElInput, ElAlert, ElTag, ElIcon, ElDrawer, ElButton } from 'element-plus'
import { Warning, View } from '@element-plus/icons-vue'
import { BaseWidget } from './BaseWidget'
import { Logger } from '../utils/logger'
import { WidgetBuilder } from '../factories/WidgetBuilder'
import { widgetFactory } from '../factories/WidgetFactory'
import { ErrorHandler } from '../utils/ErrorHandler'
import { ResponseFormWidget } from './ResponseFormWidget'
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
  
  // 🔥 详情抽屉状态（用于表格单元格中的 form 字段）
  private showDetailDrawer = ref(false)
  private detailFieldValue: FieldValue | null = null

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
    
    // 🔥 临时 Widget 不需要创建子 Widget（只用于渲染）
    if (this.isTemporary) {
      this.subWidgets = new Map()
      return
    }
    
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
    // 🔥 临时 Widget 不需要加载数据
    if (this.isTemporary) {
      return
    }
    
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
   * 
   * 🔥 关键修复：始终从子 Widget 中收集数据，确保数据完整性
   */
  getRawValueForSubmit(): Record<string, any> {
    const result: Record<string, any> = {}
    
    // 🔥 优先从子 Widget 中收集数据（最可靠的方式）
    if (this.subWidgets.size > 0 && !this.isTemporary) {
      // 🔥 有子 Widget：遍历每个子字段，递归收集数据
      this.subWidgets.forEach((widget, fieldCode) => {
        const rawWidget = widget as MarkRawWidget
        // 🔥 使用 getRawValueForSubmit() 递归收集所有嵌套数据
        if (typeof rawWidget.getRawValueForSubmit === 'function') {
          result[fieldCode] = rawWidget.getRawValueForSubmit()
        } else {
          // 如果没有 getRawValueForSubmit，使用 getValue().raw
          const fieldValue = rawWidget.getValue()
          result[fieldCode] = fieldValue?.raw
        }
      })
    } else {
      // 🔥 临时 Widget 或没有子 Widget：从 value.raw 中提取
      const currentValue = this.getValue()
      const raw = currentValue?.raw
      
      if (raw && typeof raw === 'object' && !Array.isArray(raw)) {
        // 🔥 如果 raw 是对象，遍历子字段配置，递归处理每个字段
        for (const subField of this.subFields) {
          const fieldValue = (raw as Record<string, any>)[subField.code]
          
          if (fieldValue && typeof fieldValue === 'object' && 'raw' in fieldValue && 'display' in fieldValue) {
            // 🔥 如果是 FieldValue 格式，检查是否是容器组件
            const widgetType = subField.widget?.type
            if (widgetType === 'table' || widgetType === 'form') {
              try {
                // 🔥 创建临时 Widget 来调用 getRawValueForSubmit
                const tempWidget = WidgetBuilder.createTemporary({
                  field: subField,
                  value: fieldValue as FieldValue
                })
                const rawWidget = tempWidget as MarkRawWidget
                if (typeof rawWidget.getRawValueForSubmit === 'function') {
                  result[subField.code] = rawWidget.getRawValueForSubmit()
                } else {
                  result[subField.code] = (fieldValue as FieldValue).raw
                }
              } catch (error) {
                Logger.error('[FormWidget]', `getRawValueForSubmit 失败: 字段${subField.code}`, error)
                result[subField.code] = (fieldValue as FieldValue).raw
              }
            } else {
              // 不是容器组件，直接使用 raw
              result[subField.code] = (fieldValue as FieldValue).raw
            }
          } else {
            // 不是 FieldValue 格式，直接使用
            result[subField.code] = fieldValue
          }
        }
      } else if (raw && typeof raw === 'object') {
        // raw 是对象但不是 FieldValue 格式，直接返回
        return raw as Record<string, any>
      }
    }
    
    return result
  }

  /**
   * 🔥 降级渲染：深度很深时使用 JSON 编辑器
   */
  private renderFallback(): any {
    const currentValue = this.getValue()
    const jsonValue = JSON.stringify(currentValue?.raw || {}, null, 2)
    
    return h('div', {
      class: 'form-widget-fallback',
      style: {
        marginBottom: '20px',
        width: '100%'
      }
    }, [
      h(ElCard, {
        shadow: 'hover',
        bodyStyle: { padding: '20px' }
      }, {
        header: () => h('div', {
          style: {
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            fontSize: '14px',
            fontWeight: 'bold'
          }
        }, [
          h(ElIcon, { style: { color: '#E6A23C' } }, () => h(Warning)),
          h('span', this.field.name),
          h(ElTag, { 
            type: 'warning', 
            size: 'small',
            style: { marginLeft: '8px' }
          }, () => `深度 ${this.depth} - JSON 编辑模式`)
        ]),
        default: () => [
          h(ElAlert, {
            type: 'warning',
            showIcon: true,
            closable: false,
            style: { marginBottom: '16px' }
          }, {
            default: () => `嵌套深度较深（${this.depth} 层），已切换到 JSON 编辑模式。您可以直接编辑 JSON 数据，或点击"展开表单"使用表单模式。`
          }),
          h(ElInput, {
            type: 'textarea',
            modelValue: jsonValue,
            rows: 15,
            placeholder: '请输入 JSON 数据',
            'onUpdate:modelValue': (value: string) => {
              try {
                const parsed = JSON.parse(value)
                this.updateRawValue(parsed)
              } catch (error) {
                // JSON 解析失败时不更新，但也不报错（允许用户继续编辑）
              }
            },
            style: { 
              fontFamily: 'monospace',
              fontSize: '12px'
            }
          })
        ]
      })
    ])
  }

  /**
   * 渲染 Form 组件
   */
  render() {
    // 🔥 深度很深时使用降级渲染
    if (this.shouldUseFallback) {
      return this.renderFallback()
    }
    
    // 渲染成一个卡片，包含所有子字段，以及详情抽屉
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
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            fontSize: '14px',
            fontWeight: 'bold',
            color: 'var(--el-text-color-primary)'  // 🔥 使用 CSS 变量，适配深色模式
          }
        }, [
          this.shouldShowDepthWarning && h(ElIcon, { 
            style: { color: '#E6A23C', fontSize: '16px' } 
          }, () => h(Warning)),
          h('span', this.field.name),
          this.shouldShowDepthWarning && h(ElTag, { 
            type: 'warning', 
            size: 'small',
            style: { marginLeft: '4px' }
          }, () => `深度 ${this.depth}`)
        ]),
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
      }),
      // 🔥 渲染详情抽屉（用于表格单元格中的 form 字段）
      this.renderDetailDrawer()
    ])
  }

  /**
   * 🔥 渲染表格单元格（覆盖父类方法）
   * 当 FormWidget 嵌套在 TableWidget 中时，使用简化显示，并提供查看详情功能
   */
  renderTableCell(value?: FieldValue): any {
    // 🔥 临时 Widget 或嵌套场景：使用简化显示，避免递归渲染
    if (this.isTemporary || this.depth > 2) {
      const fieldValue = value || this.getValue()
      const raw = fieldValue?.raw
      
      if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
        return h('span', { style: { color: 'var(--el-text-color-secondary)' } }, '-')
      }
      
      // 显示字段数量和摘要信息，并提供查看按钮
      const fieldCount = Object.keys(raw).length
      
      // 🔥 保存 fieldValue 用于详情抽屉
      this.detailFieldValue = fieldValue
      
      // 🔥 渲染可点击的文本和查看按钮
      return h('div', {
        style: {
          display: 'flex',
          alignItems: 'center',
          gap: '8px',
          cursor: 'pointer'
        },
        onClick: (e: Event) => {
          e.stopPropagation()
          this.showDetailDrawer.value = true
        }
      }, [
        h('span', { 
          style: { 
            color: 'var(--el-color-primary)',
            textDecoration: 'underline'
          } 
        }, `共 ${fieldCount} 个字段`),
        h(ElIcon, {
          style: { 
            fontSize: '14px',
            color: 'var(--el-color-primary)'
          }
        }, {
          default: () => h(View)
        })
      ])
    }
    
    // 非临时 Widget：使用默认格式化
    return super.renderTableCell(value)
  }
  
  /**
   * 🔥 渲染详情抽屉（用于表格单元格中的 form 字段）
   */
  private renderDetailDrawer(): any {
    if (!this.showDetailDrawer.value || !this.detailFieldValue) {
      return null
    }
    
    // 🔥 使用 ResponseFormWidget 渲染表单内容（只读模式）
    const responseWidget = new ResponseFormWidget({
      field: this.field,
      currentFieldPath: `${this.fieldPath}.detail`,
      value: this.detailFieldValue,
      onChange: () => {},
      formManager: this.formManager,
      formRenderer: this.formRenderer,
      depth: this.depth + 1
    })
    
    return h(ElDrawer, {
      modelValue: this.showDetailDrawer.value,
      title: this.field.name || '详细信息',
      size: '50%',
      destroyOnClose: true,
      onClose: () => {
        this.showDetailDrawer.value = false
        this.detailFieldValue = null
      }
    }, {
      default: () => responseWidget.render()
    })
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

