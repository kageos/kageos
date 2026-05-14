import { apiClient } from '@/architecture/infrastructure/apiClient'
import { FormGatewayImpl } from './FormGatewayImpl'

export { FormGatewayImpl } from './FormGatewayImpl'

export const formGateway = new FormGatewayImpl(apiClient)

