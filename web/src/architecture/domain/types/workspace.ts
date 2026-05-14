import type { App, ServiceTree } from './api'

export interface WorkspaceState {
  currentApp: App | null
  currentFunction: ServiceTree | null
  currentDirectory: ServiceTree | null
  serviceTree: ServiceTree[]
  loading: boolean
}

