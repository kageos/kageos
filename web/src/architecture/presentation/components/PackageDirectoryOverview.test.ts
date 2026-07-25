import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PackageDirectoryOverview from './PackageDirectoryOverview.vue'

const getDirectoryOverview = vi.fn()

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ replace: vi.fn() }),
}))

vi.mock('@/architecture/presentation/context/api/service-tree', () => ({
  getDirectoryOverview: (...args: unknown[]) => getDirectoryOverview(...args),
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

    expect(wrapper.findAll('.task-row')).toHaveLength(8)
    expect(wrapper.text()).toContain('Agent 任务 1')
    expect(wrapper.text()).not.toContain('Agent 任务 9')

    await wrapper.get('.overview-pagination.is-agent .pagination-next').trigger('click')

    expect(wrapper.findAll('.task-row')).toHaveLength(1)
    expect(wrapper.text()).toContain('Agent 任务 9')
    expect(wrapper.text()).not.toContain('Agent 任务 1')
  })
})
