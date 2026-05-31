<template>
  <div class="connector-provider-page">
    <el-card shadow="hover" class="page-card">
      <template #header>
        <div class="card-header">
          <div>
            <h2>{{ t('connectorProvider.title') }}</h2>
            <p class="header-desc">{{ t('connectorProvider.subtitle') }}</p>
          </div>
          <div class="header-actions">
            <el-button :icon="Refresh" @click="loadProviders">{{ t('common.refresh') }}</el-button>
            <el-button type="primary" :icon="Plus" @click="handleCreate">
              {{ t('connectorProvider.createProvider') }}
            </el-button>
          </div>
        </div>
      </template>

      <div class="page-body">
        <div class="callback-row">
          <div>
            <div class="callback-label">{{ t('connectorProvider.callbackUrl') }}</div>
            <div class="callback-url">{{ callbackURL }}</div>
          </div>
          <el-button :icon="CopyDocument" @click="copyCallbackURL">
            {{ t('connectorProvider.copy') }}
          </el-button>
        </div>

        <div class="summary-strip">
          <div class="summary-item">
            <span class="summary-value">{{ providers.length }}</span>
            <span class="summary-label">{{ t('connectorProvider.platformCount') }}</span>
          </div>
          <div class="summary-item">
            <span class="summary-value">{{ activeCount }}</span>
            <span class="summary-label">{{ t('connectorProvider.activeCount') }}</span>
          </div>
          <div class="summary-item">
            <span class="summary-value">{{ managedCount }}</span>
            <span class="summary-label">{{ t('connectorProvider.initializedCount') }}</span>
          </div>
          <div class="summary-item">
            <span class="summary-value">{{ secretCount }}</span>
            <span class="summary-label">{{ t('connectorProvider.secretCount') }}</span>
          </div>
        </div>

        <div class="toolbar">
          <el-input
            v-model="keyword"
            clearable
            :placeholder="t('connectorProvider.searchPlaceholder')"
            class="toolbar-search"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <div class="toolbar-summary">
            {{ t('connectorProvider.total', { count: filteredProviders.length }) }}
            <span v-if="keyword">({{ t('connectorProvider.filtered') }})</span>
          </div>
        </div>

        <el-table
          v-loading="loading"
          :data="filteredProviders"
          row-key="code"
          stripe
          style="width: 100%"
          :empty-text="t('connectorProvider.empty')"
        >
          <el-table-column :label="t('connectorProvider.provider')" min-width="220">
            <template #default="{ row }">
              <div class="provider-cell">
                <div class="provider-title">
                  <span class="provider-name">{{ row.name || row.code }}</span>
                  <el-tag v-if="row.managed" size="small" type="primary">
                    {{ t('connectorProvider.initialized') }}
                  </el-tag>
                  <el-tag v-else size="small" type="info">
                    {{ t('connectorProvider.builtIn') }}
                  </el-tag>
                </div>
                <div class="provider-code">{{ row.code }}</div>
              </div>
            </template>
          </el-table-column>

          <el-table-column :label="t('connectorProvider.status')" width="110" align="center">
            <template #default="{ row }">
              <el-tag :type="providerStatusType(row)">
                {{ providerStatusLabel(row) }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column :label="t('connectorProvider.keys')" width="170">
            <template #default="{ row }">
              <div class="secret-state">
                <el-tag :type="row.client_id ? 'success' : 'warning'" size="small">
                  {{ row.client_id ? t('connectorProvider.clientID') : t('connectorProvider.missingClientID') }}
                </el-tag>
                <el-tag :type="row.has_client_secret ? 'success' : 'warning'" size="small">
                  {{ row.has_client_secret ? 'Secret' : t('connectorProvider.missingSecret') }}
                </el-tag>
              </div>
            </template>
          </el-table-column>

          <el-table-column :label="t('connectorProvider.authEndpoint')" min-width="260" show-overflow-tooltip>
            <template #default="{ row }">
              <div class="url-cell">{{ row.auth_url || '-' }}</div>
              <div class="muted-line">{{ row.token_url || '-' }}</div>
            </template>
          </el-table-column>

          <el-table-column :label="t('connectorProvider.scope')" min-width="220">
            <template #default="{ row }">
              <div v-if="row.scopes?.length" class="scope-list">
                <el-tag
                  v-for="scope in row.scopes.slice(0, 3)"
                  :key="scope"
                  size="small"
                  effect="plain"
                >
                  {{ scope }}
                </el-tag>
                <el-tag v-if="row.scopes.length > 3" size="small" type="info" effect="plain">
                  +{{ row.scopes.length - 3 }}
                </el-tag>
              </div>
              <span v-else class="muted-line">-</span>
            </template>
          </el-table-column>

          <el-table-column :label="t('connectorProvider.updatedAt')" width="180">
            <template #default="{ row }">
              {{ formatDateTime(row.updated_at) }}
            </template>
          </el-table-column>

          <el-table-column :label="t('common.operation')" width="190" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" size="small" :icon="Edit" @click="handleEdit(row)">
                {{ t('connectorProvider.edit') }}
              </el-button>
              <el-button
                link
                type="danger"
                size="small"
                :icon="Delete"
                :disabled="!row.managed"
                @click="handleDelete(row)"
              >
                {{ t('connectorProvider.delete') }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="820px"
      :close-on-click-modal="false"
      destroy-on-close
      class="provider-dialog"
    >
      <div v-loading="dialogLoading">
        <el-form ref="formRef" :model="form" :rules="rules" label-width="140px">
          <el-divider content-position="left">{{ t('connectorProvider.basicInfo') }}</el-divider>
          <el-form-item :label="t('connectorProvider.providerCode')" prop="code">
            <el-input
              v-model="form.code"
              :disabled="dialogMode === 'edit'"
              :placeholder="t('connectorProvider.providerCodePlaceholder')"
            />
          </el-form-item>
          <el-form-item :label="t('connectorProvider.displayName')" prop="name">
            <el-input v-model="form.name" :placeholder="t('connectorProvider.displayNamePlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('connectorProvider.enabled')">
            <el-switch v-model="form.enabled" />
          </el-form-item>

          <el-divider content-position="left">{{ t('connectorProvider.oauthSecrets') }}</el-divider>
          <el-form-item :label="t('connectorProvider.clientID')" prop="client_id">
            <el-input v-model="form.client_id" :placeholder="t('connectorProvider.clientIDPlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('connectorProvider.clientSecret')">
            <el-input
              v-model="form.client_secret"
              type="password"
              show-password
              :placeholder="form.has_client_secret ? t('connectorProvider.clientSecretConfiguredPlaceholder') : t('connectorProvider.clientSecretPlaceholder')"
            />
          </el-form-item>

          <el-divider content-position="left">{{ t('connectorProvider.oauthEndpoints') }}</el-divider>
          <el-form-item :label="t('connectorProvider.authURL')" prop="auth_url">
            <el-input v-model="form.auth_url" :placeholder="t('connectorProvider.authURLPlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('connectorProvider.tokenURL')" prop="token_url">
            <el-input v-model="form.token_url" :placeholder="t('connectorProvider.tokenURLPlaceholder')" />
          </el-form-item>
          <el-form-item :label="t('connectorProvider.scopes')">
            <el-input
              v-model="form.scopes_text"
              type="textarea"
              :rows="3"
              :placeholder="t('connectorProvider.scopesPlaceholder')"
            />
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ dialogMode === 'create' ? t('connectorProvider.create') : t('connectorProvider.save') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { CopyDocument, Delete, Edit, Plus, Refresh, Search } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import {
  deleteConnectorOAuthProvider,
  getConnectorOAuthProvider,
  listConnectorOAuthProviders,
  upsertConnectorOAuthProvider,
  type ConnectorOAuthProviderInfo,
  type UpsertConnectorOAuthProviderReq
} from '@/architecture/presentation/context/api/connector'

type DialogMode = 'create' | 'edit'

interface ProviderFormState {
  code: string
  name: string
  enabled: boolean
  client_id: string
  client_secret: string
  auth_url: string
  token_url: string
  scopes_text: string
  has_client_secret: boolean
}

const providers = ref<ConnectorOAuthProviderInfo[]>([])
const { t } = useI18n()
const keyword = ref('')
const loading = ref(false)
const dialogLoading = ref(false)
const dialogVisible = ref(false)
const dialogMode = ref<DialogMode>('create')
const submitting = ref(false)
const formRef = ref<FormInstance>()
const form = reactive<ProviderFormState>(createDefaultForm())

const callbackURL = computed(() => {
  if (typeof window === 'undefined') {
    return '/connector/oauth/callback'
  }
  return `${window.location.origin}/connector/oauth/callback`
})

const dialogTitle = computed(() => {
  if (dialogMode.value === 'create') {
    return t('connectorProvider.createDialogTitle')
  }
  return t('connectorProvider.editDialogTitle', { code: form.code })
})

const filteredProviders = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) {
    return providers.value
  }
  return providers.value.filter((item) => {
    return [
      item.code,
      item.name,
      item.auth_url,
      item.token_url,
      ...(item.scopes || [])
    ].some((field) => (field || '').toLowerCase().includes(q))
  })
})

const activeCount = computed(() => providers.value.filter((item) => item.active).length)
const managedCount = computed(() => providers.value.filter((item) => item.managed).length)
const secretCount = computed(() => providers.value.filter((item) => item.has_client_secret).length)

const rules: FormRules<ProviderFormState> = {
  code: [
    { required: true, message: () => t('connectorProvider.codeRequired'), trigger: 'blur' },
    {
      pattern: /^[a-z0-9][a-z0-9_-]*$/,
      message: () => t('connectorProvider.codePattern'),
      trigger: 'blur'
    }
  ],
  name: [
    { required: true, message: () => t('connectorProvider.nameRequired'), trigger: 'blur' }
  ],
  client_id: [
    { required: true, message: () => t('connectorProvider.clientIDRequired'), trigger: 'blur' }
  ],
  auth_url: [
    { required: true, message: () => t('connectorProvider.authURLRequired'), trigger: 'blur' }
  ],
  token_url: [
    { required: true, message: () => t('connectorProvider.tokenURLRequired'), trigger: 'blur' }
  ]
}

function createDefaultForm(): ProviderFormState {
  return {
    code: '',
    name: '',
    enabled: true,
    client_id: '',
    client_secret: '',
    auth_url: '',
    token_url: '',
    scopes_text: '',
    has_client_secret: false
  }
}

function resetForm() {
  Object.assign(form, createDefaultForm())
  formRef.value?.clearValidate()
}

function applyForm(provider: ConnectorOAuthProviderInfo) {
  form.code = provider.code || ''
  form.name = provider.name || provider.code || ''
  form.enabled = provider.enabled !== false
  form.client_id = provider.client_id || ''
  form.client_secret = ''
  form.auth_url = provider.auth_url || ''
  form.token_url = provider.token_url || ''
  form.scopes_text = (provider.scopes || []).join('\n')
  form.has_client_secret = provider.has_client_secret
}

function parseScopes(value: string) {
  return value
    .split(/[\s,，;；]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function providerStatusType(row: ConnectorOAuthProviderInfo) {
  if (!row.enabled) {
    return 'info'
  }
  return row.active ? 'success' : 'warning'
}

function providerStatusLabel(row: ConnectorOAuthProviderInfo) {
  if (!row.enabled) {
    return t('connectorProvider.disabled')
  }
  return row.active ? t('connectorProvider.active') : t('connectorProvider.pending')
}

function formatDateTime(value?: string) {
  if (!value) {
    return '-'
  }
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss')
}

async function copyCallbackURL() {
  try {
    await navigator.clipboard.writeText(callbackURL.value)
    ElMessage.success(t('connectorProvider.callbackCopied'))
  } catch {
    ElMessage.error(t('connectorProvider.copyFailed'))
  }
}

async function loadProviders() {
  loading.value = true
  try {
    const resp = await listConnectorOAuthProviders()
    providers.value = resp.providers || []
  } catch (error: any) {
    ElMessage.error(error?.message || t('connectorProvider.loadFailed'))
  } finally {
    loading.value = false
  }
}

function handleCreate() {
  resetForm()
  dialogMode.value = 'create'
  dialogVisible.value = true
}

async function handleEdit(row: ConnectorOAuthProviderInfo) {
  resetForm()
  dialogMode.value = 'edit'
  dialogVisible.value = true
  dialogLoading.value = true
  try {
    const resp = await getConnectorOAuthProvider(row.code)
    applyForm(resp.provider)
  } catch (error: any) {
    ElMessage.error(error?.message || t('connectorProvider.loadProviderFailed'))
    dialogVisible.value = false
  } finally {
    dialogLoading.value = false
  }
}

function buildPayload(): UpsertConnectorOAuthProviderReq {
  const payload: UpsertConnectorOAuthProviderReq = {
    code: form.code.trim().toLowerCase(),
    name: form.name.trim(),
    enabled: form.enabled,
    client_id: form.client_id.trim(),
    auth_url: form.auth_url.trim(),
    token_url: form.token_url.trim(),
    scopes: parseScopes(form.scopes_text)
  }

  if (form.client_secret.trim()) {
    payload.client_secret = form.client_secret.trim()
  }
  return payload
}

async function handleSubmit() {
  if (!formRef.value) {
    return
  }
  try {
    await formRef.value.validate()
    submitting.value = true
    const payload = buildPayload()
    await upsertConnectorOAuthProvider(payload.code, payload)
    ElMessage.success(dialogMode.value === 'create'
      ? t('connectorProvider.providerCreated')
      : t('connectorProvider.providerSaved'))
    dialogVisible.value = false
    await loadProviders()
  } catch (error: any) {
    if (error?.message && !String(error.message).includes('validate')) {
      ElMessage.error(error.message || t('connectorProvider.submitFailed'))
    }
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row: ConnectorOAuthProviderInfo) {
  if (!row.managed) {
    return
  }
  try {
    await ElMessageBox.confirm(
      t('connectorProvider.deleteConfirm', { name: row.name || row.code }),
      t('connectorProvider.deleteTitle'),
      {
        type: 'warning',
        confirmButtonText: t('connectorProvider.deleteConfirmButton'),
        cancelButtonText: t('common.cancel')
      }
    )
    await deleteConnectorOAuthProvider(row.code)
    ElMessage.success(t('connectorProvider.providerDeleted'))
    await loadProviders()
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') {
      return
    }
    ElMessage.error(error?.message || t('connectorProvider.deleteFailed'))
  }
}

onMounted(loadProviders)
</script>

<style scoped lang="scss">
.connector-provider-page {
  min-height: 100vh;
  padding: 20px;
  background: var(--el-bg-color-page);

  .page-card {
    border-radius: 8px;
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

  .header-actions,
  .toolbar,
  .secret-state,
  .scope-list {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
  }

  .page-body {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .callback-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 14px 16px;
    border: 1px solid var(--el-border-color);
    border-radius: 8px;
    background: var(--el-fill-color-lighter);
  }

  .callback-label {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }

  .callback-url {
    margin-top: 4px;
    color: var(--el-text-color-primary);
    font-family: var(--el-font-family);
    word-break: break-all;
  }

  .summary-strip {
    display: grid;
    grid-template-columns: repeat(4, minmax(120px, 1fr));
    gap: 12px;
  }

  .summary-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 12px 14px;
    border: 1px solid var(--el-border-color);
    border-radius: 8px;
    background: var(--el-fill-color-blank);
  }

  .summary-value {
    font-size: 22px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .summary-label,
  .muted-line,
  .provider-code {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .toolbar {
    justify-content: space-between;
  }

  .toolbar-search {
    width: min(460px, 100%);
  }

  .toolbar-summary {
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }

  .provider-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .provider-title {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .provider-name {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .url-cell {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

}

@media (max-width: 900px) {
  .connector-provider-page {
    padding: 12px;

    .card-header,
    .callback-row,
    .toolbar {
      flex-direction: column;
      align-items: stretch;
    }

    .summary-strip {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
}

@media (max-width: 560px) {
  .connector-provider-page {
    .summary-strip {
      grid-template-columns: 1fr;
    }
  }
}
</style>
