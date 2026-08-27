import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MiniWorkstationSessionPanel from './MiniWorkstationSessionPanel.vue'

function mountPanel(overrides: Record<string, unknown> = {}) {
  return mount(MiniWorkstationSessionPanel, {
    props: {
      fullCodePath: '/demo/crm/customers.table',
      dirLabel: '客户表',
      sessions: [{
        session_id: 'session-1',
        title: '整理本周客户',
        status: 'running',
        created_at: '2026-08-27T08:00:00Z',
        updated_at: '2026-08-27T09:00:00Z',
      }] as any,
      activeSessionId: 'session-1',
      scope: 'current',
      sessionSourceFilter: 'human',
      automationAgents: [],
      automationMode: false,
      searchKeyword: '',
      filter: 'all',
      filters: [{ label: '全部', value: 'all' }],
      queuedCount: 0,
      loading: false,
      loadFailed: false,
      hasDifferentContext: false,
      currentContextName: '客户表',
      currentContextPath: '/demo/crm/customers.table',
      getSessionStatusClass: () => 'is-running',
      getSessionTitle: () => '整理本周客户',
      getSessionStatusLabel: () => '处理中',
      getSessionDirectoryPath: () => '客户管理 / 客户表',
      formatRelativeTime: () => '刚刚',
      ...overrides,
    },
    global: {
      stubs: {
        ElIcon: { template: '<span><slot /></span>' },
      },
    },
  })
}

describe('MiniWorkstationSessionPanel', () => {
  it('shows a compact status and time hierarchy for each session', () => {
    const wrapper = mountPanel({ scope: 'all' })

    expect(wrapper.find('.mini-session-toolbar').exists()).toBe(true)
    expect(wrapper.find('.mini-current-session-row.active').exists()).toBe(true)
    expect(wrapper.find('.mini-session-status-label').text()).toBe('处理中')
    expect(wrapper.find('.mini-current-session-directory').text()).toBe('客户管理 / 客户表')
    expect(wrapper.find('time').text()).toBe('刚刚')
  })

  it('does not repeat the directory path in current-directory scope', () => {
    const wrapper = mountPanel()

    expect(wrapper.find('.mini-current-session-directory').exists()).toBe(false)
    expect(wrapper.find('.mini-session-source-filter').exists()).toBe(false)
  })

  it('offers a new-session action in the panel header', async () => {
    const wrapper = mountPanel()

    await wrapper.find('.mini-session-head-actions button').trigger('click')

    expect(wrapper.emitted('new-session')).toHaveLength(1)
  })

  it('distinguishes filtered empty state from a truly empty directory', async () => {
    const wrapper = mountPanel({
      sessions: [],
      searchKeyword: '不存在',
      activeSessionId: undefined,
    })

    expect(wrapper.text()).toContain('暂无匹配会话')
    expect(wrapper.text()).toContain('清除筛选条件')
    await wrapper.find('.mini-current-session-row.is-draft').trigger('click')
    expect(wrapper.emitted('reset-filters')).toHaveLength(1)
    expect(wrapper.emitted('new-session')).toBeUndefined()
  })

  it('renders loading and retryable error states separately from empty state', async () => {
    const loading = mountPanel({ sessions: [], activeSessionId: undefined, loading: true })
    expect(loading.text()).toContain('正在加载会话')
    expect(loading.find('.mini-current-session-row.is-draft').exists()).toBe(false)

    const failed = mountPanel({ sessions: [], activeSessionId: undefined, loadFailed: true })
    expect(failed.text()).toContain('会话加载失败')
    await failed.find('.mini-session-state.is-error button').trigger('click')
    expect(failed.emitted('retry')).toHaveLength(1)
  })

  it('offers to restore the active session when filters hide it', async () => {
    const wrapper = mountPanel({
      sessions: [],
      searchKeyword: '其他会话',
    })

    expect(wrapper.text()).toContain('当前会话已被筛选隐藏')
    await wrapper.find('.mini-current-session-hidden button').trigger('click')
    expect(wrapper.emitted('show-active')).toHaveLength(1)
  })
})
