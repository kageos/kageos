<template>
  <div class="system-settings-page">
    <el-card shadow="hover" class="settings-card">
      <template #header>
        <div class="card-header">
          <div>
            <h2>{{ t('route.systemSettings') }}</h2>
            <p>{{ t('systemSettings.subtitle') }}</p>
          </div>
          <div class="header-actions">
            <el-button :icon="Refresh" @click="refreshActiveTab">{{ t('common.refresh') }}</el-button>
            <el-button v-if="activeTab === 'email'" type="primary" :icon="Check" :loading="saving" @click="saveSettings">
              {{ t('connectorProvider.save') }}
            </el-button>
          </div>
        </div>
      </template>

      <div class="settings-layout">
        <aside class="settings-sidebar" :aria-label="t('systemSettings.categoryLabel')">
          <button
            v-for="section in settingsSections"
            :key="section.key"
            type="button"
            class="settings-nav-item"
            :class="{ 'is-active': activeTab === section.key }"
            @click="selectSettingsSection(section.key)"
          >
            <span class="settings-nav-title">{{ section.title }}</span>
            <span class="settings-nav-desc">{{ section.desc }}</span>
          </button>
        </aside>

        <section class="settings-content">
          <div class="section-header">
            <div>
              <h3>{{ currentSection.title }}</h3>
              <p>{{ currentSection.desc }}</p>
            </div>
          </div>

          <div v-if="activeTab === 'email'" v-loading="loading" class="section-pane">
            <el-alert
              v-if="form.registration_mode === 'admin_only'"
              :title="t('systemSettings.registrationDisabled')"
              type="info"
              show-icon
              :closable="false"
            />

            <el-form ref="formRef" :model="form" label-width="180px" class="settings-form">
              <el-divider content-position="left">{{ t('systemSettings.registration') }}</el-divider>
              <el-form-item :label="t('systemSettings.registrationMode')">
                <el-radio-group v-model="form.registration_mode">
                  <el-radio-button value="admin_only">{{ t('systemSettings.adminOnly') }}</el-radio-button>
                  <el-radio-button value="email_code">{{ t('systemSettings.emailVerification') }}</el-radio-button>
                  <el-radio-button value="debug_code">{{ t('systemSettings.debugCode') }}</el-radio-button>
                </el-radio-group>
              </el-form-item>

              <el-divider content-position="left">{{ t('systemSettings.emailService') }}</el-divider>
              <el-form-item :label="t('systemSettings.emailMode')">
                <el-radio-group v-model="form.email.mode">
                  <el-radio-button value="smtp">SMTP</el-radio-button>
                  <el-radio-button value="log">Log</el-radio-button>
                </el-radio-group>
              </el-form-item>

              <el-form-item :label="t('systemSettings.smtpHost')">
                <el-input v-model="form.email.host" placeholder="smtp.example.com" />
              </el-form-item>
              <el-form-item :label="t('systemSettings.smtpPort')">
                <el-input-number v-model="form.email.port" :min="1" :max="65535" />
              </el-form-item>
              <el-form-item :label="t('systemSettings.username')">
                <el-input v-model="form.email.username" :placeholder="t('systemSettings.smtpUsernamePlaceholder')" />
              </el-form-item>
              <el-form-item :label="t('systemSettings.password')">
                <el-input
                  v-model="form.email.password"
                  type="password"
                  show-password
                  :placeholder="form.email.password_set ? t('systemSettings.smtpPasswordKeepPlaceholder') : t('systemSettings.smtpPasswordPlaceholder')"
                />
              </el-form-item>
              <el-form-item :label="t('systemSettings.from')">
                <el-input v-model="form.email.from" placeholder="noreply@example.com" />
              </el-form-item>
              <el-form-item :label="t('systemSettings.fromName')">
                <el-input v-model="form.email.from_name" placeholder="Kageos" />
              </el-form-item>

              <el-divider content-position="left">{{ t('systemSettings.testEmail') }}</el-divider>
              <el-form-item :label="t('systemSettings.recipient')">
                <div class="test-row">
                  <el-input v-model="testEmail" placeholder="admin@example.com" />
                  <el-button :icon="Message" :loading="testing" @click="sendTestEmail">
                    {{ t('systemSettings.sendTest') }}
                  </el-button>
                </div>
              </el-form-item>
            </el-form>
          </div>

          <div v-else-if="activeTab === 'login'" class="section-pane">
            <div v-loading="providersLoading" class="login-provider-section">
              <div class="provider-summary">
                <div class="summary-item">
                  <span class="summary-value">{{ authProviders.length }}</span>
                  <span class="summary-label">{{ t('systemSettings.loginPresetCount') }}</span>
                </div>
                <div class="summary-item">
                  <span class="summary-value">{{ configuredProviderCount }}</span>
                  <span class="summary-label">{{ t('systemSettings.configuredCount') }}</span>
                </div>
                <div class="summary-item">
                  <span class="summary-value">{{ enabledProviderCount }}</span>
                  <span class="summary-label">{{ t('systemSettings.enabledCount') }}</span>
                </div>
              </div>

              <el-empty v-if="!authProviders.length && !providersLoading" :description="t('systemSettings.noLoginProviders')" />

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
                        {{ t('systemSettings.callbackUrl') }}：<code>{{ callbackURL(provider) }}</code>
                      </span>
                      <el-button
                        v-if="provider.callback_path"
                        link
                        type="primary"
                        :icon="CopyDocument"
                        @click="copyCallbackURL(provider)"
                      >
                        {{ t('connectorProvider.copy') }}
                      </el-button>
                      <el-link
                        v-if="provider.docs_url"
                        type="primary"
                        :href="provider.docs_url"
                        target="_blank"
                        :underline="false"
                      >
                        {{ t('systemSettings.platformDocs') }}
                      </el-link>
                    </div>
                  </div>
                  <div class="provider-enable">
                    <span>{{ t('systemSettings.enabled') }}</span>
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
                        :active-text="t('systemSettings.on')"
                        :inactive-text="t('systemSettings.off')"
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
                          <el-button @click="fillCallbackURL(provider, field.key)">
                            {{ t('systemSettings.useCurrentCallback') }}
                          </el-button>
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
                        {{ t('systemSettings.saveProvider') }}
                      </el-button>
                      <el-button
                        :disabled="!provider.configured"
                        :loading="providerSwitching[provider.code]"
                        @click="handleProviderEnabledChange(provider, !provider.enabled)"
                      >
                        {{ provider.enabled ? t('systemSettings.disable') : t('systemSettings.enable') }}
                      </el-button>
                    </div>
                  </el-form-item>
                </el-form>
              </div>
            </div>
          </div>

          <div v-else-if="activeTab === 'tls'" class="section-pane">
            <div v-loading="tlsLoading" class="tls-section">
              <div class="tls-summary">
                <div class="summary-item">
                  <span class="summary-value">{{ tlsSettings?.mode || '-' }}</span>
                  <span class="summary-label">{{ t('systemSettings.tlsMode') }}</span>
                </div>
                <div class="summary-item">
                  <span class="summary-value">{{ tlsSettings?.ready ? t('systemSettings.ready') : t('systemSettings.pending') }}</span>
                  <span class="summary-label">{{ t('systemSettings.certificateStatus') }}</span>
                </div>
                <div class="summary-item">
                  <span class="summary-value">{{ tlsCertificateTypeLabel }}</span>
                  <span class="summary-label">{{ t('systemSettings.certificateType') }}</span>
                </div>
              </div>

              <el-alert
                v-if="tlsSettings && !['https', 'redirect'].includes(tlsSettings.mode)"
                :title="t('systemSettings.notLocalHttpsMode')"
                type="info"
                show-icon
                :closable="false"
              />
              <el-alert
                v-else-if="tlsSettings?.certificate?.is_self_signed"
                :title="t('systemSettings.selfSignedWarning')"
                type="warning"
                show-icon
                :closable="false"
              />
              <el-alert
                v-if="tlsSettings && tlsLocalHTTPSMode && !tlsSettings.writable"
                :title="t('systemSettings.tlsDirNotWritable')"
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
                <el-descriptions-item :label="t('systemSettings.baseUrl')">{{ tlsSettings.base_url || '-' }}</el-descriptions-item>
                <el-descriptions-item :label="t('systemSettings.hotReload')">{{ tlsSettings.reload_supported ? t('systemSettings.available') : t('systemSettings.unavailable') }}</el-descriptions-item>
                <el-descriptions-item :label="t('systemSettings.certFile')">{{ tlsSettings.cert_file }}</el-descriptions-item>
                <el-descriptions-item :label="t('systemSettings.keyFile')">{{ tlsSettings.key_file }}</el-descriptions-item>
                <el-descriptions-item label="Subject">{{ tlsSettings.certificate?.subject || '-' }}</el-descriptions-item>
                <el-descriptions-item label="Issuer">{{ tlsSettings.certificate?.issuer || '-' }}</el-descriptions-item>
                <el-descriptions-item label="DNS SAN">
                  {{ tlsSettings.certificate?.dns_names?.join(', ') || '-' }}
                </el-descriptions-item>
                <el-descriptions-item :label="t('systemSettings.validity')">
                  {{ certificateValidityLabel }}
                </el-descriptions-item>
              </el-descriptions>

              <el-form label-width="150px" class="tls-form">
                <el-form-item :label="t('systemSettings.certificatePem')">
                  <el-input
                    v-model="tlsForm.certificate_pem"
                    type="textarea"
                    :rows="8"
                    :placeholder="t('systemSettings.certificatePemPlaceholder')"
                    resize="vertical"
                  />
                </el-form-item>
                <el-form-item :label="t('systemSettings.privateKeyPem')">
                  <el-input
                    v-model="tlsForm.private_key_pem"
                    type="textarea"
                    :rows="8"
                    :placeholder="t('systemSettings.privateKeyPemPlaceholder')"
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
                      {{ tlsLocalHTTPSMode ? t('systemSettings.saveAndReload') : t('systemSettings.saveCertificate') }}
                    </el-button>
                    <el-button
                      :icon="Refresh"
                      :loading="tlsReloading"
                      :disabled="!tlsSettings?.reload_supported"
                      @click="reloadTLS"
                    >
                      {{ t('systemSettings.reloadOnly') }}
                    </el-button>
                  </div>
                </el-form-item>
              </el-form>
            </div>
          </div>

          <div v-else-if="activeTab === 'connectors'" class="section-pane">
            <ConnectorProviderManagementPage :key="connectorPanelKey" embedded />
          </div>

          <div v-else-if="activeTab === 'openapi'" class="section-pane">
            <OpenAPITokenManagementPage :key="openapiPanelKey" embedded />
          </div>

          <div v-else-if="activeTab === 'appearance'" class="section-pane">
            <div class="preference-grid">
              <button
                v-for="theme in availableThemes"
                :key="theme.name"
                type="button"
                class="preference-card"
                :class="{ 'is-active': currentThemeName === theme.name }"
                @click="handleThemeChange(theme.name)"
              >
                <span class="preference-card-title">{{ theme.label }}</span>
                <span class="preference-card-desc">{{ theme.mode === 'dark' ? t('workspace.darkUi') : t('workspace.lightUi') }}</span>
              </button>
            </div>
          </div>

          <div v-else-if="activeTab === 'language'" class="section-pane">
            <div class="preference-grid">
              <button
                v-for="option in localeStore.localeOptions"
                :key="option.value"
                type="button"
                class="preference-card"
                :class="{ 'is-active': option.value === localeStore.currentLocale }"
                @click="handleLocaleChange(option.value)"
              >
                <span class="preference-card-title">{{ option.flag }} {{ option.nativeLabel }}</span>
                <span class="preference-card-desc">{{ option.englishLabel }}</span>
              </button>
            </div>
          </div>
        </section>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Check, CopyDocument, Message, Refresh } from '@element-plus/icons-vue'
import { useLocaleStore, useThemeStore } from '@/architecture/presentation/context/appStoresContext'
import type { SupportedLocale } from '@/architecture/shared/i18n'
import ConnectorProviderManagementPage from '@/architecture/presentation/features/connector/pages/ConnectorProviderManagementPage.vue'
import OpenAPITokenManagementPage from '@/architecture/presentation/features/agent/pages/OpenAPITokenManagementPage.vue'
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

type SettingsTab = 'email' | 'login' | 'tls' | 'connectors' | 'openapi' | 'appearance' | 'language'

interface SettingsSection {
  key: SettingsTab
  title: string
  desc: string
}

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const testEmail = ref('')
const activeTab = ref<SettingsTab>('email')
const connectorPanelKey = ref(0)
const openapiPanelKey = ref(0)
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
const localeStore = useLocaleStore()
const themeStore = useThemeStore()

const settingsSections = computed<SettingsSection[]>(() => [
  { key: 'email', title: t('systemSettings.sections.emailTitle'), desc: t('systemSettings.sections.emailDesc') },
  { key: 'login', title: t('systemSettings.sections.loginTitle'), desc: t('systemSettings.sections.loginDesc') },
  { key: 'openapi', title: t('systemSettings.sections.openapiTitle'), desc: t('systemSettings.sections.openapiDesc') },
  { key: 'tls', title: t('systemSettings.sections.tlsTitle'), desc: t('systemSettings.sections.tlsDesc') },
  { key: 'connectors', title: t('systemSettings.sections.connectorsTitle'), desc: t('systemSettings.sections.connectorsDesc') },
  { key: 'appearance', title: t('systemSettings.sections.appearanceTitle'), desc: t('systemSettings.sections.appearanceDesc') },
  { key: 'language', title: t('systemSettings.sections.languageTitle'), desc: t('systemSettings.sections.languageDesc') },
])

const currentSection = computed(() => {
  return settingsSections.value.find((section) => section.key === activeTab.value) || settingsSections.value[0]!
})

const availableThemes = computed(() => themeStore.getAvailableThemes())
const currentThemeName = computed(() => themeStore.currentTheme.name)

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
  return tlsSettings.value.certificate.is_self_signed ? t('systemSettings.selfSigned') : t('systemSettings.trusted')
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
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.loadSettingsFailed'))
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
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.loadAuthProvidersFailed'))
  } finally {
    providersLoading.value = false
  }
}

async function loadTLSSettings() {
  tlsLoading.value = true
  try {
    tlsSettings.value = await getTLSSettings()
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.loadTLSFailed'))
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
  if (activeTab.value === 'connectors') {
    connectorPanelKey.value += 1
    return
  }
  if (activeTab.value === 'openapi') {
    openapiPanelKey.value += 1
    return
  }
  if (activeTab.value === 'appearance' || activeTab.value === 'language') {
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

function isSettingsTab(value: unknown): value is SettingsTab {
  return typeof value === 'string' && settingsSections.value.some((section) => section.key === value)
}

function selectSettingsSection(tabName: SettingsTab) {
  activeTab.value = tabName
  handleTabChange(tabName)
  router.replace({
    path: route.path,
    query: { ...route.query, tab: tabName }
  })
}

function handleThemeChange(themeName: string) {
  const theme = themeStore.getAvailableThemes().find((item) => item.name === themeName)
  if (theme) {
    themeStore.setTheme(theme)
  }
}

function handleLocaleChange(locale: string) {
  localeStore.setAppLocale(locale as SupportedLocale)
}

async function saveSettings() {
  saving.value = true
  try {
    applySettings(await updateSystemSettings(JSON.parse(JSON.stringify(form))))
    ElMessage.success(t('systemSettings.settingsSaved'))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.saveSettingsFailed'))
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
    return t('systemSettings.statusEnabled')
  }
  if (provider.status === 'disabled') {
    return t('systemSettings.statusConfiguredDisabled')
  }
  return t('systemSettings.statusUnconfigured')
}

function providerActionLabel(action: string) {
  if (action === 'qrcode') {
    return t('systemSettings.actionQrcode')
  }
  if (action === 'redirect') {
    return t('systemSettings.actionRedirect')
  }
  return action || t('systemSettings.actionAuthorize')
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
    return t('systemSettings.secretKeepPlaceholder')
  }
  if (field.placeholder) {
    return field.placeholder
  }
  return field.secret ? t('systemSettings.secretPlaceholder') : t('systemSettings.configValuePlaceholder')
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
    ElMessage.success(t('connectorProvider.callbackCopied'))
  } catch {
    ElMessage.error(t('connectorProvider.copyFailed'))
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
    ElMessage.warning(t('systemSettings.providerFieldMissing', { provider: provider.name, field: missing.label }))
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
    ElMessage.success(t('systemSettings.providerSaved'))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.providerSaveFailed'))
  } finally {
    providerSaving[provider.code] = false
  }
}

async function handleProviderEnabledChange(provider: AuthLoginProviderInfo, enabled: boolean) {
  if (enabled && !provider.configured) {
    ElMessage.warning(t('systemSettings.saveBeforeEnable'))
    return
  }
  providerSwitching[provider.code] = true
  try {
    const updated = await updateAuthLoginProviderEnabled(provider.code, enabled)
    replaceProvider(updated)
    ElMessage.success(enabled ? t('systemSettings.providerEnabled') : t('systemSettings.providerDisabled'))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.providerStatusFailed'))
  } finally {
    providerSwitching[provider.code] = false
  }
}

function handleProviderSwitchChange(provider: AuthLoginProviderInfo, value: string | number | boolean) {
  handleProviderEnabledChange(provider, Boolean(value))
}

async function sendTestEmail() {
  if (!testEmail.value.trim()) {
    ElMessage.warning(t('systemSettings.recipientRequired'))
    return
  }
  testing.value = true
  try {
    await testSystemEmail(testEmail.value.trim())
    ElMessage.success(t('systemSettings.testEmailSent'))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.testEmailFailed'))
  } finally {
    testing.value = false
  }
}

function validateTLSForm() {
  if (!tlsForm.certificate_pem.trim()) {
    ElMessage.warning(t('systemSettings.certificatePemRequired'))
    return false
  }
  if (!tlsForm.private_key_pem.trim()) {
    ElMessage.warning(t('systemSettings.privateKeyPemRequired'))
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
    ElMessage.success(tlsLocalHTTPSMode.value ? t('systemSettings.tlsSavedAndReloaded') : t('systemSettings.tlsSaved'))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.tlsSaveFailed'))
  } finally {
    tlsSaving.value = false
  }
}

async function reloadTLS() {
  tlsReloading.value = true
  try {
    tlsSettings.value = await reloadTLSCertificate()
    ElMessage.success(t('systemSettings.tlsReloaded'))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('systemSettings.tlsReloadFailed'))
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
  const tab = Array.isArray(route.query.tab) ? route.query.tab[0] : route.query.tab
  if (isSettingsTab(tab)) {
    activeTab.value = tab
    handleTabChange(tab)
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
  max-width: 1280px;
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

.settings-layout {
  display: grid;
  grid-template-columns: 252px minmax(0, 1fr);
  gap: 22px;
  align-items: start;
}

.settings-sidebar {
  display: flex;
  flex-direction: column;
  gap: 8px;
  position: sticky;
  top: 16px;
  padding: 8px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: color-mix(in srgb, var(--el-fill-color-light) 55%, var(--el-bg-color) 45%);
}

.settings-nav-item {
  width: 100%;
  min-height: 64px;
  padding: 12px 14px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: color-mix(in srgb, var(--el-bg-color) 88%, var(--el-fill-color-light) 12%);
  color: var(--el-text-color-regular);
  text-align: left;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  transition: border-color 0.18s ease, background 0.18s ease, color 0.18s ease, box-shadow 0.18s ease;
}

.settings-nav-item:hover {
  border-color: color-mix(in srgb, var(--el-color-primary) 18%, var(--el-border-color-light) 82%);
  background: color-mix(in srgb, var(--el-color-primary) 6%, var(--el-bg-color) 94%);
}

.settings-nav-item.is-active {
  border-color: color-mix(in srgb, var(--el-color-primary) 42%, var(--el-border-color) 58%);
  background: color-mix(in srgb, var(--el-color-primary) 14%, var(--el-bg-color) 86%);
  color: var(--el-color-primary);
  box-shadow: inset 3px 0 0 var(--el-color-primary);
}

.settings-nav-title {
  display: block;
  font-size: 14px;
  font-weight: 700;
  line-height: 1.3;
}

.settings-nav-desc {
  display: block;
  margin-top: 5px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.35;
}

.settings-content {
  min-width: 0;
}

.section-header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 16px;
  margin-bottom: 18px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.section-header h3 {
  margin: 0 0 6px;
  font-size: 18px;
  color: var(--el-text-color-primary);
}

.section-header p {
  margin: 0;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.section-pane {
  min-width: 0;
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

.preference-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
  max-width: 720px;
}

.preference-card {
  min-height: 82px;
  padding: 14px 16px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.18s ease, background 0.18s ease, color 0.18s ease;
}

.preference-card:hover {
  border-color: var(--el-color-primary-light-5);
}

.preference-card.is-active {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.preference-card-title {
  display: block;
  font-size: 15px;
  font-weight: 700;
}

.preference-card-desc {
  display: block;
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.45;
}

@media (max-width: 768px) {
  .system-settings-page {
    padding: 12px;
  }

  .settings-layout {
    grid-template-columns: 1fr;
  }

  .settings-sidebar {
    position: static;
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
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

@media (max-width: 520px) {
  .settings-sidebar,
  .preference-grid {
    grid-template-columns: 1fr;
  }
}
</style>
