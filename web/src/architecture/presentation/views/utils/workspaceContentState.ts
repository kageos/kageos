import type { ServiceTree } from '@/architecture/domain/types'
import { canRead } from '@/architecture/presentation/composables/useAccessControl'

export type WorkspaceContentState =
  | 'workspace-error'
  | 'resource-permission'
  | 'resource-locked'
  | 'create'
  | 'edit'
  | 'docs'
  | 'package'
  | 'function'
  | 'restoring'
  | 'empty'

export interface ResolveWorkspaceContentStateOptions {
  hasWorkspaceAccessError: boolean
  currentFunction: ServiceTree | null
  queryTab: string
  panel?: string
  hasCurrentFunctionDetail: boolean
  isRestoringWorkspaceRoute: boolean
}

export function resolveWorkspaceContentState(
  options: ResolveWorkspaceContentStateOptions
): WorkspaceContentState {
  if (options.hasWorkspaceAccessError) {
    return 'workspace-error'
  }

  if (options.currentFunction && !canRead(options.currentFunction)) {
    if (options.panel === 'permission') {
      return 'resource-permission'
    }
    return 'resource-locked'
  }

  if (options.queryTab === 'create' && options.currentFunction && options.hasCurrentFunctionDetail) {
    return 'create'
  }

  if (options.queryTab === 'edit' && options.currentFunction && options.hasCurrentFunctionDetail) {
    return 'edit'
  }

  if (options.currentFunction) {
    return options.currentFunction.type
  }

  if (options.isRestoringWorkspaceRoute) {
    return 'restoring'
  }

  return 'empty'
}
