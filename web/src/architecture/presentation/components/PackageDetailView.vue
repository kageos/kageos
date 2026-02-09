<!--
  PackageDetailView - 服务目录详情页面

  职责：
  - 显示服务目录信息
-->
<template>
  <div class="package-detail-view">
    <!-- 顶部横幅区域 -->
    <div class="hero-section">
      <div class="hero-content">
        <el-button
          @click="handleBack"
          :icon="ArrowLeft"
          circle
          class="back-button"
          size="large"
        />
        <div class="hero-info">
          <div class="hero-icon-wrapper">
            <img
              v-if="packageNode?.type === 'package'"
              src="/service-tree/custom-folder.svg"
              alt="目录"
              class="hero-icon-img"
            />
            <el-icon v-else class="hero-icon"><Folder /></el-icon>
          </div>
          <div class="hero-text">
            <h1 class="hero-title">{{ packageNode?.name || '服务目录' }}</h1>
            <p class="hero-subtitle" v-if="packageNode?.full_code_path">
              <el-icon class="path-icon"><Link /></el-icon>
              <span class="path-text">{{ packageNode.full_code_path }}</span>
              <el-button
                text
                :icon="CopyDocument"
                @click="handleCopyPath"
                class="path-copy-btn"
                size="small"
                title="复制路径"
              />
              <el-button
                text
                :icon="Clock"
                @click="handleShowUpdateHistory"
                class="path-history-btn"
                size="small"
                title="查看变更记录"
              >
                变更记录
              </el-button>
              <el-button
                v-if="canEdit"
                text
                :icon="Edit"
                @click="handleEdit"
                class="path-edit-btn"
                size="small"
                title="编辑目录"
              >
                编辑
              </el-button>
            </p>
            <p class="hero-description" v-if="packageNode?.description">
              {{ packageNode.description }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- 主要内容区域 -->
    <div class="main-content">
      <div class="detail-content">
        <!-- ⭐ 权限不足提示：当目录没有任何权限时显示 -->
        <div v-if="hasNoDirectoryPermissions" class="permission-error-wrapper">
        <el-card class="permission-error-card" shadow="hover">
          <template #header>
            <div class="permission-error-header">
              <el-icon class="permission-error-icon"><Lock /></el-icon>
              <span class="permission-error-title">权限不足</span>
            </div>
          </template>
          <div class="permission-error-content">
            <div class="permission-error-message">
              <p class="error-message-text">
                您没有 <strong>访问该目录</strong> 的权限
              </p>
            </div>
            <div v-if="packageNode?.full_code_path" class="permission-error-info">
              <el-icon><Document /></el-icon>
              <span class="info-label">资源路径：</span>
              <span class="info-value">{{ packageNode.full_code_path }}</span>
            </div>
            <div class="permission-error-actions">
              <el-button
                type="primary"
                size="default"
                @click="handleApplyPermission"
                :icon="Lock"
              >
                立即申请权限
              </el-button>
            </div>
          </div>
        </el-card>
        </div>

        <!-- ⭐ 权限申请 tab（仅管理员可见） -->
        <div v-else-if="showPermissionRequestTab" class="permission-request-section">
        <el-tabs v-model="activeTab" type="card" @tab-change="handleTabChange" class="detail-tabs">
          <el-tab-pane name="info">
            <template #label>
              <span>目录信息</span>
            </template>
            <div class="tab-content">
              <!-- 信息概览卡片 -->
              <div v-if="packageNode" class="overview-section">
                <div class="overview-card">
                  <div class="overview-item">
                    <div class="overview-icon-wrapper name-icon">
                      <el-icon class="overview-icon"><Document /></el-icon>
                    </div>
                    <div class="overview-content">
                      <div class="overview-label">目录名称</div>
                      <div class="overview-value">{{ packageNode.name }}</div>
                    </div>
                  </div>

                  <div class="overview-divider"></div>

                  <div class="overview-item">
                    <div class="overview-icon-wrapper code-icon">
                      <el-icon class="overview-icon"><Key /></el-icon>
                    </div>
                    <div class="overview-content">
                      <div class="overview-label">目录代码</div>
                      <div class="overview-value code-text">{{ packageNode.code }}</div>
                    </div>
                  </div>

                  <div class="overview-divider"></div>

                  <div class="overview-item">
                    <div class="overview-icon-wrapper count-icon">
                      <el-icon class="overview-icon"><Files /></el-icon>
                    </div>
                    <div class="overview-content">
                      <div class="overview-label">子项数量</div>
                      <div class="overview-value">
                        {{ packageNode?.children?.length || 0 }} 项
                      </div>
                    </div>
                  </div>

                  <!-- Owner 信息 -->
                  <div v-if="packageNode?.owner && packageNode.owner.trim()" class="overview-divider"></div>

                  <div v-if="packageNode?.owner && packageNode.owner.trim()" class="overview-item">
                    <div class="overview-icon-wrapper owner-icon">
                      <el-icon class="overview-icon"><Star /></el-icon>
                    </div>
                    <div class="overview-content">
                      <div class="overview-label">创建者</div>
                      <div class="overview-value">
                        <UserWidget
                          :field="ownerField"
                          :value="ownerFieldValue"
                          mode="detail"
                          field-path="owner"
                        />
                      </div>
                    </div>
                  </div>

                  <!-- 管理员信息 -->
                  <div v-if="packageNode?.admins && packageNode.admins.trim()" class="overview-divider"></div>

                  <div v-if="packageNode?.admins && packageNode.admins.trim()" class="overview-item">
                    <div class="overview-icon-wrapper admins-icon">
                      <el-icon class="overview-icon"><Avatar /></el-icon>
                    </div>
                    <div class="overview-content">
                      <div class="overview-label">管理员</div>
                      <div class="overview-value">
                        <UsersWidget
                          :field="adminsField"
                          :value="adminsFieldValue"
                          :field-path="adminsField.code"
                          mode="detail"
                        />
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 子目录和函数列表 -->
              <div class="children-section" v-if="packageNode?.children && packageNode.children.length > 0">
                <div class="section-header">
                  <h3 class="section-title">
                    <el-icon class="section-icon"><Files /></el-icon>
                    子目录和函数
                  </h3>
                  <el-tag class="section-badge" type="primary" size="small">
                    {{ packageNode.children.length }}
                  </el-tag>
                </div>

                <div class="children-grid">
                  <div
                    v-for="child in packageNode.children"
                    :key="child.id"
                    class="child-card"
                    @click="handleChildClick(child)"
                  >
                    <div class="child-card-header">
                      <div class="child-icon-wrapper" :class="child.type === 'package' ? 'package-type' : 'function-type'">
                        <!-- package 类型：使用自定义文件夹图标 -->
                        <img
                          v-if="child.type === 'package'"
                          src="/service-tree/custom-folder.svg"
                          alt="目录"
                          class="child-icon-img"
                        />
                        <!-- function 类型：根据 template_type 显示不同图标 -->
                        <template v-else-if="child.type === 'function'">
                          <!-- 表单类型：使用编辑图标 -->
                          <img
                            v-if="child.template_type === TEMPLATE_TYPE.FORM"
                            src="/service-tree/编辑.svg"
                            alt="表单"
                            class="child-icon-img"
                          />
                          <!-- 其他类型：使用组件图标 -->
                          <el-icon v-else class="child-icon">
                            <component :is="getChildFunctionIcon(child)" />
                          </el-icon>
                        </template>
                        <!-- 默认图标 -->
                        <el-icon v-else class="child-icon">
                          <Document />
                        </el-icon>
                      </div>
                      <el-tag
                        v-if="child.type === 'function'"
                        size="small"
                        :type="getTemplateTypeTag(child.template_type)"
                        class="child-type-tag"
                      >
                        {{ getTemplateTypeText(child.template_type) }}
                      </el-tag>
                    </div>
                    <div class="child-card-body">
                      <div class="child-name">{{ child.name }}</div>
                      <div class="child-description" v-if="child.description">
                        {{ child.description }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <el-empty
                v-else
                description="该目录下暂无子目录或函数"
                :image-size="120"
                class="empty-state"
              />
            </div>
          </el-tab-pane>
          
          <!-- 权限申请 tab -->
          <el-tab-pane name="permissionRequest">
            <template #label>
              <el-badge :value="packageNode?.pending_count || 0" :hidden="!packageNode?.pending_count || packageNode.pending_count === 0" :max="99">
                <span>权限申请</span>
              </el-badge>
            </template>
            <div class="tab-content">
              <PermissionRequestList
                ref="permissionRequestListRef"
                :resource-path="packageNode?.full_code_path"
                :resource-type="resourceType"
                :auto-load="activeTab === 'permissionRequest'"
              />
            </div>
          </el-tab-pane>

          <!-- 权限管理 tab -->
          <el-tab-pane name="permissionManage">
            <template #label>
              <span>权限管理</span>
            </template>
            <div class="tab-content">
              <PermissionManageList
                ref="permissionManageListRef"
                :resource-path="packageNode?.full_code_path"
                :resource-type="resourceType"
                :user="getUserFromPath(packageNode?.full_code_path) || ''"
                :app="getAppFromPath(packageNode?.full_code_path) || ''"
                :auto-load="activeTab === 'permissionManage'"
              />
            </div>
          </el-tab-pane>
        </el-tabs>
        </div>
        
        <!-- 非管理员或没有权限申请 tab 时，显示原来的内容 -->
        <div v-else-if="packageNode" class="overview-section">
        <div class="overview-card">
          <div class="overview-item">
            <div class="overview-icon-wrapper name-icon">
              <el-icon class="overview-icon"><Document /></el-icon>
            </div>
            <div class="overview-content">
              <div class="overview-label">目录名称</div>
              <div class="overview-value">{{ packageNode.name }}</div>
            </div>
          </div>

          <div class="overview-divider"></div>

          <div class="overview-item">
            <div class="overview-icon-wrapper code-icon">
              <el-icon class="overview-icon"><Key /></el-icon>
            </div>
            <div class="overview-content">
              <div class="overview-label">目录代码</div>
              <div class="overview-value code-text">{{ packageNode.code }}</div>
            </div>
          </div>

          <div class="overview-divider"></div>

          <div class="overview-item">
            <div class="overview-icon-wrapper count-icon">
              <el-icon class="overview-icon"><Files /></el-icon>
            </div>
            <div class="overview-content">
              <div class="overview-label">子项数量</div>
              <div class="overview-value">
                {{ packageNode?.children?.length || 0 }} 项
              </div>
            </div>
          </div>

          <!-- Owner 信息 -->
          <div v-if="packageNode?.owner && packageNode.owner.trim()" class="overview-divider"></div>

          <div v-if="packageNode?.owner && packageNode.owner.trim()" class="overview-item">
            <div class="overview-icon-wrapper owner-icon">
              <el-icon class="overview-icon"><Star /></el-icon>
            </div>
            <div class="overview-content">
              <div class="overview-label">创建者</div>
              <div class="overview-value">
                <UserWidget
                  :field="ownerField"
                  :value="ownerFieldValue"
                  mode="detail"
                  field-path="owner"
                />
              </div>
            </div>
          </div>

          <!-- 管理员信息 -->
          <div v-if="packageNode?.admins && packageNode.admins.trim()" class="overview-divider"></div>

          <div v-if="packageNode?.admins && packageNode.admins.trim()" class="overview-item">
            <div class="overview-icon-wrapper admins-icon">
              <el-icon class="overview-icon"><Avatar /></el-icon>
            </div>
            <div class="overview-content">
              <div class="overview-label">管理员</div>
              <div class="overview-value">
                <UsersWidget
                  :field="adminsField"
                  :value="adminsFieldValue"
                  :field-path="adminsField.code"
                  mode="detail"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- 子目录和函数列表 -->
        <div class="children-section" v-if="!hasNoDirectoryPermissions && packageNode?.children && packageNode.children.length > 0">
          <div class="section-header">
            <h3 class="section-title">
              <el-icon class="section-icon"><Files /></el-icon>
              子目录和函数
            </h3>
            <el-tag class="section-badge" type="primary" size="small">
              {{ packageNode.children.length }}
            </el-tag>
          </div>

          <div class="children-grid">
            <div
              v-for="child in packageNode.children"
              :key="child.id"
              class="child-card"
              @click="handleChildClick(child)"
            >
              <div class="child-card-header">
                <div class="child-icon-wrapper" :class="child.type === 'package' ? 'package-type' : 'function-type'">
                  <!-- package 类型：使用自定义文件夹图标 -->
                  <img
                    v-if="child.type === 'package'"
                    src="/service-tree/custom-folder.svg"
                    alt="目录"
                    class="child-icon-img"
                  />
                  <!-- function 类型：根据 template_type 显示不同图标 -->
                  <template v-else-if="child.type === 'function'">
                    <!-- 表单类型：使用编辑图标 -->
                    <img
                      v-if="child.template_type === TEMPLATE_TYPE.FORM"
                      src="/service-tree/编辑.svg"
                      alt="表单"
                      class="child-icon-img"
                    />
                    <!-- 其他类型：使用组件图标 -->
                    <el-icon v-else class="child-icon">
                      <component :is="getChildFunctionIcon(child)" />
                    </el-icon>
                  </template>
                  <!-- 默认图标 -->
                  <el-icon v-else class="child-icon">
                    <Document />
                  </el-icon>
                </div>
                <el-tag
                  v-if="child.type === 'function'"
                  size="small"
                  :type="getTemplateTypeTag(child.template_type)"
                  class="child-type-tag"
                >
                  {{ getTemplateTypeText(child.template_type) }}
                </el-tag>
              </div>
              <div class="child-card-body">
                <div class="child-name">{{ child.name }}</div>
                <div class="child-description" v-if="child.description">
                  {{ child.description }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <el-empty
          v-else-if="!hasNoDirectoryPermissions"
          description="该目录下暂无子目录或函数"
          :image-size="120"
          class="empty-state"
        />
        </div>
      </div>
    </div>

    <!-- 变更记录对话框 -->
    <DirectoryUpdateHistoryDialog
      v-model="updateHistoryDialogVisible"
      mode="directory"
      :app-id="packageNode?.app_id || 0"
      :full-code-path="packageNode?.full_code_path || ''"
    />

    <!-- 编辑对话框 -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑目录"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="editFormRef"
        :model="editForm"
        label-width="100px"
        label-position="left"
      >
        <el-form-item label="目录名称" prop="name" :rules="[{ required: true, message: '请输入目录名称', trigger: 'blur' }]">
          <el-input
            v-model="editForm.name"
            placeholder="请输入目录名称"
            maxlength="100"
            show-word-limit
          />
        </el-form-item>
        
        <el-form-item label="管理员" prop="admins">
          <UsersWidget
            v-if="editDialogVisible"
            :key="`admins-${editForm.admins || 'empty'}`"
            :field="adminsField"
            :value="editAdminsFieldValue"
            :field-path="adminsField.code"
            mode="edit"
            @update:modelValue="handleEditAdminsChange"
          />
          <div class="form-item-tip">
            可以添加多个管理员，管理员可以编辑目录信息
          </div>
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="editSubmitting" @click="handleSubmitEdit">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ArrowLeft, Folder, Document, CopyDocument, Key, Link, Files, Clock, Lock, Avatar, Edit, Star } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { ServiceTree } from '@/types'
import { extractWorkspacePath } from '@/utils/route'
import { eventBus, RouteEvent } from '../../infrastructure/eventBus'
import { serviceFactory } from '../../infrastructure/factories'
import type { IServiceProvider } from '../../domain/interfaces/IServiceProvider'
import { TEMPLATE_TYPE } from '@/utils/functionTypes'
import ChartIcon from '@/components/icons/ChartIcon.vue'
import TableIcon from '@/components/icons/TableIcon.vue'
import FormIcon from '@/components/icons/FormIcon.vue'
import DirectoryUpdateHistoryDialog from '@/components/DirectoryUpdateHistoryDialog.vue'
import { buildPermissionApplyURL, DirectoryPermission } from '@/utils/permission'
import UsersWidget from '@/architecture/presentation/widgets/UsersWidget.vue'
import UserWidget from '@/architecture/presentation/widgets/UserWidget.vue'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { WidgetType } from '@/core/constants/widget'
import { useAuthStore } from '@/stores/auth'
import { updatePackage } from '@/api/service-tree'
import PermissionRequestList from '@/components/Permission/PermissionRequestList.vue'
import PermissionManageList from '@/components/Permission/PermissionManageList.vue'

interface Props {
  packageNode?: ServiceTree | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'refresh': []
}>()

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore() // ⭐ 必须在 showPermissionRequestTab 之前初始化

// Tab 相关
const activeTab = ref('info')
const permissionRequestListRef = ref<InstanceType<typeof PermissionRequestList> | null>(null)
const permissionManageListRef = ref<InstanceType<typeof PermissionManageList> | null>(null)

// ⭐ 判断是否显示权限申请 tab
// 条件：1. 节点类型是 package 或 app  2. 用户是管理员
const showPermissionRequestTab = computed(() => {
  if (!props.packageNode) {
    return false
  }
  
  // 必须是 package 类型
  if (props.packageNode.type !== 'package') {
    return false
  }
  
  // 检查是否是管理员
  if (!props.packageNode.admins || !authStore.user?.username) {
    return false
  }
  
  const admins = props.packageNode.admins.split(',').map((a: string) => a.trim()).filter(Boolean)
  return admins.includes(authStore.user.username)
})

// ⭐ 计算资源类型（用于权限组件）
// ⭐ 所有 package 类型统一使用 directory 资源类型（包括根目录/工作空间）
const resourceType = computed<'directory'>(() => {
  return 'directory'
})

// 从路径解析 user 和 app
const getUserFromPath = (fullCodePath?: string): string => {
  if (!fullCodePath) return ''
  const pathParts = fullCodePath.split('/').filter(Boolean)
  if (pathParts.length > 0 && pathParts[0]) {
    return pathParts[0]
  }
  return ''
}

const getAppFromPath = (fullCodePath?: string): string => {
  if (!fullCodePath) return ''
  const pathParts = fullCodePath.split('/').filter(Boolean)
  if (pathParts.length > 1 && pathParts[1]) {
    return pathParts[1]
  }
  return ''
}

// 处理 tab 切换
const handleTabChange = (tabName: string) => {
  if (tabName === 'permissionRequest' && permissionRequestListRef.value) {
    // 切换到权限申请 tab 时，触发加载
    nextTick(() => {
      permissionRequestListRef.value?.loadRequests()
    })
  } else if (tabName === 'permissionManage' && permissionManageListRef.value) {
    // 切换到权限管理 tab 时，触发加载
    nextTick(() => {
      permissionManageListRef.value?.loadPermissions()
    })
  }
}

// ⭐ 监听路由 query 参数，支持通过 tab 参数指定要打开的 tab
watch(
  () => route.query.tab,
  (tab: string | string[] | null) => {
    if (tab === 'permissionRequest' && showPermissionRequestTab.value) {
      activeTab.value = 'permissionRequest'
      // 切换 tab 时触发加载
      nextTick(() => {
        if (permissionRequestListRef.value) {
          permissionRequestListRef.value.loadRequests()
        }
      })
    } else if (tab === 'permissionManage' && showPermissionRequestTab.value) {
      activeTab.value = 'permissionManage'
      // 切换 tab 时触发加载
      nextTick(() => {
        if (permissionManageListRef.value) {
          permissionManageListRef.value.loadPermissions()
        }
      })
    }
  },
  { immediate: true }
)

// 变更记录对话框
const updateHistoryDialogVisible = ref(false)

// 编辑对话框
const editDialogVisible = ref(false)
const editSubmitting = ref(false)
const editFormRef = ref()
const editForm = ref({
  name: '',
  admins: ''
})

// ⭐ 检查是否可以编辑（owner 或 admins 可以编辑）
const canEdit = computed(() => {
  if (!props.packageNode || !authStore.user?.username) {
    return false
  }
  
  const currentUser = authStore.user.username
  
  // 检查是否是 owner
  if (props.packageNode.owner && props.packageNode.owner.trim() === currentUser) {
    return true
  }
  
  // 检查是否是 admins 之一
  if (props.packageNode.admins && props.packageNode.admins.trim()) {
    const admins = props.packageNode.admins.split(',').map((s: string) => s.trim()).filter((s: string) => Boolean(s))
    if (admins.includes(currentUser)) {
      return true
    }
  }
  
  return false
})

// Owner 字段配置（用于 UserWidget）
const ownerField = computed<FieldConfig>(() => ({
  code: 'owner',
  name: '创建者',
  widget: {
    type: WidgetType.USER,
    config: {}
  }
}))

// Owner 字段值（用于 UserWidget）
const ownerFieldValue = computed<FieldValue>(() => {
  if (!props.packageNode?.owner || !props.packageNode.owner.trim()) {
    return {
      raw: null,
      display: '',
      meta: {}
    }
  }
  
  return {
    raw: props.packageNode.owner.trim(),
    display: props.packageNode.owner.trim(),
    meta: {}
  }
})

// 管理员字段配置（用于 UsersWidget）
const adminsField = computed<FieldConfig>(() => ({
  code: 'admins',
  name: '管理员',
  widget: {
    type: WidgetType.USERS,
    config: {}
  }
}))

// 管理员字段值（用于 UsersWidget）
const adminsFieldValue = computed<FieldValue>(() => {
  if (!props.packageNode?.admins || !props.packageNode.admins.trim()) {
    return {
      raw: null,
      display: '',
      meta: {}
    }
  }
  
  const admins = props.packageNode.admins.split(',').map((s: string) => s.trim()).filter((s: string) => Boolean(s))
  return {
    raw: admins.join(','),
    display: admins.join(', '),
    meta: {}
  }
})

// ⭐ 检查是否没有任何权限（根据节点类型检查对应的权限）
const hasNoDirectoryPermissions = computed(() => {
  if (!props.packageNode) {
    return false
  }
  
  // 直接使用节点上的权限信息（后端返回的最新数据，已包含继承）
  const permissions = props.packageNode.permissions
  
  // 🔥 修复：如果没有权限信息或权限为空对象，返回 false（不显示权限不足）
  // 避免空 map 导致的无限循环问题
  if (!permissions || Object.keys(permissions).length === 0) {
    return false
  }
  
  // ⭐ 所有 package 类型统一检查 directory 权限（包括根目录/工作空间）
  const permissionsToCheck: string[] = [
    DirectoryPermission.read,
    DirectoryPermission.write,
    DirectoryPermission.update,
    DirectoryPermission.delete,
    DirectoryPermission.admin
  ]
  
  // 如果所有权限都是 false，则显示权限不足
  const hasNoPerms = permissionsToCheck.every(perm => {
    // 如果权限字段不存在，也视为 false
    return permissions[perm] === false || permissions[perm] === undefined
  })
  
  return hasNoPerms
})

// 处理权限申请
function handleApplyPermission() {
  if (!props.packageNode?.full_code_path) {
    ElMessage.warning('路径信息不可用')
    return
  }
  
  // ⭐ 所有 package 类型统一申请 directory:read 权限（包括根目录/工作空间）
  const defaultAction = DirectoryPermission.read
  
  // 跳转到权限申请页面
  const applyURL = buildPermissionApplyURL(props.packageNode.full_code_path, defaultAction, undefined)
  router.push(applyURL)
}

// 返回上一级
function handleBack() {
  // 获取当前路径，去掉最后一段
  const currentPath = extractWorkspacePath(route.path)
  if (currentPath) {
    const pathSegments = currentPath.split('/').filter(Boolean)
    if (pathSegments.length > 2) {
      // 至少是 user/app/package，去掉最后一段
      pathSegments.pop()
      const parentPath = `/workspace/${pathSegments.join('/')}`
      router.push(parentPath)
    } else {
      // 回到根目录
      router.push('/workspace')
    }
  } else {
    router.push('/workspace')
  }
}

// 复制完整路径
async function handleCopyPath() {
  if (!props.packageNode?.full_code_path) {
    ElMessage.warning('路径信息不可用')
    return
  }

  try {
    await navigator.clipboard.writeText(props.packageNode.full_code_path)
    ElMessage.success('路径已复制到剪贴板')
  } catch (error) {
    // 降级方案：使用传统方法
    const textArea = document.createElement('textarea')
    textArea.value = props.packageNode.full_code_path
    textArea.style.position = 'fixed'
    textArea.style.opacity = '0'
    document.body.appendChild(textArea)
    textArea.select()
    try {
      document.execCommand('copy')
      ElMessage.success('路径已复制到剪贴板')
    } catch (err) {
      ElMessage.error('复制失败，请手动复制')
    }
    document.body.removeChild(textArea)
  }
}

// 获取模板类型标签类型
function getTemplateTypeTag(templateType: string): string {
  const typeMap: Record<string, string> = {
    'table': 'success',
    'form': 'primary',
    'chart': 'warning'
  }
  return typeMap[templateType] || 'info'
}

// 获取模板类型文本
function getTemplateTypeText(templateType: string): string {
  const typeMap: Record<string, string> = {
    'table': '表格',
    'form': '表单',
    'chart': '图表'
  }
  return typeMap[templateType] || '函数'
}

// 获取子项函数图标组件（与左侧目录树保持一致）
function getChildFunctionIcon(child: ServiceTree) {
  if (child.template_type === TEMPLATE_TYPE.TABLE) {
    return TableIcon
  } else if (child.template_type === TEMPLATE_TYPE.FORM) {
    return FormIcon
  } else if (child.template_type === TEMPLATE_TYPE.CHART) {
    return ChartIcon
  }
  // 默认使用 Document 图标
  return Document
}

// 处理显示变更记录
function handleShowUpdateHistory(): void {
  emit('update-history', props.packageNode)
}

// 编辑表单的管理员字段值
const editAdminsFieldValue = computed<FieldValue>(() => {
  if (!editForm.value.admins || !editForm.value.admins.trim()) {
    return {
      raw: null,
      display: '',
      meta: {}
    }
  }
  
  const admins = editForm.value.admins.split(',').map((s: string) => s.trim()).filter((s: string) => Boolean(s))
  return {
    raw: admins.join(','),
    display: admins.join(', '),
    meta: {}
  }
})

// 处理编辑表单中管理员字段的变化
function handleEditAdminsChange(value: FieldValue): void {
  if (value.raw) {
    editForm.value.admins = typeof value.raw === 'string' ? value.raw : String(value.raw)
  } else {
    editForm.value.admins = ''
  }
}

// 处理编辑按钮点击
function handleEdit(): void {
  if (!props.packageNode) {
    return
  }
  
  // 初始化编辑表单
  editForm.value = {
    name: props.packageNode.name || '',
    admins: props.packageNode.admins || ''
  }
  
  editDialogVisible.value = true
}

// 提交编辑
async function handleSubmitEdit(): Promise<void> {
  if (!props.packageNode) {
    return
  }
  
  // 表单验证
  if (!editFormRef.value) {
    return
  }
  
  try {
    await editFormRef.value.validate()
  } catch (error) {
    return
  }
  
  editSubmitting.value = true
  try {
    await updatePackage(props.packageNode.id, {
      name: editForm.value.name.trim(),
      admins: editForm.value.admins.trim()
    })
    
    ElMessage.success('更新成功')
    editDialogVisible.value = false
    
    // 触发刷新（通过 emit 事件或直接刷新）
    // 这里可以通过 emit 通知父组件刷新，或者直接刷新当前页面数据
    // 暂时先关闭对话框，父组件可以通过 watch packageNode 来刷新
    // 或者我们可以 emit 一个事件让父组件处理刷新
    emit('refresh')
  } catch (error: any) {
    console.error('更新目录失败:', error)
    ElMessage.error(error.message || '更新目录失败')
  } finally {
    editSubmitting.value = false
  }
}

// 处理子项点击（跳转到对应的目录或函数）
function handleChildClick(child: ServiceTree): void {
  console.log('🔍 [PackageDetailView.handleChildClick] 开始处理子项点击', {
    childName: child.name,
    childType: child.type,
    fullCodePath: child.full_code_path,
    currentPath: route.path,
    currentQuery: route.query
  })
  
  const serviceProvider: IServiceProvider = serviceFactory
  const applicationService = serviceProvider.getWorkspaceApplicationService()

  if (child.type === 'function' && child.full_code_path) {
    // 函数节点：跳转到函数页面
    const targetPath = `/workspace${child.full_code_path}`
    console.log('🔍 [PackageDetailView.handleChildClick] 函数节点', {
      targetPath,
      currentPath: route.path,
      pathMatch: route.path === targetPath
    })
    
    if (route.path !== targetPath) {
      // 触发节点点击，加载函数详情
      applicationService.triggerNodeClick(child)

      const preserveParams = {
        table: false,
        search: false,
        state: false,
        linkNavigation: false
      }
      
      console.log('🔍 [PackageDetailView.handleChildClick] 发出路由更新请求（函数）', {
        path: targetPath,
        query: {},
        queryKeys: Object.keys({}),
        queryLength: Object.keys({}).length,
        preserveParams,
        source: 'package-detail-child-click'
      })

      // 更新路由
      eventBus.emit(RouteEvent.updateRequested, {
        path: targetPath,
        query: {},
        replace: true,
        preserveParams,
        source: 'package-detail-child-click'
      })
    } else {
      // 路由已匹配，直接触发节点点击加载详情
      console.log('🔍 [PackageDetailView.handleChildClick] 路由已匹配，直接触发节点点击')
      applicationService.triggerNodeClick(child)
    }
  } else if (child.type === 'package' && child.full_code_path) {
    // 目录节点：跳转到目录详情页面
    console.log('🔍 [PackageDetailView.handleChildClick] 目录节点', {
      fullCodePath: child.full_code_path
    })
    
    applicationService.triggerNodeClick(child)

    const targetPath = `/workspace${child.full_code_path}`
    if (route.path !== targetPath) {
      const preserveParams = {
        table: false,
        search: false,
        state: false,
        linkNavigation: false
      }
      
      console.log('🔍 [PackageDetailView.handleChildClick] 发出路由更新请求（目录）', {
        path: targetPath,
        query: {},
        queryKeys: Object.keys({}),
        queryLength: Object.keys({}).length,
        preserveParams,
        source: 'package-detail-child-click-package'
      })
      
      eventBus.emit(RouteEvent.updateRequested, {
        path: targetPath,
        query: {},
        replace: true,
        preserveParams,
        source: 'package-detail-child-click-package'
      })
    }
  }
}
</script>

<style scoped lang="scss">
.package-detail-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);

  // 顶部横幅区域
  .hero-section {
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
    padding: 32px 40px;

    .hero-content {
      max-width: 1400px;
      margin: 0 auto;
      display: flex;
      align-items: center;
      gap: 24px;

      .back-button {
        flex-shrink: 0;
        background: var(--el-bg-color);
        border-color: var(--el-border-color);
        color: var(--el-text-color-regular);

        &:hover {
          background: var(--el-color-primary-light-9);
          border-color: var(--el-color-primary);
          color: var(--el-color-primary);
        }
      }

      .hero-info {
        flex: 1;
        display: flex;
        align-items: center;
        gap: 20px;
        min-width: 0;

        .hero-icon-wrapper {
          flex-shrink: 0;
          display: flex;
          align-items: flex-start;
          justify-content: center;
          padding-top: 4px;

          .hero-icon {
            font-size: 48px;
            color: var(--el-color-primary);
          }

          .hero-icon-img {
            width: 48px;
            height: 48px;
            object-fit: contain;
          }
        }

        .hero-text {
          flex: 1;
          min-width: 0;

          .hero-title {
            margin: 0 0 8px 0;
            font-size: 28px;
            font-weight: 700;
            color: var(--el-text-color-primary);
            line-height: 1.2;
          }

          .hero-subtitle {
            margin: 0 0 8px 0;
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 14px;
            color: var(--el-text-color-secondary);

            .path-icon {
              font-size: 16px;
              color: var(--el-color-primary);
            }

            .path-text {
              flex: 1;
              font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
              color: var(--el-text-color-regular);
              word-break: break-all;
            }

            .path-copy-btn,
            .path-history-btn,
            .path-edit-btn {
              flex-shrink: 0;
              color: var(--el-text-color-secondary);

              &:hover {
                color: var(--el-color-primary);
              }
            }
          }

          .hero-description {
            margin: 0;
            font-size: 15px;
            color: var(--el-text-color-regular);
            line-height: 1.6;
            padding: 12px 16px;
            background: var(--el-fill-color-lighter);
            border-radius: 8px;
            border-left: 3px solid var(--el-color-primary);
          }
        }
      }
    }
  }

  // 主要内容区域
  .main-content {
    flex: 1;
    display: flex;
    overflow: hidden;

    .detail-content {
      flex: 1;
      overflow-y: auto;
      padding: 32px 40px;
      min-width: 0;
      width: 100%;

      // 信息概览卡片
      .overview-section {
        margin-bottom: 32px;

        .overview-card {
          display: flex;
          align-items: center;
          background: var(--el-bg-color);
          border: 1px solid var(--el-border-color-lighter);
          border-radius: 16px;
          padding: 24px;
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);

          .overview-item {
            flex: 1;
            display: flex;
            align-items: center;
            gap: 16px;

            .overview-icon-wrapper {
              flex-shrink: 0;
              display: flex;
              align-items: center;
              justify-content: center;
              width: 48px;
              height: 48px;
              border-radius: 12px;

              &.name-icon {
                background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));

                .overview-icon {
                  font-size: 24px;
                  color: var(--el-color-primary);
                }
              }

              &.code-icon {
                background: linear-gradient(135deg, var(--el-color-success-light-8), var(--el-color-success-light-9));

                .overview-icon {
                  font-size: 24px;
                  color: var(--el-color-success);
                }
              }

              &.count-icon {
                background: linear-gradient(135deg, var(--el-color-warning-light-8), var(--el-color-warning-light-9));

                .overview-icon {
                  font-size: 24px;
                  color: var(--el-color-warning);
                }
              }

              &.admins-icon {
                background: linear-gradient(135deg, #f3e8ff, #e9d5ff);

                .overview-icon {
                  font-size: 24px;
                  color: #9333ea;
                }
              }
            }

            .overview-content {
              flex: 1;
              min-width: 0;

              .overview-label {
                font-size: 13px;
                color: var(--el-text-color-secondary);
                margin-bottom: 4px;
                font-weight: 500;
              }

              .overview-value {
                font-size: 18px;
                font-weight: 600;
                color: var(--el-text-color-primary);

                &.code-text {
                  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
                  color: var(--el-color-success);
                  font-size: 16px;
                }
              }
            }
          }

          .overview-divider {
            width: 1px;
            height: 48px;
            background: var(--el-border-color-lighter);
            margin: 0 24px;
          }
        }
      }

      // 子目录和函数区域
      .children-section {
        margin-top: 32px;

        .section-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: 20px;

          .section-title {
            margin: 0;
            display: flex;
            align-items: center;
            gap: 10px;
            font-size: 20px;
            font-weight: 600;
            color: var(--el-text-color-primary);

            .section-icon {
              font-size: 22px;
              color: var(--el-color-primary);
            }
          }

          .section-badge {
            font-weight: 600;
            padding: 4px 12px;
          }
        }

        .children-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
          gap: 16px;
          width: 100%;

          .child-card {
            background: var(--el-bg-color);
            border: 1px solid var(--el-border-color-lighter);
            border-radius: 12px;
            padding: 20px;
            transition: all 0.3s ease;
            cursor: pointer;
            width: 100%;
            box-sizing: border-box;

            &:hover {
              border-color: var(--el-color-primary-light-7);
              box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
              transform: translateY(-2px);
            }

            .child-card-header {
              display: flex;
              align-items: center;
              justify-content: space-between;
              margin-bottom: 16px;

              .child-icon-wrapper {
                display: flex;
                align-items: center;
                justify-content: center;
                width: 48px;
                height: 48px;
                border-radius: 12px;
                flex-shrink: 0;

                &.package-type {
                  background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));

                  .child-icon-img {
                    width: 32px;
                    height: 32px;
                    object-fit: contain;
                  }
                }

                &.function-type {
                  background: linear-gradient(135deg, var(--el-color-success-light-8), var(--el-color-success-light-9));

                  .child-icon {
                    font-size: 24px;
                    color: var(--el-color-success);
                  }

                  .child-icon-img {
                    width: 32px;
                    height: 32px;
                    object-fit: contain;
                  }
                }
              }

              .child-type-tag {
                font-weight: 500;
              }
            }

            .child-card-body {
              .child-name {
                font-size: 16px;
                font-weight: 600;
                color: var(--el-text-color-primary);
                line-height: 1.5;
                word-break: break-word;
                margin-bottom: 8px;
              }

              .child-description {
                font-size: 13px;
                color: var(--el-text-color-secondary);
                line-height: 1.6;
                word-break: break-word;
                padding-top: 8px;
                border-top: 1px solid var(--el-border-color-lighter);
              }
            }
          }
        }
      }

      .empty-state {
        margin-top: 60px;
      }

      // ⭐ 权限不足提示样式
      .permission-error-wrapper {
        display: flex;
        justify-content: center;
        align-items: center;
        min-height: 400px;
        padding: 40px 20px;
      }

      .permission-error-card {
        max-width: 600px;
        width: 100%;
        border-radius: 16px;
        border: none;
        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
        transition: all 0.3s ease;

        &:hover {
          box-shadow: 0 6px 24px rgba(0, 0, 0, 0.12);
          transform: translateY(-2px);
        }
      }

      .permission-error-header {
        display: flex;
        align-items: center;
        gap: 12px;
        font-size: 18px;
        font-weight: 600;
        color: var(--el-color-warning);
      }

      .permission-error-icon {
        font-size: 24px;
      }

      .permission-error-title {
        font-size: 18px;
      }

      .permission-error-content {
        padding: 8px 0;
      }

      .permission-error-message {
        margin-bottom: 24px;
        padding: 16px;
        background: linear-gradient(135deg, rgba(255, 193, 7, 0.1) 0%, rgba(255, 152, 0, 0.05) 100%);
        border-radius: 12px;
        border-left: 4px solid var(--el-color-warning);
      }

      .error-message-text {
        margin: 0;
        font-size: 15px;
        line-height: 1.6;
        color: var(--el-text-color-primary);

        strong {
          color: var(--el-color-warning);
          font-weight: 600;
        }
      }

      .permission-error-info {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 16px;
        padding: 12px 16px;
        background: var(--el-bg-color-page);
        border-radius: 10px;
        font-size: 14px;
        transition: all 0.2s ease;

        &:hover {
          background: var(--el-fill-color-light);
        }

        .el-icon {
          color: var(--el-color-info);
          font-size: 18px;
        }

        .info-label {
          color: var(--el-text-color-regular);
          font-weight: 500;
        }

        .info-value {
          color: var(--el-text-color-primary);
          font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
          font-size: 13px;
          word-break: break-all;
        }
      }

      .permission-error-actions {
        margin-top: 24px;
        display: flex;
        justify-content: center;
        padding-top: 16px;
        border-top: 1px solid var(--el-border-color-lighter);
      }

      // ⭐ Tab 样式（参考旧版本的 card 样式）
      .detail-tabs {
        :deep(.el-tabs__header) {
          margin-bottom: 20px;
          overflow: visible; /* 确保 badge 不被裁剪 */
        }

        :deep(.el-tabs__nav-wrap) {
          overflow: visible !important; /* 确保 badge 不被裁剪 */
        }

        :deep(.el-tabs__nav-scroll) {
          overflow: visible !important; /* 确保 badge 不被裁剪 */
        }

        :deep(.el-tabs__nav) {
          border: none;
          overflow: visible; /* 确保 badge 不被裁剪 */
        }

        :deep(.el-tabs__item) {
          height: 40px;
          line-height: 40px;
          font-size: 14px;
          color: var(--el-text-color-regular);
          border: none;
          background: var(--el-bg-color-overlay);
          margin-right: 4px;
          border-radius: 4px 4px 0 0;
          transition: all 0.3s;
          padding: 0 20px;
          overflow: visible; /* 确保 badge 不被裁剪 */

          &:hover {
            color: var(--el-color-primary);
            opacity: 0.8;
          }

          &.is-active {
            color: var(--el-color-primary);
            background: var(--el-bg-color);
            font-weight: 500;
            opacity: 1;
          }
        }

        :deep(.el-tabs__active-bar) {
          display: none; /* card 类型不需要 active-bar */
        }

        // Badge 样式
        :deep(.el-badge) {
          position: relative;
          display: inline-block;
          
          .el-badge__content {
            font-size: 11px;
            height: 18px;
            line-height: 18px;
            padding: 0 6px;
            min-width: 18px;
            border-radius: 9px;
            z-index: 10; /* 确保 badge 在最上层 */
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1); /* 添加阴影，增强可见性 */
          }
        }
      }

      .tab-content {
        padding: 0;
      }
    }
  }
}

// 响应式设计
@media (max-width: 768px) {
  .package-detail-view {
    .hero-section {
      padding: 24px 20px;

      .hero-content {
        flex-direction: column;
        align-items: stretch;
        gap: 16px;

        .hero-info {
          flex-direction: column;
          align-items: flex-start;
          gap: 16px;
        }

        .action-button {
          width: 100%;
        }
      }
    }

    .main-content {
      flex-direction: column;

      .detail-content {
        padding: 24px 20px;

        .overview-section {
          .overview-card {
            flex-direction: column;
            gap: 20px;

            .overview-divider {
              width: 100%;
              height: 1px;
              margin: 0;
            }
          }
        }

        .children-section {
          .children-grid {
            grid-template-columns: 1fr;
          }
        }
      }
    }
  }
}
</style>

