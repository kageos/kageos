import { describe, expect, it } from 'vitest'
import {
  formatWorkspaceRoleName,
  formatWorkspaceToolCallResultPreview,
  getVisibleWorkspaceToolCalls,
  getWorkspaceToolCallDisplayName
} from './workspaceRoleDisplay'

describe('workspaceRoleDisplay', () => {
  it('shows role switch tool calls alongside other tool calls', () => {
    expect(getVisibleWorkspaceToolCalls([
      { name: 'change_role' },
      { name: 'run_form_submit' },
    ])).toEqual([
      { name: 'change_role' },
      { name: 'run_form_submit' },
    ])
    expect(getWorkspaceToolCallDisplayName({ name: 'change_role' })).toBe('change_role')
  })

  it('formats internal role names as user-facing states', () => {
    expect(formatWorkspaceRoleName('app_operator', '应用操作员')).toBe('应用执行')
    expect(formatWorkspaceRoleName('', '自动化操作员')).toBe('自动执行配置')
  })

  it('summarizes change_role result data without dumping loaded documents', () => {
    const preview = formatWorkspaceToolCallResultPreview({
      name: 'change_role',
      arguments: JSON.stringify({ target_role: 'qa_engineer', execute_directory: '/demo/app/order' }),
      result_data: {
        previous_role_name: '应用搭建',
        display_name: '功能验证',
        switched: true,
        reason: '构建成功后进入验证',
        execute_directory: '/demo/app/order',
        next_action: '读取函数 schema 并测试核心流程',
        loaded_docs: [{ path: '/system/prompt/roles/qa-engineer', content: 'very long doc' }],
      },
    })

    expect(preview).toContain('已切换角色: 应用搭建 -> 功能验证')
    expect(preview).toContain('执行目录: /demo/app/order')
    expect(preview).toContain('角色文档: 已加载 1 个')
    expect(preview).not.toContain('very long doc')
  })
})
