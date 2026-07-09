import type { CapabilityBundle } from '@/architecture/domain/types'

const FALLBACK_CAPABILITY_BUNDLE_FILENAME = 'directory.json'

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
  return `${safeName}.directory.json`
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

function validateRelativeNodePath(value: string, field: string, allowEmpty = false): string {
  if (!value) {
    if (allowEmpty) return ''
    throw new Error(`${field} 不能为空`)
  }
  if (value.startsWith('/') || value.endsWith('/') || value.includes('\\')) {
    throw new Error(`${field} 必须是相对节点路径`)
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

function parentPathOf(relativePath: string): string {
  const parts = relativePath.split('/').filter(Boolean)
  if (parts.length <= 1) return ''
  return parts.slice(0, -1).join('/')
}

export function parseCapabilityBundleJson(text: string): CapabilityBundle {
  let raw: unknown
  try {
    raw = JSON.parse(text)
  } catch (_error) {
    throw new Error('JSON 格式不正确')
  }

  const object = ensurePlainObject(raw, '目录 JSON')
  if (object.schema_version !== 'capability.bundle.v1') {
    throw new Error('只支持目录 JSON（capability.bundle.v1）')
  }

  const rawPackages = object.packages
  const rawFiles = object.files
  const rawTreeNodes = object.tree_nodes
  const rawDocs = object.docs
  const rawAgentTasks = object.agent_tasks
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
  const treeNodes = Array.isArray(rawTreeNodes) ? rawTreeNodes.map((item, index) => {
    const node = ensurePlainObject(item, `tree_nodes[${index}]`)
    const relativePath = validateRelativeNodePath(ensureString(node.relative_path, `tree_nodes[${index}].relative_path`), `tree_nodes[${index}].relative_path`)
    const parentPath = validateRelativeNodePath(
      typeof node.parent_path === 'string' ? node.parent_path : '',
      `tree_nodes[${index}].parent_path`,
      true
    )
    const expectedParent = parentPathOf(relativePath)
    if (parentPath !== expectedParent) {
      throw new Error(`tree_nodes[${index}].parent_path 必须等于 relative_path 的父路径`)
    }
    const code = ensureString(node.code, `tree_nodes[${index}].code`)
    const pathCode = relativePath.split('/').filter(Boolean).at(-1)
    if (pathCode !== code) {
      throw new Error(`tree_nodes[${index}].code 必须等于 relative_path 的最后一段`)
    }
    return {
      relative_path: relativePath,
      parent_path: parentPath || undefined,
      type: ensureString(node.type, `tree_nodes[${index}].type`),
      code,
      name: typeof node.name === 'string' ? node.name : undefined,
      description: typeof node.description === 'string' ? node.description : undefined,
      tags: Array.isArray(node.tags) ? node.tags.filter((tag): tag is string => typeof tag === 'string') : undefined,
      template_type: typeof node.template_type === 'string' ? node.template_type : undefined,
      method: typeof node.method === 'string' ? node.method : undefined,
      router: typeof node.router === 'string' ? node.router : undefined,
      sort_order: typeof node.sort_order === 'number' ? node.sort_order : undefined
    }
  }) : []

  const treeNodePaths = new Set(treeNodes.map((node) => node.relative_path))
  const treeNodeByPath = new Map(treeNodes.map((node) => [node.relative_path, node]))
  treeNodes.forEach((node, index) => {
    if (node.parent_path && !treeNodePaths.has(node.parent_path)) {
      throw new Error(`tree_nodes[${index}].parent_path 未在 tree_nodes 中声明`)
    }
  })

  const docs = Array.isArray(rawDocs) ? rawDocs.map((item, index) => {
    const doc = ensurePlainObject(item, `docs[${index}]`)
    const relativePath = validateRelativeNodePath(ensureString(doc.relative_path, `docs[${index}].relative_path`), `docs[${index}].relative_path`)
    const treeNode = treeNodeByPath.get(relativePath)
    if (treeNodes.length > 0 && !treeNode) {
      throw new Error(`docs[${index}].relative_path 未在 tree_nodes 中声明`)
    }
    if (treeNode && treeNode.type !== 'docs') {
      throw new Error(`docs[${index}].relative_path 对应的 tree node 不是 docs`)
    }
    if (typeof doc.content !== 'string') {
      throw new Error(`docs[${index}].content 必须是字符串`)
    }
    return {
      relative_path: relativePath,
      name: typeof doc.name === 'string' ? doc.name : undefined,
      content: doc.content,
      format: typeof doc.format === 'string' ? doc.format : undefined,
      summary: typeof doc.summary === 'string' ? doc.summary : undefined,
      category: typeof doc.category === 'string' ? doc.category : undefined
    }
  }) : []

  const agentTasks = Array.isArray(rawAgentTasks) ? rawAgentTasks.map((item, index) => {
    const task = ensurePlainObject(item, `agent_tasks[${index}]`)
    const relativePath = validateRelativePackagePath(
      ensureString(task.relative_path, `agent_tasks[${index}].relative_path`),
      `agent_tasks[${index}].relative_path`
    )
    if (!packagePaths.has(relativePath)) {
      throw new Error(`agent_tasks[${index}].relative_path 未在 packages 中声明`)
    }
    const schedule = ensurePlainObject(task.schedule, `agent_tasks[${index}].schedule`)
    const scheduleType = ensureString(schedule.type, `agent_tasks[${index}].schedule.type`)
    if (!['atime', 'cron', 'every'].includes(scheduleType)) {
      throw new Error(`agent_tasks[${index}].schedule.type 不支持`)
    }
    if (scheduleType === 'cron' && typeof schedule.cron_expr !== 'string') {
      throw new Error(`agent_tasks[${index}].schedule.cron_expr 必须是字符串`)
    }
    if (scheduleType === 'every' && typeof schedule.interval_seconds !== 'number') {
      throw new Error(`agent_tasks[${index}].schedule.interval_seconds 必须是数字`)
    }
    if (scheduleType === 'atime' && typeof schedule.run_at !== 'string') {
      throw new Error(`agent_tasks[${index}].schedule.run_at 必须是字符串`)
    }
    const code = ensureString(task.code, `agent_tasks[${index}].code`)
    if (!code || /[\\/\s]/.test(code)) {
      throw new Error(`agent_tasks[${index}].code 必须是非空标识`)
    }
    const message = ensureString(task.message, `agent_tasks[${index}].message`)
    if (!message.trim()) {
      throw new Error(`agent_tasks[${index}].message 不能为空`)
    }
    return {
      relative_path: relativePath,
      code,
      title: typeof task.title === 'string' ? task.title : undefined,
      description: typeof task.description === 'string' ? task.description : undefined,
      message,
      enabled: typeof task.enabled === 'boolean' ? task.enabled : undefined,
      schedule: {
        type: scheduleType,
        run_at: typeof schedule.run_at === 'string' ? schedule.run_at : undefined,
        cron_expr: typeof schedule.cron_expr === 'string' ? schedule.cron_expr : undefined,
        interval_seconds: typeof schedule.interval_seconds === 'number' ? schedule.interval_seconds : undefined,
        timezone: typeof schedule.timezone === 'string' ? schedule.timezone : undefined,
        max_runs: typeof schedule.max_runs === 'number' ? schedule.max_runs : undefined
      },
      mode_code: typeof task.mode_code === 'string' ? task.mode_code : undefined,
      max_duration_seconds: typeof task.max_duration_seconds === 'number' ? task.max_duration_seconds : undefined,
      policy: typeof task.policy === 'string' ? task.policy : undefined
    }
  }) : []

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

  if (packages.length === 0 && files.length === 0 && docs.length === 0 && agentTasks.length === 0) {
    throw new Error('目录 JSON 必须包含 packages、files、docs 或 agent_tasks')
  }

  return {
    schema_version: 'capability.bundle.v1',
    name: typeof object.name === 'string' ? object.name : undefined,
    tree_nodes: treeNodes,
    docs,
    packages,
    files,
    agent_tasks: agentTasks,
    extensions: object.extensions === undefined
      ? undefined
      : ensurePlainObject(object.extensions, 'extensions') as Record<string, unknown>
  }
}
