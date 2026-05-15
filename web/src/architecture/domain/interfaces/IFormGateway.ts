import type { FunctionDetail } from '../types'

export interface FormSubmitRequest {
  functionDetail: FunctionDetail
  data: Record<string, unknown>
}

export interface IFormGateway {
  submitForm(request: FormSubmitRequest): Promise<unknown>
}
