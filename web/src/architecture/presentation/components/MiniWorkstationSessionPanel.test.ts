import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it } from 'vitest'
import MiniWorkstationSessionPanel from './MiniWorkstationSessionPanel.vue'
import type { WorkspaceSessionItem } from '@/architecture/presentation/context/api/workspace'

const SlotStub = defineComponent({ template: '<span><slot /></span>' })

function session(overrides: Partial<WorkspaceSessionItem> = {}): WorkspaceSessionItem {
  return {
    session_id: 'session-1',
    title: '人工处理',
    status: 'active',
    created_at: '2026-07-12T10:00:00Z',
    updated_at: '2026-07-12T10:00:00Z',
    ...overrides
  }
}

function mountPanel(overrides: Record<string, unknown> = {}) {
  return mount(MiniWorkstationSessionPanel, {
    props: {
      fullCodePath: '/alice/demo',
      dirLabel: 'demo',
      sessions: [session()],
      activeSessionId: undefined,
      scope: 'current',
      sessionSourceFilter: 'human',
      automationAgents: [{ task_id: 7, task_code: 'daily', task_title: '每日复盘' }],
      automationMode: false,
      searchKeyword: '',
      filter: 'all',
      filters: [{ label: '全部', value: 'all' }],
      queuedCount: 0,
      hasDifferentContext: false,
      currentContextName: 'demo',
      currentContextPath: '/alice/demo',
      getSessionStatusClass: () => 'is-active',
      getSessionTitle: (item: WorkspaceSessionItem) => item.title,
      getSessionStatusLabel: () => '会话',
      formatRelativeTime: () => '刚刚',
      ...overrides
    },
    global: {
      stubs: {
        ElIcon: SlotStub,
        MagicStick: SlotStub,
        Search: SlotStub,
        User: SlotStub
      }
    }
  })
}

describe('MiniWorkstationSessionPanel', () => {
  it('switches from human sessions to a selected automation agent', async () => {
    const wrapper = mountPanel()
    const select = wrapper.find('.mini-session-source-filter select')

    expect(select.findAll('option').map(option => option.text())).toEqual(['人工会话', '每日复盘'])
    await select.setValue('agent:7')

    expect(wrapper.emitted('update:sessionSourceFilter')?.[0]).toEqual(['agent:7'])
  })

  it('marks automation sessions and hides the cross-directory scope', () => {
    const wrapper = mountPanel({
      automationMode: true,
      sessionSourceFilter: 'agent:7',
      sessions: [session({
        source: 'automation_agent',
        automation_task_id: 7,
        automation_task_title: '每日复盘'
      })]
    })

    expect(wrapper.find('.mini-current-session-agent').text()).toContain('每日复盘')
    expect(wrapper.findAll('.mini-drawer-scope-tabs button')).toHaveLength(1)
    expect(wrapper.find('.mini-drawer-scope-tabs').text()).toContain('当前目录')
  })
})
