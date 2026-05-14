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
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import { createPackage } from '@/architecture/infrastructure/api/service-tree'
import type { ServiceTree as ServiceTreeType, CreateServiceTreeRequest } from '@/types'
import ServiceTreePanel from '@/architecture/presentation/components/ServiceTreePanel.vue'
import { useAuthStore } from '@/architecture/infrastructure/stores/auth'
import { normalizeGoPackageName, validateGoPackageName } from '@/utils/goPackageName'

export function useWorkspaceServiceTree(
  serviceProvider: IServiceProvider = serviceFactory  // 🔥 通过参数注入，提高可测试性
) {
  const route = useRoute()
  const applicationService = serviceProvider.getWorkspaceApplicationService()
  const authStore = useAuthStore()

  // 创建目录对话框状态
  const createDirectoryDialogVisible = ref(false)
  const creatingDirectory = ref(false)
  const currentParentNode = ref<ServiceTreeType | null>(null)
  
  // 获取当前用户名作为默认管理员
  const getDefaultAdmins = () => {
    return authStore.user?.username || ''
  }
  
  const createDirectoryForm = ref<CreateServiceTreeRequest>({
    user: '',
    app: '',
    name: '',
    code: '',
    parent_full_code_path: '',
    description: '',
    tags: '',
    admins: getDefaultAdmins()
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
      parent_full_code_path: parentNode ? parentNode.full_code_path : '',
      description: '',
      tags: '',
      admins: getDefaultAdmins()
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
      parent_full_code_path: '',
      description: '',
      tags: '',
      admins: getDefaultAdmins()
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
    
    const name = createDirectoryForm.value.name.trim()
    const code = normalizeGoPackageName(createDirectoryForm.value.code)

    if (!name || !code) {
      ElNotification.warning({
        title: '提示',
        message: '请输入目录名称和英文标识'
      })
      return
    }
    
    const codeError = validateGoPackageName(code, '目录英文标识')
    if (codeError) {
      ElNotification.warning({
        title: '提示',
        message: codeError
      })
      return
    }

    try {
      creatingDirectory.value = true
      const requestData: CreateServiceTreeRequest = {
        user: currentApp().user,
        app: currentApp().code,
        name,
        code,
        parent_full_code_path: createDirectoryForm.value.parent_full_code_path || '',
        description: createDirectoryForm.value.description || '',
        tags: createDirectoryForm.value.tags || '',
        admins: createDirectoryForm.value.admins || getDefaultAdmins()
      }
      
      // ⭐ 使用新的分离接口
      await createPackage(requestData)
      ElNotification.success({
        title: '成功',
        message: '创建服务目录成功'
      })
      createDirectoryDialogVisible.value = false
      resetCreateDirectoryForm(currentApp)
      
      // 刷新服务目录树，使左侧树展示最新目录（当前应用未变，triggerAppSwitch 会跳过，故用 refreshServiceTree）
      await applicationService.refreshServiceTree()
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
        const serviceTreePanel = serviceTreePanelRef()
        if (serviceTreePanel?.expandPaths) {
          serviceTreePanel.expandPaths([functionPath])
        }
      }, 300)
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
  }
}
