import { describe, expect, it } from 'vitest'
import type { ServiceTree } from '@/architecture/domain/types'
import { setLocale } from '@/architecture/shared/i18n'
import {
  buildServiceTreeNodeActionTestId,
  getServiceTreeNodeActions,
  type ServiceTreeNodeActionCommand
} from './serviceTreeNodeActions'

function node(overrides: Partial<ServiceTree>): ServiceTree {
  return {
    id: 1,
    name: '节点',
    code: 'node',
    type: 'package',
    description: '',
    tags: '',
    app_id: 1,
    ref_id: 1,
    full_code_path: '/user/app',
    created_at: '',
    updated_at: '',
    ...overrides
  }
}

function commands(actions: ReturnType<typeof getServiceTreeNodeActions>): ServiceTreeNodeActionCommand[] {
  return actions.map(action => action.command)
}

describe('serviceTreeNodeActions', () => {
  setLocale('zh-CN')

  it('shows package actions with directory import/export wording', () => {
    const actions = getServiceTreeNodeActions(node({}))

    expect(commands(actions)).toEqual([
      'create-directory',
      'create-docs',
      'open-workstation',
      'manage-access',
      'rename',
      'copy',
      'export-json',
      'import-directory'
    ])
    expect(actions.every(action => action.disabled)).toBe(true)
    expect(actions.find(action => action.command === 'export-json')?.label).toBe('导出目录')
    expect(actions.find(action => action.command === 'import-directory')?.label).toBe('导入目录')
  })

  it('keeps restricted actions visible with permission guidance', () => {
    const restricted = getServiceTreeNodeActions(node({}))
    expect(commands(restricted)).toContain('manage-access')
    expect(restricted.find(action => action.command === 'manage-access')).toMatchObject({
      disabled: true,
      disabledReason: '需要 Admin 权限'
    })

    const actions = getServiceTreeNodeActions(node({
      permissions: { read: true, admin: true }
    }))

    expect(commands(actions)).toContain('manage-access')
    expect(actions.find(action => action.command === 'manage-access')?.label).toBe('权限管理')
    expect(actions.find(action => action.command === 'manage-access')?.disabled).toBe(false)
  })

  it('only shows delete and paste for eligible non-root packages', () => {
    const actions = getServiceTreeNodeActions(
      node({
        full_code_path: '/user/app/tools',
        permissions: { read: true, admin: true }
      }),
      { hasCopiedDirectory: true }
    )

    expect(commands(actions)).toContain('delete-directory')
    expect(commands(actions)).toContain('paste')
  })

  it('shows delete-function for table deletable function nodes', () => {
    const actions = getServiceTreeNodeActions(node({
      type: 'function',
      full_code_path: '/user/app/tools/search.table',
      permissions: { delete: true }
    }))

    expect(commands(actions)).toEqual(['manage-access', 'delete-function'])
    expect(actions.find(action => action.command === 'manage-access')?.disabled).toBe(true)
  })

  it('builds stable test ids', () => {
    expect(buildServiceTreeNodeActionTestId('import-directory', node({ id: 42 }))).toBe('service-tree-action-import-directory-42')
  })
})
