import { apiClient } from '@/architecture/infrastructure/apiClient'
import { TableGatewayImpl } from './TableGatewayImpl'

export { TableGatewayImpl } from './TableGatewayImpl'

export const tableGateway = new TableGatewayImpl(apiClient)

