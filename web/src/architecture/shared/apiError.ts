import type { ApiResponse } from './apiTypes'

type ApiResponseEnvelope<T = unknown> = ApiResponse<T>
type ErrorLike = {
  code?: unknown
  msg?: unknown
  message?: unknown
  status?: unknown
  response?: {
    status?: unknown
    data?: {
      code?: unknown
      msg?: unknown
      message?: unknown
      status?: unknown
    }
  }
}

function isApiResponseEnvelope<T = unknown>(value: unknown): value is ApiResponseEnvelope<T> {
  return Boolean(
    value &&
    typeof value === 'object' &&
	(typeof (value as ApiResponseEnvelope<T>).code === 'number' || typeof (value as ApiResponseEnvelope<T>).code === 'string') &&
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
    const errorLike = error as ErrorLike
    const responseData = errorLike.response?.data
    if (typeof responseData?.msg === 'string' && responseData.msg) {
      return responseData.msg
    }
    if (typeof responseData?.message === 'string' && responseData.message) {
      return responseData.message
    }
    if (typeof errorLike.msg === 'string' && errorLike.msg) {
      return errorLike.msg
    }
    if (typeof errorLike.message === 'string' && errorLike.message) {
      return errorLike.message
    }
  }

  return fallback
}

export function getErrorStatus(error: unknown): number | undefined {
  if (!error || typeof error !== 'object') {
    return undefined
  }

  const errorLike = error as ErrorLike
  const rawStatus = errorLike.response?.status ?? errorLike.response?.data?.status ?? errorLike.status
  if (typeof rawStatus === 'number') {
    return rawStatus
  }
  if (typeof rawStatus === 'string' && rawStatus.trim()) {
    const parsed = Number(rawStatus)
    return Number.isFinite(parsed) ? parsed : undefined
  }
  return undefined
}

export function getErrorCode(error: unknown): string | number | undefined {
  if (!error || typeof error !== 'object') {
    return undefined
  }

  const errorLike = error as ErrorLike
  const rawCode = errorLike.response?.data?.code ?? errorLike.code
  return typeof rawCode === 'string' || typeof rawCode === 'number' ? rawCode : undefined
}

export function isWorkspaceForbiddenError(error: unknown): boolean {
  if (getErrorStatus(error) === 403) {
    return true
  }

  const code = getErrorCode(error)
  const message = getErrorMessage(error, '').toLowerCase()
  const mentionsWorkspace = message.includes('workspace') || message.includes('工作空间')
  const mentionsPermission = /无权限|权限不足|请求被拒绝|forbidden|permission|access denied/.test(message)

	return code === 'permission_denied' || (code === 7 && mentionsWorkspace && mentionsPermission)
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

	if (response.code === 'ok' || response.code === 0) {
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
