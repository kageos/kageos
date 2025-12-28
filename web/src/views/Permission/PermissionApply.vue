<template>
  <div class="permission-apply-wrapper">
    <div class="permission-apply">
      <el-card shadow="hover" class="apply-card">
      <template #header>
        <div class="card-header">
          <h2>权限申请</h2>
        </div>
      </template>

      <div v-if="loading" class="loading-container">
        <el-skeleton :rows="5" animated />
      </div>

      <div v-else-if="error" class="error-container">
        <el-alert
          :title="error"
          type="error"
          :closable="false"
          show-icon
        />
      </div>

      <div v-else class="apply-content">
        <div class="apply-layout">
          <!-- 左侧：资源树选择 -->
          <div class="apply-sidebar">
            <el-card shadow="never" class="tree-card">
              <template #header>
                <h3>选择资源</h3>
              </template>
              <div class="tree-container">
                <el-tree
                  ref="treeRef"
                  :data="serviceTree"
                  :props="treeProps"
                  :default-expand-all="true"
                  :expand-on-click-node="true"
                  :highlight-current="true"
                  node-key="full_code_path"
                  :default-expanded-keys="defaultExpandedKeys"
                  :current-node-key="selectedResourcePath"
                  @node-click="handleTreeNodeClick"
                  class="resource-tree"
                >
                  <template #default="{ node, data }">
                    <span class="tree-node" :class="{ 'is-selected': selectedResourcePath === data.full_code_path }">
                      <!-- app 类型：显示工作空间图标 -->
                      <img 
                        v-if="data.type === 'app'" 
                        src="/service-tree/app-copy.svg" 
                        alt="工作空间" 
                        class="node-icon app-icon-img"
                        :class="getNodeIconClass(data)"
                      />
                      <!-- package 类型：统一使用目录图标 -->
                      <img 
                        v-else-if="data.type === 'package'" 
                        src="/service-tree/custom-folder.svg" 
                        alt="目录" 
                        class="node-icon package-icon-img"
                        :class="getNodeIconClass(data)"
                      />
                      <!-- function 类型：根据 template_type 显示不同图标 -->
                      <template v-else-if="data.type === 'function'">
                        <!-- 表单类型：使用编辑图标 -->
                        <img 
                          v-if="data.template_type === TEMPLATE_TYPE.FORM"
                          src="/service-tree/编辑.svg" 
                          alt="表单" 
                          class="node-icon form-icon-img"
                          :class="getNodeIconClass(data)"
                        />
                        <!-- 其他类型：使用组件图标 -->
                        <component 
                          v-else 
                          :is="getFunctionIcon(data)"
                          class="node-icon"
                          :class="getNodeIconClass(data)"
                        />
                      </template>
                      <!-- 其他类型：显示 fx 文本 -->
                      <span v-else class="node-icon fx-icon" :class="getNodeIconClass(data)">fx</span>
                      <span class="node-label">{{ node.label }}</span>
                      
                      <!-- 节点元信息：只显示已选择的权限提示 -->
                      <div class="node-meta">
                        <el-tag
                          v-if="getNodePermissionDisplayText(data.full_code_path)"
                          size="small"
                          :type="getNodePermissionTagType(data.full_code_path)"
                          class="permission-hint-tag"
                        >
                          {{ getNodePermissionDisplayText(data.full_code_path) }}
                        </el-tag>
                      </div>
                    </span>
                  </template>
                </el-tree>
              </div>
            </el-card>
          </div>

          <!-- 中间：权限选择区域 -->
          <div class="apply-main">
            <div v-if="currentScope" class="permission-scopes">
              <div class="scope-header-main">
                <div class="scope-title-main">
                    <el-icon><Document /></el-icon>
                  <span class="scope-name-main">{{ currentScope.displayName }}</span>
                  <!-- 显示已选择的权限提示 -->
                  <div v-if="selectedPermissions.length > 0" class="selected-permissions-display">
                    <el-tag
                      v-for="action in selectedPermissions"
                      :key="action"
                      size="small"
                      type="success"
                      class="selected-permission-tag"
                    >
                      {{ getPermissionDisplayName(action) }}
                    </el-tag>
                  </div>
                  <el-tag 
                    v-else
                    size="small" 
                    :type="currentScope.resourceType === 'function' ? 'primary' : currentScope.resourceType === 'directory' ? 'success' : currentScope.resourceType === 'app' ? 'warning' : 'info'"
                  >
                    {{ currentScope.resourceType === 'function' ? '函数' : currentScope.resourceType === 'directory' ? '目录' : currentScope.resourceType === 'app' ? '工作空间' : '应用' }}
                    </el-tag>
                  </div>
                  <el-button 
                  v-if="currentScope.quickSelect"
                    type="primary" 
                    size="small"
                  @click="handleQuickSelect"
                  >
                  {{ currentScope.quickSelect.label }}
                  </el-button>
                </div>
                
              <div class="scope-path-main">
                <code>{{ currentScope.resourcePath }}</code>
                </div>
                
              <div class="permission-list">
                <div class="permission-list-header">
                  <h4 class="permission-list-title">可申请的权限</h4>
                  <el-alert
                    type="info"
                    :closable="false"
                    show-icon
                    class="permission-tip"
                  >
                    <template #default>
                      <div class="tip-content">
                        <p class="tip-text">💡 <strong>默认已选择最小权限</strong>，如需完整权限，请选择下方的"所有权权限"</p>
                        <p class="tip-text">📋 权限会自动继承给子资源，选择父目录权限后，子目录和子函数会自动获得相应权限</p>
                      </div>
                    </template>
                  </el-alert>
                </div>
                
                <!-- 小权限（具体操作权限） -->
                <div v-if="getSmallPermissions().length > 0" class="permission-section small-permissions">
                <el-checkbox-group 
                    v-model="selectedPermissions"
                  class="permission-checkbox-group"
                    @change="handlePermissionChange"
                >
                  <el-checkbox
                      v-for="permission in getSmallPermissions()"
                    :key="permission.action"
                    :label="permission.action"
                    class="permission-checkbox"
                    :class="{ 
                      'has-existing-selected': hasExistingPermission(permission.action) && selectedPermissions.includes(permission.action),
                      'has-new-selected': !hasExistingPermission(permission.action) && selectedPermissions.includes(permission.action),
                      'has-existing-unselected': hasExistingPermission(permission.action) && !selectedPermissions.includes(permission.action)
                    }"
                  >
                    <div class="permission-option">
                      <div class="permission-header">
                      <span class="permission-name">{{ permission.displayName }}</span>
                      <div class="permission-tags">
                      <el-tag 
                          v-if="hasExistingPermission(permission.action)" 
                          size="small" 
                          type="success" 
                          class="existing-tag"
                        >
                          已有权限
                        </el-tag>
                        <el-tag 
                          v-if="!hasExistingPermission(permission.action) && selectedPermissions.includes(permission.action)" 
                          size="small" 
                          type="primary" 
                          class="new-selected-tag"
                        >
                          新选
                        </el-tag>
                        <el-tag 
                          v-if="permission.isMinimal && !hasExistingPermission(permission.action)" 
                        size="small" 
                        type="info" 
                        class="minimal-tag"
                      >
                          默认选择
                      </el-tag>
                      </div>
                      </div>
                      <p class="permission-description">
                        {{ getPermissionDescription(permission.action, currentScope?.resourceType, currentScope?.resourceType === 'function' ? (findNodeInTree(serviceTree, currentScope?.resourcePath || '')?.template_type) : undefined).description }}
                      </p>
                      <div v-if="getPermissionDescription(permission.action, currentScope?.resourceType, currentScope?.resourceType === 'function' ? (findNodeInTree(serviceTree, currentScope?.resourcePath || '')?.template_type) : undefined).inheritance" class="permission-inheritance">
                        <el-icon class="inheritance-icon"><Folder /></el-icon>
                        <span class="inheritance-text">{{ getPermissionDescription(permission.action, currentScope?.resourceType, currentScope?.resourceType === 'function' ? (findNodeInTree(serviceTree, currentScope?.resourcePath || '')?.template_type) : undefined).inheritance }}</span>
                      </div>
                      <code class="permission-code">{{ permission.action }}</code>
                    </div>
                  </el-checkbox>
                </el-checkbox-group>
              </div>
                
                <!-- 分隔线 -->
                <el-divider v-if="getSmallPermissions().length > 0 && getManagePermissions().length > 0" />
                
                <!-- 大权限（所有权/管理权限） -->
                <div v-if="getManagePermissions().length > 0" class="permission-section manage-permissions">
                  <div class="manage-permissions-header">
                    <el-icon><Lock /></el-icon>
                    <span class="manage-permissions-title">所有权权限</span>
                    <el-tag size="small" type="warning" class="manage-tag">最完整权限</el-tag>
                  </div>
                  <el-alert
                    type="warning"
                    :closable="false"
                    show-icon
                    class="manage-alert"
                  >
                    <template #default>
                      <div class="alert-content">
                        <p class="alert-text"><strong>选择所有权后，将自动获得该资源的所有操作权限</strong>，无需再单独选择其他权限</p>
                        <p class="alert-text">所有权会自动继承给所有子资源，<strong>子目录和子函数都会获得完整权限</strong></p>
                      </div>
                    </template>
                  </el-alert>
                  <el-checkbox-group 
                    v-model="selectedPermissions"
                    class="permission-checkbox-group"
                    @change="handlePermissionChange"
                  >
                    <el-checkbox
                      v-for="permission in getManagePermissions()"
                      :key="permission.action"
                      :label="permission.action"
                      class="permission-checkbox manage-checkbox"
                      :class="{ 
                        'has-existing-selected': hasExistingPermission(permission.action) && selectedPermissions.includes(permission.action),
                        'has-new-selected': !hasExistingPermission(permission.action) && selectedPermissions.includes(permission.action),
                        'has-existing-unselected': hasExistingPermission(permission.action) && !selectedPermissions.includes(permission.action)
                      }"
                    >
                      <div class="permission-option">
                        <div class="permission-header">
                          <span class="permission-name">{{ permission.displayName }}</span>
                          <div class="permission-tags">
                            <el-tag 
                              v-if="hasExistingPermission(permission.action)" 
                              size="small" 
                              type="success" 
                              class="existing-tag"
                            >
                              已有权限
                            </el-tag>
                            <el-tag 
                              v-if="!hasExistingPermission(permission.action) && selectedPermissions.includes(permission.action)" 
                              size="small" 
                              type="primary" 
                              class="new-selected-tag"
                            >
                              新选
                            </el-tag>
                          </div>
                        </div>
                        <p class="permission-description">
                          {{ getPermissionDescription(permission.action, currentScope?.resourceType, currentScope?.resourceType === 'function' ? (findNodeInTree(serviceTree, currentScope?.resourcePath || '')?.template_type) : undefined).description }}
                        </p>
                        <div v-if="getPermissionDescription(permission.action, currentScope?.resourceType, currentScope?.resourceType === 'function' ? (findNodeInTree(serviceTree, currentScope?.resourcePath || '')?.template_type) : undefined).inheritance" class="permission-inheritance">
                          <el-icon class="inheritance-icon"><Folder /></el-icon>
                          <span class="inheritance-text">{{ getPermissionDescription(permission.action, currentScope?.resourceType, currentScope?.resourceType === 'function' ? (findNodeInTree(serviceTree, currentScope?.resourcePath || '')?.template_type) : undefined).inheritance }}</span>
                        </div>
                        <code class="permission-code">{{ permission.action }}</code>
                      </div>
                    </el-checkbox>
                  </el-checkbox-group>
                </div>
              </div>
            </div>
            <div v-else class="empty-state">
              <el-empty description="请从左侧树中选择一个资源" />
            </div>
          </div>

          <!-- 右侧：申请表单 -->
          <div class="apply-sidebar-right">
            <el-card shadow="never" class="form-card">
              <template #header>
                <h3>提交申请</h3>
              </template>
              <el-form
                ref="formRef"
                :model="formData"
                :rules="rules"
                label-width="80px"
                class="apply-form"
              >
                <el-form-item label="申请理由" prop="reason">
                  <el-input
                    v-model="formData.reason"
                    type="textarea"
                    :rows="6"
                    placeholder="请填写申请权限的理由，以便管理员审核"
                    maxlength="500"
                    show-word-limit
                  />
                </el-form-item>

                <el-form-item>
                  <el-button
                    type="primary"
                    :loading="submitting"
                    @click="handleSubmit"
                    style="width: 100%"
                  >
                    提交申请
                  </el-button>
                  <el-button 
                    @click="handleCancel"
                    style="width: 100%; margin-top: 12px"
                  >
                    取消
                  </el-button>
                </el-form-item>
              </el-form>
            </el-card>
          </div>
        </div>
      </div>
    </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElText, ElIcon, ElTree, ElDivider } from 'element-plus'
import { Document, Folder, Lock } from '@element-plus/icons-vue'
import ChartIcon from '@/components/icons/ChartIcon.vue'
import TableIcon from '@/components/icons/TableIcon.vue'
import FormIcon from '@/components/icons/FormIcon.vue'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import { 
  getPermissionDisplayName, 
  getPermissionScopes,
  getAvailablePermissions,
  getPermissionDescription,
  type PermissionScope
} from '@/utils/permission'
import { applyPermission, getWorkspacePermissions } from '@/api/permission'
import type { FormInstance, FormRules } from 'element-plus'
import { getAppWithServiceTree } from '@/api/app'
import { useAuthStore } from '@/stores/auth'
import type { ServiceTree, App } from '@/types'

const route = useRoute()
const router = useRouter()

// 权限信息
const permissionInfo = ref({
  resource_path: '',
  action: '',
  action_display: '',
  error_message: '',
})

// 加载状态
const loading = ref(true)
const error = ref('')
const submitting = ref(false)

// 服务树数据（包含工作空间节点）
const serviceTree = ref<ServiceTree[]>([])
const treeRef = ref<InstanceType<typeof ElTree>>()
const treeProps = {
  children: 'children',
  label: 'name'
}

// 当前工作空间信息
const currentApp = ref<App | null>(null)

// 当前选中的资源路径
const selectedResourcePath = ref<string>('')
const defaultExpandedKeys = ref<string[]>([])

// 当前选中资源的权限范围
const currentScope = ref<PermissionScope | null>(null)

// 当前资源选中的权限点
const selectedPermissions = ref<string[]>([])

// 所有资源的权限选择状态（用于级联选择）
// key: resourcePath, value: 该资源已选择的权限列表
const allResourcePermissions = ref<Map<string, string[]>>(new Map())

// 所有资源的已有权限（从后端获取）
// key: resourcePath, value: 该资源已有的权限（action -> hasPermission）
const existingPermissions = ref<Map<string, Record<string, boolean>>>(new Map())

// 表单数据
const formRef = ref<FormInstance>()
const formData = ref({
  reason: '',
})

// 表单验证规则
const rules: FormRules = {
  reason: [
    { required: true, message: '请填写申请理由', trigger: 'blur' },
    { min: 10, message: '申请理由至少需要10个字符', trigger: 'blur' },
  ],
}

// 检查是否至少选择了一个权限
const hasSelectedPermissions = computed(() => {
  return selectedPermissions.value.length > 0
})

// 获取函数图标组件（根据 template_type）
const getFunctionIcon = (data: ServiceTree) => {
  if (data.template_type === TEMPLATE_TYPE.TABLE) {
    return TableIcon
  } else if (data.template_type === TEMPLATE_TYPE.FORM) {
    return FormIcon
  } else if (data.template_type === TEMPLATE_TYPE.CHART) {
    return ChartIcon
  }
  // 默认使用 Document 图标（如果没有 template_type 或不是已知类型）
  return Document
}

// 获取节点图标样式类
const getNodeIconClass = (data: ServiceTree) => {
  if (data.type === 'app') {
    return 'app-icon'
  } else if (data.type === 'package') {
    return 'package-icon'
  } else if (data.type === 'function') {
    // 根据 template_type 返回不同的样式类
    if (data.template_type === TEMPLATE_TYPE.TABLE) {
      return 'table-icon'
    } else if (data.template_type === TEMPLATE_TYPE.FORM) {
      return 'form-icon'
    } else if (data.template_type === TEMPLATE_TYPE.CHART) {
      return 'chart-icon'
    }
    return 'function-icon'
  }
  return 'function-icon'
}

// 获取模板类型标签
const getTemplateTypeLabel = (templateType: string) => {
  const labels: Record<string, string> = {
    table: '表格',
    form: '表单',
    chart: '图表'
  }
  return labels[templateType] || templateType
}

// 获取模板类型标签样式
const getTemplateTypeTagType = (templateType: string) => {
  const types: Record<string, string> = {
    table: 'primary',
    form: 'success',
    chart: 'warning'
  }
  return types[templateType] || 'info'
}

// 初始化权限信息
onMounted(async () => {
  // 从 URL 参数中获取权限信息
  const resource = route.query.resource as string
  const action = route.query.action as string  // 可选，用于默认选中
  const templateType = route.query.templateType as string  // 可选，函数类型（table、form、chart）

  if (!resource) {
    error.value = '缺少必要的参数：resource'
    loading.value = false
    return
  }

  const resourcePath = decodeURIComponent(resource)
  permissionInfo.value = {
    resource_path: resourcePath,
    action: action || '',
    action_display: action ? getPermissionDisplayName(action) : '',
    error_message: '',
  }

  // 解析资源路径，获取 user 和 app
  const pathParts = resourcePath.split('/').filter(Boolean)
  if (pathParts.length < 2) {
    error.value = '资源路径格式错误'
    loading.value = false
    return
  }

  const user = pathParts[0]
  const app = pathParts[1]

  // 加载服务树和工作空间信息
  try {
    // ⭐ 并行加载服务树和已有权限
    const [treeResponse, permissionsResponse] = await Promise.all([
      getAppWithServiceTree(user, app),
      getWorkspacePermissions({ user, app }).catch(err => {
        // 如果获取权限失败，记录错误但不中断流程
        console.warn('获取工作空间权限失败:', err)
        return null
      })
    ])
    
    if (treeResponse) {
      // 保存工作空间信息
      currentApp.value = treeResponse.app || null
      
      // ⭐ 保存已有权限
      if (permissionsResponse && permissionsResponse.permissions) {
        const permissionsMap = new Map<string, Record<string, boolean>>()
        for (const [resourcePath, perms] of Object.entries(permissionsResponse.permissions)) {
          permissionsMap.set(resourcePath, perms)
        }
        existingPermissions.value = permissionsMap
      }
      
      // 构建包含工作空间节点的树结构
      const appNode: ServiceTree = {
        id: 0, // 临时 ID，实际不会使用
        name: treeResponse.app?.name || app,
        code: app,
        parent_id: 0,
        type: 'package' as any, // 临时使用 package 类型，但会在模板中通过 data.type === 'app' 判断
        description: '',
        tags: '',
        app_id: treeResponse.app?.id || 0,
        ref_id: 0,
        full_code_path: `/${user}/${app}`,
        created_at: treeResponse.app?.created_at || '',
        updated_at: treeResponse.app?.updated_at || '',
        children: treeResponse.service_tree || []
      } as any
      
      // 扩展类型，添加 app 类型标识
      ;(appNode as any).type = 'app'
      
      serviceTree.value = [appNode]
      
      // 设置默认选中的资源
      selectedResourcePath.value = resourcePath
      
      // 展开到选中节点的路径（包括工作空间节点）
      const expandedPaths: string[] = []
      const appPath = `/${user}/${app}`
      
      // 如果选中的资源是工作空间本身，展开工作空间节点
      if (resourcePath === appPath) {
        expandedPaths.push(appPath)
      } else {
        // 否则展开工作空间节点，然后查找子节点路径
        expandedPaths.push(appPath)
        
        const findPath = (nodes: ServiceTree[], targetPath: string): boolean => {
          for (const node of nodes) {
            const fullPath = node.full_code_path
            
            if (fullPath === targetPath) {
              // 找到目标节点
              return true
            }
            
            if (node.children && node.children.length > 0) {
              if (findPath(node.children, targetPath)) {
                // 在子节点中找到目标，展开当前节点
                if (!expandedPaths.includes(fullPath)) {
                  expandedPaths.push(fullPath)
                }
                return true
              }
            }
          }
          return false
        }
        
        // 查找并展开路径
        findPath(serviceTree.value[0].children || [], resourcePath)
      }
      
      defaultExpandedKeys.value = expandedPaths
      
      // 加载选中资源的权限范围
      await loadResourcePermissions(resourcePath, action, templateType)
    } else {
      error.value = '无法加载服务树数据'
    }
  } catch (err: any) {
    console.error('加载服务树失败:', err)
    error.value = '加载服务树失败: ' + (err?.message || '未知错误')
  }

  loading.value = false
})

// 在服务树中查找节点
const findNodeInTree = (nodes: ServiceTree[], path: string): ServiceTree | null => {
        for (const node of nodes) {
          if (node.full_code_path === path) {
            return node
          }
          if (node.children && node.children.length > 0) {
      const found = findNodeInTree(node.children, path)
            if (found) return found
          }
        }
        return null
      }

// 加载资源的权限范围
const loadResourcePermissions = async (resourcePath: string, defaultAction?: string, urlTemplateType?: string) => {
  // 解析资源路径
  const pathParts = resourcePath.split('/').filter(Boolean)
  
  if (pathParts.length < 2) {
    error.value = '资源路径格式错误'
    return
  }

  let resourceType: 'function' | 'directory' | 'app' | undefined
  let templateType: string | undefined = urlTemplateType
  
  // 从服务树中查找节点
  const node = findNodeInTree(serviceTree.value, resourcePath)
      
      if (node) {
    // 检查节点类型（支持扩展的 app 类型）
    const nodeType = (node as any).type || node.type
    if (nodeType === 'app') {
      resourceType = 'app'
    } else if (node.type === 'function') {
          resourceType = 'function'
      templateType = node.template_type || urlTemplateType
        } else if (node.type === 'package') {
          resourceType = 'directory'
        }
  } else {
    // 如果找不到节点，根据路径长度判断
    if (pathParts.length === 2) {
      resourceType = 'app'
    } else {
      resourceType = 'function'  // 默认按函数处理
    }
  }
  
  // 获取权限范围
  const parsed = resourcePath.split('/').filter(Boolean)
  const resourceName = parsed[parsed.length - 1] || '资源'
  const displayName = resourceType === 'function' 
    ? `函数：${resourceName}` 
    : resourceType === 'directory' 
    ? `目录：${resourcePath}` 
    : `工作空间：${parsed[1] || '工作空间'}`
  
  const permissions = getAvailablePermissions(resourcePath, resourceType, templateType)
  
  currentScope.value = {
    resourcePath,
    resourceType: resourceType || 'function',
    resourceName,
    displayName,
    permissions,
    quickSelect: resourceType === 'function' ? {
      label: '申请此函数的全部权限',
      actions: permissions.map(p => p.action)
    } : resourceType === 'directory' ? {
      label: '申请此目录的管理权限',
      actions: ['directory:manage']
    } : {
      label: '申请此工作空间的管理权限',
      actions: ['app:manage']
    }
  }
  
  // 设置默认选中的权限点
  const minimalPermissions = permissions
    .filter(p => p.isMinimal === true)
    .map(p => p.action)
  
  if (defaultAction && !minimalPermissions.includes(defaultAction)) {
    minimalPermissions.push(defaultAction)
  }
  
  // ⭐ 检查该资源的已有权限，并自动选中
  const existingPerms = existingPermissions.value.get(resourcePath)
  const existingActions: string[] = []
  if (existingPerms) {
    for (const [action, hasPerm] of Object.entries(existingPerms)) {
      if (hasPerm) {
        existingActions.push(action)
      }
    }
  }
  
  // 检查是否有已保存的权限选择
  const savedPermissions = allResourcePermissions.value.get(resourcePath)
  if (savedPermissions && savedPermissions.length > 0) {
    // 如果有已保存的权限选择，恢复它，并合并已有权限
    const mergedPermissions = [...new Set([...savedPermissions, ...existingActions])]
    selectedPermissions.value = mergedPermissions
    // 更新权限选择状态
    updateResourcePermissions(resourcePath, mergedPermissions)
  } else {
    // 合并最小权限和已有权限
    const mergedPermissions = [...new Set([...minimalPermissions, ...existingActions])]
    selectedPermissions.value = mergedPermissions
    // 更新权限选择状态
    updateResourcePermissions(resourcePath, mergedPermissions)
  }
}

// 更新资源的权限选择状态
const updateResourcePermissions = (resourcePath: string, permissions: string[]) => {
  if (permissions.length === 0) {
    // 如果权限为空，删除该资源的权限记录，这样树节点上的权限提示就会消失
    allResourcePermissions.value.delete(resourcePath)
  } else {
    // 否则更新权限列表
    allResourcePermissions.value.set(resourcePath, [...permissions])
  }
}

// 获取节点已选择的权限
const getSelectedPermissionsForNode = (resourcePath: string): string[] => {
  return allResourcePermissions.value.get(resourcePath) || []
}

// 获取小权限（具体操作权限，不包括管理权限）
const getSmallPermissions = () => {
  if (!currentScope.value) return []
  return currentScope.value.permissions.filter(p => !(p as any).isManage)
}

// 获取管理权限（所有权/管理权限）
const getManagePermissions = () => {
  if (!currentScope.value) return []
  return currentScope.value.permissions.filter(p => (p as any).isManage)
}

// 检查权限是否已存在
const hasExistingPermission = (action: string): boolean => {
  if (!currentScope.value) return false
  const existingPerms = existingPermissions.value.get(currentScope.value.resourcePath)
  if (!existingPerms) return false
  return existingPerms[action] === true
}

// 获取权限的简化显示名称（用于树节点显示，去掉前缀）
const getSimplifiedPermissionName = (action: string): string => {
  const fullName = getPermissionDisplayName(action)
  
  // ⭐ 如果权限显示名称就是 action 本身（说明没有映射），返回空字符串，表示不显示
  if (fullName === action) {
    return ''
  }
  
  // 简化规则映射表
  const simplifiedMap: Record<string, string> = {
    '新增表格记录': '新增',
    '更新表格记录': '更新',
    '删除表格记录': '删除',
    '表单提交': '提交',
    '目录查看': '查看',
    '目录写入': '写入',
    '目录更新': '更新',
    '目录删除': '删除',
    '工作空间查看': '查看',
    '工作空间创建': '创建',
    '工作空间更新': '更新',
    '工作空间删除': '删除',
    '工作空间部署': '部署',
    '函数查看': '查看',
    '所有权': '所有权',  // 保持不变
  }
  
  // 如果映射表中有，使用简化名称
  if (fullName in simplifiedMap) {
    return simplifiedMap[fullName]
  }
  
  // 如果没有映射，尝试通用简化：去掉"表格"、"表单"、"目录"、"工作空间"等前缀
  let simplified = fullName
    .replace(/^新增表格记录$/, '新增')
    .replace(/^更新表格记录$/, '更新')
    .replace(/^删除表格记录$/, '删除')
    .replace(/^表单提交$/, '提交')
    .replace(/^目录(.+)$/, '$1')
    .replace(/^工作空间(.+)$/, '$1')
    .replace(/^函数(.+)$/, '$1')
  
  // 如果简化后还是原名称，说明无法简化，返回空字符串
  if (simplified === fullName) {
    return ''
  }
  
  return simplified
}

// 获取节点权限显示文本（用于树节点显示）
const getNodePermissionDisplayText = (resourcePath: string): string | null => {
  // ⭐ 收集已有权限
  const existingPerms = existingPermissions.value.get(resourcePath)
  const existingPermissionsList: string[] = []
  if (existingPerms) {
    for (const [action, hasPerm] of Object.entries(existingPerms)) {
      if (hasPerm) {
        existingPermissionsList.push(action)
      }
    }
  }
  
  // ⭐ 收集新选择的权限
  const selectedPermissionsList = getSelectedPermissionsForNode(resourcePath)
  // 过滤掉内部标记（如 _has_manage_permission）
  const realSelectedPermissions = selectedPermissionsList.filter(p => !p.startsWith('_'))
  
  // ⭐ 过滤掉已经存在的权限（避免重复显示）
  const newSelectedPermissions = realSelectedPermissions.filter(action => {
    // 如果已有权限中包含该权限，则不显示
    if (existingPerms && existingPerms[action] === true) {
      return false
    }
    return true
  })
  
  // 如果既没有已有权限也没有新选择的权限，返回 null
  if (existingPermissionsList.length === 0 && newSelectedPermissions.length === 0) {
    return null
  }
  
  // ⭐ 合并显示已有权限和新选择的权限
  const parts: string[] = []
  
  // 处理已有权限
  if (existingPermissionsList.length > 0) {
    // 检查是否有管理权限（优先级最高）
    if (existingPermissionsList.some(p => p === 'directory:manage' || p === 'app:manage' || p === 'function:manage')) {
      parts.push('已有：所有权')
    } else {
      // 显示所有已有权限的简化名称（过滤掉技术性权限点，只显示友好的名称）
      const friendlyNames = existingPermissionsList
        .map(action => getSimplifiedPermissionName(action))
        .filter(name => name && name !== '') // 过滤掉空字符串（技术性权限点）
      if (friendlyNames.length > 0) {
        parts.push('已有：' + friendlyNames.join('，'))
      } else {
        // 如果都是技术性权限点，显示"已有权限"
        parts.push('已有权限')
      }
    }
  }
  
  // 处理新选择的权限（只显示不重复的）
  if (newSelectedPermissions.length > 0) {
    // 检查是否有管理权限（优先级最高）
    if (newSelectedPermissions.includes('directory:manage') || newSelectedPermissions.includes('app:manage') || newSelectedPermissions.includes('function:manage')) {
      parts.push('已选：所有权')
    } else {
      // 显示所有新选择权限的简化名称（过滤掉技术性权限点）
      const friendlyNames = newSelectedPermissions
        .map(action => getSimplifiedPermissionName(action))
        .filter(name => name && name !== '') // 过滤掉空字符串（技术性权限点）
      if (friendlyNames.length > 0) {
        parts.push('已选：' + friendlyNames.join('，'))
      }
      // 如果都是技术性权限点，不显示（避免显示 chart:read 这种）
    }
  }
  
  return parts.length > 0 ? parts.join(' | ') : null
}

// 获取节点权限标签的类型（已有权限用 info，新选择的权限用 success）
const getNodePermissionTagType = (resourcePath: string): 'info' | 'success' => {
  const existingPerms = existingPermissions.value.get(resourcePath)
  if (existingPerms) {
    const hasAnyPermission = Object.values(existingPerms).some(v => v === true)
    if (hasAnyPermission) {
      return 'info'  // 已有权限用 info 类型（蓝色）
    }
  }
  return 'success'  // 新选择的权限用 success 类型（绿色）
}

// 处理权限选择变化（实现级联选择）
const handlePermissionChange = (selectedActions: string[]) => {
  if (!currentScope.value) return
  
  const resourcePath = currentScope.value.resourcePath
  const resourceType = currentScope.value.resourceType
  
  // ⭐ 如果选择了管理权限，移除其他权限（管理权限是最大权限）
  let finalSelectedActions = [...selectedActions]
  
  if (resourceType === 'directory') {
    // 目录类型：如果选择了 directory:manage，移除其他目录权限
    if (finalSelectedActions.includes('directory:manage')) {
      finalSelectedActions = finalSelectedActions.filter(action => 
        action === 'directory:manage' || !action.startsWith('directory:')
      )
    }
  } else if (resourceType === 'app') {
    // 工作空间类型：如果选择了 app:manage，移除其他工作空间权限
    if (finalSelectedActions.includes('app:manage')) {
      finalSelectedActions = finalSelectedActions.filter(action => 
        action === 'app:manage' || !action.startsWith('app:')
      )
    }
  }
  
  // 更新 selectedPermissions（确保界面上的复选框状态正确）
  if (JSON.stringify(finalSelectedActions.sort()) !== JSON.stringify(selectedActions.sort())) {
    selectedPermissions.value = finalSelectedActions
  }
  
  // 更新当前资源的权限（如果为空数组，也要更新，表示取消所有权限）
  updateResourcePermissions(resourcePath, finalSelectedActions)
  
  // 如果是目录或应用，需要级联到子资源
  if (resourceType === 'directory' || resourceType === 'app') {
    // 查找所有子资源
    const childResources = findAllChildResources(resourcePath)
    
    // 如果当前资源取消了所有权限，也要取消子资源的权限
    if (finalSelectedActions.length === 0) {
      childResources.forEach(childPath => {
        updateResourcePermissions(childPath, [])
      })
      } else {
      // 对每个子资源应用相同的权限（使用处理后的权限列表）
      childResources.forEach(childPath => {
        // 获取子资源的类型
        const childNode = findNodeInTree(serviceTree.value, childPath)
        if (!childNode) {
          console.warn(`找不到子节点: ${childPath}`)
          return
        }
        
        // 根据子资源类型和选择的权限，确定应该应用的权限（使用处理后的权限列表）
        const childPermissions = mapPermissionsForChild(childPath, childNode, finalSelectedActions)
        // 无论是否有权限，都要更新（可能是清空）
        updateResourcePermissions(childPath, childPermissions)
      })
      
      // 调试信息（开发时使用，生产环境可删除）
      if (process.env.NODE_ENV === 'development') {
        console.log(`级联权限更新: 父资源=${resourcePath}, 子资源数量=${childResources.length}`, childResources)
      }
    }
  }
}

// 查找所有子资源（递归）
const findAllChildResources = (parentPath: string): string[] => {
  const childPaths: string[] = []
  
  // 递归遍历函数，找到所有子节点
  const traverse = (node: ServiceTree) => {
    if (!node.full_code_path) return
    
    // 如果节点是父路径的子节点（不是父路径本身）
    if (node.full_code_path !== parentPath && node.full_code_path.startsWith(parentPath + '/')) {
      childPaths.push(node.full_code_path)
    }
    
    // 继续遍历子节点
    if (node.children && node.children.length > 0) {
      for (const child of node.children) {
        traverse(child)
      }
    }
  }
  
  // 从服务树的根节点开始遍历
  for (const rootNode of serviceTree.value) {
    traverse(rootNode)
  }
  
  return childPaths
}

// 将父资源的权限映射到子资源
const mapPermissionsForChild = (childPath: string, childNode: ServiceTree, parentPermissions: string[]): string[] => {
  const childPermissions: string[] = []
  
  // 检查父资源选择的权限
  for (const parentAction of parentPermissions) {
    if (parentAction === 'directory:manage' || parentAction === 'app:manage') {
      // 管理权限：子节点显示"所有权"
      if (childNode.type === 'package') {
        // 子目录：保存 directory:manage（显示时会显示为"所有权"）
        if (!childPermissions.includes('directory:manage')) {
          childPermissions.push('directory:manage')
        }
      } else if (childNode.type === 'function') {
        // 子函数：保存所有相关权限，但显示时会显示为"所有权"
        const childType = childNode.template_type
        if (childType === TEMPLATE_TYPE.TABLE) {
          // 保存所有表格权限，但显示时显示"所有权"
          if (!childPermissions.includes('table:read')) childPermissions.push('table:read')
          if (!childPermissions.includes('table:write')) childPermissions.push('table:write')
          if (!childPermissions.includes('table:update')) childPermissions.push('table:update')
          if (!childPermissions.includes('table:delete')) childPermissions.push('table:delete')
        } else if (childType === TEMPLATE_TYPE.FORM) {
          // 保存所有表单权限，但显示时显示"所有权"
          // ⭐ form 类型只有 form:write 权限，没有 form:read
          if (!childPermissions.includes('form:write')) childPermissions.push('form:write')
        } else if (childType === TEMPLATE_TYPE.CHART) {
          // 保存所有图表权限，但显示时显示"所有权"
          if (!childPermissions.includes('chart:read')) childPermissions.push('chart:read')
    } else {
          // 保存所有函数权限，但显示时显示"所有权"
          if (!childPermissions.includes('function:read')) childPermissions.push('function:read')
          if (!childPermissions.includes('function:manage')) childPermissions.push('function:manage')
        }
        // 添加一个特殊标记，表示这是管理权限下的子节点
        if (!childPermissions.includes('_has_manage_permission')) {
          childPermissions.push('_has_manage_permission')
        }
      }
    } else if (parentAction === 'directory:write') {
      // 写入权限：子节点继承写入权限（只继承给 table 和 form）
      if (childNode.type === 'package') {
        // 子目录：继承 directory:write
        if (!childPermissions.includes('directory:write')) {
          childPermissions.push('directory:write')
        }
      } else if (childNode.type === 'function') {
        // 子函数：根据类型映射写入权限（只继承给 table 和 form）
        const childType = childNode.template_type
        if (childType === TEMPLATE_TYPE.TABLE) {
          // table 类型：映射为 table:write
          if (!childPermissions.includes('table:write')) {
            childPermissions.push('table:write')
          }
        } else if (childType === TEMPLATE_TYPE.FORM) {
          // form 类型：映射为 form:write
          if (!childPermissions.includes('form:write')) {
            childPermissions.push('form:write')
          }
        }
        // chart 和其他类型：不继承 write 权限（用户要求不要乱映射）
      }
    } else if (parentAction === 'directory:update') {
      // 更新权限：子目录继承更新权限
      if (childNode.type === 'package') {
        // 子目录：继承 directory:update
        if (!childPermissions.includes('directory:update')) {
          childPermissions.push('directory:update')
        }
      } else if (childNode.type === 'function') {
        // 子函数：根据类型映射更新权限
        const childType = childNode.template_type
        if (childType === TEMPLATE_TYPE.TABLE) {
          // table 类型：映射为 table:update
          if (!childPermissions.includes('table:update')) {
            childPermissions.push('table:update')
          }
        }
        // form、chart 和其他类型：不继承 update 权限（只有 table 有 update）
      }
    } else if (parentAction === 'directory:delete') {
      // 删除权限：子目录继承删除权限
      if (childNode.type === 'package') {
        // 子目录：继承 directory:delete
        if (!childPermissions.includes('directory:delete')) {
          childPermissions.push('directory:delete')
        }
      } else if (childNode.type === 'function') {
        // 子函数：根据类型映射删除权限
        const childType = childNode.template_type
        if (childType === TEMPLATE_TYPE.TABLE) {
          // table 类型：映射为 table:delete
          if (!childPermissions.includes('table:delete')) {
            childPermissions.push('table:delete')
          }
        }
        // form、chart 和其他类型：不继承 delete 权限（只有 table 有 delete）
      }
    } else if (parentAction === 'directory:read' || parentAction === 'app:read') {
      // 查看权限：子节点显示"查看权限"
      if (childNode.type === 'package') {
        if (!childPermissions.includes('directory:read')) {
          childPermissions.push('directory:read')
        }
      } else if (childNode.type === 'function') {
        const childType = childNode.template_type
        if (childType === TEMPLATE_TYPE.TABLE) {
          if (!childPermissions.includes('table:read')) childPermissions.push('table:read')
        } else if (childType === TEMPLATE_TYPE.FORM) {
          // ⭐ form 类型只有 form:write 权限，没有 form:read
          // 查看权限不继承给 form 类型（form 只有 write 权限）
        } else if (childType === TEMPLATE_TYPE.CHART) {
          if (!childPermissions.includes('chart:read')) childPermissions.push('chart:read')
        } else {
          if (!childPermissions.includes('function:read')) childPermissions.push('function:read')
        }
      }
    }
  }
  
  return childPermissions
}

// 处理树节点点击
const handleTreeNodeClick = (data: ServiceTree) => {
  selectedResourcePath.value = data.full_code_path
  
  // 加载权限时，如果有已保存的权限选择，恢复它
  const savedPermissions = allResourcePermissions.value.get(data.full_code_path)
  if (savedPermissions && savedPermissions.length > 0) {
    loadResourcePermissions(data.full_code_path)
    // 恢复已选择的权限
    selectedPermissions.value = savedPermissions
  } else {
    loadResourcePermissions(data.full_code_path)
  }
}

// ⭐ 快捷选择（选择当前资源的全部权限）
const handleQuickSelect = () => {
  if (currentScope.value?.quickSelect) {
    selectedPermissions.value = [...currentScope.value.quickSelect.actions]
    // 触发级联选择
    handlePermissionChange(selectedPermissions.value)
    ElMessage.success(`已选择：${currentScope.value.quickSelect.label}`)
  }
}

// 提交申请
const handleSubmit = async () => {
  if (!formRef.value) return

  // 检查是否至少选择了一个权限
  if (!hasSelectedPermissions.value) {
    ElMessage.warning('请至少选择一个权限')
    return
  }

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  submitting.value = true

  try {
    // 提交当前选中资源的权限申请
    if (!currentScope.value || selectedPermissions.value.length === 0) {
      ElMessage.warning('请至少选择一个权限')
      return
    }
    
    await applyPermission({
      resource_path: currentScope.value.resourcePath,
      actions: selectedPermissions.value,
            reason: formData.value.reason,
          })
    
    ElMessage.success('权限申请已提交')
    
    // 延迟后返回上一页
    setTimeout(() => {
      router.back()
    }, 1500)
  } catch (err: any) {
    // 显示详细的错误信息
    const errorMessage = err?.response?.data?.msg || err?.message || '提交申请失败'
    ElMessage.error(errorMessage)
  } finally {
    submitting.value = false
  }
}

// 取消申请
const handleCancel = () => {
  router.back()
}
</script>

<style scoped lang="scss">
.permission-apply-wrapper {
  width: 100%;
  height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  background: var(--el-bg-color-page);
  padding: 24px;
  box-sizing: border-box;
}

.permission-apply {
  max-width: 1600px;
  margin: 0 auto;
  padding-bottom: 40px;

  .apply-card {
    border-radius: 12px;
    border: none;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
    background: var(--el-bg-color);

    :deep(.el-card__header) {
      padding: 20px 24px;
      border-bottom: 1px solid var(--el-border-color-lighter);
      background: var(--el-fill-color-lighter);
      border-radius: 12px 12px 0 0;
    }

    :deep(.el-card__body) {
      padding: 24px;
    }

    .card-header {
      display: flex;
      align-items: center;
      gap: 12px;

      h2 {
        margin: 0;
        font-size: 22px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }
    }

    .loading-container {
      padding: 20px;
    }

    .error-container {
      padding: 20px;
    }

    .apply-content {
      .apply-layout {
        display: grid;
        grid-template-columns: 400px 1fr 320px;
        gap: 24px;
        align-items: start;
      }

      .apply-sidebar {
        position: sticky;
        top: 24px;

        .tree-card {
          border-radius: 12px;
          border: 1px solid var(--el-border-color-lighter);
          background: var(--el-bg-color);

          :deep(.el-card__header) {
            padding: 16px 20px;
            border-bottom: 1px solid var(--el-border-color-lighter);
            background: var(--el-fill-color-lighter);
            border-radius: 12px 12px 0 0;

            h3 {
              margin: 0;
              font-size: 16px;
              font-weight: 600;
              color: var(--el-text-color-primary);
            }
          }

          :deep(.el-card__body) {
            padding: 20px;
          }

          .tree-container {
            max-height: calc(100vh - 200px);
            overflow-y: auto;
            
            .resource-tree {
              :deep(.el-tree-node__content) {
                height: auto;
                padding: 0;
                margin-bottom: 2px;
              }
              
              :deep(.el-tree-node__content:hover) {
                background-color: transparent;
              }
              
              :deep(.el-tree-node__expand-icon) {
                padding: 6px;
                transition: all 0.2s ease;
                color: var(--el-text-color-secondary);
                border-radius: 2px;
                cursor: pointer;
              }
              
              :deep(.el-tree-node__expand-icon:hover) {
                background-color: var(--el-fill-color);
              }
              
              :deep(.el-tree-node.is-expanded > .el-tree-node__content .el-tree-node__expand-icon) {
                transform: rotate(90deg);
              }
              
              :deep(.el-tree-node__expand-icon.is-leaf) {
                color: transparent;
              }
              
              :deep(.el-tree-node.is-current > .el-tree-node__content) {
                background-color: transparent;
                font-weight: normal;
              }
              
              .tree-node {
                display: flex;
                align-items: center;
                gap: 8px;
                flex: 1;
                width: 100%;
                
                .node-icon {
                  width: 16px;
                  height: 16px;
                  margin-right: 8px;
                  color: #6366f1;  /* 紫色主题色（indigo-500） */
                  opacity: 0.8;
                  flex-shrink: 0;
                  transition: color 0.2s ease;
                  
                  &.app-icon {
                    color: #f59e0b; /* amber-500 - 工作空间用橙色 */
                    opacity: 0.9;
                  }
                  
                  &.app-icon-img {
                    width: 16px;
                    height: 16px;
                    object-fit: contain;
                    opacity: 0.9;
                  }
                  
                  &.package-icon {
                    color: #6366f1;
                    opacity: 0.8;
                  }
                  
                  &.package-icon-img {
                    width: 16px;
                    height: 16px;
                    object-fit: contain;
                    opacity: 0.9;
                  }
                  
                  &.table-icon {
                    color: #10b981; /* green-500 - 表格用绿色 */
                    opacity: 0.9;
                  }
                  
                  &.form-icon {
                    color: #3b82f6; /* blue-500 - 表单用蓝色 */
                    opacity: 0.9;
                  }
                  
                  &.form-icon-img {
                    width: 16px;
                    height: 16px;
                    object-fit: contain;
                    opacity: 0.9;
                  }
                  
                  &.chart-icon {
                    color: #f59e0b; /* amber-500 - 图表用橙色 */
                    opacity: 0.9;
                  }
                  
                  &.function-icon {
                    color: #6366f1; /* indigo-500 - 默认函数图标 */
                    opacity: 0.8;
                  }
                  
                  &.fx-icon {
                    font-size: 12px;
            font-weight: 600;
                    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
                    font-style: italic;
                    color: #6366f1;
                    opacity: 0.8;
                  }
                }
                
                .node-label {
                  font-size: 14px;
            color: var(--el-text-color-primary);
                  flex: 1;
                  overflow: hidden;
                  text-overflow: ellipsis;
                  white-space: nowrap;
                }
                
                .node-meta {
                  display: flex;
                  align-items: center;
                  gap: 8px;
                  flex-shrink: 0;
                  
                  .node-type-tag {
                    font-size: 10px;
                  }
                  
                  .template-tag {
                    font-size: 10px;
                  }
                  
                  .selected-permissions-hint {
                    display: flex;
                    align-items: center;
                    gap: 4px;
                    flex-wrap: wrap;
                    
                    .permission-hint-tag {
                      font-size: 10px;
                      padding: 2px 6px;
                      margin: 0;
                    }
                  }
                }
              }
              
              :deep(.el-tree-node__content) {
                height: 32px;
                padding: 0 8px;
                display: flex;
                align-items: center;

            &:hover {
                  background-color: var(--el-fill-color-light);
                }
              }
              
              :deep(.el-tree-node.is-current > .el-tree-node__content) {
                background-color: var(--el-fill-color-lighter);
                border-left: 2px solid #6366f1;
                
                .tree-node {
                  .node-label {
                    color: var(--el-text-color-primary);
                    font-weight: 500;
                  }
                  
                  .node-icon {
                    color: #6366f1;
                    opacity: 0.8;
                  }
                }
              }
              
              /* 确保子节点不受父节点选中状态影响 */
              :deep(.el-tree-node.is-current .el-tree-node__children .el-tree-node__content) {
                background-color: transparent;
                border-left: none;
              }
            }
          }
        }
      }

      .apply-main {
        min-width: 0; // 防止 grid 溢出

        .permission-scopes {
          .scope-header-main {
              display: flex;
              justify-content: space-between;
              align-items: center;
            margin-bottom: 16px;
            padding-bottom: 16px;
            border-bottom: 1px solid var(--el-border-color-lighter);

            .scope-title-main {
                display: flex;
                align-items: center;
                gap: 8px;
              flex-wrap: wrap;

              .scope-name-main {
                font-size: 18px;
                font-weight: 600;
                  color: var(--el-text-color-primary);
                }
              
              .selected-permissions-display {
                display: flex;
                align-items: center;
                gap: 6px;
                flex-wrap: wrap;
                
                .selected-permission-tag {
                  font-size: 12px;
                  padding: 4px 8px;
                }
              }
            }
          }

          .scope-path-main {
            margin-bottom: 24px;
            padding: 12px 16px;
              background: var(--el-fill-color-lighter);
            border-radius: 8px;
              border: 1px solid var(--el-border-color-lighter);

              code {
                font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
                font-size: 13px;
                color: var(--el-text-color-primary);
              }
          }

          .permission-list {
            .permission-list-header {
              margin-bottom: 20px;
              
              .permission-list-title {
                margin: 0 0 12px 0;
                font-size: 16px;
                font-weight: 600;
                color: var(--el-text-color-primary);
              }
              
              .permission-tip {
                margin-top: 12px;
                
                :deep(.el-alert__content) {
                  .tip-content {
                    .tip-text {
                      margin: 4px 0;
                      font-size: 13px;
                      line-height: 1.6;
                      color: var(--el-text-color-regular);
                      
                      &:first-child {
                        margin-top: 0;
                      }
                      
                      strong {
                        color: var(--el-color-primary);
                      }
                    }
                  }
                }
              }
            }
            
            .permission-section {
              margin-bottom: 24px;
              
              &:last-child {
                margin-bottom: 0;
              }
              
              &.manage-permissions {
                padding: 16px;
                background: var(--el-fill-color-lighter);
                border-radius: 8px;
                border: 1px solid var(--el-border-color-lighter);
                
                .manage-permissions-header {
                  display: flex;
                  align-items: center;
                  gap: 8px;
                  margin-bottom: 12px;
                  
                  .el-icon {
                    color: var(--el-color-warning);
                    font-size: 16px;
                  }
                  
                  .manage-permissions-title {
                    font-size: 16px;
                    font-weight: 600;
                    color: var(--el-text-color-primary);
                  }
                  
                  .manage-tag {
                    margin-left: auto;
                  }
                }
                
                .manage-alert {
                  margin-bottom: 16px;
                  
                  :deep(.el-alert__content) {
                    .alert-content {
                      .alert-text {
                        margin: 4px 0;
                        font-size: 13px;
                        line-height: 1.6;
                        color: var(--el-text-color-regular);
                        
                        &:first-child {
                          margin-top: 0;
                        }
                        
                        strong {
                          color: var(--el-color-warning-dark-2);
                        }
                      }
                    }
                  }
                }
              }
            }

            .permission-checkbox-group {
              display: flex;
              flex-direction: column;
              gap: 12px;
              width: 100%;

              :deep(.el-checkbox) {
                margin: 0;
                height: auto;
                align-items: flex-start;
                width: 100%;
                max-width: 100%;
                
                .el-checkbox__input {
                  margin-top: 2px;
                  flex-shrink: 0;
                }
                
                .el-checkbox__label {
                  width: 100%;
                  max-width: 100%;
                  padding-left: 8px;
                  line-height: 1.5;
                  word-break: break-word;
                  overflow-wrap: break-word;
                }
              }

              :deep(.el-checkbox.is-checked) {
                .permission-checkbox {
                  border-color: var(--el-color-primary);
                  background-color: var(--el-color-primary-light-9);
                }
              }
              
              :deep(.el-checkbox.manage-checkbox.is-checked) {
                .permission-checkbox {
                  border-color: var(--el-color-warning);
                  background-color: var(--el-color-warning-light-9);
                }
              }

              .permission-checkbox {
                width: 100%;
                max-width: 100%;
                margin: 0;
                padding: 16px;
                border: 1px solid var(--el-border-color-lighter);
                border-radius: 8px;
                transition: all 0.2s ease;
                background: var(--el-fill-color-lighter);
                min-height: auto;
                display: flex;
                flex-direction: column;
                justify-content: flex-start;
                box-sizing: border-box;

                &:hover {
                  border-color: var(--el-color-primary-light-7);
                  background-color: var(--el-fill-color);
                  transform: translateY(-1px);
                  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
                }

                .permission-option {
                  display: flex;
                  flex-direction: column;
                  align-items: flex-start;
                  gap: 8px;
                  width: 100%;
                  max-width: 100%;
                  min-width: 0;

                  .permission-header {
                    display: flex;
                    align-items: flex-start;
                    gap: 12px;
                    width: 100%;
                    max-width: 100%;
                    min-width: 0;
                    flex-wrap: wrap;

                    .permission-name {
                      font-weight: 600;
                      color: var(--el-text-color-primary);
                      font-size: 15px;
                      line-height: 1.4;
                      word-break: break-word;
                      overflow-wrap: break-word;
                      flex: 1;
                      min-width: 0;
                    }

                    .permission-tags {
                      display: flex;
                      align-items: center;
                      gap: 6px;
                      flex-wrap: wrap;
                      flex-shrink: 0;
                      
                      .existing-tag,
                      .new-selected-tag,
                      .minimal-tag {
                        flex-shrink: 0;
                      }
                    }
                  }
                  
                  .permission-description {
                    margin: 0;
                    font-size: 13px;
                    color: var(--el-text-color-regular);
                    line-height: 1.6;
                    word-break: break-word;
                    overflow-wrap: break-word;
                    width: 100%;
                  }
                  
                  .permission-inheritance {
                    display: flex;
                    align-items: flex-start;
                    gap: 8px;
                    padding: 10px 12px;
                    background: var(--el-fill-color-darker);
                    border-radius: 6px;
                    border: 1px solid var(--el-border-color);
                    width: 100%;
                    box-sizing: border-box;
                    margin-top: 4px;
                    
                    .inheritance-icon {
                      color: var(--el-text-color-regular);
                      font-size: 14px;
                      margin-top: 2px;
                      flex-shrink: 0;
                    }
                    
                    .inheritance-text {
                      font-size: 12px;
                      color: var(--el-text-color-regular);
                      line-height: 1.6;
                      flex: 1;
                      min-width: 0;
                      width: 0;
                      word-break: break-word;
                      overflow-wrap: break-word;
                    }
                  }

                  .permission-code {
                    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
                    font-size: 11px;
                    color: var(--el-text-color-secondary);
                    background: var(--el-fill-color);
                    padding: 2px 6px;
                    border-radius: 4px;
                    border: 1px solid var(--el-border-color-lighter);
                    align-self: flex-start;
                    word-break: break-all;
                  }
                  }
                }
              }
            }
          }

        .empty-state {
          display: flex;
          justify-content: center;
          align-items: center;
          min-height: 400px;
        }
      }

      .apply-sidebar-right {
        position: sticky;
        top: 24px;

        .form-card {
          border-radius: 12px;
          border: 1px solid var(--el-border-color-lighter);
          background: var(--el-bg-color);

          :deep(.el-card__header) {
            padding: 16px 20px;
            border-bottom: 1px solid var(--el-border-color-lighter);
            background: var(--el-fill-color-lighter);
            border-radius: 12px 12px 0 0;

            h3 {
              margin: 0;
              font-size: 16px;
              font-weight: 600;
              color: var(--el-text-color-primary);
            }
          }

          :deep(.el-card__body) {
            padding: 20px;
          }
        }

        .apply-form {
          .form-item-tip {
            margin-top: 8px;
          }

          :deep(.el-form-item__label) {
            font-weight: 500;
            color: var(--el-text-color-primary);
          }

          :deep(.el-textarea__inner) {
            border-radius: 8px;
            border-color: var(--el-border-color);
            background: var(--el-fill-color-lighter);
            transition: all 0.2s ease;

            &:focus {
              border-color: var(--el-color-primary);
              background: var(--el-bg-color);
            }
          }

          :deep(.el-button) {
            border-radius: 8px;
            padding: 10px 20px;
          }
        }
      }
    }
  }
}
</style>

