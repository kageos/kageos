<template>
  <div class="user-settings">
    <div class="settings-container">
      <el-card shadow="hover" class="settings-card">
        <template #header>
          <div class="card-header">
            <div class="header-title">
              <el-button
                link
                :icon="ArrowLeft"
                class="back-button"
                @click="handleBack"
              >
                {{ t('userSettings.back') }}
              </el-button>
              <h2>{{ t('userSettings.title') }}</h2>
            </div>
            <el-button :icon="Refresh" :loading="notificationsLoading" @click="loadNotificationChannels">
              {{ t('common.refresh') }}
            </el-button>
          </div>
        </template>

        <div class="settings-sections">
          <section class="settings-section">
            <div class="section-heading">
              <div>
                <h3>{{ t('userSettings.profileTitle') }}</h3>
                <p>{{ t('userSettings.profileSubtitle') }}</p>
              </div>
              <el-tag v-if="currentUser?.username" type="info" effect="plain">
                {{ currentUser.username }}
              </el-tag>
            </div>

            <el-form
              ref="formRef"
              :model="formData"
              :rules="rules"
              label-width="100px"
              class="settings-form"
            >
              <el-form-item :label="t('userSettings.avatar')">
                <div class="avatar-section">
                  <CommonUpload
                    v-model="formData.avatar"
                    :router="avatarRouter"
                    accept="image/*"
                    max-size="2MB"
                    @change="handleAvatarChange"
                  />
                  <p class="form-tip">{{ t('userSettings.avatarTip') }}</p>
                </div>
              </el-form-item>

              <el-form-item :label="t('userSettings.username')">
                <el-input
                  :value="currentUser?.username"
                  disabled
                  class="disabled-input"
                />
                <p class="form-tip">{{ t('userSettings.usernameReadonly') }}</p>
              </el-form-item>

              <el-form-item :label="t('userSettings.email')">
                <el-input
                  :value="currentUser?.email"
                  disabled
                  class="disabled-input"
                />
                <p class="form-tip">{{ t('userSettings.emailReadonly') }}</p>
              </el-form-item>

              <el-form-item :label="t('userSettings.nickname')" prop="nickname">
                <el-input
                  v-model="formData.nickname"
                  :placeholder="t('userSettings.nicknamePlaceholder')"
                  maxlength="50"
                  show-word-limit
                  clearable
                />
              </el-form-item>

              <el-form-item :label="t('userSettings.signature')" prop="signature">
                <el-input
                  v-model="formData.signature"
                  type="textarea"
                  :rows="4"
                  :placeholder="t('userSettings.signaturePlaceholder')"
                  maxlength="200"
                  show-word-limit
                />
              </el-form-item>

              <el-form-item :label="t('userSettings.gender')" prop="gender">
                <el-radio-group v-model="formData.gender">
                  <el-radio label="">{{ t('userSettings.genderUnset') }}</el-radio>
                  <el-radio label="male">{{ t('userSettings.genderMale') }}</el-radio>
                  <el-radio label="female">{{ t('userSettings.genderFemale') }}</el-radio>
                  <el-radio label="other">{{ t('userSettings.genderOther') }}</el-radio>
                </el-radio-group>
              </el-form-item>

              <el-form-item>
                <el-button
                  type="primary"
                  :icon="Check"
                  :loading="submitting"
                  @click="handleSubmit"
                >
                  {{ t('userSettings.save') }}
                </el-button>
                <el-button @click="handleReset">{{ t('common.reset') }}</el-button>
              </el-form-item>
            </el-form>
          </section>

          <el-divider />

          <section class="settings-section">
            <div class="section-heading">
              <div>
                <h3>{{ t('userSettings.notificationsTitle') }}</h3>
                <p>{{ t('userSettings.notificationsSubtitle') }}</p>
              </div>
            </div>

            <div v-loading="notificationsLoading" class="notification-channel-list">
              <div
                v-for="definition in channelDefinitions"
                :key="definition.channel"
                class="notification-channel"
              >
                <div class="channel-header">
                  <div class="channel-title">
                    <span class="channel-mark">{{ definition.mark }}</span>
                    <div>
                      <div class="channel-name-row">
                        <h4>{{ definition.name }}</h4>
                        <el-tag size="small" :type="channelStatusType(notificationForms[definition.channel])">
                          {{ channelStatusLabel(notificationForms[definition.channel]) }}
                        </el-tag>
                      </div>
                      <p>{{ definition.hint }}</p>
                      <p class="channel-guide">{{ definition.guide }}</p>
                    </div>
                  </div>
                  <el-switch
                    v-model="notificationForms[definition.channel].enabled"
                    inline-prompt
                    :active-text="t('userSettings.enabled')"
                    :inactive-text="t('userSettings.disabled')"
                  />
                </div>

                <el-form
                  :model="notificationForms[definition.channel]"
                  label-width="110px"
                  class="channel-form"
                >
                  <el-form-item :label="t('userSettings.displayName')">
                    <el-input
                      v-model="notificationForms[definition.channel].display_name"
                      :placeholder="definition.name"
                      clearable
                    />
                  </el-form-item>

                  <el-form-item label="Webhook">
                    <div class="secret-row">
                      <el-input
                        v-model="notificationForms[definition.channel].webhook_url"
                        :disabled="notificationForms[definition.channel].clear_webhook_url"
                        :placeholder="webhookPlaceholder(definition.channel)"
                        clearable
                      />
                      <el-button
                        v-if="notificationForms[definition.channel].has_webhook_url || notificationForms[definition.channel].clear_webhook_url"
                        :type="notificationForms[definition.channel].clear_webhook_url ? 'primary' : 'info'"
                        plain
                        @click="toggleClearWebhook(definition.channel)"
                      >
                      {{ notificationForms[definition.channel].clear_webhook_url ? t('userSettings.keepWebhook') : t('userSettings.clearWebhook') }}
                      </el-button>
                    </div>
                    <p class="form-tip">
                      {{ webhookStateText(notificationForms[definition.channel]) }}
                    </p>
                  </el-form-item>

                  <el-form-item :label="t('userSettings.signingSecret')">
                    <div class="secret-row">
                      <el-input
                        v-model="notificationForms[definition.channel].secret"
                        type="password"
                        show-password
                        :disabled="notificationForms[definition.channel].clear_secret"
                        :placeholder="secretPlaceholder(definition.channel)"
                        clearable
                      />
                      <el-button
                        v-if="notificationForms[definition.channel].has_secret || notificationForms[definition.channel].clear_secret"
                        :type="notificationForms[definition.channel].clear_secret ? 'primary' : 'info'"
                        plain
                        @click="toggleClearSecret(definition.channel)"
                      >
                        {{ notificationForms[definition.channel].clear_secret ? t('userSettings.keepSecret') : t('userSettings.clearSecret') }}
                      </el-button>
                    </div>
                    <p class="form-tip">
                      {{ secretStateText(notificationForms[definition.channel]) }}
                    </p>
                  </el-form-item>

                  <el-form-item>
                    <div class="channel-actions">
                      <el-button
                        type="primary"
                        :icon="Check"
                        :loading="savingNotification[definition.channel]"
                        @click="saveNotificationChannel(definition.channel)"
                      >
                        {{ t('userSettings.saveConfig') }}
                      </el-button>
                      <el-button
                        :icon="Promotion"
                        :loading="testingNotification[definition.channel]"
                        :disabled="deletingNotification[definition.channel]"
                        @click="testNotificationChannelConfig(definition.channel)"
                      >
                        {{ t('userSettings.testSend') }}
                      </el-button>
                      <el-button
                        text
                        type="danger"
                        :icon="Delete"
                        :loading="deletingNotification[definition.channel]"
                        @click="deleteNotificationChannelConfig(definition.channel)"
                      >
                        {{ t('userSettings.deleteConfig') }}
                      </el-button>
                      <span v-if="notificationForms[definition.channel].updated_at" class="updated-at">
                        {{ t('userSettings.updatedAt', { time: formatDateTime(notificationForms[definition.channel].updated_at) }) }}
                      </span>
                    </div>
                    <div class="delivery-status">
                      <el-tag size="small" :type="deliveryStatusType(notificationForms[definition.channel])">
                        {{ deliveryStatusLabel(notificationForms[definition.channel]) }}
                      </el-tag>
                      <span v-if="notificationForms[definition.channel].last_success_at">
                        {{ t('userSettings.lastSuccessAt', { time: formatDateTime(notificationForms[definition.channel].last_success_at) }) }}
                      </span>
                      <span v-if="notificationForms[definition.channel].last_test_at">
                        {{ t('userSettings.lastTestAt', { time: formatDateTime(notificationForms[definition.channel].last_test_at) }) }}
                      </span>
                      <span v-if="notificationForms[definition.channel].last_error" class="delivery-error">
                        {{ notificationForms[definition.channel].last_error }}
                      </span>
                    </div>
                  </el-form-item>
                </el-form>
              </div>
            </div>
          </section>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { ArrowLeft, Check, Delete, Promotion, Refresh } from '@element-plus/icons-vue'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import CommonUpload from '@/architecture/presentation/shared/components/CommonUpload.vue'
import {
  deleteMessageNotificationChannel,
  listMessageNotificationChannels,
  testMessageNotificationChannel,
  upsertMessageNotificationChannel,
  type MessageNotificationChannelInfo
} from '@/architecture/presentation/context/api/message'

type ChannelCode = 'feishu' | 'wecom' | 'dingtalk'

interface ChannelDefinition {
  channel: ChannelCode
  name: string
  mark: string
  hint: string
  guide: string
}

interface NotificationFormState {
  channel: ChannelCode
  enabled: boolean
  display_name: string
  webhook_url: string
  secret: string
  has_webhook_url: boolean
  has_secret: boolean
  clear_webhook_url: boolean
  clear_secret: boolean
  metadata: Record<string, string>
  updated_at?: string
  last_success_at?: string
  last_failed_at?: string
  last_test_at?: string
  last_error?: string
  fail_count: number
}

const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()

const formRef = ref<FormInstance>()
const submitting = ref(false)
const notificationsLoading = ref(false)

const currentUser = computed(() => authStore.user)

const avatarRouter = computed(() => {
  const username = currentUser.value?.username || 'default'
  return `${username}/avatar`
})

const formData = reactive({
  avatar: '',
  nickname: '',
  signature: '',
  gender: '' as '' | 'male' | 'female' | 'other'
})

const rules = computed<FormRules>(() => ({
  nickname: [
    { max: 50, message: t('userSettings.nicknameMax'), trigger: 'blur' }
  ],
  signature: [
    { max: 200, message: t('userSettings.signatureMax'), trigger: 'blur' }
  ]
}))

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

const notificationForms = reactive<Record<ChannelCode, NotificationFormState>>({
  feishu: createDefaultNotificationForm('feishu'),
  wecom: createDefaultNotificationForm('wecom'),
  dingtalk: createDefaultNotificationForm('dingtalk')
})

const savingNotification = reactive<Record<ChannelCode, boolean>>({
  feishu: false,
  wecom: false,
  dingtalk: false
})

const testingNotification = reactive<Record<ChannelCode, boolean>>({
  feishu: false,
  wecom: false,
  dingtalk: false
})

const deletingNotification = reactive<Record<ChannelCode, boolean>>({
  feishu: false,
  wecom: false,
  dingtalk: false
})

function createDefaultNotificationForm(channel: ChannelCode): NotificationFormState {
  return {
    channel,
    enabled: false,
    display_name: channelDefaultDisplayName(channel),
    webhook_url: '',
    secret: '',
    has_webhook_url: false,
    has_secret: false,
    clear_webhook_url: false,
    clear_secret: false,
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

function initFormData() {
  if (currentUser.value) {
    formData.avatar = currentUser.value.avatar || ''
    formData.nickname = currentUser.value.nickname || ''
    formData.signature = currentUser.value.signature || ''
    formData.gender = (currentUser.value.gender || '') as '' | 'male' | 'female' | 'other'
  }
}

function handleAvatarChange(url: string | null) {
  if (url) {
    formData.avatar = url
  }
}

async function handleSubmit() {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    submitting.value = true

    const updateData: {
      avatar?: string
      nickname?: string
      signature?: string
      gender?: '' | 'male' | 'female' | 'other'
    } = {}

    if (formData.avatar !== (currentUser.value?.avatar || '')) {
      updateData.avatar = formData.avatar
    }
    if (formData.nickname !== (currentUser.value?.nickname || '')) {
      updateData.nickname = formData.nickname
    }
    if (formData.signature !== (currentUser.value?.signature || '')) {
      updateData.signature = formData.signature
    }
    if (formData.gender !== (currentUser.value?.gender || '')) {
      updateData.gender = formData.gender
    }

    if (Object.keys(updateData).length === 0) {
      ElMessage.info(t('userSettings.noChanges'))
      return
    }

    await authStore.updateUser(updateData)
    ElMessage.success(t('common.updateSuccess'))
  } catch (error: any) {
    console.error('更新用户信息失败:', error)
    if (error?.message && !error.message.includes('validate')) {
      ElMessage.error(error.message || t('common.updateFailed'))
    }
  } finally {
    submitting.value = false
  }
}

function handleReset() {
  initFormData()
  formRef.value?.clearValidate()
}

function handleBack() {
  router.go(-1)
}

function resetNotificationForm(channel: ChannelCode) {
  Object.assign(notificationForms[channel], createDefaultNotificationForm(channel))
}

function applyNotificationInfo(info: MessageNotificationChannelInfo) {
  const channel = normalizeChannel(info.channel)
  if (!channel) {
    return
  }
  const form = notificationForms[channel]
  form.enabled = Boolean(info.enabled)
  form.display_name = info.display_name || channelDefaultDisplayName(channel)
  form.webhook_url = ''
  form.secret = ''
  form.has_webhook_url = Boolean(info.has_webhook_url)
  form.has_secret = Boolean(info.has_secret)
  form.clear_webhook_url = false
  form.clear_secret = false
  form.metadata = info.metadata || {}
  form.updated_at = info.updated_at
  form.last_success_at = info.last_success_at
  form.last_failed_at = info.last_failed_at
  form.last_test_at = info.last_test_at
  form.last_error = info.last_error || ''
  form.fail_count = info.fail_count || 0
}

function normalizeChannel(channel: string): ChannelCode | null {
  if (channel === 'feishu' || channel === 'wecom' || channel === 'dingtalk') {
    return channel
  }
  return null
}

async function loadNotificationChannels() {
  notificationsLoading.value = true
  try {
    channelDefinitions.value.forEach((definition) => resetNotificationForm(definition.channel))
    const resp = await listMessageNotificationChannels()
    const list = resp.list || []
    list.forEach(applyNotificationInfo)
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('userSettings.loadNotificationsFailed'))
  } finally {
    notificationsLoading.value = false
  }
}

async function saveNotificationChannel(channel: ChannelCode, options: { silent?: boolean } = {}): Promise<boolean> {
  const form = notificationForms[channel]
  if (form.enabled && !hasWebhook(form)) {
    ElMessage.warning(t('userSettings.webhookRequiredBeforeEnable', { channel: channelLabel(channel) }))
    return false
  }
  const webhookURL = form.webhook_url.trim()
  if (webhookURL && !isValidWebhookURL(channel, webhookURL)) {
    ElMessage.warning(t('userSettings.webhookInvalid', { channel: channelLabel(channel) }))
    return false
  }

  savingNotification[channel] = true
  try {
    const info = await upsertMessageNotificationChannel(channel, {
      channel,
      enabled: form.enabled,
      delivery_type: 'webhook',
      display_name: form.display_name.trim() || channelDefaultDisplayName(channel),
      webhook_url: webhookURL,
      secret: form.secret.trim(),
      clear_webhook_url: form.clear_webhook_url,
      clear_secret: form.clear_secret,
      metadata: form.metadata
    })
    applyNotificationInfo(info)
    if (!options.silent) {
      ElMessage.success(t('userSettings.configSaved', { channel: channelLabel(channel) }))
    }
    return true
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('userSettings.configSaveFailed', { channel: channelLabel(channel) }))
    return false
  } finally {
    savingNotification[channel] = false
  }
}

async function testNotificationChannelConfig(channel: ChannelCode) {
  const saved = await saveNotificationChannel(channel, { silent: true })
  if (!saved) {
    return
  }

  testingNotification[channel] = true
  try {
    await testMessageNotificationChannel(channel)
    ElMessage.success(t('userSettings.testSent', { channel: channelLabel(channel) }))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('userSettings.testFailed', { channel: channelLabel(channel) }))
  } finally {
    testingNotification[channel] = false
  }
}

async function deleteNotificationChannelConfig(channel: ChannelCode) {
  try {
    await ElMessageBox.confirm(
      t('userSettings.deleteConfirm', { channel: channelLabel(channel) }),
      t('userSettings.deleteConfirmTitle'),
      { type: 'warning', confirmButtonText: t('common.delete'), cancelButtonText: t('common.cancel') }
    )
  } catch {
    return
  }

  deletingNotification[channel] = true
  try {
    await deleteMessageNotificationChannel(channel)
    resetNotificationForm(channel)
    ElMessage.success(t('userSettings.configDeleted', { channel: channelLabel(channel) }))
  } catch (error: any) {
    ElMessage.error(error?.response?.data?.msg || error?.message || t('userSettings.configDeleteFailed', { channel: channelLabel(channel) }))
  } finally {
    deletingNotification[channel] = false
  }
}

function toggleClearWebhook(channel: ChannelCode) {
  const form = notificationForms[channel]
  form.clear_webhook_url = !form.clear_webhook_url
  if (form.clear_webhook_url) {
    form.webhook_url = ''
  }
}

function toggleClearSecret(channel: ChannelCode) {
  const form = notificationForms[channel]
  form.clear_secret = !form.clear_secret
  if (form.clear_secret) {
    form.secret = ''
  }
}

function hasWebhook(form: NotificationFormState): boolean {
  if (form.clear_webhook_url) {
    return false
  }
  return Boolean(form.webhook_url.trim() || form.has_webhook_url)
}

function channelStatusLabel(form: NotificationFormState): string {
  if (form.enabled && hasWebhook(form)) {
    return t('userSettings.statusEnabled')
  }
  if (hasWebhook(form)) {
    return t('userSettings.statusDisabled')
  }
  return t('userSettings.statusNotConfigured')
}

function channelStatusType(form: NotificationFormState): 'success' | 'warning' | 'info' {
  if (form.enabled && hasWebhook(form)) {
    return 'success'
  }
  if (hasWebhook(form)) {
    return 'info'
  }
  return 'warning'
}

function deliveryStatusLabel(form: NotificationFormState): string {
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

function deliveryStatusType(form: NotificationFormState): 'success' | 'danger' | 'info' {
  if (form.last_error) {
    return 'danger'
  }
  if (form.last_success_at) {
    return 'success'
  }
  return 'info'
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

function webhookPlaceholder(channel: ChannelCode): string {
  if (notificationForms[channel].has_webhook_url) {
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
  if (notificationForms[channel].has_secret) {
    return t('userSettings.secretConfiguredPlaceholder')
  }
  if (channel === 'feishu') return t('userSettings.feishuSecretPlaceholder')
  if (channel === 'dingtalk') return t('userSettings.dingtalkSecretPlaceholder')
  return t('userSettings.wecomSecretPlaceholder')
}

function webhookStateText(form: NotificationFormState): string {
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

function secretStateText(form: NotificationFormState): string {
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

onMounted(async () => {
  if (!authStore.isAuthenticated) {
    router.push('/login')
    return
  }

  try {
    await authStore.fetchUserInfo()
  } catch (error) {
    console.error('获取用户信息失败:', error)
  }

  initFormData()
  await loadNotificationChannels()
})
</script>

<style scoped>
.user-settings {
  min-height: 100vh;
  background: var(--bg-primary);
  padding: 32px 40px;
}

.settings-container {
  max-width: 1280px;
  margin: 0 auto;
}

.settings-card {
  margin-top: 20px;
  border-radius: var(--border-radius-xl);
  border: 1px solid var(--border-light);
  box-shadow: var(--app-shell-panel-shadow-soft);
  background: var(--bg-secondary);
}

.card-header,
.header-title,
.section-heading,
.channel-header,
.channel-title,
.channel-name-row,
.secret-row,
.channel-actions {
  display: flex;
  align-items: center;
}

.card-header {
  justify-content: space-between;
  gap: 16px;
}

.header-title {
  min-width: 0;
  gap: 12px;
}

.card-header .back-button {
  margin-left: -8px;
  padding: 4px 8px;
}

.card-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.settings-sections {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.settings-section {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.section-heading {
  justify-content: space-between;
  gap: 16px;
}

.section-heading h3 {
  margin: 0 0 6px;
  font-size: 17px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.section-heading p,
.channel-title p {
  margin: 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.channel-title .channel-guide {
  margin-top: 4px;
  color: var(--el-text-color-placeholder);
}

.settings-form {
  max-width: 760px;
}

.avatar-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.form-tip {
  width: 100%;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin: 6px 0 0;
  line-height: 1.5;
}

.disabled-input {
  opacity: 0.6;
}

.notification-channel-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.notification-channel {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 18px;
  background: var(--el-bg-color);
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
  font-size: 16px;
  color: var(--el-text-color-primary);
}

.channel-form {
  max-width: 840px;
}

.secret-row {
  width: 100%;
  gap: 10px;
}

.secret-row .el-input {
  flex: 1;
  min-width: 0;
}

.channel-actions {
  width: 100%;
  gap: 10px;
  flex-wrap: wrap;
}

.delivery-status {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  width: 100%;
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

@media (max-width: 720px) {
  .user-settings {
    padding: 12px;
  }

  .settings-container {
    max-width: none;
  }

  .card-header,
  .section-heading,
  .channel-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .secret-row {
    align-items: stretch;
    flex-direction: column;
  }

  .updated-at {
    width: 100%;
    margin-left: 0;
  }
}
</style>
