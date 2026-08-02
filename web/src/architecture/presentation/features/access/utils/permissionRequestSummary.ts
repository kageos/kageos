import type { PermissionRequest } from '@/architecture/presentation/context/api/permission'
import { buildAppResourcePath, parseResourcePath } from '@/architecture/shared/resourcePath'

export interface PermissionRequestSummary {
  ownPendingCount: number
  reviewPendingCount: number
  totalCount: number
}

export function getPermissionRequestWorkspaceRoot(resourcePath: string): string {
  const parsed = parseResourcePath(resourcePath)
  return parsed ? buildAppResourcePath(parsed.user, parsed.app) : ''
}

export function summarizePermissionRequests(
  resourcePath: string,
  ownPendingRequests: PermissionRequest[],
  reviewPendingRequests: PermissionRequest[],
): PermissionRequestSummary {
  const ownIDs = new Set(
    ownPendingRequests
      .filter(request => request.resource_path === resourcePath && request.status === 'pending')
      .map(request => request.id),
  )
  const reviewIDs = new Set(
    reviewPendingRequests
      .filter(request => request.resource_path === resourcePath && request.status === 'pending')
      .map(request => request.id),
  )

  return {
    ownPendingCount: ownIDs.size,
    reviewPendingCount: reviewIDs.size,
    totalCount: new Set([...ownIDs, ...reviewIDs]).size,
  }
}
