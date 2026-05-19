import { effectScope, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useOperateLogSection } from './useOperateLogSection'
import type { TableOperateLog } from '@/architecture/presentation/context/api/operateLog'

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
      } satisfies TableOperateLog

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
})
