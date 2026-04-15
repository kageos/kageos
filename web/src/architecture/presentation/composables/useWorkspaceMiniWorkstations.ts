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

  function openNewMiniWs(initialSessionId?: string, overridePath?: string, overrideName?: string, initialMaximized = false) {
    const ctx = workstationContext.value
    const fullCodePath = overridePath || ctx?.fullCodePath
    if (!fullCodePath) return

    const dirName = overrideName || ctx?.dirName || fullCodePath.split('/').filter(Boolean).pop() || '工作台'
    const existing = miniWsList.value.find(
      (mini: MiniWsInstance) => mini.fullCodePath === fullCodePath && mini.initialSessionId === (initialSessionId || '')
    )
    if (existing) {
      existing.visible = true
      return
    }

    const offset = miniWsList.value.filter((mini: MiniWsInstance) => mini.visible).length * 40
    miniWsList.value.push({
      id: String(++miniIdCounter),
      fullCodePath,
      dirName,
      initialSessionId: initialSessionId || '',
      visible: true,
      offset: initialMaximized ? 0 : offset,
      initialPosition: initialMaximized ? undefined : 'center',
      initialMaximized,
    })
  }

  function syncMiniWsQueryParam(open: boolean, sid?: string) {
    const query = { ...route.query }
    if (open) {
      query.mws = 'open'
      if (sid) {
        query.mws_sid = sid
      } else {
        delete query.mws_sid
      }
      const ctx = workstationContext.value
      if (ctx) {
        query.mws_path = ctx.fullCodePath
        query.mws_name = ctx.dirName
      }
    } else {
      delete query.mws
      delete query.mws_sid
      delete query.mws_path
      delete query.mws_name
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
    const existing = miniWsList.value.find(
      (mini: MiniWsInstance) => mini.fullCodePath === fullCodePath && mini.initialSessionId === sessionId
    )
    if (existing) {
      existing.visible = true
      return
    }

    miniWsList.value.push({
      id: String(++miniIdCounter),
      fullCodePath,
      dirName,
      initialSessionId: sessionId,
      visible: true,
      offset: 0,
      initialPosition: 'center',
      initialMaximized: true,
    })
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
    if (route.query.mws === 'open') {
      const mwsSid = typeof route.query.mws_sid === 'string' ? route.query.mws_sid : ''
      const mwsPath = typeof route.query.mws_path === 'string' ? route.query.mws_path : ''
      const mwsName = typeof route.query.mws_name === 'string' ? route.query.mws_name : ''
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
    openNewMiniWs,
    handleMiniMinimize,
    handleMiniRemove,
    handleMiniMaximizeChange,
    handleWorkspaceOpenWorkstation,
    initializeFromRoute,
  }
}
