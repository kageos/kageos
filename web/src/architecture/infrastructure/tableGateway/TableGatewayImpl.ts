import type { IApiClient } from '@/architecture/domain/interfaces/IApiClient'
import type {
  ITableGateway,
  TableLoadRequest,
  TableUpdateRequest
} from '@/architecture/domain/interfaces/ITableGateway'
import type { FunctionDetail, TableListResponse, TableRow } from '@/architecture/domain/types'
import { getChangedFields } from '@/architecture/domain/utils/objectDiff'

function requireFunctionRouter(functionDetail: FunctionDetail): string {
  const router = functionDetail.router
  if (!router) {
    throw new Error('函数路由不存在')
  }
  return router
}

function toFullCodePath(router: string): string {
  return router.startsWith('/') ? router : `/${router}`
}

function buildUpdatePayload(
  id: number | string,
  newData: Record<string, any>,
  oldData?: Record<string, any>
): Record<string, any> {
  if (oldData) {
    const { updates, oldValues } = getChangedFields(oldData, newData)
    return {
      id,
      updates,
      old_values: oldValues
    }
  }

  return {
    id,
    ...newData
  }
}

export class TableGatewayImpl implements ITableGateway {
  constructor(private apiClient: IApiClient) {}

  loadRows(request: TableLoadRequest): Promise<TableListResponse> {
    const fullCodePath = toFullCodePath(requireFunctionRouter(request.functionDetail))
    return this.apiClient.get<TableListResponse>(
      `/workspace/api/v1/table/search${fullCodePath}`,
      request.params
    )
  }

  addRow(functionDetail: FunctionDetail, data: Record<string, any>): Promise<TableRow> {
    const fullCodePath = toFullCodePath(requireFunctionRouter(functionDetail))
    return this.apiClient.post<TableRow>(
      `/workspace/api/v1/table/create${fullCodePath}`,
      data
    )
  }

  updateRow(request: TableUpdateRequest): Promise<TableRow> {
    const fullCodePath = toFullCodePath(requireFunctionRouter(request.functionDetail))
    return this.apiClient.put<TableRow>(
      `/workspace/api/v1/table/update${fullCodePath}`,
      buildUpdatePayload(request.id, request.data, request.oldData)
    )
  }

  async deleteRow(functionDetail: FunctionDetail, id: number | string): Promise<void> {
    const fullCodePath = toFullCodePath(requireFunctionRouter(functionDetail))
    const ids = [typeof id === 'string' ? parseInt(id, 10) : id]
    await this.apiClient.delete<void>(
      `/workspace/api/v1/table/delete${fullCodePath}`,
      { ids }
    )
  }
}

