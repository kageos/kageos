import { describe, expect, it, vi } from 'vitest'
import { FormGatewayImpl } from './FormGatewayImpl'

describe('FormGatewayImpl', () => {
  it('submits form data through the standard form submit endpoint', async () => {
    const apiClient = {
      get: vi.fn(),
      post: vi.fn().mockResolvedValue({ ok: true }),
      put: vi.fn(),
      delete: vi.fn()
    }
    const gateway = new FormGatewayImpl(apiClient)

    const response = await gateway.submitForm({
      functionDetail: {
        method: 'POST',
        router: '/test/form-submit'
      } as any,
      data: { name: 'Alice' }
    })

    expect(apiClient.post).toHaveBeenCalledWith(
      '/workspace/api/v1/form/submit/test/form-submit',
      { name: 'Alice' }
    )
    expect(response).toEqual({ ok: true })
  })

  it('passes GET form submissions as query params', async () => {
    const apiClient = {
      get: vi.fn().mockResolvedValue({ ok: true }),
      post: vi.fn(),
      put: vi.fn(),
      delete: vi.fn()
    }
    const gateway = new FormGatewayImpl(apiClient)

    await gateway.submitForm({
      functionDetail: {
        method: 'GET',
        router: 'test/form-search'
      } as any,
      data: { keyword: 'Alice' }
    })

    expect(apiClient.get).toHaveBeenCalledWith(
      '/workspace/api/v1/form/submit/test/form-search',
      { keyword: 'Alice' }
    )
  })
})

