import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { setLocale, translate } from '@/architecture/shared/i18n'
import WorkspaceUpdateLogDetails from './WorkspaceUpdateLogDetails.vue'
import { workspaceUpdateSummary, workspaceUpdateTitle, type WorkspaceUpdateLog } from '../composables/workspaceUpdateLog'

function logWithChanges(): WorkspaceUpdateLog {
  return {
    status: 'success',
    old_values_json: { version: 'v12' },
    new_values_json: { version: 'v13' },
    details_json: {
      outcome: 'completed',
      changes: {
        mode: 'diff',
        added: [{ name: '导入客户', path: '/alice/ops/import.form', type: 'form' }],
        updated: [{ name: '客户列表', path: '/alice/ops/customers.table', type: 'table' }],
        deleted: [{ name: '旧版统计', path: '/alice/ops/old.chart', type: 'chart' }],
        synced: [],
      },
    },
  }
}

describe('workspace update logs', () => {
  it('renders historical function names, types and paths with version/count summary', () => {
    const wrapper = mount(WorkspaceUpdateLogDetails, { props: { log: logWithChanges() } })
    expect(wrapper.text()).toContain('v12 → v13')
    expect(wrapper.text()).toContain('功能新增 1 项、修改 1 项、删除 1 项')
    expect(wrapper.findAll('.update-group')).toHaveLength(3)
    expect(wrapper.text()).toContain('旧版统计')
    expect(wrapper.text()).toContain('图表')
    expect(wrapper.text()).toContain('/alice/ops/old.chart')
  })

  it('does not present forced synchronization as additions', () => {
    const log = logWithChanges()
    log.details_json!.force_diff = true
    log.details_json!.changes!.mode = 'resync'
    log.details_json!.changes!.synced = log.details_json!.changes!.added
    const wrapper = mount(WorkspaceUpdateLogDetails, { props: { log } })
    expect(wrapper.findAll('.update-group')).toHaveLength(1)
    expect(wrapper.get('h4').text()).toContain('重新同步的功能')
    expect(workspaceUpdateSummary(log, translate)).not.toContain('功能新增')
  })

  it('distinguishes unavailable historical details from an empty diff', () => {
    const legacy = mount(WorkspaceUpdateLogDetails, { props: { log: { status: 'success' } } })
    expect(legacy.text()).toContain('功能变更明细不可用')
    expect(legacy.text()).not.toContain('未检测到')
    const log = logWithChanges()
    log.details_json!.changes = { mode: 'diff', added: [], updated: [], deleted: [], synced: [] }
    expect(workspaceUpdateSummary(log, translate)).toContain('未检测到功能定义变更')
  })

  it('shows synchronization warnings and failed finalization without claiming completion', () => {
    const log = logWithChanges()
    log.details_json!.warnings = ['应用已发布，但函数元数据同步失败']
    const wrapper = mount(WorkspaceUpdateLogDetails, { props: { log } })
    expect(wrapper.text()).toContain('不能据此认定平台元数据已全部同步')
    expect(wrapper.text()).toContain(log.details_json!.warnings[0])
    expect(workspaceUpdateTitle(log, translate)).toBe('已发布，存在告警')
    log.status = 'failed'
    log.details_json!.outcome = 'finalization_failed'
    expect(workspaceUpdateTitle(log, translate)).toBe('已发布，版本记录未完成')
  })

  it('does not show published changes for source-only writes and escapes log text', () => {
    const log = logWithChanges()
    log.details_json!.write_only = true
    log.details_json!.change_description = '<script>alert(1)</script>'
    const wrapper = mount(WorkspaceUpdateLogDetails, { props: { log } })
    expect(wrapper.findAll('.update-group')).toHaveLength(0)
    expect(wrapper.text()).toContain('本次未编译或发布')
    expect(wrapper.find('script').exists()).toBe(false)
    expect(wrapper.text()).toContain('<script>alert(1)</script>')
  })

  it('does not claim a failed source-only write was published', () => {
    const log = logWithChanges()
    log.status = 'failed'
    log.details_json!.write_only = true
    log.details_json!.outcome = 'finalization_failed'
    expect(workspaceUpdateTitle(log, translate)).toBe('更新工作空间失败')
    expect(workspaceUpdateSummary(log, translate)).toContain('仅写入操作未完成')
  })

  it('renders English summaries and function groups', () => {
    setLocale('en-US')
    const wrapper = mount(WorkspaceUpdateLogDetails, { props: { log: logWithChanges() } })
    expect(wrapper.text()).toContain('Functions: 1 added, 1 updated, 1 deleted')
    expect(wrapper.text()).toContain('Deleted functions')
  })
})
