<template>
  <div class="system-settings-page">
    <el-card shadow="hover" class="settings-card">
      <template #header>
        <div class="card-header">
          <div>
            <h2>{{ t('route.systemSettings') }}</h2>
            <p>Email delivery, registration policy, login methods, and HTTPS certificates are managed by the system owner.</p>
          </div>
          <div class="header-actions">
            <el-button :icon="Connection" @click="router.push('/connectors/providers')">
              {{ t('workspace.connectorConfig') }}
            </el-button>
            <el-button :icon="Refresh" @click="refreshActiveTab">{{ t('common.refresh') }}</el-button>
            <el-button v-if="activeTab === 'email'" type="primary" :icon="Check" :loading="saving" @click="saveSettings">
              {{ t('connectorProvider.save') }}
            </el-button>
          </div>
        </div>
      </template>

      <div v-loading="loading" class="settings-body">
        <el-tabs v-model="activeTab" class="settings-tabs" @tab-change="handleTabChange">
          <el-tab-pane label="邮件与注册" name="email">
            <el-alert
              v-if="form.registration_mode === 'admin_only'"
              title="Self-service registration is disabled. Users must be created by system."
              type="info"
              show-icon
              :closable="false"
            />

            <el-form ref="formRef" :model="form" label-width="180px" class="settings-form">
              <el-divider content-position="left">Registration</el-divider>
              <el-form-item label="Registration mode">
                <el-radio-group v-model="form.registration_mode">
                  <el-radio-button value="admin_only">Admin only</el-radio-button>
                  <el-radio-button value="email_code">Email verification</el-radio-button>
                  <el-radio-button value="debug_code">Debug code</el-radio-button>
                </el-radio-group>
              </el-form-item>

              <el-divider content-position="left">Email service</el-divider>
              <el-form-item label="Email mode">
                <el-radio-group v-model="form.email.mode">
                  <el-radio-button value="smtp">SMTP</el-radio-button>
                  <el-radio-button value="log">Log</el-radio-button>
                </el-radio-group>
              </el-form-item>

              <el-form-item label="SMTP host">
                <el-input v-model="form.email.host" placeholder="smtp.example.com" />
              </el-form-item>
              <el-form-item label="SMTP port">
                <el-input-number v-model="form.email.port" :min="1" :max="65535" />
              </el-form-item>
              <el-form-item label="Username">
                <el-input v-model="form.email.username" placeholder="SMTP account username" />
              </el-form-item>
              <el-form-item label="Password">
                <el-input
                  v-model="form.email.password"
                  type="password"
                  show-password
                  :placeholder="form.email.password_set ? 'Already configured; leave blank to keep current password' : 'SMTP password'"
                />
              </el-form-item>
              <el-form-item label="From">
                <el-input v-model="form.email.from" placeholder="noreply@example.com" />
              </el-form-item>
              <el-form-item label="From name">
                <el-input v-model="form.email.from_name" placeholder="Kageos" />
              </el-form-item>

              <el-divider content-position="left">Test email</el-divider>
              <el-form-item label="Recipient">
                <div class="test-row">
                  <el-input v-model="testEmail" placeholder="admin@example.com" />
                  <el-button :icon="Message" :loading="testing" @click="sendTestEmail">Send test</el-button>
                </div>
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <el-tab-pane label="登录配置" name="login">
            <div v-loading="providersLoading" class="login-provider-section">
              <div class="provider-summary">
                <div class="summary-item">
                  <span class="summary-value">{{ authProviders.length }}</span>
                  <span class="summary-label">预置方式</span>
                </div>
                <div class="summary-item">
                  <span class="summary-value">{{ configuredProviderCount }}</span>
                  <span class="summary-label">已配置</span>
                </div>
                <div class="summary-item">
                  <span class="summary-value">{{ enabledProviderCount }}</span>
                  <span class="summary-label">已启用</span>
                </div>
              </div>

              <el-empty v-if="!authProviders.length && !providersLoading" description="暂无预置登录方式" />

              <div
                v-for="provider in authProviders"
                :key="provider.code"
                class="provider-panel"
              >
                <div class="provider-panel-header">
                  <div class="provider-heading">
                    <div class="provider-title-row">
                      <span class="provider-name">{{ provider.name }}</span>
                      <el-tag :type="providerStatusType(provider)" size="small">
                        {{ providerStatusLabel(provider) }}
                      </el-tag>
                      <el-tag size="small" effect="plain">
                        {{ providerActionLabel(provider.action) }}
                      </el-tag>
                    </div>
                    <p>{{ provider.description }}</p>
                    <div class="provider-meta">
                      <span v-if="provider.callback_path" class="callback-path">
                        回调地址：<code>{{ callbackURL(provider) }}</code>
                      </span>
                      <el-button
                        v-if="provider.callback_path"
                        link
                        type="primary"
                        :icon="CopyDocument"
                        @click="copyCallbackURL(provider)"
                      >
                        复制
                      </el-button>
                      <el-link
                        v-if="provider.docs_url"
                        type="primary"
                        :href="provider.docs_url"
                        target="_blank"
                        :underline="false"
                      >
                        平台文档
                      </el-link>
                    </div>
                  </div>
                  <div class="provider-enable">
                    <span>启用</span>
                    <el-switch
                      :model-value="provider.enabled"
                      :disabled="!provider.configured"
                      :loading="providerSwitching[provider.code]"
                      @change="handleProviderSwitchChange(provider, $event)"
                    />
                  </div>
                </div>

                <el-form
                  v-if="providerConfigs[provider.code]"
                  :model="providerConfigs[provider.code]"
                  label-width="160px"
                  class="provider-form"
                >
                  <el-form-item
                    v-for="field in provider.fields"
                    :key="field.key"
                    :label="field.label"
                    :required="field.required"
                  >
                    <div class="provider-field">
                      <el-switch
                        v-if="field.type === 'boolean'"
                        :model-value="providerBooleanFieldValue(provider, field)"
                        active-text="开启"
                        inactive-text="关闭"
                        @change="setProviderFieldValue(provider, field, $event)"
                      />
                      <el-input
                        v-else
                        :model-value="providerFieldValue(provider, field)"
                        :type="fieldInputType(field)"
                        :show-password="field.secret"
                        :placeholder="fieldPlaceholder(field)"
                        @update:model-value="setProviderFieldValue(provider, field, $event)"
                      >
                        <template v-if="isCallbackField(field)" #append>
                          <el-button @click="fillCallbackURL(provider, field.key)">使用当前回调</el-button>
                        </template>
                      </el-input>
                      <div v-if="field.help" class="field-help">{{ field.help }}</div>
                    </div>
                  </el-form-item>
                  <el-form-item>
                    <div class="provider-form-actions">
                      <el-button
                        type="primary"
                        :icon="Check"
                        :loading="providerSaving[provider.code]"
                        @click="saveProviderConfig(provider)"
                      >
                        保存配置
                      </el-button>
                      <el-button
                        :disabled="!provider.configured"
                        :loading="providerSwitching[provider.code]"
                        @click="handleProviderEnabledChange(provider, !provider.enabled)"
                      >
                        {{ provider.enabled ? '停用' : '启用' }}
                      </el-button>
                    </div>
                  </el-form-item>
                </el-form>
              </div>
            </div>
          </el-tab-pane>

          <el-tab-pane label="HTTPS 证书" name="tls">
            <div v-loading="tlsLoading" class="tls-section">
              <div class="tls-summary">
                <div class="summary-item">
                  <span class="summary-value">{{ tlsSettings?.mode || '-' }}</span>
                  <span class="summary-label">TLS 模式</span>
                </div>
                <div class="summary-item">
                  <span class="summary-value">{{ tlsSettings?.ready ? 'Ready' : 'Pending' }}</span>
                  <span class="summary-label">证书状态</span>
                </div>
                <div class="summary-item">
                  <span class="summary-value">{{ tlsCertificateTypeLabel }}</span>
                  <span class="summary-label">证书类型</span>
                </div>
              </div>

              <el-alert
                v-if="tlsSettings && !['https', 'redirect'].includes(tlsSettings.mode)"
                title="当前不是本机 HTTPS 模式，证书可保存但不会被 Nginx 使用。"
                type="info"
                show-icon
                :closable="false"
              />
              <el-alert
                v-else-if="tlsSettings?.certificate?.is_self_signed"
                title="当前证书是自签证书，浏览器会提示不受信任。"
                type="warning"
                show-icon
                :closable="false"
              />
              <el-alert
                v-if="tlsSettings && tlsLocalHTTPSMode && !tlsSettings.writable"
                title="证书目录当前不可写，请检查部署挂载或文件权限。"
                type="error"
                show-icon
                :closable="false"
              />
              <el-alert
                v-if="tlsSettings?.message"
                :title="tlsSettings.message"
                type="warning"
                show-icon
                :closable="false"
              />

              <el-descriptions v-if="tlsSettings" :column="2" border class="tls-descriptions">
                <el-descriptions-item label="访问地址">{{ tlsSettings.base_url || '-' }}</el-descriptions-item>
                <el-descriptions-item label="热加载">{{ tlsSettings.reload_supported ? '可用' : '不可用' }}</el-descriptions-item>
                <el-descriptions-item label="证书文件">{{ tlsSettings.cert_file }}</el-descriptions-item>
                <el-descriptions-item label="私钥文件">{{ tlsSettings.key_file }}</el-descriptions-item>
                <el-descriptions-item label="Subject">{{ tlsSettings.certificate?.subject || '-' }}</el-descriptions-item>
                <el-descriptions-item label="Issuer">{{ tlsSettings.certificate?.issuer || '-' }}</el-descriptions-item>
                <el-descriptions-item label="DNS SAN">
                  {{ tlsSettings.certificate?.dns_names?.join(', ') || '-' }}
                </el-descriptions-item>
                <el-descriptions-item label="有效期">
                  {{ certificateValidityLabel }}
                </el-descriptions-item>
              </el-descriptions>

              <el-form label-width="150px" class="tls-form">
                <el-form-item label="证书 PEM">
                  <el-input
                    v-model="tlsForm.certificate_pem"
                    type="textarea"
                    :rows="8"
                    placeholder="粘贴证书 PEM 内容"
                    resize="vertical"
                  />
                </el-form-item>
                <el-form-item label="私钥 PEM">
                  <el-input
                    v-model="tlsForm.private_key_pem"
                    type="textarea"
                    :rows="8"
                    placeholder="粘贴私钥 PEM 内容"
                    resize="vertical"
                  />
                </el-form-item>
                <el-form-item>
                  <div class="tls-actions">
                    <el-button
                      type="primary"
                      :icon="Check"
                      :loading="tlsSaving"
                      :disabled="!tlsSettings?.writable"
                      @click="saveTLSCertificate"
                    >
                      {{ tlsLocalHTTPSMode ? '保存并热加载' : '保存证书' }}
                    </el-button>
                    <el-button
                      :icon="Refresh"
                      :loading="tlsReloading"
                      :disabled="!tlsSettings?.reload_supported"
                      @click="reloadTLS"
                    >
                      仅热加载
                    </el-button>
                  </div>
                </el-form-item>
              </el-form>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Check, Connection, CopyDocument, Message, Refresh } from '@element-plus/icons-vue'
import {
  getSystemSettings,
  getTLSSettings,
  listAuthLoginProviders,
  reloadTLSCertificate,
  updateSystemSettings,
  updateAuthLoginProviderConfig,
  updateAuthLoginProviderEnabled,
  updateTLSCertificate,
  testSystemEmail,
  type AuthLoginProviderField,
  type AuthLoginProviderInfo,
  type SystemSettings,
  type TLSSettings
} from '@/architecture/presentation/context/api/system-settings'

type SettingsTab = 'email' | 'login' | 'tls'

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const testEmail = ref('')
const activeTab = ref<SettingsTab>('email')
const tlsLoading = ref(false)
const tlsSaving = ref(false)
const tlsReloading = ref(false)
const tlsSettings = ref<TLSSettings | null>(null)
const providersLoading = ref(false)
const authProviders = ref<AuthLoginProviderInfo[]>([])
const providerConfigs = reactive<Record<string, Record<string, string>>>({})
const providerSaving = reactive<Record<string, boolean>>({})
const providerSwitching = reactive<Record<string, boolean>>({})
const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const form = reactive<SystemSettings>({
  registration_mode: 'admin_only',
  email: {
    mode: 'smtp',
    host: '',
    port: 587,
    username: '',
    password: '',
    password_set: false,
    from: '',
    from_name: 'Kageos',
  },
})

const configuredProviderCount = computed(() => authProviders.value.filter((provider) => provider.configured).length)
const enabledProviderCount = computed(() => authProviders.value.filter((provider) => provider.enabled).length)
const tlsCertificateTypeLabel = computed(() => {
  if (!tlsSettings.value?.certificate) {
    return '-'
  }
  return tlsSettings.value.certificate.is_self_signed ? 'Self-signed' : 'Trusted'
})
const tlsLocalHTTPSMode = computed(() => ['https', 'redirect'].includes(tlsSettings.value?.mode || ''))
const certificateValidityLabel = computed(() => {
  const cert = tlsSettings.value?.certificate
  if (!cert?.not_before || !cert?.not_after) {
    return '-'
  }
  return `${formatDateTime(cert.not_before)} - ${formatDateTime(cert.not_after)}`
})

const tlsForm = reactive({
  certificate_pem: '',
  private_key_pem: '',
})

function applySettings(settings: SystemSettings) {
  form.registration_mode = settings.registration_mode
  form.email = {
    ...settings.email,
    password: '',
  }
}

function applyProviderConfigDraft(provider: AuthLoginProviderInfo) {
  const config: Record<string, string> = {}
  provider.fields.forEach((field) => {
    config[field.key] = field.secret ? '' : field.value || ''
  })
  providerConfigs[provider.code] = config
}

function replaceProvider(updated: AuthLoginProviderInfo) {
  const index = authProviders.value.findIndex((item) => item.code === updated.code)
  if (index >= 0) {
    authProviders.value[index] = updated
  } else {
    authProviders.value.push(updated)
  }
  applyProviderConfigDraft(updated)
}

async function loadSettings() {
  loading.value = true
  try {
    applySettings(await getSystemSettings())
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || 'Failed to load settings')
  } finally {
    loading.value = false
  }
}

async function loadAuthProviders() {
  providersLoading.value = true
  try {
    const resp = await listAuthLoginProviders()
    authProviders.value = resp.providers || []
    authProviders.value.forEach(applyProviderConfigDraft)
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || '获取登录方式配置失败')
  } finally {
    providersLoading.value = false
  }
}

async function loadTLSSettings() {
  tlsLoading.value = true
  try {
    tlsSettings.value = await getTLSSettings()
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || '获取 HTTPS 证书状态失败')
  } finally {
    tlsLoading.value = false
  }
}

async function refreshActiveTab() {
  if (activeTab.value === 'login') {
    await loadAuthProviders()
    return
  }
  if (activeTab.value === 'tls') {
    await loadTLSSettings()
    return
  }
  await loadSettings()
}

function handleTabChange(tabName: string | number) {
  if (tabName === 'login' && !authProviders.value.length) {
    loadAuthProviders()
  }
  if (tabName === 'tls' && !tlsSettings.value) {
    loadTLSSettings()
  }
}

async function saveSettings() {
  saving.value = true
  try {
    applySettings(await updateSystemSettings(JSON.parse(JSON.stringify(form))))
    ElMessage.success('Settings saved')
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || 'Failed to save settings')
  } finally {
    saving.value = false
  }
}

function providerStatusType(provider: AuthLoginProviderInfo) {
  if (provider.status === 'enabled') {
    return 'success'
  }
  if (provider.status === 'disabled') {
    return 'warning'
  }
  return 'info'
}

function providerStatusLabel(provider: AuthLoginProviderInfo) {
  if (provider.status === 'enabled') {
    return '已启用'
  }
  if (provider.status === 'disabled') {
    return '已配置未启用'
  }
  return '未配置'
}

function providerActionLabel(action: string) {
  if (action === 'qrcode') {
    return '扫码'
  }
  if (action === 'redirect') {
    return '跳转授权'
  }
  return action || '授权'
}

function fieldInputType(field: AuthLoginProviderField) {
  if (field.secret) {
    return 'password'
  }
  if (field.type === 'url') {
    return 'url'
  }
  return 'text'
}

function fieldPlaceholder(field: AuthLoginProviderField) {
  if (field.secret && field.value_set) {
    return '已配置，留空则保留当前密钥'
  }
  if (field.placeholder) {
    return field.placeholder
  }
  return field.secret ? '请输入密钥' : '请输入配置值'
}

function isCallbackField(field: AuthLoginProviderField) {
  return field.key === 'redirect_url' || field.key === 'callback_url'
}

function providerFieldValue(provider: AuthLoginProviderInfo, field: AuthLoginProviderField) {
  return providerConfigs[provider.code]?.[field.key] || ''
}

function providerBooleanFieldValue(provider: AuthLoginProviderInfo, field: AuthLoginProviderField) {
  return providerFieldValue(provider, field) === 'true'
}

function setProviderFieldValue(provider: AuthLoginProviderInfo, field: AuthLoginProviderField, value: string | number | boolean) {
  let config = providerConfigs[provider.code]
  if (!config) {
    config = {}
    providerConfigs[provider.code] = config
  }
  if (field.type === 'boolean') {
    config[field.key] = value ? 'true' : 'false'
    return
  }
  config[field.key] = String(value ?? '')
}

function callbackURL(provider: AuthLoginProviderInfo) {
  const path = provider.callback_path || ''
  if (!path) {
    return ''
  }
  if (/^https?:\/\//i.test(path)) {
    return path
  }
  if (typeof window === 'undefined') {
    return path
  }
  return `${window.location.origin}${path.startsWith('/') ? path : `/${path}`}`
}

function fillCallbackURL(provider: AuthLoginProviderInfo, fieldKey: string) {
  const config = providerConfigs[provider.code]
  if (!config) {
    return
  }
  config[fieldKey] = callbackURL(provider)
}

async function copyCallbackURL(provider: AuthLoginProviderInfo) {
  try {
    await navigator.clipboard.writeText(callbackURL(provider))
    ElMessage.success('回调地址已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

function buildProviderConfigPayload(provider: AuthLoginProviderInfo) {
  const draft = providerConfigs[provider.code] || {}
  const payload: Record<string, string> = {}
  provider.fields.forEach((field) => {
    const value = (draft[field.key] || '').trim()
    if (field.secret && !value) {
      return
    }
    payload[field.key] = value
  })
  return payload
}

function validateProviderConfig(provider: AuthLoginProviderInfo) {
  const draft = providerConfigs[provider.code] || {}
  const missing = provider.fields.find((field) => {
    if (!field.required) {
      return false
    }
    if (field.secret && field.value_set) {
      return false
    }
    return !(draft[field.key] || '').trim()
  })
  if (missing) {
    ElMessage.warning(`请先填写 ${provider.name} 的 ${missing.label}`)
    return false
  }
  return true
}

async function saveProviderConfig(provider: AuthLoginProviderInfo) {
  if (!validateProviderConfig(provider)) {
    return
  }
  providerSaving[provider.code] = true
  try {
    const updated = await updateAuthLoginProviderConfig(provider.code, buildProviderConfigPayload(provider))
    replaceProvider(updated)
    ElMessage.success('登录方式配置已保存')
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || '保存登录方式配置失败')
  } finally {
    providerSaving[provider.code] = false
  }
}

async function handleProviderEnabledChange(provider: AuthLoginProviderInfo, enabled: boolean) {
  if (enabled && !provider.configured) {
    ElMessage.warning('请先保存完整配置后再启用')
    return
  }
  providerSwitching[provider.code] = true
  try {
    const updated = await updateAuthLoginProviderEnabled(provider.code, enabled)
    replaceProvider(updated)
    ElMessage.success(enabled ? '登录方式已启用' : '登录方式已停用')
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || '更新登录方式状态失败')
  } finally {
    providerSwitching[provider.code] = false
  }
}

function handleProviderSwitchChange(provider: AuthLoginProviderInfo, value: string | number | boolean) {
  handleProviderEnabledChange(provider, Boolean(value))
}

async function sendTestEmail() {
  if (!testEmail.value.trim()) {
    ElMessage.warning('Enter a recipient email first')
    return
  }
  testing.value = true
  try {
    await testSystemEmail(testEmail.value.trim())
    ElMessage.success('Test email sent')
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || 'Failed to send test email')
  } finally {
    testing.value = false
  }
}

function validateTLSForm() {
  if (!tlsForm.certificate_pem.trim()) {
    ElMessage.warning('请填写证书 PEM')
    return false
  }
  if (!tlsForm.private_key_pem.trim()) {
    ElMessage.warning('请填写私钥 PEM')
    return false
  }
  return true
}

async function saveTLSCertificate() {
  if (!validateTLSForm()) {
    return
  }
  tlsSaving.value = true
  try {
    tlsSettings.value = await updateTLSCertificate({
      certificate_pem: tlsForm.certificate_pem,
      private_key_pem: tlsForm.private_key_pem,
      reload: tlsLocalHTTPSMode.value,
    })
    tlsForm.certificate_pem = ''
    tlsForm.private_key_pem = ''
    ElMessage.success(tlsLocalHTTPSMode.value ? 'HTTPS 证书已保存并热加载' : 'HTTPS 证书已保存')
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || '保存 HTTPS 证书失败')
  } finally {
    tlsSaving.value = false
  }
}

async function reloadTLS() {
  tlsReloading.value = true
  try {
    tlsSettings.value = await reloadTLSCertificate()
    ElMessage.success('HTTPS 证书已热加载')
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || '热加载 HTTPS 证书失败')
  } finally {
    tlsReloading.value = false
  }
}

function formatDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

onMounted(() => {
  if (route.query.tab === 'login') {
    activeTab.value = 'login'
    loadAuthProviders()
  } else if (route.query.tab === 'tls') {
    activeTab.value = 'tls'
    loadTLSSettings()
  }
  loadSettings()
})
</script>

<style scoped>
.system-settings-page {
  min-height: 100vh;
  padding: 24px;
  background: var(--el-bg-color-page);
}

.settings-card {
  max-width: 1120px;
  margin: 0 auto;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.card-header h2 {
  margin: 0 0 6px;
  font-size: 20px;
}

.card-header p {
  margin: 0;
  color: var(--el-text-color-secondary);
}

.header-actions,
.test-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.settings-body {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.settings-tabs {
  width: 100%;
}

.settings-form {
  max-width: 760px;
}

.test-row {
  width: 100%;
}

.login-provider-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 160px;
}

.tls-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 160px;
}

.provider-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.tls-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.summary-item {
  min-width: 0;
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-fill-color-lighter);
}

.summary-value {
  display: block;
  font-size: 22px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  line-height: 1.2;
}

.summary-label {
  display: block;
  margin-top: 4px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.provider-panel {
  padding: 18px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.provider-panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}

.provider-heading {
  min-width: 0;
  flex: 1;
}

.provider-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.provider-name {
  font-size: 16px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.provider-heading p {
  margin: 8px 0 0;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.provider-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.callback-path {
  min-width: 0;
  word-break: break-all;
}

.callback-path code {
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}

.provider-enable {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.provider-form {
  max-width: 860px;
}

.tls-form {
  max-width: 900px;
}

.tls-descriptions {
  max-width: 980px;
}

.provider-field {
  width: 100%;
}

.field-help {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.provider-form-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.tls-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

@media (max-width: 768px) {
  .system-settings-page {
    padding: 12px;
  }

  .card-header,
  .provider-panel-header {
    flex-direction: column;
    align-items: stretch;
  }

  .provider-summary,
  .tls-summary {
    grid-template-columns: 1fr;
  }

  .settings-form,
  .provider-form,
  .tls-form {
    max-width: none;
  }

  .test-row,
  .tls-actions {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
