/**
 * Widget 渲染辅助工具函数
 * 提取公共的渲染逻辑，减少代码重复
 */

/**
 * 创建 ElInput 的 slots 对象
 * @param prepend 前缀文本
 * @param append 后缀文本
 * @returns slots 对象
 */
export function createInputSlots(prepend?: string, append?: string): Record<string, () => string> | {} {
  const slots: Record<string, () => string> = {}
  
  if (prepend) {
    slots.prepend = () => prepend
  }
  
  if (append) {
    slots.append = () => append
  }
  
  return slots
}

/**
 * 获取禁用状态
 * @param configDisabled 配置中的 disabled
 * @param tablePermission 表的权限
 * @returns 是否禁用
 */
export function getDisabledState(configDisabled?: boolean, tablePermission?: string): boolean {
  return configDisabled || tablePermission === 'read'
}

/**
 * 获取占位符文本
 * @param placeholder 配置的占位符
 * @param fieldName 字段名称
 * @returns 占位符文本
 */
export function getPlaceholder(placeholder?: string, fieldName?: string): string {
  return placeholder || `请输入${fieldName || ''}`
}

/**
 * 获取选择占位符文本
 * @param placeholder 配置的占位符
 * @param fieldName 字段名称
 * @returns 占位符文本
 */
export function getSelectPlaceholder(placeholder?: string, fieldName?: string): string {
  return placeholder || `请选择${fieldName || ''}`
}

