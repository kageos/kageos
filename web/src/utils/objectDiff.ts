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
  // 🔥 修复：null 和 undefined 视为相等，空字符串和 null 也视为相等（用于表单字段对比）
  if (a === null || a === undefined || a === '') {
    return b === null || b === undefined || b === ''
  }
  if (b === null || b === undefined || b === '') {
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
 * 🔥 重要修复（2025-01-XX）：
 * 问题：表单提交时，`prepareSubmitDataWithTypeConversion` 会返回所有字段的数据，包括那些未修改的字段。
 * 如果某个字段在表单中未初始化或用户未修改，它的值可能是 `null`。
 * 之前的逻辑会将 `null` 与旧值（如 "进行中"）对比，认为这是变更，导致未修改的字段被误判为变更。
 * 
 * 修复：如果新值是 `null`/`undefined`/空字符串，但旧值不是空的，说明这个字段可能是未初始化的，
 * 应该忽略它，不包含在 `updates` 中。这样可以避免将未修改的字段误判为变更。
 * 
 * 示例：
 * - 旧值：{ task_name: "测试3", task_status: "进行中", domain: "前端" }
 * - 新值：{ task_name: "测试4", task_status: null, domain: null }  // 用户只修改了 task_name
 * - 结果：{ updates: { task_name: "测试4" } }  // 只包含 task_name，task_status 和 domain 被忽略
 * 
 * ⚠️ 注意：这个修复只适用于更新场景。如果是新增场景，新值为 `null` 应该被视为有效值。
 * 
 * @param oldValues 旧值对象（完整的记录数据）
 * @param newValues 新值对象（用户提交的表单数据，包含所有字段，包括未修改的）
 * @returns 包含 updates（变更字段的新值）和 oldValues（变更字段的旧值）
 * 
 * @example
 * const old = { id: 1, name: "801", type: "小型", created_at: 1234567890 }
 * const new = { name: "802", type: null }  // 用户只修改了 name，type 未初始化
 * const { updates, oldValues } = getChangedFields(old, new)
 * // updates = { name: "802" }  // 只包含 name，type 被忽略
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

    // 🔥 关键修复：如果新值是 null/undefined/空字符串，且旧值不是空的
    // 说明这个字段可能是未初始化的（用户没有修改它），应该忽略它
    // 这样可以避免将未修改的字段（值为 null）误判为变更
    // 
    // 场景：表单提交时，`prepareSubmitDataWithTypeConversion` 会返回所有字段的数据
    // 如果某个字段在表单中未初始化或用户未修改，它的值可能是 `null`
    // 如果旧值是 "进行中"，新值是 `null`，之前的逻辑会认为这是变更
    // 但实际上用户并没有修改这个字段，所以应该忽略它
    // 
    // ⚠️ 重要：这个逻辑只适用于更新场景，如果是新增场景，新值为 null 应该被视为有效值
    const newValueIsEmpty = newValue === null || newValue === undefined || newValue === ''
    const oldValueIsEmpty = oldValue === null || oldValue === undefined || oldValue === ''
    
    // 如果新值是空的，但旧值不是空的，说明这个字段可能是未初始化的，忽略它
    if (newValueIsEmpty && !oldValueIsEmpty) {
      // 忽略这个字段，不包含在 updates 中
      continue
    }

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

