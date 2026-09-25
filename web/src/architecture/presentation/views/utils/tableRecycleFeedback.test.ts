import { describe, expect, it } from 'vitest'
import { tableRecycleFeedback } from './tableRecycleFeedback'

describe('recycle feedback', () => {
  for (const action of ['恢复', '彻底删除'] as const) {
    it(`${action}: preserves zero and reports partial results`, () => {
      expect(tableRecycleFeedback(action, 0, 3)).toEqual({ type: 'warning', message: `未${action}任何记录（选中 3 条），请刷新列表核对。` })
      expect(tableRecycleFeedback(action, 1, 3)).toEqual({ type: 'warning', message: `已${action} 1 条，另有 2 条未处理，请刷新列表核对。` })
      expect(tableRecycleFeedback(action, 3, 3).type).toBe('success')
    })
  }
  it('never invents success for missing or invalid counts', () => {
    for (const value of [undefined, null, -1, 1.5, 4, '3', NaN]) {
      expect(tableRecycleFeedback('恢复', value, 3).type).toBe('warning')
    }
  })
})
