import axios from 'axios'

export const queryParamsSerializer = {
  indexes: null
} as const

export function serializeQueryParams(params: Record<string, unknown>): string {
  const uri = axios.getUri({
    url: '/',
    params,
    paramsSerializer: queryParamsSerializer
  })

  return uri.split('?', 2)[1] || ''
}
