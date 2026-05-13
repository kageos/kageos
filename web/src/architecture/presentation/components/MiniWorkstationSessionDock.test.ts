import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it } from 'vitest'
import type { WorkspaceSessionItem } from '@/api/workspace'
import MiniWorkstationSessionDock from './MiniWorkstationSessionDock.vue'

const SlotStub = defineComponent({
  template: '<span><slot /></span>',
})

function createSession(overrides: Partial<WorkspaceSessionItem> = {}): WorkspaceSessionItem {
  return {
    session_id: 'session-1',
    title: '减法链路',
    status: 'active',
    agent_name: '产品经理',
    full_code_path: '/Users/demo/subtract',
    directory_name: 'subtract',
    created_at: '2026-05-13T08:00:00Z',
    updated_at: '2026-05-13T09:00:00Z',
    ...overrides,
  }
}

function mountDock(props: Record<string, unknown> = {}) {
  return mount(MiniWorkstationSessionDock, {
    props: {
      summarySessions: [
        createSession(),
        createSession({ session_id: 'session-2', title: '测试会话', status: 'generating' }),
      ],
      centerCount: 4,
      directoryLabel: 'subtract',
      sessionId: 'session-1',
      getSessionStatusClass: (session: WorkspaceSessionItem) => (
        session.status === 'generating' ? 'is-running' : 'is-active'
      ),
      getSessionStatusKind: (session: WorkspaceSessionItem) => (
        session.status === 'generating' ? 'running' : 'active'
      ),
      getSessionTitle: (session: WorkspaceSessionItem) => session.title,
      getSessionSubtitle: (session: WorkspaceSessionItem) => `${session.directory_name} · ${session.agent_name}`,
      ...props,
    },
    global: {
      stubs: {
        ElIcon: SlotStub,
        Plus: SlotStub,
      },
    },
  })
}

describe('MiniWorkstationSessionDock', () => {
  it('emits commands for center and new session actions', async () => {
    const wrapper = mountDock()

    await wrapper.find('.mini-session-center-btn').trigger('click')
    await wrapper.find('.mini-session-new-btn').trigger('click')

    expect(wrapper.emitted('openCenter')).toHaveLength(1)
    expect(wrapper.emitted('newSession')).toHaveLength(1)
  })

  it('emits selected summary session', async () => {
    const wrapper = mountDock()

    await wrapper.findAll('.mini-session-summary-card')[1]!.trigger('click')

    expect(wrapper.emitted('select')?.[0]?.[0]).toMatchObject({ session_id: 'session-2' })
  })

  it('shows draft action when there are no summary sessions', async () => {
    const wrapper = mountDock({ summarySessions: [] })

    expect(wrapper.find('.mini-session-summary-title').text()).toBe('新建会话')
    await wrapper.find('.mini-session-summary-card').trigger('click')

    expect(wrapper.emitted('newSession')).toHaveLength(1)
  })
})
