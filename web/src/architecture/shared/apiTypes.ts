export interface ApiResponse<T = unknown> {
	code: string | number
  data: T
  message?: string
  msg?: string
  metadata?: Record<string, unknown>
}
