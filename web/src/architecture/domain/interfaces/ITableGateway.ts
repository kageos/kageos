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
  data: Record<string, unknown>
  oldData?: Record<string, unknown>
}

export interface TableAddOptions {
  operation?: 'create' | 'import'
}

export interface ITableGateway {
  loadRows(request: TableLoadRequest): Promise<TableListResponse>
  addRow(functionDetail: FunctionDetail, data: Record<string, unknown>, options?: TableAddOptions): Promise<TableRow>
  updateRow(request: TableUpdateRequest): Promise<TableRow>
  deleteRow(functionDetail: FunctionDetail, id: number | string): Promise<void>
}
