<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { InfoFilled } from '@element-plus/icons-vue'
import { extractWorkspacePath } from '@/utils/route'
import { Logger } from '@/core/utils/logger'
import { getAppList, createApp, updateApp, deleteApp, getAppDetailByUserAndCode, getAppWithServiceTree } from '@/api'
import type { App, CreateAppRequest } from '@/types'
import AppSwitcher from '@/components/AppSwitcher.vue'
import UserSearchInput from '@/components/UserSearchInput.vue'
import type { ServiceTree } from '@/types'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

// 当前选中的应用
const currentApp = ref<App | null>(null)
// 应用列表
const appList = ref<App[]>([])
// 加载状态
const loadingApps = ref(false)
// 服务目录树（提供给Workspace页面使用）
const serviceTree = ref<ServiceTree[]>([])
const loadingTree = ref(false)

// 创建应用对话框
const createAppDialogVisible = ref(false)
const creatingApp = ref(false)
const createAppForm = ref<CreateAppRequest>({
  is_public: true, // 默认公开
  code: '',
  name: '',
  admins: '' // 管理员列表，逗号分隔的用户名
})

// 管理员数组（用于 UserSearchInput 组件，多选模式返回数组）
const adminsArray = ref<string[]>([])

// 监听 adminsArray 变化，转换为逗号分隔的字符串
watch(adminsArray, (newVal) => {
  createAppForm.value.admins = newVal.length > 0 ? newVal.join(',') : ''
}, { immediate: true })

// 监听对话框显示状态，在打开时初始化管理员
watch(createAppDialogVisible, (isVisible) => {
  if (isVisible) {
    console.log('[Watch] 对话框打开，当前 adminsArray:', adminsArray.value)
    // 对话框打开时，默认添加当前用户为管理员
    const currentUsername = authStore.user?.username
    console.log('[Watch] 当前用户:', currentUsername)
    
    if (currentUsername && adminsArray.value.length === 0) {
      adminsArray.value = [currentUsername]
      console.log('[Watch] 已设置管理员数组:', adminsArray.value)
    }
  }
})

// 从路由中解析应用信息
const parseAppFromRoute = () => {
  // 直接从 window.location.pathname 获取完整路径（最可靠的方式）
  // 例如：/workspace/luobei/test4/crm/hr -> luobei/test4/crm/hr
  let fullPath = ''
  
  const currentPath = window.location.pathname
  
  if (currentPath.startsWith('/workspace/')) {
    // 从完整路径中提取 workspace 之后的部分
    fullPath = extractWorkspacePath(currentPath)
  } else {
    // 回退方案：尝试从 route.path 或 route.fullPath 获取
    if (route.path.startsWith('/workspace/')) {
      fullPath = extractWorkspacePath(route.path)
    } else if (route.fullPath && route.fullPath.startsWith('/workspace/')) {
      fullPath = extractWorkspacePath(route.fullPath.split('?')[0])
    }
  }
  
  if (!fullPath) {
    return null
  }
  
  // 分割路径段，过滤空字符串
  const pathSegments = fullPath.split('/').filter(Boolean)
  
  if (pathSegments.length < 2) {
    return null
  }
  
  const [user, appCode] = pathSegments
  
  // 从应用列表中找到匹配的应用
  const app = appList.value.find((a: App) => a.user === user && a.code === appCode)
  return app || null
}

// 获取应用列表
const fetchAppList = async () => {
  try {
    loadingApps.value = true
    const items = await getAppList()
    appList.value = items
    
    // 尝试从路由中获取应用信息
    const routeApp = parseAppFromRoute()
    if (routeApp) {
      // 不更新路由，因为路由已经有完整路径
      await switchApp(routeApp, false)
    } else {
      if (!currentApp.value && items.length > 0 && items[0]) {
        // 如果没有路由应用且有应用列表，选择第一个并更新路由
        await switchApp(items[0], true)
      }
    }
  } catch (error) {
    console.error('获取应用列表失败:', error)
    ElMessage.error('获取工作空间列表失败')
  } finally {
    loadingApps.value = false
  }
}

// 监听路由变化，自动切换应用
watch(() => route.fullPath, async () => {
  const currentPath = window.location.pathname
  // 如果应用列表已加载，尝试根据路由切换应用
  if (appList.value.length > 0 && currentPath.startsWith('/workspace/')) {
    const routeApp = parseAppFromRoute()
    if (routeApp && (!currentApp.value || currentApp.value.id !== routeApp.id)) {
      // 不更新路由，因为路由已经有完整路径
      await switchApp(routeApp, false)
    }
  }
}, { immediate: false })

// 切换应用（可选：是否更新路由）
const switchApp = async (app: App, updateRoute = true) => {
  currentApp.value = app
  // 加载服务目录树并发送事件
  await loadServiceTree(app)
  // 发送应用切换事件
  window.dispatchEvent(new CustomEvent('app-switched', { detail: { app } }))
  // 只有在需要更新路由且当前路由不匹配时才更新
  if (updateRoute) {
    const currentPath = route.params.path
    let shouldUpdate = false
    
    // 检查路由是否匹配当前应用
    if (!currentPath) {
      shouldUpdate = true
    } else {
      let pathSegments: string[] = []
      if (Array.isArray(currentPath)) {
        pathSegments = currentPath as string[]
      } else if (typeof currentPath === 'string' && currentPath) {
        pathSegments = currentPath.split('/').filter(Boolean)
      }
      
      // 如果路由的前两个段（user/app）不匹配，或者路由为空，才更新
      if (pathSegments.length < 2 || pathSegments[0] !== app.user || pathSegments[1] !== app.code) {
        shouldUpdate = true
      }
    }
    
    if (shouldUpdate) {
      // 如果路由中没有完整路径信息，更新到应用的根路径
      router.push(`/workspace/${app.user}/${app.code}`)
    }
  }
}

// 加载服务目录树（使用合并接口，同时获取应用详情和服务目录树）
const loadServiceTree = async (app: App) => {
  if (!app || !app.user || !app.code) {
    serviceTree.value = []
    return
  }
  
  try {
    loadingTree.value = true
    
    // ⭐ 使用合并接口获取应用详情和服务目录树（减少请求次数）
    // 传递 user 和 app，而不是只传 code
    const workspaceData = await getAppWithServiceTree(app.user, app.code)
    
    if (workspaceData && workspaceData.app && workspaceData.service_tree) {
      // 更新应用详情（确保是最新的）
      const updatedApp = workspaceData.app
      const appIndex = appList.value.findIndex(a => a.id === updatedApp.id)
      if (appIndex >= 0) {
        appList.value[appIndex] = updatedApp
      }
      
      // 如果当前应用是更新的应用，更新 currentApp
      if (currentApp.value && currentApp.value.id === updatedApp.id) {
        currentApp.value = updatedApp
      }
      
      // 设置服务目录树
      serviceTree.value = workspaceData.service_tree || []
      
      // 发送服务目录树更新事件到Workspace页面
      window.dispatchEvent(new CustomEvent('service-tree-updated', { detail: { tree: serviceTree.value } }))
    }
  } catch (error) {
    console.error('[MainLayout] 获取工作空间数据失败:', error)
    ElMessage.error('获取服务目录树失败')
    serviceTree.value = []
  } finally {
    loadingTree.value = false
  }
}

// 打开创建应用对话框
const handleCreateApp = () => {
  createAppForm.value = {
    code: '',
    name: '',
    is_public: true,
    admins: ''
  }
  
  // ⭐ 默认把当前登录用户添加到管理员列表
  const currentUsername = authStore.user?.username
  console.log('[创建工作空间] 当前用户:', currentUsername)
  console.log('[创建工作空间] authStore.user:', authStore.user)
  
  if (currentUsername) {
    adminsArray.value = [currentUsername]
    console.log('[创建工作空间] 已设置管理员数组:', adminsArray.value)
  } else {
    adminsArray.value = []
    console.log('[创建工作空间] 警告：当前用户为空，无法设置默认管理员')
  }
  
  createAppDialogVisible.value = true
}

// 提交创建应用
const handleSubmitCreateApp = async () => {
  // 表单验证
  if (!createAppForm.value.name || !createAppForm.value.code) {
    ElMessage.warning('请输入名称和英文标识')
    return
  }
  
  // 验证代码格式（只能包含小写字母、数字和下划线）
  if (!/^[a-z0-9_]+$/.test(createAppForm.value.code)) {
    ElMessage.warning('英文标识只能包含小写字母、数字和下划线')
    return
  }
  
  // 验证代码长度
  if (createAppForm.value.code.length < 2 || createAppForm.value.code.length > 50) {
    ElMessage.warning('英文标识长度必须在 2-50 个字符之间')
    return
  }

  try {
    creatingApp.value = true
    const createResponse = await createApp(createAppForm.value)
    ElMessage.success('工作空间创建成功')
    createAppDialogVisible.value = false
    
    // ⭐ 使用创建响应中的信息获取工作空间详情和服务目录树（合并接口，减少请求次数）
    if (createResponse && createResponse.user && createResponse.app) {
      try {
        // ⭐ 使用合并接口获取工作空间详情和服务目录树
        // 传递 user 和 app，而不是只传 code
        const workspaceData = await getAppWithServiceTree(createResponse.user, createResponse.app)
        
        if (workspaceData && workspaceData.app && workspaceData.app.user && workspaceData.app.code) {
          const newApp = workspaceData.app
          
          // 将新应用添加到列表（如果不在列表中的话）
          const existsInList = appList.value.some(a => a.id === newApp.id)
          if (!existsInList) {
            appList.value.push(newApp)
          }
          
      currentApp.value = newApp
          
          // 设置服务目录树（从合并接口获取）
          serviceTree.value = workspaceData.service_tree || []
          
      // 先跳转路由
      await router.push(`/workspace/${newApp.user}/${newApp.code}`)
          
          // 发送服务目录树更新事件
          window.dispatchEvent(new CustomEvent('service-tree-updated', { detail: { tree: serviceTree.value } }))
          
      // 发送应用切换事件
      window.dispatchEvent(new CustomEvent('app-switched', { detail: { app: newApp } }))
        } else {
          // 如果获取详情失败，使用创建响应中的信息直接跳转
          await router.push(`/workspace/${createResponse.user}/${createResponse.app}`)
        }
      } catch (error) {
        // 如果获取详情失败，使用创建响应中的信息直接跳转
        console.error('[MainLayout] 获取工作空间数据失败:', error)
        await router.push(`/workspace/${createResponse.user}/${createResponse.app}`)
      }
    }
  } catch (error: any) {
    console.error('[MainLayout] 创建应用失败:', error)
    // 🔥 统一使用 msg 字段
    const errorMessage = error?.response?.data?.msg || error?.message || '创建工作空间失败'
    ElMessage.error(errorMessage)
  } finally {
    creatingApp.value = false
  }
}

// 监听刷新服务目录树事件
const handleRefreshServiceTree = () => {
  if (currentApp.value) {
    loadServiceTree(currentApp.value)
  }
}

// 监听 Workspace 组件就绪事件，重新发送当前状态
const handleWorkspaceReady = () => {
  if (currentApp.value) {
    // 重新发送应用切换事件
    window.dispatchEvent(new CustomEvent('app-switched', { detail: { app: currentApp.value } }))
    
    // 重新发送服务树更新事件
    if (serviceTree.value.length > 0) {
      window.dispatchEvent(new CustomEvent('service-tree-updated', { detail: { tree: serviceTree.value } }))
    }
  }
}

// 更新应用（重新编译）
const handleUpdateApp = async (app: App) => {
  try {
    // 使用 ElMessage.info 显示加载提示，并设置较长的持续时间
    const loadingMessage = ElMessage({
      message: '正在重新编译工作空间...',
      type: 'info',
      duration: 0, // 不自动关闭
      showClose: false
    })
    
    await updateApp(app.code)
    
    // 关闭加载提示
    loadingMessage.close()
    ElMessage.success('应用更新成功')
    
    // 刷新应用列表
    await fetchAppList()
    
    // 如果更新的是当前应用，重新加载服务树
    if (currentApp.value && currentApp.value.code === app.code) {
      await loadServiceTree(currentApp.value)
    }
  } catch (error: any) {
    console.error('[MainLayout] 更新应用失败:', error)
      // 🔥 统一使用 msg 字段
      const errorMessage = error?.response?.data?.msg || error?.message || '更新应用失败'
    ElMessage.error(errorMessage)
  }
}

// 删除应用
const handleDeleteApp = async (app: App) => {
  try {
    // 确认对话框
    await ElMessageBox.confirm(
      `确定要删除应用「${app.name || app.code}」吗？此操作不可恢复。`,
      '删除应用',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'el-button--danger'
      }
    )
    
    const loadingMessage = ElMessage({
      message: '正在删除应用...',
      type: 'info',
      duration: 0,
      showClose: false
    })
    
    await deleteApp(app.code)
    
    loadingMessage.close()
    ElMessage.success('应用删除成功')
    
    // 如果删除的是当前应用，需要切换到其他应用
    if (currentApp.value && currentApp.value.code === app.code) {
      // 刷新应用列表
      await fetchAppList()
      
      // 切换到第一个应用（如果还有应用）
      if (appList.value.length > 0 && appList.value[0]) {
        await switchApp(appList.value[0], true)
      } else {
        // 没有其他应用了，清空当前应用
        currentApp.value = null
        serviceTree.value = []
        router.push('/workspace')
      }
    } else {
      // 只刷新应用列表
      await fetchAppList()
    }
  } catch (error: any) {
    // 用户取消删除不报错
    if (error === 'cancel') {
      return
    }
    console.error('[MainLayout] 删除应用失败:', error)
      // 🔥 统一使用 msg 字段
      const errorMessage = error?.response?.data?.msg || error?.message || '删除应用失败'
    ElMessage.error(errorMessage)
  }
}

// 组件挂载时获取应用列表
onMounted(() => {
  // 🔥 如果是测试路由，不加载应用列表
  if (route.path.startsWith('/test/')) {
    return
  }
  
  fetchAppList()
  window.addEventListener('refresh-service-tree', handleRefreshServiceTree as EventListener)
  window.addEventListener('workspace-ready', handleWorkspaceReady as EventListener)
})

onUnmounted(() => {
  window.removeEventListener('refresh-service-tree', handleRefreshServiceTree as EventListener)
  window.removeEventListener('workspace-ready', handleWorkspaceReady as EventListener)
})
</script>

<template>
  <div class="main-layout">
    <!-- 主内容区 -->
    <main class="main-layout__content">
      <router-view />
    </main>

    <!-- 应用切换器（底部固定） -->
    <AppSwitcher
      :current-app="currentApp"
      :app-list="appList"
      :loading-apps="loadingApps"
      :service-tree="serviceTree"
      @switch-app="switchApp"
      @create-app="handleCreateApp"
      @update-app="handleUpdateApp"
      @delete-app="handleDeleteApp"
      @load-apps="fetchAppList"
    />

    <!-- 创建工作空间对话框 -->
    <el-dialog
      v-model="createAppDialogVisible"
      title="创建新工作空间"
      width="800px"
      :close-on-click-modal="false"
      @close="() => {
        console.log('[对话框关闭] 重置表单')
        createAppForm.value = {
          code: '',
          name: '',
          is_public: true,
          admins: ''
        }
        adminsArray.value = []
      }"
    >
      <el-form :model="createAppForm" label-width="90px">
        <el-form-item label="名称" required>
          <el-input
            v-model="createAppForm.name"
            placeholder="请输入名称（如：清北大学、首都市政府、xxx图书馆、xxx医院、xxx银行、xxx科技公司）"
            maxlength="100"
            show-word-limit
            clearable
          />
        </el-form-item>
        <el-form-item label="英文标识" required>
          <el-input
            v-model="createAppForm.code"
            placeholder="请输入英文标识（如：tsinghua、pku_gsm）"
            maxlength="50"
            show-word-limit
            clearable
            @input="createAppForm.code = createAppForm.code.toLowerCase()"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            英文标识只能包含小写字母、数字和下划线，长度 2-50 个字符
          </div>
        </el-form-item>
        <el-form-item label="是否公开">
          <el-switch
            v-model="createAppForm.is_public"
            active-text="公开"
            inactive-text="私有"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            公开的工作空间可以被其他用户搜索到，私有的工作空间只有您自己可以看到
          </div>
        </el-form-item>
        <el-form-item label="管理员">
          <UserSearchInput
            v-model="adminsArray"
            placeholder="搜索并选择管理员（可多选）"
            :multiple="true"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            可以设置多个管理员，管理员拥有工作空间的管理权限
          </div>
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createAppDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmitCreateApp" :loading="creatingApp">
            创建
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.main-layout {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
}

.main-layout__content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 6px;
  display: flex;
  align-items: center;
  gap: 4px;
}
</style>
