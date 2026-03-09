<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Loading } from '@element-plus/icons-vue'
import { createUserBySecret } from '@/api/auth'

const router = useRouter()

const form = reactive({
  username: '',
  password: ''
})

const formRef = ref()
const loading = ref(false)

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '用户名 3～20 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' }
  ]
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    loading.value = true
    await createUserBySecret({
      username: form.username,
      password: form.password
    })
    ElMessage.success('创建成功，可使用该账号登录')
    router.push('/login')
  } catch (e: any) {
    const msg = e?.response?.data?.msg || e?.message || '创建失败'
    ElMessage.error(msg)
  } finally {
    loading.value = false
  }
}

const goLogin = () => router.push('/login')
</script>

<template>
  <div class="create-test-user-container">
    <div class="background-decoration">
      <div class="decoration-circle circle-1"></div>
      <div class="decoration-circle circle-2"></div>
      <div class="decoration-circle circle-3"></div>
    </div>

    <div class="form-section">
      <div class="card">
        <div class="card-header">
          <div class="header-icon">
            <el-icon><User /></el-icon>
          </div>
          <h2 class="title">创建测试用户</h2>
          <p class="subtitle">仅 system 超管可操作，一键创建用户无需邮箱验证</p>
        </div>

        <el-form ref="formRef" :model="form" :rules="rules" label-width="0" size="large" class="form">
          <el-form-item prop="username">
            <el-input
              v-model="form.username"
              placeholder="用户名（3～20 字符）"
              :prefix-icon="User"
              clearable
            />
          </el-form-item>
          <el-form-item prop="password">
            <el-input
              v-model="form.password"
              type="password"
              placeholder="密码（至少 6 位）"
              :prefix-icon="Lock"
              show-password
              clearable
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="large" :loading="loading" class="submit-btn" @click="handleSubmit">
              <template #loading>
                <el-icon class="is-loading"><Loading /></el-icon>
              </template>
              {{ loading ? '创建中...' : '创建用户' }}
            </el-button>
          </el-form-item>
        </el-form>

        <div class="footer">
          <el-button type="text" @click="goLogin">返回登录</el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.create-test-user-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  background: linear-gradient(135deg, #0f0f1a 0%, #1a1a2e 50%, #16213e 100%);
}

.background-decoration {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.decoration-circle {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.4;
}

.circle-1 { width: 400px; height: 400px; background: #6366f1; top: -100px; right: -100px; }
.circle-2 { width: 300px; height: 300px; background: #8b5cf6; bottom: 20%; left: -80px; }
.circle-3 { width: 200px; height: 200px; background: #a855f7; bottom: -50px; right: 20%; }

.form-section {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 420px;
  padding: 24px;
}

.card {
  background: rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(16px);
  border-radius: 20px;
  padding: 36px;
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.card-header {
  text-align: center;
  margin-bottom: 28px;
}

.header-icon {
  width: 56px;
  height: 56px;
  margin: 0 auto 16px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  color: #fff;
}

.title {
  font-size: 22px;
  font-weight: 600;
  color: #fff;
  margin: 0 0 8px;
}

.subtitle {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.6);
  margin: 0;
  line-height: 1.5;
}

.form :deep(.el-input__wrapper) {
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  box-shadow: none;
}

.form :deep(.el-input__wrapper:hover),
.form :deep(.el-input__wrapper.is-focus) {
  border-color: rgba(99, 102, 241, 0.5);
}

.submit-btn {
  width: 100%;
  height: 48px;
  border-radius: 12px;
  font-size: 16px;
}

.footer {
  text-align: center;
  margin-top: 16px;
}

.footer .el-button {
  color: rgba(255, 255, 255, 0.7);
}
</style>
