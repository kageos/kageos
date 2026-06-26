type ToolCallLike = {
  name?: string
}

const internalToolCallNames = new Set(['change_role'])

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
  return internalToolCallNames.has(normalizeToolCallName(call?.name))
}

export function getVisibleWorkspaceToolCalls<T extends ToolCallLike>(
  calls: readonly T[] | null | undefined
): T[] {
  return (calls ?? []).filter((call) => !isInternalWorkspaceToolCall(call))
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
