import { describe, expect, it } from 'vitest'
import { formatWorkspaceRoleName, getVisibleWorkspaceToolCalls } from './workspaceRoleDisplay'

describe('workspaceRoleDisplay', () => {
  it('hides internal role handoff tool calls', () => {
    expect(getVisibleWorkspaceToolCalls([
      { name: 'change_role' },
      { name: 'run_form_submit' },
    ])).toEqual([{ name: 'run_form_submit' }])
  })

  it('formats internal role names as user-facing states', () => {
    expect(formatWorkspaceRoleName('app_operator', '应用操作员')).toBe('应用执行')
    expect(formatWorkspaceRoleName('', '自动化操作员')).toBe('自动执行配置')
  })
})
