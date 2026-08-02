import type { ServiceTree } from '@/architecture/domain/types'
import { normalizeResourcePath } from '@/architecture/shared/resourcePath'

export interface PermissionRequestViewSummary {
  ownPendingCount: number
  reviewPendingCount: number
  totalCount: number
}

interface PermissionRequestPathCounts {
  own_pending_count?: number
  review_pending_count?: number
}

function hasReadAccess(node: ServiceTree): boolean {
  const permissions = node.permissions
  return Boolean(permissions?.read || permissions?.admin || permissions?.owner)
}

/**
 * 仅保留当前账号可读取的节点，同时保留通向可读后代的目录，避免树结构断裂。
 */
export function filterServiceTreeByReadAccess(nodes: ServiceTree[]): ServiceTree[] {
  return nodes.flatMap((node) => {
    const children = filterServiceTreeByReadAccess(node.children || [])
    if (!hasReadAccess(node) && children.length === 0) return []
    return [{ ...node, children }]
  })
}

/**
 * 将后端返回的节点直属申请数向祖先目录汇总。叶子保持自己的数量，父目录展示整棵子树总数。
 */
export function aggregatePermissionRequestSummaries(
  nodes: ServiceTree[],
  directCountsByPath: Record<string, PermissionRequestPathCounts>,
): Record<string, PermissionRequestViewSummary> {
  const summaries: Record<string, PermissionRequestViewSummary> = {}

  const visit = (node: ServiceTree): PermissionRequestViewSummary => {
    const path = normalizeResourcePath(node.full_code_path || '')
    const direct = directCountsByPath[path] || {}
    const summary: PermissionRequestViewSummary = {
      ownPendingCount: Number(direct.own_pending_count || 0),
      reviewPendingCount: Number(direct.review_pending_count || 0),
      totalCount: 0,
    }

    for (const child of node.children || []) {
      const childSummary = visit(child)
      summary.ownPendingCount += childSummary.ownPendingCount
      summary.reviewPendingCount += childSummary.reviewPendingCount
    }
    summary.totalCount = summary.ownPendingCount + summary.reviewPendingCount
    if (path) summaries[path] = summary
    return summary
  }

  for (const node of nodes) visit(node)
  return summaries
}

/** 返回需要自动展开的目录 ID，使所有带申请徽章的后代尽快可见。 */
export function collectPermissionRequestExpandedDirectoryIds(
  nodes: ServiceTree[],
  summaries: Record<string, PermissionRequestViewSummary>,
): number[] {
  const ids: number[] = []
  const visit = (node: ServiceTree) => {
    const path = normalizeResourcePath(node.full_code_path || '')
    if (node.type === 'package' && Number(summaries[path]?.totalCount || 0) > 0) {
      ids.push(Number(node.id))
    }
    for (const child of node.children || []) visit(child)
  }
  for (const node of nodes) visit(node)
  return ids
}
