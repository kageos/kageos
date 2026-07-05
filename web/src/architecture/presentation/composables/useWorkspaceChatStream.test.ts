import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useWorkspaceChatStream, type StreamEventHandler, type UseWorkspaceChatStreamReturn } from './useWorkspaceChatStream'

vi.mock('@/architecture/presentation/context/appStoresContext', () => ({
  useAuthStore: () => ({
    user: { username: 'demo' },
    userName: 'demo',
  }),
}))

function setupStream() {
  let stream: UseWorkspaceChatStreamReturn | undefined
  const wrapper = mount(defineComponent({
    setup() {
      stream = useWorkspaceChatStream()
      return () => h('div')
    },
  }))
  if (!stream) throw new Error('stream setup failed')
  stream.setMessages([
    { role: 'user', user: 'demo', content: '看一下系统', created_at: '2026-07-05T10:00:00Z' },
    { role: 'assistant', user: 'demo', content: '', tool_calls: [], blocks: [], created_at: '2026-07-05T10:00:00Z' },
  ])
  return { stream, wrapper }
}

function setupEmptyStream() {
  let stream: UseWorkspaceChatStreamReturn | undefined
  const wrapper = mount(defineComponent({
    setup() {
      stream = useWorkspaceChatStream()
      return () => h('div')
    },
  }))
  if (!stream) throw new Error('stream setup failed')
  return { stream, wrapper }
}

function deferred<T = void>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('useWorkspaceChatStream', () => {
  it('matches streamed tool calls by round and index instead of reusing index zero across rounds', () => {
    const { stream, wrapper } = setupStream()

    stream.handleEvent('tool_calls_stream_delta', {
      updates: [
        { id: 'call_change', index: 0, round: 0, name: 'change_role', delta: '{"target_role":"reviewer"}' },
      ],
    })
    stream.handleEvent('tool_calls_stream_delta', {
      updates: [
        { id: 'call_read', index: 0, round: 1, delta: '{"directory":"/system/app"}' },
      ],
    })

    expect(stream.messages.value[1]?.tool_calls?.map(call => call.name)).toEqual(['change_role', ''])
    expect(stream.messages.value[1]?.tool_calls?.map(call => `${call.round}:${call.index}:${call.id}`)).toEqual([
      '0:0:call_change',
      '1:0:call_read',
    ])

    stream.handleEvent('tool_call', {
      id: 'call_change',
      index: 0,
      round: 0,
      name: 'change_role',
      status: 'ok',
      arguments: '{"target_role":"reviewer"}',
    })
    stream.handleEvent('tool_call', {
      id: 'call_read',
      index: 0,
      round: 1,
      name: 'read_dir',
      status: 'running',
      arguments: '{"directory":"/system/app"}',
    })

    expect(stream.messages.value[1]?.tool_calls?.map(call => `${call.name}:${call.status}`)).toEqual([
      'change_role:ok',
      'read_dir:running',
    ])

    wrapper.unmount()
  })

  it('ignores stale SSE events after starting a new session', async () => {
    const { stream, wrapper } = setupEmptyStream()
    const oldStarted = deferred<StreamEventHandler>()
    const oldDone = deferred()
    const newStarted = deferred<StreamEventHandler>()
    const newDone = deferred()

    const oldSend = stream.send('第一轮', async (onEvent) => {
      oldStarted.resolve(onEvent)
      await oldDone.promise
    })
    const oldOnEvent = await oldStarted.promise
    oldOnEvent('session', { session_id: 'session-old' })
    oldOnEvent('content', { content: '旧会话输出' })

    expect(stream.sessionId.value).toBe('session-old')
    expect(stream.messages.value[1]?.content).toBe('旧会话输出')

    stream.sending.value = false
    stream.sessionId.value = undefined
    stream.setMessages([])

    const newSend = stream.send('第二轮', async (onEvent) => {
      newStarted.resolve(onEvent)
      await newDone.promise
    })
    const newOnEvent = await newStarted.promise

    expect(oldOnEvent('content', { content: '不应该串到新会话' })).toBe(false)
    expect(oldOnEvent('session', { session_id: 'session-old' })).toBe(false)
    expect(stream.sessionId.value).toBeUndefined()
    expect(stream.messages.value[1]?.content).toBe('')

    newOnEvent('session', { session_id: 'session-new' })
    newOnEvent('content', { content: '新会话输出' })
    expect(stream.sessionId.value).toBe('session-new')
    expect(stream.messages.value[1]?.content).toBe('新会话输出')

    oldDone.resolve()
    await oldSend
    expect(stream.sending.value).toBe(true)

    newOnEvent('done', { session_id: 'session-new' })
    newDone.resolve()
    await newSend
    expect(stream.sending.value).toBe(false)

    wrapper.unmount()
  })
})
