import { computed, effectScope, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useWorkspaceFunctionTabs } from './useWorkspaceFunctionTabs'

describe('useWorkspaceFunctionTabs', () => {
  it('opens the detail tab from the panel query', () => {
    const scope = effectScope()

    try {
      const route = {
        path: '/workspace/demo/app/function',
        query: {
          _panel: 'detail'
        }
      } as any
      const router = {
        replace: vi.fn()
      } as any
      const currentFunction = computed(() => ({
        type: 'function',
        full_code_path: '/demo/app/function'
      }) as any)
      const currentFunctionDetail = ref({
        template_type: 'table'
      } as any)

      const tabs = scope.run(() => useWorkspaceFunctionTabs({
        route,
        router,
        currentFunction,
        currentFunctionDetail
      }))!

      expect(tabs.functionActiveTab.value).toBe('detail')
    } finally {
      scope.stop()
    }
  })

  it('normalizes retired panel deep links back to content', () => {
    const scope = effectScope()

    try {
      const route = {
        path: '/workspace/demo/app/function',
        query: {
          _panel: 'permission'
        }
      } as any
      const router = {
        replace: vi.fn()
      } as any
      const currentFunction = computed(() => ({
        type: 'function',
        full_code_path: '/demo/app/function'
      }) as any)
      const currentFunctionDetail = ref({
        template_type: 'table'
      } as any)

      const tabs = scope.run(() => useWorkspaceFunctionTabs({
        route,
        router,
        currentFunction,
        currentFunctionDetail
      }))!

      expect(tabs.functionActiveTab.value).toBe('content')
    } finally {
      scope.stop()
    }
  })

  it('normalizes unsupported tab changes back to content', () => {
    const scope = effectScope()

    try {
      const route = {
        path: '/workspace/demo/app/function',
        query: {}
      } as any
      const router = {
        replace: vi.fn()
      } as any
      const currentFunction = computed(() => ({
        type: 'function',
        full_code_path: '/demo/app/function'
      }) as any)
      const currentFunctionDetail = ref({
        template_type: 'table'
      } as any)

      const tabs = scope.run(() => useWorkspaceFunctionTabs({
        route,
        router,
        currentFunction,
        currentFunctionDetail
      }))!

      tabs.handleFunctionTabChange('permissionManage')

      expect(tabs.functionActiveTab.value).toBe('content')
      expect(router.replace).not.toHaveBeenCalled()
    } finally {
      scope.stop()
    }
  })
})
