import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PackageDirectoryOverview from './PackageDirectoryOverview.vue'

const getDirectoryOverview = vi.fn()
const resumeTimerTask = vi.fn()

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ replace: vi.fn() }),
}))

vi.mock('@/architecture/presentation/context/api/service-tree', () => ({
  getDirectoryOverview: (...args: unknown[]) => getDirectoryOverview(...args),
}))

vi.mock('@/architecture/presentation/context/api/timer', () => ({
  resumeTimerTask: (...args: unknown[]) => resumeTimerTask(...args),
}))

const PaginationStub = defineComponent({
  props: {
    currentPage: { type: Number, default: 1 },
    total: { type: Number, default: 0 },
  },
  emits: ['update:currentPage'],
  template: `
    <button
      type="button"
      class="pagination-next"
      @click="$emit('update:currentPage', currentPage + 1)"
    >
      {{ currentPage }}/{{ total }}
    </button>
  `,
})

const EmptyStub = defineComponent({
  template: '<div class="el-empty" />',
})

function agentTask(index: number) {
  return {
    kind: 'agent',
    resource_path: `/alice/ops/task-${index}`,
    resource_name: `目录 ${index}`,
    task: {
      id: index,
      title: `Agent 任务 ${index}`,
      status: 'pending',
      schedule: { type: 'every', interval_seconds: 60 },
      resource_key: `/alice/ops/task-${index}`,
      inflight_execution_id: '',
      last_error_message: '',
    },
  }
}

function mountOverview() {
  return mount(PackageDirectoryOverview, {
    props: {
      packageNode: {
        type: 'package',
        full_code_path: '/alice/ops',
        children: [],
      } as any,
    },
    global: {
      directives: {
        loading: {},
      },
      stubs: {
        ElAlert: true,
        ElButton: true,
        ElEmpty: EmptyStub,
        ElIcon: true,
        ElPagination: PaginationStub,
        ElTag: true,
      },
    },
  })
}

describe('PackageDirectoryOverview', () => {
  beforeEach(() => {
    getDirectoryOverview.mockReset()
    resumeTimerTask.mockReset()
  })

  it('paginates scheduled Agent tasks after the first eight items', async () => {
    getDirectoryOverview.mockResolvedValue({
      stats: {
        directories: 9,
        functions: 0,
        docs: 0,
        total_run_count: 0,
        scheduled_function_tasks: 0,
        scheduled_agent_tasks: 9,
        running_tasks: 0,
        failed_tasks: 0,
        paused_tasks: 0,
      },
      scheduled_function_tasks: [],
      scheduled_agent_tasks: Array.from({ length: 9 }, (_, index) => agentTask(index + 1)),
      warnings: [],
    })

    const wrapper = mountOverview()
    await flushPromises()

    const agentPanel = wrapper.findAll('.scheduled-panel')[1]!
    expect(agentPanel.findAll('.task-row')).toHaveLength(8)
    expect(agentPanel.text()).toContain('Agent 任务 1')
    expect(agentPanel.text()).not.toContain('Agent 任务 9')

    await wrapper.get('.overview-pagination.is-agent .pagination-next').trigger('click')

    expect(agentPanel.findAll('.task-row')).toHaveLength(1)
    expect(agentPanel.text()).toContain('Agent 任务 9')
    expect(agentPanel.text()).not.toContain('Agent 任务 1')
  })

  it('floats a compact roster and can expand all Agent employees on directory details', async () => {
    const tasks = Array.from({ length: 16 }, (_, index) => agentTask(index + 1))
    tasks[0]!.task.inflight_execution_id = 'execution-1'
    tasks[1]!.task.last_error_message = '连接工单系统失败'

    getDirectoryOverview.mockResolvedValue({
      stats: {
        directories: 0,
        functions: 0,
        docs: 0,
        total_run_count: 0,
        scheduled_function_tasks: 0,
        scheduled_agent_tasks: 16,
        running_tasks: 1,
        failed_tasks: 0,
        paused_tasks: 0,
      },
      scheduled_function_tasks: [],
      scheduled_agent_tasks: tasks,
      warnings: [],
    })

    const wrapper = mountOverview()
    await flushPromises()

    const roster = wrapper.get('[data-testid="agent-presence-float"]')
    expect(roster.findAll('.agent-presence-employee')).toHaveLength(3)
    expect(roster.get('.agent-presence-employee').classes()).toContain('is-working')
    expect(roster.findAll('[data-agent-variant="employee"]')[0]!.attributes('data-agent-state')).toBe('working')
    expect(roster.findAll('[data-agent-variant="employee"]')[1]!.attributes('data-agent-state')).toBe('failed')
    expect(roster.text()).toContain('Agent 任务 1')
    expect(roster.text()).toContain('正在处理')
    expect(roster.text()).toContain('已启动 16/16')
    expect(roster.text()).toContain('1 名需关注')
    expect(roster.text()).toContain('展开其余 13 名员工')
    expect(roster.text()).not.toContain('Agent 任务 16')

    const agentPanel = wrapper.findAll('.scheduled-panel')[1]!
    expect(agentPanel.text()).toContain('数字员工')
    expect(agentPanel.text()).not.toContain('Agent 值守员工')
    const sectionLogo = agentPanel.get('.scheduled-panel-agent-mark')
    expect(sectionLogo.attributes('data-agent-variant')).toBe('mark')
    expect(sectionLogo.attributes('data-agent-state')).toBe('working')
    const listEmployeeMarks = agentPanel.findAll('.task-row-agent-avatar [data-agent-variant="mark"]')
    expect(listEmployeeMarks).toHaveLength(8)
    expect(listEmployeeMarks[0]!.attributes('data-agent-state')).toBe('working')
    expect(listEmployeeMarks[1]!.attributes('data-agent-state')).toBe('failed')
    expect(listEmployeeMarks[2]!.attributes('data-agent-state')).toBe('ready')

    const expandButton = roster.get('button[aria-controls="agent-presence-team"]')
    expect(expandButton.attributes('aria-expanded')).toBe('false')
    await expandButton.trigger('click')

    expect(roster.classes()).toContain('is-expanded')
    expect(roster.findAll('.agent-presence-employee')).toHaveLength(16)
    expect(roster.text()).toContain('Agent 任务 16')
    expect(roster.text()).toContain('收起员工')

    await expandButton.trigger('click')

    expect(roster.classes()).not.toContain('is-expanded')
    expect(roster.findAll('.agent-presence-employee')).toHaveLength(3)
  })

  it('clearly marks a paused employee as not started and offers a direct start action', async () => {
    const pausedTask = agentTask(1)
    pausedTask.task.status = 'paused'
    const startedTask = agentTask(1)

    getDirectoryOverview
      .mockResolvedValueOnce({
        stats: {
          directories: 0,
          functions: 0,
          docs: 0,
          total_run_count: 0,
          scheduled_function_tasks: 0,
          scheduled_agent_tasks: 1,
          running_tasks: 0,
          failed_tasks: 0,
          paused_tasks: 1,
        },
        scheduled_function_tasks: [],
        scheduled_agent_tasks: [pausedTask],
        warnings: [],
      })
      .mockResolvedValueOnce({
        stats: {
          directories: 0,
          functions: 0,
          docs: 0,
          total_run_count: 0,
          scheduled_function_tasks: 0,
          scheduled_agent_tasks: 1,
          running_tasks: 0,
          failed_tasks: 0,
          paused_tasks: 0,
        },
        scheduled_function_tasks: [],
        scheduled_agent_tasks: [startedTask],
        warnings: [],
      })
    resumeTimerTask.mockResolvedValue(undefined)

    const wrapper = mountOverview()
    await flushPromises()

    const roster = wrapper.get('[data-testid="agent-presence-float"]')
    expect(roster.text()).toContain('已启动 0/1')
    expect(roster.text()).toContain('未启动')
    expect(
      wrapper
        .findAll('.scheduled-panel')[1]!
        .get('.task-row-agent-avatar [data-agent-variant="mark"]')
        .attributes('data-agent-state')
    ).toBe('paused')

    const startButton = roster.get('button[aria-label="启动 Agent 任务 1"]')
    await startButton.trigger('click')
    await flushPromises()

    expect(resumeTimerTask).toHaveBeenCalledWith(1)
    expect(getDirectoryOverview).toHaveBeenCalledTimes(2)
    expect(roster.text()).toContain('已启动 1/1')
    expect(roster.find('button[aria-label="启动 Agent 任务 1"]').exists()).toBe(false)
  })
})
