import type { UserInfo } from '@/architecture/domain/types'
import type { Department } from '@/architecture/presentation/context/api/department'

export type SearchValue = string | number | boolean | null | undefined | unknown

export type SearchOption = {
  label: string
  value: SearchValue
  userInfo?: UserInfo
  departmentInfo?: Department
}

export interface SearchInputConfig {
  component: string
  props?: Record<string, unknown>
  onRemoteMethod?: (query: string) => Promise<SearchOption[]>
  onInitOptions?: (value: SearchValue | SearchValue[]) => Promise<SearchOption[]>
}
