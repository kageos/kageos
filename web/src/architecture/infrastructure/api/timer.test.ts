import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/architecture/infrastructure/apiClient/request', () => ({
  authFetch: vi.fn(),
}))

vi.mock('@/architecture/infrastructure/config/runtime', () => ({
  getApiBaseURL: () => 'http://localhost',
}))

import { authFetch } from '@/architecture/infrastructure/apiClient/request'
import { listTimerTasks } from './timer'

describe('timer API responses', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('preserves the HTTP status when an error response is not JSON', async () => {
    vi.mocked(authFetch).mockResolvedValue(new Response('bad gateway', {
      status: 502,
      headers: { 'Content-Type': 'text/plain' },
    }))

    await expect(listTimerTasks()).rejects.toThrow('HTTP 502: bad gateway')
  })
})
