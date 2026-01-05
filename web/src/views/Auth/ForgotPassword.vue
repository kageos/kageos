<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Message, Loading, Lock } from '@element-plus/icons-vue'
import { forgotPassword, sendEmailCode } from '@/api/auth'

const router = useRouter()

// 表单数据
const formData = reactive({
  email: '',
  code: '',
  password: '',
  confirmPassword: ''
})

// 表单引用
const formRef = ref()

// 加载状态
const loading = ref(false)
const codeLoading = ref(false)
const countdown = ref(0)

// 表单验证规则
const validateConfirmPassword = (rule: any, value: string, callback: Function) => {
  if (!value) {
    callback(new Error('请再次输入密码'))
  } else if (value !== formData.password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const rules = {
  email: [
    { required: true, message: '请输入邮箱地址', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱地址', trigger: 'blur' }
  ],
  code: [
    { required: true, message: '请输入验证码', trigger: 'blur' },
    { len: 6, message: '验证码长度为6位', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, max: 50, message: '密码长度在 6 到 50 个字符', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

// 发送验证码
const handleSendCode = async () => {
  if (!formData.email) {
    ElMessage.warning('请先输入邮箱地址')
    return
  }

  try {
    codeLoading.value = true
    await sendEmailCode(formData.email, 'forgot_password')
    ElMessage.success('验证码已发送到您的邮箱，请查收')
    
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
    const message = error?.response?.data?.msg || error?.message || '发送验证码失败'
    ElMessage.error(message)
  } finally {
    codeLoading.value = false
  }
}

// 提交忘记密码请求
const handleSubmit = async () => {
  try {
    await formRef.value.validate()
    loading.value = true

    await forgotPassword({
      email: formData.email,
      code: formData.code,
      password: formData.password
    })

    ElMessage.success('密码重置成功，请使用新密码登录')
    
    // 跳转到登录页
    setTimeout(() => {
      router.push('/login')
    }, 2000)
  } catch (error: any) {
    console.error('忘记密码失败:', error)
    if (error?.errors) {
      // 表单验证错误，不显示错误消息
      return
    }
    const message = error?.response?.data?.msg || error?.message || '操作失败，请重试'
    ElMessage.error(message)
  } finally {
    loading.value = false
  }
}

// 返回登录页
const goToLogin = () => {
  router.push('/login')
}

// 处理回车键
const handleKeyPress = (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    handleSubmit()
  }
}
</script>

<template>
  <div class="forgot-password-container" @keypress="handleKeyPress">
    <!-- 背景装饰 -->
    <div class="background-decoration">
      <div class="decoration-circle circle-1"></div>
      <div class="decoration-circle circle-2"></div>
      <div class="decoration-circle circle-3"></div>
    </div>

    <!-- 左侧品牌展示 -->
    <div class="forgot-password-brand">
      <div class="brand-content">
        <div class="brand-logo-wrapper">
          <div class="logo-glow"></div>
          <div class="brand-logo">
            <img alt="AI Agent OS" class="logo" src="@/assets/logo.svg" />
          </div>
        </div>
        <h1 class="brand-title">
          <span class="title-gradient">忘记密码</span>
        </h1>
        <p class="brand-subtitle">
          请输入您的邮箱地址和验证码<br />
          设置新密码即可完成重置
        </p>
      </div>
    </div>

    <!-- 右侧表单 -->
    <div class="forgot-password-form-section">
      <div class="forgot-password-card">
        <div class="card-header">
          <div class="header-icon">
            <el-icon><Message /></el-icon>
          </div>
          <h2 class="form-title">找回密码</h2>
          <p class="form-subtitle">验证邮箱并设置新密码</p>
        </div>

        <el-form
          ref="formRef"
          :model="formData"
          :rules="rules"
          label-width="0"
          size="large"
          class="forgot-password-form"
        >
          <el-form-item prop="email">
            <el-input
              v-model="formData.email"
              placeholder="请输入邮箱地址"
              :prefix-icon="Message"
              clearable
              size="large"
              class="form-input"
            />
          </el-form-item>

          <el-form-item prop="code">
            <div class="code-input-wrapper">
              <el-input
                v-model="formData.code"
                placeholder="请输入验证码"
                maxlength="6"
                clearable
                size="large"
                class="form-input code-input"
              />
              <el-button
                :disabled="countdown > 0"
                :loading="codeLoading"
                @click="handleSendCode"
                class="code-button"
              >
                <template #loading>
                  <el-icon class="is-loading"><Loading /></el-icon>
                </template>
                <span v-if="countdown > 0">{{ countdown }}秒后重试</span>
                <span v-else>发送验证码</span>
              </el-button>
            </div>
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              v-model="formData.password"
              type="password"
              placeholder="请输入新密码"
              :prefix-icon="Lock"
              show-password
              clearable
              size="large"
              class="form-input"
            />
          </el-form-item>

          <el-form-item prop="confirmPassword">
            <el-input
              v-model="formData.confirmPassword"
              type="password"
              placeholder="请再次输入新密码"
              :prefix-icon="Lock"
              show-password
              clearable
              size="large"
              class="form-input"
              @keyup.enter="handleSubmit"
            />
          </el-form-item>

          <el-form-item class="submit-btn-item">
            <el-button
              type="primary"
              size="large"
              :loading="loading"
              class="submit-btn"
              @click="handleSubmit"
            >
              <template #loading>
                <el-icon class="is-loading"><Loading /></el-icon>
              </template>
              <span v-if="!loading">提交</span>
              <span v-else>提交中...</span>
            </el-button>
          </el-form-item>

          <div class="form-footer">
            <el-button type="text" @click="goToLogin" class="back-link">
              ← 返回登录
            </el-button>
          </div>
        </el-form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.forgot-password-container {
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

.forgot-password-brand {
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
  margin: 0;
  color: rgba(255, 255, 255, 0.95);
  font-weight: 300;
}

.forgot-password-form-section {
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

.forgot-password-card {
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

.form-title {
  font-size: 32px;
  font-weight: 700;
  color: #1a202c;
  margin: 0 0 8px 0;
  letter-spacing: -0.5px;
}

.form-subtitle {
  font-size: 15px;
  color: #718096;
  margin: 0;
  font-weight: 400;
}

.forgot-password-form {
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

.code-input-wrapper {
  display: flex;
  gap: 12px;
}

.code-input {
  flex: 1;
}

.code-button {
  flex-shrink: 0;
  white-space: nowrap;
  border-radius: 12px;
  height: 52px;
  padding: 0 20px;
}

.submit-btn-item {
  margin-bottom: 32px;
}

.submit-btn {
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

.submit-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.4);
}

.submit-btn:active {
  transform: translateY(0);
}

.form-footer {
  text-align: center;
}

.back-link {
  font-size: 15px;
  font-weight: 600;
  color: #667eea;
  padding: 0;
  transition: all 0.3s ease;
}

.back-link:hover {
  color: #764ba2;
  transform: translateX(-2px);
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
  .forgot-password-brand {
    padding: 0 60px;
  }

  .forgot-password-form-section {
    width: 520px;
    padding: 60px 40px;
  }
}

@media (max-width: 968px) {
  .forgot-password-container {
    flex-direction: column;
  }

  .forgot-password-brand {
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
  }

  .forgot-password-form-section {
    width: 100%;
    padding: 60px 40px;
    box-shadow: 0 -10px 40px rgba(0, 0, 0, 0.1);
  }
}

@media (max-width: 640px) {
  .forgot-password-brand {
    padding: 60px 20px 40px;
  }

  .brand-title {
    font-size: 32px;
  }

  .brand-subtitle {
    font-size: 14px;
  }

  .forgot-password-form-section {
    padding: 40px 24px;
  }

  .form-title {
    font-size: 28px;
  }

  .header-icon {
    width: 56px;
    height: 56px;
  }

  .header-icon .el-icon {
    font-size: 28px;
  }

  .code-input-wrapper {
    flex-direction: column;
  }

  .code-button {
    width: 100%;
  }
}

@media (max-width: 480px) {
  .decoration-circle {
    display: none;
  }
}
</style>

