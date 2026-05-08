import { describe, expect, it } from 'vitest'
import type { FunctionDetail } from '@/architecture/domain/interfaces/IFunctionLoader'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import {
  buildDetailLookupSearchRequest,
  buildDetailEditFormState,
  buildDetailBaseQuery,
  buildEditFunctionDetail,
  findDetailIdField,
  findDetailRowMatch,
  filterDetailInitialData,
  getEditableFieldCodes,
  resolveDetailRouteRequest,
  shouldWaitForDetailTableData
} from './workspaceDetailRuntime'
import { getFormRequestFields } from '@/utils/functionSchemaSelectors'

const tableFunctionDetail: FunctionDetail = {
  id: 1,
  code: 'members',
  name: '成员表',
  template_type: TEMPLATE_TYPE.TABLE,
  router: '/members',
  schema: {
    version: 1,
    type: 'table',
    table: {
      request: [],
      fields: [
        {
          code: 'id',
          name: 'ID',
          hide: { scenes: ['create', 'update'] },
          widget: { type: 'ID' }
        },
        {
          code: 'name',
          name: '姓名',
          hide: { scenes: ['list', 'create'] },
          widget: { type: 'input' }
        },
        {
          code: 'title',
          name: '标题',
          widget: { type: 'input' }
        },
        {
          code: 'created_at',
          name: '创建时间',
          hide: { scenes: ['create', 'update'] },
          widget: { type: 'datetime' }
        }
      ]
    }
  }
}

describe('workspaceDetailRuntime', () => {
  it('builds edit function detail from editable table response fields', () => {
    const result = buildEditFunctionDetail(tableFunctionDetail)

    expect(result?.template_type).toBe(TEMPLATE_TYPE.FORM)
    expect(result?.schema?.form?.response).toEqual([])
    expect(getFormRequestFields(result).map(field => field.code)).toEqual(['name', 'title'])
  })

  it('keeps form function detail unchanged', () => {
    const formDetail: FunctionDetail = {
      id: 2,
      code: 'profile',
      name: '资料',
      template_type: TEMPLATE_TYPE.FORM,
      schema: {
        version: 1,
        type: 'form',
        form: {
          request: [
            {
              code: 'name',
              name: '姓名',
              widget: { type: 'input' }
            }
          ],
          response: []
        }
      }
    }

    expect(buildEditFunctionDetail(formDetail)).toBe(formDetail)
  })

  it('filters drawer initialData down to editable fields only', () => {
    const editFunctionDetail = buildEditFunctionDetail(tableFunctionDetail)
    const rowData = {
      id: 10,
      name: 'Alice',
      title: 'Engineer',
      created_at: 1710000000000
    }

    expect(filterDetailInitialData({
      rowData,
      editFunctionDetail
    })).toEqual({
      name: 'Alice',
      title: 'Engineer'
    })
  })

  it('marks edit form as ready when row data contains editable fields', () => {
    const editFunctionDetail = buildEditFunctionDetail(tableFunctionDetail)
    const rowData = {
      id: 10,
      name: 'Alice',
      title: 'Engineer',
      created_at: 1710000000000
    }

    expect(buildDetailEditFormState({
      rowData,
      editFunctionDetail
    })).toEqual({
      readiness: 'ready',
      editableFieldCodes: ['name', 'title'],
      initialData: {
        name: 'Alice',
        title: 'Engineer'
      }
    })
  })

  it('marks edit form as missing values when editable fields are absent from row data', () => {
    const editFunctionDetail = buildEditFunctionDetail(tableFunctionDetail)
    const rowData = {
      id: 10,
      created_at: 1710000000000
    }

    expect(buildDetailEditFormState({
      rowData,
      editFunctionDetail
    })).toEqual({
      readiness: 'missing-edit-values',
      editableFieldCodes: ['name', 'title'],
      initialData: {}
    })
  })

  it('marks edit form as having no editable fields when request schema is empty', () => {
    const editFunctionDetail: FunctionDetail = {
      id: 3,
      code: 'readonly-members',
      name: '只读成员表',
      template_type: TEMPLATE_TYPE.FORM,
      schema: {
        version: 1,
        type: 'form',
        form: {
          request: [],
          response: []
        }
      }
    }

    expect(buildDetailEditFormState({
      rowData: { id: 10, name: 'Alice' },
      editFunctionDetail
    })).toEqual({
      readiness: 'no-editable-fields',
      editableFieldCodes: [],
      initialData: {}
    })
  })

  it('resolves detail route requests from query params', () => {
    expect(resolveDetailRouteRequest({
      _tab: 'detail',
      _id: ['abc-42']
    })).toEqual({
      rowId: 'abc-42',
      key: 'detail:abc-42'
    })

    expect(resolveDetailRouteRequest({
      _tab: 'run',
      _id: '42'
    })).toBeNull()
  })

  it('finds matching detail rows by id or _id with string equality', () => {
    expect(findDetailRowMatch([
      { id: 1, name: 'Alice' },
      { _id: 'row-2', name: 'Bob' }
    ], 'row-2')).toEqual({
      row: { _id: 'row-2', name: 'Bob' },
      index: 1
    })
  })

  it('waits for initial table data during setup and function-loaded triggers only', () => {
    expect(shouldWaitForDetailTableData({
      loading: false,
      dataLength: 0,
      trigger: 'setup'
    })).toBe(true)

    expect(shouldWaitForDetailTableData({
      loading: false,
      dataLength: 0,
      trigger: 'function-loaded'
    })).toBe(true)

    expect(shouldWaitForDetailTableData({
      loading: false,
      dataLength: 0,
      trigger: 'route-change'
    })).toBe(false)

    expect(shouldWaitForDetailTableData({
      loading: true,
      dataLength: 10,
      trigger: 'route-change'
    })).toBe(true)
  })

  it('detects the detail id field from table response schema', () => {
    expect(findDetailIdField(tableFunctionDetail)?.code).toBe('id')
    expect(findDetailIdField(null)).toBeNull()
  })

  it('builds a standalone detail lookup search request without mutating table state', () => {
    expect(buildDetailLookupSearchRequest({
      detail: tableFunctionDetail,
      idFieldCode: 'id',
      rowId: '42'
    })).toEqual({
      url: '/workspace/api/v1/table/search/members',
      params: {
        id: '42',
        page: 1,
        page_size: 20
      }
    })
  })

  it('builds detail base query without detail params or form draft keys', () => {
    const editFunctionDetail = buildEditFunctionDetail(tableFunctionDetail)
    const query = {
      _tab: 'detail',
      _id: '10',
      page: '2',
      name: 'alice',
      title: 'draft-title',
      extra: ['a', 'b']
    }

    expect(buildDetailBaseQuery({
      query,
      editableFieldCodes: getEditableFieldCodes(editFunctionDetail)
    })).toEqual({
      page: '2',
      extra: ['a', 'b']
    })
  })
})
