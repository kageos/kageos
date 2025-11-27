<template>
  <div class="workspace-container">
    <div v-loading="loading" class="workspace-layout">
      <!-- 左侧服务目录树 -->
      <div class="left-sidebar">
        <ServiceTreePanel
          ref="serviceTreePanelRef"
          :tree-data="serviceTree"
          :loading="loadingTree"
          :current-node-id="currentFunction?.id || null"
          :current-function="currentFunction"
          @node-click="handleNodeClick"
          @create-directory="handleCreateDirectory"
          @copy-link="handleCopyLink"
          @fork-group="handleForkGroup"
        />
      </div>

      <!-- 中间函数渲染区域 -->
      <div class="function-renderer-container">
        <!-- 顶部工具栏 -->
        <div class="top-toolbar">
          <!-- 左侧：应用信息 -->
          <div class="left-section">
            <span v-if="currentApp" class="app-info">
              {{ currentApp.name }}
            </span>
          </div>
          
          <!-- 右侧：工具按钮 -->
          <div class="right-section">
            <!-- PWA 安装按钮 -->
            <el-tooltip
              v-if="showInstallButton"
              content="下载到桌面"
              placement="bottom"
            >
              <el-button
                circle
                @click="handleInstall"
                class="install-button"
              >
                <el-icon><Download /></el-icon>
              </el-button>
            </el-tooltip>
            
            <!-- 主题切换按钮 -->
            <ThemeToggle />
            
            <!-- 用户菜单 -->
            <el-dropdown
              v-if="isAuthenticated"
              trigger="click"
              placement="bottom-end"
              @command="handleUserCommand"
              class="user-menu-dropdown"
            >
              <div class="user-info">
                <el-avatar
                  :size="32"
                  :src="userAvatar"
                  class="user-avatar"
                >
                  <el-icon><User /></el-icon>
                </el-avatar>
                <span class="user-name">{{ userName || '用户' }}</span>
                <el-icon class="dropdown-icon"><ArrowDown /></el-icon>
              </div>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item disabled>
                    <div class="user-menu-header">
                      <div class="user-menu-name">{{ userName || '用户' }}</div>
                      <div class="user-menu-email">{{ userEmail || '' }}</div>
                    </div>
                  </el-dropdown-item>
                  <el-dropdown-item command="settings">
                    <el-icon><Setting /></el-icon>
                    <span>个人设置</span>
                  </el-dropdown-item>
                  <el-dropdown-item divided command="logout">
                    <el-icon><SwitchButton /></el-icon>
                    <span>退出登录</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            
            <!-- 未登录时显示登录按钮 -->
            <el-button
              v-else
              type="primary"
              @click="handleLogin"
              class="login-button"
            >
              登录
            </el-button>
          </div>
        </div>

        <!-- 右侧边栏控制按钮 -->
        <div class="sidebar-controls" v-if="currentFunction">
          <div class="right-controls">
            <el-button
              v-if="!showRightSidebar"
              link
              @click="toggleRightSidebar"
              class="sidebar-toggle"
              title="显示函数信息"
            >
              <el-icon><ArrowLeft /></el-icon>
              显示函数信息
            </el-button>
            
            <el-button
              v-if="showRightSidebar"
              link
              @click="toggleRightSidebar"
              class="sidebar-toggle"
              title="隐藏函数信息"
            >
              <el-icon><ArrowRight /></el-icon>
              隐藏函数信息
            </el-button>
          </div>
        </div>

        <!-- Loading 状态 -->
        <div v-if="isLoadingFunction" class="loading-container" v-loading="true" element-loading-text="正在加载函数详情...">
          <div style="height: 400px;"></div>
        </div>
        
        <!-- 根据状态显示不同内容 -->
        <template v-else-if="activeTab === 'create' && currentFunction">
          <!-- Create Tab：新增页面 -->
          <div class="form-page">
            <div class="form-page-header">
              <el-button @click="backToList" :icon="ArrowLeft">返回列表</el-button>
              <h2 class="form-page-title">新增数据</h2>
            </div>
            <div class="form-page-content">
              <!-- TODO: FormRenderer组件 -->
              <el-empty description="FormRenderer待实现" />
            </div>
            <div class="form-page-footer">
              <el-button @click="backToList">取消</el-button>
              <el-button type="primary">提交</el-button>
            </div>
          </div>
        </template>
        
        <template v-else-if="activeTab === 'edit' && currentFunction">
          <!-- Edit Tab：编辑页面 -->
          <div class="form-page">
            <div class="form-page-header">
              <el-button @click="backToList" :icon="ArrowLeft">返回列表</el-button>
              <h2 class="form-page-title">编辑数据</h2>
            </div>
            <div class="form-page-content">
              <!-- TODO: FormRenderer组件 -->
              <el-empty description="FormRenderer待实现" />
            </div>
            <div class="form-page-footer">
              <el-button @click="backToList">取消</el-button>
              <el-button type="primary">保存</el-button>
            </div>
          </div>
        </template>
        
        <template v-else-if="currentFunction && currentFunction.type === 'function' && functionDetail">
          <!-- Function 类型：显示函数渲染器 -->
          <div class="function-renderer-content">
            <!-- Table类型：显示 TableRenderer -->
            <TableRenderer
              v-if="functionDetail.template_type === 'table'"
              :function-data="functionDetail"
              :current-function="currentFunction"
            />
            
            <!-- Form类型：显示 FormRenderer（新架构） -->
            <div v-else-if="functionDetail.template_type === 'form'" class="form-container">
              <FormRenderer
                :function-detail="functionDetail"
                :initial-data="formInitialData"
              />
            </div>
            
            <!-- 其他类型：待实现 -->
            <div v-else class="function-info-placeholder">
              <h3>{{ currentFunction.name || currentFunction.code }}</h3>
              <p>{{ currentFunction.description || '函数详情' }}</p>
              <el-empty :description="`${functionDetail.template_type} 类型渲染器待实现`" />
            </div>
          </div>
        </template>
        
        <template v-else-if="currentFunction && currentFunction.type === 'function' && !functionDetail && !isLoadingFunction">
          <!-- Function 类型但函数详情加载失败 -->
          <div class="function-renderer-content">
            <el-empty description="加载函数详情失败" />
          </div>
        </template>
        
        <template v-else-if="currentFunction && currentFunction.type === 'package'">
          <!-- Package 类型：显示包概览 -->
          <div class="package-page">
            <div class="package-header">
              <h2>{{ currentFunction.name || currentFunction.code }}</h2>
              <p v-if="currentFunction.description" class="package-description">
                {{ currentFunction.description }}
              </p>
            </div>
            <div class="package-content">
              <el-empty description="请从左侧选择一个函数查看详情" :image-size="120">
                <template #image>
                  <el-icon :size="120" color="var(--el-text-color-placeholder)">
                    <Folder />
                  </el-icon>
                </template>
              </el-empty>
            </div>
          </div>
        </template>
        
        <template v-else>
          <!-- Welcome Info：欢迎信息 -->
          <div class="welcome-info">
            <el-empty description="请从左侧服务目录树中选择一个节点" :image-size="120">
              <template #image>
                <el-icon :size="120" color="var(--el-text-color-placeholder)">
                  <Grid />
                </el-icon>
              </template>
              <p class="welcome-tip">选择一个函数或包以开始工作</p>
            </el-empty>
          </div>
        </template>
      </div>

      <!-- 右侧函数信息面板 -->
      <div 
        v-if="currentFunction && showRightSidebar" 
        class="right-sidebar"
      >
        <div class="function-info-panel">
          <h3>函数信息</h3>
          <div class="info-section">
            <div class="info-item">
              <span class="label">名称：</span>
              <span class="value">{{ currentFunction.name || currentFunction.code }}</span>
            </div>
            <div class="info-item">
              <span class="label">代码：</span>
              <span class="value">{{ currentFunction.code }}</span>
            </div>
            <div class="info-item">
              <span class="label">类型：</span>
              <span class="value">{{ currentFunction.type }}</span>
            </div>
            <div class="info-item">
              <span class="label">路径：</span>
              <span class="value">{{ currentFunction.full_code_path }}</span>
            </div>
            <div v-if="currentFunction.description" class="info-item">
              <span class="label">描述：</span>
              <span class="value">{{ currentFunction.description }}</span>
            </div>
          </div>
          <!-- TODO: FunctionInfoPanel组件 -->
        </div>
      </div>
    </div>
    
    <!-- 创建服务目录对话框 -->
    <el-dialog
      v-model="createDirectoryDialogVisible"
      :title="currentParentNode ? `在「${currentParentNode.name || currentParentNode.code}」下创建服务目录` : '创建服务目录'"
      width="520px"
      :close-on-click-modal="false"
      @close="() => {
        createDirectoryForm = {
          user: currentApp?.user || '',
          app: currentApp?.code || '',
          name: '',
          code: '',
          parent_id: 0,
          description: '',
          tags: ''
        }
        currentParentNode = null
      }"
    >
      <el-form :model="createDirectoryForm" label-width="90px">
        <el-form-item label="目录名称" required>
          <el-input
            v-model="createDirectoryForm.name"
            placeholder="请输入目录名称（如：用户管理）"
            maxlength="50"
            show-word-limit
            clearable
          />
        </el-form-item>
        <el-form-item label="目录代码" required>
          <el-input
            v-model="createDirectoryForm.code"
            placeholder="请输入目录代码，如：user"
            maxlength="50"
            show-word-limit
            clearable
            @input="createDirectoryForm.code = createDirectoryForm.code.toLowerCase()"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            目录代码只能包含小写字母、数字和下划线
          </div>
        </el-form-item>
        <el-form-item label="描述">
          <el-input
            v-model="createDirectoryForm.description"
            type="textarea"
            :rows="3"
            placeholder="请输入目录描述（可选）"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>
        <el-form-item label="标签">
          <el-input
            v-model="createDirectoryForm.tags"
            placeholder="请输入标签，多个标签用逗号分隔（可选）"
            maxlength="100"
            clearable
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="createDirectoryDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmitCreateDirectory" :loading="creatingDirectory">
            创建
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 🔥 应用切换器（底部固定） -->
    <AppSwitcher
      :current-app="currentApp"
      :app-list="appList"
      :loading-apps="loadingApps"
      @switch-app="switchApp"
      @create-app="showCreateAppDialog"
      @update-app="handleUpdateApp"
      @delete-app="handleDeleteApp"
      @load-apps="loadAppList"
    />

    <!-- 创建应用对话框 -->
    <el-dialog
      v-model="createAppDialogVisible"
      title="创建新应用"
      width="520px"
      :close-on-click-modal="false"
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
          <el-button type="primary" @click="submitCreateApp" :loading="creatingApp">
            创建
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- Fork 函数组对话框 -->
    <FunctionForkDialog
      v-model="forkDialogVisible"
      :source-full-group-code="forkSourceGroupCode || undefined"
      :source-group-name="forkSourceGroupName || undefined"
      :current-app="currentApp || undefined"
      @success="handleForkSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, ArrowRight, Grid, InfoFilled, Folder, User, ArrowDown, SwitchButton, Setting, Download } from '@element-plus/icons-vue'
import { ElMessage, ElDialog, ElForm, ElFormItem, ElInput, ElButton, ElIcon, ElAvatar, ElDropdown, ElDropdownMenu, ElDropdownItem, ElTooltip } from 'element-plus'
import ServiceTreePanel from '@/components/ServiceTreePanel.vue'
import TableRenderer from '@/components/TableRenderer.vue'
import FormRenderer from '@/core/renderers-v2/FormRenderer.vue'
import AppSwitcher from '@/components/AppSwitcher.vue'
import ThemeToggle from '@/components/ThemeToggle.vue'
import FunctionForkDialog from '@/components/FunctionForkDialog.vue'
import { getFunctionDetail, getFunctionByPath } from '@/api/function'
import { createServiceTree } from '@/api/service-tree'
import { useAppManager } from '@/composables/useAppManager'
import { useServiceTree } from '@/composables/useServiceTree'
import { usePWAInstall } from '@/composables/usePWAInstall'
import { useAuthStore } from '@/stores/auth'
import { Logger } from '@/core/utils/logger'
import type { ServiceTree, CreateServiceTreeRequest, CreateAppRequest, Function as FunctionType } from '@/types'

const route = useRoute()
const router = useRouter()

// 用户认证
const authStore = useAuthStore()

// 🔥 使用 Composables（组件化逻辑）
const {
  currentApp,
  appList,
  loading: loadingApps,
  loadAppList,
  parseAppFromRoute,
  switchApp: switchToApp,
  handleCreateApp,
  handleUpdateApp,
  handleDeleteApp
} = useAppManager()

const {
  serviceTree,
  loading: loadingTree,
  currentNode: currentFunction,
  loadServiceTree: loadServiceTreeData,
  locateNodeByRoute,
  handleCreateDirectory: createDirectory
} = useServiceTree()

// PWA 安装功能
const { showInstallButton, install: installPWA } = usePWAInstall()

// ServiceTreePanel 的引用
const serviceTreePanelRef = ref<InstanceType<typeof ServiceTreePanel> | null>(null)

// 加载状态
const loading = ref(false)
// 函数详情数据
const functionDetail = ref<FunctionType | null>(null)
// 正在加载函数详情
const loadingFunctionDetail = ref(false)

// 创建应用对话框
const createAppDialogVisible = ref(false)
const creatingApp = ref(false)
const createAppForm = ref<CreateAppRequest>({
  code: '',
  name: ''
})
// 当前正在定位的路径（防止重复定位）
const currentLocatingPath = ref<string | null>(null)
// 右侧边栏显示状态
const showRightSidebar = ref(false)
// 当前激活的Tab
const activeTab = computed(() => (route.query.tab as string) || 'run')
// 是否正在加载函数
const isLoadingFunction = ref(false)

// 🔥 从 URL 查询参数中提取表单初始数据
const formInitialData = computed(() => {
  const initialData: Record<string, any> = {}
  const query = route.query
  
  // 遍历所有查询参数，如果字段在 request 中，添加到 initialData
  if (functionDetail.value?.request) {
    functionDetail.value.request.forEach((field: any) => {
      const fieldCode = field.code
      if (query[fieldCode] !== undefined && query[fieldCode] !== null && query[fieldCode] !== '') {
        const value = query[fieldCode]
        // 🔥 类型转换：根据字段类型转换值
        if (field.data?.type === 'int' || field.data?.type === 'integer') {
          const intValue = parseInt(String(value), 10)
          if (!isNaN(intValue)) {
            initialData[fieldCode] = intValue
          }
        } else if (field.data?.type === 'float' || field.data?.type === 'number') {
          const floatValue = parseFloat(String(value))
          if (!isNaN(floatValue)) {
            initialData[fieldCode] = floatValue
          }
        } else if (field.data?.type === 'bool' || field.data?.type === 'boolean') {
          initialData[fieldCode] = value === 'true' || value === '1' || value === 1 || value === true
        } else {
          initialData[fieldCode] = value
        }
      }
    })
  }
  
  return initialData
})

// 创建目录对话框
const createDirectoryDialogVisible = ref(false)
const creatingDirectory = ref(false)
const createDirectoryForm = ref<CreateServiceTreeRequest>({
  user: '',
  app: '',
  name: '',
  code: '',
  parent_id: 0,
  description: '',
  tags: ''
})

// 🔥 初始化：加载应用列表并切换应用
const initializeWorkspace = async () => {
  const items = await loadAppList()
  
  // 尝试从路由解析应用
  const app = parseAppFromRoute()
  if (app) {
    // 🔥 保留当前路径（刷新时保持 URL 不变）
    await switchApp(app, true)
  } else if (items.length > 0) {
    // 没有应用路径时，切换到第一个应用
    await switchApp(items[0], false)
  }
}

// 🔥 切换应用（封装 Composable 的方法，添加额外逻辑）
const switchApp = async (app: any, preserveRoute = false) => {
  currentFunction.value = null
  showRightSidebar.value = false
  
  // 🔥 如果 preserveRoute 为 true，保留当前路径（用于刷新时）
  // 否则更新路由到应用根路径
  if (preserveRoute) {
    // 只更新 currentApp，不更新路由
    await switchToApp(app, false)
  } else {
    // 正常切换应用，更新路由
    await switchToApp(app, true)
  }
  
  // 加载服务树
  await loadServiceTreeData(app)
  
  // 🔥 定位节点并加载函数详情（使用 handleLocateNode，它会加载函数详情）
  nextTick(() => {
    handleLocateNode()
    // 应用切换完成、服务树加载完成后，检查 forked 参数
    checkAndExpandForkedPaths()
  })
}

// 🔥 显示创建应用对话框
const showCreateAppDialog = () => {
  createAppForm.value = {
    code: '',
    name: ''
  }
  createAppDialogVisible.value = true
}

// 🔥 提交创建应用
const submitCreateApp = async () => {
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
    const newApp = await handleCreateApp(createAppForm.value)
    
    if (newApp) {
      createAppDialogVisible.value = false
      
      // 切换到新创建的应用
      await switchApp(newApp)
    }
  } catch (error: any) {
    Logger.error('Workspace', '创建应用失败', error)
  } finally {
    creatingApp.value = false
  }
}

// 根据路由路径定位到对应的节点（简化版，调用 Composable）
const handleLocateNode = () => {
  const currentPath = window.location.pathname
  let fullPath = ''
  
  if (currentPath.startsWith('/workspace/')) {
    fullPath = currentPath.replace('/workspace/', '').replace(/^\/+|\/+$/g, '')
  }
  
  if (!fullPath) {
    currentLocatingPath.value = null
    return
  }
  
  // 如果正在定位同一个路径，跳过
  if (currentLocatingPath.value === fullPath) {
    return
  }
  
  // 分割路径段
  const pathSegments = fullPath.split('/').filter(Boolean)
  
  if (pathSegments.length < 2) {
    // 至少需要 user 和 app
    currentLocatingPath.value = null
    return
  }
  
  // 确保当前应用匹配
  const [user, app] = pathSegments
  
  if (!currentApp.value) {
    currentLocatingPath.value = null
    return
  }
  
  if (currentApp.value.user !== user || currentApp.value.code !== app) {
    currentLocatingPath.value = null
    return
  }
  
  // 如果路径长度只有2（只有user和app），说明是应用的根路径，不选中任何节点
  if (pathSegments.length === 2) {
    currentFunction.value = null
    showRightSidebar.value = false
    functionDetail.value = null
    currentLocatingPath.value = fullPath
    return
  }
  
  // 查找对应的节点
  const targetPath = `/${pathSegments.join('/')}`
  
  // 标记正在定位此路径
  currentLocatingPath.value = fullPath
  
  const findNodeByPath = (nodes: ServiceTree[], targetPath: string): ServiceTree | null => {
    for (const node of nodes) {
      if (node.full_code_path === targetPath) {
        return node
      }
      if (node.children && node.children.length > 0) {
        const found = findNodeByPath(node.children, targetPath)
        if (found) return found
      }
    }
    return null
  }
  
  const targetNode = findNodeByPath(serviceTree.value, targetPath)
  if (targetNode) {
    // 如果节点相同且已经加载过详情，不重复加载
    const isSameNode = currentFunction.value?.id === targetNode.id
    
    currentFunction.value = targetNode
    
    if (targetNode.type === 'function') {
      // 默认不展开右侧边栏，让用户需要时再手动展开
      // showRightSidebar.value = true
      // 如果是函数类型，需要加载函数详情
      // 只有在节点不同，或者还没有加载过详情时才加载
      if (!isSameNode || !functionDetail.value) {
        // 🔥 优先使用 ref_id，如果没有则使用 full_code_path
        if (targetNode.ref_id && targetNode.ref_id > 0) {
          loadFunctionDetail(targetNode.ref_id)
        } else if (targetNode.full_code_path) {
          loadFunctionDetailByPath(targetNode.full_code_path)
        } else {
          Logger.warn('Workspace', '节点没有 ref_id 和 full_code_path，无法加载函数详情')
          ElMessage.warning('无法加载函数详情：节点信息不完整')
        }
      }
    } else {
      showRightSidebar.value = false
      functionDetail.value = null
    }
  } else {
    currentLocatingPath.value = null
  }
}

// 监听刷新服务目录树事件
const handleRefreshServiceTree = async () => {
  if (currentApp.value) {
    console.log('[Workspace] 刷新服务目录树:', currentApp.value.user + '/' + currentApp.value.code)
    // 重新加载服务树数据
    await loadServiceTreeData(currentApp.value)
    // 刷新后重新定位节点
    nextTick(() => {
      handleLocateNode()
    })
  }
}

// 检查并展开 forked 路径
const checkAndExpandForkedPaths = () => {
  const forkedParam = route.query.forked as string
  console.log('[Workspace] 检查 forked 参数:', forkedParam)
  console.log('[Workspace] 当前应用:', currentApp.value ? `${currentApp.value.user}/${currentApp.value.code}` : 'null')
  console.log('[Workspace] serviceTree 长度:', serviceTree.value.length)
  console.log('[Workspace] serviceTreePanelRef:', serviceTreePanelRef.value)
  
  // 检查当前应用是否匹配 URL 中的应用
  const pathSegments = route.path.replace('/workspace/', '').split('/').filter(Boolean)
  if (pathSegments.length >= 2) {
    const [urlUser, urlApp] = pathSegments
    if (currentApp.value && (currentApp.value.user !== urlUser || currentApp.value.code !== urlApp)) {
      console.log('[Workspace] ⚠️ 应用不匹配，等待应用切换完成')
      console.log('[Workspace]    URL 应用:', `${urlUser}/${urlApp}`)
      console.log('[Workspace]    当前应用:', `${currentApp.value.user}/${currentApp.value.code}`)
      return // 应用不匹配，不展开
    }
  }
  
  if (forkedParam && serviceTree.value.length > 0 && serviceTreePanelRef.value && currentApp.value) {
    const forkedPaths = decodeURIComponent(forkedParam).split(',').filter(Boolean)
    console.log('[Workspace] 解析后的路径列表:', forkedPaths)
    
    // 验证路径是否属于当前应用
    const validPaths = forkedPaths.filter(path => {
      const pathMatch = path.match(/^\/([^/]+)\/([^/]+)/)
      if (pathMatch) {
        const [, pathUser, pathApp] = pathMatch
        const isValid = pathUser === currentApp.value?.user && pathApp === currentApp.value?.code
        if (!isValid) {
          console.log('[Workspace] ⚠️ 路径不属于当前应用，跳过:', path)
        }
        return isValid
      }
      return false
    })
    
    if (validPaths.length > 0) {
      console.log('[Workspace] 有效路径列表:', validPaths)
      nextTick(() => {
        setTimeout(() => {
          if (serviceTreePanelRef.value && serviceTreePanelRef.value.expandPaths) {
            console.log('[Workspace] 调用 expandPaths')
            serviceTreePanelRef.value.expandPaths(validPaths)
          } else {
            console.log('[Workspace] ⚠️ serviceTreePanelRef 或 expandPaths 不存在')
          }
        }, 500) // 延迟确保树完全渲染
      })
    } else {
      console.log('[Workspace] ⚠️ 没有有效的路径可以展开')
    }
  }
}

// 监听路由变化
watch(() => route.fullPath, async () => {
  console.log('[Workspace] ========== 路由变化 ==========')
  console.log('[Workspace] 新路由:', route.fullPath)
  console.log('[Workspace] 当前应用:', currentApp.value ? `${currentApp.value.user}/${currentApp.value.code}` : 'null')
  console.log('[Workspace] 服务树节点数:', serviceTree.value.length)
  
  // 从路由解析应用
  const app = parseAppFromRoute()
  if (app) {
    // 如果应用不匹配，需要切换应用
    if (!currentApp.value || currentApp.value.id !== app.id) {
      console.log('[Workspace] 路由变化检测到应用不匹配，切换应用')
      console.log('[Workspace]    URL 应用:', `${app.user}/${app.code}`)
      console.log('[Workspace]    当前应用:', currentApp.value ? `${currentApp.value.user}/${currentApp.value.code}` : 'null')
      // 切换应用（保留路由，因为路由已经变化了）
      await switchApp(app, true)
      // switchApp 完成后会自动检查 forked 参数
      return
    }
  }
  
  if (serviceTree.value.length > 0 && currentApp.value) {
    nextTick(() => {
      console.log('[Workspace] 路由变化后开始定位节点')
      handleLocateNode()  // 🔥 使用 handleLocateNode，它会加载函数详情
      // 注意：不在这里检查 forked 参数，因为应用可能还没切换完成
      // forked 参数会在应用切换完成、服务树加载完成后检查
    })
  } else {
    console.log('[Workspace] ⚠️ 路由变化但条件不满足，不定位节点')
    console.log('[Workspace]    服务树是否为空:', serviceTree.value.length === 0)
    console.log('[Workspace]    当前应用是否为空:', !currentApp.value)
  }
}, { immediate: false })

// 监听当前应用变化
watch(currentApp, () => {
  console.log('[Workspace] ========== 当前应用变化 ==========')
  console.log('[Workspace] 新应用:', currentApp.value ? `${currentApp.value.user}/${currentApp.value.code}` : 'null')
  console.log('[Workspace] 服务树节点数:', serviceTree.value.length)
  if (serviceTree.value.length > 0 && currentApp.value) {
    nextTick(() => {
      console.log('[Workspace] 应用变化后开始定位节点')
      handleLocateNode()  // 🔥 使用 handleLocateNode，它会加载函数详情
      // 检查 forked 参数
      checkAndExpandForkedPaths()
    })
  } else {
    console.log('[Workspace] ⚠️ 应用变化但条件不满足，不定位节点')
  }
})

// 监听服务树变化，检查 forked 参数
watch(() => serviceTree.value.length, (newLength: number) => {
  if (newLength > 0 && currentApp.value && route.query.forked) {
    console.log('[Workspace] 服务树加载完成，检查 forked 参数')
    checkAndExpandForkedPaths()
  }
})

// 监听应用切换事件（从 MainLayout 或其他组件发送）
const handleAppSwitched = async (event: CustomEvent) => {
  const app = event.detail?.app
  if (app && appList.value.length > 0) {
    console.log('[Workspace] ========== 收到 app-switched 事件 ==========')
    console.log('[Workspace] 目标应用:', app.user + '/' + app.code)
    
    // 从应用列表中找到对应的应用对象（确保使用最新的应用数据）
    const targetApp = appList.value.find((a: App) => a.id === app.id || (a.user === app.user && a.code === app.code))
    if (targetApp) {
      console.log('[Workspace] 找到目标应用，切换应用')
      // 使用 switchApp 方法切换应用（这会更新 currentApp 并加载服务树）
      await switchApp(targetApp, true) // preserveRoute = true，因为路由已经跳转了
    } else {
      console.log('[Workspace] ⚠️ 未找到目标应用，尝试使用事件中的应用对象')
      // 如果找不到，直接使用事件中的应用对象
      await switchToApp(app, false) // 不更新路由，因为路由已经跳转了
      await loadServiceTreeData(app)
    }
  }
}

// 加载函数详情（通过 ref_id）
const loadFunctionDetail = async (refId: number) => {
  try {
    loadingFunctionDetail.value = true
    console.log('[Workspace] 加载函数详情, ref_id:', refId)
    const detail = await getFunctionDetail(refId)
    console.log('[Workspace] 函数详情:', detail)
    functionDetail.value = detail
  } catch (error: any) {
    console.error('[Workspace] 加载函数详情失败:', error)
    const errorMessage = error?.response?.data?.message || error?.message || '加载函数详情失败'
    console.error('[Workspace] 错误详情:', errorMessage)
    ElMessage.error(errorMessage)
    functionDetail.value = null
  } finally {
    loadingFunctionDetail.value = false
  }
}

// 🔥 加载函数详情（通过路径，作为备选方案）
const loadFunctionDetailByPath = async (fullCodePath: string) => {
  try {
    loadingFunctionDetail.value = true
    console.log('[Workspace] 通过路径加载函数详情:', fullCodePath)
    const detail = await getFunctionByPath(fullCodePath)
    console.log('[Workspace] 函数详情:', detail)
    functionDetail.value = detail
  } catch (error: any) {
    console.error('[Workspace] 通过路径加载函数详情失败:', error)
    const errorMessage = error?.response?.data?.message || error?.message || '加载函数详情失败'
    console.error('[Workspace] 错误详情:', errorMessage)
    ElMessage.error(errorMessage)
    functionDetail.value = null
  } finally {
    loadingFunctionDetail.value = false
  }
}

// 处理服务目录节点点击
const handleNodeClick = async (node: ServiceTree) => {
  console.log('点击节点:', node)
  currentFunction.value = node
  
  // 更新路由到当前节点的路径
  if (node.full_code_path) {
    // full_code_path格式: /user/app/path...
    // 去掉开头的 /，作为路由路径
    const path = node.full_code_path.substring(1)
    router.push(`/workspace/${path}`)
  }
  
  if (node.type === 'function') {
    // 如果是函数，加载函数详情，但默认不展开右侧边栏
    // showRightSidebar.value = true  // 注释掉，让用户需要时手动展开
    isLoadingFunction.value = true
    
    // 🔥 加载函数详情（优先使用 ref_id，否则使用路径）
    if (node.ref_id && node.ref_id > 0) {
      await loadFunctionDetail(node.ref_id)
    } else if (node.full_code_path) {
      await loadFunctionDetailByPath(node.full_code_path)
    } else {
      console.warn('[Workspace] ⚠️ 节点没有 ref_id 和 full_code_path，无法加载函数详情')
      ElMessage.warning('无法加载函数详情：节点信息不完整')
    }
    
    isLoadingFunction.value = false
  } else {
    // 如果是包，隐藏右侧边栏，清空函数详情
    showRightSidebar.value = false
    functionDetail.value = null
  }
}

// 切换右侧边栏
const toggleRightSidebar = () => {
  showRightSidebar.value = !showRightSidebar.value
}

// 用户相关
const isAuthenticated = computed(() => authStore.isAuthenticated)
const userEmail = computed(() => authStore.userEmail || authStore.user?.email || '')
const userAvatar = computed(() => authStore.user?.avatar || '')

// 用户显示名称：username(昵称) 或 username
const userName = computed(() => {
  const user = authStore.user
  if (!user) return '用户'
  const username = user.username || ''
  const nickname = user.nickname || ''
  if (nickname) {
    return `${username}(${nickname})`
  }
  return username
})

// 处理用户菜单命令
const handleUserCommand = async (command: string) => {
  if (command === 'logout') {
    try {
      await authStore.logout()
    } catch (error) {
      console.error('登出失败:', error)
    }
  } else if (command === 'settings') {
    router.push('/user/settings')
  }
}

// 跳转到登录页
const handleLogin = () => {
  router.push('/login')
}

// 处理PWA安装
const handleInstall = async () => {
  const success = await installPWA()
  if (success) {
    ElMessage.success('应用已成功安装到桌面')
  } else {
    ElMessage.info('安装已取消')
  }
}

// 返回列表
const backToList = () => {
  router.push({ query: { ...route.query, tab: 'run' } })
  currentFunction.value = null
  showRightSidebar.value = false
}

// 当前创建目录的父节点
const currentParentNode = ref<ServiceTree | null>(null)

// 打开创建目录对话框（可选择父节点）
const handleCreateDirectory = (parentNode?: ServiceTree) => {
  if (!currentApp.value) {
    ElMessage.warning('请先选择一个应用')
    return
  }
  currentParentNode.value = parentNode || null
  createDirectoryForm.value = {
    user: currentApp.value.user,
    app: currentApp.value.code,
    name: '',
    code: '',
    parent_id: parentNode ? Number(parentNode.id) : 0,
    description: '',
    tags: ''
  }
  createDirectoryDialogVisible.value = true
}

// 复制链接
const handleCopyLink = (node: ServiceTree) => {
  const link = `${window.location.origin}${window.location.pathname}?node=${node.id}`
  navigator.clipboard.writeText(link).then(() => {
    ElMessage.success('链接已复制到剪贴板')
  }).catch(() => {
    ElMessage.error('复制链接失败')
  })
}

// Fork 函数组
const forkDialogVisible = ref(false)
const forkSourceGroupCode = ref('')
const forkSourceGroupName = ref('')

const handleForkGroup = (node: ServiceTree | null) => {
  // 如果传入了节点，使用它；否则打开对话框让用户选择
  if (node) {
    if (!node.full_group_code) {
      ElMessage.warning('该节点没有函数组代码，无法克隆')
      return
    }
    forkSourceGroupCode.value = node.full_group_code
    forkSourceGroupName.value = node.group_name || node.name || ''
  } else {
    // 没有传入节点，清空预设值，让用户在对话框中选择
    forkSourceGroupCode.value = ''
    forkSourceGroupName.value = ''
  }
  forkDialogVisible.value = true
}

// Fork 成功后的回调
const handleForkSuccess = () => {
  // 刷新服务目录树
  if (currentApp.value) {
    loadServiceTreeData(currentApp.value)
  }
  ElMessage.success('克隆完成！请刷新页面查看新功能')
}

// 提交创建目录
const handleSubmitCreateDirectory = async () => {
  if (!currentApp.value) {
    ElMessage.warning('请先选择一个应用')
    return
  }
  
  if (!createDirectoryForm.value.name || !createDirectoryForm.value.code) {
    ElMessage.warning('请输入目录名称和代码')
    return
  }
  
  // 验证代码格式
  if (!/^[a-z0-9_]+$/.test(createDirectoryForm.value.code)) {
    ElMessage.warning('目录代码只能包含小写字母、数字和下划线')
    return
  }

  // 🔥 确保当前应用信息完整
  if (!currentApp.value.user || !currentApp.value.code) {
    ElMessage.warning('当前应用信息不完整，请重新选择应用')
    console.error('[Workspace] 当前应用信息不完整:', {
      currentApp: currentApp.value,
      user: currentApp.value?.user,
      code: currentApp.value?.code
    })
    return
  }

  try {
    creatingDirectory.value = true
    // 确保使用当前应用的信息
    const requestData: CreateServiceTreeRequest = {
      user: currentApp.value.user,
      app: currentApp.value.code,
      name: createDirectoryForm.value.name,
      code: createDirectoryForm.value.code,
      parent_id: createDirectoryForm.value.parent_id || 0,
      description: createDirectoryForm.value.description || '',
      tags: createDirectoryForm.value.tags || ''
    }
    console.log('[Workspace] 创建服务目录请求数据:', requestData)
    console.log('[Workspace] 当前应用信息:', {
      id: currentApp.value.id,
      user: currentApp.value.user,
      code: currentApp.value.code,
      name: currentApp.value.name
    })
    
    await createServiceTree(requestData)
    ElMessage.success('创建服务目录成功')
    createDirectoryDialogVisible.value = false
    
    // 刷新服务目录树
    window.dispatchEvent(new CustomEvent('refresh-service-tree'))
  } catch (error: any) {
    console.error('[Workspace] 创建服务目录失败:', error)
    console.error('[Workspace] 错误详情:', error?.response?.data)
    const errorMessage = error?.response?.data?.msg || error?.response?.data?.message || error?.message || '创建服务目录失败'
    ElMessage.error(errorMessage)
  } finally {
    creatingDirectory.value = false
  }
}

onMounted(() => {
  console.log('[Workspace] ========== 组件已挂载 ==========')
  
  // 🔥 初始化 Workspace
  initializeWorkspace()
  
  // 保留刷新服务树事件（用于其他地方触发刷新）
  window.addEventListener('refresh-service-tree', handleRefreshServiceTree as EventListener)
  // 监听应用切换事件
  window.addEventListener('app-switched', handleAppSwitched as EventListener)
  
  // 组件挂载后，检查是否需要定位节点
  // 使用 setTimeout 确保所有初始化事件都已处理
  setTimeout(() => {
    console.log('[Workspace] 组件挂载后检查状态')
    console.log('[Workspace] 当前应用:', currentApp.value ? `${currentApp.value.user}/${currentApp.value.code}` : 'null')
    console.log('[Workspace] 服务树节点数:', serviceTree.value.length)
    console.log('[Workspace] 当前路径:', window.location.pathname)
    
    // 检查 URL 参数中是否有新克隆的路径
    checkAndExpandForkedPaths()
    
    // 如果有服务树和应用，尝试定位
    if (serviceTree.value.length > 0 && currentApp.value) {
      console.log('[Workspace] 条件满足，开始定位节点')
      nextTick(() => {
        handleLocateNode()  // 🔥 使用 handleLocateNode，它会加载函数详情
      })
    } else {
      console.log('[Workspace] 条件不满足，等待事件')
    }
  }, 200)
})

onUnmounted(() => {
  window.removeEventListener('refresh-service-tree', handleRefreshServiceTree as EventListener)
  window.removeEventListener('app-switched', handleAppSwitched as EventListener)
})
</script>

<style scoped>
.workspace-container {
  width: 100%;
  height: 100%;
  overflow: hidden;
}

.workspace-layout {
  display: flex;
  height: 100%;
  width: 100%;
}

/* 左侧边栏 */
.left-sidebar {
  width: 300px;
  flex-shrink: 0;
  overflow: hidden;
  border-right: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}

/* 中间函数渲染区域 */
.function-renderer-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--el-bg-color-page);
  position: relative;
}

/* 顶部工具栏 */
.top-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-light);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.top-toolbar .left-section {
  display: flex;
  align-items: center;
  gap: 12px;
}

.top-toolbar .app-info {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.top-toolbar .right-section {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* 用户菜单 */
.user-menu-dropdown {
  cursor: pointer;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px;
  border-radius: 20px;
  transition: background-color 0.2s;
}

.user-info:hover {
  background-color: var(--el-fill-color-light);
}

.user-avatar {
  flex-shrink: 0;
}

.user-name {
  font-size: 14px;
  color: var(--el-text-color-primary);
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dropdown-icon {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  transition: transform 0.2s;
}

.user-menu-dropdown.is-open .dropdown-icon {
  transform: rotate(180deg);
}

.login-button {
  font-size: 14px;
}

/* 用户菜单下拉项 */
.user-menu-header {
  padding: 4px 0;
  min-width: 160px;
}

.user-menu-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
}

.user-menu-email {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.el-dropdown-menu__item[disabled] {
  cursor: default;
  opacity: 1;
}

/* 右侧边栏控制按钮 */
.sidebar-controls {
  position: absolute;
  top: 70px;
  right: 16px;
  z-index: 100;
}

.right-controls {
  display: flex;
  gap: 8px;
}

.sidebar-toggle {
  padding: 8px 12px;
}

/* 加载容器 */
.loading-container {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

/* 表单页面 */
.form-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 24px;
  overflow-y: auto;
}

.form-page-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-light);
}

.form-page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.form-page-content {
  flex: 1;
  overflow-y: auto;
}

.form-page-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-light);
  margin-top: 24px;
}

/* 函数渲染内容 */
.function-renderer-content {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
  overflow-x: visible;
  position: relative;
  z-index: 1;
}

.function-info-placeholder {
  max-width: 800px;
  margin: 0 auto;
  text-align: center;
  padding: 40px 20px;
}

.function-info-placeholder h3 {
  font-size: 24px;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
}

.function-info-placeholder p {
  font-size: 14px;
  color: var(--el-text-color-regular);
  margin-bottom: 24px;
}

/* 包页面 */
.package-page {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
}

.package-page {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.package-header {
  padding: 24px;
  border-bottom: 1px solid var(--el-border-color-light);
}

.package-header h2 {
  font-size: 24px;
  color: var(--el-text-color-primary);
  margin: 0 0 12px 0;
}

.package-description {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  margin: 0;
}

.package-content {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}

/* 欢迎信息 */
.welcome-info {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}

.welcome-tip {
  margin-top: 16px;
  color: var(--el-text-color-secondary);
  font-size: 14px;
}

/* 右侧边栏 */
.right-sidebar {
  width: 350px;
  flex-shrink: 0;
  overflow-y: auto;
  border-left: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}

.function-info-panel {
  padding: 24px;
}

.function-info-panel h3 {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--el-border-color-light);
}

.info-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-item .label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-weight: 500;
}

.info-item .value {
  font-size: 14px;
  color: var(--el-text-color-primary);
  word-break: break-all;
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 6px;
  display: flex;
  align-items: center;
  gap: 4px;
}

/* Form 容器样式 */
.form-container {
  width: 100%;
  max-width: 100%;
  padding: 0 20px;
}

.form-header {
  padding: 24px 24px 16px;
  border-bottom: 1px solid var(--el-border-color-light);
  margin-bottom: 24px;
}

.form-header h2 {
  font-size: 24px;
  color: var(--el-text-color-primary);
  margin: 0 0 12px 0;
  font-weight: 600;
}

.form-description {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  margin: 0;
  line-height: 1.6;
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