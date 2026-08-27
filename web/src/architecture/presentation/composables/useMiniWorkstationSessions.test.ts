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

import {
  getFinishedSessions,
  getRunningSessions,
  getWorkspaceMessages,
  getWorkspaceSessions,
} from '@/architecture/presentation/context/api/workspace'
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

  it('exposes current-directory loading failures for a retryable UI', async () => {
    vi.mocked(getWorkspaceSessions).mockRejectedValueOnce(new Error('offline'))
    const sessions = useMiniWorkstationSessions({
      fullCodePath: ref('/system/demo'),
      initialSessionId: ref(undefined),
      maximized: ref(false),
      sending: ref(false),
      sessionId: ref(undefined),
      setMessages: vi.fn(),
      sessionSourceFilter: ref('human'),
    })

    await sessions.loadMiniSessions()

    expect(sessions.loadingSessions.value).toBe(false)
    expect(sessions.sessionLoadFailed.value).toBe(true)
    expect(sessions.miniSessionList.value).toEqual([])
  })

  it('keeps partial global session results without reporting a total failure', async () => {
    vi.mocked(getRunningSessions).mockRejectedValueOnce(new Error('running unavailable'))
    vi.mocked(getFinishedSessions).mockResolvedValueOnce({
      sessions: [{
        session_id: 'finished-1',
        title: '已完成任务',
        status: 'done',
        created_at: '2026-08-27T00:00:00Z',
        updated_at: '2026-08-27T01:00:00Z',
      }],
    } as any)
    const sessions = useMiniWorkstationSessions({
      fullCodePath: ref('/system/demo'),
      initialSessionId: ref(undefined),
      maximized: ref(false),
      sending: ref(false),
      sessionId: ref(undefined),
      setMessages: vi.fn(),
      sessionSourceFilter: ref('human'),
    })

    await sessions.loadGlobalSessions()

    expect(sessions.globalSessionLoadFailed.value).toBe(false)
    expect(sessions.globalSessionList.value.map(item => item.session_id)).toEqual(['finished-1'])
  })
})
