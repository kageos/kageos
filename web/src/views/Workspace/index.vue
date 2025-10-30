<template>
  <div class="workspace-container">
    <div v-loading="loading" class="workspace-layout">
      <!-- 左侧服务目录树 -->
      <div class="left-sidebar">
        <ServiceTreePanel
          :tree-data="serviceTree"
          :loading="loadingTree"
          :current-node-id="currentFunction?.id || null"
          @node-click="handleNodeClick"
          @create-directory="handleCreateDirectory"
          @copy-link="handleCopyLink"
        />
      </div>

      <!-- 中间函数渲染区域 -->
      <div class="function-renderer-container">
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
            />
            
            <!-- Form类型：显示 FormRenderer -->
            <div v-else-if="functionDetail.template_type === 'form'" class="form-container">
              <div class="form-header">
                <h2>{{ currentFunction.name || currentFunction.code }}</h2>
                <p v-if="currentFunction.description" class="form-description">
                  {{ currentFunction.description }}
                </p>
              </div>
              <FormRenderer
                :fields="functionDetail.request || []"
                :response-fields="functionDetail.response || []"
                :method="functionDetail.method"
                :router="functionDetail.router"
                mode="form"
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, ArrowRight, Grid, InfoFilled, Folder } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import ServiceTreePanel from '@/components/ServiceTreePanel.vue'
import TableRenderer from '@/components/TableRenderer.vue'
import FormRenderer from '@/components/FormRenderer.vue'
import { createServiceTree } from '@/api/service-tree'
import { getFunctionDetail } from '@/api/function'
import type { App, ServiceTree, CreateServiceTreeRequest, Function as FunctionType } from '@/types'

const route = useRoute()
const router = useRouter()

// 当前选中的应用（从MainLayout获取）
const currentApp = ref<App | null>(null)
// 服务目录树
const serviceTree = ref<ServiceTree[]>([])
const loadingTree = ref(false)
// 加载状态
const loading = ref(false)
// 当前选中的节点/函数
const currentFunction = ref<ServiceTree | null>(null)
// 函数详情数据
const functionDetail = ref<FunctionType | null>(null)
// 正在加载函数详情
const loadingFunctionDetail = ref(false)
// 当前正在定位的路径（防止重复定位）
const currentLocatingPath = ref<string | null>(null)
// 右侧边栏显示状态
const showRightSidebar = ref(false)
// 当前激活的Tab
const activeTab = computed(() => (route.query.tab as string) || 'run')
// 是否正在加载函数
const isLoadingFunction = ref(false)
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

// 监听应用切换事件
const handleAppSwitch = (event: CustomEvent) => {
  console.log('[Workspace] ========== 应用切换 ==========')
  console.log('[Workspace] 新应用:', event.detail.app?.user + '/' + event.detail.app?.code)
  currentApp.value = event.detail.app
  // 清空当前函数
  currentFunction.value = null
  showRightSidebar.value = false
  console.log('[Workspace] 当前服务树节点数:', serviceTree.value.length)
  // 应用切换后，如果服务树已加载，尝试定位节点
  if (serviceTree.value.length > 0) {
    nextTick(() => {
      console.log('[Workspace] 应用切换后开始定位节点')
      locateNodeByRoute()
    })
  }
}

// 监听服务目录树更新事件
const handleServiceTreeUpdate = (event: CustomEvent) => {
  console.log('[Workspace] ========== 服务目录树更新 ==========')
  console.log('[Workspace] 服务树节点数:', event.detail.tree?.length || 0)
  serviceTree.value = event.detail.tree || []
  loadingTree.value = false
  // 树更新后，等待 DOM 更新，然后根据路由路径定位到对应节点
  nextTick(() => {
    console.log('[Workspace] nextTick 后开始定位节点')
    locateNodeByRoute()
  })
}

// 根据路由路径定位到对应的节点
const locateNodeByRoute = () => {
  // 直接从 window.location.pathname 获取完整路径
  const currentPath = window.location.pathname
  let fullPath = ''
  
  if (currentPath.startsWith('/workspace/')) {
    fullPath = currentPath.replace('/workspace/', '').replace(/^\/+|\/+$/g, '')
  }
  
  console.log('[定位] window.location.pathname:', currentPath)
  console.log('[定位] 提取的完整路径:', fullPath)
  
  if (!fullPath) {
    console.log('[定位] 路径为空，不进行定位')
    currentLocatingPath.value = null
    return
  }
  
  // 如果正在定位同一个路径，跳过
  if (currentLocatingPath.value === fullPath) {
    console.log('[定位] ⏭️ 正在定位此路径，跳过重复定位')
    return
  }
  
  // 分割路径段
  const pathSegments = fullPath.split('/').filter(Boolean)
  console.log('[定位] 路径段:', pathSegments)
  
  if (pathSegments.length < 2) {
    // 至少需要 user 和 app
    console.log('[定位] 路径段不足，需要至少 user 和 app')
    currentLocatingPath.value = null
    return
  }
  
  // 确保当前应用匹配
  const [user, app] = pathSegments
  console.log('[定位] 解析到的 user:', user, 'app:', app)
  console.log('[定位] 当前应用:', currentApp.value ? `${currentApp.value.user}/${currentApp.value.code}` : 'null')
  
  if (!currentApp.value) {
    console.log('[定位] ❌ 当前应用为空，无法定位')
    currentLocatingPath.value = null
    return
  }
  
  if (currentApp.value.user !== user || currentApp.value.code !== app) {
    console.log('[定位] ❌ 应用不匹配')
    console.log('[定位]    期望:', `${user}/${app}`)
    console.log('[定位]    实际:', `${currentApp.value.user}/${currentApp.value.code}`)
    currentLocatingPath.value = null
    return
  }
  
  console.log('[定位] ✅ 应用匹配成功')
  
  // 如果路径长度只有2（只有user和app），说明是应用的根路径，不选中任何节点
  if (pathSegments.length === 2) {
    console.log('[定位] 根路径，不选中任何节点')
    currentFunction.value = null
    showRightSidebar.value = false
    functionDetail.value = null
    currentLocatingPath.value = fullPath
    return
  }
  
  // 查找对应的节点
  const targetPath = `/${pathSegments.join('/')}`
  console.log('[定位] 目标路径:', targetPath)
  
  // 标记正在定位此路径
  currentLocatingPath.value = fullPath
  
  const findNodeByPath = (nodes: ServiceTree[], targetPath: string): ServiceTree | null => {
    for (const node of nodes) {
      console.log('[定位] 检查节点:', node.full_code_path, '===', targetPath, '?', node.full_code_path === targetPath)
      if (node.full_code_path === targetPath) {
        console.log('[定位] ✅ 找到节点:', node)
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
    
    console.log('[定位] ✅✅✅ 定位成功，设置当前节点:', targetNode.name, targetNode.full_code_path)
    console.log('[定位] 节点 ID:', targetNode.id, '类型:', targetNode.type)
    console.log('[定位] 是否相同节点:', isSameNode)
    
    currentFunction.value = targetNode
    
    if (targetNode.type === 'function') {
      showRightSidebar.value = true
      // 如果是函数类型，需要加载函数详情
      // 只有在节点不同，或者还没有加载过详情时才加载
      if (targetNode.ref_id && (!isSameNode || !functionDetail.value)) {
        console.log('[定位] 加载函数详情, ref_id:', targetNode.ref_id)
        loadFunctionDetail(targetNode.ref_id)
      } else {
        console.log('[定位] ⏭️ 跳过重复加载函数详情')
      }
    } else {
      showRightSidebar.value = false
      functionDetail.value = null
    }
  } else {
    console.log('[定位] ❌❌❌ 未找到匹配的节点')
    console.log('[定位] 目标路径:', targetPath)
    console.log('[定位] 服务树节点数:', serviceTree.value.length)
    if (serviceTree.value.length > 0) {
      console.log('[定位] 服务树内容:', JSON.stringify(serviceTree.value.map((n: ServiceTree) => ({ 
        name: n.name, 
        path: n.full_code_path,
        children: n.children?.length || 0
      })), null, 2))
    }
    currentLocatingPath.value = null
  }
}

// 监听刷新服务目录树事件
const handleRefreshServiceTree = () => {
  if (currentApp.value) {
    window.dispatchEvent(new CustomEvent('refresh-service-tree'))
  }
}

// 监听路由变化
watch(() => route.fullPath, () => {
  console.log('[Workspace] ========== 路由变化 ==========')
  console.log('[Workspace] 新路由:', route.fullPath)
  console.log('[Workspace] 当前应用:', currentApp.value ? `${currentApp.value.user}/${currentApp.value.code}` : 'null')
  console.log('[Workspace] 服务树节点数:', serviceTree.value.length)
  if (serviceTree.value.length > 0 && currentApp.value) {
    nextTick(() => {
      console.log('[Workspace] 路由变化后开始定位节点')
      locateNodeByRoute()
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
      locateNodeByRoute()
    })
  } else {
    console.log('[Workspace] ⚠️ 应用变化但条件不满足，不定位节点')
  }
})

// 加载函数详情
const loadFunctionDetail = async (refId: number) => {
  try {
    loadingFunctionDetail.value = true
    console.log('[Workspace] 加载函数详情, ref_id:', refId)
    const detail = await getFunctionDetail(refId)
    console.log('[Workspace] 函数详情:', detail)
    functionDetail.value = detail
  } catch (error) {
    console.error('[Workspace] 加载函数详情失败:', error)
    ElMessage.error('加载函数详情失败')
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
    // 如果是函数，加载函数详情
    showRightSidebar.value = true
    isLoadingFunction.value = true
    
    // 加载函数详情
    if (node.ref_id) {
      await loadFunctionDetail(node.ref_id)
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
    console.log('创建服务目录请求数据:', requestData)
    await createServiceTree(requestData)
    ElMessage.success('创建服务目录成功')
    createDirectoryDialogVisible.value = false
    
    // 刷新服务目录树
    window.dispatchEvent(new CustomEvent('refresh-service-tree'))
  } catch (error: any) {
    console.error('创建服务目录失败:', error)
    const errorMessage = error?.response?.data?.message || error?.message || '创建服务目录失败'
    ElMessage.error(errorMessage)
  } finally {
    creatingDirectory.value = false
  }
}

onMounted(() => {
  console.log('[Workspace] ========== 组件已挂载 ==========')
  window.addEventListener('app-switched', handleAppSwitch as EventListener)
  window.addEventListener('service-tree-updated', handleServiceTreeUpdate as EventListener)
  window.addEventListener('refresh-service-tree', handleRefreshServiceTree as EventListener)
  
  // 组件挂载后，发送"workspace-ready"事件，请求 MainLayout 重新发送当前状态
  console.log('[Workspace] 发送 workspace-ready 事件，请求初始状态')
  window.dispatchEvent(new CustomEvent('workspace-ready'))
  
  // 组件挂载后，检查是否需要定位节点
  // 使用 setTimeout 确保所有初始化事件都已处理
  setTimeout(() => {
    console.log('[Workspace] 组件挂载后检查状态')
    console.log('[Workspace] 当前应用:', currentApp.value ? `${currentApp.value.user}/${currentApp.value.code}` : 'null')
    console.log('[Workspace] 服务树节点数:', serviceTree.value.length)
    console.log('[Workspace] 当前路径:', window.location.pathname)
    
    // 如果有服务树和应用，尝试定位
    if (serviceTree.value.length > 0 && currentApp.value) {
      console.log('[Workspace] 条件满足，开始定位节点')
      nextTick(() => {
        locateNodeByRoute()
      })
    } else {
      console.log('[Workspace] 条件不满足，等待事件')
    }
  }, 200)
})

onUnmounted(() => {
  window.removeEventListener('app-switched', handleAppSwitch as EventListener)
  window.removeEventListener('service-tree-updated', handleServiceTreeUpdate as EventListener)
  window.removeEventListener('refresh-service-tree', handleRefreshServiceTree as EventListener)
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

/* 右侧边栏控制按钮 */
.sidebar-controls {
  position: absolute;
  top: 16px;
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
  max-width: 900px;
  margin: 0 auto;
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
</style>