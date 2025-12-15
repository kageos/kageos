/**
 * 函数模板类型常量
 */

/**
 * 函数模板类型（与后端返回的 template_type 字段对齐）
 */
export const TEMPLATE_TYPE = {
  TABLE: 'table',
  FORM: 'form',
  CHART: 'chart'
} as const

/**
 * 函数模板类型值
 */
export type TemplateType = typeof TEMPLATE_TYPE[keyof typeof TEMPLATE_TYPE]

