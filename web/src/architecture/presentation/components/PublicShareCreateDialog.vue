<template>
  <el-dialog
    :model-value="modelValue"
    :title="createdShare ? t('publicSharePanel.createdDialogTitle') : t('publicSharePanel.createDialogTitle')"
    width="min(560px, calc(100vw - 24px))"
    :close-on-click-modal="false"
    class="public-share-dialog"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div v-if="createdShare" class="share-created-result">
      <div class="created-heading">
        <el-icon class="created-icon"><CircleCheckFilled /></el-icon>
        <div>
          <h3>{{ createdShare.title }}</h3>
          <p>{{ t('publicSharePanel.createdResultHint') }}</p>
        </div>
      </div>

      <div class="created-content">
        <div class="qr-box">
          <el-skeleton v-if="qrGenerating" :rows="5" animated />
          <img v-else-if="qrDataUrl" :src="qrDataUrl" class="qr-image" :alt="t('publicSharePanel.qrAlt')" />
          <el-empty v-else :description="t('publicSharePanel.qrGenerateFailed')" :image-size="72" />
        </div>

        <div class="created-details">
          <p v-if="createdShare.description" class="created-description">{{ createdShare.description }}</p>
          <div class="created-meta">
            <div>
              <span>{{ t('publicSharePanel.expirationTime') }}</span>
              <strong>{{ createdShare.expires_at ? formatDate(createdShare.expires_at) : t('publicSharePanel.permanent') }}</strong>
            </div>
            <div>
              <span>{{ t('publicSharePanel.submissionCount') }}</span>
              <strong>{{ createdShare.max_uses > 0 ? t('publicSharePanel.maxUses', { count: createdShare.max_uses }) : t('publicSharePanel.unlimited') }}</strong>
            </div>
          </div>
          <div class="created-link-label">{{ t('publicSharePanel.publicLink') }}</div>
          <button class="created-link" type="button" @click="copyLink">{{ publicLink }}</button>
          <p class="management-hint">{{ t('publicSharePanel.manageLaterHint') }}</p>
        </div>
      </div>
    </div>

    <el-form v-else label-position="top">
      <el-form-item :label="t('publicSharePanel.shareTitle')" required>
        <el-input
          v-model="form.title"
          maxlength="80"
          show-word-limit
          :placeholder="t('publicSharePanel.titlePlaceholder')"
        />
      </el-form-item>

      <el-form-item :label="t('publicSharePanel.description')">
        <el-input
          v-model="form.description"
          type="textarea"
          :rows="3"
          maxlength="300"
          show-word-limit
          :placeholder="t('publicSharePanel.descriptionPlaceholder')"
        />
      </el-form-item>

      <el-form-item :label="t('publicSharePanel.expirationTime')" required>
        <el-radio-group v-model="expireMode" class="expire-mode-group">
          <el-radio-button label="7d">{{ t('publicSharePanel.sevenDays') }}</el-radio-button>
          <el-radio-button label="30d">{{ t('publicSharePanel.thirtyDays') }}</el-radio-button>
          <el-radio-button label="custom">{{ t('publicSharePanel.custom') }}</el-radio-button>
          <el-radio-button label="never">{{ t('publicSharePanel.neverExpires') }}</el-radio-button>
        </el-radio-group>
        <el-date-picker
          v-if="expireMode === 'custom'"
          v-model="customExpiresAt"
          type="datetime"
          :disabled-date="isPastDate"
          :placeholder="t('publicSharePanel.expirationPlaceholder')"
          class="custom-expire-picker"
        />
        <div class="form-tip">{{ expirationHint }}</div>
      </el-form-item>

      <el-form-item :label="t('publicSharePanel.submissionCount')">
        <el-radio-group v-model="limitMode">
          <el-radio-button label="unlimited">{{ t('publicSharePanel.unlimited') }}</el-radio-button>
          <el-radio-button label="limited">{{ t('publicSharePanel.limited') }}</el-radio-button>
        </el-radio-group>
        <el-input-number
          v-if="limitMode === 'limited'"
          v-model="maxUses"
          :min="1"
          :step="10"
          controls-position="right"
          class="max-uses-input"
        />
      </el-form-item>

      <el-alert
        v-if="presetCount > 0"
        :title="t('publicSharePanel.presetValuesNotice', { count: presetCount })"
        type="info"
        :closable="false"
        show-icon
      />
    </el-form>

    <template #footer>
      <div v-if="createdShare" class="created-footer">
        <el-button @click="openLink">{{ t('publicSharePanel.openLink') }}</el-button>
        <el-button :disabled="!qrDataUrl" @click="downloadQrCode">{{ t('publicSharePanel.downloadQr') }}</el-button>
        <el-button type="primary" @click="copyLink">{{ t('publicSharePanel.copyLink') }}</el-button>
        <el-button @click="emit('update:modelValue', false)">{{ t('common.close') }}</el-button>
      </div>
      <template v-else>
        <el-button @click="emit('update:modelValue', false)">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="creating" @click="submit">
          {{ t('publicSharePanel.createAndGenerateQr') }}
        </el-button>
      </template>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { CircleCheckFilled } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'
import {
  createPublicShare,
  type PublicShareItem,
} from '@/architecture/presentation/context/api/publicShare'
import { getErrorMessage } from '@/architecture/shared/apiError'

const props = withDefaults(defineProps<{
  modelValue: boolean
  fullCodePath: string
  defaultTitle?: string
  defaultDescription?: string
  presetValues?: Record<string, unknown>
}>(), {
  defaultTitle: '',
  defaultDescription: '',
  presetValues: () => ({}),
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'created', share: PublicShareItem): void
}>()

const { t } = useI18n()
const creating = ref(false)
const expireMode = ref<'7d' | '30d' | 'custom' | 'never'>('30d')
const limitMode = ref<'unlimited' | 'limited'>('unlimited')
const customExpiresAt = ref<Date | null>(null)
const maxUses = ref(100)
const form = reactive({ title: '', description: '' })
const createdShare = ref<PublicShareItem | null>(null)
const qrDataUrl = ref('')
const qrGenerating = ref(false)

const presetCount = computed(() => Object.keys(props.presetValues || {}).length)
const expirationHint = computed(() => {
  if (expireMode.value === 'never') return t('publicSharePanel.neverExpiresHint')
  if (expireMode.value === 'custom') return t('publicSharePanel.customExpirationHint')
  const count = expireMode.value === '7d' ? 7 : 30
  return t('publicSharePanel.expiresInDays', { count })
})
const publicLink = computed(() => {
  if (!createdShare.value) return ''
  return createdShare.value.public_url
})

watch(() => props.modelValue, (visible) => {
  if (!visible) return
  form.title = props.defaultTitle.trim()
  form.description = props.defaultDescription.trim()
  expireMode.value = '30d'
  limitMode.value = 'unlimited'
  customExpiresAt.value = null
  maxUses.value = 100
  createdShare.value = null
  qrDataUrl.value = ''
  qrGenerating.value = false
})

function isPastDate(date: Date): boolean {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return date.getTime() < today.getTime()
}

function resolvedExpiresAt(): string | null {
  if (expireMode.value === 'never') return null
  if (expireMode.value === 'custom') return customExpiresAt.value?.toISOString() || null
  const expiresAt = new Date()
  expiresAt.setDate(expiresAt.getDate() + (expireMode.value === '7d' ? 7 : 30))
  return expiresAt.toISOString()
}

async function submit(): Promise<void> {
  if (!props.fullCodePath) {
    ElMessage.warning(t('publicSharePanel.pathNotReady'))
    return
  }
  if (!form.title.trim()) {
    ElMessage.warning(t('publicSharePanel.titleRequired'))
    return
  }
  const expiresAt = resolvedExpiresAt()
  if (expireMode.value === 'custom' && !expiresAt) {
    ElMessage.warning(t('publicSharePanel.expirationRequired'))
    return
  }
  if (expiresAt && new Date(expiresAt).getTime() <= Date.now()) {
    ElMessage.warning(t('publicSharePanel.expirationFutureRequired'))
    return
  }

  creating.value = true
  try {
    const share = await createPublicShare({
      full_code_path: props.fullCodePath,
      title: form.title.trim(),
      description: form.description.trim(),
      expires_at: expiresAt,
      max_uses: limitMode.value === 'limited' ? maxUses.value : 0,
      preset_values: props.presetValues,
    })
    createdShare.value = share
    emit('created', share)
    await generateQrCode()
  } catch (error) {
    ElMessage.error(getErrorMessage(error, t('publicSharePanel.createFailed')))
  } finally {
    creating.value = false
  }
}

async function generateQrCode(): Promise<void> {
  if (!publicLink.value) return
  qrGenerating.value = true
  qrDataUrl.value = ''
  try {
    qrDataUrl.value = await QRCode.toDataURL(publicLink.value, {
      errorCorrectionLevel: 'M',
      margin: 2,
      width: 256,
      color: { dark: '#111827', light: '#ffffff' },
    })
  } catch {
    ElMessage.error(t('publicSharePanel.qrGenerateFailed'))
  } finally {
    qrGenerating.value = false
  }
}

async function copyLink(): Promise<void> {
  if (!publicLink.value) return
  await navigator.clipboard.writeText(publicLink.value)
  ElMessage.success(t('publicSharePanel.linkCopied'))
}

function openLink(): void {
  if (publicLink.value) window.open(publicLink.value, '_blank', 'noopener,noreferrer')
}

function downloadQrCode(): void {
  if (!qrDataUrl.value || !createdShare.value) return
  const link = document.createElement('a')
  link.href = qrDataUrl.value
  link.download = `${safeFileName(createdShare.value.title || createdShare.value.share_id)}-qrcode.png`
  link.click()
}

function safeFileName(value: string): string {
  return value.trim().replace(/[\\/:*?"<>|]+/g, '-') || 'public-share'
}

function formatDate(value: string): string {
  return new Date(value).toLocaleString()
}
</script>

<style scoped>
.expire-mode-group {
  display: flex;
  max-width: 100%;
  flex-wrap: wrap;
}

.custom-expire-picker,
.max-uses-input {
  width: 100%;
  margin-top: 12px;
}

.form-tip {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.share-created-result {
  display: grid;
  gap: 20px;
}

.created-heading {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.created-icon {
  flex: 0 0 auto;
  margin-top: 1px;
  color: var(--el-color-success);
  font-size: 28px;
}

.created-heading h3,
.created-heading p,
.created-description,
.management-hint {
  margin: 0;
}

.created-heading h3 {
  color: var(--el-text-color-primary);
  font-size: 18px;
  line-height: 1.4;
}

.created-heading p,
.management-hint {
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.55;
}

.created-content {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  align-items: center;
  gap: 20px;
}

.qr-box {
  display: grid;
  place-items: center;
  min-height: 220px;
  padding: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  background: #fff;
}

.qr-image {
  display: block;
  width: 200px;
  height: 200px;
}

.created-details {
  min-width: 0;
}

.created-description {
  margin-bottom: 14px;
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.created-meta {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 14px;
}

.created-meta > div {
  min-width: 0;
  padding: 10px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
}

.created-meta span,
.created-meta strong {
  display: block;
  overflow-wrap: anywhere;
}

.created-meta span,
.created-link-label {
  color: var(--el-text-color-secondary);
  font-size: 11px;
  line-height: 1.4;
}

.created-meta strong {
  margin-top: 4px;
  color: var(--el-text-color-primary);
  font-size: 12px;
  line-height: 1.45;
}

.created-link {
  display: block;
  width: 100%;
  margin-top: 5px;
  padding: 9px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--el-color-primary);
  font: inherit;
  font-size: 12px;
  line-height: 1.45;
  text-align: left;
  word-break: break-all;
  cursor: pointer;
}

.created-footer {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
}

.created-footer :deep(.el-button + .el-button) {
  margin-left: 0;
}

@media (max-width: 520px) {
  .expire-mode-group :deep(.el-radio-button) {
    flex: 1 0 50%;
  }

  .expire-mode-group :deep(.el-radio-button__inner) {
    width: 100%;
  }

  .share-created-result {
    gap: 16px;
  }

  .created-content {
    grid-template-columns: 1fr;
    gap: 14px;
  }

  .qr-box {
    width: min(236px, 100%);
    min-height: 236px;
    margin: 0 auto;
  }

  .qr-image {
    width: min(216px, 100%);
    height: auto;
  }

  .created-footer {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .created-footer :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }
}
</style>
