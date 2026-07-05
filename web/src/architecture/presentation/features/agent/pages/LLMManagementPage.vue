<template>
  <div class="llm-management-page">
    <el-card shadow="hover" class="page-card">
      <template #header>
        <div class="card-header">
          <div>
            <h2>{{ t('llmManagement.title') }}</h2>
            <p class="header-desc">{{ t('llmManagement.subtitle') }}</p>
          </div>
          <div class="header-actions">
            <el-button :icon="Refresh" @click="handleRefresh">{{ t('common.refresh') }}</el-button>
            <el-button
              v-if="activeScope === 'mine'"
              type="primary"
              :icon="Plus"
              @click="handleCreate"
            >
              {{ t('llmManagement.createConfig') }}
            </el-button>
          </div>
        </div>
      </template>

      <div class="page-body">
        <div class="default-panel" v-loading="defaultLoading">
          <div class="default-panel-label">{{ t('llmManagement.currentDefault') }}</div>
          <template v-if="defaultConfig">
            <div class="default-panel-main">
              <div>
                <div class="default-name">{{ defaultConfig.name }}</div>
                <div class="default-meta">
                  {{ defaultConfig.model }} · {{ protocolLabel(defaultConfig.protocol) }}
                </div>
              </div>
              <div class="default-tags">
                <el-tag type="warning">{{ t('llmManagement.defaultTag') }}</el-tag>
                <el-tag :type="defaultConfig.visibility === 0 ? 'success' : 'info'">
                  {{ visibilityLabel(defaultConfig.visibility) }}
                </el-tag>
              </div>
            </div>
          </template>
          <template v-else>
            <div class="default-empty">{{ t('llmManagement.noDefault') }}</div>
          </template>
        </div>

        <el-tabs v-model="activeScope" class="scope-tabs" @tab-change="handleScopeChange">
          <el-tab-pane :label="t('llmManagement.myConfigs')" name="mine" />
          <el-tab-pane :label="t('llmManagement.publicMarket')" name="market" />
        </el-tabs>

        <div class="toolbar">
          <el-input
            v-model="keyword"
            clearable
            :placeholder="t('llmManagement.searchPlaceholder')"
            class="toolbar-search"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <div class="toolbar-summary">
            {{ t('llmManagement.total', { count: filteredConfigs.length }) }}
            <span v-if="keyword">{{ t('llmManagement.filtered') }}</span>
          </div>
        </div>

        <el-table
          v-loading="loading"
          :data="filteredConfigs"
          stripe
          style="width: 100%"
          :empty-text="t('llmManagement.empty')"
        >
          <el-table-column :label="t('llmManagement.config')" min-width="240">
            <template #default="{ row }">
              <div class="name-cell">
                <div class="name-line">
                  <span class="config-name">{{ row.name }}</span>
                  <el-tag size="small">{{ providerLabel(row.provider) }}</el-tag>
                  <el-tag v-if="row.is_default" type="warning" size="small">{{ t('llmManagement.defaultTag') }}</el-tag>
                  <el-tag v-if="row.is_admin" type="primary" size="small">{{ t('llmManagement.manageable') }}</el-tag>
                </div>
                <div class="meta-line">{{ row.model }} · {{ protocolLabel(row.protocol) }}</div>
              </div>
            </template>
          </el-table-column>

          <el-table-column :label="t('llmManagement.visibility')" width="120" align="center">
            <template #default="{ row }">
              <el-tag :type="row.visibility === 0 ? 'success' : 'info'">
                {{ visibilityLabel(row.visibility) }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column label="API Key" width="110" align="center">
            <template #default="{ row }">
              <el-tag :type="row.has_api_key ? 'success' : 'info'">
                {{ row.has_api_key ? t('llmManagement.configured') : t('llmManagement.notConfigured') }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column prop="api_base" label="API Base" min-width="220" show-overflow-tooltip />

          <el-table-column :label="t('llmManagement.timeoutToken')" width="140" align="center">
            <template #default="{ row }">
              <div>{{ row.timeout }}s</div>
              <div class="muted-line">{{ row.max_tokens }} tokens</div>
            </template>
          </el-table-column>

          <el-table-column prop="admin" :label="t('llmManagement.admin')" min-width="180" show-overflow-tooltip />

          <el-table-column :label="t('llmManagement.updatedAt')" width="180">
            <template #default="{ row }">
              {{ formatDateTime(row.updated_at) }}
            </template>
          </el-table-column>

          <el-table-column :label="t('common.operation')" width="220" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="row.is_admin && !row.is_default"
                link
                type="warning"
                size="small"
                @click="handleSetDefault(row)"
              >
                {{ t('llmManagement.setDefault') }}
              </el-button>
              <el-button
                v-if="row.is_admin"
                link
                type="primary"
                size="small"
                @click="handleEdit(row)"
              >
                {{ t('llmManagement.edit') }}
              </el-button>
              <el-button
                v-if="row.is_admin"
                link
                type="danger"
                size="small"
                @click="handleDelete(row)"
              >
                {{ t('common.delete') }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? t('llmManagement.createDialogTitle') : t('llmManagement.editDialogTitle')"
      width="720px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <div v-loading="dialogLoading">
        <el-form
          ref="formRef"
          :model="form"
          :rules="rules"
          label-width="110px"
        >
          <el-form-item :label="t('llmManagement.configName')" prop="name">
            <el-input v-model="form.name" :placeholder="t('llmManagement.configNamePlaceholder')" />
          </el-form-item>

          <el-form-item :label="t('llmManagement.model')" prop="model">
            <el-input v-model="form.model" :placeholder="t('llmManagement.modelPlaceholder')" />
          </el-form-item>

          <el-row :gutter="12">
            <el-col :xs="24" :sm="12">
              <el-form-item :label="t('llmManagement.provider')" prop="provider">
                <el-select v-model="form.provider" style="width: 100%" @change="handleProviderChange">
                  <el-option
                    v-for="option in providerOptions"
                    :key="option.value"
                    :label="option.label"
                    :value="option.value"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12">
              <el-form-item :label="t('llmManagement.protocol')" prop="protocol">
                <el-select v-model="form.protocol" style="width: 100%" @change="handleProtocolChange">
                  <el-option
                    v-for="option in protocolOptions"
                    :key="option.value"
                    :label="option.label"
                    :value="option.value"
                  />
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>

          <el-form-item label="API Key">
            <el-input
              v-model="form.api_key"
              type="password"
              show-password
              :placeholder="t('llmManagement.apiKeyPlaceholder')"
            />
          </el-form-item>

          <el-form-item label="API Base">
            <el-input v-model="form.api_base" :placeholder="t('llmManagement.apiBasePlaceholder')" />
          </el-form-item>

          <el-row :gutter="12">
            <el-col :xs="24" :sm="12">
              <el-form-item :label="t('llmManagement.endpointPath')">
                <el-input v-model="form.endpoint_path" :placeholder="endpointPathPlaceholder" />
              </el-form-item>
            </el-col>
            <el-col :xs="24" :sm="12">
              <el-form-item :label="t('llmManagement.apiVersion')">
                <el-input v-model="form.api_version" :placeholder="apiVersionPlaceholder" />
              </el-form-item>
            </el-col>
          </el-row>

          <el-form-item :label="t('llmManagement.authScheme')">
            <el-select v-model="form.auth_scheme" clearable style="width: 220px" :placeholder="authSchemePlaceholder">
              <el-option
                v-for="option in authSchemeOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
          </el-form-item>

          <el-form-item :label="t('llmManagement.timeout')">
            <el-input-number v-model="form.timeout" :min="1" :max="3600" style="width: 180px" />
            <span class="form-suffix">{{ t('llmManagement.seconds') }}</span>
          </el-form-item>

          <el-form-item :label="t('llmManagement.maxToken')">
            <el-input-number v-model="form.max_tokens" :min="1" :max="1048576" style="width: 180px" />
          </el-form-item>

          <el-form-item :label="t('llmManagement.visibility')">
            <el-radio-group v-model="form.visibility">
              <el-radio :value="0">{{ t('llmManagement.public') }}</el-radio>
              <el-radio :value="1">{{ t('llmManagement.private') }}</el-radio>
            </el-radio-group>
          </el-form-item>

          <el-form-item :label="t('llmManagement.admin')">
            <el-input
              v-model="form.admin"
              :placeholder="t('llmManagement.adminPlaceholder')"
            />
          </el-form-item>

          <el-form-item :label="t('llmManagement.setAsDefault')">
            <el-switch v-model="form.is_default" />
          </el-form-item>

          <el-form-item :label="t('llmManagement.extraConfig')" prop="extra_config">
            <el-input
              v-model="form.extra_config"
              type="textarea"
              :rows="6"
              :placeholder="t('llmManagement.extraConfigPlaceholder')"
            />
          </el-form-item>

          <el-form-item :label="t('llmManagement.headers')" prop="headers">
            <el-input
              v-model="form.headers"
              type="textarea"
              :rows="4"
              :placeholder="t('llmManagement.headersPlaceholder')"
            />
          </el-form-item>

          <el-form-item :label="t('llmManagement.capabilities')" prop="capabilities">
            <el-input
              v-model="form.capabilities"
              type="textarea"
              :rows="4"
              :placeholder="t('llmManagement.capabilitiesPlaceholder')"
            />
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <el-button :icon="Connection" :loading="probing" @click="handleProbe">
          {{ t('llmManagement.probe') }}
        </el-button>
        <el-button @click="dialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ dialogMode === 'create' ? t('llmManagement.create') : t('llmManagement.save') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Connection, Plus, Refresh, Search } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import {
  createLLM,
  deleteLLM,
  getDefaultLLM,
  getLLM,
  getLLMList,
  probeLLM,
  setDefaultLLM,
  updateLLM,
  type LLMCreateReq,
  type LLMInfo,
  type LLMUpdateReq
} from '@/architecture/presentation/context/api/agent'

type Scope = 'mine' | 'market'
type DialogMode = 'create' | 'edit'

interface LLMFormState {
  id: number | null
  name: string
  provider: string
  protocol: string
  model: string
  api_key: string
  api_base: string
  endpoint_path: string
  api_version: string
  auth_scheme: string
  headers: string
  timeout: number
  max_tokens: number
  extra_config: string
  capabilities: string
  is_default: boolean
  visibility: number
  admin: string
}

const DEFAULT_TIMEOUT = 300
const DEFAULT_MAX_TOKENS = 8196
const DEFAULT_PROVIDER = 'openai'
const DEFAULT_PROTOCOL = 'openai_chat_completions'

const { t } = useI18n()

const protocolDefaults: Record<string, { provider: string; endpointPath: string; apiVersion: string; authScheme: string }> = {
  openai_chat_completions: {
    provider: 'openai',
    endpointPath: '/chat/completions',
    apiVersion: '',
    authScheme: 'bearer'
  },
  openai_responses: {
    provider: 'openai',
    endpointPath: '/responses',
    apiVersion: '',
    authScheme: 'bearer'
  },
  anthropic_messages: {
    provider: 'anthropic',
    endpointPath: '/v1/messages',
    apiVersion: '2023-06-01',
    authScheme: 'x-api-key'
  }
}

const activeScope = ref<Scope>('mine')
const keyword = ref('')
const loading = ref(false)
const defaultLoading = ref(false)
const dialogLoading = ref(false)
const dialogVisible = ref(false)
const dialogMode = ref<DialogMode>('create')
const submitting = ref(false)
const probing = ref(false)

const configs = ref<LLMInfo[]>([])
const defaultConfig = ref<LLMInfo | null>(null)

const formRef = ref<FormInstance>()
const form = reactive<LLMFormState>(createDefaultForm())

const providerOptions = computed(() => [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' }
])

const protocolOptions = computed(() => {
  const options = [
    { value: 'openai_chat_completions', label: 'OpenAI Chat Completions' },
    { value: 'openai_responses', label: 'OpenAI Responses' },
    { value: 'anthropic_messages', label: 'Anthropic Messages' }
  ]
  if (form.provider === 'anthropic') {
    return options.filter((option) => option.value === 'anthropic_messages')
  }
  return options.filter((option) => option.value !== 'anthropic_messages')
})

const authSchemeOptions = computed(() => [
  { value: 'bearer', label: 'Bearer' },
  { value: 'x-api-key', label: 'x-api-key' },
  { value: 'none', label: t('llmManagement.authNone') }
])

const endpointPathPlaceholder = computed(() => protocolDefaults[form.protocol]?.endpointPath || '/chat/completions')
const apiVersionPlaceholder = computed(() => protocolDefaults[form.protocol]?.apiVersion || t('llmManagement.apiVersionPlaceholder'))
const authSchemePlaceholder = computed(() => protocolDefaults[form.protocol]?.authScheme || 'bearer')

const filteredConfigs = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return configs.value
  return configs.value.filter((item) => {
    return [
      item.name,
      item.provider,
      item.protocol,
      item.model,
      item.api_base,
      item.endpoint_path,
      item.admin
    ].some((field) => (field || '').toLowerCase().includes(q))
  })
})

const rules = computed<FormRules<LLMFormState>>(() => ({
  name: [
    { required: true, message: t('llmManagement.nameRequired'), trigger: 'blur' }
  ],
  provider: [
    { required: true, message: t('llmManagement.providerRequired'), trigger: 'change' }
  ],
  protocol: [
    { required: true, message: t('llmManagement.protocolRequired'), trigger: 'change' }
  ],
  model: [
    { required: true, message: t('llmManagement.modelRequired'), trigger: 'blur' }
  ],
  extra_config: [
    createJSONValidator(() => t('llmManagement.extraConfigInvalid'))
  ],
  headers: [
    createJSONValidator(() => t('llmManagement.headersInvalid'))
  ],
  capabilities: [
    createJSONValidator(() => t('llmManagement.capabilitiesInvalid'))
  ]
}))

function createJSONValidator(message: () => string) {
  return {
    validator: (_rule: unknown, value: string, callback: (error?: Error) => void) => {
      const text = (value || '').trim()
      if (!text) {
        callback()
        return
      }
      try {
        JSON.parse(text)
        callback()
      } catch {
        callback(new Error(message()))
      }
    },
    trigger: 'blur'
  }
}

function createDefaultForm(): LLMFormState {
  return {
    id: null,
    name: '',
    provider: DEFAULT_PROVIDER,
    protocol: DEFAULT_PROTOCOL,
    model: '',
    api_key: '',
    api_base: '',
    endpoint_path: '',
    api_version: '',
    auth_scheme: '',
    headers: '',
    timeout: DEFAULT_TIMEOUT,
    max_tokens: DEFAULT_MAX_TOKENS,
    extra_config: '',
    capabilities: '',
    is_default: false,
    visibility: 0,
    admin: ''
  }
}

function resetForm() {
  Object.assign(form, createDefaultForm())
  formRef.value?.clearValidate()
}

function applyForm(info: Partial<LLMInfo>) {
  form.id = info.id ?? null
  form.name = info.name || ''
  form.provider = info.provider || DEFAULT_PROVIDER
  form.protocol = info.protocol || defaultProtocolForProvider(form.provider)
  form.model = info.model || ''
  form.api_key = info.api_key || ''
  form.api_base = info.api_base || ''
  form.endpoint_path = info.endpoint_path || ''
  form.api_version = info.api_version || ''
  form.auth_scheme = info.auth_scheme || ''
  form.headers = info.headers || ''
  form.timeout = info.timeout || DEFAULT_TIMEOUT
  form.max_tokens = info.max_tokens || DEFAULT_MAX_TOKENS
  form.extra_config = info.extra_config || ''
  form.capabilities = info.capabilities || ''
  form.is_default = Boolean(info.is_default)
  form.visibility = typeof info.visibility === 'number' ? info.visibility : 0
  form.admin = info.admin || ''
}

function defaultProtocolForProvider(provider: string) {
  return provider === 'anthropic' ? 'anthropic_messages' : DEFAULT_PROTOCOL
}

function inferProviderProtocol(provider: string, protocol: string, apiBase: string, endpointPath: string) {
  const base = (apiBase || '').trim().toLowerCase().replace(/\/+$/, '')
  const endpoint = (endpointPath || '').trim().toLowerCase()
  let nextProvider = (provider || '').trim() || DEFAULT_PROVIDER
  let nextProtocol = (protocol || '').trim() || defaultProtocolForProvider(nextProvider)

  if (
    nextProvider === 'openai' &&
    nextProtocol === 'openai_chat_completions' &&
    (endpoint.includes('responses') || base.endsWith('/responses'))
  ) {
    nextProtocol = 'openai_responses'
  }

  if (base.includes('anthropic') || endpoint.includes('messages')) {
    nextProvider = 'anthropic'
    nextProtocol = 'anthropic_messages'
  }

  return { provider: nextProvider, protocol: nextProtocol }
}

function applyEndpointProtocolInference() {
  const inferred = inferProviderProtocol(form.provider, form.protocol, form.api_base, form.endpoint_path)
  form.provider = inferred.provider
  form.protocol = inferred.protocol
}

function handleProviderChange() {
  const expected = defaultProtocolForProvider(form.provider)
  if (!protocolOptions.value.some((option) => option.value === form.protocol)) {
    form.protocol = expected
  }
  applyProtocolDefaults()
}

function handleProtocolChange() {
  const defaults = protocolDefaults[form.protocol]
  if (defaults?.provider && form.provider !== defaults.provider) {
    form.provider = defaults.provider
  }
  applyProtocolDefaults(true)
}

function applyProtocolDefaults(force = false) {
  const defaults = protocolDefaults[form.protocol]
  if (!defaults) return
  if (force || !form.endpoint_path.trim()) {
    form.endpoint_path = defaults.endpointPath
  }
  if (force || !form.api_version.trim()) {
    form.api_version = defaults.apiVersion
  }
  if (force || !form.auth_scheme.trim()) {
    form.auth_scheme = defaults.authScheme
  }
}

function providerLabel(provider: string) {
  return provider === 'anthropic' ? 'Anthropic' : 'OpenAI'
}

function protocolLabel(protocol: string) {
  switch (protocol) {
    case 'openai_responses':
      return 'Responses'
    case 'anthropic_messages':
      return 'Messages'
    default:
      return 'Chat Completions'
  }
}

function visibilityLabel(visibility: number) {
  return visibility === 1 ? t('llmManagement.private') : t('llmManagement.public')
}

function formatDateTime(value: string) {
  if (!value) return '-'
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss')
}

async function loadConfigs() {
  loading.value = true
  try {
    const resp = await getLLMList({
      scope: activeScope.value,
      page: 1,
      page_size: 200
    })
    configs.value = resp.configs || []
  } catch (error: any) {
    console.error('加载 LLM 配置失败:', error)
    ElMessage.error(error?.message || t('llmManagement.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadDefaultConfig() {
  defaultLoading.value = true
  try {
    defaultConfig.value = await getDefaultLLM()
  } catch {
    defaultConfig.value = null
  } finally {
    defaultLoading.value = false
  }
}

async function handleRefresh() {
  await Promise.all([loadConfigs(), loadDefaultConfig()])
}

async function handleScopeChange() {
  keyword.value = ''
  await loadConfigs()
}

function handleCreate() {
  resetForm()
  dialogMode.value = 'create'
  dialogVisible.value = true
}

async function handleEdit(row: LLMInfo) {
  dialogMode.value = 'edit'
  dialogVisible.value = true
  dialogLoading.value = true
  resetForm()

  try {
    const detail = await getLLM({ id: row.id })
    applyForm(detail)
  } catch (error: any) {
    console.error('加载 LLM 配置详情失败:', error)
    ElMessage.error(error?.message || t('llmManagement.loadDetailFailed'))
    dialogVisible.value = false
  } finally {
    dialogLoading.value = false
  }
}

function buildCreatePayload(): LLMCreateReq {
  applyEndpointProtocolInference()
  return {
    name: form.name.trim(),
    provider: form.provider.trim(),
    protocol: form.protocol.trim(),
    model: form.model.trim(),
    api_key: form.api_key.trim(),
    api_base: form.api_base.trim(),
    endpoint_path: form.endpoint_path.trim(),
    api_version: form.api_version.trim(),
    auth_scheme: form.auth_scheme.trim(),
    headers: form.headers.trim(),
    timeout: form.timeout,
    max_tokens: form.max_tokens,
    extra_config: form.extra_config.trim(),
    capabilities: form.capabilities.trim(),
    is_default: form.is_default,
    visibility: form.visibility,
    admin: form.admin.trim()
  }
}

function buildUpdatePayload(): LLMUpdateReq {
  return {
    id: form.id || 0,
    ...buildCreatePayload()
  }
}

function buildProbePayload() {
  applyEndpointProtocolInference()
  return {
    id: form.id || undefined,
    provider: form.provider.trim(),
    protocol: form.protocol.trim(),
    model: form.model.trim(),
    api_key: form.api_key.trim(),
    api_base: form.api_base.trim(),
    endpoint_path: form.endpoint_path.trim(),
    api_version: form.api_version.trim(),
    auth_scheme: form.auth_scheme.trim(),
    headers: form.headers.trim(),
    extra_config: form.extra_config.trim(),
    max_tokens: Math.min(Math.max(form.max_tokens || 64, 1), 256),
    timeout: Math.min(Math.max(form.timeout || 30, 1), 120)
  }
}

async function handleProbe() {
  if (!form.headers.trim() && !form.extra_config.trim() && !form.capabilities.trim()) {
    // no-op; keep JSON validation only for fields the user touched
  } else {
    try {
      for (const text of [form.headers, form.extra_config, form.capabilities]) {
        if (text.trim()) JSON.parse(text)
      }
    } catch {
      ElMessage.error(t('llmManagement.jsonInvalid'))
      return
    }
  }

  probing.value = true
  try {
    const resp = await probeLLM(buildProbePayload())
    if (!resp.ok) {
      ElMessage.error(resp.error || t('llmManagement.probeFailed'))
      return
    }
    form.provider = resp.provider || form.provider
    form.protocol = resp.protocol || form.protocol
    form.api_base = resp.api_base || form.api_base
    form.endpoint_path = resp.endpoint_path || form.endpoint_path
    form.api_version = resp.api_version || form.api_version
    form.auth_scheme = resp.auth_scheme || form.auth_scheme
    if (!form.model.trim() && resp.model) {
      form.model = resp.model
    }
    if (resp.capabilities) {
      form.capabilities = JSON.stringify(resp.capabilities, null, 2)
    }
    const label = protocolLabel(form.protocol)
    ElMessage.success(t('llmManagement.probeSuccess', { protocol: label }))
  } catch (error: any) {
    console.error('检测 LLM 协议失败:', error)
    ElMessage.error(error?.message || t('llmManagement.probeFailed'))
  } finally {
    probing.value = false
  }
}

async function handleSubmit() {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    submitting.value = true

    if (dialogMode.value === 'create') {
      await createLLM(buildCreatePayload())
      ElMessage.success(t('llmManagement.createSuccess'))
    } else {
      await updateLLM(buildUpdatePayload())
      ElMessage.success(t('llmManagement.saveSuccess'))
    }

    dialogVisible.value = false
    await handleRefresh()
  } catch (error: any) {
    if (error?.message && !String(error.message).includes('validate')) {
      console.error('提交 LLM 配置失败:', error)
      ElMessage.error(error.message || t('llmManagement.submitFailed'))
    }
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row: LLMInfo) {
  try {
    await ElMessageBox.confirm(
      t('llmManagement.deleteConfirm', { name: row.name }),
      t('llmManagement.deleteTitle'),
      {
        type: 'warning',
        confirmButtonText: t('common.delete'),
        cancelButtonText: t('common.cancel')
      }
    )

    await deleteLLM({ id: row.id })
    ElMessage.success(t('llmManagement.deleteSuccess'))
    await handleRefresh()
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') return
    console.error('删除 LLM 配置失败:', error)
    ElMessage.error(error?.message || t('llmManagement.deleteFailed'))
  }
}

async function handleSetDefault(row: LLMInfo) {
  try {
    await setDefaultLLM({ id: row.id })
    ElMessage.success(t('llmManagement.setDefaultSuccess', { name: row.name }))
    await handleRefresh()
  } catch (error: any) {
    console.error('设置默认 LLM 失败:', error)
    ElMessage.error(error?.message || t('llmManagement.setDefaultFailed'))
  }
}

onMounted(async () => {
  await handleRefresh()
})
</script>

<style scoped lang="scss">
.llm-management-page {
  padding: 20px;

  .page-card {
    border-radius: 16px;
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

  .default-panel {
    padding: 18px 20px;
    border-radius: 14px;
    background:
      linear-gradient(135deg, rgba(var(--el-color-primary-rgb, 69, 88, 200), 0.08), rgba(14, 165, 233, 0.04)),
      var(--el-fill-color-blank);
    border: 1px solid var(--el-border-color);
    box-shadow: var(--box-shadow-sm);
  }

  .default-panel-label {
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.08em;
    color: var(--el-color-primary);
    text-transform: uppercase;
    margin-bottom: 10px;
  }

  .default-panel-main {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .default-name {
    font-size: 20px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .default-meta {
    margin-top: 4px;
    color: var(--el-text-color-regular);
  }

  .default-tags {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .default-empty {
    color: var(--el-text-color-secondary);
  }

  .scope-tabs {
    margin-top: -6px;

    :deep(.el-tabs__header) {
      margin: 0;
    }

    :deep(.el-tabs__nav-wrap) {
      display: flex;
      justify-content: flex-start;
    }

    :deep(.el-tabs__nav-wrap::after),
    :deep(.el-tabs__active-bar) {
      display: none;
    }

    :deep(.el-tabs__nav-scroll) {
      display: flex;
      justify-content: flex-start;
    }

    :deep(.el-tabs__nav) {
      display: inline-flex;
      gap: 2px;
      padding: 3px;
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 10px;
      background: var(--el-fill-color-lighter);
    }

    :deep(.el-tabs__item) {
      height: 32px;
      padding: 0 12px;
      border-radius: 8px;
      color: var(--el-text-color-secondary);
      font-size: 13px;
      font-weight: 650;
    }

    :deep(.el-tabs__item.is-active) {
      background: var(--el-fill-color-blank);
      color: var(--el-color-primary);
      box-shadow: 0 1px 3px rgba(15, 23, 42, 0.08);
    }
  }

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
  }

  .toolbar-search {
    width: min(420px, 100%);
  }

  .toolbar-summary {
    color: var(--el-text-color-secondary);
    font-size: 14px;
  }

  .name-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .name-line {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .config-name {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .meta-line,
  .muted-line {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .form-suffix {
    margin-left: 12px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
}

@media (max-width: 768px) {
  .llm-management-page {
    padding: 12px;

    .card-header,
    .default-panel-main,
    .toolbar {
      flex-direction: column;
      align-items: stretch;
    }

    .header-actions {
      justify-content: flex-start;
    }

    .toolbar-search {
      width: 100%;
    }
  }
}
</style>
