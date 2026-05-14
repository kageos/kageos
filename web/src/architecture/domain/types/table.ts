import type { FunctionDetail } from './field'

export interface TableRow {
  id: number | string
  [key: string]: any
}

export interface TableListResponse {
  items: TableRow[]
  paginated?: {
    current_page: number
    page_size: number
    total_count: number
    total_pages: number
  }
}

export interface TableSearchParams {
  [key: string]: any
}

export interface SortParams {
  field: string
  order: 'asc' | 'desc'
}

export interface SortItem {
  field: string
  order: 'asc' | 'desc'
}

export interface TableState {
  data: TableRow[]
  loading: boolean
  searchParams: TableSearchParams
  searchForm: Record<string, any>
  sortParams: SortParams | null
  sorts: SortItem[]
  hasManualSort: boolean
  pagination: {
    currentPage: number
    pageSize: number
    total: number
  }
}

export interface TableDataHook {
  name: string
  priority: number
  execute: (functionDetail: FunctionDetail, tableData: TableRow[]) => Promise<void>
}
