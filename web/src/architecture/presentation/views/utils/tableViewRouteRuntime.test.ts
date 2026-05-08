import { describe, expect, it } from 'vitest'
import {
  buildTableAddDialogOpenRequest,
  buildTableCreateDialogCloseRequest,
  buildTableDetailRowPayload,
  buildTableLinkRouteRequest,
  resolveTableAddDialogVisibility
} from './tableViewRouteRuntime'

describe('tableViewRouteRuntime', () => {
  it('builds internal link route request with parsed query params', () => {
    expect(buildTableLinkRouteRequest('/workspace/demo?page=2&keyword=alice')).toEqual({
      path: '/workspace/demo',
      query: {
        page: '2',
        keyword: 'alice'
      },
      replace: false,
      preserveParams: {
        linkNavigation: true
      }
    })
  })

  it('keeps raw business params for add-dialog link navigation', () => {
    expect(
      buildTableLinkRouteRequest('/workspace/demo/resume_list.table?_tab=OnTableAddRow&_link_type=table&job_id=2&owner=alice')
    ).toEqual({
      path: '/workspace/demo/resume_list.table',
      query: {
        _tab: 'OnTableAddRow',
        _link_type: 'table',
        job_id: '2',
        owner: 'alice'
      },
      replace: false,
      preserveParams: {
        linkNavigation: true
      }
    })
  })

  it('opens add dialog by preserving current query and setting _tab', () => {
    expect(
      buildTableAddDialogOpenRequest({
        page: 3,
        status: 'open',
        _node_type: 'table'
      })
    ).toEqual({
      query: {
        page: '3',
        status: 'open',
        _node_type: 'table',
        _tab: 'OnTableAddRow'
      },
      replace: true,
      preserveParams: {
        state: true
      }
    })
  })

  it('closes add dialog and removes form draft params while keeping table/search params', () => {
    expect(
      buildTableCreateDialogCloseRequest({
        routeQuery: {
          _tab: 'OnTableAddRow',
          page: '2',
          status: 'open',
          name: 'alice',
          email: 'a@example.com'
        },
        responseFieldCodes: ['name', 'email']
      })
    ).toEqual({
      query: {
        page: '2',
        status: 'open'
      },
      replace: true,
      preserveParams: {
        table: true,
        search: true,
        state: true
      }
    })
  })

  it('does not build close request when add dialog is not active', () => {
    expect(
      buildTableCreateDialogCloseRequest({
        routeQuery: {
          page: '2'
        },
        responseFieldCodes: ['name']
      })
    ).toBeNull()
  })

  it('builds detail payload with index and shared table data', () => {
    expect(
      buildTableDetailRowPayload({
        row: { id: 2, name: 'bob' },
        tableData: [
          { id: 1, name: 'alice' },
          { id: 2, name: 'bob' }
        ],
        initialMode: 'edit'
      })
    ).toEqual({
      row: { id: 2, name: 'bob' },
      index: 1,
      tableData: [
        { id: 1, name: 'alice' },
        { id: 2, name: 'bob' }
      ],
      initialMode: 'edit'
    })
  })

  it('resolves add dialog visibility from route state', () => {
    expect(
      resolveTableAddDialogVisibility({
        query: { _tab: 'OnTableAddRow' },
        hasAddCallback: true,
        isMounted: true,
        currentVisible: false
      })
    ).toBe(true)

    expect(
      resolveTableAddDialogVisibility({
        query: {},
        hasAddCallback: true,
        isMounted: true,
        currentVisible: true
      })
    ).toBe(false)
  })
})
