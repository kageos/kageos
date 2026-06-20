import type { App, ServiceTree } from './api'

export type WorkspaceAccessErrorType = 'forbidden' | 'load_failed'

export interface WorkspaceAccessError {
  type: WorkspaceAccessErrorType
  message: string
  resourcePath: string
}

export interface WorkspaceState {
  currentApp: App | null
  currentFunction: ServiceTree | null
  currentDirectory: ServiceTree | null
  serviceTree: ServiceTree[]
  loading: boolean
  accessError: WorkspaceAccessError | null
}
