import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it } from 'vitest'
import type { WorkspaceSessionItem } from '@/architecture/presentation/context/api/workspace'
import MiniWorkstationSessionCenter from './MiniWorkstationSessionCenter.vue'

const SlotStub = defineComponent({
  template: '<span><slot /></span>',
})

function createSession(overrides: Partial<WorkspaceSessionItem> = {}): WorkspaceSessionItem {
  return {
    session_id: 'session-1',
    title: '客户减法链路',
    status: 'active',
    full_code_path: '/Users/demo/customer',
    directory_name: 'customer',
    created_at: '2026-05-13T08:00:00Z',
    updated_at: '2026-05-13T09:00:00Z',
    ...overrides,
  }
}

function mountCenter(props: Record<string, unknown> = {}) {
  const currentSession = createSession()
  const recentSession = createSession({
    session_id: 'session-2',
    title: '跨目录会话',
    full_code_path: '/Users/demo/other',
    directory_name: 'other',
    status: 'done',
  })

  return mount(MiniWorkstationSessionCenter, {
    props: {
      open: true,
      currentDirectorySessions: [currentSession],
      recentSessions: [recentSession],
      currentDirectoryTotal: 2,
      recentSourceTotal: 3,
      loadingCurrent: false,
      loadingRecent: false,
      fullCodePath: '/Users/demo/customer',
      directoryLabel: 'customer',
      sessionId: 'session-1',
      sessionFilters: [
        { label: '全部', value: 'all' },
        { label: '执行中', value: 'running' },
      ],
      sessionSearchKeyword: '',
      sessionFilter: 'all',
      formatRelativeTime: () => '刚刚',
      getSessionStatusClass: (session: WorkspaceSessionItem) => (
        session.status === 'done' ? 'is-done' : 'is-running'
      ),
      getSessionTitle: (session: WorkspaceSessionItem) => session.title,
      getSessionCenterSubtitle: (session: WorkspaceSessionItem) => `${session.directory_name} · ${session.role_display_name || session.status}`,
      getSessionStatusLabel: (session: WorkspaceSessionItem) => (session.status === 'done' ? '完成' : '执行中'),
      ...props,
    },
    global: {
      directives: {
        loading: {},
      },
      stubs: {
        ElIcon: SlotStub,
        Close: SlotStub,
        Search: SlotStub,
      },
    },
  })
}

describe('MiniWorkstationSessionCenter', () => {
  it('emits search and filter updates', async () => {
    const wrapper = mountCenter()

    await wrapper.find('.mini-session-search input').setValue('客户')
    await wrapper.findAll('.mini-session-filters button')[1]!.trigger('click')

    expect(wrapper.emitted('update:sessionSearchKeyword')?.[0]).toEqual(['客户'])
    expect(wrapper.emitted('update:sessionFilter')?.[0]).toEqual(['running'])
  })

  it('emits the selected session from either column', async () => {
    const wrapper = mountCenter()
    const rows = wrapper.findAll('.mini-session-row')

    await rows[0]!.trigger('click')
    await rows[1]!.trigger('click')

    expect(wrapper.emitted('select')?.[0]?.[0]).toMatchObject({ session_id: 'session-1' })
    expect(wrapper.emitted('select')?.[1]?.[0]).toMatchObject({ session_id: 'session-2' })
  })

  it('emits close from backdrop and close button', async () => {
    const wrapper = mountCenter()

    await wrapper.find('.mini-session-center').trigger('click')
    await wrapper.find('.mini-session-close').trigger('click')

    expect(wrapper.emitted('close')).toHaveLength(2)
  })
})
