import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

vi.mock('@/architecture/presentation/context/api/workspace', () => ({
  cancelWorkspaceChat: vi.fn(),
  getFinishedSessions: vi.fn(),
  getRunningSessions: vi.fn(),
  getWorkspaceMessages: vi.fn(),
  getWorkspaceSessionSSEStatus: vi.fn(),
  getWorkspaceSessions: vi.fn(),
}))

import { getWorkspaceMessages } from '@/architecture/presentation/context/api/workspace'
import { useMiniWorkstationSessions } from './useMiniWorkstationSessions'

describe('useMiniWorkstationSessions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('does not let a stale session response overwrite the selected session', async () => {
    let resolveRequest: ((value: { messages: [] }) => void) | undefined
    vi.mocked(getWorkspaceMessages).mockReturnValue(new Promise(resolve => {
      resolveRequest = resolve
    }))
    const sessionId = ref<string | undefined>('session-a')
    const setMessages = vi.fn()
    const sessions = useMiniWorkstationSessions({
      fullCodePath: ref('/system/demo'),
      initialSessionId: ref(undefined),
      maximized: ref(false),
      sending: ref(false),
      sessionId,
      setMessages,
      sessionSourceFilter: ref('human'),
    })

    const pending = sessions.loadMiniSessionMessages('session-a')
    sessionId.value = 'session-b'
    resolveRequest?.({ messages: [] })
    await pending

    expect(setMessages).not.toHaveBeenCalled()
  })
})
