import { effectScope, nextTick, ref } from 'vue'
import { flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { workspaceChatStream, type WorkspaceChatMessageFile } from '@/architecture/presentation/context/api/workspace'
import { useMiniWorkstationComposer, type UseMiniWorkstationComposerOptions } from './useMiniWorkstationComposer'

vi.mock('@/architecture/presentation/context/api/agent', () => ({ getLLMList: vi.fn() }))
vi.mock('@/architecture/presentation/context/api/workspace', () => ({ workspaceChatStream: vi.fn(async () => {}) }))
vi.mock('element-plus', () => ({ ElMessage: { success: vi.fn(), warning: vi.fn(), error: vi.fn() } }))
vi.mock('@/architecture/shared/i18n', () => ({ translate: (key: string) => key }))

function setup() {
  vi.mocked(workspaceChatStream).mockClear()
  const scope = effectScope()
  const sending = ref(false)
  const state = {
    sessionId: ref<string | undefined>('A'), fullCodePath: ref('/system/app'), maximized: ref(false),
    inputText: ref('message'), inputRef: ref(undefined), attachedFiles: ref<WorkspaceChatMessageFile[]>([]),
    uploading: ref(false), sending,
    beforeSend: vi.fn<NonNullable<UseMiniWorkstationComposerOptions['beforeSend']>>(() => false),
    sendMessage: vi.fn<UseMiniWorkstationComposerOptions['sendMessage']>(async (_text, stream) => {
      sending.value = true
      try { await stream(() => {}) } finally { sending.value = false }
    })
  }
  const composer = scope.run(() => useMiniWorkstationComposer(state))!
  return { state, composer, dispose: () => scope.stop() }
}

describe('workstation sending guards', () => {
  it('blocks both button and Enter sends during uploads and preserves the draft', async () => {
    const { state, composer, dispose } = setup()
    try {
      state.uploading.value = true
      state.attachedFiles.value = [{ ref: 'file-1' } as WorkspaceChatMessageFile]
      await composer.handleSend()
      composer.onInputEnter(new KeyboardEvent('keydown', { key: 'Enter', cancelable: true }))
      await flushPromises()
      expect(state.sendMessage).not.toHaveBeenCalled()
      expect(state.inputText.value).toBe('message')
      expect(state.attachedFiles.value).toHaveLength(1)
      state.uploading.value = false
      await composer.handleSend()
      expect(workspaceChatStream).toHaveBeenCalledWith(expect.objectContaining({ message: expect.objectContaining({ files: 'file-1' }) }), expect.any(Function), expect.any(Object))
    } finally { dispose() }
  })

  it('keeps queued messages in their original session and directory', async () => {
    const { state, composer, dispose } = setup()
    try {
      state.sending.value = true
      await composer.handleSend()
      state.sending.value = false
      state.sessionId.value = 'B'
      await flushPromises()
      expect(state.sendMessage).not.toHaveBeenCalled()
      expect(composer.queuedCount.value).toBe(0)
      state.sessionId.value = 'A'
      state.fullCodePath.value = '/other'
      await flushPromises()
      expect(state.sendMessage).not.toHaveBeenCalled()
      state.fullCodePath.value = '/system/app'
      await flushPromises()
      expect(workspaceChatStream).toHaveBeenCalledTimes(1)
      expect(workspaceChatStream).toHaveBeenCalledWith(expect.objectContaining({ session_id: 'A', full_code_path: '/system/app' }), expect.any(Function), expect.any(Object))
    } finally { dispose() }
  })

  it('rechecks queued messages and retains them when confirmation is pending', async () => {
    const { state, composer, dispose } = setup()
    try {
      state.sending.value = true
      await composer.handleSend()
      state.beforeSend.mockReturnValue({ cancel: true, preserveDraft: true })
      state.sending.value = false
      await flushPromises()
      expect(state.beforeSend).toHaveBeenCalledWith({ text: 'message', files: null })
      expect(state.sendMessage).not.toHaveBeenCalled()
      expect(composer.queuedCount.value).toBe(1)
      state.beforeSend.mockReturnValue({ interactionAction: 'continue_development' })
      await composer.retryQueuedMessages()
      expect(composer.queuedCount.value).toBe(0)
      expect(workspaceChatStream).toHaveBeenCalledWith(expect.objectContaining({ message: expect.objectContaining({ interaction_action: 'continue_development' }) }), expect.any(Function), expect.any(Object))
    } finally { dispose() }
  })

  it('does not redirect a message when the session changes during an async check', async () => {
    const { state, composer, dispose } = setup()
    let resolve!: (value: boolean) => void
    try {
      state.beforeSend.mockImplementation(() => new Promise(r => { resolve = r }))
      const pending = composer.handleSend()
      state.sessionId.value = 'B'
      resolve(false)
      await pending
      expect(state.sendMessage).not.toHaveBeenCalled()
      expect(state.inputText.value).toBe('message')
    } finally { dispose() }
  })

  it('does not redirect a queued message during an async check', async () => {
    const { state, composer, dispose } = setup()
    let resolve!: (value: boolean) => void
    try {
      state.sending.value = true
      await composer.handleSend()
      state.beforeSend.mockImplementation(() => new Promise(r => { resolve = r }))
      state.sending.value = false
      await nextTick()
      state.sessionId.value = 'B'
      resolve(false)
      await flushPromises()
      expect(state.sendMessage).not.toHaveBeenCalled()
      state.beforeSend.mockReturnValue(false)
      state.sessionId.value = 'A'
      await flushPromises()
      expect(state.sendMessage).toHaveBeenCalledTimes(1)
    } finally { dispose() }
  })

  it('keeps a draft if the first session has not been assigned yet', async () => {
    const { state, composer, dispose } = setup()
    try {
      state.sending.value = true
      state.sessionId.value = undefined
      await composer.handleSend()
      expect(state.inputText.value).toBe('message')
      expect(composer.queuedCount.value).toBe(0)
    } finally { dispose() }
  })

  it('drains multiple messages once each in order', async () => {
    const { state, composer, dispose } = setup()
    try {
      state.sending.value = true
      await composer.handleSend()
      state.inputText.value = 'second'
      await composer.handleSend()
      state.sending.value = false
      await flushPromises()
      expect(state.sendMessage.mock.calls.map(call => call[0])).toEqual(['message', 'second'])
      expect(state.beforeSend).toHaveBeenCalledTimes(2)
    } finally { dispose() }
  })
})

it('pauses the queue during uploads and resumes after they finish', async () => {
  const { state, composer, dispose } = setup()
  try {
    state.sending.value = true
    await composer.handleSend()
    state.uploading.value = true
    state.sending.value = false
    await flushPromises()
    expect(state.beforeSend).not.toHaveBeenCalled()
    expect(state.sendMessage).not.toHaveBeenCalled()
    state.uploading.value = false
    await flushPromises()
    expect(state.sendMessage).toHaveBeenCalledTimes(1)
  } finally { dispose() }
})

it('keeps queued messages when the pre-send check throws', async () => {
  const { state, composer, dispose } = setup()
  try {
    state.sending.value = true
    await composer.handleSend()
    state.beforeSend.mockRejectedValue(new Error('offline'))
    state.sending.value = false
    await flushPromises()
    expect(composer.queuedCount.value).toBe(1)
    expect(state.sendMessage).not.toHaveBeenCalled()
  } finally { dispose() }
})

it('preserves text typed during the pre-send check and rejects duplicate submissions', async () => {
  const { state, composer, dispose } = setup()
  let finish!: (value: boolean) => void
  try {
    state.beforeSend.mockImplementation(() => new Promise(resolve => { finish = resolve }))
    const first = composer.handleSend()
    await composer.handleSend()
    state.inputText.value = 'new draft'
    finish(false)
    await first
    expect(state.sendMessage).toHaveBeenCalledTimes(1)
    expect(state.inputText.value).toBe('new draft')
  } finally { dispose() }
})
