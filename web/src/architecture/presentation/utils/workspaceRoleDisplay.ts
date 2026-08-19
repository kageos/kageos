type ToolCallLike = {
  name?: string
  arguments?: string
  result_data?: unknown
}

const ROLE_CHANGE_TOOL_NAME = 'change_role'

const workspaceRoleLabels: Record<string, string> = {
  router: '智能调度',
  product_manager: '需求整理',
  app_developer: '应用搭建',
  maintenance_engineer: '应用维护',
  qa_engineer: '功能验证',
  app_operator: '应用执行',
  automation_operator: '自动执行配置',
  build_engineer: '构建修复',
  data_operator: '数据处理',
  platform_engineer: '平台集成',
  reviewer: '只读分析'
}

const workspaceRoleAliases: Record<string, string> = {
  工作台调度员: 'router',
  产品经理: 'product_manager',
  应用开发工程师: 'app_developer',
  应用维护工程师: 'maintenance_engineer',
  测试工程师: 'qa_engineer',
  应用执行: 'app_operator',
  应用操作员: 'app_operator',
  自动执行配置: 'automation_operator',
  自动化操作员: 'automation_operator',
  构建修复工程师: 'build_engineer',
  '数据/文件处理工程师': 'data_operator',
  平台集成工程师: 'platform_engineer',
  代码审查分析师: 'reviewer'
}

export function isInternalWorkspaceToolCall(call: ToolCallLike | null | undefined): boolean {
  void call
  return false
}

export function getVisibleWorkspaceToolCalls<T extends ToolCallLike>(
  calls: readonly T[] | null | undefined
): T[] {
  return [...(calls ?? [])]
}

export function getWorkspaceToolCallDisplayName(call: ToolCallLike | null | undefined): string {
  return String(call?.name || '').trim()
}

export function formatWorkspaceToolCallResultPreview(call: ToolCallLike | null | undefined): string {
  if (normalizeToolCallName(call?.name) !== ROLE_CHANGE_TOOL_NAME) return ''

  const data = asRecord(call?.result_data)
  if (!data) return ''
  const args = parseToolArguments(call?.arguments)

  const previousRole = firstNonEmpty(
    stringFromRecord(data, 'previous_role_name'),
    formatWorkspaceRoleName(stringFromRecord(data, 'previous_role')),
    formatWorkspaceRoleName(stringFromRecord(args, 'current_role'))
  )
  const currentRole = firstNonEmpty(
    stringFromRecord(data, 'display_name'),
    formatWorkspaceRoleName(stringFromRecord(data, 'role_id')),
    formatWorkspaceRoleName(stringFromRecord(data, 'current_role')),
    formatWorkspaceRoleName(stringFromRecord(args, 'target_role'))
  )
  const switched = boolFromRecord(data, 'switched')
  const transition = previousRole && currentRole && previousRole !== currentRole
    ? `${previousRole} -> ${currentRole}`
    : currentRole || previousRole

  const lines: string[] = []
  if (transition) {
    lines.push(`${switched === false ? '当前角色' : '已切换角色'}: ${transition}`)
  } else {
    lines.push('已调用角色切换')
  }

  const handoff = asRecord(data?.handoff)
  const executeDirectory = firstNonEmpty(
    stringFromRecord(data, 'execute_directory'),
    stringFromRecord(handoff, 'execute_directory'),
    stringFromRecord(args, 'execute_directory')
  )
  if (executeDirectory) lines.push(`执行服务目录: ${executeDirectory}`)

  const reason = stringFromRecord(data, 'reason')
  if (reason) lines.push(`原因: ${compactPreviewText(reason, 180)}`)

  const nextAction = stringFromRecord(data, 'next_action')
  if (nextAction) lines.push(`下一步: ${compactPreviewText(nextAction, 180)}`)

  const loadedDocs = arrayFromRecord(data, 'loaded_docs')
  const missingDocs = arrayFromRecord(data, 'missing_docs')
  if (loadedDocs.length > 0 || missingDocs.length > 0) {
    lines.push(`角色文档: 已加载 ${loadedDocs.length} 个${missingDocs.length > 0 ? `，缺失 ${missingDocs.length} 个` : ''}`)
  }

  return lines.join('\n')
}

export function formatWorkspaceRoleName(roleId?: string, displayName?: string): string {
  const normalizedRoleId = normalizeWorkspaceRoleId(roleId)
    || normalizeWorkspaceRoleId(displayName)
  if (normalizedRoleId && workspaceRoleLabels[normalizedRoleId]) {
    return workspaceRoleLabels[normalizedRoleId]
  }
  return compactRoleDisplayName(displayName || roleId || '')
}

function normalizeToolCallName(name?: string): string {
  return String(name || '').trim().toLowerCase()
}

function normalizeWorkspaceRoleId(value?: string): string {
  const raw = String(value || '').trim()
  if (!raw) return ''

  const normalized = raw.toLowerCase().replace(/-/g, '_')
  if (workspaceRoleLabels[normalized]) return normalized

  const aliased = workspaceRoleAliases[raw]
  if (aliased) return aliased

  for (const roleId of Object.keys(workspaceRoleLabels)) {
    if (normalized.includes(roleId)) return roleId
  }
  return ''
}

function compactRoleDisplayName(value: string): string {
  return value.trim().replace(/\s+/g, ' ')
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value != null && typeof value === 'object' && !Array.isArray(value)
    ? value as Record<string, unknown>
    : null
}

function parseToolArguments(value?: string): Record<string, unknown> | null {
  const text = String(value || '').trim()
  if (!text) return null
  try {
    return asRecord(JSON.parse(text))
  } catch {
    return null
  }
}

function stringFromRecord(record: Record<string, unknown> | null, key: string): string {
  const value = record?.[key]
  return typeof value === 'string' ? value.trim() : ''
}

function boolFromRecord(record: Record<string, unknown> | null, key: string): boolean | undefined {
  const value = record?.[key]
  return typeof value === 'boolean' ? value : undefined
}

function arrayFromRecord(record: Record<string, unknown> | null, key: string): unknown[] {
  const value = record?.[key]
  return Array.isArray(value) ? value : []
}

function firstNonEmpty(...values: string[]): string {
  for (const value of values) {
    const text = compactRoleDisplayName(value)
    if (text) return text
  }
  return ''
}

function compactPreviewText(value: string, maxLength: number): string {
  const text = compactRoleDisplayName(value)
  if (text.length <= maxLength) return text
  return `${text.slice(0, Math.max(0, maxLength - 1))}…`
}
