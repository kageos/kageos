import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import {
  collectMessageToolCalls,
  useMiniWorkstationDebugCopy
} from './useMiniWorkstationDebugCopy'

describe('useMiniWorkstationDebugCopy', () => {
  it('collects tool calls from assistant blocks first', () => {
    const calls = collectMessageToolCalls({
      role: 'assistant',
      content: '',
      tool_calls: [{ name: 'legacy_tool', status: 'ok' }],
      blocks: [
        { type: 'content', text: 'hello' },
        {
          type: 'tool_calls',
          calls: [
            { name: 'write_go_file', status: 'ok' },
            { name: 'build_workspace', status: 'error', error: 'compile failed' }
          ]
        }
      ]
    } as any)

    expect(calls.map(call => call.name)).toEqual(['write_go_file', 'build_workspace'])
  })

  it('builds debug tool steps from messages', () => {
    const { debugToolSteps, debugSuccessCount, debugErrorCount } = useMiniWorkstationDebugCopy({
      messages: ref([
        {
          role: 'assistant',
          content: '',
          tool_calls: [
            { name: 'write_go_file', status: 'ok', arguments: '{"path":"main.go"}', result: 'done' },
            { name: 'build_workspace', status: 'error', error: 'compile failed' }
          ]
        }
      ] as any),
      fullCodePath: () => '/luobei/demo',
      dirName: () => 'demo',
      displayPath: ref('demo'),
      sessionId: ref('s1')
    })

    expect(debugToolSteps.value).toHaveLength(2)
    expect(debugToolSteps.value[0]).toMatchObject({
      index: 1,
      name: 'write_go_file',
      statusLabel: '成功',
      statusClass: 'ok'
    })
    expect(debugToolSteps.value[1]).toMatchObject({
      index: 2,
      name: 'build_workspace',
      statusLabel: '失败',
      statusClass: 'error'
    })
    expect(debugSuccessCount.value).toBe(1)
    expect(debugErrorCount.value).toBe(1)
  })
})
