<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Message, Check, Loading } from '@element-plus/icons-vue'
import { register as registerApi, sendEmailCode } from '@/api/auth'
import type { RegisterRequest } from '@/types'

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
    const message = error?.response?.data?.message || error?.message || '注册失败，请重试'
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
    const message = error?.response?.data?.message || error?.message || '发送验证码失败，请重试'
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
            <img alt="AI Agent OS" class="logo" src="@/assets/logo.svg" />
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
  min-height: 100vh;
  display: flex;
  background: linear-gradient(135deg, #22c55e 0%, #16a34a 50%, #15803d 100%);
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
  background: linear-gradient(135deg, #ffffff 0%, #dcfce7 100%);
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
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  transition: all 0.3s ease;
}

.step-item:hover {
  background: rgba(255, 255, 255, 0.15);
  transform: translateX(8px);
}

.step-icon {
  flex-shrink: 0;
}

.step-number {
  width: 48px;
  height: 48px;
  background: rgba(255, 255, 255, 0.2);
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
  color: rgba(255, 255, 255, 0.8);
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
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(20px);
  box-shadow: -10px 0 40px rgba(0, 0, 0, 0.1);
}

.register-card {
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
  background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
  border-radius: 16px;
  color: white;
}

.header-icon .el-icon {
  font-size: 32px;
}

.register-title {
  font-size: 32px;
  font-weight: 700;
  color: #1a202c;
  margin: 0 0 8px 0;
  letter-spacing: -0.5px;
}

.register-subtitle {
  font-size: 15px;
  color: #718096;
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
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  border-radius: 12px;
  padding: 0 16px;
  transition: all 0.3s ease;
  border: 1px solid #e2e8f0;
}

:deep(.el-input__wrapper:hover) {
  box-shadow: 0 4px 12px rgba(34, 197, 94, 0.15);
  border-color: #22c55e;
}

:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 4px 16px rgba(34, 197, 94, 0.2);
  border-color: #22c55e;
}

:deep(.el-input__inner) {
  height: 52px;
  font-size: 15px;
  color: #1a202c;
}

:deep(.el-input__inner::placeholder) {
  color: #a0aec0;
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
  background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
  border: none;
  transition: all 0.3s ease;
}

.code-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(34, 197, 94, 0.4);
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
  background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
  border: none;
  transition: all 0.3s ease;
  box-shadow: 0 4px 12px rgba(34, 197, 94, 0.3);
}

.register-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(34, 197, 94, 0.4);
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
  color: #718096;
}

.login-link {
  font-size: 15px;
  font-weight: 600;
  color: #22c55e;
  padding: 0;
  transition: all 0.3s ease;
}

.login-link:hover {
  color: #16a34a;
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
    box-shadow: 0 -10px 40px rgba(0, 0, 0, 0.1);
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