import { computed, effectScope, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { usePackageDetailTabs } from './usePackageDetailTabs'
import { useWorkspaceFunctionTabs } from './useWorkspaceFunctionTabs'

describe('archive resource navigation', () => {
  it('opens and persists archive tabs within directory and function URLs', () => {
    const scope = effectScope()
    try {
      const packageRoute = { path: '/workspace/alice/ops/team', query: { _panel: 'logArchives' } } as any
      const functionRoute = { path: '/workspace/alice/ops/team/a.form', query: { _panel: 'logArchives' } } as any
      const packageRouter = { replace: vi.fn() } as any
      const functionRouter = { replace: vi.fn() } as any
      const directory = scope.run(() => usePackageDetailTabs({route: packageRoute, router: packageRouter, currentPackageNode: computed(() => ({type: 'package', full_code_path: '/alice/ops/team'}) as any)}))!
      const fn = scope.run(() => useWorkspaceFunctionTabs({route: functionRoute, router: functionRouter, currentFunction: computed(() => ({type: 'function', full_code_path: '/alice/ops/team/a.form'}) as any), currentFunctionDetail: ref({template_type: 'form'} as any)}))!
      expect(directory.activeTab.value).toBe('logArchives')
      expect(fn.functionActiveTab.value).toBe('logArchives')
      packageRoute.query = {}; functionRoute.query = {}
      directory.handlePackageTabChange('logArchives')
      fn.handleFunctionTabChange('logArchives')
      expect(packageRouter.replace).toHaveBeenLastCalledWith(expect.objectContaining({path: packageRoute.path, query: expect.objectContaining({_panel: 'logArchives'})}))
      expect(functionRouter.replace).toHaveBeenLastCalledWith(expect.objectContaining({path: functionRoute.path, query: expect.objectContaining({_panel: 'logArchives'})}))
    } finally { scope.stop() }
  })
})
