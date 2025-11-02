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
    <!-- 背景装饰 -->
    <div class="background-decoration">
      <div class="decoration-circle circle-1"></div>
      <div class="decoration-circle circle-2"></div>
      <div class="decoration-circle circle-3"></div>
    </div>

    <!-- 左侧品牌展示 -->
    <div class="login-brand">
      <div class="brand-content">
        <div class="brand-logo-wrapper">
          <div class="logo-glow"></div>
          <div class="brand-logo">
            <img alt="AI Agent OS" class="logo" src="@/assets/logo.svg" />
          </div>
        </div>
        <h1 class="brand-title">
          <span class="title-gradient">AI Agent OS</span>
        </h1>
        <p class="brand-subtitle">
          新一代智能代理操作系统<br />
          让AI应用开发像描述一样简单
        </p>
        <div class="brand-features">
          <div class="feature-item">
            <div class="feature-icon">
              <el-icon><Check /></el-icon>
            </div>
            <div class="feature-text">
              <span class="feature-title">智能代码生成</span>
              <span class="feature-desc">基于自然语言生成生产代码</span>
            </div>
          </div>
          <div class="feature-item">
            <div class="feature-icon">
              <el-icon><Check /></el-icon>
            </div>
            <div class="feature-text">
              <span class="feature-title">自动API渲染</span>
              <span class="feature-desc">零代码构建完整应用界面</span>
            </div>
          </div>
          <div class="feature-item">
            <div class="feature-icon">
              <el-icon><Check /></el-icon>
            </div>
            <div class="feature-text">
              <span class="feature-title">物理多租户</span>
              <span class="feature-desc">完全隔离的安全运行环境</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 右侧登录表单 -->
    <div class="login-form-section">
      <div class="login-card">
        <div class="card-header">
          <div class="header-icon">
            <el-icon><User /></el-icon>
          </div>
          <h2 class="login-title">欢迎回来</h2>
          <p class="login-subtitle">登录您的账号以继续使用</p>
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
              class="form-input"
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
              class="form-input"
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
              <span v-if="!loading">登录</span>
              <span v-else>登录中...</span>
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
  background: linear-gradient(135deg, #667eea 0%, #764ba2 50%, #f093fb 100%);
  position: relative;
  overflow: hidden;
}

/* 背景装饰动画 */
.background-decoration {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  overflow: hidden;
  z-index: 0;
}

.decoration-circle {
  position: absolute;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  animation: float 20s infinite ease-in-out;
}

.circle-1 {
  width: 400px;
  height: 400px;
  top: -150px;
  left: -150px;
  animation-delay: 0s;
}

.circle-2 {
  width: 300px;
  height: 300px;
  bottom: -100px;
  right: -100px;
  animation-delay: 5s;
}

.circle-3 {
  width: 250px;
  height: 250px;
  top: 50%;
  right: -50px;
  animation-delay: 10s;
}

@keyframes float {
  0%, 100% {
    transform: translate(0, 0) scale(1);
    opacity: 0.3;
  }
  33% {
    transform: translate(30px, -30px) scale(1.1);
    opacity: 0.5;
  }
  66% {
    transform: translate(-20px, 20px) scale(0.9);
    opacity: 0.4;
  }
}

.login-brand {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 80px;
  position: relative;
  z-index: 1;
}

.brand-content {
  text-align: center;
  color: white;
  max-width: 500px;
  animation: fadeInUp 0.8s ease-out;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.brand-logo-wrapper {
  position: relative;
  display: inline-block;
  margin-bottom: 32px;
}

.logo-glow {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 120px;
  height: 120px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 50%;
  filter: blur(20px);
  animation: pulse 3s infinite;
}

@keyframes pulse {
  0%, 100% {
    transform: translate(-50%, -50%) scale(1);
    opacity: 0.3;
  }
  50% {
    transform: translate(-50%, -50%) scale(1.2);
    opacity: 0.5;
  }
}

.brand-logo {
  position: relative;
  z-index: 1;
  margin-bottom: 0;
}

.logo {
  width: 80px;
  height: 80px;
  filter: brightness(0) invert(1);
  animation: rotate 20s linear infinite;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.brand-title {
  font-size: 48px;
  font-weight: 800;
  margin: 0 0 16px 0;
  letter-spacing: -1px;
}

.title-gradient {
  background: linear-gradient(135deg, #ffffff 0%, #e0e7ff 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.brand-subtitle {
  font-size: 18px;
  line-height: 1.8;
  margin: 0 0 56px 0;
  color: rgba(255, 255, 255, 0.95);
  font-weight: 300;
}

.brand-features {
  text-align: left;
  max-width: 400px;
  margin: 0 auto;
}

.feature-item {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 28px;
  padding: 20px;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  transition: all 0.3s ease;
}

.feature-item:hover {
  background: rgba(255, 255, 255, 0.15);
  transform: translateX(8px);
}

.feature-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 12px;
  flex-shrink: 0;
}

.feature-icon .el-icon {
  color: #fff;
  font-size: 20px;
  font-weight: bold;
}

.feature-text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.feature-title {
  font-size: 16px;
  font-weight: 600;
  color: white;
  display: block;
}

.feature-desc {
  font-size: 13px;
  color: rgba(255, 255, 255, 0.8);
  display: block;
  line-height: 1.5;
}

.login-form-section {
  width: 600px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 80px 60px;
  position: relative;
  z-index: 1;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(20px);
  box-shadow: -10px 0 40px rgba(0, 0, 0, 0.1);
}

.login-card {
  width: 100%;
  max-width: 440px;
  animation: slideInRight 0.8s ease-out;
}

@keyframes slideInRight {
  from {
    opacity: 0;
    transform: translateX(30px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

.card-header {
  text-align: center;
  margin-bottom: 48px;
}

.header-icon {
  width: 64px;
  height: 64px;
  margin: 0 auto 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px;
  color: white;
}

.header-icon .el-icon {
  font-size: 32px;
}

.login-title {
  font-size: 32px;
  font-weight: 700;
  color: #1a202c;
  margin: 0 0 8px 0;
  letter-spacing: -0.5px;
}

.login-subtitle {
  font-size: 15px;
  color: #718096;
  margin: 0;
  font-weight: 400;
}

.login-form {
  margin-bottom: 32px;
}

:deep(.el-form-item) {
  margin-bottom: 28px;
}

:deep(.el-input__wrapper) {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  padding: 0 16px;
  transition: all 0.3s ease;
  border: 1px solid #e2e8f0;
}

:deep(.el-input__wrapper:hover) {
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.15);
  border-color: #667eea;
}

:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 4px 16px rgba(102, 126, 234, 0.2);
  border-color: #667eea;
}

:deep(.el-input__inner) {
  height: 52px;
  font-size: 15px;
  color: #1a202c;
}

:deep(.el-input__inner::placeholder) {
  color: #a0aec0;
}

.login-btn-item {
  margin-bottom: 32px;
}

.login-btn {
  width: 100%;
  height: 52px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 12px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  transition: all 0.3s ease;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
}

.login-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.4);
}

.login-btn:active {
  transform: translateY(0);
}

.login-footer {
  text-align: center;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.login-tip {
  font-size: 15px;
  color: #718096;
}

.register-link {
  font-size: 15px;
  font-weight: 600;
  color: #667eea;
  padding: 0;
  transition: all 0.3s ease;
}

.register-link:hover {
  color: #764ba2;
  transform: translateX(2px);
}

/* Element Plus 样式覆盖 */
:deep(.el-form-item__error) {
  padding-top: 6px;
  font-size: 13px;
  color: #f56565;
}

:deep(.el-input__prefix) {
  color: #a0aec0;
}

:deep(.el-input__suffix) {
  color: #a0aec0;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .login-brand {
    padding: 0 60px;
  }

  .login-form-section {
    width: 520px;
    padding: 60px 40px;
  }
}

@media (max-width: 968px) {
  .login-container {
    flex-direction: column;
  }

  .login-brand {
    width: 100%;
    padding: 80px 40px 60px;
    min-height: auto;
    flex: none;
  }

  .brand-title {
    font-size: 40px;
  }

  .brand-subtitle {
    font-size: 16px;
    margin-bottom: 40px;
  }

  .brand-features {
    max-width: 100%;
  }

  .login-form-section {
    width: 100%;
    padding: 60px 40px;
    box-shadow: 0 -10px 40px rgba(0, 0, 0, 0.1);
  }
}

@media (max-width: 640px) {
  .login-brand {
    padding: 60px 20px 40px;
  }

  .brand-title {
    font-size: 32px;
  }

  .brand-subtitle {
    font-size: 14px;
  }

  .feature-item {
    padding: 16px;
    margin-bottom: 16px;
  }

  .login-form-section {
    padding: 40px 24px;
  }

  .login-title {
    font-size: 28px;
  }

  .header-icon {
    width: 56px;
    height: 56px;
  }

  .header-icon .el-icon {
    font-size: 28px;
  }
}

@media (max-width: 480px) {
  .decoration-circle {
    display: none;
  }
}
</style>