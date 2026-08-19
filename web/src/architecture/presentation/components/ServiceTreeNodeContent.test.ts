import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ServiceTreeNodeContent from './ServiceTreeNodeContent.vue'

const getDirectoryOverview = vi.fn()

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/architecture/presentation/context/api/service-tree', () => ({
  getDirectoryOverview: (...args: unknown[]) => getDirectoryOverview(...args),
}))

const PopoverStub = defineComponent({
  emits: ['show'],
  template: `
    <div class="el-popover-stub">
      <slot name="reference" />
      <button type="button" class="popover-show" @click="$emit('show')">show</button>
      <div class="popover-content"><slot /></div>
    </div>
  `,
})

describe('ServiceTreeNodeContent', () => {
  beforeEach(() => {
    getDirectoryOverview.mockReset()
  })

  it('uses only one Agent logo as the directory marker', () => {
    const wrapper = mount(ServiceTreeNodeContent, {
      props: {
        node: {
          id: 1,
          type: 'package',
          name: '工单巡检',
          full_code_path: '/alice/ops',
        } as any,
        showScheduledAgentBadge: true,
        scheduledAgentBadgeTitle: '2 名数字员工已启动',
        scheduledAgentState: 'enabled',
      },
      global: {
        stubs: {
          ElIcon: true,
        },
      },
    })

    const marker = wrapper.get('.scheduled-agent-badge')
    const node = wrapper.get('.tree-node')
    expect(marker.text()).toBe('')
    expect(marker.attributes('aria-label')).toBe('2 名数字员工已启动')
    expect(marker.classes()).toContain('is-enabled')
    expect(node.classes()).not.toContain('has-scheduled-agent')
    expect(node.classes()).toContain('agent-state-enabled')
    expect(marker.get('[data-agent-variant="mark"]').attributes('data-agent-state')).toBe('ready')
    expect(wrapper.text()).not.toContain('员工在值守')
  })

  it('marks the directory employee icon as failed when an employee needs attention', () => {
    const wrapper = mount(ServiceTreeNodeContent, {
      props: {
        node: {
          id: 2,
          type: 'package',
          name: '日报',
          full_code_path: '/alice/reports',
        } as any,
        showScheduledAgentBadge: true,
        scheduledAgentBadgeTitle: '1 名数字员工需要关注',
        scheduledAgentState: 'failed',
      },
    })

    const marker = wrapper.get('[data-agent-variant="mark"]')
    expect(wrapper.get('.tree-node').classes()).toContain('agent-state-failed')
    expect(marker.attributes('data-agent-state')).toBe('failed')
    const image = marker.get('img')
    expect(image.attributes('src')).toContain('service-icon.webp')
  })

  it('shows a clickable lock and switches to the pending marker after a request is submitted', async () => {
    const wrapper = mount(ServiceTreeNodeContent, {
      props: {
        node: {
          id: 20,
          type: 'package',
          name: '受限服务目录',
          full_code_path: '/alice/ops/locked',
        } as any,
        showAccessLock: true,
        accessLockTitle: '暂无读取权限，点击申请',
      },
    })

    const lock = wrapper.get('[data-testid="service-tree-access-lock"]')
    expect(lock.attributes('aria-label')).toBe('暂无读取权限，点击申请')
    expect(lock.classes()).not.toContain('is-pending')
    await lock.trigger('click')
    expect(wrapper.emitted('access-request-click')).toHaveLength(1)

    await wrapper.setProps({
      accessRequestPending: true,
      accessLockTitle: '权限申请待审批',
    })
    expect(lock.classes()).toContain('is-pending')
    expect(lock.attributes('aria-label')).toBe('权限申请待审批')
  })

  it('shows a clickable permission request count beside the resource', async () => {
    const wrapper = mount(ServiceTreeNodeContent, {
      props: {
        node: {
          id: 21,
          type: 'package',
          name: '审批服务目录',
          full_code_path: '/alice/ops/review',
        } as any,
        showPermissionRequestBadge: true,
        permissionRequestBadgeValue: 3,
        permissionRequestBadgeClass: 'needs-review',
        permissionRequestBadgeTitle: '3 个权限申请待处理',
      },
    })

    const badge = wrapper.get('[data-testid="service-tree-permission-request-badge"]')
    expect(badge.classes()).toContain('needs-review')
    expect(badge.attributes('title')).toBe('3 个权限申请待处理')
    await badge.trigger('click')
    expect(wrapper.emitted('permission-request-click')).toHaveLength(1)
  })

  it('shows compact platform logos for configured notification routes', async () => {
    const wrapper = mount(ServiceTreeNodeContent, {
      props: {
        node: {
          id: 22,
          type: 'package',
          name: '站点巡检',
          full_code_path: '/alice/ops/site-monitor',
        } as any,
        showNotificationRouteBadge: true,
        notificationRouteBadgeTitle: '通知发送到飞书和企业微信',
        notificationRouteChannels: ['feishu', 'wecom', 'feishu'],
      },
      global: {
        stubs: {
          ElIcon: true,
        },
      },
    })

    const badge = wrapper.get('.notification-route-badge')
    const logos = badge.findAll('.notification-route-logo')
    expect(badge.classes()).toContain('has-channel-logos')
    expect(badge.attributes('title')).toBe('通知发送到飞书和企业微信')
    expect(badge.attributes('aria-label')).toBe('通知发送到飞书和企业微信')
    expect(logos).toHaveLength(2)
    expect(logos[0]!.attributes('src')).toContain('data:image/jpeg;base64,')

    await badge.trigger('click')
    expect(wrapper.emitted('notification-route-click')).toHaveLength(1)
  })

  it('reduces inherited route noise on resource nodes while keeping directory routes visible', () => {
    const resource = mount(ServiceTreeNodeContent, {
      props: {
        node: {
          id: 23,
          type: 'function',
          name: '巡检日志',
          full_code_path: '/alice/ops/site-monitor/logs.table',
        } as any,
        showNotificationRouteBadge: true,
        notificationRouteBadgeTone: 'inherited',
        notificationRouteBadgeTitle: '继承飞书通知',
        notificationRouteChannels: ['feishu'],
      },
      global: { stubs: { ElIcon: true } },
    })
    expect(resource.get('.notification-route-badge').classes()).toContain('is-idle-hidden')

    const directory = mount(ServiceTreeNodeContent, {
      props: {
        node: {
          id: 24,
          type: 'package',
          name: '站点巡检',
          full_code_path: '/alice/ops/site-monitor',
        } as any,
        showNotificationRouteBadge: true,
        notificationRouteBadgeTone: 'inherited',
        notificationRouteBadgeTitle: '继承飞书通知',
        notificationRouteChannels: ['feishu'],
      },
      global: { stubs: { ElIcon: true } },
    })
    expect(directory.get('.notification-route-badge').classes()).not.toContain('is-idle-hidden')
  })

  it.each([
    ['running', 'working'],
    ['enabled', 'ready'],
    ['paused', 'paused'],
    ['failed', 'failed'],
  ] as const)('maps the %s directory color state to the %s employee pose', (directoryState, employeeState) => {
    const wrapper = mount(ServiceTreeNodeContent, {
      props: {
        node: {
          id: 3,
          type: 'package',
          name: '客户回访',
          full_code_path: '/alice/customers',
        } as any,
        showScheduledAgentBadge: true,
        scheduledAgentBadgeTitle: '数字员工状态',
        scheduledAgentState: directoryState,
      },
    })

    expect(wrapper.get('.tree-node').classes()).toContain(`agent-state-${directoryState}`)
    expect(wrapper.get('[data-agent-variant="mark"]').attributes('data-agent-state')).toBe(employeeState)
  })

  it('shows actionable employee details when hovering the service-tree marker', async () => {
    getDirectoryOverview.mockResolvedValue({
      stats: {
        directories: 0,
        functions: 0,
        docs: 0,
        total_run_count: 0,
        scheduled_function_tasks: 0,
        scheduled_agent_tasks: 5,
        running_tasks: 1,
        failed_tasks: 1,
        paused_tasks: 1,
      },
      scheduled_function_tasks: [],
      scheduled_agent_tasks: [
        {
          kind: 'agent',
          resource_path: '/alice/ops/tickets',
          resource_name: '工单巡检',
          task: {
            id: 1,
            title: '紧急工单处理',
            status: 'pending',
            executor_key: 'agent.session',
            executor_payload: { message: '检查新工单，符合规则时自动处理' },
            schedule: { type: 'every', interval_seconds: 300 },
            run_count: 8,
            inflight_execution_id: 11,
            next_run_at: '2026-07-28T01:00:00Z',
          },
        },
        {
          kind: 'agent',
          resource_path: '/alice/ops/tickets',
          resource_name: '工单巡检',
          task: {
            id: 2,
            title: '超时工单提醒',
            description: '发现超时工单后联系负责人',
            status: 'pending',
            executor_key: 'agent.session',
            schedule: { type: 'cron', cron_expr: '0 9 * * *' },
            run_count: 3,
            last_error_message: '工单系统连接失败',
          },
        },
        ...Array.from({ length: 3 }, (_, index) => ({
          kind: 'agent' as const,
          resource_path: '/alice/ops/tickets',
          resource_name: '工单巡检',
          task: {
            id: index + 3,
            title: `待命员工 ${index + 1}`,
            status: 'pending' as const,
            executor_key: 'agent.session',
            schedule: { type: 'every' as const, interval_seconds: 600 },
            run_count: 0,
          },
        })),
      ],
      warnings: [],
    })

    const wrapper = mount(ServiceTreeNodeContent, {
      props: {
        node: {
          id: 4,
          type: 'package',
          name: '客户工单',
          full_code_path: '/alice/ops/tickets',
          scheduled_agent_tasks: 5,
          enabled_agent_tasks: 4,
          running_agent_tasks: 1,
          failed_agent_tasks: 1,
          admins: 'alice,bob',
        } as any,
        showScheduledAgentBadge: true,
        scheduledAgentBadgeTitle: '1 名数字员工正在处理，1 名需要关注',
        scheduledAgentState: 'failed',
      },
      global: {
        stubs: {
          ElPopover: PopoverStub,
        },
      },
    })

    await wrapper.get('.popover-show').trigger('click')
    await flushPromises()

    expect(getDirectoryOverview).toHaveBeenCalledWith('/alice/ops/tickets')
    const card = wrapper.get('[data-testid="scheduled-agent-hover-card"]')
    expect(card.text()).toContain('数字员工 · 客户工单')
    expect(card.text()).toContain('负责人alice、bob')
    expect(card.text()).toContain('紧急工单处理')
    expect(card.text()).toContain('检查新工单，符合规则时自动处理')
    expect(card.text()).toContain('超时工单提醒')
    expect(card.text()).toContain('工单系统连接失败')
    expect(card.text()).toContain('还有 1 名员工')
    expect(card.findAll('.scheduled-agent-hover-task')).toHaveLength(4)
    expect(card.findAll('.scheduled-agent-hover-task')[0]!.classes()).toContain('is-working')
    expect(card.findAll('.scheduled-agent-hover-task')[1]!.classes()).toContain('is-failed')

    await card.get('.scheduled-agent-hover-open').trigger('click')
    expect(wrapper.emitted('scheduled-agent-click')).toHaveLength(1)
  })
})
