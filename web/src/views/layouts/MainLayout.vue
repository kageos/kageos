<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { InfoFilled } from '@element-plus/icons-vue'
import { getAppList, createApp, updateApp, deleteApp } from '@/api'
import { getServiceTree } from '@/api/service-tree'
import type { App, CreateAppRequest } from '@/types'
import AppSwitcher from '@/components/AppSwitcher.vue'
import type { ServiceTree } from '@/types'

const route = useRoute()
const router = useRouter()

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
  code: '',
  name: ''
})

// 从路由中解析应用信息
const parseAppFromRoute = () => {
  // 直接从 window.location.pathname 获取完整路径（最可靠的方式）
  // 例如：/workspace/luobei/test4/crm/hr -> luobei/test4/crm/hr
  let fullPath = ''
  
  const currentPath = window.location.pathname
  console.log('[MainLayout] window.location.pathname:', currentPath)
  
  if (currentPath.startsWith('/workspace/')) {
    // 从完整路径中提取 workspace 之后的部分
    fullPath = currentPath.replace('/workspace/', '').replace(/^\/+|\/+$/g, '')
  } else {
    // 回退方案：尝试从 route.path 或 route.fullPath 获取
    if (route.path.startsWith('/workspace/')) {
      fullPath = route.path.replace('/workspace/', '').replace(/^\/+|\/+$/g, '')
    } else if (route.fullPath && route.fullPath.startsWith('/workspace/')) {
      fullPath = route.fullPath.split('?')[0].replace('/workspace/', '').replace(/^\/+|\/+$/g, '')
    }
  }
  
  console.log('[MainLayout] route.path:', route.path)
  console.log('[MainLayout] route.fullPath:', route.fullPath)
  console.log('[MainLayout] route.params.path:', route.params.path)
  console.log('[MainLayout] 提取的完整路径:', fullPath)
  
  if (!fullPath) {
    console.log('[MainLayout] 路径为空')
    return null
  }
  
  // 分割路径段，过滤空字符串
  const pathSegments = fullPath.split('/').filter(Boolean)
  console.log('[MainLayout] 路径段:', pathSegments)
  
  if (pathSegments.length < 2) {
    console.log('[MainLayout] 路径段不足（需要至少 user 和 app）')
    return null
  }
  
  const [user, appCode] = pathSegments
  console.log('[MainLayout] 解析出 user:', user, 'appCode:', appCode)
  console.log('[MainLayout] 应用列表:', appList.value.map((a: App) => ({ user: a.user, code: a.code })))
  
  // 从应用列表中找到匹配的应用
  const app = appList.value.find((a: App) => a.user === user && a.code === appCode)
  console.log('[MainLayout] 找到的应用:', app)
  return app || null
}

// 获取应用列表
const fetchAppList = async () => {
  try {
    loadingApps.value = true
    const items = await getAppList()
    appList.value = items
    
    console.log('[MainLayout] ========== 应用列表加载完成 ==========')
    console.log('[MainLayout] 应用数量:', items.length)
    console.log('[MainLayout] 应用列表:', items.map((a: App) => `${a.user}/${a.code}`))
    console.log('[MainLayout] 当前 URL:', window.location.href)
    console.log('[MainLayout] 当前 pathname:', window.location.pathname)
    
    // 尝试从路由中获取应用信息
    const routeApp = parseAppFromRoute()
    if (routeApp) {
      console.log('[MainLayout] ✅ 从路由解析到应用:', routeApp.user + '/' + routeApp.code)
      // 不更新路由，因为路由已经有完整路径
      await switchApp(routeApp, false)
    } else {
      console.log('[MainLayout] ⚠️ 无法从路由解析应用')
      if (!currentApp.value && items.length > 0 && items[0]) {
        // 如果没有路由应用且有应用列表，选择第一个并更新路由
        console.log('[MainLayout] 选择第一个应用:', items[0].user + '/' + items[0].code)
        await switchApp(items[0], true)
      }
    }
  } catch (error) {
    console.error('获取应用列表失败:', error)
    ElMessage.error('获取应用列表失败')
  } finally {
    loadingApps.value = false
  }
}

// 监听路由变化，自动切换应用
watch(() => route.fullPath, async () => {
  const currentPath = window.location.pathname
  console.log('[MainLayout] 路由变化（fullPath）:', route.fullPath, 'pathname:', currentPath)
  // 如果应用列表已加载，尝试根据路由切换应用
  if (appList.value.length > 0 && currentPath.startsWith('/workspace/')) {
    const routeApp = parseAppFromRoute()
    if (routeApp && (!currentApp.value || currentApp.value.id !== routeApp.id)) {
      console.log('[MainLayout] 路由变化，切换应用:', routeApp)
      // 不更新路由，因为路由已经有完整路径
      await switchApp(routeApp, false)
    }
  }
}, { immediate: false })

// 切换应用（可选：是否更新路由）
const switchApp = async (app: App, updateRoute = true) => {
  console.log('[MainLayout] ========== 切换应用 ==========')
  console.log('[MainLayout] 目标应用:', app.user + '/' + app.code)
  console.log('[MainLayout] 是否更新路由:', updateRoute)
  currentApp.value = app
  // 加载服务目录树并发送事件
  await loadServiceTree(app)
  // 发送应用切换事件
  console.log('[MainLayout] 发送 app-switched 事件')
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

// 加载服务目录树
const loadServiceTree = async (app: App) => {
  if (!app || !app.user || !app.code) {
    serviceTree.value = []
    return
  }
  
  try {
    console.log('[MainLayout] 开始加载服务目录树:', app.user + '/' + app.code)
    loadingTree.value = true
    const tree = await getServiceTree(app.user, app.code)
    serviceTree.value = tree || []
    console.log('[MainLayout] 服务目录树加载完成，节点数:', serviceTree.value.length)
    // 发送服务目录树更新事件到Workspace页面
    console.log('[MainLayout] 发送 service-tree-updated 事件')
    window.dispatchEvent(new CustomEvent('service-tree-updated', { detail: { tree: serviceTree.value } }))
  } catch (error) {
    console.error('[MainLayout] 获取服务目录树失败:', error)
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
    name: ''
  }
  createAppDialogVisible.value = true
}

// 提交创建应用
const handleSubmitCreateApp = async () => {
  // 表单验证
  if (!createAppForm.value.name || !createAppForm.value.code) {
    ElMessage.warning('请输入应用名称和代码')
    return
  }
  
  // 验证代码格式（只能包含小写字母、数字和下划线）
  if (!/^[a-z0-9_]+$/.test(createAppForm.value.code)) {
    ElMessage.warning('应用代码只能包含小写字母、数字和下划线')
    return
  }
  
  // 验证代码长度
  if (createAppForm.value.code.length < 2 || createAppForm.value.code.length > 50) {
    ElMessage.warning('应用代码长度必须在 2-50 个字符之间')
    return
  }

  try {
    creatingApp.value = true
    console.log('[MainLayout] 创建应用请求:', createAppForm.value)
    const newApp = await createApp(createAppForm.value)
    console.log('[MainLayout] 应用创建成功:', newApp)
    ElMessage.success('应用创建成功')
    createAppDialogVisible.value = false
    
    // 刷新应用列表
    await fetchAppList()
    
    // 切换到新创建的应用并跳转到工作空间
    if (newApp) {
      console.log('[MainLayout] 跳转到新应用工作空间:', `${newApp.user}/${newApp.code}`)
      currentApp.value = newApp
      // 先跳转路由
      await router.push(`/workspace/${newApp.user}/${newApp.code}`)
      // 然后加载服务目录树
      await loadServiceTree(newApp)
      // 发送应用切换事件
      window.dispatchEvent(new CustomEvent('app-switched', { detail: { app: newApp } }))
    }
  } catch (error: any) {
    console.error('[MainLayout] 创建应用失败:', error)
    const errorMessage = error?.response?.data?.message || error?.message || '创建应用失败'
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
  console.log('[MainLayout] ========== 收到 workspace-ready 事件 ==========')
  console.log('[MainLayout] 当前应用:', currentApp.value ? `${currentApp.value.user}/${currentApp.value.code}` : 'null')
  console.log('[MainLayout] 服务树节点数:', serviceTree.value.length)
  
  if (currentApp.value) {
    // 重新发送应用切换事件
    console.log('[MainLayout] 重新发送 app-switched 事件')
    window.dispatchEvent(new CustomEvent('app-switched', { detail: { app: currentApp.value } }))
    
    // 重新发送服务树更新事件
    if (serviceTree.value.length > 0) {
      console.log('[MainLayout] 重新发送 service-tree-updated 事件')
      window.dispatchEvent(new CustomEvent('service-tree-updated', { detail: { tree: serviceTree.value } }))
    }
  }
}

// 更新应用（重新编译）
const handleUpdateApp = async (app: App) => {
  try {
    console.log('[MainLayout] 开始更新应用:', app.code)
    // 使用 ElMessage.info 显示加载提示，并设置较长的持续时间
    const loadingMessage = ElMessage({
      message: '正在重新编译应用...',
      type: 'info',
      duration: 0, // 不自动关闭
      showClose: false
    })
    
    await updateApp(app.code)
    
    // 关闭加载提示
    loadingMessage.close()
    ElMessage.success('应用更新成功')
    console.log('[MainLayout] 应用更新成功:', app.code)
    
    // 刷新应用列表
    await fetchAppList()
    
    // 如果更新的是当前应用，重新加载服务树
    if (currentApp.value && currentApp.value.code === app.code) {
      await loadServiceTree(currentApp.value)
    }
  } catch (error: any) {
    console.error('[MainLayout] 更新应用失败:', error)
    const errorMessage = error?.response?.data?.message || error?.message || '更新应用失败'
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
    
    console.log('[MainLayout] 开始删除应用:', app.code)
    const loadingMessage = ElMessage({
      message: '正在删除应用...',
      type: 'info',
      duration: 0,
      showClose: false
    })
    
    await deleteApp(app.code)
    
    loadingMessage.close()
    ElMessage.success('应用删除成功')
    console.log('[MainLayout] 应用删除成功:', app.code)
    
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
    const errorMessage = error?.response?.data?.message || error?.message || '删除应用失败'
    ElMessage.error(errorMessage)
  }
}

// 组件挂载时获取应用列表
onMounted(() => {
  // 🔥 如果是测试路由，不加载应用列表
  if (route.path.startsWith('/test/')) {
    console.log('[MainLayout] 测试路由，跳过应用列表加载')
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
      @switch-app="switchApp"
      @create-app="handleCreateApp"
      @update-app="handleUpdateApp"
      @delete-app="handleDeleteApp"
      @load-apps="fetchAppList"
    />

    <!-- 创建应用对话框 -->
    <el-dialog
      v-model="createAppDialogVisible"
      title="创建新应用"
      width="520px"
      :close-on-click-modal="false"
      @close="() => {
        createAppForm = {
          code: '',
          name: ''
        }
      }"
    >
      <el-form :model="createAppForm" label-width="90px">
        <el-form-item label="应用名称" required>
          <el-input
            v-model="createAppForm.name"
            placeholder="请输入应用名称（如：客户管理系统）"
            maxlength="100"
            show-word-limit
            clearable
          />
        </el-form-item>
        <el-form-item label="应用代码" required>
          <el-input
            v-model="createAppForm.code"
            placeholder="请输入应用代码（如：crm）"
            maxlength="50"
            show-word-limit
            clearable
            @input="createAppForm.code = createAppForm.code.toLowerCase()"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            应用代码只能包含小写字母、数字和下划线，长度 2-50 个字符
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
  overflow: hidden;
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