import { computed, effectScope } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { usePackageDetailTabs } from './usePackageDetailTabs'

describe('usePackageDetailTabs', () => {
  const currentPackageNode = computed(() => ({
    type: 'package',
    full_code_path: '/demo/app/directory'
  }) as any)

  it('opens a package tab from the panel query', () => {
    const scope = effectScope()

    try {
      const route = {
        path: '/workspace/demo/app/directory',
        query: {
          _panel: 'detail'
        }
      } as any
      const router = {
        replace: vi.fn()
      } as any

      const tabs = scope.run(() => usePackageDetailTabs({
        route,
        router,
        currentPackageNode
      }))!

      expect(tabs.activeTab.value).toBe('detail')
      expect(router.replace).not.toHaveBeenCalled()
    } finally {
      scope.stop()
    }
  })

  it('syncs package tab changes to the URL panel query', () => {
    const scope = effectScope()

    try {
      const route = {
        path: '/workspace/demo/app/directory',
        query: {}
      } as any
      const router = {
        replace: vi.fn()
      } as any

      const tabs = scope.run(() => usePackageDetailTabs({
        route,
        router,
        currentPackageNode
      }))!

      tabs.handlePackageTabChange('permission')

      expect(tabs.activeTab.value).toBe('permission')
      expect(router.replace).toHaveBeenCalledWith({
        path: '/workspace/demo/app/directory',
        query: {
          _panel: 'permission'
        }
      })
    } finally {
      scope.stop()
    }
  })

  it('opens the notification tab from the panel query', () => {
    const scope = effectScope()

    try {
      const route = {
        path: '/workspace/demo/app/directory',
        query: {
          _panel: 'notification'
        }
      } as any
      const router = {
        replace: vi.fn()
      } as any

      const tabs = scope.run(() => usePackageDetailTabs({
        route,
        router,
        currentPackageNode
      }))!

      expect(tabs.activeTab.value).toBe('notification')
    } finally {
      scope.stop()
    }
  })

  it('restores scheduled agent tabs from legacy scheduled route state', () => {
    const scope = effectScope()

    try {
      const route = {
        path: '/workspace/demo/app/directory',
        query: {
          _scheduled: 'open',
          _scheduled_kind: 'agent',
          _scheduled_task_id: '7'
        }
      } as any
      const router = {
        replace: vi.fn()
      } as any

      const tabs = scope.run(() => usePackageDetailTabs({
        route,
        router,
        currentPackageNode
      }))!

      expect(tabs.activeTab.value).toBe('scheduledAgentTask')
    } finally {
      scope.stop()
    }
  })
})
