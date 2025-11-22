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
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElForm } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import CommonUpload from '@/components/CommonUpload.vue'
import type { FormRules } from 'element-plus'

const router = useRouter()
const authStore = useAuthStore()

// 表单引用
const formRef = ref<InstanceType<typeof ElForm>>()

// 提交状态
const submitting = ref(false)

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

// 组件挂载时初始化
onMounted(() => {
  // 如果用户未登录，跳转到登录页
  if (!authStore.isAuthenticated) {
    router.push('/login')
    return
  }
  initFormData()
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
</style>

