<script setup lang="ts">
import { computed, ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { User, Lock, Message, Loading } from '@element-plus/icons-vue'
import { register as registerApi, sendEmailCode } from '@/architecture/presentation/context/api/auth'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import type { RegisterRequest } from '@/architecture/domain/types'
import LanguageSwitcher from '@/architecture/presentation/components/LanguageSwitcher.vue'

const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()
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
const rules = computed(() => ({
  username: [
    { required: true, message: t('auth.usernameRequired'), trigger: 'blur' },
    { min: 3, max: 32, message: t('auth.usernameLength'), trigger: 'blur' },
    { pattern: /^[a-z][a-z0-9_]{2,31}$/, message: t('auth.usernamePattern'), trigger: 'blur' }
  ],
  email: [
    { required: true, message: t('auth.emailRequired'), trigger: 'blur' },
    { type: 'email', message: t('auth.emailInvalid'), trigger: 'blur' }
  ],
  password: [
    { required: true, message: t('auth.passwordRequired'), trigger: 'blur' },
    { min: 6, max: 50, message: t('auth.passwordLength'), trigger: 'blur' }
  ],
  code: [
    { required: true, message: t('auth.codeRequired'), trigger: 'blur' },
    { len: 6, message: t('auth.codeLength'), trigger: 'blur' }
  ]
}))

const normalizeUserCodeInput = (value: string | number) => {
  registerForm.username = String(value ?? '')
    .toLowerCase()
    .replace(/[-.\s]+/g, '_')
    .replace(/[^a-z0-9_]/g, '')
    .replace(/_+/g, '_')
    .replace(/^[^a-z]+/g, '')
}

// 处理注册
const handleRegister = async () => {
  let registrationSucceeded = false
  try {
    await registerFormRef.value.validate()
    loading.value = true

    const payload: RegisterRequest = {
      username: registerForm.username.trim().toLowerCase(),
      email: registerForm.email.trim(),
      password: registerForm.password,
      code: registerForm.code
    }

    await registerApi(payload)
    registrationSucceeded = true

    ElMessage.success(t('auth.registerSuccess'))
    await authStore.login({
      username: payload.username,
      password: payload.password
    }, {
      notify: false
    })
  } catch (error: any) {
    if (registrationSucceeded) {
      console.error('Automatic login after registration failed:', error)
      const message = error?.response?.data?.msg || error?.message || t('auth.loginFailed')
      ElMessage.error(message)
      await router.replace('/login')
      return
    }
    console.error('Register failed:', error)
    // 🔥 统一使用 msg 字段
    const message = error?.response?.data?.msg || error?.message || t('auth.registerFailed')
    ElMessage.error(message)
  } finally {
    loading.value = false
  }
}

// 发送验证码
const sendVerificationCode = async () => {
  if (!registerForm.email) {
    ElMessage.warning(t('auth.emailFirst'))
    return
  }

  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(registerForm.email)) {
    ElMessage.warning(t('auth.emailInvalid'))
    return
  }

  try {
    sendingCode.value = true
    const resp = await sendEmailCode(registerForm.email)
    if (resp?.debug_code) {
      registerForm.code = resp.debug_code
      ElMessage.success(`Dev verification code: ${resp.debug_code}`)
    } else {
      ElMessage.success(t('auth.emailCodeSent'))
    }

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
    // 🔥 统一使用 msg 字段
    const message = error?.response?.data?.msg || error?.message || t('auth.emailCodeFailed')
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
  <div class="register-container" data-testid="register-page" @keypress="handleKeyPress">
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
    <div class="register-brand">
      <div class="brand-content">
        <div class="brand-logo-wrapper">
          <div class="logo-glow"></div>
          <div class="brand-logo">
            <img alt="Kageos" class="logo" src="@/architecture/presentation/assets/logo.svg" />
          </div>
        </div>
        <h1 class="brand-title">
          <span class="title-gradient">{{ t('auth.registerHeroTitle') }}</span>
        </h1>
        <p class="brand-subtitle">
          {{ t('auth.registerHeroSubtitle') }}
        </p>
        <div class="brand-steps">
          <div class="step-item">
            <div class="step-icon">
              <div class="step-number">1</div>
            </div>
            <div class="step-content">
              <span class="step-title">{{ t('auth.registerStepAccountTitle') }}</span>
              <span class="step-desc">{{ t('auth.registerStepAccountDesc') }}</span>
            </div>
          </div>
          <div class="step-item">
            <div class="step-icon">
              <div class="step-number">2</div>
            </div>
            <div class="step-content">
              <span class="step-title">{{ t('auth.registerStepEmailTitle') }}</span>
              <span class="step-desc">{{ t('auth.registerStepEmailDesc') }}</span>
            </div>
          </div>
          <div class="step-item">
            <div class="step-icon">
              <div class="step-number">3</div>
            </div>
            <div class="step-content">
              <span class="step-title">{{ t('auth.registerStepStartTitle') }}</span>
              <span class="step-desc">{{ t('auth.registerStepStartDesc') }}</span>
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
          <h2 class="register-title">{{ t('auth.registerTitle') }}</h2>
          <p class="register-subtitle">{{ t('auth.registerSubtitle') }}</p>
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
              data-testid="register-username"
              :model-value="registerForm.username"
              :placeholder="t('auth.usernamePlaceholder')"
              :prefix-icon="User"
              clearable
              size="large"
              class="form-input"
              @update:model-value="normalizeUserCodeInput"
            />
          </el-form-item>

          <el-form-item prop="email">
            <el-input
              data-testid="register-email"
              v-model="registerForm.email"
              :placeholder="t('auth.emailPlaceholder')"
              :prefix-icon="Message"
              clearable
              size="large"
              class="form-input"
            />
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              data-testid="register-password"
              v-model="registerForm.password"
              type="password"
              :placeholder="t('auth.passwordPlaceholder')"
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
                data-testid="register-code"
                v-model="registerForm.code"
                :placeholder="t('auth.codePlaceholder')"
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
                {{ countdown > 0 ? t('auth.secondsRetry', { seconds: countdown }) : t('auth.sendCode') }}
              </el-button>
            </div>
          </el-form-item>

          <el-form-item class="register-btn-item">
            <el-button
              data-testid="register-submit"
              type="primary"
              size="large"
              :loading="loading"
              class="register-btn"
              @click="handleRegister"
            >
              <template #loading>
                <el-icon class="is-loading"><Loading /></el-icon>
              </template>
              <span v-if="!loading">{{ t('auth.registerButton') }}</span>
              <span v-else>{{ t('auth.registerLoading') }}</span>
            </el-button>
          </el-form-item>

          <div class="register-footer">
            <span class="register-tip">{{ t('auth.hasAccount') }}</span>
            <el-button type="text" @click="goToLogin" class="login-link">
              {{ t('auth.loginNow') }}
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
  --auth-input-bg: rgba(248, 252, 249, 0.96);
  --auth-input-bg-focus: rgba(255, 255, 255, 0.98);
  --auth-input-border: rgba(80, 143, 105, 0.3);
  --auth-input-text: #174630;
  --auth-input-placeholder: #7f9b8c;
  --auth-input-icon: rgba(22, 163, 74, 0.58);
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
  filter: drop-shadow(0 16px 32px rgba(103, 232, 249, 0.22));
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

.form-input {
  --el-input-text-color: var(--auth-input-text);
  --el-input-placeholder-color: var(--auth-input-placeholder);
  --el-input-icon-color: var(--auth-input-icon);
  --el-input-bg-color: var(--auth-input-bg);
  --el-input-border-color: var(--auth-input-border);
  --el-input-hover-border-color: rgba(22, 163, 74, 0.42);
  --el-input-focus-border-color: var(--auth-accent);
  --el-text-color-regular: var(--auth-input-text);
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
  color: var(--auth-input-text);
}

:deep(.el-input__wrapper:hover) {
  box-shadow: 0 10px 24px rgba(22, 163, 74, 0.1);
  border-color: rgba(22, 163, 74, 0.42);
}

:deep(.el-input__wrapper.is-focus) {
  background: var(--auth-input-bg-focus);
  box-shadow: 0 0 0 4px var(--auth-accent-soft);
  border-color: var(--auth-accent);
}

:deep(.el-input__inner) {
  height: 52px;
  font-size: 15px;
  color: var(--auth-input-text) !important;
  -webkit-text-fill-color: var(--auth-input-text);
  caret-color: var(--auth-accent) !important;
  font-weight: 600;
}

:deep(.el-input__inner::placeholder) {
  color: var(--auth-input-placeholder) !important;
  -webkit-text-fill-color: var(--auth-input-placeholder);
  font-weight: 400;
}

:deep(.el-input__inner:-webkit-autofill) {
  -webkit-text-fill-color: var(--auth-input-text);
  box-shadow: 0 0 0 1000px var(--auth-input-bg-focus) inset;
  caret-color: var(--auth-accent);
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
  color: var(--auth-input-icon);
}

:deep(.el-input__suffix) {
  color: var(--auth-input-icon);
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

<style>
.register-container .form-input.el-input .el-input__inner,
.register-container .form-input.el-input input.el-input__inner {
  color: #174630 !important;
  -webkit-text-fill-color: #174630 !important;
  caret-color: #16a34a !important;
}

.register-container .form-input.el-input .el-input__inner::placeholder,
.register-container .form-input.el-input input.el-input__inner::placeholder {
  color: #7f9b8c !important;
  -webkit-text-fill-color: #7f9b8c !important;
}

.register-container .form-input.el-input .el-input__prefix,
.register-container .form-input.el-input .el-input__suffix,
.register-container .form-input.el-input .el-input__password,
.register-container .form-input.el-input .el-input__clear {
  color: rgba(22, 163, 74, 0.58) !important;
}

.register-container .form-input.el-input .el-input__wrapper {
  background-color: rgba(248, 252, 249, 0.96) !important;
  border-color: rgba(80, 143, 105, 0.3) !important;
}

.register-container .form-input.el-input .el-input__wrapper.is-focus {
  background-color: rgba(255, 255, 255, 0.98) !important;
  border-color: #16a34a !important;
}

.register-container .auth-language .language-switcher {
  border-color: rgba(22, 163, 74, 0.24) !important;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(248, 250, 252, 0.8)) !important;
  color: #174630 !important;
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.1) !important;
  backdrop-filter: blur(16px);
}

.register-container .auth-language .language-switcher:hover,
.register-container .auth-language .language-switcher:focus-visible {
  border-color: rgba(22, 163, 74, 0.46) !important;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.96), rgba(248, 250, 252, 0.9)) !important;
  box-shadow: 0 18px 38px rgba(15, 23, 42, 0.14) !important;
  transform: translateY(-1px);
}

.register-container .auth-language .language-switcher__code,
.register-container .auth-language .language-switcher__arrow {
  color: #7f9b8c !important;
}
</style>
