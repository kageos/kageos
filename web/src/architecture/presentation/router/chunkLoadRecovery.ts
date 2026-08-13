const CHUNK_LOAD_ERROR_PATTERNS = [
  /failed to fetch dynamically imported module/i,
  /importing a module script failed/i,
  /error loading dynamically imported module/i,
  /chunkloaderror/i,
  /loading chunk .+ failed/i,
]

export const CHUNK_LOAD_RECOVERY_KEY = 'kageos:chunk-load-recovery'

export interface ChunkLoadRecoveryRuntime {
  readMarker: () => string | null
  writeMarker: (marker: string) => void
  clearMarker: () => void
  reload: () => void
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }
  return typeof error === 'string' ? error : ''
}

export function isChunkLoadError(error: unknown): boolean {
  const message = errorMessage(error)
  return CHUNK_LOAD_ERROR_PATTERNS.some((pattern) => pattern.test(message))
}

export function createChunkLoadRecovery(runtime: ChunkLoadRecoveryRuntime) {
  return {
    recover(error: unknown, target: string): boolean {
      if (!isChunkLoadError(error)) {
        return false
      }

      try {
        if (runtime.readMarker() === target) {
          return false
        }
        runtime.writeMarker(target)
        runtime.reload()
        return true
      } catch {
        return false
      }
    },

    clear(): void {
      try {
        runtime.clearMarker()
      } catch {
        // sessionStorage can be unavailable in restrictive browser modes.
      }
    },
  }
}

export function createBrowserChunkLoadRecovery() {
  return createChunkLoadRecovery({
    readMarker: () => window.sessionStorage.getItem(CHUNK_LOAD_RECOVERY_KEY),
    writeMarker: (marker) => window.sessionStorage.setItem(CHUNK_LOAD_RECOVERY_KEY, marker),
    clearMarker: () => window.sessionStorage.removeItem(CHUNK_LOAD_RECOVERY_KEY),
    reload: () => window.location.reload(),
  })
}
