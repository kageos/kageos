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

export interface CapabilityBundle {
  schema_version: 'capability.bundle.v1'
  name?: string
  files: CapabilityBundleFile[]
  packages?: CapabilityBundlePackage[]
}
