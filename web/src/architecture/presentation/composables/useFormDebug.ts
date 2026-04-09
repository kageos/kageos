import { computed, ref, type ComputedRef } from 'vue'
import { ElMessage } from 'element-plus'
import type { FormDomainService } from '../../domain/services/FormDomainService'
import type { FieldConfig, FieldValue } from '../../domain/types'
import type { FormStateManager } from '../../infrastructure/stateManager/FormStateManager'

interface UseFormDebugOptions {
  stateManager: FormStateManager
  domainService: FormDomainService
  requestFields: ComputedRef<FieldConfig[]>
}

export function useFormDebug(options: UseFormDebugOptions) {
  const showDebugDialog = ref(false)
  const debugActiveTab = ref('request')

  const debugRequestData = computed(() => {
    try {
      const submitData = options.domainService.getSubmitData(options.requestFields.value)
      return JSON.stringify(submitData, null, 2)
    } catch {
      return JSON.stringify({ error: '获取提交数据失败' }, null, 2)
    }
  })

  const debugResponseData = computed(() => {
    const state = options.stateManager.getState()
    if (!state.response) {
      return ''
    }
    try {
      return JSON.stringify(state.response, null, 2)
    } catch {
      return JSON.stringify({ error: '格式化响应数据失败' }, null, 2)
    }
  })

  const debugRawData = computed(() => {
    const state = options.stateManager.getState()
    try {
      const rawData: Record<string, any> = {}
      state.data.forEach((value: FieldValue, key: string) => {
        rawData[key] = {
          raw: value.raw,
          display: value.display,
          dataType: value.dataType || 'unknown',
          widgetType: value.widgetType || 'unknown',
          meta: value.meta
        }
      })
      return JSON.stringify(rawData, null, 2)
    } catch {
      return JSON.stringify({ error: '格式化原始数据失败' }, null, 2)
    }
  })

  const copyToClipboard = async (text: string): Promise<void> => {
    try {
      await navigator.clipboard.writeText(text)
      ElMessage.success('已复制到剪贴板')
    } catch {
      ElMessage.error('复制失败，请手动复制')
    }
  }

  return {
    showDebugDialog,
    debugActiveTab,
    debugRequestData,
    debugResponseData,
    debugRawData,
    copyToClipboard,
  }
}
