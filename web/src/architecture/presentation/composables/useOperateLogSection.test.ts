import { effectScope, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useOperateLogSection } from './useOperateLogSection'

vi.mock('@/architecture/presentation/context/appStoresContext', () => ({
  useUserInfoStore: () => ({
    batchGetUserInfo: vi.fn().mockResolvedValue([])
  })
}))

vi.mock('@/architecture/presentation/context/api/function', () => ({
  getFunctionByPath: vi.fn()
}))

describe('useOperateLogSection', () => {
  it('uses schema field names when summarizing current function logs', () => {
    const scope = effectScope()

    try {
      const section = scope.run(() => useOperateLogSection({
        fullCodePath: ref('/liubeiluo/ee/work/ticket_management/ticket_list.table'),
        rowId: ref(0),
        functionDetail: ref({
          template_type: 'table',
          schema: {
            type: 'table',
            table: {
              fields: [
                { code: 'title', name: '工单标题', widget: { type: 'input', config: {} }, data: { type: 'string' } },
                { code: 'status', name: '工单状态', widget: { type: 'select', config: {} }, data: { type: 'string' } },
                { code: 'handler', name: '处理人', widget: { type: 'user', config: {} }, data: { type: 'string' } },
              ]
            }
          }
        }),
        autoLoad: ref(false),
        scope: ref('function')
      }))!

      const log = {
        id: 1,
        tenant_user: 'liubeiluo',
        request_user: 'liubeiluo',
        action: 'OnTableUpdateRow',
        app: 'ee',
        full_code_path: '/liubeiluo/ee/work/ticket_management/ticket_list.table',
        row_id: 12,
        updates: {
          title: '函数渲染界面',
          status: '待办',
          handler: 'liubeiluo'
        },
        old_values: {},
        created_at: '2026-05-19T00:00:00Z'
      }

      expect(section.getLogSummary(log)).toContain('工单标题: 函数渲染界面')
      expect(section.getLogSummary(log)).toContain('工单状态: 待办')
      expect(section.getLogSummary(log)).toContain('处理人: liubeiluo')
      expect(section.getLogSummary(log)).not.toContain('title:')
      expect(section.getLogSummary(log)).not.toContain('status:')
      expect(section.getLogSummary(log)).not.toContain('handler:')
    } finally {
      scope.stop()
    }
  })

  it('reads structured table audit details for status, duration, and error summary', () => {
    const scope = effectScope()

    try {
      const section = scope.run(() => useOperateLogSection({
        fullCodePath: ref('/alice/ops/tickets.table'),
        rowId: ref(42),
        functionDetail: ref({
          template_type: 'table',
          schema: { type: 'table', table: { fields: [] } }
        }),
        autoLoad: ref(false),
        scope: ref('row')
      }))!

      const log = {
        id: 2,
        tenant_user: 'alice',
        request_user: 'bob',
        action: 'OnTableUpdateRow',
        app: 'ops',
        full_code_path: '/alice/ops/tickets.table',
        row_id: 42,
        updates: {},
        old_values: {},
        created_at: '2026-05-19T00:00:00Z',
        status: 'failed',
        version: 'v10',
        details_json: {
          duration_millis: 35,
          response_body: {
            code: 500,
            error: 'boom',
            total_cost_mill: 35
          }
        }
      }

      expect(section.getLogDuration(log)).toBe(35)
      expect(section.getLogStatusLabel(log)).toBe('Failed')
      expect(section.getLogSummary(log)).toBe('boom')
      expect(section.getLogMetaEntries(log)).toEqual(expect.arrayContaining([
        { label: 'Duration', value: '35ms' },
        { label: 'Version', value: 'v10' },
        { label: 'Error', value: 'boom' },
      ]))
    } finally {
      scope.stop()
    }
  })
})
