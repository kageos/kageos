/**
 * 函数类型常量
 */

/**
 * 函数模板类型
 */
export const FUNCTION_TYPE = {
  TABLE: 'table',
  FORM: 'form'
} as const

/**
 * 函数模板类型值
 */
export type FunctionType = typeof FUNCTION_TYPE[keyof typeof FUNCTION_TYPE]

