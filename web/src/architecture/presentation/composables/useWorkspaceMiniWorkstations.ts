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
}

export function useWorkspaceMiniWorkstations(options: UseWorkspaceMiniWorkstationsOptions) {
  const { route, router, workstationContext, buildWorkspacePath } = options

  const miniWsList = ref<MiniWsInstance[]>([])
  let miniIdCounter = 0

  function resolveDirName(fullCodePath: string, overrideName?: string) {
    const ctx = workstationContext.value
    return overrideName || ctx?.dirName || fullCodePath.split('/').filter(Boolean).pop() || '工作台'
  }

  function upsertPrimaryMiniWs(
    fullCodePath: string,
    dirName: string,
    initialSessionId = '',
    initialMaximized = false,
    preferMatch?: (mini: MiniWsInstance) => boolean
  ) {
    const normalizedSessionId = initialSessionId.trim()
    let existingIndex = preferMatch ? miniWsList.value.findIndex(preferMatch) : -1
    if (existingIndex === -1) {
      existingIndex = miniWsList.value.findIndex((mini: MiniWsInstance) => mini.visible)
    }
    if (existingIndex === -1 && miniWsList.value.length > 0) {
      existingIndex = 0
    }

    if (existingIndex !== -1) {
      const existing = miniWsList.value[existingIndex]
      if (!existing) return
      const contextChanged = existing.fullCodePath !== fullCodePath
        || existing.dirName !== dirName
        || existing.initialSessionId !== normalizedSessionId
      const nextMini: MiniWsInstance = {
        ...existing,
        id: contextChanged ? String(++miniIdCounter) : existing.id,
        fullCodePath,
        dirName,
        initialSessionId: normalizedSessionId,
        visible: true,
        offset: 0,
        initialPosition: initialMaximized ? undefined : 'center',
        initialMaximized,
      }
      miniWsList.value = [nextMini]
      return
    }

    miniWsList.value.push({
      id: String(++miniIdCounter),
      fullCodePath,
      dirName,
      initialSessionId: normalizedSessionId,
      visible: true,
      offset: 0,
      initialPosition: initialMaximized ? undefined : 'center',
      initialMaximized,
    })
  }

  function openAmbientMiniWs(overridePath?: string, overrideName?: string) {
    const ctx = workstationContext.value
    const fullCodePath = overridePath || ctx?.fullCodePath
    if (!fullCodePath) return

    upsertPrimaryMiniWs(fullCodePath, resolveDirName(fullCodePath, overrideName))
  }

  function openNewMiniWs(initialSessionId?: string, overridePath?: string, overrideName?: string, initialMaximized = false) {
    const ctx = workstationContext.value
    const fullCodePath = overridePath || ctx?.fullCodePath
    if (!fullCodePath) return

    const dirName = resolveDirName(fullCodePath, overrideName)
    const normalizedSessionId = (initialSessionId || '').trim()
    if (normalizedSessionId) {
      upsertPrimaryMiniWs(
        fullCodePath,
        dirName,
        normalizedSessionId,
        initialMaximized,
        (mini: MiniWsInstance) => mini.fullCodePath === fullCodePath && mini.initialSessionId === normalizedSessionId
      )
      return
    } else {
      upsertPrimaryMiniWs(fullCodePath, dirName, '', initialMaximized)
      return
    }
  }

  function syncMiniWsQueryParam(open: boolean, sid?: string) {
    const query = { ...route.query }
    if (open) {
      // mini 工作台状态只给前端/平台使用，必须用 `_` key，
      // 不能占用 sdk-app 业务参数名。
      query._mws = 'open'
      if (sid) {
        query._mws_sid = sid
      } else {
        delete query._mws_sid
      }
      const ctx = workstationContext.value
      if (ctx) {
        query._mws_path = ctx.fullCodePath
        query._mws_name = ctx.dirName
      }
    } else {
      delete query._mws
      delete query._mws_sid
      delete query._mws_path
      delete query._mws_name
    }
    router.replace({ path: route.path, query })
  }

  function handleMiniMinimize(id: string) {
    const mini = miniWsList.value.find((item: MiniWsInstance) => item.id === id)
    if (mini) mini.visible = false
    syncMiniWsQueryParam(false)
  }

  function handleMiniRemove(id: string) {
    miniWsList.value = miniWsList.value.filter((item: MiniWsInstance) => item.id !== id)
    syncMiniWsQueryParam(false)
  }

  function handleMiniMaximizeChange(payload: { maximized: boolean; sessionId?: string }) {
    if (payload.maximized) {
      syncMiniWsQueryParam(true, payload.sessionId)
    } else {
      syncMiniWsQueryParam(false)
    }
  }

  function openMiniWsForTask(fullCodePath: string, sessionId: string) {
    const dirName = fullCodePath.split('/').filter(Boolean).pop() || '工作台'
    upsertPrimaryMiniWs(
      fullCodePath,
      dirName,
      sessionId,
      true,
      (mini: MiniWsInstance) => mini.fullCodePath === fullCodePath && mini.initialSessionId === sessionId
    )
  }

  function restoreMiniWorkstation(options?: {
    fullCodePath?: string
    dirName?: string
    sessionId?: string
    initialMaximized?: boolean
  }) {
    const restore = (fullCodePath: string, dirName: string) => {
      openNewMiniWs(
        options?.sessionId || undefined,
        fullCodePath,
        dirName,
        !!options?.initialMaximized
      )
    }

    if (options?.fullCodePath) {
      const dirName = options.dirName || options.fullCodePath.split('/').filter(Boolean).pop() || '工作台'
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

  function handleWorkspaceOpenWorkstation(payload: { full_code_path?: string; session_id?: string; open_as_mini?: boolean }) {
    const fullCodePath = (payload?.full_code_path || '').trim()
    if (!fullCodePath) return

    const targetPath = buildWorkspacePath(fullCodePath)
    const dirName = fullCodePath.split('/').filter(Boolean).pop() || '工作台'
    const openMini = () => {
      if (payload.open_as_mini) {
        openMiniWsForTask(fullCodePath, payload.session_id || '')
        return
      }
      openNewMiniWs(payload.session_id || undefined, fullCodePath, dirName)
    }

    if (route.path !== targetPath) {
      router.push({ path: targetPath, query: { ...route.query } }).then(() => {
        nextTick(() => openMini())
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
      restoreMiniWorkstation({
        fullCodePath: mwsPath || undefined,
        dirName: mwsName || undefined,
        sessionId: mwsSid || undefined,
        initialMaximized: true,
      })
    }
  }

  return {
    miniWsList,
    openAmbientMiniWs,
    openNewMiniWs,
    handleMiniMinimize,
    handleMiniRemove,
    handleMiniMaximizeChange,
    handleWorkspaceOpenWorkstation,
    initializeFromRoute,
  }
}
