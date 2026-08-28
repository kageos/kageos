export interface NetworkUnavailableContext {
  hostname?: string
  online?: boolean
}

function getRuntimeHostname(): string {
  return typeof window === 'undefined' ? '' : window.location.hostname
}

function getRuntimeOnlineState(): boolean {
  return typeof navigator === 'undefined' ? true : navigator.onLine
}

export function isLocalKageOSHostname(hostname: string): boolean {
  const normalized = hostname.trim().toLowerCase()
  return normalized === 'localhost'
    || normalized.endsWith('.localhost')
    || normalized === '127.0.0.1'
    || normalized === '::1'
    || normalized === '[::1]'
}

export function getNetworkUnavailableMessage(context: NetworkUnavailableContext = {}): string {
  const online = context.online ?? getRuntimeOnlineState()
  if (!online) {
    return '网络已断开，请检查网络连接'
  }

  const hostname = context.hostname ?? getRuntimeHostname()
  if (isLocalKageOSHostname(hostname)) {
    return '无法连接本地 kageos 服务，请先启动本地服务后重试'
  }

  return '无法连接 kageos 服务，请检查网络连接或稍后重试'
}
