import type { IApiClient } from '@/architecture/domain/interfaces/IApiClient'
import type { FormSubmitRequest, IFormGateway } from '@/architecture/domain/interfaces/IFormGateway'

function toFullCodePath(router: string | undefined): string {
  return router?.startsWith('/') ? router : `/${router || ''}`
}

export class FormGatewayImpl implements IFormGateway {
  constructor(private apiClient: IApiClient) {}

  submitForm(request: FormSubmitRequest): Promise<unknown> {
    const fullCodePath = toFullCodePath(request.functionDetail.router)
    const url = `/workspace/api/v1/form/submit${fullCodePath}`
    const method = request.functionDetail.method?.toUpperCase() || 'POST'

    if (method === 'GET') {
      return this.apiClient.get(url, request.data)
    }

    return this.apiClient.post(url, request.data)
  }
}
