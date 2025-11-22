/**
 * 字符串工具函数
 * 🔥 统一处理字符串分割、合并、清理等操作，消除重复代码
 */

/**
 * 将逗号分隔的字符串分割为数组，并清理空白字符
 * @param str 逗号分隔的字符串
 * @returns 清理后的字符串数组
 * 
 * @example
 * parseCommaSeparatedString('a, b, c') // ['a', 'b', 'c']
 * parseCommaSeparatedString('a,,b') // ['a', 'b']
 * parseCommaSeparatedString('') // []
 */
export function parseCommaSeparatedString(str: string | null | undefined): string[] {
  if (!str || typeof str !== 'string') {
    return []
  }
  return str.split(',').map(v => v.trim()).filter(v => v)
}

/**
 * 将数组合并为逗号分隔的字符串
 * @param arr 字符串数组
 * @returns 逗号分隔的字符串
 * 
 * @example
 * joinCommaSeparatedString(['a', 'b', 'c']) // 'a,b,c'
 * joinCommaSeparatedString([]) // ''
 */
export function joinCommaSeparatedString(arr: (string | number)[]): string {
  if (!Array.isArray(arr) || arr.length === 0) {
    return ''
  }
  return arr.map(v => String(v)).join(',')
}

/**
 * 生成搜索组件的占位符文本
 * @param fieldName 字段名称
 * @param type 占位符类型
 * @returns 占位符文本
 * 
 * @example
 * generatePlaceholder('用户名', 'select') // '请选择用户名'
 * generatePlaceholder('用户名', 'input') // '请输入用户名'
 * generatePlaceholder('用户名', 'search') // '搜索用户名'
 */
export function generatePlaceholder(fieldName: string, type: 'select' | 'input' | 'search' | 'start' | 'end' | 'min' | 'max'): string {
  const prefixMap: Record<string, string> = {
    select: '请选择',
    input: '请输入',
    search: '搜索',
    start: '开始',
    end: '结束',
    min: '最小',
    max: '最大'
  }
  
  const prefix = prefixMap[type] || '请输入'
  return `${prefix}${fieldName}`
}

