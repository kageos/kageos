import { ref } from 'vue'
import { getLLMList, type LLMInfo } from '@/architecture/presentation/context/api/agent'

function llmProtocolLabel(protocol: string): string {
  switch ((protocol || '').trim()) {
    case 'openai_responses':
      return 'Responses'
    case 'anthropic_messages':
      return 'Messages'
    default:
      return 'Chat'
  }
}

function llmEndpointLabel(llm: LLMInfo): string {
  const endpoint = (llm.endpoint_path || '').trim()
  const base = (llm.api_base || '').trim().replace(/\/+$/, '')
  if (!base) {
    return endpoint
  }
  try {
    const url = new URL(base)
    const basePath = url.pathname.replace(/\/+$/, '')
    const endpointPath = endpoint ? (endpoint.startsWith('/') ? endpoint : `/${endpoint}`) : ''
    return `${url.host}${basePath}${endpointPath}`
  } catch {
    if (endpoint) {
      return `${base}${endpoint.startsWith('/') ? endpoint : `/${endpoint}`}`.replace(/^https?:\/\//, '')
    }
    return base.replace(/^https?:\/\//, '')
  }
}

export function llmOptionLabel(llm: LLMInfo): string {
  const endpoint = llmEndpointLabel(llm)
  const suffix = endpoint ? ` · ${endpoint}` : ''
  return `#${llm.id} ${llm.name} (${llm.model} · ${llmProtocolLabel(llm.protocol)}${suffix})`
}

export function useLLMConfigOptions() {
  const llmList = ref<LLMInfo[]>([])
  const llmLoading = ref(false)
  const llmLoaded = ref(false)

  async function loadLLMOptions(force = false): Promise<void> {
    if (llmLoading.value || (llmLoaded.value && !force)) {
      return
    }
    llmLoading.value = true
    try {
      const results = await Promise.allSettled([
        getLLMList({ scope: 'market', page: 1, page_size: 200 }) as Promise<{ configs?: LLMInfo[] }>,
        getLLMList({ scope: 'mine', page: 1, page_size: 200 }) as Promise<{ configs?: LLMInfo[] }>,
      ])
      const merged = new Map<number, LLMInfo>()
      for (const result of results) {
        if (result.status !== 'fulfilled') {
          continue
        }
        for (const llm of result.value?.configs ?? []) {
          merged.set(llm.id, llm)
        }
      }
      llmList.value = Array.from(merged.values()).sort((a, b) => b.id - a.id)
      llmLoaded.value = results.some((result) => result.status === 'fulfilled')
    } catch {
      llmList.value = []
    } finally {
      llmLoading.value = false
    }
  }

  function handleLLMSelectVisibleChange(visible: boolean): void {
    if (visible) {
      void loadLLMOptions()
    }
  }

  function llmConfigLabel(id: number, defaultLabel: string): string {
    if (!id) {
      return defaultLabel
    }
    const matched = llmList.value.find((llm) => llm.id === id)
    return matched ? llmOptionLabel(matched) : `#${id}`
  }

  return {
    llmList,
    llmLoading,
    loadLLMOptions,
    handleLLMSelectVisibleChange,
    llmConfigLabel,
    llmOptionLabel,
  }
}
