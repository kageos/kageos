import type { FunctionDetail } from '../types'

export interface FormSubmitRequest {
  functionDetail: FunctionDetail
  data: Record<string, any>
}

export interface IFormGateway {
  submitForm(request: FormSubmitRequest): Promise<any>
}

