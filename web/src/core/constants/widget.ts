/**
 * Widget 类型常量
 * 🔥 与后端 sdk/agent-app/widget/widget.go 中的常量对齐
 */

// Widget 类型常量（与后端 Type* 常量对齐）
export const WidgetType = {
  INPUT: 'input',
  TEXT: 'text',
  TEXT_AREA: 'text_area',
  SELECT: 'select',
  SWITCH: 'switch',
  TIMESTAMP: 'timestamp',
  USER: 'user',
  ID: 'ID',
  NUMBER: 'number',
  FLOAT: 'float',
  FILES: 'files',
  CHECKBOX: 'checkbox',
  RADIO: 'radio',
  MULTI_SELECT: 'multiselect',
  TABLE: 'table',
  FORM: 'form'
} as const

/**
 * 数据类型常量
 * 🔥 与后端 sdk/agent-app/widget/widget.go 中的 DataType* 常量对齐
 */
export const DataType = {
  STRING: 'string',
  INT: 'int',
  BOOL: 'bool',
  STRINGS: '[]string',
  INTS: '[]int',
  FLOATS: '[]float',
  TIMESTAMP: 'timestamp',
  FLOAT: 'float',
  FILES: 'files',
  STRUCT: 'struct',
  STRUCTS: '[]struct'
} as const

/**
 * Widget 类型别名映射（用于兼容不同的命名）
 */
export const WidgetTypeAliases: Record<string, string> = {
  'text': WidgetType.INPUT,      // text 别名 input
  'textarea': WidgetType.TEXT_AREA,  // textarea 别名 text_area
  'ID': WidgetType.INPUT         // ID 字段使用 input 组件
}

/**
 * 获取标准化的 Widget 类型（处理别名）
 */
export function normalizeWidgetType(type: string | undefined | null): string {
  if (!type) return WidgetType.INPUT
  return WidgetTypeAliases[type] || type
}

