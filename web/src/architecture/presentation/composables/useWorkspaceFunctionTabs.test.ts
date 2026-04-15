import { computed, effectScope, nextTick, ref } from 'vue'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { useWorkspaceFunctionTabs } from './useWorkspaceFunctionTabs'

const { successMock, warningMock, errorMock } = vi.hoisted(() => ({
  successMock: vi.fn(),
  warningMock: vi.fn(),
  errorMock: vi.fn()
}))

vi.mock('element-plus', () => ({
  ElMessage: {
    success: successMock,
    warning: warningMock,
    error: errorMock
  }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: {
      username: 'liubeiluo'
    }
  })
}))

vi.mock('@/utils/permissionActors', () => ({
  isServiceTreeNodeAdmin: () => false
}))

describe('useWorkspaceFunctionTabs', () => {
  beforeEach(() => {
    successMock.mockReset()
    warningMock.mockReset()
    errorMock.mockReset()
  })

  it('waits for the form view ref before applying an operate log', async () => {
    const scope = effectScope()

    try {
      const route = {
        path: '/workspace',
        query: {}
      } as any
      const router = {
        replace: vi.fn()
      } as any
      const currentFunction = computed(() => ({
        type: 'function',
        full_code_path: '/demo.form'
      }) as any)
      const currentFunctionDetail = ref({
        template_type: 'form'
      } as any)

      const tabs = scope.run(() => useWorkspaceFunctionTabs({
        route,
        router,
        currentFunction,
        currentFunctionDetail
      }))!

      const applyOperateLog = vi.fn(async () => undefined)
      const applyPromise = tabs.handleApplyFormOperateLog({
        requestBody: {
          text_input: '测试',
          progress: 50
        }
      })

      await nextTick()
      tabs.functionFormViewRef.value = {
        applyOperateLog
      }

      await applyPromise

      expect(applyOperateLog).toHaveBeenCalledTimes(1)
      expect(applyOperateLog).toHaveBeenCalledWith({
        requestBody: {
          text_input: '测试',
          progress: 50
        },
        responseBody: undefined,
        responseMetadata: undefined
      })
      expect(successMock).toHaveBeenCalledWith('已将执行记录回填到表单')
      expect(warningMock).not.toHaveBeenCalled()
      expect(errorMock).not.toHaveBeenCalled()
    } finally {
      scope.stop()
    }
  })

  it('reapplies the operate log when the form view ref is replaced mid-flight', async () => {
    const scope = effectScope()

    try {
      const route = {
        path: '/workspace',
        query: {}
      } as any
      const router = {
        replace: vi.fn()
      } as any
      const currentFunction = computed(() => ({
        type: 'function',
        full_code_path: '/demo.form'
      }) as any)
      const currentFunctionDetail = ref({
        template_type: 'form'
      } as any)

      const tabs = scope.run(() => useWorkspaceFunctionTabs({
        route,
        router,
        currentFunction,
        currentFunctionDetail
      }))!

      const secondRef = {
        applyOperateLog: vi.fn(async () => undefined)
      }
      const firstRef = {
        applyOperateLog: vi.fn(async () => {
          tabs.functionFormViewRef.value = secondRef
        })
      }

      tabs.functionFormViewRef.value = firstRef

      await tabs.handleApplyFormOperateLog({
        requestBody: {
          text_input: '重放'
        }
      })

      expect(firstRef.applyOperateLog).toHaveBeenCalledTimes(1)
      expect(secondRef.applyOperateLog).toHaveBeenCalledTimes(1)
      expect(successMock).toHaveBeenCalledWith('已将执行记录回填到表单')
      expect(warningMock).not.toHaveBeenCalled()
      expect(errorMock).not.toHaveBeenCalled()
    } finally {
      scope.stop()
    }
  })
})
