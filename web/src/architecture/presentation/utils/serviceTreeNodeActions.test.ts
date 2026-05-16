import { describe, expect, it } from 'vitest'
import type { ServiceTree } from '@/architecture/domain/types'
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
  it('shows package actions with capability bundle wording', () => {
    const actions = getServiceTreeNodeActions(node({}))

    expect(commands(actions)).toEqual([
      'create-directory',
      'create-docs',
      'open-workstation',
      'rename',
      'copy',
      'export-json',
      'import-json'
    ])
    expect(actions.find(action => action.command === 'export-json')?.label).toBe('导出能力包')
    expect(actions.find(action => action.command === 'import-json')?.label).toBe('导入能力包')
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
    expect(buildServiceTreeNodeActionTestId('import-json', node({ id: 42 }))).toBe('service-tree-action-import-json-42')
  })
})
