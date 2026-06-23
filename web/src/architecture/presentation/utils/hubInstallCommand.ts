export interface HubInstallInput {
  bundleUrl: string
  installKey?: string
  displaySource: string
}

const DEFAULT_HUB_REGISTRY = 'api.kageos.com'
const DEFAULT_HUB_TAG = 'latest'

export function tokenizeInstallCommand(command: string): string[] {
  const tokens: string[] = []
  let current = ''
  let quote: '"' | "'" | '' = ''
  let escaping = false

  for (const char of command) {
    if (escaping) {
      current += char
      escaping = false
      continue
    }
    if (char === '\\') {
      escaping = true
      continue
    }
    if (quote) {
      if (char === quote) {
        quote = ''
      } else {
        current += char
      }
      continue
    }
    if (char === '"' || char === "'") {
      quote = char
      continue
    }
    if (/\s/.test(char)) {
      if (current) {
        tokens.push(current)
        current = ''
      }
      continue
    }
    current += char
  }
  if (current) {
    tokens.push(current)
  }
  return tokens
}

export function parseHubInstallInput(input: string, explicitInstallKey = ''): HubInstallInput | null {
  const trimmed = input.trim()
  if (!trimmed) return null

  const tokens = tokenizeInstallCommand(trimmed)
  if (tokens.length >= 3 && tokens[0] === 'kageos' && tokens[1] === 'install') {
    const source = tokens[2] || ''
    const installKey = parseInstallKey(tokens, 3, explicitInstallKey)
    return normalizeHubInstallSource(source, installKey)
  }

  const urlMatch = trimmed.match(/https?:\/\/[^\s"'<>]+/i)
  if (urlMatch?.[0]) {
    return normalizeHubInstallSource(urlMatch[0].replace(/[),.;]+$/g, ''), explicitInstallKey)
  }

  if (!/\s/.test(trimmed)) {
    return normalizeHubInstallSource(trimmed, explicitInstallKey)
  }
  return null
}

function parseInstallKey(tokens: string[], startIndex: number, explicitInstallKey: string): string {
  let installKey = explicitInstallKey.trim()
  for (let index = startIndex; index < tokens.length; index += 1) {
    const token = tokens[index] || ''
    if (token === '--key' || token === '--install-key') {
      installKey = tokens[index + 1] || installKey
      index += 1
    } else if (token.startsWith('--key=')) {
      installKey = token.slice('--key='.length)
    } else if (token.startsWith('--install-key=')) {
      installKey = token.slice('--install-key='.length)
    }
  }
  return installKey.trim()
}

function normalizeHubInstallSource(source: string, explicitInstallKey: string): HubInstallInput | null {
  source = source.trim().replace(/[),.;]+$/g, '')
  if (!source) return null

  if (/^https?:\/\//i.test(source)) {
    return normalizeURLSource(source, explicitInstallKey)
  }
  return normalizeDockerLikeSource(source, explicitInstallKey)
}

function normalizeURLSource(source: string, explicitInstallKey: string): HubInstallInput | null {
  try {
    const url = new URL(source)
    const queryInstallKey = readInstallKeyFromURL(url)
    stripInstallKeyQuery(url)

    const dockerLikePath = normalizeDockerLikePath(url.pathname)
    if (dockerLikePath) {
      return {
        bundleUrl: buildBundleURL(url.origin, dockerLikePath.owner, dockerLikePath.app, dockerLikePath.ref),
        installKey: firstNonEmpty(explicitInstallKey, queryInstallKey),
        displaySource: `${dockerLikePath.owner}/${dockerLikePath.app}:${dockerLikePath.ref}`
      }
    }

    return {
      bundleUrl: url.toString(),
      installKey: firstNonEmpty(explicitInstallKey, queryInstallKey),
      displaySource: url.toString()
    }
  } catch {
    return null
  }
}

function normalizeDockerLikeSource(source: string, explicitInstallKey: string): HubInstallInput | null {
  const parts = source.split('/').filter(Boolean)
  if (parts.length < 2 || parts.length > 3) return null

  let registry = DEFAULT_HUB_REGISTRY
  let ownerIndex = 0
  const firstPart = parts[0]
  if (!firstPart) return null
  if (looksLikeRegistry(firstPart)) {
    registry = firstPart
    ownerIndex = 1
  }
  if (parts.length - ownerIndex !== 2) return null

  const owner = parts[ownerIndex]
  const appPart = parts[ownerIndex + 1]
  if (!owner || !appPart) return null
  const appRef = splitAppRef(appPart)
  if (!isValidOwnerSegment(owner) || !appRef || !isValidRefSegment(appRef.app) || !isValidRefSegment(appRef.ref)) {
    return null
  }

  return {
    bundleUrl: buildBundleURL(registryOrigin(registry), owner, appRef.app, appRef.ref),
    installKey: explicitInstallKey.trim() || undefined,
    displaySource: `${owner}/${appRef.app}:${appRef.ref}`
  }
}

function normalizeDockerLikePath(pathname: string): { owner: string; app: string; ref: string } | null {
  const parts = pathname.split('/').filter(Boolean)
  if (parts.length !== 2) return null

  const owner = parts[0]
  const appPart = parts[1]
  if (!owner || !appPart) return null
  const appRef = splitAppRef(appPart)
  if (!isValidOwnerSegment(owner) || !appRef || !isValidRefSegment(appRef.app) || !isValidRefSegment(appRef.ref)) {
    return null
  }
  return { owner, app: appRef.app, ref: appRef.ref }
}

function splitAppRef(value: string): { app: string; ref: string } | null {
  const colonIndex = value.lastIndexOf(':')
  if (colonIndex < 0) {
    return { app: value, ref: DEFAULT_HUB_TAG }
  }
  const app = value.slice(0, colonIndex)
  const ref = value.slice(colonIndex + 1)
  if (!app || !ref) return null
  return { app, ref }
}

function buildBundleURL(origin: string, owner: string, app: string, ref: string): string {
  return `${origin.replace(/\/+$/, '')}/api/v1/applications/${encodeURIComponent(owner)}/${encodeURIComponent(app)}/${encodeURIComponent(ref)}/bundle`
}

function registryOrigin(registry: string): string {
  const scheme = isLocalRegistry(registry) ? 'http' : 'https'
  return `${scheme}://${registry}`
}

function looksLikeRegistry(value: string): boolean {
  return value === 'localhost' || value.includes('.') || value.includes(':') || value.startsWith('[')
}

function isLocalRegistry(value: string): boolean {
  const bracketHost = value.match(/^\[([^\]]+)\]/)?.[1]
  const host = bracketHost || value.split(':')[0]
  return host === 'localhost' || host === '127.0.0.1' || host === '::1'
}

function isValidRefSegment(value: string): boolean {
  return /^[A-Za-z0-9][A-Za-z0-9_.-]*$/.test(value)
}

function isValidOwnerSegment(value: string): boolean {
  return /^[A-Za-z0-9][A-Za-z0-9_.@-]*$/.test(value)
}

function readInstallKeyFromURL(url: URL): string {
  return url.searchParams.get('install_key') || url.searchParams.get('key') || ''
}

function stripInstallKeyQuery(url: URL) {
  url.searchParams.delete('install_key')
  url.searchParams.delete('key')
}

function firstNonEmpty(...values: string[]): string | undefined {
  for (const value of values) {
    const trimmed = value.trim()
    if (trimmed) return trimmed
  }
  return undefined
}
