import { computed, ref } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { ChatMessage } from './useWorkspaceChatStream'
import {
  buildWorkspaceInteractionFromArtifact,
  useMiniWorkstationPendingInteraction
} from './useMiniWorkstationPendingInteraction'

const recordWorkspaceInteractionEventMock = vi.hoisted(() => vi.fn(async () => {}))
const resolveWorkspaceSessionInteractionMock = vi.hoisted(() => vi.fn(async () => {}))
const createWorkspaceHandoffMock = vi.hoisted(() => vi.fn(async () => ({
  session_id: 'handoff-session',
  content: 'handoff content',
  display_content: 'handoff display',
  artifact_kind: 'agent_app_prd'
})))

vi.mock('@/architecture/presentation/context/api/workspace', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/architecture/presentation/context/api/workspace')>()
  return {
    ...actual,
    recordWorkspaceInteractionEvent: recordWorkspaceInteractionEventMock,
    resolveWorkspaceSessionInteraction: resolveWorkspaceSessionInteractionMock,
    createWorkspaceHandoff: createWorkspaceHandoffMock
  }
})

function assistantWithArtifact(resultData: unknown): ChatMessage {
  return {
    role: 'assistant',
    content: '',
    tool_calls: [{
      name: 'write_prd',
      status: 'ok',
      result_data: resultData
    }]
  }
}

function createHarness(messagesValue: ChatMessage[]) {
  const messages = ref<ChatMessage[]>(messagesValue)
  const sessionId = ref<string | undefined>('session-1')
  const fullCodePath = ref('/alice/demo/ticket')
  const sending = ref(false)
  const currentSessionDisablesPendingInteraction = ref(false)
  const outputRef = ref<HTMLElement>()
  const loadMiniSessions = vi.fn()
  const loadGlobalSessions = vi.fn()
  const loadMiniSessionMessages = vi.fn(async () => {})
  const sendTextToSession = vi.fn(async () => true)

  const api = useMiniWorkstationPendingInteraction({
    messages: computed(() => messages.value),
    sessionId,
    fullCodePath,
    sending,
    currentSessionDisablesPendingInteraction,
    outputRef,
    loadMiniSessions,
    loadGlobalSessions,
    loadMiniSessionMessages,
    sendTextToSession
  })

  return {
    api,
    messages,
    sessionId,
    fullCodePath,
    sending,
    currentSessionDisablesPendingInteraction,
    loadMiniSessions,
    loadGlobalSessions,
    loadMiniSessionMessages,
    sendTextToSession
  }
}

describe('useMiniWorkstationPendingInteraction', () => {
  beforeEach(() => {
    recordWorkspaceInteractionEventMock.mockClear()
    resolveWorkspaceSessionInteractionMock.mockClear()
    createWorkspaceHandoffMock.mockClear()
  })

  it('builds pending interactions from tool artifacts', () => {
    const interaction = buildWorkspaceInteractionFromArtifact({
      kind: 'agent_app_prd',
      interaction: {
        id: 'prd-1',
        status: 'pending_confirmation',
        blocking: true,
        title: '确认 PRD'
      }
    })

    expect(interaction).toMatchObject({
      id: 'prd-1',
      card_type: 'prd_confirmation',
      artifact_kind: 'agent_app_prd',
      status: 'pending_confirmation',
      blocking: true,
      title: '确认 PRD'
    })
  })

  it('blocks normal sending when a PRD confirmation is pending', async () => {
    const { api } = createHarness([
      assistantWithArtifact({
        kind: 'agent_app_prd',
        interaction: {
          id: 'prd-1',
          status: 'pending_confirmation',
          card_type: 'prd_confirmation',
          blocking: true
        }
      })
    ])

    expect(api.pendingInteraction.value?.id).toBe('prd-1')
    expect(api.composerBlocked.value).toBe(true)

    await expect(api.handleBeforeSend({ text: '继续实现', files: null })).resolves.toMatchObject({
      cancel: true,
      preserveDraft: true
    })
    expect(recordWorkspaceInteractionEventMock).not.toHaveBeenCalled()
  })

  it('records continue development before sending through build repair interactions', async () => {
    const { api, loadMiniSessionMessages } = createHarness([
      assistantWithArtifact({
        kind: 'agent_app_build_failure',
        interaction: {
          id: 'repair-1',
          status: 'pending_build_repair',
          card_type: 'build_repair',
          blocking: true,
          artifact_kind: 'agent_app_build_failure'
        }
      })
    ])

    expect(api.composerBlocked.value).toBe(false)
    await expect(api.handleBeforeSend({ text: '我来继续改', files: null })).resolves.toMatchObject({
      interactionAction: 'continue_development'
    })

    expect(recordWorkspaceInteractionEventMock).toHaveBeenCalledWith(expect.objectContaining({
      session_id: 'session-1',
      action: 'continue_development',
      interaction_id: 'repair-1',
      card_type: 'build_repair',
      artifact_kind: 'agent_app_build_failure'
    }))
    expect(loadMiniSessionMessages).toHaveBeenCalledWith('session-1')
    expect(api.pendingInteraction.value).toBeNull()
  })
})
