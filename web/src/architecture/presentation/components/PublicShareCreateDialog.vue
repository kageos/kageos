<template>
  <el-dialog
    :model-value="modelValue"
    :title="t('publicSharePanel.createDialogTitle')"
    width="min(520px, calc(100vw - 32px))"
    :close-on-click-modal="false"
    class="public-share-dialog"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-form label-position="top">
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
      <el-button @click="emit('update:modelValue', false)">{{ t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="creating" @click="submit">
        {{ t('publicSharePanel.createAndGenerateQr') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
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

const presetCount = computed(() => Object.keys(props.presetValues || {}).length)
const expirationHint = computed(() => {
  if (expireMode.value === 'never') return t('publicSharePanel.neverExpiresHint')
  if (expireMode.value === 'custom') return t('publicSharePanel.customExpirationHint')
  const count = expireMode.value === '7d' ? 7 : 30
  return t('publicSharePanel.expiresInDays', { count })
})

watch(() => props.modelValue, (visible) => {
  if (!visible) return
  form.title = props.defaultTitle.trim()
  form.description = props.defaultDescription.trim()
  expireMode.value = '30d'
  limitMode.value = 'unlimited'
  customExpiresAt.value = null
  maxUses.value = 100
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
    emit('update:modelValue', false)
    emit('created', share)
  } catch (error) {
    ElMessage.error(getErrorMessage(error, t('publicSharePanel.createFailed')))
  } finally {
    creating.value = false
  }
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

@media (max-width: 520px) {
  .expire-mode-group :deep(.el-radio-button) {
    flex: 1 0 50%;
  }

  .expire-mode-group :deep(.el-radio-button__inner) {
    width: 100%;
  }
}
</style>
