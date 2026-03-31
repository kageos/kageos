import type { ApiResponse } from '@/types'

type ApiResponseEnvelope<T = unknown> = ApiResponse<T>

function isApiResponseEnvelope<T = unknown>(value: unknown): value is ApiResponseEnvelope<T> {
  return Boolean(
    value &&
    typeof value === 'object' &&
    typeof (value as ApiResponseEnvelope<T>).code === 'number' &&
    (
      'data' in (value as Record<string, unknown>) ||
      'msg' in (value as Record<string, unknown>) ||
      'message' in (value as Record<string, unknown>)
    )
  )
}

export function getErrorMessage(error: unknown, fallback: string = '操作失败，请重试'): string {
  if (typeof error === 'string' && error.trim()) {
    return error
  }

  if (error && typeof error === 'object') {
    const responseData = (error as any).response?.data
    if (responseData?.msg) {
      return responseData.msg
    }
    if (responseData?.message) {
      return responseData.message
    }
    if ((error as any).msg) {
      return (error as any).msg
    }
    if ((error as any).message) {
      return (error as any).message
    }
  }

  return fallback
}

export function createBusinessError(
  payload: { msg?: string; message?: string } | undefined,
  fallback: string
): Error & { response?: { data: { msg: string } } } {
  const message = payload?.msg || payload?.message || fallback
  const error = new Error(message) as Error & { response?: { data: { msg: string } } }
  error.response = {
    data: {
      msg: message
    }
  }
  return error
}

export function unwrapApiResponseData<T>(response: T | ApiResponseEnvelope<T>, fallback: string): T {
  if (!isApiResponseEnvelope<T>(response)) {
    return response as T
  }

  if (response.code === 0) {
    const normalizedData = response.data as T
    if (
      response.metadata &&
      normalizedData &&
      typeof normalizedData === 'object' &&
      !Array.isArray(normalizedData) &&
      !(normalizedData instanceof Blob) &&
      !('_metadata' in (normalizedData as Record<string, unknown>))
    ) {
      ;(normalizedData as Record<string, unknown>)._metadata = response.metadata
    }
    return normalizedData
  }

  throw createBusinessError(response, fallback)
}
