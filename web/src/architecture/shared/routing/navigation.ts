import type { RouteLocationRaw, Router } from 'vue-router'

let appRouter: Router | null = null

export function setAppRouter(router: Router): void {
  appRouter = router
}

export function getCurrentRoutePath(): string {
  return appRouter?.currentRoute.value.path ?? window.location.pathname
}

export function getCurrentRouteFullPath(): string {
  return appRouter?.currentRoute.value.fullPath ?? `${window.location.pathname}${window.location.search}`
}

export async function navigateTo(target: RouteLocationRaw): Promise<void> {
  if (appRouter) {
    await appRouter.push(target)
    return
  }

  if (typeof target === 'string') {
    window.location.assign(target)
    return
  }

  if ('path' in target && typeof target.path === 'string') {
    const searchParams = new URLSearchParams()
    const query = target.query ?? {}

    Object.entries(query).forEach(([key, value]) => {
      if (value === undefined || value === null) {
        return
      }

      if (Array.isArray(value)) {
        value.forEach((entry) => {
          if (entry !== undefined && entry !== null) {
            searchParams.append(key, String(entry))
          }
        })
        return
      }

      searchParams.set(key, String(value))
    })

    const queryString = searchParams.toString()
    window.location.assign(`${target.path}${queryString ? `?${queryString}` : ''}`)
  }
}
