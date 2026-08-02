import { reactive } from 'vue'
import {
  getPermissionRequestSummary,
  type PermissionRequestPathSummary,
} from '@/architecture/presentation/context/api/permission'
import { normalizeResourcePath } from '@/architecture/shared/resourcePath'

export interface PermissionRequestSummaryState {
  paths: Record<string, PermissionRequestPathSummary>
  ownPendingCount: number
  reviewPendingCount: number
  loaded: boolean
}

export interface PermissionRequestSummaryTreeNode {
  full_code_path?: string
  permission_requests?: PermissionRequestPathSummary
  children?: PermissionRequestSummaryTreeNode[]
}

const states = reactive<Record<string, PermissionRequestSummaryState>>({})
const inFlight = new Map<string, Promise<void>>()

function emptyState(): PermissionRequestSummaryState {
  return {
    paths: {},
    ownPendingCount: 0,
    reviewPendingCount: 0,
    loaded: false,
  }
}

export function getPermissionRequestSummaryState(workspaceRoot: string): PermissionRequestSummaryState {
  const root = normalizeResourcePath(workspaceRoot)
  if (!states[root]) states[root] = emptyState()
  return states[root]
}

export function seedPermissionRequestSummaryFromTree(
  workspaceRoot: string,
  nodes: PermissionRequestSummaryTreeNode[],
): void {
  const root = normalizeResourcePath(workspaceRoot)
  if (!root) return
  const state = getPermissionRequestSummaryState(root)
  const paths: Record<string, PermissionRequestPathSummary> = {}
  let ownPendingCount = 0
  let reviewPendingCount = 0

  const visit = (items: PermissionRequestSummaryTreeNode[]) => {
    for (const node of items) {
      const path = normalizeResourcePath(node.full_code_path || '')
      const own = Number(node.permission_requests?.own_pending_count || 0)
      const review = Number(node.permission_requests?.review_pending_count || 0)
      if (path && (own > 0 || review > 0)) {
        paths[path] = {
          own_pending_count: own,
          review_pending_count: review,
        }
        ownPendingCount += own
        reviewPendingCount += review
      }
      visit(node.children || [])
    }
  }
  visit(nodes)

  state.paths = paths
  state.ownPendingCount = ownPendingCount
  state.reviewPendingCount = reviewPendingCount
  state.loaded = true
}

export async function loadPermissionRequestSummary(
  workspaceRoot: string,
  options: { force?: boolean } = {},
): Promise<void> {
  const root = normalizeResourcePath(workspaceRoot)
  if (!root) return
  const state = getPermissionRequestSummaryState(root)
  if (state.loaded && !options.force) return
  const existing = inFlight.get(root)
  if (existing) return existing

  const request = getPermissionRequestSummary(root)
    .then((response) => {
      state.paths = response.paths || {}
      state.ownPendingCount = Number(response.own_pending_count || 0)
      state.reviewPendingCount = Number(response.review_pending_count || 0)
      state.loaded = true
    })
    .finally(() => {
      inFlight.delete(root)
    })
  inFlight.set(root, request)
  return request
}

export function permissionRequestPathSummary(
  state: PermissionRequestSummaryState,
  resourcePath: string,
): PermissionRequestPathSummary {
  return state.paths[normalizeResourcePath(resourcePath)] || {
    own_pending_count: 0,
    review_pending_count: 0,
  }
}

export function ownPendingPermissionRequestPaths(state: PermissionRequestSummaryState): Set<string> {
  return new Set(Object.entries(state.paths)
    .filter(([, summary]) => Number(summary.own_pending_count || 0) > 0)
    .map(([path]) => path))
}
