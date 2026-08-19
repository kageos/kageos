<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, reactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { ArrowRight, InfoFilled, User, Lock, Check, Loading } from '@element-plus/icons-vue'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import type { LoginRequest } from '@/architecture/domain/types'
import {
  completeWechatLoginAttempt,
  createWechatLoginAttempt,
  getLoginAnnouncement,
  listLoginMethods,
  type LoginAnnouncement,
  type WechatLoginAttempt,
  type LoginMethodInfo
} from '@/architecture/presentation/context/api/auth'
import LanguageSwitcher from '@/architecture/presentation/components/LanguageSwitcher.vue'
import { BRAND_LOGO_192_URL } from '@/architecture/domain/utils/builtinUserAvatar'
import LegalConsent from '@/architecture/presentation/features/legal/components/LegalConsent.vue'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { t, locale } = useI18n()

// 表单数据
const loginForm = reactive<LoginRequest>({
  username: '',
  password: ''
})

// 表单引用
const loginFormRef = ref()

// 加载状态
const loading = ref(false)
const methodsLoading = ref(false)
const loginMethods = ref<LoginMethodInfo[]>([])
const loginAnnouncement = ref<LoginAnnouncement | null>(null)
const announcementDialogVisible = ref(false)
const wechatDialogVisible = ref(false)
const wechatAttempt = ref<WechatLoginAttempt | null>(null)
const wechatLoading = ref(false)
const wechatStatus = ref<'idle' | 'waiting' | 'error'>('idle')
let wechatPollTimer: number | undefined
const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()

const announcementSummary = computed(() => {
  const markdown = loginAnnouncement.value?.markdown || ''
  return markdown
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/[#>*_`~\[\]()|!-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
})

// 表单验证规则
const rules = computed(() => ({
  username: [
    { required: true, message: t('auth.usernameRequired'), trigger: 'blur' },
    { min: 3, max: 32, message: t('auth.usernameLength'), trigger: 'blur' }
  ],
  password: [
    { required: true, message: t('auth.passwordRequired'), trigger: 'blur' },
    { min: 6, max: 50, message: t('auth.passwordLength'), trigger: 'blur' }
  ]
}))

// 处理登录
const handleLogin = async () => {
  try {
    await loginFormRef.value.validate()
    loading.value = true

    await authStore.login(loginForm)

    // 登录成功后跳转到首页
    await router.push('/')
  } catch (error: any) {
    console.error('Login failed:', error)
    // 🔥 统一使用 msg 字段
    const message = error?.response?.data?.msg || error?.message || t('auth.loginFailed')
    ElMessage.error(message)
  } finally {
    loading.value = false
  }
}

const loadLoginMethods = async () => {
  methodsLoading.value = true
  try {
    const resp = await listLoginMethods()
    loginMethods.value = resp.methods || []
  } catch {
    loginMethods.value = []
  } finally {
    methodsLoading.value = false
  }
}

const loadLoginAnnouncement = async () => {
  try {
    const announcement = await getLoginAnnouncement()
    loginAnnouncement.value = announcement.enabled && announcement.markdown ? announcement : null
  } catch {
    loginAnnouncement.value = null
  }
}

const providerMark = (provider: string) => {
  const code = provider.toLowerCase()
  if (code.includes('google')) return 'G'
  if (code.includes('github')) return 'GH'
  if (code.includes('wechat')) return '微'
  return provider.slice(0, 2).toUpperCase()
}

const providerButtonClass = (provider: string) => {
  const code = provider.toLowerCase()
  if (code.includes('google')) return 'oauth-btn--google'
  if (code.includes('github')) return 'oauth-btn--github'
  if (code.includes('wechat')) return 'oauth-btn--wechat'
  return 'oauth-btn--default'
}

const redirectAfter = () => typeof route.query.redirect === 'string' ? route.query.redirect : '/workspace'

const clearWechatPolling = () => {
  if (wechatPollTimer !== undefined) {
    window.clearTimeout(wechatPollTimer)
    wechatPollTimer = undefined
  }
}

const scheduleWechatPoll = () => {
  clearWechatPolling()
  const delay = Math.max(1000, wechatAttempt.value?.poll_after_ms || 2000)
  wechatPollTimer = window.setTimeout(pollWechatLogin, delay)
}

const pollWechatLogin = async () => {
  const attemptToken = wechatAttempt.value?.attempt_token
  if (!attemptToken || !wechatDialogVisible.value) {
    return
  }
  try {
    const result = await completeWechatLoginAttempt(attemptToken)
    if (!wechatDialogVisible.value || wechatAttempt.value?.attempt_token !== attemptToken) {
      return
    }
    if (result.status === 'pending') {
      scheduleWechatPoll()
      return
    }
    clearWechatPolling()
    wechatDialogVisible.value = false
    if (result.registration_required && result.registration_ticket) {
      const params = new URLSearchParams()
      params.set('ticket', result.registration_ticket)
      params.set('redirect_after', result.redirect_after || redirectAfter())
      window.location.assign(`/auth/oauth/register#${params.toString()}`)
      return
    }
    if (!result.token || !result.refresh_token) {
      throw new Error(t('auth.wechatInvalidResult'))
    }
    await authStore.completeOAuthLogin(result.token, result.refresh_token, result.redirect_after)
  } catch (error: any) {
    clearWechatPolling()
    wechatStatus.value = 'error'
    ElMessage.error(error?.response?.data?.msg || error?.message || t('auth.wechatExpired'))
  }
}

const startWechatLogin = async () => {
  clearWechatPolling()
  wechatLoading.value = true
  wechatStatus.value = 'idle'
  wechatAttempt.value = null
  wechatDialogVisible.value = true
  try {
    wechatAttempt.value = await createWechatLoginAttempt(redirectAfter())
    wechatStatus.value = 'waiting'
    scheduleWechatPoll()
  } catch (error: any) {
    wechatStatus.value = 'error'
    ElMessage.error(error?.response?.data?.msg || error?.message || t('auth.wechatStartFailed'))
  } finally {
    wechatLoading.value = false
  }
}

const handleWechatDialogClosed = () => {
  clearWechatPolling()
  wechatAttempt.value = null
  wechatStatus.value = 'idle'
}

const handleLoginMethod = (method: LoginMethodInfo) => {
  if (method.action === 'qrcode') {
    startWechatLogin()
    return
  }
  if (method.action === 'redirect') {
    const authorizePath = method.authorize_path || ''
    if (!authorizePath) {
      ElMessage.error(t('auth.oauthStartFailed'))
      return
    }
    const params = new URLSearchParams()
    params.set('redirect_after', redirectAfter())
    window.location.assign(`${authorizePath}?${params.toString()}`)
    return
  }
  ElMessage.info(t('auth.oauthFlowPending'))
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

onMounted(() => {
  loadLoginMethods()
  loadLoginAnnouncement()
  preloadMarkdown()
  if (typeof route.query.oauth_error === 'string' && route.query.oauth_error) {
    ElMessage.error(route.query.oauth_error)
  }
})

onBeforeUnmount(clearWechatPolling)
</script>

<template>
  <div class="login-container" data-testid="login-page" @keypress="handleKeyPress">
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
    <div class="login-brand">
      <div class="brand-content">
        <div class="brand-logo-wrapper">
          <div class="logo-glow"></div>
          <div class="brand-logo">
            <img alt="kageos" class="logo" :src="BRAND_LOGO_192_URL" width="80" height="80" decoding="async" />
          </div>
        </div>
        <h1 class="brand-title">
          <span class="title-gradient">kageos</span>
        </h1>
        <p class="brand-subtitle">
          {{ t('auth.brandSubtitle') }}
        </p>
        <div class="brand-features">
          <div class="feature-item">
            <div class="feature-icon">
              <el-icon><Check /></el-icon>
            </div>
            <div class="feature-text">
              <span class="feature-title">{{ t('auth.featureCodeTitle') }}</span>
              <span class="feature-desc">{{ t('auth.featureCodeDesc') }}</span>
            </div>
          </div>
          <div class="feature-item">
            <div class="feature-icon">
              <el-icon><Check /></el-icon>
            </div>
            <div class="feature-text">
              <span class="feature-title">{{ t('auth.featureRenderTitle') }}</span>
              <span class="feature-desc">{{ t('auth.featureRenderDesc') }}</span>
            </div>
          </div>
          <div class="feature-item">
            <div class="feature-icon">
              <el-icon><Check /></el-icon>
            </div>
            <div class="feature-text">
              <span class="feature-title">{{ t('auth.featureTenantTitle') }}</span>
              <span class="feature-desc">{{ t('auth.featureTenantDesc') }}</span>
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
          <h2 class="login-title">{{ t('auth.loginTitle') }}</h2>
          <p class="login-subtitle">{{ t('auth.loginSubtitle') }}</p>
        </div>

        <button
          v-if="loginAnnouncement"
          type="button"
          class="login-announcement"
          @click="announcementDialogVisible = true"
        >
          <el-icon class="login-announcement-icon"><InfoFilled /></el-icon>
          <span class="login-announcement-copy">
            <strong>{{ t('auth.loginAnnouncementDefaultTitle') }}</strong>
            <span>{{ announcementSummary }}</span>
          </span>
          <span class="login-announcement-action">
            {{ t('auth.viewLoginAnnouncement') }}
            <el-icon><ArrowRight /></el-icon>
          </span>
        </button>

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
              :placeholder="t('auth.usernamePlaceholder')"
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
              :placeholder="t('auth.passwordPlaceholder')"
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
              <span v-if="!loading">{{ t('auth.loginButton') }}</span>
              <span v-else>{{ t('auth.loginLoading') }}</span>
            </el-button>
          </el-form-item>

          <div v-if="loginMethods.length || methodsLoading" class="external-login">
            <div class="login-divider">
              <span>{{ t('auth.otherLoginMethods') }}</span>
            </div>
            <div class="oauth-methods">
              <el-button
                v-for="method in loginMethods"
                :key="method.provider"
                class="oauth-btn"
                :class="providerButtonClass(method.provider)"
                @click="handleLoginMethod(method)"
              >
                <span class="oauth-mark">{{ providerMark(method.provider) }}</span>
                <span class="oauth-label">{{ method.label }}</span>
              </el-button>
            </div>
          </div>

          <LegalConsent :locale="locale" class="login-legal" />

          <div class="login-footer">
            <div class="footer-top">
              <el-button type="text" @click="goToForgotPassword" class="forgot-password-link">
                {{ t('auth.forgotPassword') }}
              </el-button>
            </div>
            <div class="footer-bottom">
            <span class="login-tip">{{ t('auth.noAccount') }}</span>
            <el-button type="text" @click="goToRegister" class="register-link">
              {{ t('auth.registerNow') }}
            </el-button>
            </div>
          </div>
        </el-form>
      </div>
    </div>

    <el-dialog
      v-model="announcementDialogVisible"
      :title="t('auth.loginAnnouncementDefaultTitle')"
      width="min(640px, calc(100vw - 32px))"
      align-center
      class="login-announcement-dialog"
    >
      <div
        class="login-announcement-markdown"
        v-html="renderMarkdown(loginAnnouncement?.markdown || '')"
      />
    </el-dialog>

    <el-dialog
      v-model="wechatDialogVisible"
      :title="t('auth.wechatDialogTitle')"
      width="360px"
      align-center
      :close-on-click-modal="false"
      @closed="handleWechatDialogClosed"
    >
      <div v-loading="wechatLoading" class="wechat-login-panel">
        <img
          v-if="wechatAttempt?.qr_code_url"
          :src="wechatAttempt.qr_code_url"
          :alt="t('auth.wechatDialogTitle')"
          class="wechat-qrcode"
        />
        <div v-else-if="wechatStatus === 'error'" class="wechat-error">
          {{ t('auth.wechatStartFailed') }}
        </div>
        <div class="wechat-hint">
          {{ wechatStatus === 'error' ? t('auth.wechatRetryHint') : t('auth.wechatScanHint') }}
        </div>
        <el-button v-if="wechatStatus === 'error'" type="primary" @click="startWechatLogin">
          {{ t('auth.wechatRetry') }}
        </el-button>
      </div>
    </el-dialog>
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
  --auth-input-bg: rgba(248, 251, 255, 0.96);
  --auth-input-bg-focus: rgba(255, 255, 255, 0.98);
  --auth-input-border: rgba(93, 130, 188, 0.3);
  --auth-input-text: #1e3a5f;
  --auth-input-placeholder: #7d91ad;
  --auth-input-icon: rgba(22, 119, 255, 0.58);
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
  filter: drop-shadow(0 16px 32px rgba(103, 232, 249, 0.22));
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

.login-announcement {
  margin: -24px 0 24px;
  width: 100%;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  color: var(--auth-text);
  text-align: left;
  background: rgba(22, 119, 255, 0.055);
  border: 1px solid rgba(22, 119, 255, 0.16);
  border-radius: 12px;
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease, transform 0.2s ease;
}

.login-announcement:hover {
  background: rgba(22, 119, 255, 0.085);
  border-color: rgba(22, 119, 255, 0.3);
  transform: translateY(-1px);
}

.login-announcement:focus-visible {
  outline: 3px solid rgba(22, 119, 255, 0.2);
  outline-offset: 2px;
}

.login-announcement-icon {
  flex: 0 0 auto;
  font-size: 20px;
  color: var(--auth-accent);
}

.login-announcement-copy {
  display: grid;
  min-width: 0;
  flex: 1;
  gap: 2px;
}

.login-announcement-copy strong {
  font-size: 14px;
  font-weight: 650;
}

.login-announcement-copy > span {
  overflow: hidden;
  color: var(--auth-text-soft);
  font-size: 13px;
  line-height: 1.4;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.login-announcement-action {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  gap: 3px;
  color: var(--auth-accent);
  font-size: 13px;
  font-weight: 600;
}

.login-announcement-markdown {
  color: var(--el-text-color-primary);
  font-size: 15px;
  line-height: 1.75;
  overflow-wrap: anywhere;
}

.login-announcement-markdown :deep(:first-child) {
  margin-top: 0;
}

.login-announcement-markdown :deep(:last-child) {
  margin-bottom: 0;
}

.login-announcement-markdown :deep(table) {
  width: 100%;
  border-collapse: collapse;
}

.login-announcement-markdown :deep(th),
.login-announcement-markdown :deep(td) {
  padding: 10px 12px;
  text-align: left;
  border: 1px solid var(--el-border-color-light);
}

.login-announcement-markdown :deep(th) {
  background: var(--el-fill-color-light);
}

.login-announcement-markdown :deep(code) {
  padding: 2px 6px;
  background: var(--el-fill-color-light);
  border-radius: 5px;
}

@media (max-width: 640px) {
  .login-announcement-action {
    font-size: 0;
  }
}

.form-input {
  --el-input-text-color: var(--auth-input-text);
  --el-input-placeholder-color: var(--auth-input-placeholder);
  --el-input-icon-color: var(--auth-input-icon);
  --el-input-bg-color: var(--auth-input-bg);
  --el-input-border-color: var(--auth-input-border);
  --el-input-hover-border-color: rgba(22, 119, 255, 0.42);
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
  box-shadow: 0 10px 24px rgba(22, 119, 255, 0.1);
  border-color: rgba(22, 119, 255, 0.42);
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

.external-login {
  margin: -4px 0 28px;
}

.login-divider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  color: var(--auth-text-soft);
  font-size: 13px;
}

.login-divider::before,
.login-divider::after {
  content: '';
  height: 1px;
  flex: 1;
  background: rgba(148, 163, 184, 0.34);
}

.oauth-methods {
  display: grid;
  gap: 10px;
}

.oauth-btn {
  width: 100%;
  min-height: 48px;
  justify-content: center;
  border-radius: 12px;
  border-color: rgba(148, 163, 184, 0.32);
  background: rgba(255, 255, 255, 0.72);
  color: var(--auth-text);
  font-weight: 700;
  transition: transform 0.2s ease, box-shadow 0.2s ease, border-color 0.2s ease;
}

.oauth-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 12px 26px rgba(15, 23, 42, 0.1);
}

.oauth-mark {
  display: inline-flex;
  width: 26px;
  height: 26px;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  font-size: 12px;
  font-weight: 800;
  line-height: 1;
}

.oauth-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.oauth-btn--google .oauth-mark {
  background: #fff;
  color: #1a73e8;
  border: 1px solid rgba(26, 115, 232, 0.22);
}

.oauth-btn--github .oauth-mark {
  background: #111827;
  color: #fff;
}

.oauth-btn--wechat .oauth-mark {
  background: #16a34a;
  color: #fff;
}

.oauth-btn--default .oauth-mark {
  background: var(--auth-accent-soft);
  color: var(--auth-accent);
}

.wechat-login-panel {
  min-height: 280px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
}

.wechat-qrcode {
  width: 220px;
  height: 220px;
  border-radius: 8px;
  object-fit: contain;
  background: #fff;
}

.wechat-hint,
.wechat-error {
  color: #64748b;
  font-size: 14px;
  line-height: 1.6;
  text-align: center;
}

.wechat-error {
  color: var(--el-color-danger);
}

.login-legal {
  margin: -8px 0 18px;
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
  color: var(--auth-input-icon);
}

:deep(.el-input__suffix) {
  color: var(--auth-input-icon);
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
    height: 100dvh;
    min-height: 100dvh;
    overflow-x: hidden;
    overflow-y: auto;
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
    padding: 52px 20px 28px;
  }

  .brand-logo-wrapper {
    margin-bottom: 16px;
  }

  .brand-title {
    font-size: 32px;
  }

  .brand-subtitle {
    font-size: 14px;
    margin-bottom: 0;
  }

  .brand-features {
    display: none;
  }

  .login-form-section {
    padding: 24px 16px 32px;
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

<style>
.login-container .form-input.el-input .el-input__inner,
.login-container .form-input.el-input input.el-input__inner {
  color: var(--auth-input-text) !important;
  -webkit-text-fill-color: var(--auth-input-text) !important;
  caret-color: var(--auth-accent) !important;
}

.login-container .form-input.el-input .el-input__inner::placeholder,
.login-container .form-input.el-input input.el-input__inner::placeholder {
  color: var(--auth-input-placeholder) !important;
  -webkit-text-fill-color: var(--auth-input-placeholder) !important;
}

.login-container .form-input.el-input .el-input__prefix,
.login-container .form-input.el-input .el-input__suffix,
.login-container .form-input.el-input .el-input__password,
.login-container .form-input.el-input .el-input__clear {
  color: var(--auth-input-icon) !important;
}

.login-container .form-input.el-input .el-input__wrapper {
  background-color: var(--auth-input-bg) !important;
  border-color: var(--auth-input-border) !important;
}

.login-container .form-input.el-input .el-input__wrapper.is-focus {
  background-color: var(--auth-input-bg-focus) !important;
  border-color: var(--auth-accent) !important;
}

.login-container .auth-language .language-switcher {
  border-color: var(--auth-input-border) !important;
  background: linear-gradient(180deg, var(--auth-input-bg-focus), var(--auth-input-bg)) !important;
  color: var(--auth-input-text) !important;
  box-shadow: var(--app-auth-card-shadow-soft, 0 14px 34px rgba(15, 23, 42, 0.1)) !important;
  backdrop-filter: blur(16px);
}

.login-container .auth-language .language-switcher:hover,
.login-container .auth-language .language-switcher:focus-visible {
  border-color: var(--auth-accent) !important;
  background: linear-gradient(180deg, var(--auth-input-bg-focus), var(--auth-input-bg)) !important;
  box-shadow: var(--app-auth-primary-shadow-hover, 0 18px 38px rgba(15, 23, 42, 0.14)) !important;
  transform: translateY(-1px);
}

.login-container .auth-language .language-switcher__code,
.login-container .auth-language .language-switcher__arrow {
  color: var(--auth-text-soft) !important;
}
</style>
