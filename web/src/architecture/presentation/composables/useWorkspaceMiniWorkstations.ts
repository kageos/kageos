import { nextTick, ref, watch, type ComputedRef } from 'vue'
import type { RouteLocationNormalizedLoaded, Router } from 'vue-router'

export interface MiniWsInstance {
  id: string
  fullCodePath: string
  dirName: string
  initialSessionId: string
  visible: boolean
  offset: number
  initialPosition?: 'center'
  initialExpanded?: boolean
  initialMaximized?: boolean
}

export interface WorkstationContext {
  fullCodePath: string
  dirName: string
}

export interface UseWorkspaceMiniWorkstationsOptions {
  route: RouteLocationNormalizedLoaded
  router: Router
  workstationContext: ComputedRef<WorkstationContext | null>
  buildWorkspacePath: (fullCodePath: string) => string
  resolvePathName?: (fullCodePath: string) => string | undefined
}

interface MiniWsRouteState {
  sessionId?: string
  ctx?: WorkstationContext
  expanded?: boolean
  maximized?: boolean
}

export function useWorkspaceMiniWorkstations(options: UseWorkspaceMiniWorkstationsOptions) {
  const { route, router, workstationContext, buildWorkspacePath, resolvePathName } = options

  const miniWsList = ref<MiniWsInstance[]>([])
  let miniIdCounter = 0
  let pendingExplicitOpenPath = ''

  function normalizeFullCodePath(fullCodePath: string) {
    return (fullCodePath || '').trim().replace(/\/+$/g, '')
  }

  function resolveDirName(fullCodePath: string, overrideName?: string) {
    const normalizedPath = normalizeFullCodePath(fullCodePath)
    const ctx = workstationContext.value
    const ctxName = ctx && normalizeFullCodePath(ctx.fullCodePath) === normalizedPath ? ctx.dirName : ''
    return resolvePathName?.(normalizedPath)
      || overrideName
      || ctxName
      || normalizedPath.split('/').filter(Boolean).pop()
      || '工作台'
  }

  function findPreferredMiniForPath(fullCodePath: string, sessionId = '') {
    const normalizedPath = normalizeFullCodePath(fullCodePath)
    const normalizedSessionId = sessionId.trim()
    const candidates = [...miniWsList.value]
      .reverse()
      .filter((mini: MiniWsInstance) => normalizeFullCodePath(mini.fullCodePath) === normalizedPath)

    if (normalizedSessionId) {
      return candidates.find((mini: MiniWsInstance) => mini.initialSessionId === normalizedSessionId)
    }

    return candidates.find((mini: MiniWsInstance) => !!mini.initialSessionId) || candidates[0]
  }

  function upsertPrimaryMiniWs(
    fullCodePath: string,
    dirName: string,
    initialSessionId = '',
    initialMaximized?: boolean,
    initialExpanded?: boolean,
    preferMatch?: (mini: MiniWsInstance) => boolean
  ): MiniWsInstance | undefined {
    const normalizedSessionId = initialSessionId.trim()
    let existingIndex = preferMatch ? miniWsList.value.findIndex(preferMatch) : -1
    if (existingIndex === -1) {
      existingIndex = miniWsList.value.findIndex((mini: MiniWsInstance) =>
        mini.fullCodePath === fullCodePath && mini.initialSessionId === normalizedSessionId
      )
    }

    if (existingIndex !== -1) {
      const existing = miniWsList.value[existingIndex]
      if (!existing) return
      const contextChanged = existing.fullCodePath !== fullCodePath
        || existing.dirName !== dirName
        || existing.initialSessionId !== normalizedSessionId
      const nextInitialMaximized = initialMaximized ?? existing.initialMaximized ?? false
      const nextInitialExpanded = initialExpanded ?? existing.initialExpanded ?? true
      const nextMini: MiniWsInstance = {
        ...existing,
        id: contextChanged ? String(++miniIdCounter) : existing.id,
        fullCodePath,
        dirName,
        initialSessionId: normalizedSessionId,
        visible: nextInitialExpanded !== false,
        offset: 0,
        initialPosition: nextInitialMaximized ? undefined : 'center',
        initialExpanded: nextInitialExpanded,
        initialMaximized: nextInitialMaximized,
      }
      miniWsList.value = miniWsList.value.map((mini: MiniWsInstance, index: number) =>
        index === existingIndex ? nextMini : { ...mini, visible: false }
      )
      return nextMini
    }

    const nextInitialMaximized = initialMaximized ?? true
    const nextInitialExpanded = initialExpanded ?? true
    const nextMini: MiniWsInstance = {
      id: String(++miniIdCounter),
      fullCodePath,
      dirName,
      initialSessionId: normalizedSessionId,
      visible: nextInitialExpanded !== false,
      offset: 0,
      initialPosition: nextInitialMaximized ? undefined : 'center',
      initialExpanded: nextInitialExpanded,
      initialMaximized: nextInitialMaximized,
    }
    miniWsList.value = [
      ...miniWsList.value.map((mini: MiniWsInstance) => ({ ...mini, visible: false })),
      nextMini
    ]
    return nextMini
  }

  function openAmbientMiniWs(overridePath?: string, overrideName?: string) {
    const ctx = workstationContext.value
    const fullCodePath = overridePath || ctx?.fullCodePath
    if (!fullCodePath) return
    if (pendingExplicitOpenPath === fullCodePath) return

    const dirName = resolveDirName(fullCodePath, overrideName)
    const visibleIndex = miniWsList.value.findIndex((mini: MiniWsInstance) => mini.visible)
    const visibleMini = visibleIndex >= 0 ? miniWsList.value[visibleIndex] : null
    if (visibleMini?.fullCodePath === fullCodePath) {
      if (visibleMini.dirName !== dirName) {
        miniWsList.value = miniWsList.value.map((mini: MiniWsInstance, index: number) =>
          index === visibleIndex ? { ...mini, dirName } : mini
        )
      }
      return
    }
    if (visibleMini) {
      return
    }

    const existingIndex = miniWsList.value.findIndex((mini: MiniWsInstance) =>
      mini.fullCodePath === fullCodePath && !mini.initialSessionId
    )
    const ambientMini: MiniWsInstance = {
      ...(existingIndex >= 0 ? miniWsList.value[existingIndex]! : {}),
      id: existingIndex >= 0 ? miniWsList.value[existingIndex]!.id : String(++miniIdCounter),
      fullCodePath,
      dirName,
      initialSessionId: '',
      visible: false,
      offset: 0,
      initialPosition: 'center',
      initialExpanded: false,
      initialMaximized: false,
    }
    if (existingIndex >= 0) {
      miniWsList.value = miniWsList.value.map((mini: MiniWsInstance, index: number) =>
        index === existingIndex ? ambientMini : mini
      )
      return
    }
    miniWsList.value = [
      ...miniWsList.value,
      ambientMini
    ]
  }

  function openNewMiniWs(
    initialSessionId?: string,
    overridePath?: string,
    overrideName?: string,
    initialMaximized?: boolean,
    initialExpanded?: boolean,
    forceNew = false
  ) {
    const ctx = workstationContext.value
    const fullCodePath = overridePath || ctx?.fullCodePath
    if (!fullCodePath) return

    const dirName = resolveDirName(fullCodePath, overrideName)
    const normalizedSessionId = (initialSessionId || '').trim()
    if (normalizedSessionId) {
      return upsertPrimaryMiniWs(
        fullCodePath,
        dirName,
        normalizedSessionId,
        initialMaximized,
        initialExpanded,
        (mini: MiniWsInstance) => mini.fullCodePath === fullCodePath && mini.initialSessionId === normalizedSessionId
      )
    } else if (!forceNew) {
      const existingForPath = findPreferredMiniForPath(fullCodePath)
      if (existingForPath?.initialSessionId) {
        return upsertPrimaryMiniWs(
          fullCodePath,
          existingForPath.dirName || dirName,
          existingForPath.initialSessionId,
          initialMaximized,
          initialExpanded,
          (mini: MiniWsInstance) => mini.id === existingForPath.id
        )
      }
      return upsertPrimaryMiniWs(fullCodePath, dirName, '', initialMaximized, initialExpanded)
    }
    return upsertPrimaryMiniWs(fullCodePath, dirName, '', initialMaximized, initialExpanded)
  }

  function normalizeRouteBool(value: unknown, defaultValue: boolean) {
    const raw = Array.isArray(value) ? value[0] : value
    if (raw === undefined || raw === null) return defaultValue
    const normalized = String(raw).trim().toLowerCase()
    if (['1', 'true', 'yes', 'open', 'expanded', 'maximized'].includes(normalized)) return true
    if (['0', 'false', 'no', 'closed', 'collapsed', 'normal'].includes(normalized)) return false
    return defaultValue
  }

  function syncMiniWsQueryParam(open: boolean, state: MiniWsRouteState = {}) {
    const query = { ...route.query }
    if (open && state.expanded !== false) {
      // mini 工作台状态只给前端/平台使用，必须用 `_` key，
      // 不能占用 sdk-app 业务参数名。
      query._mws = 'open'
      if (state.sessionId) {
        query._mws_sid = state.sessionId
      } else {
        delete query._mws_sid
      }
      const ctx = state.ctx || workstationContext.value
      if (ctx) {
        query._mws_path = ctx.fullCodePath
        query._mws_name = resolveDirName(ctx.fullCodePath, ctx.dirName)
      }
      query._mws_expanded = '1'
      query._mws_maximized = state.maximized ? '1' : '0'
    } else {
      delete query._mws
      delete query._mws_sid
      delete query._mws_path
      delete query._mws_name
      delete query._mws_expanded
      delete query._mws_maximized
    }
    router.replace({ path: route.path, query })
  }

  function handleMiniMinimize(id: string) {
    const mini = miniWsList.value.find((item: MiniWsInstance) => item.id === id)
    if (mini) mini.visible = false
    syncMiniWsQueryParam(false)
  }

  function hideVisibleMiniWs() {
    const visibleMini = miniWsList.value.find((mini: MiniWsInstance) => mini.visible)
    if (!visibleMini) return false
    handleMiniMinimize(visibleMini.id)
    return true
  }

  function handleMiniRemove(id: string) {
    miniWsList.value = miniWsList.value.filter((item: MiniWsInstance) => item.id !== id)
    syncMiniWsQueryParam(false)
  }

  function handleMiniMaximizeChange(id: string, payload: { maximized: boolean; sessionId?: string }) {
    const mini = miniWsList.value.find((item: MiniWsInstance) => item.id === id)
    if (mini) {
      mini.initialMaximized = payload.maximized
    }
    const sessionId = payload.sessionId !== undefined ? payload.sessionId : mini?.initialSessionId
    syncMiniWsQueryParam(true, {
      sessionId,
      ctx: mini ? { fullCodePath: mini.fullCodePath, dirName: mini.dirName } : undefined,
      expanded: mini?.initialExpanded !== false,
      maximized: payload.maximized,
    })
  }

  function handleMiniExpandedChange(id: string, payload: { expanded: boolean; sessionId?: string }) {
    const mini = miniWsList.value.find((item: MiniWsInstance) => item.id === id)
    if (mini) {
      mini.initialExpanded = payload.expanded
      if (!payload.expanded) {
        mini.initialMaximized = false
        mini.visible = false
      }
    }
    const sessionId = payload.sessionId !== undefined ? payload.sessionId : mini?.initialSessionId
    syncMiniWsQueryParam(true, {
      sessionId,
      ctx: mini ? { fullCodePath: mini.fullCodePath, dirName: mini.dirName } : undefined,
      expanded: payload.expanded,
      maximized: !!mini?.initialMaximized,
    })
  }

  function handleMiniTaskStarted(id: string, sessionId: string) {
    const normalizedSessionId = sessionId.trim()
    if (!normalizedSessionId) return

    miniWsList.value = miniWsList.value.map((mini: MiniWsInstance) => {
      if (mini.id !== id) return mini
      return {
        ...mini,
        initialSessionId: normalizedSessionId,
        initialExpanded: true,
        initialMaximized: true,
      }
    })
    const updatedMini = miniWsList.value.find((mini: MiniWsInstance) => mini.id === id)
    if (updatedMini?.visible) {
      syncMiniWsQueryParam(true, {
        sessionId: normalizedSessionId,
        ctx: { fullCodePath: updatedMini.fullCodePath, dirName: updatedMini.dirName },
        expanded: updatedMini.initialExpanded !== false,
        maximized: !!updatedMini.initialMaximized,
      })
    }
  }

  function restoreMiniWorkstation(options?: {
    fullCodePath?: string
    dirName?: string
    sessionId?: string
    initialExpanded?: boolean
    initialMaximized?: boolean
  }) {
    const restore = (fullCodePath: string, dirName: string) => {
      openNewMiniWs(
        options?.sessionId || undefined,
        fullCodePath,
        dirName,
        !!options?.initialMaximized,
        options?.initialExpanded !== false
      )
    }

    if (options?.fullCodePath) {
      const dirName = resolveDirName(options.fullCodePath, options.dirName)
      nextTick(() => restore(options.fullCodePath!, dirName))
      return
    }

    const stopRestore = watch(workstationContext, (ctx: WorkstationContext | null) => {
      if (ctx?.fullCodePath) {
        restore(ctx.fullCodePath, ctx.dirName)
        stopRestore()
      }
    }, { immediate: true })

    setTimeout(() => stopRestore(), 10000)
  }

  function handleWorkspaceOpenWorkstation(payload: {
    full_code_path?: string
    session_id?: string
    directory_name?: string
    initial_maximized?: boolean
    open_as_mini?: boolean
    force_new?: boolean
  }) {
    const fullCodePath = (payload?.full_code_path || '').trim()
    if (!fullCodePath) return

    const targetPath = buildWorkspacePath(fullCodePath)
    const normalizedSessionId = (payload.session_id || '').trim()
    const existingMini = findPreferredMiniForPath(fullCodePath, normalizedSessionId)
    const dirName = payload.directory_name || existingMini?.dirName || resolveDirName(fullCodePath)
    const openMini = () => {
      const initialMaximized = payload.initial_maximized === undefined
        ? true
        : !!payload.initial_maximized
      const openedMini = openNewMiniWs(
        payload.session_id || undefined,
        fullCodePath,
        dirName,
        initialMaximized,
        true,
        !!payload.force_new
      )
      const nextSessionId = payload.session_id || openedMini?.initialSessionId || undefined
      syncMiniWsQueryParam(true, {
        sessionId: nextSessionId,
        ctx: { fullCodePath, dirName },
        expanded: true,
        maximized: !!openedMini?.initialMaximized,
      })
    }

    if (route.path !== targetPath) {
      pendingExplicitOpenPath = fullCodePath
      router.push({ path: targetPath, query: {} }).then(() => {
        nextTick(() => openMini())
      }).finally(() => {
        pendingExplicitOpenPath = ''
      })
    } else {
      nextTick(() => openMini())
    }
  }

  function initializeFromRoute() {
    if (route.query._mws === 'open') {
      const mwsSid = typeof route.query._mws_sid === 'string' ? route.query._mws_sid : ''
      const mwsPath = typeof route.query._mws_path === 'string' ? route.query._mws_path : ''
      const mwsName = typeof route.query._mws_name === 'string' ? route.query._mws_name : ''
      const initialExpanded = normalizeRouteBool(route.query._mws_expanded, true)
      if (!initialExpanded) {
        syncMiniWsQueryParam(false)
        return
      }
      restoreMiniWorkstation({
        fullCodePath: mwsPath || undefined,
        dirName: mwsName || undefined,
        sessionId: mwsSid || undefined,
        initialExpanded,
        initialMaximized: normalizeRouteBool(route.query._mws_maximized, true),
      })
    }
  }

  return {
    miniWsList,
    openAmbientMiniWs,
    openNewMiniWs,
    handleMiniMinimize,
    hideVisibleMiniWs,
    handleMiniRemove,
    handleMiniMaximizeChange,
    handleMiniExpandedChange,
    handleMiniTaskStarted,
    handleWorkspaceOpenWorkstation,
    initializeFromRoute,
  }
}
