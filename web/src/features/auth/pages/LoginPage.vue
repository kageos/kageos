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
    // 🔥 统一使用 msg 字段
    const message = error?.response?.data?.msg || error?.message || '登录失败，请检查用户名和密码'
    ElMessage.error(message)
  } finally {
    loading.value = false
  }
}

// 跳转到注册页
const goToRegister = () => {
  router.push('/register')
}

// 跳转到忘记密码页
const goToForgotPassword = () => {
  router.push('/forgot-password')
}

// 处理回车键
const handleKeyPress = (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    handleLogin()
  }
}
</script>

<template>
  <div class="login-container" data-testid="login-page" @keypress="handleKeyPress">
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
          data-testid="login-form"
        >
          <el-form-item prop="username">
            <el-input
              v-model="loginForm.username"
              placeholder="请输入用户名"
              :prefix-icon="User"
              clearable
              size="large"
              class="form-input"
              data-testid="login-username"
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
              data-testid="login-password"
              @keyup.enter="handleLogin"
            />
          </el-form-item>

          <el-form-item class="login-btn-item">
            <el-button
              type="primary"
              size="large"
              :loading="loading"
              class="login-btn"
              data-testid="login-submit"
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
            <div class="footer-top">
              <el-button type="text" @click="goToForgotPassword" class="forgot-password-link">
                忘记密码？
              </el-button>
            </div>
            <div class="footer-bottom">
            <span class="login-tip">还没有账号？</span>
            <el-button type="text" @click="goToRegister" class="register-link">
              立即注册
            </el-button>
            </div>
          </div>
        </el-form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  --auth-accent: #1677ff;
  --auth-accent-strong: #0958d9;
  --auth-accent-soft: rgba(22, 119, 255, 0.14);
  --auth-brand-start: #1e2f56;
  --auth-brand-end: #35538f;
  --auth-surface: rgba(244, 247, 251, 0.88);
  --auth-card-bg: rgba(255, 255, 255, 0.8);
  --auth-card-border: rgba(148, 163, 184, 0.26);
  --auth-input-bg: rgba(255, 255, 255, 0.96);
  --auth-input-border: #cbd5e1;
  --auth-text: #0f172a;
  --auth-text-muted: #475569;
  --auth-text-soft: #64748b;
  min-height: 100vh;
  display: flex;
  background:
    linear-gradient(112deg, var(--auth-brand-start) 0%, var(--auth-brand-end) 47%, #e9eff6 47%, #f4f7fb 100%);
  position: relative;
  overflow: hidden;
  isolation: isolate;
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
  background: rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(16px);
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
  color: rgba(255, 255, 255, 0.96);
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
  background: rgba(255, 255, 255, 0.14);
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
  letter-spacing: -1.5px;
}

.title-gradient {
  background: linear-gradient(135deg, #ffffff 0%, #dbe7ff 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.brand-subtitle {
  font-size: 18px;
  line-height: 1.8;
  margin: 0 0 56px 0;
  color: rgba(226, 232, 240, 0.88);
  font-weight: 400;
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
  background: rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(14px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  transition: all 0.3s ease;
}

.feature-item:hover {
  background: rgba(255, 255, 255, 0.12);
  transform: translateX(8px);
}

.feature-icon {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.16);
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
  color: rgba(226, 232, 240, 0.72);
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
  background:
    linear-gradient(180deg, rgba(244, 247, 251, 0.78), rgba(236, 242, 248, 0.92));
  backdrop-filter: blur(18px);
  border-left: 1px solid rgba(148, 163, 184, 0.18);
}

.login-card {
  width: 100%;
  max-width: 440px;
  padding: 40px 36px 34px;
  background: var(--auth-card-bg);
  border: 1px solid var(--auth-card-border);
  border-radius: 28px;
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.14);
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
  background: linear-gradient(135deg, var(--auth-accent) 0%, var(--auth-accent-strong) 100%);
  border-radius: 16px;
  color: white;
  box-shadow: 0 12px 30px rgba(22, 119, 255, 0.26);
}

.header-icon .el-icon {
  font-size: 32px;
}

.login-title {
  font-size: 32px;
  font-weight: 700;
  color: var(--auth-text);
  margin: 0 0 8px 0;
  letter-spacing: -0.5px;
}

.login-subtitle {
  font-size: 15px;
  color: var(--auth-text-soft);
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
  background: var(--auth-input-bg);
  box-shadow: none;
  border-radius: 12px;
  padding: 0 16px;
  transition: all 0.3s ease;
  border: 1px solid var(--auth-input-border);
}

:deep(.el-input__wrapper:hover) {
  box-shadow: 0 10px 24px rgba(22, 119, 255, 0.1);
  border-color: rgba(22, 119, 255, 0.42);
}

:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 4px var(--auth-accent-soft);
  border-color: var(--auth-accent);
}

:deep(.el-input__inner) {
  height: 52px;
  font-size: 15px;
  color: var(--auth-text);
}

:deep(.el-input__inner::placeholder) {
  color: #94a3b8;
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
  background: linear-gradient(135deg, var(--auth-accent) 0%, var(--auth-accent-strong) 100%);
  border: none;
  transition: all 0.3s ease;
  box-shadow: 0 14px 30px rgba(22, 119, 255, 0.22);
}

.login-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 18px 36px rgba(22, 119, 255, 0.28);
}

.login-btn:active {
  transform: translateY(0);
}

.login-footer {
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.footer-top {
  display: flex;
  justify-content: flex-end;
}

.footer-bottom {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.forgot-password-link {
  font-size: 14px;
  color: var(--auth-text-soft);
  padding: 0;
  transition: all 0.3s ease;
}

.forgot-password-link:hover {
  color: var(--auth-accent);
}

.login-tip {
  font-size: 15px;
  color: var(--auth-text-soft);
}

.register-link {
  font-size: 15px;
  font-weight: 600;
  color: var(--auth-accent);
  padding: 0;
  transition: all 0.3s ease;
}

.register-link:hover {
  color: var(--auth-accent-strong);
  transform: translateX(2px);
}

/* Element Plus 样式覆盖 */
:deep(.el-form-item__error) {
  padding-top: 6px;
  font-size: 13px;
  color: #dc2626;
}

:deep(.el-input__prefix) {
  color: #94a3b8;
}

:deep(.el-input__suffix) {
  color: #94a3b8;
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
    border-left: none;
    border-top: 1px solid rgba(148, 163, 184, 0.18);
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

  .login-card {
    padding: 32px 24px 26px;
    border-radius: 24px;
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
