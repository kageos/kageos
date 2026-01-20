<!--
  WorkspaceDetailView - 工作空间详情页面

  职责：
  - 显示工作空间基本信息
  - 提供权限申请功能
  - 提供权限管理功能（仅管理员）
-->
<template>
  <div class="workspace-detail-view">
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
              src="/工作空间.svg"
              alt="工作空间"
              class="hero-icon-img"
            />
          </div>
          <div class="hero-text">
            <h1 class="hero-title">{{ workspaceInfo?.name || '工作空间' }}</h1>
            <p class="hero-subtitle">
              <el-icon class="path-icon"><Link /></el-icon>
              <span class="path-text">{{ workspaceFullPath }}</span>
              <el-button
                text
                :icon="CopyDocument"
                @click="handleCopyPath"
                class="path-copy-btn"
                size="small"
                title="复制路径"
              />
              <el-button
                v-if="canEdit"
                text
                :icon="Edit"
                @click="handleEdit"
                class="path-edit-btn"
                size="small"
                title="编辑工作空间"
              >
                编辑
              </el-button>
            </p>
            <p class="hero-description" v-if="workspaceInfo?.description">
              {{ workspaceInfo.description }}
            </p>
          </div>
        </div>
      </div>
    </div>

    <!-- 主要内容区域：标签页 -->
    <div class="main-content">
      <el-tabs v-model="activeTab" class="workspace-tabs">
        <!-- Tab 1: 基本信息 -->
        <el-tab-pane label="基本信息" name="info">
          <div class="info-section" v-loading="loading">
            <el-descriptions :column="2" border>
              <el-descriptions-item label="工作空间名称">
                {{ workspaceInfo?.name }}
              </el-descriptions-item>
              <el-descriptions-item label="工作空间代码">
                {{ workspaceInfo?.code }}
              </el-descriptions-item>
              <el-descriptions-item label="创建时间">
                {{ formatTime(workspaceInfo?.created_at) }}
              </el-descriptions-item>
              <el-descriptions-item label="更新时间">
                {{ formatTime(workspaceInfo?.updated_at) }}
              </el-descriptions-item>
              <el-descriptions-item label="创建者">
                {{ workspaceInfo?.created_by }}
              </el-descriptions-item>
              <el-descriptions-item label="管理员">
                <el-tag
                  v-for="admin in adminList"
                  :key="admin"
                  class="admin-tag"
                  type="success"
                >
                  {{ admin }}
                </el-tag>
                <span v-if="adminList.length === 0" class="text-muted">暂无</span>
              </el-descriptions-item>
              <el-descriptions-item label="标签" :span="2">
                <el-tag
                  v-for="tag in tagList"
                  :key="tag"
                  class="tag-item"
                >
                  {{ tag }}
                </el-tag>
                <span v-if="tagList.length === 0" class="text-muted">暂无</span>
              </el-descriptions-item>
              <el-descriptions-item label="描述" :span="2">
                {{ workspaceInfo?.description || '暂无描述' }}
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </el-tab-pane>

        <!-- Tab 2: 权限申请 -->
        <el-tab-pane label="权限申请" name="apply">
          <div class="apply-section">
            <el-card class="apply-form-card" shadow="never">
              <template #header>
                <div class="card-header">
                  <el-icon><Stamp /></el-icon>
                  <span>申请工作空间权限</span>
                </div>
              </template>
              <el-form
                ref="applyFormRef"
                :model="applyForm"
                :rules="applyFormRules"
                label-width="100px"
              >
                <el-form-item label="选择角色" prop="roleCode">
                  <el-select v-model="applyForm.roleCode" placeholder="请选择要申请的角色">
                    <el-option
                      v-for="role in availableRoles"
                      :key="role.code"
                      :label="role.name"
                      :value="role.code"
                    >
                      <span>{{ role.name }}</span>
                      <span class="role-description">{{ role.description }}</span>
                    </el-option>
                  </el-select>
                </el-form-item>
                <el-form-item label="申请理由" prop="reason">
                  <el-input
                    v-model="applyForm.reason"
                    type="textarea"
                    :rows="4"
                    placeholder="请说明您申请该权限的理由"
                    maxlength="500"
                    show-word-limit
                  />
                </el-form-item>
                <el-form-item>
                  <el-button type="primary" @click="handleSubmitApply" :loading="submitting">
                    提交申请
                  </el-button>
                  <el-button @click="handleResetApplyForm">重置</el-button>
                </el-form-item>
              </el-form>
            </el-card>

            <!-- 我的申请记录 -->
            <div class="my-requests-section">
              <h3 class="section-title">
                <el-icon><List /></el-icon>
                我的申请记录
              </h3>
              <PermissionRequestList
                :resource-path="workspaceFullPath"
                :show-my-requests="true"
              />
            </div>
          </div>
        </el-tab-pane>

        <!-- Tab 3: 权限管理（仅管理员可见） -->
        <el-tab-pane label="权限管理" name="manage" v-if="canManagePermission">
          <div class="manage-section">
            <PermissionManageList
              :resource-path="workspaceFullPath"
              :resource-type="'app'"
              :show-assign-button="true"
            />
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import {
  ArrowLeft,
  Link,
  CopyDocument,
  Edit,
  Stamp,
  List
} from '@element-plus/icons-vue'
import { useClipboard } from '@vueuse/core'
import { getAppDetailByUserAndCode } from '@/api'
import { useAuthStore } from '@/stores/auth'
import PermissionRequestList from '@/components/Permission/PermissionRequestList.vue'
import PermissionManageList from '@/components/Permission/PermissionManageList.vue'

interface Props {
  user: string
  app: string
}

const props = defineProps<Props>()
const emit = defineEmits(['back', 'edit'])

const authStore = useAuthStore()
const { copy } = useClipboard()

// 状态
const activeTab = ref('info')
const loading = ref(false)
const submitting = ref(false)
const workspaceInfo = ref<any>(null)

// 申请表单
const applyFormRef = ref<FormInstance>()
const applyForm = ref({
  roleCode: '',
  reason: ''
})

const applyFormRules: FormRules = {
  roleCode: [
    { required: true, message: '请选择角色', trigger: 'change' }
  ],
  reason: [
    { required: true, message: '请填写申请理由', trigger: 'blur' },
    { min: 10, message: '申请理由至少10个字符', trigger: 'blur' }
  ]
}

// 可申请的角色列表
const availableRoles = ref([
  {
    code: 'app_reader',
    name: '读者',
    description: '可以查看工作空间内容'
  },
  {
    code: 'app_writer',
    name: '编辑者',
    description: '可以编辑工作空间内容'
  },
  {
    code: 'app_admin',
    name: '管理员',
    description: '可以管理工作空间和权限'
  }
])

// 计算属性
const workspaceFullPath = computed(() => {
  return `/${props.user}/${props.app}`
})

const adminList = computed(() => {
  if (!workspaceInfo.value?.admins) return []
  return workspaceInfo.value.admins.split(',').map((admin: string) => admin.trim()).filter(Boolean)
})

const tagList = computed(() => {
  if (!workspaceInfo.value?.tags) return []
  return workspaceInfo.value.tags.split(',').map((tag: string) => tag.trim()).filter(Boolean)
})

const canEdit = computed(() => {
  // 检查当前用户是否是工作空间管理员
  if (!workspaceInfo.value || !authStore.user?.username) {
    return false
  }
  const admins = adminList.value
  return admins.includes(authStore.user.username)
})

const canManagePermission = computed(() => {
  // 检查当前用户是否是工作空间管理员
  if (!workspaceInfo.value || !authStore.user?.username) {
    return false
  }
  const admins = adminList.value
  return admins.includes(authStore.user.username)
})

// 方法
const loadWorkspaceInfo = async () => {
  loading.value = true
  try {
    const response = await getAppDetailByUserAndCode(props.user, props.app)
    workspaceInfo.value = response
  } catch (error: any) {
    console.error('加载工作空间信息失败:', error)
    ElMessage.error(error.message || '加载工作空间信息失败')
  } finally {
    loading.value = false
  }
}

const formatTime = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

const handleBack = () => {
  emit('back')
}

const handleCopyPath = async () => {
  await copy(workspaceFullPath.value)
  ElMessage.success('路径已复制到剪贴板')
}

const handleEdit = () => {
  emit('edit')
}

const handleSubmitApply = async () => {
  if (!applyFormRef.value) return

  await applyFormRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      // TODO: 调用申请权限 API
      // await applyPermission({
      //   resourcePath: workspaceFullPath.value,
      //   resourceType: 'app',
      //   roleCode: applyForm.value.roleCode,
      //   reason: applyForm.value.reason
      // })
      
      ElMessage.success('权限申请已提交，请等待审批')
      handleResetApplyForm()
      
      // 刷新申请记录
      // TODO: 触发 PermissionRequestList 刷新
    } catch (error: any) {
      console.error('提交申请失败:', error)
      ElMessage.error(error.message || '提交申请失败')
    } finally {
      submitting.value = false
    }
  })
}

const handleResetApplyForm = () => {
  applyFormRef.value?.resetFields()
}

// 生命周期
onMounted(() => {
  loadWorkspaceInfo()
})
</script>

<style lang="scss" scoped>
.workspace-detail-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: var(--el-bg-color);
  overflow: hidden;
}

/* 顶部横幅区域 */
.hero-section {
  background: var(--el-bg-color-page);
  border-bottom: 1px solid var(--el-border-color);
  padding: 32px;
  flex-shrink: 0;
}

.hero-content {
  max-width: 1400px;
  margin: 0 auto;
  display: flex;
  align-items: flex-start;
  gap: 20px;
}

.back-button {
  flex-shrink: 0;
  background-color: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color);
  
  &:hover {
    background-color: var(--el-fill-color);
    border-color: var(--el-color-primary);
    color: var(--el-color-primary);
  }
}

.hero-info {
  flex: 1;
  display: flex;
  gap: 20px;
  align-items: flex-start;
}

.hero-icon-wrapper {
  flex-shrink: 0;
  width: 80px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-fill-color-light);
  border-radius: 16px;
  border: 1px solid var(--el-border-color-lighter);
}

.hero-icon-img {
  width: 48px;
  height: 48px;
  object-fit: contain;
}

.hero-text {
  flex: 1;
  min-width: 0;
}

.hero-title {
  font-size: 32px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin: 0 0 12px 0;
  line-height: 1.3;
}

.hero-subtitle {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--el-text-color-regular);
  font-size: 14px;
  margin: 0 0 12px 0;
  flex-wrap: wrap;
}

.path-icon {
  font-size: 16px;
  color: var(--el-color-primary);
}

.path-text {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  background-color: var(--el-fill-color);
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 13px;
  color: var(--el-text-color-primary);
}

.path-copy-btn,
.path-edit-btn {
  padding: 4px 8px;
  height: auto;
  
  &:hover {
    color: var(--el-color-primary);
  }
}

.hero-description {
  color: var(--el-text-color-regular);
  font-size: 14px;
  line-height: 1.6;
  margin: 0;
  max-width: 800px;
}

/* 主要内容区域 */
.main-content {
  flex: 1;
  overflow: auto;
  padding: 24px 32px;
}

.workspace-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 24px;
  }
}

/* 基本信息 */
.info-section {
  background: var(--el-bg-color);
  border-radius: 8px;
  padding: 24px;
  border: 1px solid var(--el-border-color-lighter);
}

.admin-tag,
.tag-item {
  margin-right: 8px;
  margin-bottom: 4px;
}

.text-muted {
  color: var(--el-text-color-secondary);
  font-style: italic;
}

/* 权限申请 */
.apply-section {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.apply-form-card {
  border-radius: 8px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  
  :deep(.el-card__header) {
    background: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  
  :deep(.el-card__body) {
    background: var(--el-bg-color);
  }
  
  .card-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    font-size: 16px;
  }
}

.role-description {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-left: 8px;
}

.my-requests-section {
  .section-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
    margin: 0 0 16px 0;
  }
}

/* 权限管理 */
.manage-section {
  background: var(--el-bg-color);
  border-radius: 8px;
  padding: 24px;
  border: 1px solid var(--el-border-color-lighter);
}
</style>
