<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Message, Check, Loading } from '@element-plus/icons-vue'
import { register as registerApi, sendEmailCode } from '@/architecture/infrastructure/api/auth'
import type { RegisterRequest } from '@/architecture/domain/types'

const router = useRouter()

// 表单数据
const registerForm = reactive<RegisterRequest>({
  username: '',
  email: '',
  password: '',
  code: ''
})

// 表单引用
const registerFormRef = ref()

// 加载状态
const loading = ref(false)
const sendingCode = ref(false)
const countdown = ref(0)

// 表单验证规则
const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 50, message: '用户名长度在 2 到 50 个字符', trigger: 'blur' },
    { pattern: /^[a-zA-Z0-9_]+$/, message: '用户名只能包含字母、数字和下划线', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '请输入邮箱地址', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 50, message: '密码长度在 6 到 50 个字符', trigger: 'blur' }
  ],
  code: [
    { required: true, message: '请输入验证码', trigger: 'blur' },
    { len: 6, message: '验证码长度为 6 位', trigger: 'blur' }
  ]
}

// 处理注册
const handleRegister = async () => {
  try {
    await registerFormRef.value.validate()
    loading.value = true

    await registerApi(registerForm)

    ElMessage.success('注册成功！请登录')
    await router.push('/login')
  } catch (error: any) {
    console.error('注册失败:', error)
    // 🔥 统一使用 msg 字段
    const message = error?.response?.data?.msg || error?.message || '注册失败，请重试'
    ElMessage.error(message)
  } finally {
    loading.value = false
  }
}

// 发送验证码
const sendVerificationCode = async () => {
  if (!registerForm.email) {
    ElMessage.warning('请先输入邮箱地址')
    return
  }

  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(registerForm.email)) {
    ElMessage.warning('请输入正确的邮箱地址')
    return
  }

  try {
    sendingCode.value = true
    await sendEmailCode(registerForm.email)
    ElMessage.success('验证码已发送到您的邮箱')

    // 开始倒计时
    countdown.value = 60
    const timer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        clearInterval(timer)
      }
    }, 1000)
  } catch (error: any) {
    console.error('发送验证码失败:', error)
    // 🔥 统一使用 msg 字段
    const message = error?.response?.data?.msg || error?.message || '发送验证码失败，请重试'
    ElMessage.error(message)
  } finally {
    sendingCode.value = false
  }
}

// 跳转到登录页
const goToLogin = () => {
  router.push('/login')
}

// 处理回车键
const handleKeyPress = (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    handleRegister()
  }
}
</script>

<template>
  <div class="register-container" @keypress="handleKeyPress">
    <!-- 背景装饰 -->
    <div class="background-decoration">
      <div class="decoration-circle circle-1"></div>
      <div class="decoration-circle circle-2"></div>
      <div class="decoration-circle circle-3"></div>
    </div>

    <!-- 左侧品牌展示 -->
    <div class="register-brand">
      <div class="brand-content">
        <div class="brand-logo-wrapper">
          <div class="logo-glow"></div>
          <div class="brand-logo">
            <img alt="AI Agent OS" class="logo" src="@/architecture/presentation/assets/logo.svg" />
          </div>
        </div>
        <h1 class="brand-title">
          <span class="title-gradient">加入我们</span>
        </h1>
        <p class="brand-subtitle">
          开启AI应用开发的新旅程<br />
          描述即生成，想法即产品
        </p>
        <div class="brand-steps">
          <div class="step-item">
            <div class="step-icon">
              <div class="step-number">1</div>
            </div>
            <div class="step-content">
              <span class="step-title">注册账号</span>
              <span class="step-desc">快速创建您的专属账号</span>
            </div>
          </div>
          <div class="step-item">
            <div class="step-icon">
              <div class="step-number">2</div>
            </div>
            <div class="step-content">
              <span class="step-title">验证邮箱</span>
              <span class="step-desc">确保账号安全可靠</span>
            </div>
          </div>
          <div class="step-item">
            <div class="step-icon">
              <div class="step-number">3</div>
            </div>
            <div class="step-content">
              <span class="step-title">开始使用</span>
              <span class="step-desc">创造您的第一个AI应用</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 右侧注册表单 -->
    <div class="register-form-section">
      <div class="register-card">
        <div class="card-header">
          <div class="header-icon">
            <el-icon><User /></el-icon>
          </div>
          <h2 class="register-title">创建新账号</h2>
          <p class="register-subtitle">填写您的信息以完成注册</p>
        </div>

        <el-form
          ref="registerFormRef"
          :model="registerForm"
          :rules="rules"
          label-width="0"
          size="large"
          class="register-form"
        >
          <el-form-item prop="username">
            <el-input
              v-model="registerForm.username"
              placeholder="请输入用户名"
              :prefix-icon="User"
              clearable
              size="large"
              class="form-input"
            />
          </el-form-item>

          <el-form-item prop="email">
            <el-input
              v-model="registerForm.email"
              placeholder="请输入邮箱地址"
              :prefix-icon="Message"
              clearable
              size="large"
              class="form-input"
            />
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              v-model="registerForm.password"
              type="password"
              placeholder="请输入密码"
              :prefix-icon="Lock"
              show-password
              clearable
              size="large"
              class="form-input"
            />
          </el-form-item>

          <el-form-item prop="code">
            <div class="code-input-group">
              <el-input
                v-model="registerForm.code"
                placeholder="请输入验证码"
                maxlength="6"
                clearable
                size="large"
                class="form-input"
              />
              <el-button
                type="primary"
                size="large"
                :disabled="countdown > 0 || sendingCode"
                :loading="sendingCode"
                @click="sendVerificationCode"
                class="code-btn"
              >
                {{ countdown > 0 ? `${countdown}s` : '发送验证码' }}
              </el-button>
            </div>
          </el-form-item>

          <el-form-item class="register-btn-item">
            <el-button
              type="primary"
              size="large"
              :loading="loading"
              class="register-btn"
              @click="handleRegister"
            >
              <template #loading>
                <el-icon class="is-loading"><Loading /></el-icon>
              </template>
              <span v-if="!loading">立即注册</span>
              <span v-else>注册中...</span>
            </el-button>
          </el-form-item>

          <div class="register-footer">
            <span class="register-tip">已有账号？</span>
            <el-button type="text" @click="goToLogin" class="login-link">
              立即登录
            </el-button>
          </div>
        </el-form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.register-container {
  --auth-accent: #16a34a;
  --auth-accent-strong: #15803d;
  --auth-accent-soft: rgba(22, 163, 74, 0.14);
  --auth-brand-start: #183c2b;
  --auth-brand-end: #256246;
  --auth-surface: rgba(244, 247, 251, 0.88);
  --auth-card-bg: rgba(255, 255, 255, 0.8);
  --auth-card-border: rgba(148, 163, 184, 0.24);
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

.register-brand {
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
  background: linear-gradient(135deg, #ffffff 0%, #d6f5df 100%);
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

.brand-steps {
  text-align: left;
  max-width: 400px;
  margin: 0 auto;
}

.step-item {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 24px;
  padding: 20px;
  background: rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(14px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  transition: all 0.3s ease;
}

.step-item:hover {
  background: rgba(255, 255, 255, 0.12);
  transform: translateX(8px);
}

.step-icon {
  flex-shrink: 0;
}

.step-number {
  width: 48px;
  height: 48px;
  background: rgba(255, 255, 255, 0.16);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 18px;
  color: white;
}

.step-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.step-title {
  font-size: 16px;
  font-weight: 600;
  color: white;
  display: block;
}

.step-desc {
  font-size: 13px;
  color: rgba(226, 232, 240, 0.72);
  display: block;
  line-height: 1.5;
}

.register-form-section {
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

.register-card {
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
  box-shadow: 0 12px 30px rgba(22, 163, 74, 0.24);
}

.header-icon .el-icon {
  font-size: 32px;
}

.register-title {
  font-size: 32px;
  font-weight: 700;
  color: var(--auth-text);
  margin: 0 0 8px 0;
  letter-spacing: -0.5px;
}

.register-subtitle {
  font-size: 15px;
  color: var(--auth-text-soft);
  margin: 0;
  font-weight: 400;
}

.register-form {
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
  box-shadow: 0 10px 24px rgba(22, 163, 74, 0.1);
  border-color: rgba(22, 163, 74, 0.42);
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

.code-input-group {
  display: flex;
  gap: 12px;
}

.code-input-group .el-input {
  flex: 1;
}

.code-btn {
  white-space: nowrap;
  min-width: 140px;
  border-radius: 12px;
  height: 52px;
  font-weight: 600;
  background: linear-gradient(135deg, var(--auth-accent) 0%, var(--auth-accent-strong) 100%);
  border: none;
  transition: all 0.3s ease;
  box-shadow: 0 12px 28px rgba(22, 163, 74, 0.18);
}

.code-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 16px 34px rgba(22, 163, 74, 0.26);
}

.code-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.register-btn-item {
  margin-bottom: 32px;
}

.register-btn {
  width: 100%;
  height: 52px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--auth-accent) 0%, var(--auth-accent-strong) 100%);
  border: none;
  transition: all 0.3s ease;
  box-shadow: 0 14px 30px rgba(22, 163, 74, 0.22);
}

.register-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 18px 36px rgba(22, 163, 74, 0.28);
}

.register-btn:active {
  transform: translateY(0);
}

.register-footer {
  text-align: center;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.register-tip {
  font-size: 15px;
  color: var(--auth-text-soft);
}

.login-link {
  font-size: 15px;
  font-weight: 600;
  color: var(--auth-accent);
  padding: 0;
  transition: all 0.3s ease;
}

.login-link:hover {
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
  .register-brand {
    padding: 0 60px;
  }

  .register-form-section {
    width: 520px;
    padding: 60px 40px;
  }
}

@media (max-width: 968px) {
  .register-container {
    flex-direction: column;
  }

  .register-brand {
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

  .brand-steps {
    max-width: 100%;
  }

  .register-form-section {
    width: 100%;
    padding: 60px 40px;
    border-left: none;
    border-top: 1px solid rgba(148, 163, 184, 0.18);
  }
}

@media (max-width: 640px) {
  .register-brand {
    padding: 60px 20px 40px;
  }

  .brand-title {
    font-size: 32px;
  }

  .brand-subtitle {
    font-size: 14px;
  }

  .step-item {
    padding: 16px;
    margin-bottom: 16px;
  }

  .register-form-section {
    padding: 40px 24px;
  }

  .register-card {
    padding: 32px 24px 26px;
    border-radius: 24px;
  }

  .register-title {
    font-size: 28px;
  }

  .header-icon {
    width: 56px;
    height: 56px;
  }

  .header-icon .el-icon {
    font-size: 28px;
  }

  .code-input-group {
    flex-direction: column;
    gap: 12px;
  }

  .code-btn {
    width: 100%;
    min-width: auto;
  }
}

@media (max-width: 480px) {
  .decoration-circle {
    display: none;
  }
}
</style>
