import type { CapabilityBundle } from '@/architecture/infrastructure/api/service-tree'

const FALLBACK_CAPABILITY_BUNDLE_FILENAME = 'capability-bundle.json'

function sanitizeFilenamePart(value: string): string {
  return value
    .trim()
    .replace(/[\\/:*?"<>|]+/g, '-')
    .replace(/\s+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 80)
}

export function buildCapabilityBundleFileName(bundle: CapabilityBundle, sourcePath?: string): string {
  const pathName = sourcePath?.split('/').filter(Boolean).pop()
  const rawName = bundle.name || pathName || bundle.packages?.[0]?.path || 'capability'
  const safeName = sanitizeFilenamePart(rawName)
  if (!safeName) {
    return FALLBACK_CAPABILITY_BUNDLE_FILENAME
  }
  return `${safeName}.capability-bundle.json`
}

export function downloadCapabilityBundleFile(bundle: CapabilityBundle, sourcePath?: string): void {
  const blob = new Blob([`${JSON.stringify(bundle, null, 2)}\n`], {
    type: 'application/json;charset=utf-8'
  })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = buildCapabilityBundleFileName(bundle, sourcePath)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

function ensurePlainObject(value: unknown, field: string): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${field} 必须是对象`)
  }
  return value as Record<string, unknown>
}

function ensureString(value: unknown, field: string): string {
  if (typeof value !== 'string') {
    throw new Error(`${field} 必须是字符串`)
  }
  if (value !== value.trim()) {
    throw new Error(`${field} 不能包含首尾空格`)
  }
  return value
}

function validateRelativePackagePath(value: string, field: string, allowEmpty = false): string {
  if (!value) {
    if (allowEmpty) return ''
    throw new Error(`${field} 不能为空`)
  }
  if (value.startsWith('/') || value.endsWith('/') || value.includes('\\')) {
    throw new Error(`${field} 必须是相对 package 路径`)
  }
  const parts = value.split('/')
  if (parts.some((part) => !part || part === '.' || part === '..' || part.startsWith('.'))) {
    throw new Error(`${field} 包含非法路径片段`)
  }
  if (parts[0] === 'namespace' || value.includes('code/api')) {
    throw new Error(`${field} 不能包含工作空间路径`)
  }
  return value
}

function validateRelativeFileName(value: string, field: string): string {
  if (!value || value.includes('/') || value.includes('\\') || value.startsWith('.')) {
    throw new Error(`${field} 必须是目录内直接文件名`)
  }
  if (value === '.' || value === '..' || value === 'init_.go') {
    throw new Error(`${field} 文件名非法`)
  }
  if (!value.includes('.')) {
    throw new Error(`${field} 必须包含文件扩展名`)
  }
  return value
}

export function parseCapabilityBundleJson(text: string): CapabilityBundle {
  let raw: unknown
  try {
    raw = JSON.parse(text)
  } catch (error) {
    throw new Error('JSON 格式不正确')
  }

  const object = ensurePlainObject(raw, '能力包')
  if (object.schema_version !== 'capability.bundle.v1') {
    throw new Error('只支持 capability.bundle.v1')
  }

  const rawPackages = object.packages
  const rawFiles = object.files
  const packages = Array.isArray(rawPackages) ? rawPackages.map((item, index) => {
    const pkg = ensurePlainObject(item, `packages[${index}]`)
    const packagePath = validateRelativePackagePath(ensureString(pkg.path, `packages[${index}].path`), `packages[${index}].path`)
    return {
      path: packagePath,
      name: typeof pkg.name === 'string' ? pkg.name : undefined,
      description: typeof pkg.description === 'string' ? pkg.description : undefined,
      tags: typeof pkg.tags === 'string' ? pkg.tags : undefined
    }
  }) : []

  const packagePaths = new Set(packages.map((pkg) => pkg.path))
  const files = Array.isArray(rawFiles) ? rawFiles.map((item, index) => {
    const file = ensurePlainObject(item, `files[${index}]`)
    const packagePath = validateRelativePackagePath(
      typeof file.package_path === 'string' ? file.package_path : '',
      `files[${index}].package_path`,
      true
    )
    if (packagePath && !packagePaths.has(packagePath)) {
      throw new Error(`files[${index}].package_path 未在 packages 中声明`)
    }
    if (typeof file.content !== 'string') {
      throw new Error(`files[${index}].content 必须是字符串`)
    }
    return {
      package_path: packagePath,
      path: validateRelativeFileName(ensureString(file.path, `files[${index}].path`), `files[${index}].path`),
      content: file.content
    }
  }) : []

  if (packages.length === 0 && files.length === 0) {
    throw new Error('能力包必须包含 packages 或 files')
  }

  return {
    schema_version: 'capability.bundle.v1',
    name: typeof object.name === 'string' ? object.name : undefined,
    packages,
    files
  }
}
