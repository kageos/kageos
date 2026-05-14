import { describe, expect, it } from 'vitest'
import {
  buildBatchDeleteIds,
  hasFunctionCallback,
  resolveTableDetailEditAccess,
  resolveTableActionCommand
} from './tableViewActionRuntime'

describe('tableViewActionRuntime', () => {
  it('detects function callbacks from schema callback arrays', () => {
    expect(hasFunctionCallback(['OnTableAddRow', 'OnTableUpdateRow'], 'OnTableUpdateRow')).toBe(true)
    expect(hasFunctionCallback(undefined, 'OnTableUpdateRow')).toBe(false)
  })

  it('distinguishes unsupported edit from allowed edit', () => {
    expect(resolveTableDetailEditAccess({ supportsUpdate: false })).toBe('unsupported')
    expect(resolveTableDetailEditAccess({ supportsUpdate: true })).toBe('allowed')
  })

  it('routes action commands to link, edit or delete', () => {
    expect(
      resolveTableActionCommand({
        command: 'link:profile'
      })
    ).toEqual({
      type: 'link',
      fieldCode: 'profile'
    })

    expect(
      resolveTableActionCommand({
        command: 'update'
      })
    ).toEqual({
      type: 'detail',
      initialMode: 'edit'
    })

    expect(
      resolveTableActionCommand({
        command: 'delete'
      })
    ).toEqual({
      type: 'delete'
    })
  })

  it('extracts numeric ids for batch delete from row.id or explicit id field', () => {
    expect(
      buildBatchDeleteIds(
        [
          { id: 1, name: 'alice' },
          { user_id: 2, name: 'bob' },
          { id: '3', name: 'charlie' },
          { name: 'missing' }
        ] as any,
        'user_id'
      )
    ).toEqual([1, 2])
  })
})
