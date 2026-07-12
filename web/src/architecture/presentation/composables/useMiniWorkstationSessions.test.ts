import { describe, expect, it } from 'vitest'
import { resolveWorkspaceSessionSourceFilter } from './useMiniWorkstationSessions'

describe('resolveWorkspaceSessionSourceFilter', () => {
  it('defaults to human sessions', () => {
    expect(resolveWorkspaceSessionSourceFilter('human')).toEqual({ session_scope: 'human' })
    expect(resolveWorkspaceSessionSourceFilter('')).toEqual({ session_scope: 'human' })
  })

  it('resolves a selected automation agent task', () => {
    expect(resolveWorkspaceSessionSourceFilter('agent:42')).toEqual({
      session_scope: 'automation',
      automation_task_id: 42
    })
  })

  it('rejects invalid automation task identifiers', () => {
    expect(resolveWorkspaceSessionSourceFilter('agent:0')).toEqual({ session_scope: 'human' })
    expect(resolveWorkspaceSessionSourceFilter('agent:nope')).toEqual({ session_scope: 'human' })
  })
})
