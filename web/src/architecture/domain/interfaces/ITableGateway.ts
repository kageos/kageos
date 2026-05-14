import type {
  FunctionDetail,
  TableListResponse,
  TableRow,
  TableSearchParams
} from '../types'

export interface TableLoadRequest {
  functionDetail: FunctionDetail
  params: TableSearchParams
}

export interface TableUpdateRequest {
  functionDetail: FunctionDetail
  id: number | string
  data: Record<string, any>
  oldData?: Record<string, any>
}

export interface ITableGateway {
  loadRows(request: TableLoadRequest): Promise<TableListResponse>
  addRow(functionDetail: FunctionDetail, data: Record<string, any>): Promise<TableRow>
  updateRow(request: TableUpdateRequest): Promise<TableRow>
  deleteRow(functionDetail: FunctionDetail, id: number | string): Promise<void>
}

