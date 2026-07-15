import { describe, expect, it, vi } from 'vitest'
import { TableGatewayImpl } from './TableGatewayImpl'

describe('TableGatewayImpl', () => {
  it('sends table updates through the standard table update endpoint with changed fields only', async () => {
    const apiClient = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn().mockResolvedValue({
        id: 2,
        name: 'Bobby',
        status: 'closed'
      }),
      delete: vi.fn()
    }
    const gateway = new TableGatewayImpl(apiClient)

    const response = await gateway.updateRow({
      functionDetail: { router: '/workspace/demo/users' } as any,
      id: 2,
      data: {
        name: 'Bobby',
        status: 'closed'
      },
      oldData: {
        id: 2,
        name: 'Bob',
        status: 'closed'
      }
    })

    expect(apiClient.put).toHaveBeenCalledWith(
      '/workspace/api/v1/table/update/workspace/demo/users',
      {
        id: 2,
        updates: {
          name: 'Bobby'
        },
        old_values: {
          name: 'Bob'
        }
      }
    )
    expect(response).toEqual({
      id: 2,
      name: 'Bobby',
      status: 'closed'
    })
  })
})

