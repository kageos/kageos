/**
 * useWorkspaceServiceTree - 服务树管理 Composable
 * 
 * 职责：
 * - 服务树节点关联
 * - 服务树展开逻辑
 * - 目录创建
 */

import { ref, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { extractWorkspacePath } from '@/utils/route'
import { ElNotification } from 'element-plus'
import { serviceFactory } from '../../infrastructure/factories'
import { createServiceTree } from '@/api/service-tree'
import type { ServiceTree as ServiceTreeType, CreateServiceTreeRequest } from '@/types'
import type { App } from '../../domain/services/WorkspaceDomainService'
import ServiceTreePanel from '@/components/ServiceTreePanel.vue'

export function useWorkspaceServiceTree() {
  const route = useRoute()
  const applicationService = serviceFactory.getWorkspaceApplicationService()

  // 创建目录对话框状态
  const createDirectoryDialogVisible = ref(false)
  const creatingDirectory = ref(false)
  const currentParentNode = ref<ServiceTreeType | null>(null)
  const createDirectoryForm = ref<CreateServiceTreeRequest>({
    user: '',
    app: '',
    name: '',
    code: '',
    parent_id: 0,
    description: '',
    tags: ''
  })

  // 处理创建目录
  const handleCreateDirectory = (parentNode: ServiceTreeType | null, currentApp: () => any) => {
    if (!currentApp()) {
      ElNotification.warning({
        title: '提示',
        message: '请先选择一个应用'
      })
      return
    }
    currentParentNode.value = parentNode || null
    createDirectoryForm.value = {
      user: currentApp().user,
      app: currentApp().code,
      name: '',
      code: '',
      parent_id: parentNode ? Number(parentNode.id) : 0,
      description: '',
      tags: ''
    }
    createDirectoryDialogVisible.value = true
  }

  // 重置创建目录表单
  const resetCreateDirectoryForm = (currentApp?: (() => any) | any) => {
    // 处理 currentApp 可能是函数或值的情况
    let app: any = null
    if (typeof currentApp === 'function') {
      app = currentApp()
    } else if (currentApp) {
      app = currentApp
    }
    
    createDirectoryForm.value = {
      user: app?.user || '',
      app: app?.code || '',
      name: '',
      code: '',
      parent_id: 0,
      description: '',
      tags: ''
    }
    currentParentNode.value = null
  }

  // 提交创建目录
  const handleSubmitCreateDirectory = async (currentApp: () => any) => {
    if (!currentApp()) {
      ElNotification.warning({
        title: '提示',
        message: '请先选择一个应用'
      })
      return
    }
    
    if (!createDirectoryForm.value.name || !createDirectoryForm.value.code) {
      ElNotification.warning({
        title: '提示',
        message: '请输入目录名称和代码'
      })
      return
    }
    
    // 验证代码格式
    if (!/^[a-z0-9_]+$/.test(createDirectoryForm.value.code)) {
      ElNotification.warning({
        title: '提示',
        message: '目录代码只能包含小写字母、数字和下划线'
      })
      return
    }

    try {
      creatingDirectory.value = true
      const requestData: CreateServiceTreeRequest = {
        user: currentApp().user,
        app: currentApp().code,
        name: createDirectoryForm.value.name,
        code: createDirectoryForm.value.code,
        parent_id: createDirectoryForm.value.parent_id || 0,
        description: createDirectoryForm.value.description || '',
        tags: createDirectoryForm.value.tags || ''
      }
      
      await createServiceTree(requestData)
      ElNotification.success({
        title: '成功',
        message: '创建服务目录成功'
      })
      createDirectoryDialogVisible.value = false
      resetCreateDirectoryForm(currentApp)
      
      // 刷新服务目录树
      if (currentApp()) {
        const app = currentApp()
        const appForService: App = {
          id: app.id,
          user: app.user,
          code: app.code,
          name: app.name,
          nats_id: app.nats_id || 0,
          host_id: app.host_id || 0,
          status: (app.status || 'enabled') as 'enabled' | 'disabled',
          version: app.version || '',
          created_at: app.created_at || '',
          updated_at: app.updated_at || ''
        }
        await applicationService.triggerAppSwitch(appForService)
      }
    } catch (error: any) {
      // 🔥 统一使用 msg 字段
      const errorMessage = error?.response?.data?.msg || error?.message || '创建服务目录失败'
      ElNotification.error({
        title: '错误',
        message: errorMessage
      })
    } finally {
      creatingDirectory.value = false
    }
  }

  // 展开当前路由对应的路径（刷新时自动展开）
  const expandCurrentRoutePath = (
    serviceTree: () => ServiceTreeType[],
    serviceTreePanelRef: () => InstanceType<typeof ServiceTreePanel> | null,
    currentApp: () => any
  ) => {
    if (serviceTree().length === 0 || !serviceTreePanelRef() || !currentApp()) {
      return
    }
    
    const fullPath = extractWorkspacePath(route.path)
    if (!fullPath) return
    
    const pathSegments = fullPath.split('/').filter(Boolean)
    if (pathSegments.length < 3) return // 至少需要 user/app/function
    
    const functionPath = '/' + pathSegments.join('/')
    
    nextTick(() => {
      setTimeout(() => {
        if (serviceTreePanelRef() && serviceTreePanelRef()!.expandPaths) {
          serviceTreePanelRef()!.expandPaths([functionPath])
        }
      }, 300)
    })
  }

  // 检查并展开 forked 路径
  const checkAndExpandForkedPaths = (
    serviceTree: () => ServiceTreeType[],
    serviceTreePanelRef: () => InstanceType<typeof ServiceTreePanel> | null,
    currentApp: () => any
  ) => {
    const forkedParam = route.query._forked as string
    if (!forkedParam) return
    
    // 检查当前应用是否匹配 URL 中的应用
    const pathSegments = extractWorkspacePath(route.path).split('/').filter(Boolean)
    if (pathSegments.length >= 2) {
      const [urlUser, urlApp] = pathSegments
      if (currentApp() && (currentApp().user !== urlUser || currentApp().code !== urlApp)) {
        return // 应用不匹配，不展开
      }
    }
    
    if (forkedParam && serviceTree().length > 0 && serviceTreePanelRef() && currentApp()) {
      const forkedPaths = decodeURIComponent(forkedParam).split(',').filter(Boolean)
      
      // 验证路径是否属于当前应用
      const validPaths = forkedPaths.filter(path => {
        const pathMatch = path.match(/^\/([^/]+)\/([^/]+)/)
        if (pathMatch) {
          const [, pathUser, pathApp] = pathMatch
          const isValid = pathUser === currentApp()?.user && pathApp === currentApp()?.code
          return isValid
        }
        return false
      })
      
      if (validPaths.length > 0) {
        nextTick(() => {
          setTimeout(() => {
            if (serviceTreePanelRef() && serviceTreePanelRef()!.expandPaths) {
              serviceTreePanelRef()!.expandPaths(validPaths)
            }
          }, 500) // 延迟确保树完全渲染
        })
      }
    }
  }

  // 处理复制链接
  const handleCopyLink = (node: ServiceTreeType) => {
    const link = `${window.location.origin}/workspace${node.full_code_path}`
    navigator.clipboard.writeText(link).then(() => {
      ElNotification.success({
        title: '成功',
        message: '链接已复制到剪贴板'
      })
    }).catch(() => {
      ElNotification.error({
        title: '错误',
        message: '复制链接失败'
      })
    })
  }

  return {
    // 状态
    createDirectoryDialogVisible,
    creatingDirectory,
    currentParentNode,
    createDirectoryForm,
    
    // 方法
    handleCreateDirectory,
    resetCreateDirectoryForm,
    handleSubmitCreateDirectory,
    expandCurrentRoutePath,
    checkAndExpandForkedPaths,
    handleCopyLink
  }
}

