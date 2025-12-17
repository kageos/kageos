/**
 * useWorkspaceRouting - 路由管理 Composable
 * 
 * 职责：
 * - 路由同步到 Tab
 * - 从路由恢复 Tab
 * - 路由变化处理
 */

import { watch, ref, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { extractWorkspacePath } from '@/utils/route'
import { preserveQueryParamsForTable, preserveQueryParamsForForm, isFunctionGroupDetail } from '@/utils/queryParams'
import { serviceFactory } from '../../infrastructure/factories'
import { eventBus, RouteEvent, WorkspaceEvent } from '../../infrastructure/eventBus'
import { Logger } from '@/core/utils/logger'
import type { ServiceTree, App } from '../../domain/services/WorkspaceDomainService'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/types'

export function useWorkspaceRouting(options: {
  tabs: () => any[]
  activeTabId: () => string
  serviceTree: () => ServiceTreeType[]
  currentApp: () => AppType | null
  appList: () => AppType[]
  loadAppList: () => Promise<void>
  findNodeByPath: (tree: ServiceTreeType[], path: string) => ServiceTreeType | null
  checkAndExpandForkedPaths: () => void
  expandCurrentRoutePath: () => void
}) {
  const route = useRoute()
  const router = useRouter()
  const stateManager = serviceFactory.getWorkspaceStateManager()
  const applicationService = serviceFactory.getWorkspaceApplicationService()

  // 防重复调用保护
  let isLoadingAppFromRoute = false
  let isSyncingRouteToTab = false
  let lastSavedTabId: string | null = null // 🔥 记录上次保存的 Tab ID，防止重复保存
  let lastProcessedUpdateCompleted: { path: string, source: string } | null = null // 🔥 记录上次处理的 updateCompleted 事件，防止重复处理

  // 从路由同步到 Tab 状态（路由变化时调用）
  const syncRouteToTab = async () => {
    // 🔥 防重复调用保护
    if (isSyncingRouteToTab) {
      Logger.debug('useWorkspaceRouting', 'syncRouteToTab 正在执行，跳过重复调用', { path: route.path })
      return
    }
    
    const fullPath = extractWorkspacePath(route.path)
    
    if (!fullPath) {
      // 空路径，不处理
      return
    }
    
    // 🔥 检查是否是函数组详情页面（_node_type=function_group）
    // 如果是函数组，不需要查找 Tab，直接返回（函数组详情页面会自己处理）
    if (isFunctionGroupDetail(route.query)) {
      Logger.debug('useWorkspaceRouting', '检测到函数组详情页面，跳过 Tab 查找', { path: route.path })
      return
    }
    
    Logger.debug('useWorkspaceRouting', 'syncRouteToTab 开始执行', { path: route.path, fullPath })
    isSyncingRouteToTab = true
    
    try {
      // 解析路径，找到对应的 Tab
      const targetTab = options.tabs().find(t => {
        const tabPath = t.path?.replace(/^\//, '') || ''
        const routePath = fullPath?.replace(/^\//, '') || ''
        return tabPath === routePath
      })
      
      if (targetTab) {
        // Tab 已存在，激活它（不触发路由更新）
        // 🔥 只有在 Tab ID 不同时才激活，避免重复调用
        const currentActiveTabId = options.activeTabId()
        if (currentActiveTabId !== targetTab.id) {
          Logger.debug('useWorkspaceRouting', '激活 Tab', { 
            currentActiveTabId, 
            targetTabId: targetTab.id 
          })
          applicationService.activateTab(targetTab.id)
        } else {
          Logger.debug('useWorkspaceRouting', 'Tab 已激活，跳过', { 
            tabId: targetTab.id 
          })
        }
        
        // 🔥 Tab 激活后，保存 Tab 的路由状态（用于 workspace-node-click 场景）
        // 因为 workspace-node-click 时，路由更新完成时 Tab 可能还没有激活
        // 所以在这里保存，确保保存的是正确的 Tab ID
        await nextTick() // 等待 activateTab 完成
        const currentTabId = options.activeTabId()
        if (currentTabId === targetTab.id) {
          // 🔥 防止重复保存：如果已经保存过这个 Tab 的路由状态，且路由没有变化，则跳过
          const currentPath = route.path
          const currentQuery = { ...route.query }
          const shouldSave = lastSavedTabId !== targetTab.id // 如果 Tab ID 变化了，需要保存
          
          if (shouldSave) {
            // 确保 Tab 已经激活，再保存路由状态
            // 通过事件通知 RouteManager 保存路由状态
            // 🔥 使用当前路由的 path 和 query，确保保存的是正确的路由状态
            eventBus.emit(RouteEvent.updateRequested, {
              path: currentPath,
              query: currentQuery,
              replace: false, // 不实际更新路由，只是触发保存
              preserveParams: {
                state: true
              },
              source: 'sync-route-to-tab-save-state',
              meta: { tabId: targetTab.id, path: currentPath, query: currentQuery } // 🔥 传递 Tab ID 和路由状态，确保保存正确
            } as any)
            lastSavedTabId = targetTab.id // 🔥 记录已保存的 Tab ID
          }
        }
      
        // 🔥 Tab 切换时，即使 Tab 已经激活，也需要确保函数详情已加载
        // 因为 Tab 切换时，路由已经更新了，函数界面需要刷新
        if (targetTab.node && targetTab.node.type === 'function') {
          const detail = stateManager.getFunctionDetail(targetTab.node)
          if (!detail) {
            // 使用 handleNodeClick 加载函数详情
            applicationService.handleNodeClick(targetTab.node)
          } else {
            // 🔥 函数详情已加载，但 Tab 切换时路由已更新，需要触发函数界面刷新
            // 发出函数加载完成事件，让 FormView/TableView 重新初始化
            eventBus.emit(WorkspaceEvent.functionLoaded, {
              function: targetTab.node,
              detail: detail
            })
          }
        } else if (targetTab.node && targetTab.node.type === 'package') {
          // 🔥 如果是 package 类型，确保设置了当前函数
          applicationService.triggerNodeClick(targetTab.node)
        }
      } else {
        // Tab 不存在，从路由打开新 Tab
        // 注意：这里需要确保服务树已加载，否则无法找到节点
        if (options.serviceTree().length > 0) {
          await loadAppFromRoute()
        }
      }
    } finally {
      isSyncingRouteToTab = false
      Logger.debug('useWorkspaceRouting', 'syncRouteToTab 执行完成', { path: route.path })
    }
  }

  // 从路由解析应用并加载（主要用于刷新时）
  const loadAppFromRoute = async () => {
    // 防止重复调用
    if (isLoadingAppFromRoute) {
      return
    }
    
    // 提取路径
    const fullPath = extractWorkspacePath(route.path)
    
    if (!fullPath) {
      return
    }

    const pathSegments = fullPath.split('/').filter(Boolean)
    if (pathSegments.length < 2) {
      return
    }

    const [user, appCode] = pathSegments
    
    try {
      isLoadingAppFromRoute = true
      
      // 确保应用列表已加载
      if (options.appList().length === 0) {
        await options.loadAppList()
      }
      
      // 从已加载的应用列表中查找
      const app = options.appList().find((a: AppType) => a.user === user && a.code === appCode)
      
      if (!app) {
        return
      }
      
      const targetAppId = app.id
      let appSwitched = false
      const pendingAppId = ref<number | string | null>(null)

      // 检查当前应用是否已经是目标应用
      const currentAppState = options.currentApp()
      if (!currentAppState || String(currentAppState.id) !== String(targetAppId)) {
        // 需要切换应用
        if (String(pendingAppId.value) !== String(targetAppId)) {
          pendingAppId.value = targetAppId
          try {
            const appForService: App = {
              id: app.id,
              user: app.user,
              code: app.code,
              name: app.name
            }
            await applicationService.triggerAppSwitch(appForService)
            appSwitched = true
          } catch (error) {
            // 静默失败
            pendingAppId.value = null
            return
          }
        }
      }

      // 处理子路径（打开 Tab）
      if (pathSegments.length > 2) {
        const functionPath = '/' + pathSegments.join('/') // 构造完整路径，如 /luobei/demo/crm/list
        
        // 检查是否有 _tab 参数（create/edit/detail/OnTableAddRow 模式）
        const tabParam = route.query._tab as string
        if (tabParam === 'create' || tabParam === 'edit' || tabParam === 'detail' || tabParam === 'OnTableAddRow') {
          // create/edit/detail/OnTableAddRow 模式不需要打开 Tab，直接加载函数详情
          const tryLoadFunction = () => {
            const tree = options.serviceTree()
            if (tree && tree.length > 0) {
              const node = options.findNodeByPath(tree, functionPath)
              if (node) {
                const serviceNode: ServiceTree = node as any
                // 设置当前函数，但不打开 Tab
                applicationService.handleNodeClick(serviceNode)
              }
            }
          }
          
          if (appSwitched) {
            let retries = 0
            const interval = setInterval(() => {
              if (options.serviceTree().length > 0 || retries > 10) {
                clearInterval(interval)
                tryLoadFunction()
              }
              retries++
            }, 200)
          } else {
            tryLoadFunction()
          }
          
          // 检查 _forked 参数，自动展开路径
          if (route.query._forked) {
            nextTick(() => {
              options.checkAndExpandForkedPaths()
            })
          }
          
          return // create/edit/detail/OnTableAddRow 模式不打开 Tab
        }
        
        // 检查 _forked 参数，自动展开路径
        if (route.query._forked) {
          nextTick(() => {
            options.checkAndExpandForkedPaths()
          })
        }
        
        // 尝试查找节点并打开/激活 Tab
        // 使用早期返回优化条件判断
        const tryOpenTab = async () => {
          const tree = options.serviceTree()
          
          // 早期返回：服务树为空
          if (!tree || tree.length === 0) {
            return
          }
          
          const node = options.findNodeByPath(tree, functionPath)
          
          // 早期返回：节点不存在
          if (!node) {
            return
          }
          
          const serviceNode: ServiceTree = node as any
          
          // 🔥 如果是目录节点，只设置当前函数，不打开 Tab
          if (serviceNode.type === 'package') {
            applicationService.triggerNodeClick(serviceNode)
            return
          }
          
          // 🔥 注意：_link_type 参数的处理已移至 setupRouteWatch 中的 link-widget updateCompleted 事件监听
          // 这里不再处理 _link_type，避免在路由更新完成前就清除参数
          
          // 检查 Tab 是否存在
          const tabsArray = Array.isArray(options.tabs()) ? options.tabs() : []
          const existingTab = tabsArray.find(t => 
            t.path === serviceNode.full_code_path || t.path === String(serviceNode.id)
          )
          
          if (existingTab) {
            // Tab 已存在，激活它（不触发路由更新）
            if (options.activeTabId() !== existingTab.id) {
              isSyncingRouteToTab = true
              applicationService.activateTab(existingTab.id)
              isSyncingRouteToTab = false
            }
            
            // 无论是否激活，都检查函数详情是否已加载
            if (existingTab.node && existingTab.node.type === 'function') {
              const detail = stateManager.getFunctionDetail(existingTab.node)
              if (!detail) {
                applicationService.handleNodeClick(existingTab.node)
              }
            }
            return
          }
          
          // Tab 不存在，打开新 Tab
          applicationService.triggerNodeClick(serviceNode)
        }

        // 等待服务树加载
        if (appSwitched) {
          let retries = 0
          const interval = setInterval(async () => {
            if (options.serviceTree().length > 0 || retries > 10) {
              clearInterval(interval)
              await tryOpenTab()
            }
            retries++
          }, 200)
        } else {
          await tryOpenTab()
        }
        
        // 展开目录树
        if (route.query._forked) {
          nextTick(() => {
            options.checkAndExpandForkedPaths()
          })
        } else {
          options.expandCurrentRoutePath()
        }
      }
    } catch (error) {
      // 静默失败
    } finally {
      isLoadingAppFromRoute = false
    }
  }

  // 设置路由监听
  // 🔥 阶段4：改为监听 RouteEvent.routeChanged 事件，而不是直接 watch route
  // 这样可以避免程序触发的路由更新导致循环，并且不需要防抖
  const setupRouteWatch = () => {
    // 监听路由变化（用户操作：浏览器前进/后退）
    eventBus.on(RouteEvent.routeChanged, async (payload: { path: string, query: any, source: string }) => {
      // 🔥 只处理用户操作（浏览器前进/后退）或外部变化，不处理程序触发的更新
      // 注意：程序触发的更新不会发出事件（RouteManager.isUpdating 为 true 时）
      if (payload.source === 'router-change') {
        syncRouteToTab()
      }
    })
    
    // 🔥 监听路由更新完成事件（程序触发的更新）
    // 当来源是 workspace-node-click 或 tab 切换相关时，需要主动触发 syncRouteToTab
    // 因为程序触发的路由更新不会发出 routeChanged 事件
    eventBus.on(RouteEvent.updateCompleted, async (payload: { path: string, query: any, source: string }) => {
      // 🔥 处理 link-widget：清除 _link_type 参数并同步路由到 Tab
      if (payload.source === 'link-widget') {
        // link 跳转完成后，清除 _link_type 临时参数
        // 使用 payload.query（来自 RouteManager 的 updateCompleted 事件），确保包含所有 link 跳转的参数
        const preservedQuery: Record<string, string | string[]> = {}
        Object.keys(payload.query).forEach(key => {
          if (key !== '_link_type') {
            const value = payload.query[key]
            if (value !== null && value !== undefined) {
              preservedQuery[key] = Array.isArray(value) 
                ? value.filter(v => v !== null).map(v => String(v))
                : String(value)
            }
          }
        })
        
        Logger.debug('useWorkspaceRouting', 'link-widget 完成，准备清除 _link_type 并同步路由', {
          originalQuery: payload.query,
          preservedQuery,
          path: payload.path
        })
        
        // 🔥 先同步路由到 Tab（确保 Tab 和函数已更新，页面会刷新）
        // 使用 nextTick 确保路由已经更新完成
        await nextTick()
        syncRouteToTab()
        
        // 🔥 发出路由更新请求，清除 _link_type
        // 🔥 关键：使用 preservedQuery（已经包含了所有 link 跳转的参数，除了 _link_type）
        // 并且设置 linkNavigation: true，确保 RouteManager 不会覆盖这些参数
        eventBus.emit(RouteEvent.updateRequested, {
          path: payload.path,
          query: preservedQuery,  // 🔥 这里已经包含了 eq、in 等所有参数
          replace: true,
          preserveParams: {
            linkNavigation: true  // 保持 linkNavigation: true，确保 RouteManager 不会覆盖 preservedQuery 中的参数
          },
          source: 'workspace-routing-clear-link-type'
        })
        return
      }
      
      // 处理 workspace-node-click：需要创建/激活 Tab
      // 处理 workspace-node-click-package：需要设置当前函数（package 类型）
      // 处理 tab 切换相关：需要刷新函数界面（确保函数详情已加载）
      // 注意：tab-switch 是 RouteManager.handleTabSwitch 发出的，tab-switch-activeTabId 和 tab-click 是 useWorkspaceTabs 发出的
      if (payload.source === 'workspace-node-click' || 
          payload.source === 'workspace-node-click-package' ||
          payload.source === 'tab-switch' || 
          payload.source === 'tab-switch-activeTabId' || 
          payload.source === 'tab-click') {
        // 🔥 防重复处理：如果已经处理过相同的 updateCompleted 事件，跳过
        const eventKey = `${payload.source}:${payload.path}`
        if (lastProcessedUpdateCompleted && 
            lastProcessedUpdateCompleted.path === payload.path && 
            lastProcessedUpdateCompleted.source === payload.source) {
          Logger.debug('useWorkspaceRouting', '跳过重复的 updateCompleted 事件', { 
            source: payload.source, 
            path: payload.path 
          })
          return
        }
        lastProcessedUpdateCompleted = { path: payload.path, source: payload.source }
        
        // 使用 nextTick 确保路由已经更新完成
        await nextTick()
        
        // 🔥 如果是 workspace-node-click-package，需要确保设置了当前函数
        if (payload.source === 'workspace-node-click-package') {
          const fullPath = extractWorkspacePath(payload.path)
          if (fullPath) {
            const pathSegments = fullPath.split('/').filter(Boolean)
            if (pathSegments.length >= 3) {
              const functionPath = '/' + pathSegments.join('/')
              const tree = options.serviceTree()
              if (tree && tree.length > 0) {
                const node = options.findNodeByPath(tree, functionPath)
                if (node && node.type === 'package') {
                  const serviceNode: ServiceTree = node as any
                  applicationService.triggerNodeClick(serviceNode)
                }
              }
            }
          }
        }
        
        syncRouteToTab()
        
        // 🔥 清除记录，允许下次处理（使用 setTimeout 延迟清除，避免快速连续触发）
        setTimeout(() => {
          if (lastProcessedUpdateCompleted?.path === payload.path && 
              lastProcessedUpdateCompleted?.source === payload.source) {
            lastProcessedUpdateCompleted = null
          }
        }, 100)
      }
    })
  }

  return {
    syncRouteToTab,
    loadAppFromRoute,
    setupRouteWatch,
    isSyncingRouteToTab: () => isSyncingRouteToTab
  }
}

