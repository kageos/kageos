<template>
  <div class="user-settings">
    <div class="settings-container">
      <el-card shadow="hover" class="settings-card">
        <template #header>
          <div class="card-header">
            <el-button
              link
              :icon="ArrowLeft"
              @click="handleBack"
              class="back-button"
            >
              返回
            </el-button>
            <h2>个人设置</h2>
          </div>
        </template>

        <el-form
          ref="formRef"
          :model="formData"
          :rules="rules"
          label-width="100px"
          class="settings-form"
        >
          <!-- 头像 -->
          <el-form-item label="头像">
            <div class="avatar-section">
              <CommonUpload
                v-model="formData.avatar"
                :router="avatarRouter"
                accept="image/*"
                max-size="2MB"
                @change="handleAvatarChange"
              />
              <p class="form-tip">支持 JPG、PNG 格式，最大 2MB</p>
            </div>
          </el-form-item>

          <!-- 用户名（只读） -->
          <el-form-item label="用户名">
            <el-input
              :value="currentUser?.username"
              disabled
              class="disabled-input"
            />
            <p class="form-tip">用户名不可修改</p>
          </el-form-item>

          <!-- 邮箱（只读） -->
          <el-form-item label="邮箱">
            <el-input
              :value="currentUser?.email"
              disabled
              class="disabled-input"
            />
            <p class="form-tip">邮箱不可修改</p>
          </el-form-item>

          <!-- 昵称 -->
          <el-form-item label="昵称" prop="nickname">
            <el-input
              v-model="formData.nickname"
              placeholder="请输入昵称"
              maxlength="50"
              show-word-limit
              clearable
            />
          </el-form-item>

          <!-- 个人签名 -->
          <el-form-item label="个人签名" prop="signature">
            <el-input
              v-model="formData.signature"
              type="textarea"
              :rows="4"
              placeholder="请输入个人签名/简介"
              maxlength="200"
              show-word-limit
            />
          </el-form-item>

          <!-- 性别 -->
          <el-form-item label="性别" prop="gender">
            <el-radio-group v-model="formData.gender">
              <el-radio label="">不设置</el-radio>
              <el-radio label="male">男</el-radio>
              <el-radio label="female">女</el-radio>
              <el-radio label="other">其他</el-radio>
            </el-radio-group>
          </el-form-item>

          <!-- 提交按钮 -->
          <el-form-item>
            <el-button
              type="primary"
              :loading="submitting"
              @click="handleSubmit"
            >
              保存
            </el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <el-card shadow="hover" class="settings-card">
        <template #header>
          <div class="token-header">
            <div>
              <h2>OpenAPI Tokens</h2>
              <p>用于服务间调用，可长期有效，也可以随时吊销。</p>
            </div>
            <el-button type="primary" @click="createTokenDialogVisible = true">
              创建 Token
            </el-button>
          </div>
        </template>

        <el-table :data="openapiTokens" v-loading="tokensLoading" empty-text="暂无 OpenAPI Token">
          <el-table-column prop="name" label="名称" min-width="180" />
          <el-table-column prop="token_prefix" label="前缀" width="150" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.revoked_at ? 'danger' : 'success'">
                {{ row.revoked_at ? '已吊销' : '可用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="last_used_at" label="最后使用" min-width="170">
            <template #default="{ row }">
              {{ row.last_used_at || '从未使用' }}
            </template>
          </el-table-column>
          <el-table-column prop="expires_at" label="过期时间" min-width="170">
            <template #default="{ row }">
              {{ row.expires_at || '永不过期' }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="110" fixed="right">
            <template #default="{ row }">
              <el-button
                link
                type="danger"
                :disabled="!!row.revoked_at"
                @click="handleRevokeToken(row.id)"
              >
                吊销
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-dialog v-model="createTokenDialogVisible" title="创建 OpenAPI Token" width="520px">
        <el-form label-width="90px">
          <el-form-item label="名称">
            <el-input v-model="newTokenName" placeholder="kageos-hub-production" />
          </el-form-item>
          <el-form-item label="过期时间">
            <el-date-picker
              v-model="newTokenExpiresAt"
              type="datetime"
              placeholder="留空表示永不过期"
              style="width: 100%"
            />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="createTokenDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="creatingToken" @click="handleCreateToken">
            创建
          </el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="createdTokenVisible" title="Token 只显示一次" width="640px">
        <p class="form-tip">请现在保存这个 Token，关闭弹窗后无法再次查看明文。</p>
        <el-input
          v-model="createdSecretToken"
          readonly
          type="textarea"
          :rows="3"
          class="token-secret"
        />
        <template #footer>
          <el-button @click="copyCreatedToken">复制</el-button>
          <el-button type="primary" @click="createdTokenVisible = false">完成</el-button>
        </template>
      </el-dialog>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElForm } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import CommonUpload from '@/architecture/presentation/shared/components/CommonUpload.vue'
import type { FormRules } from 'element-plus'
import {
  createOpenAPIToken,
  listOpenAPITokens,
  revokeOpenAPIToken,
  type OpenAPITokenInfo,
} from '@/architecture/presentation/context/api/user'

const router = useRouter()
const authStore = useAuthStore()

// 表单引用
const formRef = ref<InstanceType<typeof ElForm>>()

// 提交状态
const submitting = ref(false)
const tokensLoading = ref(false)
const creatingToken = ref(false)
const createTokenDialogVisible = ref(false)
const createdTokenVisible = ref(false)
const newTokenName = ref('')
const newTokenExpiresAt = ref<Date | null>(null)
const createdSecretToken = ref('')
const openapiTokens = ref<OpenAPITokenInfo[]>([])

// 当前用户
const currentUser = computed(() => authStore.user)

// 头像上传路由
const avatarRouter = computed(() => {
  const username = currentUser.value?.username || 'default'
  return `${username}/avatar`
})

// 表单数据
const formData = reactive({
  avatar: '',
  nickname: '',
  signature: '',
  gender: '' as '' | 'male' | 'female' | 'other'
})

// 表单验证规则
const rules: FormRules = {
  nickname: [
    { max: 50, message: '昵称长度不能超过50个字符', trigger: 'blur' }
  ],
  signature: [
    { max: 200, message: '个人签名长度不能超过200个字符', trigger: 'blur' }
  ]
}

// 初始化表单数据
function initFormData() {
  if (currentUser.value) {
    formData.avatar = currentUser.value.avatar || ''
    formData.nickname = currentUser.value.nickname || ''
    formData.signature = currentUser.value.signature || ''
    formData.gender = (currentUser.value.gender || '') as '' | 'male' | 'female' | 'other'
  }
}

// 头像变化
function handleAvatarChange(url: string | null) {
  if (url) {
    formData.avatar = url
  }
}

// 提交表单
async function handleSubmit() {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    submitting.value = true

    // 构建更新数据（只包含有值的字段）
    const updateData: {
      avatar?: string
      nickname?: string
      signature?: string
      gender?: '' | 'male' | 'female' | 'other'
    } = {}

    if (formData.avatar !== (currentUser.value?.avatar || '')) {
      updateData.avatar = formData.avatar
    }
    if (formData.nickname !== (currentUser.value?.nickname || '')) {
      updateData.nickname = formData.nickname
    }
    if (formData.signature !== (currentUser.value?.signature || '')) {
      updateData.signature = formData.signature
    }
    if (formData.gender !== (currentUser.value?.gender || '')) {
      updateData.gender = formData.gender
    }

    // 如果没有需要更新的字段
    if (Object.keys(updateData).length === 0) {
      ElMessage.info('没有需要更新的内容')
      return
    }

    // 调用更新接口
    await authStore.updateUser(updateData)
    
    // 更新成功，可以返回上一页或刷新
    ElMessage.success('更新成功')
  } catch (error: any) {
    console.error('更新用户信息失败:', error)
    if (error?.message && !error.message.includes('validate')) {
      ElMessage.error(error.message || '更新失败')
    }
  } finally {
    submitting.value = false
  }
}

// 重置表单
function handleReset() {
  initFormData()
  formRef.value?.clearValidate()
}

// 返回上一页
function handleBack() {
  router.go(-1)
}

async function loadOpenAPITokens() {
  tokensLoading.value = true
  try {
    const resp = await listOpenAPITokens()
    openapiTokens.value = resp.tokens || []
  } catch (error: any) {
    ElMessage.error(error?.message || '加载 OpenAPI Token 失败')
  } finally {
    tokensLoading.value = false
  }
}

async function handleCreateToken() {
  if (!newTokenName.value.trim()) {
    ElMessage.warning('请输入 Token 名称')
    return
  }
  creatingToken.value = true
  try {
    const resp = await createOpenAPIToken({
      name: newTokenName.value.trim(),
      expires_at: newTokenExpiresAt.value ? newTokenExpiresAt.value.toISOString() : undefined,
    })
    createdSecretToken.value = resp.secret_token
    createdTokenVisible.value = true
    createTokenDialogVisible.value = false
    newTokenName.value = ''
    newTokenExpiresAt.value = null
    await loadOpenAPITokens()
  } catch (error: any) {
    ElMessage.error(error?.message || '创建 OpenAPI Token 失败')
  } finally {
    creatingToken.value = false
  }
}

async function handleRevokeToken(id: number) {
  try {
    await ElMessageBox.confirm('吊销后使用该 Token 的服务调用会立即失效，确定继续吗？', '吊销 Token', {
      type: 'warning',
      confirmButtonText: '吊销',
      cancelButtonText: '取消',
    })
    await revokeOpenAPIToken(id)
    ElMessage.success('已吊销')
    await loadOpenAPITokens()
  } catch (error: any) {
    if (error !== 'cancel') {
      ElMessage.error(error?.message || '吊销失败')
    }
  }
}

async function copyCreatedToken() {
  await navigator.clipboard.writeText(createdSecretToken.value)
  ElMessage.success('已复制')
}

// 组件挂载时初始化
onMounted(async () => {
  if (!authStore.isAuthenticated) {
    router.push('/login')
    return
  }
  
  try {
    await authStore.fetchUserInfo()
  } catch (error) {
    console.error('获取用户信息失败:', error)
  }
  
  initFormData()
  await loadOpenAPITokens()
})
</script>

<style scoped>
.user-settings {
  min-height: 100vh;
  background: var(--el-bg-color-page);
  padding: 20px;
}

.settings-container {
  max-width: 800px;
  margin: 0 auto;
}

.settings-card {
  margin-top: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.card-header .back-button {
  margin-left: -8px;
  padding: 4px 8px;
}

.card-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  flex: 1;
}

.settings-form {
  margin-top: 20px;
}

.avatar-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin: 0;
}

.disabled-input {
  opacity: 0.6;
}

.token-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.token-header h2 {
  margin: 0;
  font-size: 20px;
}

.token-header p {
  margin: 4px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.token-secret {
  margin-top: 12px;
}

</style>
