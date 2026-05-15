export interface ApiResponse<T = unknown> {
  code: number
  data: T
  message?: string
  msg?: string
  metadata?: Record<string, unknown>
}
