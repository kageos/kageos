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
      'rename',
      'copy',
      'export-json',
      'import-directory'
    ])
    expect(actions.find(action => action.command === 'export-json')?.label).toBe('导出目录')
    expect(actions.find(action => action.command === 'import-directory')?.label).toBe('导入目录')
  })

  it('shows permission management only for nodes with admin access', () => {
    expect(commands(getServiceTreeNodeActions(node({})))).not.toContain('manage-access')

    const actions = getServiceTreeNodeActions(node({
      permissions: { read: true, admin: true }
    }))

    expect(commands(actions)).toContain('manage-access')
    expect(actions.find(action => action.command === 'manage-access')?.label).toBe('权限管理')
  })

  it('only shows delete and paste for eligible non-root packages', () => {
    const actions = getServiceTreeNodeActions(
      node({
        full_code_path: '/user/app/tools'
      }),
      { hasCopiedDirectory: true }
    )

    expect(commands(actions)).toContain('delete-directory')
    expect(commands(actions)).toContain('paste')
  })

  it('shows delete-function for table deletable function nodes', () => {
    const actions = getServiceTreeNodeActions(node({
      type: 'function',
      full_code_path: '/user/app/tools/search.table'
    }))

    expect(commands(actions)).toEqual(['delete-function'])
  })

  it('builds stable test ids', () => {
    expect(buildServiceTreeNodeActionTestId('import-directory', node({ id: 42 }))).toBe('service-tree-action-import-directory-42')
  })
})
