/**
 * 搜索相关常量
 * 🔥 统一管理搜索类型、组件名称等常量，避免硬编码
 */

/**
 * 搜索类型常量
 */
export const SearchType = {
  EQ: 'eq',           // 精确匹配
  LIKE: 'like',       // 模糊匹配
  IN: 'in',          // 包含查询（IN 操作符）
  CONTAINS: 'contains', // 包含查询（FIND_IN_SET，用于多选场景）
  GTE: 'gte',        // 大于等于
  LTE: 'lte',        // 小于等于
  GT: 'gt',          // 大于
  LT: 'lt'           // 小于
} as const

/**
 * 搜索组件名称常量
 */
export const SearchComponent = {
  EL_INPUT: 'ElInput',
  EL_SELECT: 'ElSelect',
  EL_SWITCH: 'ElSwitch',
  EL_DATE_PICKER: 'ElDatePicker',
  RANGE_INPUT: 'RangeInput',
  NUMBER_RANGE_INPUT: 'NumberRangeInput'
} as const

/**
 * 搜索配置常量
 */
export const SearchConfig = {
  DEFAULT_INPUT_WIDTH: '200px',
  DEFAULT_RANGE_WIDTH: '400px',
  DEFAULT_NUMBER_RANGE_WIDTH: '160px', // 数字范围输入框宽度
  DEBOUNCE_DELAY: 300,        // 防抖延迟（毫秒）
  INTERNAL_UPDATE_DELAY: 350, // 内部更新延迟（防抖时间 + 缓冲）
  MAX_COLLAPSE_TAGS: 3,       // 最大折叠标签数
  DEFAULT_PAGE_SIZE: 20       // 默认分页大小
} as const

/**
 * 检查搜索类型是否包含指定类型
 * @param searchType 搜索类型字符串（可能包含多个，用逗号分隔）
 * @param type 要检查的类型
 * @returns 如果包含返回 true
 */
export function hasSearchType(searchType: string | undefined | null, type: string): boolean {
  if (!searchType) return false
  return searchType.includes(type)
}

/**
 * 检查搜索类型是否包含多个类型（AND 关系）
 * @param searchType 搜索类型字符串
 * @param types 要检查的类型数组
 * @returns 如果都包含返回 true
 */
export function hasAllSearchTypes(searchType: string | undefined | null, types: string[]): boolean {
  if (!searchType) return false
  return types.every(type => searchType.includes(type))
}
