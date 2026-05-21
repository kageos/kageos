<script setup lang="ts">
import { computed, ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Message, Loading, Lock } from '@element-plus/icons-vue'
import { forgotPassword, sendEmailCode } from '@/architecture/presentation/context/api/auth'
import LanguageSwitcher from '@/architecture/presentation/components/LanguageSwitcher.vue'

const router = useRouter()
const { t } = useI18n()

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
    callback(new Error(t('auth.confirmPasswordRequired')))
  } else if (value !== formData.password) {
    callback(new Error(t('auth.confirmPasswordMismatch')))
  } else {
    callback()
  }
}

const rules = computed(() => ({
  email: [
    { required: true, message: t('auth.emailRequired'), trigger: 'blur' },
    { type: 'email', message: t('auth.emailInvalid'), trigger: 'blur' }
  ],
  code: [
    { required: true, message: t('auth.codeRequired'), trigger: 'blur' },
    { len: 6, message: t('auth.codeLength'), trigger: 'blur' }
  ],
  password: [
    { required: true, message: t('auth.passwordRequired'), trigger: 'blur' },
    { min: 6, max: 50, message: t('auth.passwordLength'), trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: t('auth.confirmPasswordRequired'), trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}))

// 发送验证码
const handleSendCode = async () => {
  if (!formData.email) {
    ElMessage.warning(t('auth.emailFirst'))
    return
  }

  try {
    codeLoading.value = true
    await sendEmailCode(formData.email, 'forgot_password')
    ElMessage.success(t('auth.emailCodeSent'))
    
    // 开始倒计时
    countdown.value = 60
    const timer = setInterval(() => {
      countdown.value--
      if (countdown.value <= 0) {
        clearInterval(timer)
      }
    }, 1000)
  } catch (error: any) {
    console.error('Send verification code failed:', error)
    const message = error?.response?.data?.msg || error?.message || t('auth.emailCodeFailed')
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

    ElMessage.success(t('auth.passwordResetSuccess'))
    
    // 跳转到登录页
    setTimeout(() => {
      router.push('/login')
    }, 2000)
  } catch (error: any) {
    console.error('Forgot password failed:', error)
    if (error?.errors) {
      // 表单验证错误，不显示错误消息
      return
    }
    const message = error?.response?.data?.msg || error?.message || t('auth.operationFailed')
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
    <div class="auth-language">
      <LanguageSwitcher />
    </div>

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
            <img alt="Kageos" class="logo" src="@/architecture/presentation/assets/logo.svg" />
          </div>
        </div>
        <h1 class="brand-title">
          <span class="title-gradient">{{ t('auth.forgotHeroTitle') }}</span>
        </h1>
        <p class="brand-subtitle">
          {{ t('auth.forgotHeroSubtitle') }}
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
          <h2 class="form-title">{{ t('auth.forgotTitle') }}</h2>
          <p class="form-subtitle">{{ t('auth.forgotSubtitle') }}</p>
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
              :placeholder="t('auth.emailPlaceholder')"
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
                :placeholder="t('auth.codePlaceholder')"
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
                <span v-if="countdown > 0">{{ t('auth.secondsRetry', { seconds: countdown }) }}</span>
                <span v-else>{{ t('auth.sendCode') }}</span>
              </el-button>
            </div>
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              v-model="formData.password"
              type="password"
              :placeholder="t('auth.newPasswordPlaceholder')"
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
              :placeholder="t('auth.confirmPasswordPlaceholder')"
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
              <span v-if="!loading">{{ t('auth.submitButton') }}</span>
              <span v-else>{{ t('auth.submitLoading') }}</span>
            </el-button>
          </el-form-item>

          <div class="form-footer">
            <el-button type="text" @click="goToLogin" class="back-link">
              {{ t('auth.backToLogin') }}
            </el-button>
          </div>
        </el-form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.forgot-password-container {
  --auth-accent: #1677ff;
  --auth-accent-strong: #0958d9;
  --auth-accent-soft: rgba(22, 119, 255, 0.14);
  --auth-brand-start: #1e2f56;
  --auth-brand-end: #35538f;
  --auth-card-bg: rgba(255, 255, 255, 0.8);
  --auth-card-border: rgba(148, 163, 184, 0.26);
  --auth-input-bg: rgba(255, 255, 255, 0.96);
  --auth-input-border: #cbd5e1;
  --auth-text: #0f172a;
  --auth-text-soft: #64748b;
  min-height: 100vh;
  display: flex;
  background:
    linear-gradient(112deg, var(--auth-brand-start) 0%, var(--auth-brand-end) 47%, #e9eff6 47%, #f4f7fb 100%);
  position: relative;
  overflow: hidden;
  isolation: isolate;
}

.auth-language {
  position: absolute;
  top: 20px;
  right: 24px;
  z-index: 3;
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
  background: linear-gradient(135deg, #ffffff 0%, #e0e7ff 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.brand-subtitle {
  font-size: 18px;
  line-height: 1.8;
  margin: 0;
  color: rgba(226, 232, 240, 0.88);
  font-weight: 400;
}

.forgot-password-form-section {
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

.forgot-password-card {
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

.form-title {
  font-size: 32px;
  font-weight: 700;
  color: var(--auth-text);
  margin: 0 0 8px 0;
  letter-spacing: -0.5px;
}

.form-subtitle {
  font-size: 15px;
  color: var(--auth-text-soft);
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
  background: linear-gradient(135deg, var(--auth-accent) 0%, var(--auth-accent-strong) 100%);
  border: none;
  color: white;
  box-shadow: 0 12px 28px rgba(22, 119, 255, 0.18);
}

.code-button:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 16px 34px rgba(22, 119, 255, 0.26);
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
  background: linear-gradient(135deg, var(--auth-accent) 0%, var(--auth-accent-strong) 100%);
  border: none;
  transition: all 0.3s ease;
  box-shadow: 0 14px 30px rgba(22, 119, 255, 0.22);
}

.submit-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 18px 36px rgba(22, 119, 255, 0.28);
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
  color: var(--auth-accent);
  padding: 0;
  transition: all 0.3s ease;
}

.back-link:hover {
  color: var(--auth-accent-strong);
  transform: translateX(-2px);
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
    border-left: none;
    border-top: 1px solid rgba(148, 163, 184, 0.18);
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

  .forgot-password-card {
    padding: 32px 24px 26px;
    border-radius: 24px;
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

<style>
.forgot-password-container .auth-language .language-switcher {
  border-color: rgba(22, 119, 255, 0.24) !important;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(248, 250, 252, 0.8)) !important;
  color: #1e3a5f !important;
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.1) !important;
  backdrop-filter: blur(16px);
}

.forgot-password-container .auth-language .language-switcher:hover,
.forgot-password-container .auth-language .language-switcher:focus-visible {
  border-color: rgba(22, 119, 255, 0.46) !important;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(248, 250, 252, 0.9)) !important;
  box-shadow: 0 18px 38px rgba(15, 23, 42, 0.14) !important;
  transform: translateY(-1px);
}

.forgot-password-container .auth-language .language-switcher__code,
.forgot-password-container .auth-language .language-switcher__arrow {
  color: #64748b !important;
}
</style>
