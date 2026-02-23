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
import { preserveQueryParamsForTable, preserveQueryParamsForForm } from '@/utils/queryParams'
import { serviceFactory } from '../../infrastructure/factories'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import { eventBus, RouteEvent, WorkspaceEvent } from '../../infrastructure/eventBus'
import { RouteSource } from '@/utils/routeSource'
import { Logger } from '@/core/utils/logger'
import { getAppWithServiceTree } from '@/api/app'
import type { ServiceTree, App } from '../../domain/services/WorkspaceDomainService'
import type { App as AppType, ServiceTree as ServiceTreeType } from '@/types'

export function useWorkspaceRouting(
  options: {
    serviceTree: () => ServiceTreeType[]
    currentApp: () => AppType | null
    appList: () => AppType[]
    loadAppList: () => Promise<void>
    findNodeByPath: (tree: ServiceTreeType[], path: string) => ServiceTreeType | null
    expandCurrentRoutePath: () => void
  },
  serviceProvider: IServiceProvider = serviceFactory  // 🔥 通过参数注入，提高可测试性
) {
  const route = useRoute()
  const router = useRouter()
  const stateManager = serviceProvider.getWorkspaceStateManager()
  const applicationService = serviceProvider.getWorkspaceApplicationService()

  // 防重复调用保护
  let isLoadingAppFromRoute = false
  let isSyncingRouteToTab = false
  let lastProcessedUpdateCompleted: { path: string, source: string } | null = null // 🔥 记录上次处理的 updateCompleted 事件，防止重复处理

  // 从路由同步到当前函数（路由变化时调用）
  const syncRouteToTab = async () => {
    // 🔥 防重复调用保护
    if (isSyncingRouteToTab) {
      return
    }
    
    const fullPath = extractWorkspacePath(route.path)
    
    if (!fullPath) {
      // 空路径，不处理
      return
    }

    isSyncingRouteToTab = true
    
    try {
      // 从路由路径找到对应的节点
      const tree = options.serviceTree()
      if (tree.length === 0) {
        // 服务树未加载，等待加载
        return
      }
      
      const functionPath = '/' + fullPath.split('/').filter(Boolean).join('/')
      const node = options.findNodeByPath(tree, functionPath)
      
      if (node) {
        const serviceNode: ServiceTree = node as any
        
        // 检查当前函数是否已经是目标节点
        const currentFunction = stateManager.getCurrentFunction()
        if (currentFunction && (
          currentFunction.id === serviceNode.id || 
          currentFunction.full_code_path === serviceNode.full_code_path
        )) {
          // 🔥 已是目标节点：仅当 URL 只是打开/关闭详情（_tab=detail&_id）时不重新加载，避免表格 initializeTable 和骨架屏
          const onlyDetailParams = route.query._tab === 'detail' && route.query._id
          if (onlyDetailParams && serviceNode.type === 'function') {
            return
          }
          // 已经是目标节点，直接触发节点点击（会加载函数详情）
          if (serviceNode.type === 'function') {
            applicationService.handleNodeClick(serviceNode)
          }
          return
        }
        
        // 触发节点点击，加载函数详情
        applicationService.triggerNodeClick(serviceNode)
      } else {
        // 节点不存在，尝试从路由加载应用（可能是新应用）
          await loadAppFromRoute()
      }
    } finally {
      isSyncingRouteToTab = false
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
      
      // 优化：如果路由中有应用信息，直接使用合并接口获取应用详情和服务目录树
      // 不需要先加载整个应用列表
      // triggerAppSwitch 内部会调用合并接口，所以我们可以直接构造一个临时的 app 对象
      const currentAppState = options.currentApp()
      
      // 检查当前应用是否已经是目标应用（通过 user 和 code 匹配，因为 id 可能还没有）
      if (currentAppState && currentAppState.user === user && currentAppState.code === appCode) {
        // 当前应用已经是目标应用，不需要切换
        return
      }
      
      // 构造临时 app 对象（只有基本信息，triggerAppSwitch 会通过合并接口获取完整信息）
      const appForService: App = {
        id: 0, // 临时 ID，triggerAppSwitch 会通过合并接口获取真实的 ID
        user: user,
        code: appCode,
        name: '' // 临时名称，triggerAppSwitch 会通过合并接口获取真实的名称
      }
      
      try {
        // triggerAppSwitch 内部会使用合并接口获取应用详情和服务目录树
        // 这样就不需要先加载整个应用列表了
        await applicationService.triggerAppSwitch(appForService)
      } catch (error) {
        Logger.error('useWorkspaceRouting', '切换应用失败', error)
        // 如果切换失败，回退到加载应用列表
        if (options.appList().length === 0) {
          await options.loadAppList()
          const foundApp = options.appList().find((a: AppType) => a.user === user && a.code === appCode)
          if (foundApp) {
            const appForServiceFallback: App = {
              id: foundApp.id,
              user: foundApp.user,
              code: foundApp.code,
              name: foundApp.name
            }
            await applicationService.triggerAppSwitch(appForServiceFallback)
          }
        }
        return
      }
      
      // 标记已切换（用于后续处理）
      let appSwitched = true

      // ⭐ 处理根路径和子路径（统一逻辑）
      // 根路径：pathSegments.length === 2，如 /system/official
      // 子路径：pathSegments.length > 2，如 /system/official/agent/plugin
      if (pathSegments.length >= 2) {
        const functionPath = '/' + pathSegments.join('/') // 构造完整路径，如 /luobei/demo 或 /luobei/demo/crm/list
        
        // 🔥 统一的节点加载逻辑（适用于 app/package/function）
        const tryLoadNode = () => {
          const tree = options.serviceTree()
          if (tree && tree.length > 0) {
            const node = options.findNodeByPath(tree, functionPath)
            if (node) {
              const serviceNode: ServiceTree = node as any
              Logger.debug('[useWorkspaceRouting]', '选中节点', { 
                nodeId: serviceNode.id, 
                nodeName: serviceNode.name,
                nodeType: serviceNode.type,
                nodePath: functionPath
              })
              // 触发节点点击，选中节点并显示详情
              applicationService.handleNodeClick(serviceNode)
            }
          }
        }
        
        // 检查是否有 _tab 参数（create/edit/detail/OnTableAddRow 模式）
        const tabParam = route.query._tab as string
        if (tabParam === 'create' || tabParam === 'edit' || tabParam === 'detail' || tabParam === 'OnTableAddRow') {
          // create/edit/detail/OnTableAddRow 模式不需要打开 Tab，直接加载节点详情
          const tryLoadFunction = tryLoadNode
          
          // 🔥 使用 once 监听器，确保只执行一次，避免无限循环
          if (appSwitched) {
            // 使用 once 替代 on，确保监听器只执行一次
            eventBus.once(WorkspaceEvent.serviceTreeLoaded, async () => {
              await nextTick()
              tryLoadFunction()
            })
            // 如果服务树已经加载，直接执行
            if (options.serviceTree().length > 0) {
              tryLoadFunction()
            }
          } else {
            tryLoadFunction()
          }
          
          return // create/edit/detail/OnTableAddRow 模式不打开 Tab
        }
        
        // 🔥 根路径（app 节点）：只选中节点，不打开 Tab
        if (pathSegments.length === 2) {
          // 使用 once 监听器，确保只执行一次
          if (appSwitched) {
            eventBus.once(WorkspaceEvent.serviceTreeLoaded, async () => {
              await nextTick()
              tryLoadNode()
            })
            // 如果服务树已经加载，直接执行
            if (options.serviceTree().length > 0) {
              tryLoadNode()
            }
          } else {
            tryLoadNode()
          }
          
          return // 根路径处理完成
        }
        
        // 🔥 子路径（package/function 节点）：打开/激活 Tab
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
          
          // 直接触发节点点击，加载函数详情
          applicationService.triggerNodeClick(serviceNode)
        }

        // 🔥 使用 once 监听器，确保只执行一次，避免无限循环
        if (appSwitched) {
          // 使用 once 替代 on，确保监听器只执行一次
          eventBus.once(WorkspaceEvent.serviceTreeLoaded, async () => {
            await nextTick()
            await tryOpenTab()
          })
          // 如果服务树已经加载，直接执行
          if (options.serviceTree().length > 0) {
            await tryOpenTab()
          }
        } else {
          await tryOpenTab()
        }
        
        // 展开目录树
        options.expandCurrentRoutePath()
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
          source: RouteSource.WORKSPACE_ROUTING_CLEAR_LINK_TYPE
        })
        return
      }
      
      // 处理 workspace-node-click：需要加载函数详情
      // 处理 workspace-node-click-package：需要设置当前函数（package 类型）
      if (payload.source === RouteSource.WORKSPACE_NODE_CLICK || 
          payload.source === RouteSource.WORKSPACE_NODE_CLICK_PACKAGE) {
        // 🔥 防重复处理：如果已经处理过相同的 updateCompleted 事件，跳过
        const eventKey = `${payload.source}:${payload.path}`
        if (lastProcessedUpdateCompleted && 
            lastProcessedUpdateCompleted.path === payload.path && 
            lastProcessedUpdateCompleted.source === payload.source) {
          return
        }
        lastProcessedUpdateCompleted = { path: payload.path, source: payload.source }
        
        // 使用 nextTick 确保路由已经更新完成
        await nextTick()
        
        // 🔥 如果是 workspace-node-click，需要触发节点点击来加载函数详情
        if (payload.source === RouteSource.WORKSPACE_NODE_CLICK) {
          const fullPath = extractWorkspacePath(payload.path)
          if (fullPath) {
            const pathSegments = fullPath.split('/').filter(Boolean)
            if (pathSegments.length >= 3) {
              const functionPath = '/' + pathSegments.join('/')
              const tree = options.serviceTree()
              if (tree && tree.length > 0) {
                const node = options.findNodeByPath(tree, functionPath)
                if (node && node.type === 'function') {
                  const serviceNode: ServiceTree = node as any
                  // 🔥 触发节点点击，确保函数详情已加载
                  applicationService.triggerNodeClick(serviceNode)
                  // 清除记录，允许下次处理
                  setTimeout(() => {
                    if (lastProcessedUpdateCompleted?.path === payload.path && 
                        lastProcessedUpdateCompleted?.source === payload.source) {
                      lastProcessedUpdateCompleted = null
                    }
                  }, 100)
                  return
                }
              }
            }
          }
        }
        
        // ⭐ 优化：workspace-node-click-package 不需要再次触发节点点击
        // 因为在 handleNodeClick 中已经调用过 triggerNodeClick 了
        // 这里只需要确保路由已同步即可
        if (payload.source === 'workspace-node-click-package') {
          // 不需要再次调用 triggerNodeClick，因为已经在 handleNodeClick 中调用过了
          // 只需要清除记录，允许下次处理
          setTimeout(() => {
            if (lastProcessedUpdateCompleted?.path === payload.path && 
                lastProcessedUpdateCompleted?.source === payload.source) {
              lastProcessedUpdateCompleted = null
            }
          }, 100)
          return
        }
        
        // 对于 tab 切换相关事件，只同步路由到 Tab
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

