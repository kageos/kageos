import { describe, expect, it } from 'vitest'
import { splitWorkspaceThinkBlocks, stripWorkspaceThinkBlocks } from './useWorkspaceThinkFilter'

describe('stripWorkspaceThinkBlocks', () => {
  it('removes complete think blocks from assistant display content', () => {
    expect(stripWorkspaceThinkBlocks('<think>内部思考</think>\n\n最终答案')).toBe('最终答案')
  })

  it('removes an unfinished think tail', () => {
    expect(stripWorkspaceThinkBlocks('前置答案\n<think>未闭合思考')).toBe('前置答案\n')
  })

  it('keeps regular content unchanged', () => {
    expect(stripWorkspaceThinkBlocks('普通答案')).toBe('普通答案')
  })

  it('splits old assistant content into thinking and answer segments', () => {
    expect(splitWorkspaceThinkBlocks('<think>内部思考</think>\n\n最终答案')).toEqual([
      { type: 'thinking', text: '内部思考' },
      { type: 'content', text: '最终答案' }
    ])
  })
})
