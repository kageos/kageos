export type SearchOption = {
  label: string
  value: any
  userInfo?: any
  departmentInfo?: any
}

export interface SearchInputConfig {
  component: string
  props?: Record<string, any>
  onRemoteMethod?: (query: string) => Promise<SearchOption[]>
  onInitOptions?: (value: any) => Promise<SearchOption[]>
}
