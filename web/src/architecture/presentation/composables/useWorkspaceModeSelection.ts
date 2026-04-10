import { computed, ref, watch, type Ref } from 'vue'
import { getWorkspaceModes, type WorkspaceModeItem, type WorkspaceSessionItem } from '@/api/workspace'
import { Logger } from '@/core/utils/logger'

const DEFAULT_MODE_CODE = 'dev'

function getModeStorageKey(fullCodePath: string): string {
  return `workspace-mode-code:${fullCodePath}`
}

function normalizeModeCode(code: string | null | undefined): string {
  const normalized = String(code || '').trim()
  return normalized || DEFAULT_MODE_CODE
}

function buildFallbackMode(code: string): WorkspaceModeItem {
  return {
    id: -1,
    code,
    name: `${code}（当前会话）`,
    description: '',
    system_prompt_fragment: '',
    tool_names: [],
    agent_id: null,
    sort_order: 0,
    is_builtin: false,
  }
}

export function useWorkspaceModeSelection(fullCodePath: Ref<string>) {
  const modeList = ref<WorkspaceModeItem[]>([])
  const modeLoading = ref(false)
  const selectedModeCode = ref<string>(DEFAULT_MODE_CODE)

  const modeOptions = computed<WorkspaceModeItem[]>(() => {
    const list = [...modeList.value]
    const current = normalizeModeCode(selectedModeCode.value)
    if (!list.some(mode => mode.code === current)) {
      list.unshift(buildFallbackMode(current))
    }
    return list
  })

  function persistSelectedMode() {
    const path = fullCodePath.value.trim()
    if (!path) return
    try {
      localStorage.setItem(getModeStorageKey(path), normalizeModeCode(selectedModeCode.value))
    } catch {
      // ignore storage failures
    }
  }

  function restoreSelectedMode(path: string) {
    if (!path) {
      selectedModeCode.value = DEFAULT_MODE_CODE
      return
    }
    try {
      selectedModeCode.value = normalizeModeCode(localStorage.getItem(getModeStorageKey(path)))
    } catch {
      selectedModeCode.value = DEFAULT_MODE_CODE
    }
  }

  function ensureValidSelectedMode() {
    if (modeList.value.length === 0) {
      selectedModeCode.value = normalizeModeCode(selectedModeCode.value)
      return
    }
    const current = normalizeModeCode(selectedModeCode.value)
    if (modeList.value.some(mode => mode.code === current)) {
      selectedModeCode.value = current
      return
    }
    const devMode = modeList.value.find(mode => mode.code === DEFAULT_MODE_CODE)
    selectedModeCode.value = devMode?.code || modeList.value[0]?.code || DEFAULT_MODE_CODE
  }

  async function loadModes() {
    modeLoading.value = true
    try {
      const response = await getWorkspaceModes({ page: 1, page_size: 200 })
      modeList.value = response.list || []
      ensureValidSelectedMode()
    } catch (error) {
      Logger.error('[useWorkspaceModeSelection]', '加载工作台模式失败', { error })
      modeList.value = []
      selectedModeCode.value = normalizeModeCode(selectedModeCode.value)
    } finally {
      modeLoading.value = false
    }
  }

  function setSelectedModeCode(code: string) {
    selectedModeCode.value = normalizeModeCode(code)
    persistSelectedMode()
  }

  function applySessionMode(session?: Pick<WorkspaceSessionItem, 'mode_code'> | null) {
    const code = normalizeModeCode(session?.mode_code)
    selectedModeCode.value = code
    persistSelectedMode()
  }

  watch(
    fullCodePath,
    (path) => {
      restoreSelectedMode(path.trim())
      if (path.trim()) {
        void loadModes()
      } else {
        modeList.value = []
      }
    },
    { immediate: true }
  )

  return {
    modeList,
    modeOptions,
    modeLoading,
    selectedModeCode,
    loadModes,
    setSelectedModeCode,
    applySessionMode,
  }
}
