/**
 * 对象对比工具
 * 用于对比旧值和新值，找出变更的字段
 */

/**
 * 深度对比两个值是否相等
 * 支持基本类型、对象、数组的深度对比
 */
function isEqual(a: any, b: any): boolean {
  // 处理 null 和 undefined
  if (a === null || a === undefined) {
    return b === null || b === undefined
  }
  if (b === null || b === undefined) {
    return false
  }

  // 处理基本类型
  if (typeof a !== 'object' || typeof b !== 'object') {
    return a === b
  }

  // 处理数组
  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) {
      return false
    }
    for (let i = 0; i < a.length; i++) {
      if (!isEqual(a[i], b[i])) {
        return false
      }
    }
    return true
  }

  // 处理对象
  if (Array.isArray(a) || Array.isArray(b)) {
    return false
  }

  const keysA = Object.keys(a)
  const keysB = Object.keys(b)

  if (keysA.length !== keysB.length) {
    return false
  }

  for (const key of keysA) {
    if (!keysB.includes(key)) {
      return false
    }
    if (!isEqual(a[key], b[key])) {
      return false
    }
  }

  return true
}

/**
 * 对比旧值和新值，找出变更的字段
 * 
 * ⚠️ 关键逻辑：
 * - 只对比新值中存在的字段（新值中没有的字段，说明用户没有修改，不应该出现在 updates 中）
 * - 如果新值中某个字段的值与旧值不同，才认为是变更
 * - 不处理"删除的字段"逻辑，因为表单提交时，用户没有修改的字段不应该出现在新值中
 * 
 * @param oldValues 旧值对象（完整的记录数据）
 * @param newValues 新值对象（用户提交的表单数据，只包含用户修改的字段）
 * @returns 包含 updates（变更字段的新值）和 oldValues（变更字段的旧值）
 * 
 * @example
 * const old = { id: 1, name: "801", type: "小型", created_at: 1234567890 }
 * const new = { name: "802" }  // 用户只修改了 name
 * const { updates, oldValues } = getChangedFields(old, new)
 * // updates = { name: "802" }  // 只包含用户修改的字段
 * // oldValues = { name: "801" }
 */
export function getChangedFields(
  oldValues: Record<string, any>,
  newValues: Record<string, any>
): {
  updates: Record<string, any>    // 只包含变更的字段（新值）
  oldValues: Record<string, any>    // 变更字段的旧值
} {
  const updates: Record<string, any> = {}
  const oldValuesChanged: Record<string, any> = {}

  // ⚠️ 关键：只遍历新值中存在的字段
  // 如果新值中没有某个字段，说明用户没有修改它，不应该出现在 updates 中
  for (const key in newValues) {
    const newValue = newValues[key]
    const oldValue = oldValues[key]

    // 深度对比：只有当值真正发生变化时，才认为是变更
    if (!isEqual(newValue, oldValue)) {
      updates[key] = newValue
      oldValuesChanged[key] = oldValue
    }
  }

  // ⚠️ 注意：不再处理"删除的字段"逻辑
  // 因为表单提交时，用户没有修改的字段不应该出现在 newValues 中
  // 如果 newValues 中没有某个字段，说明用户没有修改它，不应该出现在 updates 中

  return {
    updates,
    oldValues: oldValuesChanged
  }
}

