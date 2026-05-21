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

export interface CapabilityBundle {
  schema_version: 'capability.bundle.v1'
  name?: string
  tree_nodes?: CapabilityBundleTreeNode[]
  docs?: CapabilityBundleDoc[]
  files: CapabilityBundleFile[]
  packages?: CapabilityBundlePackage[]
}
