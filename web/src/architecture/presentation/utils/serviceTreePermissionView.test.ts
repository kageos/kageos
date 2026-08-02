import { describe, expect, it } from 'vitest'
import type { ServiceTree } from '@/architecture/domain/types'
import {
  aggregatePermissionRequestSummaries,
  collectPermissionRequestExpandedDirectoryIds,
  filterServiceTreeByReadAccess,
} from './serviceTreePermissionView'

function node(input: Partial<ServiceTree> & Pick<ServiceTree, 'id' | 'full_code_path'>): ServiceTree {
  return {
    name: String(input.id),
    code: String(input.id),
    type: 'package',
    description: '',
    tags: '',
    app_id: 1,
    ref_id: 1,
    created_at: '',
    updated_at: '',
    ...input,
  }
}

describe('serviceTreePermissionView', () => {
  it('filters unreadable leaves but preserves an unreadable ancestor of a readable node', () => {
    const tree = [node({
      id: 1,
      full_code_path: '/system/demo',
      children: [
        node({ id: 2, full_code_path: '/system/demo/allowed', permissions: { read: true } }),
        node({ id: 3, full_code_path: '/system/demo/hidden', permissions: { read: false } }),
      ],
    })]

    const filtered = filterServiceTreeByReadAccess(tree)
    expect(filtered).toHaveLength(1)
    expect(filtered[0]?.children?.map(item => item.id)).toEqual([2])
  })

  it('aggregates descendant request counts while retaining leaf counts', () => {
    const tree = [node({
      id: 1,
      full_code_path: '/system/demo',
      children: [
        node({ id: 2, full_code_path: '/system/demo/a', type: 'function' }),
        node({ id: 3, full_code_path: '/system/demo/b', type: 'function' }),
      ],
    })]
    const summaries = aggregatePermissionRequestSummaries(tree, {
      '/system/demo/a': { review_pending_count: 2 },
      '/system/demo/b': { own_pending_count: 3 },
    })

    expect(summaries['/system/demo']).toEqual({ ownPendingCount: 3, reviewPendingCount: 2, totalCount: 5 })
    expect(summaries['/system/demo/a']?.totalCount).toBe(2)
    expect(summaries['/system/demo/b']?.totalCount).toBe(3)
    expect(collectPermissionRequestExpandedDirectoryIds(tree, summaries)).toEqual([1])
  })
})
