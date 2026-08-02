import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { seedPermissionRequestSummaryFromTree } from '@/architecture/presentation/features/access/utils/permissionRequestSummaryStore'
import PermissionPanel from './PermissionPanel.vue'

const permissionApi = vi.hoisted(() => ({
  listPermissionAssignments: vi.fn(),
  listMyPermissionRequests: vi.fn(),
  listPendingPermissionRequests: vi.fn(),
  listPermissionRequestHistory: vi.fn(),
  approvePermissionRequest: vi.fn(),
  rejectPermissionRequest: vi.fn(),
  cancelPermissionRequest: vi.fn(),
}))

vi.mock('@/architecture/presentation/context/api/permission', () => permissionApi)

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

function permissionRequest(id: number, resourcePath: string, status = 'pending') {
  return {
    id,
    tenant_user: 'system',
    app: 'democase',
    requester: 'bob',
    resource_path: resourcePath,
    requested_role: 'member',
    reason: '需要使用',
    status,
    created_at: '2026-08-01T00:00:00Z',
    updated_at: '2026-08-01T00:00:00Z',
  }
}

function permissionNode(admin: boolean) {
  return {
    id: 1,
    type: 'package',
    name: '排行榜',
    full_code_path: '/system/democase/hangla_rank',
    permissions: {
      read: true,
      write: true,
      update: true,
      delete: admin,
      admin,
      owner: false,
    },
    permission_requests: {
      own_pending_count: 1,
      review_pending_count: admin ? 1 : 0,
    },
  } as any
}

function findTab(wrapper: ReturnType<typeof mount>, label: string) {
  const tab = wrapper.findAll('.el-tabs__item').find(item => item.text().includes(label))
  if (!tab) throw new Error(`tab not found: ${label}`)
  return tab
}

describe('PermissionPanel', () => {
  beforeEach(() => {
    Object.values(permissionApi).forEach(mock => mock.mockReset())
    seedPermissionRequestSummaryFromTree('/system/democase', [permissionNode(true)])
    permissionApi.listPermissionAssignments.mockResolvedValue({ assignments: [] })
    permissionApi.listMyPermissionRequests.mockResolvedValue({
      requests: [
        permissionRequest(1, '/system/democase/hangla_rank'),
        permissionRequest(2, '/system/democase/other'),
      ],
    })
    permissionApi.listPendingPermissionRequests.mockResolvedValue({
      requests: [permissionRequest(3, '/system/democase/hangla_rank')],
    })
    permissionApi.listPermissionRequestHistory.mockResolvedValue({
      requests: [permissionRequest(4, '/system/democase/hangla_rank', 'approved')],
    })
  })

  it('flattens permission navigation and loads only the selected request view', async () => {
    const wrapper = mount(PermissionPanel, {
      props: { node: permissionNode(true), embedded: true },
      global: {
        stubs: {
          UsersWidget: true,
          DepartmentDisplay: true,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('权限成员')
    expect(wrapper.text()).toContain('待我审批')
    expect(wrapper.text()).toContain('我的申请')
    expect(wrapper.text()).toContain('审批记录')
    expect(wrapper.text()).not.toContain('权限申请记录')
    expect(permissionApi.listPermissionAssignments).toHaveBeenCalledTimes(1)
    expect(permissionApi.listPendingPermissionRequests).not.toHaveBeenCalled()
    expect(permissionApi.listMyPermissionRequests).not.toHaveBeenCalled()
    expect(permissionApi.listPermissionRequestHistory).not.toHaveBeenCalled()
    expect(findTab(wrapper, '待我审批').text()).toContain('1')
    expect(findTab(wrapper, '我的申请').text()).toContain('1')

    await findTab(wrapper, '待我审批').trigger('click')
    await flushPromises()
    expect(permissionApi.listPendingPermissionRequests).toHaveBeenCalledTimes(1)
    expect(permissionApi.listMyPermissionRequests).not.toHaveBeenCalled()
    expect(permissionApi.listPermissionRequestHistory).not.toHaveBeenCalled()
    expect(wrapper.get('.permission-section-count.is-review').text()).toBe('1')

    await findTab(wrapper, '我的申请').trigger('click')
    await flushPromises()
    expect(permissionApi.listMyPermissionRequests).toHaveBeenCalledTimes(1)
    expect(permissionApi.listPermissionRequestHistory).not.toHaveBeenCalled()
    expect(wrapper.get('.permission-section-count:not(.is-review)').text()).toBe('1')

    await findTab(wrapper, '审批记录').trigger('click')
    await flushPromises()
    expect(permissionApi.listPermissionRequestHistory).toHaveBeenCalledTimes(1)

    await findTab(wrapper, '待我审批').trigger('click')
    await flushPromises()
    expect(permissionApi.listPendingPermissionRequests).toHaveBeenCalledTimes(1)
  })

  it('shows readers only the member list and their own requests', async () => {
    const wrapper = mount(PermissionPanel, {
      props: { node: permissionNode(false), embedded: true },
      global: {
        stubs: {
          UsersWidget: true,
          DepartmentDisplay: true,
        },
      },
    })
    await flushPromises()

    const labels = wrapper.findAll('.el-tabs__item').map(item => item.text())
    expect(findTab(wrapper, '权限成员').exists()).toBe(true)
    expect(findTab(wrapper, '我的申请').text()).toContain('1')
    expect(labels.some(label => label.includes('待我审批'))).toBe(false)
    expect(labels.some(label => label.includes('审批记录'))).toBe(false)
    expect(permissionApi.listPermissionAssignments).toHaveBeenCalledTimes(1)
    expect(permissionApi.listMyPermissionRequests).not.toHaveBeenCalled()
  })
})
