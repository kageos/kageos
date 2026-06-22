export const PUBLIC_CONNECTOR_PROVIDERS = ['github', 'notion'] as const

const PUBLIC_CONNECTOR_PROVIDER_SET = new Set<string>(PUBLIC_CONNECTOR_PROVIDERS)

export function normalizeConnectorProvider(provider: string | null | undefined): string {
  return String(provider || '').trim().toLowerCase()
}

export function isPublicConnectorProvider(provider: string | null | undefined): boolean {
  return PUBLIC_CONNECTOR_PROVIDER_SET.has(normalizeConnectorProvider(provider))
}
