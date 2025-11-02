/**
 * WidgetBuilder - 统一 Widget 创建逻辑
 * 
 * 核心目标：
 * 1. 消除 Widget 创建的重复代码（原本散落在 5 个地方）
 * 2. 统一参数命名和顺序
 * 3. 提供类型安全
 * 4. 简化 Widget 创建流程
 */

import type { FieldConfig, FieldValue } from '../types/field'
import type { ReactiveFormDataManager } from '../managers/ReactiveFormDataManager'
import type { FormRendererContext, WidgetRenderProps } from '../types/widget'
import type { BaseWidget } from '../widgets/BaseWidget'
import { widgetFactory } from './WidgetFactory'

/**
 * 标准 Widget 创建选项
 */
export interface WidgetCreateOptions {
  /** 字段配置 */
  field: FieldConfig
  
  /** 字段路径（如：'name', 'products[0].name'） */
  fieldPath: string
  
  /** 表单数据管理器 */
  formManager: ReactiveFormDataManager
  
  /** FormRenderer 上下文 */
  formRenderer: FormRendererContext | null
  
  /** 嵌套深度（默认 0） */
  depth?: number
  
  /** 初始值（可选，默认从 formManager 读取） */
  initialValue?: FieldValue
  
  /** 值变化回调（可选，默认使用 formManager.setValue） */
  onChange?: (newValue: FieldValue) => void
}

/**
 * 临时 Widget 创建选项（用于表格渲染、搜索输入等场景）
 */
export interface TemporaryWidgetCreateOptions {
  /** 字段配置 */
  field: FieldConfig
  
  /** 初始值（可选） */
  value?: FieldValue
  
  /** FormManager（可选，临时Widget通常不需要） */
  formManager?: ReactiveFormDataManager | null
}

/**
 * WidgetBuilder - Widget 工厂的高级封装
 */
export class WidgetBuilder {
  /**
   * 创建标准 Widget（用于表单）
   * 
   * 使用场景：
   * - FormRenderer 创建顶层 Widget
   * - ListWidget 创建表单 Widget
   * - FormWidget 创建子 Widget
   * 
   * @example
   * ```typescript
   * const widget = WidgetBuilder.create({
   *   field: field,
   *   fieldPath: 'products[0].name',
   *   formManager: this.formManager,
   *   formRenderer: this.formRenderer,
   *   depth: 1
   * })
   * ```
   */
  static create(options: WidgetCreateOptions): BaseWidget {
    const {
      field,
      fieldPath,
      formManager,
      formRenderer,
      depth = 0,
      initialValue,
      onChange
    } = options
    
    // 获取 Widget 类
    const WidgetClass = widgetFactory.getWidgetClass(field.widget?.type || 'input')
    
    // 准备初始值
    const value = initialValue || formManager.getValue(fieldPath)
    
    // 准备 onChange 回调
    const handleChange = onChange || ((newValue: FieldValue) => {
      formManager.setValue(fieldPath, newValue)
    })
    
    // 构造 Widget 属性
    const widgetProps: WidgetRenderProps = {
      field: field,
      currentFieldPath: fieldPath,
      value: value,
      onChange: handleChange,
      formManager: formManager,
      formRenderer: formRenderer,
      depth: depth
    }
    
    // 创建 Widget 实例
    return new WidgetClass(widgetProps)
  }
  
  /**
   * 创建临时 Widget（用于表格单元格、搜索输入等只读场景）
   * 
   * 使用场景：
   * - SearchInput.vue 渲染搜索框配置
   * - ListWidget.renderCellByWidget() 渲染表格单元格
   * - 其他不需要实际数据管理的临时渲染
   * 
   * 注意：临时 Widget 的 formManager 为 null，Widget 必须能够处理这种情况
   * 
   * @example
   * ```typescript
   * // 用于表格渲染
   * const tempWidget = WidgetBuilder.createTemporary({
   *   field: field,
   *   value: cellValue
   * })
   * return tempWidget.renderTableCell(cellValue)
   * 
   * // 用于搜索输入配置
   * const tempWidget = WidgetBuilder.createTemporary({
   *   field: field
   * })
   * return tempWidget.renderSearchInput(searchType)
   * ```
   */
  static createTemporary(options: TemporaryWidgetCreateOptions): BaseWidget {
    const {
      field,
      value,
      formManager = null
    } = options
    
    // 准备初始值
    const initialValue = value || { raw: null, display: '', meta: {} }
    
    // 获取 Widget 类
    const WidgetClass = widgetFactory.getWidgetClass(field.widget?.type || 'input')
    
    // 构造 Widget 属性（formManager 可以为 null）
    const widgetProps: WidgetRenderProps = {
      field: field,
      currentFieldPath: `_temp_.${field.code}`,  // 临时路径
      value: initialValue,
      onChange: () => {},  // 空回调（临时 Widget 不需要修改数据）
      formManager: formManager as any,  // 允许为 null
      formRenderer: null,  // 临时 Widget 不需要 formRenderer
      depth: 0
    }
    
    // 创建 Widget 实例
    return new WidgetClass(widgetProps)
  }
  
  /**
   * 批量创建 Widget（用于容器组件）
   * 
   * 使用场景：
   * - ListWidget 创建多个表单项
   * - FormWidget 创建多个子字段
   * 
   * @example
   * ```typescript
   * const widgets = WidgetBuilder.createBatch(
   *   fields,
   *   (field) => `products[0].${field.code}`,
   *   formManager,
   *   formRenderer,
   *   1
   * )
   * ```
   */
  static createBatch(
    fields: FieldConfig[],
    getFieldPath: (field: FieldConfig, index: number) => string,
    formManager: ReactiveFormDataManager,
    formRenderer: FormRendererContext | null,
    depth: number = 0
  ): Map<string, BaseWidget> {
    const widgets = new Map<string, BaseWidget>()
    
    fields.forEach((field, index) => {
      const fieldPath = getFieldPath(field, index)
      const widget = this.create({
        field,
        fieldPath,
        formManager,
        formRenderer,
        depth
      })
      widgets.set(field.code, widget)
    })
    
    return widgets
  }
}

