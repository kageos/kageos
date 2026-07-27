export interface CapabilityBundleFile {
  package_path?: string
  path: string
  content: string
}

export interface CapabilityBundlePackage {
  path: string
  name?: string
  description?: string
  tags?: string
}

export interface CapabilityBundleTreeNode {
  relative_path: string
  parent_path?: string
  type: string
  code: string
  name?: string
  description?: string
  tags?: string[]
  template_type?: string
  method?: string
  router?: string
  sort_order?: number
}

export interface CapabilityBundleDoc {
  relative_path: string
  name?: string
  content: string
  format?: string
  summary?: string
  category?: string
}

export interface CapabilityBundleAgentTaskSchedule {
  type: 'atime' | 'cron' | 'every' | string
  run_at?: string
  cron_expr?: string
  interval_seconds?: number
  timezone?: string
  max_runs?: number
}

export interface CapabilityBundleAgentTask {
  relative_path: string
  code: string
  title?: string
  description?: string
  message: string
  enabled?: boolean
  schedule: CapabilityBundleAgentTaskSchedule
  mode_code?: string
  max_duration_seconds?: number
  policy?: string
  origin?: 'manifest' | 'user' | string
}

export interface CapabilityBundleScheduledFunction {
  relative_path: string
  code: string
  title?: string
  description?: string
  template_type?: string
  action: string
  method?: string
  body?: unknown
  default_enabled: boolean
  schedule: CapabilityBundleAgentTaskSchedule
  managed_by?: 'app_manifest' | 'capability_bundle' | string
  origin?: 'manifest' | 'user' | string
}

export interface CapabilityBundleDirectoryMetadata {
  code: string
  name?: string
  description?: string
  tags?: string[]
  source_revision?: string
  release_version?: string
}

export interface CapabilityBundleMetadata {
  directory?: CapabilityBundleDirectoryMetadata
}

export interface CapabilityBundle {
  schema_version: 'capability.bundle.v1'
  name?: string
  metadata?: CapabilityBundleMetadata
  tree_nodes?: CapabilityBundleTreeNode[]
  docs?: CapabilityBundleDoc[]
  files: CapabilityBundleFile[]
  scheduled_functions?: CapabilityBundleScheduledFunction[]
  agent_tasks?: CapabilityBundleAgentTask[]
  packages?: CapabilityBundlePackage[]
  extensions?: Record<string, unknown>
}
