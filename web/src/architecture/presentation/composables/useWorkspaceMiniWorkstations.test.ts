import { computed, nextTick, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useWorkspaceMiniWorkstations, type WorkstationContext } from './useWorkspaceMiniWorkstations'

function createHarness(
  initialContext: WorkstationContext | null = null,
  options: { resolvePathName?: (fullCodePath: string) => string | undefined } = {}
) {
  const route = {
    path: '/workspace/current',
    query: {} as Record<string, any>
  }
  const context = ref<WorkstationContext | null>(initialContext)
  const router = {
    replace: vi.fn((location: { path: string; query?: Record<string, any> }) => {
      route.path = location.path
      route.query = location.query || {}
      return Promise.resolve()
    }),
    push: vi.fn((location: { path: string; query?: Record<string, any> }) => {
      route.path = location.path
      route.query = location.query || {}
      return Promise.resolve()
    })
  }

  const api = useWorkspaceMiniWorkstations({
    route: route as any,
    router: router as any,
    workstationContext: computed(() => context.value),
    buildWorkspacePath: (fullCodePath: string) => `/workspace${fullCodePath}`,
    resolvePathName: options.resolvePathName
  })

  return { api, context, route, router }
}

function openMiniWsQuery(input: {
  sessionId?: string
  fullCodePath: string
  name: string
  maximized?: '0' | '1'
}) {
  return {
    _open: 'session',
    _focus: 'workspace_session',
    ...(input.sessionId ? { _session_id: input.sessionId } : {}),
    _source_path: input.fullCodePath,
    _mws: 'open',
    ...(input.sessionId ? { _mws_sid: input.sessionId } : {}),
    _mws_path: input.fullCodePath,
    _mws_name: input.name,
    _mws_expanded: '1',
    _mws_maximized: input.maximized ?? '1'
  }
}

describe('useWorkspaceMiniWorkstations', () => {
  it('keeps multiple session workstations visible at the same time', () => {
    const { api } = createHarness()

    api.openNewMiniWs('session-a', '/user/app/a', 'A')
    api.openNewMiniWs('session-b', '/user/app/b', 'B')

    expect(api.miniWsList.value).toHaveLength(2)
    expect(api.miniWsList.value.find(item => item.initialSessionId === 'session-a')?.visible).toBe(true)
    expect(api.miniWsList.value.find(item => item.initialSessionId === 'session-b')?.visible).toBe(true)

    api.openNewMiniWs('session-a', '/user/app/a', 'A')

    expect(api.miniWsList.value).toHaveLength(2)
    expect(api.miniWsList.value.find(item => item.initialSessionId === 'session-a')?.visible).toBe(true)
    expect(api.miniWsList.value.find(item => item.initialSessionId === 'session-b')?.visible).toBe(true)
  })

  it('does not replace an active session with an ambient draft for the same path', () => {
    const { api } = createHarness()

    api.openNewMiniWs('session-a', '/user/app/a', 'A')
    api.openAmbientMiniWs('/user/app/a', 'A Renamed')

    expect(api.miniWsList.value).toHaveLength(1)
    expect(api.miniWsList.value[0]).toMatchObject({
      fullCodePath: '/user/app/a',
      dirName: 'A Renamed',
      initialSessionId: 'session-a',
      visible: true
    })
  })

  it('writes maximized URL state from the selected mini workstation context', () => {
    const { api, context, router } = createHarness({ fullCodePath: '/user/app/current', dirName: 'Current' })

    api.openNewMiniWs('session-a', '/user/app/other', 'Other', false, true)
    const mini = api.miniWsList.value[0]!
    context.value = { fullCodePath: '/user/app/current', dirName: 'Current' }

    api.handleMiniMaximizeChange(mini.id, { maximized: true, sessionId: 'session-a' })

    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/current',
      query: openMiniWsQuery({ sessionId: 'session-a', fullCodePath: '/user/app/other', name: 'Other' })
    })
  })

  it('clears workstation query state and hides workstation when collapsed', () => {
    const { api, router } = createHarness()

    api.openNewMiniWs('session-a', '/user/app/a', 'A')
    const mini = api.miniWsList.value[0]!

    api.handleMiniExpandedChange(mini.id, { expanded: false, sessionId: 'session-a' })

    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/current',
      query: {}
    })
    expect(api.miniWsList.value[0]).toMatchObject({
      initialExpanded: false,
      visible: false
    })
  })

  it('hides the currently visible workstation from the shared shortcut path', () => {
    const { api, router } = createHarness()

    api.openNewMiniWs('session-a', '/user/app/a', 'A')
    api.openNewMiniWs('session-b', '/user/app/b', 'B')

    expect(api.hideVisibleMiniWs()).toBe(true)
    expect(api.miniWsList.value.find(item => item.initialSessionId === 'session-b')?.visible).toBe(false)
    expect(api.miniWsList.value.find(item => item.initialSessionId === 'session-a')?.visible).toBe(true)
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/current',
      query: {}
    })
  })

  it('reports false when shortcut hide has no visible workstation', () => {
    const { api, router } = createHarness()

    api.openAmbientMiniWs('/user/app/a', 'A')

    expect(api.hideVisibleMiniWs()).toBe(false)
    expect(router.replace).not.toHaveBeenCalled()
  })

  it('restores a minimized window and brings it to the front', () => {
    const { api } = createHarness()

    api.openNewMiniWs('session-a', '/user/app/a', 'A', false, true)
    api.openNewMiniWs('session-b', '/user/app/b', 'B', false, true)
    const first = api.miniWsList.value.find(item => item.initialSessionId === 'session-a')!
    const second = api.miniWsList.value.find(item => item.initialSessionId === 'session-b')!

    api.handleMiniMinimize(first.id)
    expect(first.visible).toBe(false)

    api.restoreMiniWs(first.id)

    expect(first.visible).toBe(true)
    expect(first.zIndex).toBeGreaterThan(second.zIndex)
  })

  it('persists a session selected inside the current workstation', () => {
    const { api, router } = createHarness()

    api.openNewMiniWs('session-a', '/user/app/a', 'A', false, true)
    const mini = api.miniWsList.value[0]!

    api.handleMiniSessionChange(mini.id, 'session-b')

    expect(mini.initialSessionId).toBe('session-b')
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/current',
      query: openMiniWsQuery({ sessionId: 'session-b', fullCodePath: '/user/app/a', name: 'A', maximized: '0' })
    })
  })

  it('reopens a hidden workstation with its previous maximized state when no override is provided', async () => {
    const { api, route, router } = createHarness()
    route.path = '/workspace/user/app/a'

    api.openNewMiniWs('session-a', '/user/app/a', 'A', true, true)
    const mini = api.miniWsList.value[0]!

    api.handleMiniMinimize(mini.id)
    api.handleWorkspaceOpenWorkstation({
      full_code_path: '/user/app/a',
      session_id: 'session-a',
      open_as_mini: true
    })

    await nextTick()

    expect(api.miniWsList.value[0]).toMatchObject({
      initialExpanded: true,
      initialMaximized: true,
      visible: true
    })
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/user/app/a',
      query: openMiniWsQuery({ sessionId: 'session-a', fullCodePath: '/user/app/a', name: 'A' })
    })
  })

  it('opens the last session workstation for a path when the launcher has no session id', async () => {
    const { api, route, router } = createHarness()
    route.path = '/workspace/user/app/a'

    api.openNewMiniWs('', '/user/app/a', 'A')
    const mini = api.miniWsList.value[0]!
    api.handleMiniTaskStarted(mini.id, 'session-a')
    api.handleMiniMinimize(mini.id)

    api.handleWorkspaceOpenWorkstation({
      full_code_path: '/user/app/a',
      open_as_mini: true
    })

    await nextTick()

    expect(api.miniWsList.value).toHaveLength(1)
    expect(api.miniWsList.value[0]).toMatchObject({
      initialSessionId: 'session-a',
      initialExpanded: true,
      initialMaximized: true,
      visible: true
    })
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/user/app/a',
      query: openMiniWsQuery({ sessionId: 'session-a', fullCodePath: '/user/app/a', name: 'A' })
    })
  })

  it('opens a blank workstation draft when force_new is requested', async () => {
    const { api, route, router } = createHarness()
    route.path = '/workspace/user/app/a'

    api.openNewMiniWs('session-a', '/user/app/a', 'A')

    api.handleWorkspaceOpenWorkstation({
      full_code_path: '/user/app/a',
      directory_name: 'A',
      force_new: true,
      open_as_mini: true
    })

    await nextTick()

    expect(api.miniWsList.value).toHaveLength(2)
    expect(api.miniWsList.value.find(item => item.initialSessionId === 'session-a')).toMatchObject({
      visible: true
    })
    expect(api.miniWsList.value.find(item => item.initialSessionId === '')).toMatchObject({
      fullCodePath: '/user/app/a',
      dirName: 'A',
      initialExpanded: true,
      initialMaximized: true,
      visible: true
    })
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/user/app/a',
      query: openMiniWsQuery({ fullCodePath: '/user/app/a', name: 'A' })
    })
  })

  it('prefers the last real session over an ambient draft when reopening a directory', async () => {
    const { api, route, router } = createHarness()
    route.path = '/workspace/user/app/a'

    api.openNewMiniWs('session-a', '/user/app/a', 'A')
    api.handleMiniMinimize(api.miniWsList.value[0]!.id)
    api.openAmbientMiniWs('/user/app/a', 'A')

    api.handleWorkspaceOpenWorkstation({
      full_code_path: '/user/app/a',
      open_as_mini: true
    })

    await nextTick()

    expect(api.miniWsList.value.find(item => item.initialSessionId === 'session-a')).toMatchObject({
      initialExpanded: true,
      initialMaximized: true,
      visible: true
    })
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/user/app/a',
      query: openMiniWsQuery({ sessionId: 'session-a', fullCodePath: '/user/app/a', name: 'A' })
    })
  })

  it('allows an explicit maximized override when reopening an existing workstation', async () => {
    const { api, route, router } = createHarness()
    route.path = '/workspace/user/app/a'

    api.openNewMiniWs('session-a', '/user/app/a', 'A', true, true)
    api.handleWorkspaceOpenWorkstation({
      full_code_path: '/user/app/a',
      session_id: 'session-a',
      initial_maximized: false,
      open_as_mini: true
    })

    await nextTick()

    expect(api.miniWsList.value[0]).toMatchObject({
      initialExpanded: true,
      initialMaximized: false,
      visible: true
    })
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/user/app/a',
      query: openMiniWsQuery({ sessionId: 'session-a', fullCodePath: '/user/app/a', name: 'A', maximized: '0' })
    })
  })

  it('writes expanded URL state with underscored platform query keys', () => {
    const { api, router } = createHarness()

    api.openNewMiniWs('session-a', '/user/app/a', 'A')
    const mini = api.miniWsList.value[0]!

    api.handleMiniExpandedChange(mini.id, { expanded: true, sessionId: 'session-a' })

    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/current',
      query: openMiniWsQuery({ sessionId: 'session-a', fullCodePath: '/user/app/a', name: 'A' })
    })
    expect(api.miniWsList.value[0]).toMatchObject({
      initialExpanded: true,
      initialMaximized: true,
      visible: true
    })
  })

  it('restores expanded and maximized state from underscored route params', async () => {
    const { api, route } = createHarness()
    route.query = {
      _mws: 'open',
      _mws_sid: 'session-a',
      _mws_path: '/user/app/a',
      _mws_name: 'A',
      _mws_expanded: '1',
      _mws_maximized: '1'
    }

    api.initializeFromRoute()
    await nextTick()

    expect(api.miniWsList.value).toHaveLength(1)
    expect(api.miniWsList.value[0]).toMatchObject({
      fullCodePath: '/user/app/a',
      dirName: 'A',
      initialSessionId: 'session-a',
      initialExpanded: true,
      initialMaximized: true,
      visible: true
    })
  })

  it('restores expanded and maximized state from canonical platform route params', async () => {
    const { api, route } = createHarness()
    route.query = {
      _open: 'session',
      _focus: 'workspace_session',
      _session_id: 'session-a',
      _source_path: '/user/app/a',
      _mws_expanded: '1',
      _mws_maximized: '1'
    }

    api.initializeFromRoute()
    await nextTick()

    expect(api.miniWsList.value).toHaveLength(1)
    expect(api.miniWsList.value[0]).toMatchObject({
      fullCodePath: '/user/app/a',
      dirName: 'a',
      initialSessionId: 'session-a',
      initialExpanded: true,
      initialMaximized: true,
      visible: true
    })
  })

  it('switches workstation routes without carrying stale query state', async () => {
    const { api, route, router } = createHarness()
    route.path = '/workspace/user/app/a'
    route.query = {
      _mws: 'open',
      _mws_sid: 'old-session',
      _mws_path: '/user/app/a',
      _mws_expanded: '0',
      keyword: 'stale'
    }

    api.handleWorkspaceOpenWorkstation({
      full_code_path: '/user/app/b',
      session_id: 'session-b',
      open_as_mini: true
    })

    await Promise.resolve()
    await nextTick()

    expect(router.push).toHaveBeenCalledWith({
      path: '/workspace/user/app/b',
      query: {}
    })
    expect(api.miniWsList.value).toHaveLength(1)
    expect(api.miniWsList.value[0]).toMatchObject({
      fullCodePath: '/user/app/b',
      initialSessionId: 'session-b',
      initialExpanded: true,
      initialMaximized: true,
      visible: true
    })
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/user/app/b',
      query: openMiniWsQuery({ sessionId: 'session-b', fullCodePath: '/user/app/b', name: 'b' })
    })
  })

  it('cleans collapsed workstation route params during initialization', async () => {
    const { api, route, router } = createHarness()
    route.query = {
      _mws: 'open',
      _mws_sid: 'session-a',
      _mws_path: '/user/app/a',
      _mws_name: 'A',
      _mws_expanded: '0',
      _mws_maximized: '0',
      keyword: 'keep'
    }

    api.initializeFromRoute()
    await nextTick()

    expect(api.miniWsList.value).toHaveLength(0)
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/current',
      query: {
        keyword: 'keep'
      }
    })
  })

  it('keeps ambient workstation hidden by default without URL state', () => {
    const { api, router } = createHarness({ fullCodePath: '/user/app/a', dirName: 'A' })

    api.openAmbientMiniWs()

    expect(api.miniWsList.value).toHaveLength(1)
    expect(api.miniWsList.value[0]).toMatchObject({
      fullCodePath: '/user/app/a',
      initialExpanded: false,
      visible: false
    })
    expect(router.replace).not.toHaveBeenCalled()
  })

  it('keeps the visible workstation open when the selected directory changes', () => {
    const { api, router } = createHarness({ fullCodePath: '/user/app/a', dirName: 'A' })

    api.openNewMiniWs('session-a', '/user/app/a', 'A')
    api.openAmbientMiniWs('/user/app/b', 'B')

    expect(api.miniWsList.value).toHaveLength(1)
    expect(api.miniWsList.value.find(item => item.initialSessionId === 'session-a')).toMatchObject({
      fullCodePath: '/user/app/a',
      visible: true
    })
    expect(api.miniWsList.value.find(item => item.fullCodePath === '/user/app/b')).toBeUndefined()
    expect(router.replace).not.toHaveBeenCalled()
  })

  it('uses resolved directory name instead of the code path segment', async () => {
    const { api, router } = createHarness(null, {
      resolvePathName: (fullCodePath: string) => fullCodePath === '/user/app/customer_admin' ? '客户管理' : undefined
    })

    api.handleWorkspaceOpenWorkstation({
      full_code_path: '/user/app/customer_admin',
      session_id: 'session-customer',
      open_as_mini: true
    })

    await Promise.resolve()
    await nextTick()

    expect(api.miniWsList.value).toHaveLength(1)
    expect(api.miniWsList.value[0]).toMatchObject({
      fullCodePath: '/user/app/customer_admin',
      dirName: '客户管理',
      initialSessionId: 'session-customer',
      visible: true
    })
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/user/app/customer_admin',
      query: openMiniWsQuery({ sessionId: 'session-customer', fullCodePath: '/user/app/customer_admin', name: '客户管理' })
    })
  })

  it('uses session directory_name when opening a workstation session', async () => {
    const { api, router } = createHarness()

    api.handleWorkspaceOpenWorkstation({
      full_code_path: '/user/app/customer_admin',
      session_id: 'session-customer',
      directory_name: '客户管理',
      open_as_mini: true
    })

    await Promise.resolve()
    await nextTick()

    expect(api.miniWsList.value[0]).toMatchObject({
      fullCodePath: '/user/app/customer_admin',
      dirName: '客户管理',
      initialSessionId: 'session-customer',
      visible: true
    })
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/user/app/customer_admin',
      query: openMiniWsQuery({ sessionId: 'session-customer', fullCodePath: '/user/app/customer_admin', name: '客户管理' })
    })
  })

  it('opens workstation sessions maximized when requested', async () => {
    const { api, router } = createHarness()

    api.handleWorkspaceOpenWorkstation({
      full_code_path: '/user/app/customer_admin',
      session_id: 'session-customer',
      directory_name: '客户管理',
      initial_maximized: true,
      open_as_mini: true
    })

    await Promise.resolve()
    await nextTick()

    expect(api.miniWsList.value[0]).toMatchObject({
      initialExpanded: true,
      initialMaximized: true,
      visible: true
    })
    expect(router.replace).toHaveBeenLastCalledWith({
      path: '/workspace/user/app/customer_admin',
      query: openMiniWsQuery({ sessionId: 'session-customer', fullCodePath: '/user/app/customer_admin', name: '客户管理' })
    })
  })
})
