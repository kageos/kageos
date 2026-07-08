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
            <el-button :icon="QuestionFilled" @click="openCurrentDocs">
              {{ t('systemSettings.viewDocs') }}
            </el-button>
          </div>

          <div v-if="activeTab === 'email'" v-loading="loading" class="section-pane">
            <el-alert
              v-if="form.registration_mode === 'admin_only'"
              :title="t('systemSettings.registrationDisabled')"
              type="info"
              show-icon
              :closable="false"
            />

            <el-form ref="formRef" :model="form" label-width="120px" class="settings-form">
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
                  label-width="120px"
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

          <div v-else-if="activeTab === 'connectors'" class="section-pane">
            <ConnectorProviderManagementPage :key="connectorPanelKey" embedded />
          </div>

          <div v-else-if="activeTab === 'openapi'" class="section-pane">
            <OpenAPITokenManagementPage :key="openapiPanelKey" embedded />
          </div>

          <div v-else-if="activeTab === 'users'" class="section-pane">
            <SystemUserManagementPage :key="usersPanelKey" />
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
import { Check, CopyDocument, Message, QuestionFilled, Refresh } from '@element-plus/icons-vue'
import { useLocaleStore, useThemeStore } from '@/architecture/presentation/context/appStoresContext'
import type { SupportedLocale } from '@/architecture/shared/i18n'
import { getKageosDocsURL, openExternalURL, type KageosDocSlug } from '@/architecture/shared/config/externalLinks'
import { featureFlags } from '@/architecture/shared/config/features'
import ConnectorProviderManagementPage from '@/architecture/presentation/features/connector/pages/ConnectorProviderManagementPage.vue'
import OpenAPITokenManagementPage from '@/architecture/presentation/features/agent/pages/OpenAPITokenManagementPage.vue'
import SystemUserManagementPage from '@/architecture/presentation/features/system/pages/SystemUserManagementPage.vue'
import {
  getSystemSettings,
  listAuthLoginProviders,
  updateSystemSettings,
  updateAuthLoginProviderConfig,
  updateAuthLoginProviderEnabled,
  testSystemEmail,
  type AuthLoginProviderField,
  type AuthLoginProviderInfo,
  type SystemSettings
} from '@/architecture/presentation/context/api/system-settings'

type SettingsTab = 'email' | 'login' | 'connectors' | 'openapi' | 'users' | 'appearance' | 'language'

interface SettingsSection {
  key: SettingsTab
  title: string
  desc: string
}

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const testEmail = ref('')
const defaultSettingsTab: SettingsTab = 'login'
const activeTab = ref<SettingsTab>(defaultSettingsTab)
const connectorPanelKey = ref(0)
const openapiPanelKey = ref(0)
const usersPanelKey = ref(0)
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

const allSettingsSections = computed<SettingsSection[]>(() => [
  { key: 'email', title: t('systemSettings.sections.emailTitle'), desc: t('systemSettings.sections.emailDesc') },
  { key: 'login', title: t('systemSettings.sections.loginTitle'), desc: t('systemSettings.sections.loginDesc') },
  { key: 'users', title: t('systemSettings.sections.usersTitle'), desc: t('systemSettings.sections.usersDesc') },
  { key: 'openapi', title: t('systemSettings.sections.openapiTitle'), desc: t('systemSettings.sections.openapiDesc') },
  { key: 'connectors', title: t('systemSettings.sections.connectorsTitle'), desc: t('systemSettings.sections.connectorsDesc') },
  { key: 'appearance', title: t('systemSettings.sections.appearanceTitle'), desc: t('systemSettings.sections.appearanceDesc') },
  { key: 'language', title: t('systemSettings.sections.languageTitle'), desc: t('systemSettings.sections.languageDesc') },
])

const settingsSections = computed<SettingsSection[]>(() => {
  return allSettingsSections.value.filter((section) => {
    if (section.key === 'email') return featureFlags.systemEmailSettings
    if (section.key === 'connectors') return featureFlags.connectorSettings
    if (section.key === 'openapi') return featureFlags.openapiTokens
    return true
  })
})

const settingsDocSlugMap: Record<SettingsTab, KageosDocSlug> = {
  email: 'runtime',
  login: 'login',
  connectors: 'connectors',
  openapi: 'api',
  users: 'runtime',
  appearance: 'docs',
  language: 'docs',
}

const currentSection = computed(() => {
  return settingsSections.value.find((section) => section.key === activeTab.value) || settingsSections.value[0]!
})

const currentDocsURL = computed(() => {
  return getKageosDocsURL(settingsDocSlugMap[activeTab.value], localeStore.currentLocale)
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

async function refreshActiveTab() {
  if (activeTab.value === 'login') {
    await loadAuthProviders()
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
  if (activeTab.value === 'users') {
    usersPanelKey.value += 1
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

function openCurrentDocs() {
  openExternalURL(currentDocsURL.value)
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

onMounted(() => {
  const tab = Array.isArray(route.query.tab) ? route.query.tab[0] : route.query.tab
  if (isSettingsTab(tab)) {
    activeTab.value = tab
  } else if (!isSettingsTab(activeTab.value)) {
    activeTab.value = settingsSections.value[0]?.key || defaultSettingsTab
  }
  if (activeTab.value === 'email') {
    loadSettings()
  } else {
    handleTabChange(activeTab.value)
  }
})
</script>

<style scoped>
.system-settings-page {
  min-height: 100vh;
  padding: 32px 40px;
  background: var(--bg-primary);
}

.settings-card {
  width: 100%;
  max-width: 1440px;
  margin: 0 auto;
  border-radius: var(--border-radius-xl);
  border: 1px solid var(--border-light);
  box-shadow: var(--app-shell-panel-shadow-soft);
  background: var(--bg-secondary);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 24px 32px;
  border-bottom: 1px solid var(--border-light);
}

.card-header h2 {
  margin: 0 0 6px;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
}

.card-header p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
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
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 0;
  align-items: stretch;
}

.settings-sidebar {
  border-right: 1px solid var(--border-light);
  background: var(--bg-secondary);
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  border-radius: var(--border-radius-xl) 0 0 var(--border-radius-xl);
}

.settings-nav-item {
  width: 100%;
  min-height: 64px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  border: 1px solid transparent;
  border-radius: var(--border-radius-base);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  cursor: pointer;
  text-align: left;
  transition: all 0.25s cubic-bezier(0.25, 0.8, 0.25, 1);
}

.settings-nav-item:hover {
  background: var(--el-fill-color);
}

.settings-nav-item.is-active {
  background: var(--el-fill-color-blank);
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(var(--color-primary-rgb), 0.1);
}

.settings-nav-title {
  display: block;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.3;
}

.settings-nav-desc {
  display: block;
  margin-top: 4px;
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.35;
}

.settings-content {
  min-width: 0;
  background: transparent;
  border-radius: 0 var(--border-radius-xl) var(--border-radius-xl) 0;
  padding: 32px 40px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding-bottom: 20px;
  margin-bottom: 24px;
  border-bottom: 1px solid var(--border-light);
}

.section-header h3 {
  margin: 0 0 8px;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.section-header p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
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

.provider-summary {
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
  font-weight: 600;
  color: var(--text-primary);
}

.provider-heading p {
  margin: 8px 0 0;
  color: var(--text-secondary);
  line-height: 1.5;
}

.provider-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
  font-size: 13px;
  color: var(--text-secondary);
}

.callback-path {
  min-width: 0;
  word-break: break-all;
}

.callback-path code {
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  color: var(--text-primary);
}

.provider-enable {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-secondary);
  white-space: nowrap;
}

.provider-form {
  max-width: 860px;
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

.preference-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
  max-width: 720px;
}

.preference-card {
  min-height: 82px;
  padding: 16px 20px;
  border: 1px solid transparent;
  border-radius: var(--border-radius-lg);
  background: var(--bg-tertiary);
  color: var(--text-regular);
  text-align: left;
  cursor: pointer;
  transition: all 0.25s cubic-bezier(0.25, 0.8, 0.25, 1);
}

.preference-card:hover {
  background: var(--el-fill-color);
}

.preference-card.is-active {
  background: var(--el-fill-color-blank);
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(var(--color-primary-rgb), 0.15);
  color: var(--color-primary);
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

  .provider-summary {
    grid-template-columns: 1fr;
  }

  .settings-form,
  .provider-form {
    max-width: none;
  }

  .test-row {
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
