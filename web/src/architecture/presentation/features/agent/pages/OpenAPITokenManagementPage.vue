<template>
  <div class="openapi-token-page" :class="{ 'is-embedded': embedded }">
    <component :is="embedded ? 'div' : ElCard" :shadow="embedded ? undefined : 'hover'" class="page-card">
      <template v-if="!embedded" #header>
        <div class="card-header">
          <div>
            <h2>{{ t('openapiToken.title') }}</h2>
            <p class="header-desc">{{ t('openapiToken.subtitle') }}</p>
          </div>
          <div class="header-actions">
            <el-button :icon="Refresh" :loading="tokensLoading" @click="loadOpenAPITokens">
              {{ t('common.refresh') }}
            </el-button>
            <el-button type="primary" :icon="Plus" @click="createTokenDialogVisible = true">
              {{ t('openapiToken.create') }}
            </el-button>
          </div>
        </div>
      </template>

      <div class="page-body">
        <div v-if="embedded" class="embedded-actions">
          <div class="toolbar-summary">
            {{ tokenSummary }}
          </div>
          <div class="header-actions">
            <el-button :icon="Refresh" :loading="tokensLoading" @click="loadOpenAPITokens">
              {{ t('common.refresh') }}
            </el-button>
            <el-button type="primary" :icon="Plus" @click="createTokenDialogVisible = true">
              {{ t('openapiToken.create') }}
            </el-button>
          </div>
        </div>
        <div v-if="!embedded" class="toolbar">
          <div class="toolbar-summary">
            {{ tokenSummary }}
          </div>
        </div>

        <el-table
          :data="openapiTokens"
          v-loading="tokensLoading"
          stripe
          style="width: 100%"
          :empty-text="t('openapiToken.empty')"
        >
          <el-table-column prop="name" :label="t('openapiToken.name')" min-width="180" />
          <el-table-column prop="token_prefix" :label="t('openapiToken.prefix')" width="150" />
          <el-table-column :label="t('openapiToken.status')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.revoked_at ? 'danger' : 'success'">
                {{ row.revoked_at ? t('openapiToken.revoked') : t('openapiToken.available') }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="last_used_at" :label="t('openapiToken.lastUsedAt')" min-width="170">
            <template #default="{ row }">
              {{ row.last_used_at || t('openapiToken.neverUsed') }}
            </template>
          </el-table-column>
          <el-table-column prop="expires_at" :label="t('openapiToken.expiresAt')" min-width="170">
            <template #default="{ row }">
              {{ row.expires_at || t('openapiToken.neverExpires') }}
            </template>
          </el-table-column>
          <el-table-column prop="created_at" :label="t('openapiToken.createdAt')" min-width="170" />
          <el-table-column :label="t('common.operation')" width="110" fixed="right">
            <template #default="{ row }">
              <el-button
                link
                type="danger"
                :disabled="!!row.revoked_at"
                @click="handleRevokeToken(row.id)"
              >
                {{ t('openapiToken.revoke') }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </component>

    <el-dialog v-model="createTokenDialogVisible" :title="t('openapiToken.createTitle')" width="520px">
      <el-form label-width="90px">
        <el-form-item :label="t('openapiToken.name')">
          <el-input v-model="newTokenName" placeholder="kageos-hub-production" />
        </el-form-item>
        <el-form-item :label="t('openapiToken.expiresAt')">
          <el-date-picker
            v-model="newTokenExpiresAt"
            type="datetime"
            :placeholder="t('openapiToken.expiresAtPlaceholder')"
            style="width: 100%"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createTokenDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="creatingToken" @click="handleCreateToken">
          {{ t('openapiToken.create') }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="createdTokenVisible" :title="t('openapiToken.secretOnceTitle')" width="640px">
      <p class="form-tip">{{ t('openapiToken.secretOnceTip') }}</p>
      <el-input
        v-model="createdSecretToken"
        readonly
        type="textarea"
        :rows="3"
        class="token-secret"
      />
      <template #footer>
        <el-button @click="copyCreatedToken">{{ t('connectorProvider.copy') }}</el-button>
        <el-button type="primary" @click="createdTokenVisible = false">{{ t('openapiToken.done') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElCard, ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh } from '@element-plus/icons-vue'
import {
  createOpenAPIToken,
  listOpenAPITokens,
  revokeOpenAPIToken,
  type OpenAPITokenInfo,
} from '@/architecture/presentation/context/api/user'

withDefaults(defineProps<{
  embedded?: boolean
}>(), {
  embedded: false
})

const { t } = useI18n()
const tokensLoading = ref(false)
const creatingToken = ref(false)
const createTokenDialogVisible = ref(false)
const createdTokenVisible = ref(false)
const newTokenName = ref('')
const newTokenExpiresAt = ref<Date | null>(null)
const createdSecretToken = ref('')
const openapiTokens = ref<OpenAPITokenInfo[]>([])

const availableTokenCount = computed(() => {
  return openapiTokens.value.filter((token) => !token.revoked_at).length
})
const tokenSummary = computed(() => {
  return t('openapiToken.total', {
    count: openapiTokens.value.length,
    available: availableTokenCount.value,
  })
})

async function loadOpenAPITokens() {
  tokensLoading.value = true
  try {
    const resp = await listOpenAPITokens()
    openapiTokens.value = resp.tokens || []
  } catch (error: any) {
    ElMessage.error(error?.message || t('openapiToken.loadFailed'))
  } finally {
    tokensLoading.value = false
  }
}

async function handleCreateToken() {
  if (!newTokenName.value.trim()) {
    ElMessage.warning(t('openapiToken.nameRequired'))
    return
  }
  creatingToken.value = true
  try {
    const resp = await createOpenAPIToken({
      name: newTokenName.value.trim(),
      expires_at: newTokenExpiresAt.value ? newTokenExpiresAt.value.toISOString() : undefined,
    })
    createdSecretToken.value = resp.secret_token
    createdTokenVisible.value = true
    createTokenDialogVisible.value = false
    newTokenName.value = ''
    newTokenExpiresAt.value = null
    await loadOpenAPITokens()
  } catch (error: any) {
    ElMessage.error(error?.message || t('openapiToken.createFailed'))
  } finally {
    creatingToken.value = false
  }
}

async function handleRevokeToken(id: number) {
  try {
    await ElMessageBox.confirm(t('openapiToken.revokeConfirm'), t('openapiToken.revokeTitle'), {
      type: 'warning',
      confirmButtonText: t('openapiToken.revoke'),
      cancelButtonText: t('common.cancel'),
    })
    await revokeOpenAPIToken(id)
    ElMessage.success(t('openapiToken.revokedSuccess'))
    await loadOpenAPITokens()
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error(error?.message || t('openapiToken.revokeFailed'))
    }
  }
}

async function copyCreatedToken() {
  await navigator.clipboard.writeText(createdSecretToken.value)
  ElMessage.success(t('openapiToken.copied'))
}

onMounted(loadOpenAPITokens)
</script>

<style scoped lang="scss">
.openapi-token-page {
  padding: 20px;

  .page-card {
    border-radius: 8px;
  }

  &.is-embedded {
    padding: 0;

    .page-card {
      border: 0;
      box-shadow: none;
      background: transparent;
    }
  }

  .card-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;

    h2 {
      margin: 0;
      font-size: 24px;
      font-weight: 700;
      color: var(--el-text-color-primary);
    }

    .header-desc {
      margin: 8px 0 0;
      color: var(--el-text-color-secondary);
      font-size: 14px;
    }
  }

  .header-actions {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
  }

  .page-body {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .embedded-actions,
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }

  .toolbar-summary {
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }

  .form-tip {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin: 0;
  }

  .token-secret {
    margin-top: 12px;
  }
}

@media (max-width: 768px) {
  .openapi-token-page {
    padding: 12px;

    .card-header {
      flex-direction: column;
      align-items: stretch;
    }

    .header-actions {
      justify-content: flex-start;
    }

    .toolbar {
      justify-content: flex-start;
    }
  }
}
</style>
