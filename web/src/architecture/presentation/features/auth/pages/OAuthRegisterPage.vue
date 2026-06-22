<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import { Check, Loading, User } from '@element-plus/icons-vue'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import {
  confirmOAuthRegistration,
  getOAuthRegistrationIntent,
  type OAuthRegistrationIntent
} from '@/architecture/presentation/context/api/auth'
import UserAvatar from '@/architecture/presentation/shared/components/UserAvatar.vue'

const router = useRouter()
const authStore = useAuthStore()
const formRef = ref<FormInstance>()
const loading = ref(false)
const submitting = ref(false)
const intent = ref<OAuthRegistrationIntent | null>(null)

const form = reactive({
  username: '',
  nickname: '',
})

const codePattern = /^[a-z][a-z0-9_]{2,31}$/

const rules = computed<FormRules>(() => ({
  username: [
    { required: true, message: '请输入用户 code', trigger: 'blur' },
    { pattern: codePattern, message: '用户 code 只能使用 3-32 位小写字母、数字、下划线，且必须以小写字母开头', trigger: 'blur' },
  ],
  nickname: [
    { required: true, message: '请输入显示名称', trigger: 'blur' },
    { max: 100, message: '显示名称不能超过 100 个字符', trigger: 'blur' },
  ],
}))

const ticket = computed(() => {
  const params = new URLSearchParams(window.location.hash.replace(/^#/, ''))
  return params.get('ticket') || ''
})

const expiresAtText = computed(() => {
  if (!intent.value?.expires_at) {
    return ''
  }
  return new Date(intent.value.expires_at).toLocaleString()
})

function normalizeCodeInput(value: string | number) {
  const normalized = String(value ?? '')
    .toLowerCase()
    .replace(/[-.\s]+/g, '_')
    .replace(/[^a-z0-9_]/g, '')
    .replace(/_+/g, '_')
    .replace(/^[^a-z]+/g, '')
  form.username = normalized
}

function selectSuggestion(code: string) {
  form.username = code
  formRef.value?.validateField('username')
}

async function loadIntent() {
  if (!ticket.value) {
    ElMessage.error('授权注册确认不存在，请重新授权登录')
    await router.replace('/login')
    return
  }
  loading.value = true
  try {
    const data = await getOAuthRegistrationIntent(ticket.value)
    intent.value = data
    form.username = data.suggested_code || data.code_suggestions?.[0] || ''
    form.nickname = data.nickname || form.username
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || '授权注册确认已失效')
    await router.replace('/login')
  } finally {
    loading.value = false
  }
}

async function submitRegistration() {
  if (!intent.value) {
    return
  }
  await formRef.value?.validate()
  submitting.value = true
  try {
    const result = await confirmOAuthRegistration(ticket.value, {
      username: form.username.trim(),
      nickname: form.nickname.trim(),
    })
    await authStore.completeOAuthLogin(result.token, result.refresh_token, result.redirect_after)
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || '授权注册失败')
  } finally {
    submitting.value = false
  }
}

function backToLogin() {
  router.replace('/login')
}

onMounted(loadIntent)
</script>

<template>
  <div class="oauth-register-page">
    <div class="register-shell">
      <div v-loading="loading" class="register-card">
        <div class="register-header">
          <img alt="Kageos" class="brand-logo" src="@/architecture/presentation/assets/logo.svg" />
          <div class="register-title-block">
            <h1>完成授权注册</h1>
            <p>确认账号标识后进入工作空间</p>
          </div>
        </div>

        <template v-if="intent">
          <div class="register-content">
            <div class="oauth-profile">
              <UserAvatar :src="intent.avatar" :size="48" :alt="intent.email || intent.provider_name" />
              <div class="profile-main">
                <div class="provider-name">{{ intent.provider_name }} 授权账号</div>
                <div class="profile-email">{{ intent.email || '未提供邮箱' }}</div>
              </div>
            </div>

            <el-alert
              title="用户 code 会用于目录、URL 和工作空间标识，注册后不建议修改。"
              type="info"
              show-icon
              :closable="false"
              class="code-alert"
            />

            <el-form
              ref="formRef"
              :model="form"
              :rules="rules"
              label-position="top"
              size="large"
              class="confirm-form"
            >
              <el-form-item label="用户 code" prop="username">
                <el-input
                  :model-value="form.username"
                  :prefix-icon="User"
                  maxlength="32"
                  show-word-limit
                  placeholder="例如 beiluo"
                  @update:model-value="normalizeCodeInput"
                />
              </el-form-item>

              <div v-if="intent.code_suggestions?.length" class="suggestions">
                <span class="suggestion-label">可选建议</span>
                <div class="suggestion-list">
                  <el-button
                    v-for="code in intent.code_suggestions"
                    :key="code"
                    size="small"
                    :type="form.username === code ? 'primary' : 'default'"
                    @click="selectSuggestion(code)"
                  >
                    {{ code }}
                  </el-button>
                </div>
              </div>

              <el-form-item label="显示名称" prop="nickname">
                <el-input
                  v-model="form.nickname"
                  maxlength="100"
                  show-word-limit
                  placeholder="用于界面展示，可后续修改"
                />
              </el-form-item>

              <div v-if="expiresAtText" class="expiry-text">
                本次确认有效期至 {{ expiresAtText }}
              </div>

              <div class="form-actions">
                <el-button @click="backToLogin">返回登录</el-button>
                <el-button type="primary" :loading="submitting" :icon="submitting ? Loading : Check" @click="submitRegistration">
                  确认并进入
                </el-button>
              </div>
            </el-form>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.oauth-register-page {
  --auth-accent: #1677ff;
  --auth-accent-soft: rgba(22, 119, 255, 0.12);
  --auth-card-bg: rgba(248, 251, 255, 0.86);
  --auth-card-bg-strong: rgba(244, 248, 253, 0.92);
  --auth-card-border: rgba(127, 148, 179, 0.28);
  --auth-input-bg: rgba(245, 249, 254, 0.94);
  --auth-input-bg-focus: rgba(250, 252, 255, 0.98);
  --auth-input-border: rgba(93, 130, 188, 0.3);
  --auth-input-text: #1e3a5f;
  --auth-input-placeholder: #7d91ad;
  --auth-text: #172033;
  --auth-text-muted: #475569;
  --auth-text-soft: #64748b;
  --text-primary: var(--auth-text);
  --text-regular: var(--auth-text-muted);
  --text-secondary: var(--auth-text-soft);
  --el-bg-color: var(--auth-card-bg);
  --el-bg-color-overlay: var(--auth-card-bg-strong);
  --el-fill-color-blank: var(--auth-input-bg);
  --el-fill-color-light: rgba(235, 242, 250, 0.72);
  --el-fill-color-lighter: rgba(241, 246, 252, 0.82);
  --el-border-color: var(--auth-card-border);
  --el-border-color-light: rgba(148, 163, 184, 0.22);
  --el-text-color-primary: var(--auth-text);
  --el-text-color-regular: var(--auth-text-muted);
  --el-text-color-secondary: var(--auth-text-soft);
  --el-input-bg-color: var(--auth-input-bg);
  --el-input-text-color: var(--auth-input-text);
  --el-input-border-color: var(--auth-input-border);
  --el-input-hover-border-color: rgba(22, 119, 255, 0.42);
  --el-input-focus-border-color: var(--auth-accent);
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  color: var(--auth-text);
  background:
    radial-gradient(circle at 18% 16%, rgba(22, 119, 255, 0.11), transparent 28%),
    linear-gradient(135deg, #edf3fa 0%, #f7f9fc 48%, #eef5ff 100%);
}

.register-shell {
  width: min(100%, 500px);
}

.brand-logo {
  width: 42px;
  height: 42px;
  flex: 0 0 auto;
}

.register-title-block h1 {
  margin: 0;
  font-size: 20px;
  line-height: 1.2;
  color: var(--auth-text);
}

.register-title-block p {
  margin: 4px 0 0;
  color: var(--auth-text-soft);
  font-size: 14px;
}

.register-card {
  min-height: 440px;
  overflow: hidden;
  border: 1px solid var(--auth-card-border);
  border-radius: 8px;
  background: var(--auth-card-bg);
  box-shadow: 0 20px 46px rgba(37, 62, 97, 0.14);
  backdrop-filter: blur(18px);
}

.register-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 24px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.22);
  background: var(--auth-card-bg-strong);
}

.register-content {
  padding: 24px;
}

.oauth-profile {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  margin-bottom: 16px;
  padding: 14px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  border-radius: 8px;
  background: linear-gradient(180deg, rgba(246, 250, 255, 0.86), rgba(237, 244, 252, 0.82));
}

.profile-main {
  min-width: 0;
  flex: 1;
}

.provider-name {
  font-weight: 700;
  color: var(--auth-text);
}

.profile-email {
  margin-top: 4px;
  color: var(--auth-text-soft);
  font-size: 13px;
  word-break: break-all;
}

.code-alert {
  margin-bottom: 18px;
  border: 1px solid rgba(22, 119, 255, 0.14);
  background: rgba(22, 119, 255, 0.07);
}

.code-alert :deep(.el-alert__title) {
  color: #28507f;
}

.code-alert :deep(.el-alert__icon) {
  color: #2c7be5;
}

.confirm-form {
  margin-top: 0;
}

.confirm-form :deep(.el-form-item__label) {
  padding-bottom: 6px;
  color: var(--auth-text-muted) !important;
  font-weight: 600;
  line-height: 1.4;
}

.oauth-register-page :deep(.confirm-form.el-form .el-form-item__label),
.oauth-register-page :deep(.confirm-form.el-form .el-form-item__content) {
  color: var(--auth-text-muted) !important;
}

.confirm-form :deep(.el-input__wrapper) {
  min-height: 44px;
  border-radius: 8px;
  border: 1px solid var(--auth-input-border);
  background: var(--auth-input-bg) !important;
  background-color: var(--auth-input-bg) !important;
  box-shadow: none;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
}

.confirm-form :deep(.el-input__wrapper:hover) {
  border-color: rgba(22, 119, 255, 0.42);
  box-shadow: 0 10px 24px rgba(22, 119, 255, 0.08);
}

.confirm-form :deep(.el-input__wrapper.is-focus) {
  border-color: var(--auth-accent);
  background: var(--auth-input-bg-focus) !important;
  box-shadow: 0 0 0 4px var(--auth-accent-soft);
}

.confirm-form :deep(.el-input__inner) {
  background: transparent !important;
  background-color: transparent !important;
  color: var(--auth-input-text) !important;
  -webkit-text-fill-color: var(--auth-input-text) !important;
  caret-color: var(--auth-accent);
  font-weight: 600;
}

.confirm-form :deep(.el-input__inner::placeholder) {
  color: var(--auth-input-placeholder) !important;
  -webkit-text-fill-color: var(--auth-input-placeholder) !important;
  font-weight: 400;
}

.confirm-form :deep(.el-input__prefix),
.confirm-form :deep(.el-input__suffix),
.confirm-form :deep(.el-input__count),
.confirm-form :deep(.el-input__count-inner) {
  background: transparent !important;
  color: var(--auth-text-soft) !important;
}

.confirm-form :deep(.el-input__count-inner) {
  background: var(--auth-input-bg) !important;
}

.confirm-form :deep(input:-webkit-autofill),
.confirm-form :deep(input:-webkit-autofill:hover),
.confirm-form :deep(input:-webkit-autofill:focus) {
  box-shadow: 0 0 0 1000px var(--auth-input-bg-focus) inset !important;
  -webkit-text-fill-color: var(--auth-input-text) !important;
  caret-color: var(--auth-accent);
}

.confirm-form :deep(.el-form-item) {
  margin-bottom: 18px;
}

.suggestions {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin: -6px 0 18px;
}

.suggestion-label,
.expiry-text {
  color: #6b7788;
  font-size: 13px;
}

.suggestion-label {
  min-width: 58px;
  padding-top: 5px;
  line-height: 1.4;
}

.suggestion-list {
  display: flex;
  flex: 1;
  flex-wrap: wrap;
  gap: 8px;
}

.suggestion-list .el-button {
  margin-left: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
}

.suggestion-list :deep(.el-button:not(.el-button--primary)) {
  border-color: rgba(93, 130, 188, 0.24);
  background: rgba(245, 249, 254, 0.76);
  color: var(--auth-text-muted);
}

.expiry-text {
  margin: 2px 0 16px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 4px;
}

.form-actions :deep(.el-button:not(.el-button--primary)) {
  border-color: rgba(93, 130, 188, 0.26);
  background: rgba(245, 249, 254, 0.82);
  color: var(--auth-text-muted);
}

.form-actions :deep(.el-button:not(.el-button--primary):hover),
.suggestion-list :deep(.el-button:not(.el-button--primary):hover) {
  border-color: rgba(22, 119, 255, 0.34);
  background: rgba(235, 244, 255, 0.9);
  color: #1d5fbf;
}

@media (max-width: 560px) {
  .oauth-register-page {
    align-items: stretch;
    padding: 16px;
  }

  .register-card {
    min-height: auto;
  }

  .register-header,
  .register-content {
    padding: 18px;
  }

  .suggestions {
    flex-direction: column;
    gap: 8px;
  }

  .suggestion-label {
    min-width: 0;
    padding-top: 0;
  }

  .form-actions {
    flex-direction: column-reverse;
  }

  .form-actions .el-button {
    width: 100%;
    margin-left: 0;
  }
}
</style>
