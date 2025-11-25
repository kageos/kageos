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
  SLIDER: 'slider',
  RATE: 'rate',
  COLOR: 'color',
  RICH_TEXT: 'richtext',
  TABLE: 'table',
  FORM: 'form',
  LINK: 'link',
  PROGRESS: 'progress'
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
 * 判断数据类型是否为字符串类型（用于多选组件的提交格式判断）
 * @param dataType 字段的 data.type 值
 * @returns 如果是字符串类型返回 true，否则返回 false
 */
export function isStringDataType(dataType: string | undefined | null): boolean {
  return dataType === DataType.STRING
}

/**
 * 判断数据类型是否为数组类型（用于多选组件的提交格式判断）
 * @param dataType 字段的 data.type 值
 * @returns 如果是数组类型返回 true，否则返回 false
 */
export function isArrayDataType(dataType: string | undefined | null): boolean {
  return dataType === DataType.STRINGS || 
         dataType === DataType.INTS ||
         dataType === DataType.FLOATS ||
         dataType === DataType.STRUCTS
}

/**
 * 获取多选组件的默认数据类型
 * @returns 默认数据类型（数组类型）
 */
export function getMultiSelectDefaultDataType(): string {
  return DataType.STRINGS
}

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

