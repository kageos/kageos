import { describe, expect, it } from 'vitest'
import type { ServiceTree } from '@/types'
import { DirectoryPermission, TablePermission } from '@/utils/permission'
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
    permissions: {},
    ...overrides
  }
}

function commands(actions: ReturnType<typeof getServiceTreeNodeActions>): ServiceTreeNodeActionCommand[] {
  return actions.map(action => action.command)
}

describe('serviceTreeNodeActions', () => {
  it('shows read/write package actions with capability bundle wording', () => {
    const actions = getServiceTreeNodeActions(node({
      permissions: {
        [DirectoryPermission.read]: true,
        [DirectoryPermission.write]: true,
        [DirectoryPermission.update]: true
      }
    }))

    expect(commands(actions)).toEqual([
      'apply-permission',
      'create-directory',
      'create-docs',
      'create-board',
      'open-workstation',
      'rename',
      'copy',
      'export-json',
      'import-json',
      'publish-to-hub'
    ])
    expect(actions.find(action => action.command === 'export-json')?.label).toBe('导出能力包')
    expect(actions.find(action => action.command === 'import-json')?.label).toBe('导入能力包')
  })

  it('only shows delete and paste for eligible non-root packages', () => {
    const actions = getServiceTreeNodeActions(
      node({
        full_code_path: '/user/app/tools',
        permissions: {
          [DirectoryPermission.read]: true,
          [DirectoryPermission.write]: true,
          [DirectoryPermission.delete]: true
        }
      }),
      { hasCopiedDirectory: true }
    )

    expect(commands(actions)).toContain('delete-directory')
    expect(commands(actions)).toContain('paste')
  })

  it('switches hub actions between publish and push', () => {
    const permissions = {
      [DirectoryPermission.read]: true,
      [DirectoryPermission.write]: true
    }

    expect(commands(getServiceTreeNodeActions(node({ permissions })))).toContain('publish-to-hub')
    expect(commands(getServiceTreeNodeActions(node({
      hub_full_code_path: '/hub/tools',
      permissions
    })))).toContain('push-to-hub')
  })

  it('shows delete-function for table deletable function nodes', () => {
    const actions = getServiceTreeNodeActions(node({
      type: 'function',
      full_code_path: '/user/app/tools/search.table',
      permissions: {
        [TablePermission.delete]: true
      }
    }))

    expect(commands(actions)).toEqual(['apply-permission', 'delete-function'])
  })

  it('builds stable test ids', () => {
    expect(buildServiceTreeNodeActionTestId('import-json', node({ id: 42 }))).toBe('service-tree-action-import-json-42')
  })
})
