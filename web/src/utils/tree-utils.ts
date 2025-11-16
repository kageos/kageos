/**
 * 服务树工具函数
 * 用于处理服务树的通用逻辑，避免代码重复
 */

import type { ServiceTree } from '@/types'

// 扩展 ServiceTree 类型，支持虚拟节点
export interface ExtendedServiceTree extends ServiceTree {
  isGroup?: boolean
  isPending?: boolean
}

/**
 * 生成函数组节点的唯一 ID（基于 groupCode 的哈希）
 */
export function generateGroupId(groupCode: string): number {
  let hash = 0
  for (let i = 0; i < groupCode.length; i++) {
    const char = groupCode.charCodeAt(i)
    hash = ((hash << 5) - hash) + char
    hash = hash & hash // 转换为 32 位整数
  }
  return -Math.abs(hash || Date.now())
}

/**
 * 创建虚拟函数组节点
 */
export function createGroupNode(
  groupCode: string,
  groupName: string,
  parentNode: ServiceTree,
  includeChildren = true
): ExtendedServiceTree {
  const groupId = generateGroupId(groupCode)
  
  return {
    id: groupId,
    name: groupName,
    code: `__group__${groupCode}`,
    parent_id: parentNode.id,
    type: 'package',
    description: '',
    tags: '',
    app_id: parentNode.app_id,
    ref_id: 0,
    full_code_path: `${parentNode.full_code_path}/__group__${groupCode}`,
    full_group_code: groupCode,
    group_name: groupName,
    created_at: '',
    updated_at: '',
    children: includeChildren ? [] : [],
    isGroup: true
  } as ExtendedServiceTree
}

/**
 * 按 full_group_code 分组函数
 */
export function groupFunctionsByCode(functions: ServiceTree[]): {
  grouped: Map<string, ServiceTree[]>
  ungrouped: ServiceTree[]
} {
  const grouped = new Map<string, ServiceTree[]>()
  const ungrouped: ServiceTree[] = []
  
  functions.forEach(func => {
    if (func.full_group_code && func.full_group_code.trim() !== '') {
      if (!grouped.has(func.full_group_code)) {
        grouped.set(func.full_group_code, [])
      }
      grouped.get(func.full_group_code)!.push(func)
    } else {
      ungrouped.push(func)
    }
  })
  
  return { grouped, ungrouped }
}

/**
 * 获取函数组名称（优先使用 group_name，否则从 groupCode 提取）
 */
export function getGroupName(funcs: ServiceTree[], groupCode: string): string {
  return funcs[0]?.group_name || groupCode.split('/').pop() || groupCode
}

