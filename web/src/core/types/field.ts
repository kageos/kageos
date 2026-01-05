/**
 * 字段配置类型定义
 * 🔥 100% 对齐后端 Field 结构
 * 
 * 🔥 统一类型系统：使用 WidgetTypes 命名空间统一管理所有 Widget 相关类型
 */

/**
 * 🔥 统一的 Widget 类型命名空间
 * 所有 Widget 相关类型都在此命名空间下，便于管理和查找
 */
export namespace WidgetTypes {
  /**
   * Widget 渲染模式
   */
  export type WidgetMode = 'edit' | 'response' | 'table-cell' | 'detail' | 'search'

  /**
   * Widget 配置（基础）
   */
  export interface WidgetConfig {
    type: string  // 'input', 'select', 'multiselect', 'table', etc.
    config?: Record<string, any>  // 各 Widget 的特定配置
    text?: string  // 链接文本等
    icon?: string  // 图标
  }

  /**
   * 字段元数据
   */
  export interface FieldMeta {
    dataType: string  // 'string', 'number', 'boolean', 'array', 'object'
    isRequired: boolean
    isReadonly: boolean
    minLength?: number
    maxLength?: number
    min?: number
    max?: number
    options?: string[]  // oneof 的选项
  }

  /**
   * 字段配置（完整版）
   */
  export interface FieldConfig {
    code: string
    name: string
    desc?: string
    type?: string  // 'string', 'int', 'float', '[]struct', 'struct', etc.
    validation?: string  // 验证规则，如 "required,min=1,max=100"
    search?: string
    widget: WidgetConfig
    data?: {
      type?: string
      format?: string
      example?: string
    }
    callbacks?: string[]  // 字段级别的回调，如 ['OnSelectFuzzy']
    table_permission?: string  // 'read', 'update', 'create', '' (全部权限)
    field_name?: string  // 🔥 Go 字段名（用于验证规则中的字段引用，如 required_if=MemberType vip）
    depend_on?: string  // 🔥 依赖的字段 code，当依赖字段值变化时，该字段会被清空
    
    // 🔥 嵌套字段（后端返回的是 "children"，用于 list/struct 类型）
    children?: FieldConfig[]
    
    // 🔥 前端增强字段（由 FieldPathEnhancer 自动添加）
    field_path?: string  // 'name', 'products[0].name'
    parent_path?: string
    depth?: number
    index_in_parent?: number
    meta?: FieldMeta
  }

  /**
   * FieldValue 数据结构
   * 
   * 🔥 通用字段设计：
   * - raw: 原始值（提交给后端）
   * - display: 显示值（前端展示）
   * - dataType: 数据类型（field.data.type，用于提交前判断和转换）
   * - widgetType: 组件类型（field.widget.type，用于提交前判断和转换）
   * - meta: 元数据（组件特定的扩展信息）
   * 
   * 设计原则：
   * - dataType 和 widgetType 是通用字段，所有组件都应该设置
   * - 方便后续在提交前做类型判断和转换
   * - 避免特殊逻辑，支持未来更复杂的场景
   */
  export interface FieldValue {
    raw: any  // 原始值（提交给后端）
    display: string  // 显示值（前端展示）
    dataType?: string  // 🔥 数据类型（field.data.type，如 'string', '[]string', 'int', 'float' 等）
    widgetType?: string  // 🔥 组件类型（field.widget.type，如 'text', 'select', 'multiselect', 'table' 等）
    meta?: {
      displayInfo?: any  // Select/MultiSelect 的详细信息
      rowStatistics?: Record<string, any>  // MultiSelect 行内聚合
      listStatistics?: Record<string, any>  // List 层聚合
      fromCallback?: boolean
      [key: string]: any  // 其他组件特定的元数据
    }
  }

  /**
   * 验证规则
   */
  export interface ValidationRule {
    type: string
    message?: string
    [key: string]: any
  }

  /**
   * 权限配置
   */
  export interface PermissionConfig {
    read?: boolean
    write?: boolean
    delete?: boolean
  }
}

/**
 * 🔥 向后兼容：导出常用类型（保持现有代码可用）
 * 新代码建议使用 WidgetTypes 命名空间
 */
export type WidgetMode = WidgetTypes.WidgetMode
export type WidgetConfig = WidgetTypes.WidgetConfig
export type FieldConfig = WidgetTypes.FieldConfig
export type FieldValue = WidgetTypes.FieldValue
export type FieldMeta = WidgetTypes.FieldMeta
export type ValidationRule = WidgetTypes.ValidationRule
export type PermissionConfig = WidgetTypes.PermissionConfig

/**
 * 函数详情
 */
export interface FunctionDetail {
  code: string
  name: string
  description?: string
  method: string  // 'GET', 'POST', etc.
  router: string
  template_type: string  // 'form', 'table', 'chart'
  request: FieldConfig[]  // 请求参数（表单字段）
  response: FieldConfig[]  // 响应参数（表格列）
  callbacks?: string[]  // 回调类型，如 ['OnTableAddRow', 'OnSelectFuzzy']
  permissions?: Record<string, boolean>  // ⭐ 权限信息（企业版功能）：权限点 -> 是否有权限
  created_by?: string  // 创建者用户名
}

