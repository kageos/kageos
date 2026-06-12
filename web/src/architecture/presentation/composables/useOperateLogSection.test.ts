import { effectScope, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useOperateLogSection } from './useOperateLogSection'

const getOperateLogsMock = vi.hoisted(() => vi.fn())

vi.mock('@/architecture/presentation/context/appStoresContext', () => ({
  useUserInfoStore: () => ({
    batchGetUserInfo: vi.fn().mockResolvedValue([]),
  }),
}))

vi.mock('@/architecture/presentation/context/api/operateLog', () => ({
  getOperateLogs: getOperateLogsMock,
}))

vi.mock('@/architecture/presentation/context/api/function', () => ({
  getFunctionByPath: vi.fn(),
}))

const flushPromises = () => new Promise((resolve) => setTimeout(resolve, 0))

describe('useOperateLogSection', () => {
  it('uses schema field names when summarizing current function logs', () => {
    const scope = effectScope()

    try {
      const section = scope.run(() =>
        useOperateLogSection({
          fullCodePath: ref('/liubeiluo/ee/work/ticket_management/ticket_list.table'),
          rowId: ref(0),
          functionDetail: ref({
            template_type: 'table',
            schema: {
              type: 'table',
              table: {
                fields: [
                  {
                    code: 'title',
                    name: '工单标题',
                    widget: { type: 'input', config: {} },
                    data: { type: 'string' },
                  },
                  {
                    code: 'status',
                    name: '工单状态',
                    widget: { type: 'select', config: {} },
                    data: { type: 'string' },
                  },
                  {
                    code: 'handler',
                    name: '处理人',
                    widget: { type: 'user', config: {} },
                    data: { type: 'string' },
                  },
                ],
              },
            },
          }),
          autoLoad: ref(false),
          scope: ref('function'),
        }),
      )!

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
          handler: 'liubeiluo',
        },
        old_values: {},
        created_at: '2026-05-19T00:00:00Z',
      }

      expect(section.getLogSummary(log)).toContain('工单标题: 函数渲染界面')
      expect(section.getLogSummary(log)).toContain('工单状态: 待办')
      expect(section.getLogSummary(log)).toContain('处理人: liubeiluo')
      expect(section.getLogSummary(log)).not.toContain('title:')
      expect(section.getLogSummary(log)).not.toContain('status:')
      expect(section.getLogSummary(log)).not.toContain('handler:')

      const changes = section.getChangeEntries(log)
      expect(changes[0]?.field?.code).toBe('title')
      expect(changes[0]?.field?.widget?.type).toBe('input')
    } finally {
      scope.stop()
    }
  })

  it('reads structured table audit details for status, duration, and error summary', () => {
    const scope = effectScope()

    try {
      const section = scope.run(() =>
        useOperateLogSection({
          fullCodePath: ref('/alice/ops/tickets.table'),
          rowId: ref(42),
          functionDetail: ref({
            template_type: 'table',
            schema: { type: 'table', table: { fields: [] } },
          }),
          autoLoad: ref(false),
          scope: ref('row'),
        }),
      )!

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
            total_cost_mill: 35,
          },
        },
      }

      expect(section.getLogDuration(log)).toBe(35)
      expect(section.getLogStatusLabel(log)).toBe('Failed')
      expect(section.getLogSummary(log)).toBe('boom')
      expect(section.getLogMetaEntries(log)).toEqual(
        expect.arrayContaining([
          { label: 'Duration', value: '35ms' },
          { label: 'Version', value: 'v10' },
          { label: 'Error', value: 'boom' },
        ]),
      )
    } finally {
      scope.stop()
    }
  })

  it('labels scheduled task audit source', () => {
    const scope = effectScope()

    try {
      const section = scope.run(() =>
        useOperateLogSection({
          fullCodePath: ref('/alice/ops/tickets.table'),
          rowId: ref(42),
          functionDetail: ref({
            template_type: 'table',
            schema: { type: 'table', table: { fields: [] } },
          }),
          autoLoad: ref(false),
          scope: ref('row'),
        }),
      )!

      expect(section.getSourceLabel('scheduled_task')).toBe('Scheduled task')
      expect(section.sourceOptions.value).toEqual(expect.arrayContaining([{ label: 'Scheduled task', value: 'scheduled_task' }]))
      expect(
        section.getLogMetaEntries({
          id: 3,
          tenant_user: 'alice',
          request_user: 'alice',
          action: 'OnTableUpdateRow',
          app: 'ops',
          full_code_path: '/alice/ops/tickets.table',
          row_id: 42,
          created_at: '2026-05-19T00:00:00Z',
          source: 'scheduled_task',
        }),
      ).toEqual(expect.arrayContaining([{ label: 'Source', value: 'Scheduled task' }]))
    } finally {
      scope.stop()
    }
  })

  it('labels scheduled function execution and does not hide it behind function resource type filters', async () => {
    const scope = effectScope()
    getOperateLogsMock.mockReset()
    getOperateLogsMock.mockResolvedValue({ logs: [], total: 0, page: 1, page_size: 12 })

    try {
      const section = scope.run(() =>
        useOperateLogSection({
          fullCodePath: ref('/alice/ops/reports.chart'),
          rowId: ref(0),
          functionDetail: ref({
            template_type: 'chart',
            schema: { type: 'chart' },
          }),
          autoLoad: ref(false),
          scope: ref('function'),
        }),
      )!

      expect(section.getActionLabel('scheduled_function_execute')).toBe('Scheduled execution')
      expect(section.actionOptions.value).toEqual(
        expect.arrayContaining([{ label: 'Scheduled execution', value: 'scheduled_function_execute' }]),
      )

      section.load()
      await flushPromises()

      expect(getOperateLogsMock).toHaveBeenCalledWith(
        expect.objectContaining({
          resource_path: '/alice/ops/reports.chart',
        }),
      )
      expect(getOperateLogsMock.mock.calls[0]?.[0]).not.toHaveProperty('resource_type')
    } finally {
      scope.stop()
    }
  })

  it('renders legacy public form submit as a readable form submit', () => {
    const scope = effectScope()

    try {
      const section = scope.run(() =>
        useOperateLogSection({
          fullCodePath: ref('/system/demos/brandcheck/brand_availability.form'),
          rowId: ref(0),
          functionDetail: ref({
            template_type: 'form',
            schema: { type: 'form' },
          }),
          autoLoad: ref(false),
          scope: ref('function'),
        }),
      )!

      const log = {
        id: 21,
        tenant_user: 'system',
        request_user: 'guest_anon_123',
        action: 'public_form_submit',
        app: 'demos',
        full_code_path: '/system/demos/brandcheck/brand_availability.form',
        row_id: 0,
        status: 'success',
        summary: 'Public form submitted',
        source: 'public_share',
        created_at: '2026-05-22T08:41:13Z',
      }

      expect(section.getActionLabel(log.action)).toBe('Public submit')
      expect(section.getLogTitle(log)).toBe('Submitted public form')
      expect(section.getLogSummary(log)).toBe('Public form submitted')
      expect(section.actionOptions.value).toEqual(expect.arrayContaining([{ label: 'Public submit', value: 'public_form_submit' }]))
    } finally {
      scope.stop()
    }
  })

  it('renders service tree action enums as readable resource changes', () => {
    const scope = effectScope()

    try {
      const section = scope.run(() =>
        useOperateLogSection({
          fullCodePath: ref('/system/demos/meeting'),
          rowId: ref(0),
          functionDetail: ref(null),
          autoLoad: ref(false),
          scope: ref('directory'),
        }),
      )!

      const log = {
        id: 22,
        tenant_user: 'system',
        request_user: 'system',
        action: 'service_tree.node.created',
        app: 'demos',
        full_code_path: '/system/demos/meeting',
        row_id: 0,
        resource_type: 'directory',
        status: 'success',
        summary: 'system created package /system/demos/meeting',
        details_json: {
          node_type: 'package',
          full_code_path: '/system/demos/meeting',
        },
        created_at: '2026-06-12T00:58:55Z',
      }

      expect(section.getActionLabel(log.action)).toBe('Create')
      expect(section.getLogTitle(log)).toBe('Created directory')
      expect(section.getLogSummary(log)).toBe('directory /system/demos/meeting was created')
      expect(section.getActionTagType(log.action)).toBe('success')
      expect(section.actionOptions.value).toEqual(
        expect.arrayContaining([{ label: 'Resource created', value: 'service_tree.node.created' }]),
      )
    } finally {
      scope.stop()
    }
  })

  it('syncs expanded log row keys from table expand changes', () => {
    const scope = effectScope()

    try {
      const section = scope.run(() =>
        useOperateLogSection({
          fullCodePath: ref('/alice/ops/tickets.table'),
          rowId: ref(42),
          functionDetail: ref({
            template_type: 'table',
            schema: { type: 'table', table: { fields: [] } },
          }),
          autoLoad: ref(false),
          scope: ref('row'),
        }),
      )!

      const firstLog = {
        id: 11,
        tenant_user: 'alice',
        request_user: 'alice',
        action: 'OnTableUpdateRow',
        app: 'ops',
        full_code_path: '/alice/ops/tickets.table',
        row_id: 42,
        created_at: '2026-05-19T00:00:00Z',
      }
      const secondLog = {
        ...firstLog,
        id: 12,
      }

      section.handleLogExpandChange(firstLog, [secondLog])

      expect(section.expandedLogIds.value).toEqual([12])
    } finally {
      scope.stop()
    }
  })
})
