/**
 * 字段配置类型定义
 * 🔥 100% 对齐后端 Field 结构
 */

/**
 * Widget 配置（基础）
 */
export interface WidgetConfig {
  type: string  // 'input', 'select', 'multiselect', 'table', etc.
  config?: Record<string, any>  // 各 Widget 的特定配置
}

/**
 * 字段配置
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
 * FieldValue 数据结构
 */
export interface FieldValue {
  raw: any  // 原始值（提交给后端）
  display: string  // 显示值
  meta?: {
    displayInfo?: any  // Select/MultiSelect 的详细信息
    rowStatistics?: Record<string, any>  // MultiSelect 行内聚合
    listStatistics?: Record<string, any>  // List 层聚合
    dataType?: string
    fromCallback?: boolean
  }
}

/**
 * 函数详情
 */
export interface FunctionDetail {
  code: string
  name: string
  description?: string
  method: string  // 'GET', 'POST', etc.
  router: string
  template_type: string  // 'form', 'table'
  request: FieldConfig[]  // 请求参数（表单字段）
  response: FieldConfig[]  // 响应参数（表格列）
  callbacks?: string[]  // 回调类型，如 ['OnTableAddRow', 'OnSelectFuzzy']
}

