import { describe, expect, it } from 'vitest'
import { TablePermission } from '@/utils/permission'
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

  it('distinguishes unsupported edit from missing update permission', () => {
    expect(resolveTableDetailEditAccess({ supportsUpdate: false, canUpdate: true })).toBe('unsupported')
    expect(resolveTableDetailEditAccess({ supportsUpdate: true, canUpdate: false })).toBe('no-permission')
    expect(resolveTableDetailEditAccess({ supportsUpdate: true, canUpdate: true })).toBe('allowed')
  })

  it('routes action commands to link, edit, delete or no-permission', () => {
    expect(
      resolveTableActionCommand({
        command: 'link:profile',
        canUpdate: false,
        canDelete: false
      })
    ).toEqual({
      type: 'link',
      fieldCode: 'profile'
    })

    expect(
      resolveTableActionCommand({
        command: 'update',
        canUpdate: true,
        canDelete: false
      })
    ).toEqual({
      type: 'detail',
      initialMode: 'edit'
    })

    expect(
      resolveTableActionCommand({
        command: 'delete',
        canUpdate: false,
        canDelete: false
      })
    ).toEqual({
      type: 'no-permission',
      action: TablePermission.delete
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
