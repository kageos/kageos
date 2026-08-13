import { describe, expect, it, vi } from 'vitest'
import { createChunkLoadRecovery, isChunkLoadError } from './chunkLoadRecovery'

describe('chunk load recovery', () => {
  it.each([
    'Failed to fetch dynamically imported module: https://app.kageos.com/assets/Page-old.js',
    'Importing a module script failed.',
    'ChunkLoadError: Loading chunk 42 failed',
  ])('recognizes stale deployment chunk errors: %s', (message) => {
    expect(isChunkLoadError(new TypeError(message))).toBe(true)
  })

  it('ignores unrelated navigation failures', () => {
    expect(isChunkLoadError(new Error('request was unauthorized'))).toBe(false)
  })

  it('reloads once for the same target and can recover again after a successful navigation', () => {
    let marker: string | null = null
    const reload = vi.fn()
    const recovery = createChunkLoadRecovery({
      readMarker: () => marker,
      writeMarker: (value) => {
        marker = value
      },
      clearMarker: () => {
        marker = null
      },
      reload,
    })
    const error = new TypeError('Failed to fetch dynamically imported module: /assets/Page-old.js')

    expect(recovery.recover(error, '/agent/llm')).toBe(true)
    expect(recovery.recover(error, '/agent/llm')).toBe(false)
    expect(reload).toHaveBeenCalledTimes(1)

    recovery.clear()
    expect(recovery.recover(error, '/agent/llm')).toBe(true)
    expect(reload).toHaveBeenCalledTimes(2)
  })
})
