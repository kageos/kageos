/**
 * 字段配置类型定义
 *
 * 单一来源：
 * - architecture/domain/types 作为架构内门面对外暴露
 * - 真实定义统一收口在 src/architecture/domain/types/field.ts
 */

/**
 * 统一的 Widget 类型命名空间
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
  export type FieldScene = 'list' | 'create' | 'update'

  export interface FieldHideConfig {
    /** 前端隐藏场景：list=列表，create=新增表单，update=编辑表单。未配置表示三个场景均展示；table/form 容器组件不会作为列表列渲染。 */
    scenes?: FieldScene[]
  }

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
    hide?: FieldHideConfig  // 前端隐藏场景，不存在表示列表/新增/编辑均展示；table/form 容器不会作为列表列渲染
    field_name?: string  // Go 字段名（用于验证规则中的字段引用，如 required_if=MemberType vip）
    depend_on?: string  // 依赖的字段 code，当依赖字段值变化时，该字段会被清空

    // 嵌套字段（后端返回的是 "children"，用于 list/struct 类型）
    children?: FieldConfig[]

    // 前端增强字段（由 FieldPathEnhancer 自动添加）
    field_path?: string  // 'name', 'products[0].name'
    parent_path?: string
    depth?: number
    index_in_parent?: number
    meta?: FieldMeta
  }

  /**
   * FieldValue 数据结构
   */
  export interface FieldValue {
    raw: any  // 原始值（提交给后端）
    display: string  // 显示值（前端展示）
    dataType?: string  // 数据类型（field.data.type，如 'string', '[]string', 'int', 'float' 等）
    widgetType?: string  // 组件类型（field.widget.type，如 'text', 'select', 'multiselect', 'table' 等）
    meta?: {
      displayInfo?: any  // Select/MultiSelect 的详细信息
      rowStatistics?: Record<string, any>  // MultiSelect 行内聚合
      listStatistics?: Record<string, any>  // List 层聚合
      fromCallback?: boolean
      [key: string]: any
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

}

/**
 * 向后兼容：导出常用类型
 */
export type WidgetMode = WidgetTypes.WidgetMode
export type WidgetConfig = WidgetTypes.WidgetConfig
export type FieldConfig = WidgetTypes.FieldConfig
export type FieldScene = WidgetTypes.FieldScene
export type FieldHideConfig = WidgetTypes.FieldHideConfig
export type FieldValue = WidgetTypes.FieldValue
export type FieldMeta = WidgetTypes.FieldMeta
export type ValidationRule = WidgetTypes.ValidationRule

export interface FormFunctionSchema {
  request: FieldConfig[]
  response: FieldConfig[]
}

export interface TableFunctionSchema {
  request: FieldConfig[]
  fields: FieldConfig[]
}

export interface ChartFunctionSchema {
  request: FieldConfig[]
  response?: FieldConfig[]
}

export interface FunctionSchema {
  version: number
  type: 'form' | 'table' | 'chart'
  form?: FormFunctionSchema
  table?: TableFunctionSchema
  chart?: ChartFunctionSchema
  callbacks?: string[]
}

/**
 * 函数详情
 */
export interface FunctionConnectorStatus {
  provider: string
  required?: boolean
  connected: boolean
  connection_id?: string
  display_name?: string
  provider_name?: string
  provider_logo_url?: string
  provider_brand_color?: string
  provider_account_url?: string
  profile?: ConnectorConnectionProfile
  resolved_from?: string
  required_scopes?: string[]
  granted_scopes?: string[]
  missing_scopes?: string[]
  message?: string
}

export interface ConnectorResourceSummary {
  page_count?: number
  database_count?: number
  samples?: string[]
}

export interface ConnectorConnectionProfile {
  provider?: string
  display_name?: string
  account_id?: string
  account_name?: string
  avatar_url?: string
  account_url?: string
  workspace_id?: string
  workspace_name?: string
  workspace_icon?: string
  resource_summary?: ConnectorResourceSummary
  last_enriched_at?: string
}

export interface FunctionConnectorEndpoint {
  provider?: string
  method?: string
  url?: string
  name?: string
  desc?: string
  required_scopes?: string[]
}

export interface FunctionDetail {
  id?: number
  app_id?: number
  tree_id?: number
  code?: string
  name?: string
  description?: string
  method?: string  // 'GET', 'POST', etc.
  router?: string
  has_config?: boolean
  create_tables?: string
  connectors?: string[]
  connector_endpoints?: FunctionConnectorEndpoint[]
  connector_status?: FunctionConnectorStatus[]
  template_type?: string  // 'form', 'table', 'chart'
  schema?: FunctionSchema  // 函数配置唯一来源
  created_by?: string  // 创建者用户名
  created_at?: string
  updated_at?: string
  [key: string]: any
}
