<template>
  <div class="notification-route-panel" v-loading="loading">
    <div class="route-panel-header">
      <div>
        <h3>{{ t('notificationRoute.title') }}</h3>
        <p>{{ t('notificationRoute.subtitle') }}</p>
      </div>
      <el-button :icon="Refresh" :disabled="!normalizedScopePath" @click="loadRoutes">
        {{ t('common.refresh') }}
      </el-button>
    </div>

    <el-empty v-if="!normalizedScopePath" :description="t('notificationRoute.noNode')" />

    <template v-else>
      <div class="scope-summary">
        <span class="scope-label">{{ t('notificationRoute.scopePath') }}</span>
        <code>{{ normalizedScopePath }}</code>
      </div>

      <el-alert
        v-if="inheritedRouteCount > 0"
        :title="inheritedSummary"
        type="info"
        :closable="false"
        show-icon
      />
      <el-alert
        v-else
        :title="t('notificationRoute.noInherited')"
        type="info"
        :closable="false"
        show-icon
      />

      <div class="route-channel-list">
        <section
          v-for="definition in channelDefinitions"
          :key="definition.channel"
          class="route-channel-section"
        >
          <el-alert
            v-if="routeForms[definition.channel].configured && !routeForms[definition.channel].enabled"
            class="route-disabled-alert"
            type="warning"
            :closable="false"
            show-icon
            :title="t('notificationRoute.configuredButDisabled')"
          />
          <div class="channel-header">
            <div class="channel-title">
              <span class="channel-mark">{{ definition.mark }}</span>
              <div>
                <div class="channel-name-row">
                  <h4>{{ definition.name }}</h4>
                  <el-tag size="small" :type="channelStatusType(routeForms[definition.channel])">
                    {{ channelStatusLabel(routeForms[definition.channel]) }}
                  </el-tag>
                  <el-tag
                    v-if="!routeForms[definition.channel].configured && inheritedByChannel[definition.channel]"
                    size="small"
                    effect="plain"
                  >
                    {{ t('notificationRoute.inheritedBadge') }}
                  </el-tag>
                  <el-tag
                    v-if="routeForms[definition.channel].configured"
                    size="small"
                    effect="plain"
                    type="success"
                  >
                    {{ t('notificationRoute.currentBadge') }}
                  </el-tag>
                </div>
                <p>{{ definition.hint }}</p>
                <p v-if="inheritedByChannel[definition.channel] && !routeForms[definition.channel].configured" class="channel-guide">
                  {{ t('notificationRoute.inheritedFrom', { path: inheritedByChannel[definition.channel]?.scope_path }) }}
                </p>
                <p v-else class="channel-guide">{{ definition.guide }}</p>
              </div>
            </div>
            <el-switch
              v-model="routeForms[definition.channel].enabled"
              inline-prompt
              :active-text="t('userSettings.enabled')"
              :inactive-text="t('userSettings.disabled')"
            />
          </div>

          <el-form
            :model="routeForms[definition.channel]"
            label-width="118px"
            class="route-form"
          >
            <el-form-item :label="t('userSettings.displayName')">
              <el-input
                v-model="routeForms[definition.channel].display_name"
                :placeholder="channelDefaultDisplayName(definition.channel)"
                clearable
              />
            </el-form-item>

            <el-form-item label="Webhook">
              <div class="secret-row">
                <el-input
                  v-model="routeForms[definition.channel].webhook_url"
                  :disabled="routeForms[definition.channel].clear_webhook_url"
                  :placeholder="webhookPlaceholder(definition.channel)"
                  clearable
                />
                <el-button
                  v-if="routeForms[definition.channel].has_webhook_url || routeForms[definition.channel].clear_webhook_url"
                  :type="routeForms[definition.channel].clear_webhook_url ? 'primary' : 'info'"
                  plain
                  @click="toggleClearWebhook(definition.channel)"
                >
                  {{ routeForms[definition.channel].clear_webhook_url ? t('userSettings.keepWebhook') : t('userSettings.clearWebhook') }}
                </el-button>
              </div>
              <p class="form-tip">{{ webhookStateText(routeForms[definition.channel]) }}</p>
            </el-form-item>

            <el-form-item :label="t('userSettings.signingSecret')">
              <div class="secret-row">
                <el-input
                  v-model="routeForms[definition.channel].secret"
                  type="password"
                  show-password
                  :disabled="routeForms[definition.channel].clear_secret"
                  :placeholder="secretPlaceholder(definition.channel)"
                  clearable
                />
                <el-button
                  v-if="routeForms[definition.channel].has_secret || routeForms[definition.channel].clear_secret"
                  :type="routeForms[definition.channel].clear_secret ? 'primary' : 'info'"
                  plain
                  @click="toggleClearSecret(definition.channel)"
                >
                  {{ routeForms[definition.channel].clear_secret ? t('userSettings.keepSecret') : t('userSettings.clearSecret') }}
                </el-button>
              </div>
              <p class="form-tip">{{ secretStateText(routeForms[definition.channel]) }}</p>
            </el-form-item>

            <el-form-item :label="t('notificationRoute.requireAuth')">
              <div class="auth-row">
                <el-switch v-model="routeForms[definition.channel].require_auth" />
                <span>{{ t('notificationRoute.requireAuthHint') }}</span>
              </div>
            </el-form-item>

            <el-form-item>
              <div class="channel-actions">
                <el-button
                  type="primary"
                  :icon="Check"
                  :loading="savingRoute[definition.channel]"
                  @click="saveRoute(definition.channel)"
                >
                  {{ t('userSettings.saveConfig') }}
                </el-button>
                <el-button
                  :icon="Promotion"
                  :loading="testingRoute[definition.channel]"
                  :disabled="deletingRoute[definition.channel]"
                  @click="testRoute(definition.channel)"
                >
                  {{ t('userSettings.testSend') }}
                </el-button>
                <el-button
                  text
                  type="danger"
                  :icon="Delete"
                  :loading="deletingRoute[definition.channel]"
                  :disabled="!routeForms[definition.channel].configured"
                  @click="deleteRoute(definition.channel)"
                >
                  {{ t('userSettings.deleteConfig') }}
                </el-button>
                <span v-if="routeForms[definition.channel].updated_at" class="updated-at">
                  {{ t('userSettings.updatedAt', { time: formatDateTime(routeForms[definition.channel].updated_at) }) }}
                </span>
              </div>
              <div class="delivery-status">
                <el-tag size="small" :type="deliveryStatusType(routeForms[definition.channel])">
                  {{ deliveryStatusLabel(routeForms[definition.channel]) }}
                </el-tag>
                <span v-if="routeForms[definition.channel].last_success_at">
                  {{ t('userSettings.lastSuccessAt', { time: formatDateTime(routeForms[definition.channel].last_success_at) }) }}
                </span>
                <span v-if="routeForms[definition.channel].last_test_at">
                  {{ t('userSettings.lastTestAt', { time: formatDateTime(routeForms[definition.channel].last_test_at) }) }}
                </span>
                <span v-if="routeForms[definition.channel].last_error" class="delivery-error">
                  {{ routeForms[definition.channel].last_error }}
                </span>
              </div>
            </el-form-item>
          </el-form>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Check, Delete, Promotion, Refresh } from '@element-plus/icons-vue'
import {
  deleteMessageNotificationRoute,
  listMessageNotificationRoutes,
  testMessageNotificationRoute,
  upsertMessageNotificationRoute,
  type MessageNotificationRouteInfo
} from '@/architecture/presentation/context/api/message'

type ChannelCode = 'feishu' | 'wecom' | 'dingtalk'

interface ChannelDefinition {
  channel: ChannelCode
  name: string
  mark: string
  hint: string
  guide: string
}

interface RouteFormState {
  channel: ChannelCode
  configured: boolean
  enabled: boolean
  display_name: string
  webhook_url: string
  secret: string
  has_webhook_url: boolean
  has_secret: boolean
  clear_webhook_url: boolean
  clear_secret: boolean
  require_auth: boolean
  metadata: Record<string, string>
  updated_at?: string
  last_success_at?: string
  last_failed_at?: string
  last_test_at?: string
  last_error?: string
  fail_count: number
}

const props = withDefaults(defineProps<{
  scopePath?: string
  scopeType?: 'workspace' | 'directory' | 'function' | string
}>(), {
  scopePath: '',
  scopeType: '',
})

const { t } = useI18n()
const loading = ref(false)

const normalizedScopePath = computed(() => normalizeScopePath(props.scopePath))
const resolvedScopeType = computed(() => props.scopeType || inferScopeType(normalizedScopePath.value))

const channelDefinitions = computed<ChannelDefinition[]>(() => [
  {
    channel: 'feishu',
    name: t('userSettings.channelFeishu'),
    mark: t('userSettings.channelFeishuMark'),
    hint: t('userSettings.channelFeishuHint'),
    guide: t('userSettings.channelFeishuGuide')
  },
  {
    channel: 'wecom',
    name: t('userSettings.channelWecom'),
    mark: t('userSettings.channelWecomMark'),
    hint: t('userSettings.channelWecomHint'),
    guide: t('userSettings.channelWecomGuide')
  },
  {
    channel: 'dingtalk',
    name: t('userSettings.channelDingtalk'),
    mark: t('userSettings.channelDingtalkMark'),
    hint: t('userSettings.channelDingtalkHint'),
    guide: t('userSettings.channelDingtalkGuide')
  }
])

const routeForms = reactive<Record<ChannelCode, RouteFormState>>({
  feishu: createDefaultRouteForm('feishu'),
  wecom: createDefaultRouteForm('wecom'),
  dingtalk: createDefaultRouteForm('dingtalk')
})

const inheritedByChannel = reactive<Record<ChannelCode, MessageNotificationRouteInfo | null>>({
  feishu: null,
  wecom: null,
  dingtalk: null
})

const savingRoute = reactive<Record<ChannelCode, boolean>>({
  feishu: false,
  wecom: false,
  dingtalk: false
})

const testingRoute = reactive<Record<ChannelCode, boolean>>({
  feishu: false,
  wecom: false,
  dingtalk: false
})

const deletingRoute = reactive<Record<ChannelCode, boolean>>({
  feishu: false,
  wecom: false,
  dingtalk: false
})

const inheritedRouteCount = computed(() => {
  return channelDefinitions.value.filter((definition) => Boolean(inheritedByChannel[definition.channel])).length
})

const inheritedSummary = computed(() => {
  const paths = new Set<string>()
  channelDefinitions.value.forEach((definition) => {
    const inherited = inheritedByChannel[definition.channel]
    if (inherited?.scope_path) {
      paths.add(inherited.scope_path)
    }
  })
  if (paths.size === 0) {
    return t('notificationRoute.noInherited')
  }
  return t('notificationRoute.inheritedFrom', { path: [...paths].join(', ') })
})

function createDefaultRouteForm(channel: ChannelCode): RouteFormState {
  return {
    channel,
    configured: false,
    enabled: true,
    display_name: channelDefaultDisplayName(channel),
    webhook_url: '',
    secret: '',
    has_webhook_url: false,
    has_secret: false,
    clear_webhook_url: false,
    clear_secret: false,
    require_auth: true,
    metadata: {},
    fail_count: 0
  }
}

function channelLabel(channel: ChannelCode): string {
  if (channel === 'feishu') return t('userSettings.channelFeishu')
  if (channel === 'wecom') return t('userSettings.channelWecom')
  return t('userSettings.channelDingtalk')
}

function channelDefaultDisplayName(channel: ChannelCode): string {
  return t('userSettings.channelDefaultName', { channel: channelLabel(channel) })
}

function normalizeChannel(channel: string): ChannelCode | null {
  if (channel === 'feishu' || channel === 'wecom' || channel === 'dingtalk') {
    return channel
  }
  return null
}

function resetRouteForm(channel: ChannelCode) {
  Object.assign(routeForms[channel], createDefaultRouteForm(channel))
}

function applyRouteInfo(info: MessageNotificationRouteInfo) {
  const channel = normalizeChannel(info.channel)
  if (!channel) {
    return
  }
  const form = routeForms[channel]
  form.configured = true
  form.enabled = Boolean(info.enabled)
  form.display_name = info.display_name || channelDefaultDisplayName(channel)
  form.webhook_url = ''
  form.secret = ''
  form.has_webhook_url = Boolean(info.has_webhook_url)
  form.has_secret = Boolean(info.has_secret)
  form.clear_webhook_url = false
  form.clear_secret = false
  form.require_auth = info.require_auth !== false
  form.metadata = info.metadata || {}
  form.updated_at = info.updated_at
  form.last_success_at = info.last_success_at
  form.last_failed_at = info.last_failed_at
  form.last_test_at = info.last_test_at
  form.last_error = info.last_error || ''
  form.fail_count = info.fail_count || 0
}

function applyInheritedRoutes(rows: MessageNotificationRouteInfo[]) {
  channelDefinitions.value.forEach((definition) => {
    inheritedByChannel[definition.channel] = null
  })
  rows.forEach((row) => {
    const channel = normalizeChannel(row.channel)
    if (!channel || inheritedByChannel[channel]) {
      return
    }
    inheritedByChannel[channel] = row
  })
}

async function loadRoutes() {
  const scopePath = normalizedScopePath.value
  if (!scopePath) {
    return
  }
  loading.value = true
  try {
    channelDefinitions.value.forEach((definition) => resetRouteForm(definition.channel))
    const directResp = await listMessageNotificationRoutes(scopePath)
    ;(directResp.list || []).forEach(applyRouteInfo)

    const inheritedRows: MessageNotificationRouteInfo[] = []
    const ancestorPaths = getAncestorScopePaths(scopePath)
    for (const ancestorPath of ancestorPaths) {
      const resp = await listMessageNotificationRoutes(ancestorPath)
      inheritedRows.push(...(resp.list || []))
    }
    applyInheritedRoutes(inheritedRows)
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('notificationRoute.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function saveRoute(channel: ChannelCode, options: { silent?: boolean } = {}): Promise<boolean> {
  const scopePath = normalizedScopePath.value
  if (!scopePath) {
    ElMessage.warning(t('notificationRoute.noNode'))
    return false
  }
  const form = routeForms[channel]
  if (form.enabled && !hasWebhook(form)) {
    ElMessage.warning(t('userSettings.webhookRequiredBeforeEnable', { channel: channelLabel(channel) }))
    return false
  }
  const webhookURL = form.webhook_url.trim()
  if (webhookURL && !isValidWebhookURL(channel, webhookURL)) {
    ElMessage.warning(t('userSettings.webhookInvalid', { channel: channelLabel(channel) }))
    return false
  }

  savingRoute[channel] = true
  try {
    const info = await upsertMessageNotificationRoute(channel, {
      scope_path: scopePath,
      scope_type: resolvedScopeType.value,
      channel,
      enabled: form.enabled,
      delivery_type: 'webhook',
      display_name: form.display_name.trim() || channelDefaultDisplayName(channel),
      require_auth: form.require_auth,
      webhook_url: webhookURL,
      secret: form.secret.trim(),
      clear_webhook_url: form.clear_webhook_url,
      clear_secret: form.clear_secret,
      metadata: form.metadata
    })
    applyRouteInfo(info)
    if (!options.silent) {
      ElMessage.success(t('notificationRoute.saved', { channel: channelLabel(channel) }))
    }
    return true
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('notificationRoute.saveFailed'))
    return false
  } finally {
    savingRoute[channel] = false
  }
}

async function testRoute(channel: ChannelCode) {
  const saved = await saveRoute(channel, { silent: true })
  if (!saved) {
    return
  }

  testingRoute[channel] = true
  try {
    await testMessageNotificationRoute(channel, normalizedScopePath.value)
    if (routeForms[channel].enabled) {
      ElMessage.success(t('notificationRoute.testSent', { channel: channelLabel(channel) }))
    } else {
      ElMessage.warning(t('notificationRoute.testSentButDisabled', { channel: channelLabel(channel) }))
    }
    await loadRoutes()
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('notificationRoute.testFailed', { channel: channelLabel(channel) }))
  } finally {
    testingRoute[channel] = false
  }
}

async function deleteRoute(channel: ChannelCode) {
  try {
    await ElMessageBox.confirm(
      t('notificationRoute.deleteConfirm', { channel: channelLabel(channel) }),
      t('notificationRoute.deleteTitle'),
      { type: 'warning', confirmButtonText: t('common.delete'), cancelButtonText: t('common.cancel') }
    )
  } catch {
    return
  }

  deletingRoute[channel] = true
  try {
    await deleteMessageNotificationRoute(channel, normalizedScopePath.value)
    resetRouteForm(channel)
    ElMessage.success(t('notificationRoute.deleted', { channel: channelLabel(channel) }))
    await loadRoutes()
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('notificationRoute.deleteFailed'))
  } finally {
    deletingRoute[channel] = false
  }
}

function toggleClearWebhook(channel: ChannelCode) {
  const form = routeForms[channel]
  form.clear_webhook_url = !form.clear_webhook_url
  if (form.clear_webhook_url) {
    form.webhook_url = ''
  }
}

function toggleClearSecret(channel: ChannelCode) {
  const form = routeForms[channel]
  form.clear_secret = !form.clear_secret
  if (form.clear_secret) {
    form.secret = ''
  }
}

function hasWebhook(form: RouteFormState): boolean {
  if (form.clear_webhook_url) {
    return false
  }
  return Boolean(form.webhook_url.trim() || form.has_webhook_url)
}

function channelStatusLabel(form: RouteFormState): string {
  if (form.enabled && hasWebhook(form)) {
    return t('userSettings.statusEnabled')
  }
  if (hasWebhook(form)) {
    return t('userSettings.statusDisabled')
  }
  return t('userSettings.statusNotConfigured')
}

function channelStatusType(form: RouteFormState): 'success' | 'warning' | 'info' {
  if (form.enabled && hasWebhook(form)) {
    return 'success'
  }
  if (hasWebhook(form)) {
    return 'info'
  }
  return 'warning'
}

function deliveryStatusLabel(form: RouteFormState): string {
  if (form.last_error) {
    return form.fail_count > 0
      ? t('userSettings.deliveryFailedCount', { count: form.fail_count })
      : t('userSettings.deliveryFailed')
  }
  if (form.last_success_at) {
    return t('userSettings.deliveryHealthy')
  }
  if (form.last_test_at) {
    return t('userSettings.deliveryTested')
  }
  return t('userSettings.deliveryNotSent')
}

function deliveryStatusType(form: RouteFormState): 'success' | 'danger' | 'info' {
  if (form.last_error) {
    return 'danger'
  }
  if (form.last_success_at) {
    return 'success'
  }
  return 'info'
}

function webhookPlaceholder(channel: ChannelCode): string {
  if (routeForms[channel].has_webhook_url) {
    return t('userSettings.webhookConfiguredPlaceholder')
  }
  if (channel === 'feishu') {
    return 'https://open.feishu.cn/open-apis/bot/v2/hook/...'
  }
  if (channel === 'wecom') {
    return 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...'
  }
  return 'https://oapi.dingtalk.com/robot/send?access_token=...'
}

function secretPlaceholder(channel: ChannelCode): string {
  if (routeForms[channel].has_secret) {
    return t('userSettings.secretConfiguredPlaceholder')
  }
  if (channel === 'feishu') return t('userSettings.feishuSecretPlaceholder')
  if (channel === 'dingtalk') return t('userSettings.dingtalkSecretPlaceholder')
  return t('userSettings.wecomSecretPlaceholder')
}

function webhookStateText(form: RouteFormState): string {
  if (form.clear_webhook_url) {
    return t('userSettings.webhookWillClear')
  }
  if (form.has_webhook_url && !form.webhook_url.trim()) {
    return t('userSettings.webhookSavedHidden')
  }
  if (form.webhook_url.trim()) {
    return t('userSettings.webhookWillOverwrite')
  }
  return t('userSettings.webhookNotSaved')
}

function secretStateText(form: RouteFormState): string {
  if (form.clear_secret) {
    return t('userSettings.secretWillClear')
  }
  if (form.has_secret && !form.secret.trim()) {
    return t('userSettings.secretSavedHidden')
  }
  if (form.secret.trim()) {
    return t('userSettings.secretWillOverwrite')
  }
  return t('userSettings.secretNotSaved')
}

function isValidWebhookURL(channel: ChannelCode, raw: string): boolean {
  try {
    const url = new URL(raw)
    if (url.protocol !== 'https:') {
      return false
    }
    if (channel === 'feishu') {
      return ['open.feishu.cn', 'open.larksuite.com'].includes(url.hostname) && url.pathname.startsWith('/open-apis/bot/')
    }
    if (channel === 'wecom') {
      return url.hostname === 'qyapi.weixin.qq.com' && url.pathname === '/cgi-bin/webhook/send'
    }
    return url.hostname === 'oapi.dingtalk.com' && url.pathname === '/robot/send'
  } catch {
    return false
  }
}

function formatDateTime(value?: string): string {
  if (!value) {
    return ''
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

function normalizeScopePath(path: string | undefined): string {
  const value = String(path || '').trim()
  if (!value) {
    return ''
  }
  return `/${value.replace(/^\/+/, '').replace(/\/+$/, '')}`
}

function getAncestorScopePaths(path: string): string[] {
  const normalized = normalizeScopePath(path)
  if (!normalized) {
    return []
  }
  const parts = normalized.split('/').filter(Boolean)
  const ancestors: string[] = []
  for (let length = parts.length - 1; length >= 2; length -= 1) {
    ancestors.push(`/${parts.slice(0, length).join('/')}`)
  }
  return ancestors
}

function inferScopeType(path: string): string {
  const parts = normalizeScopePath(path).split('/').filter(Boolean)
  if (parts.length <= 2) {
    return 'workspace'
  }
  return 'directory'
}

watch(
  () => normalizedScopePath.value,
  () => {
    void loadRoutes()
  }
)

onMounted(() => {
  void loadRoutes()
})
</script>

<style scoped lang="scss">
.notification-route-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 320px;
}

.route-panel-header,
.channel-header,
.channel-title,
.channel-name-row,
.secret-row,
.channel-actions,
.delivery-status,
.auth-row,
.scope-summary {
  display: flex;
  align-items: center;
}

.route-panel-header {
  justify-content: space-between;
  gap: 16px;
}

.route-panel-header h3 {
  margin: 0 0 6px;
  color: var(--el-text-color-primary);
  font-size: 17px;
  font-weight: 600;
}

.route-panel-header p,
.channel-title p,
.form-tip,
.auth-row span {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.scope-summary {
  gap: 8px;
  min-width: 0;
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}

.scope-label {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.scope-summary code {
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 12px;
  overflow-wrap: anywhere;
}

.route-channel-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.route-channel-section {
  padding: 18px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  background: var(--el-bg-color);
}

.route-disabled-alert {
  margin-bottom: 14px;
}

.channel-header {
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.channel-title {
  min-width: 0;
  gap: 12px;
}

.channel-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  border-radius: 8px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-weight: 700;
}

.channel-name-row {
  gap: 8px;
  flex-wrap: wrap;
}

.channel-name-row h4 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 16px;
}

.channel-title .channel-guide {
  margin-top: 4px;
  color: var(--el-text-color-placeholder);
}

.route-form {
  max-width: 860px;
}

.secret-row {
  width: 100%;
  gap: 10px;
}

.secret-row .el-input {
  flex: 1;
  min-width: 0;
}

.form-tip {
  width: 100%;
  margin: 6px 0 0;
  font-size: 12px;
}

.auth-row {
  gap: 10px;
  flex-wrap: wrap;
}

.channel-actions {
  width: 100%;
  gap: 10px;
  flex-wrap: wrap;
}

.delivery-status {
  width: 100%;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.delivery-error {
  min-width: 0;
  max-width: 100%;
  color: var(--el-color-danger);
  overflow-wrap: anywhere;
}

.updated-at {
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

@media (max-width: 768px) {
  .route-panel-header,
  .channel-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .route-form {
    :deep(.el-form-item) {
      display: block;
    }

    :deep(.el-form-item__label) {
      justify-content: flex-start;
      width: auto !important;
      margin-bottom: 6px;
    }
  }

  .secret-row,
  .channel-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .updated-at {
    margin-left: 0;
  }
}
</style>
