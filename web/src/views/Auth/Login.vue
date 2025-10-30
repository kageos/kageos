<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Check, Loading } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import type { LoginRequest } from '@/types'

const router = useRouter()
const authStore = useAuthStore()

// 表单数据
const loginForm = reactive<LoginRequest>({
  username: '',
  password: ''
})

// 表单引用
const loginFormRef = ref()

// 加载状态
const loading = ref(false)

// 表单验证规则
const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 50, message: '用户名长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 50, message: '密码长度在 6 到 50 个字符', trigger: 'blur' }
  ]
}

// 处理登录
const handleLogin = async () => {
  try {
    await loginFormRef.value.validate()
    loading.value = true

    await authStore.login(loginForm)

    // 登录成功后跳转到首页
    await router.push('/')
  } catch (error: any) {
    console.error('登录失败:', error)
    const message = error?.response?.data?.message || error?.message || '登录失败，请检查用户名和密码'
    ElMessage.error(message)
  } finally {
    loading.value = false
  }
}

// 跳转到注册页
const goToRegister = () => {
  router.push('/register')
}

// 处理回车键
const handleKeyPress = (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    handleLogin()
  }
}
</script>

<template>
  <div class="login-container" @keypress="handleKeyPress">
    <!-- 左侧品牌展示 -->
    <div class="login-brand">
      <div class="brand-content">
        <div class="brand-logo">
          <img alt="AI Agent OS" class="logo" src="@/assets/logo.svg" />
        </div>
        <h1 class="brand-title">AI Agent OS</h1>
        <p class="brand-subtitle">
          新一代智能代理操作系统<br />
          为企业提供完整的AI代理管理和服务治理解决方案
        </p>
        <div class="brand-features">
          <div class="feature-item">
            <el-icon><Check /></el-icon>
            <span>智能代理管理</span>
          </div>
          <div class="feature-item">
            <el-icon><Check /></el-icon>
            <span>服务目录管理</span>
          </div>
          <div class="feature-item">
            <el-icon><Check /></el-icon>
            <span>实时监控系统</span>
          </div>
          <div class="feature-item">
            <el-icon><Check /></el-icon>
            <span>API网关服务</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 右侧登录表单 -->
    <div class="login-form-section">
      <div class="login-card">
        <div class="login-header">
          <h2>欢迎登录</h2>
          <p>请输入您的账号信息进行登录</p>
        </div>

        <el-form
          ref="loginFormRef"
          :model="loginForm"
          :rules="rules"
          label-width="0"
          size="large"
          class="login-form"
        >
          <el-form-item prop="username">
            <el-input
              v-model="loginForm.username"
              placeholder="请输入用户名"
              :prefix-icon="User"
              clearable
              size="large"
            />
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              v-model="loginForm.password"
              type="password"
              placeholder="请输入密码"
              :prefix-icon="Lock"
              show-password
              clearable
              size="large"
              @keyup.enter="handleLogin"
            />
          </el-form-item>

          <el-form-item class="login-btn-item">
            <el-button
              type="primary"
              size="large"
              :loading="loading"
              class="login-btn"
              @click="handleLogin"
            >
              <template #loading>
                <el-icon class="is-loading"><Loading /></el-icon>
              </template>
              登录
            </el-button>
          </el-form-item>

          <div class="login-footer">
            <span class="login-tip">还没有账号？</span>
            <el-button type="text" @click="goToRegister" class="register-link">
              立即注册
            </el-button>
          </div>
        </el-form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  background: #f0f2f5;
}

.login-brand {
  flex: 1;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 60px;
  position: relative;
  overflow: hidden;
}

.login-brand::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: url('data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1440 320"><path fill="%23ffffff" fill-opacity="0.1" d="M0,96L48,112C96,128,192,160,288,160C384,160,480,128,576,122.7C672,117,768,139,864,133.3C960,128,1056,85,1152,80C1248,75,1344,107,1392,122.7L1440,138.7L1440,320L1392,320C1344,320,1248,320,1152,320C1056,320,960,320,864,320C768,320,672,320,576,320C480,320,384,320,288,320C192,320,96,320,48,320L0,320Z"></path></svg>') no-repeat bottom;
  background-size: cover;
}

.brand-content {
  text-align: center;
  color: white;
  z-index: 1;
  position: relative;
}

.brand-logo {
  margin-bottom: 24px;
}

.logo {
  width: 80px;
  height: 80px;
  filter: brightness(0) invert(1);
}

.brand-title {
  font-size: 36px;
  font-weight: 700;
  margin: 0 0 16px 0;
  color: white;
  letter-spacing: -0.5px;
}

.brand-subtitle {
  font-size: 16px;
  line-height: 1.6;
  margin: 0 0 48px 0;
  color: rgba(255, 255, 255, 0.9);
  max-width: 400px;
}

.brand-features {
  text-align: left;
  max-width: 300px;
  margin: 0 auto;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  color: rgba(255, 255, 255, 0.9);
  font-size: 14px;
}

.feature-item .el-icon {
  color: #4ade80;
  font-size: 16px;
}

.login-form-section {
  width: 520px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: white;
  padding: 60px;
}

.login-card {
  width: 100%;
  max-width: 400px;
}

.login-header {
  text-align: center;
  margin-bottom: 40px;
}

.login-header h2 {
  font-size: 28px;
  font-weight: 600;
  color: #1f2937;
  margin: 0 0 8px 0;
}

.login-header p {
  font-size: 14px;
  color: #6b7280;
  margin: 0;
}

.login-form {
  margin-bottom: 24px;
}

:deep(.el-form-item) {
  margin-bottom: 24px;
}

:deep(.el-input__wrapper) {
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  padding: 0 16px;
}

:deep(.el-input__inner) {
  height: 48px;
  font-size: 15px;
}

.login-btn-item {
  margin-bottom: 32px;
}

.login-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 500;
  border-radius: 8px;
}

.login-footer {
  text-align: center;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.login-tip {
  font-size: 14px;
  color: #6b7280;
}

.register-link {
  font-size: 14px;
  font-weight: 500;
  color: #3b82f6;
  padding: 0;
}

.register-link:hover {
  color: #2563eb;
}

/* Element Plus 样式覆盖 */
:deep(.el-form-item__error) {
  padding-top: 4px;
  font-size: 12px;
}

:deep(.el-input__prefix) {
  color: #9ca3af;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .login-brand {
    padding: 0 40px;
  }

  .login-form-section {
    width: 480px;
    padding: 40px;
  }
}

@media (max-width: 768px) {
  .login-container {
    flex-direction: column;
  }

  .login-brand {
    width: 100%;
    padding: 60px 20px;
    min-height: auto;
  }

  .brand-content {
    max-width: 400px;
  }

  .brand-title {
    font-size: 28px;
  }

  .brand-subtitle {
    font-size: 14px;
  }

  .login-form-section {
    width: 100%;
    padding: 40px 20px;
  }

  .login-card {
    max-width: 100%;
  }
}

@media (max-width: 480px) {
  .brand-features {
    display: none;
  }

  .login-form-section {
    padding: 30px 20px;
  }
}
</style>