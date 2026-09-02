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
              <div class="default-config">
                <div
                  class="provider-logo provider-logo--large"
                  :style="providerBrandStyle(defaultConfig)"
                >
                  <img
                    v-if="providerBrand(defaultConfig).logo"
                    :src="providerBrand(defaultConfig).logo"
                    :alt="`${providerBrand(defaultConfig).name} logo`"
                  />
                  <el-icon v-else><Cpu /></el-icon>
                </div>
                <div>
                  <div class="default-name">{{ defaultConfig.name }}</div>
                  <div class="default-meta">
                    <span class="provider-name">{{ providerBrand(defaultConfig).name }}</span>
                    <span class="meta-divider">·</span>
                    <span>{{ defaultConfig.model }}</span>
                  </div>
                  <div class="default-source">
                    {{ providerLabel(defaultConfig.provider) }} / {{ protocolLabel(defaultConfig.protocol) }}
                    <template v-if="endpointHost(defaultConfig.api_base)">
                      <span class="meta-divider">·</span>
                      {{ endpointHost(defaultConfig.api_base) }}
                    </template>
                  </div>
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
          class="desktop-config-table"
          style="width: 100%"
          :empty-text="t('llmManagement.empty')"
        >
          <el-table-column :label="t('llmManagement.config')" min-width="350">
            <template #default="{ row }">
              <div class="name-cell">
                <div class="provider-logo" :style="providerBrandStyle(row)">
                  <img
                    v-if="providerBrand(row).logo"
                    :src="providerBrand(row).logo"
                    :alt="`${providerBrand(row).name} logo`"
                  />
                  <el-icon v-else><Cpu /></el-icon>
                </div>
                <div class="config-details">
                  <div class="name-line">
                    <span class="config-name">{{ row.name }}</span>
                    <el-tag v-if="row.is_default" type="warning" size="small" effect="light">{{ t('llmManagement.defaultTag') }}</el-tag>
                    <el-tag v-if="row.is_admin" type="primary" size="small" effect="plain">{{ t('llmManagement.manageable') }}</el-tag>
                  </div>
                  <div class="model-line">{{ row.model }}</div>
                  <div class="provider-line">
                    <span class="provider-name">{{ providerBrand(row).name }}</span>
                    <template v-if="endpointHost(row.api_base)">
                      <span class="meta-divider">·</span>
                      <span class="endpoint-host">{{ endpointHost(row.api_base) }}</span>
                    </template>
                  </div>
                </div>
              </div>
            </template>
          </el-table-column>

          <el-table-column :label="t('llmManagement.protocol')" width="170">
            <template #default="{ row }">
              <div class="protocol-cell">
                <span>{{ providerLabel(row.provider) }}</span>
                <span class="muted-line">{{ protocolLabel(row.protocol) }}</span>
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

          <el-table-column :label="t('llmManagement.connection')" width="120" align="center">
            <template #default="{ row }">
              <div :class="['connection-state', { 'is-ready': row.has_api_key }]">
                <span class="connection-dot" />
                {{ row.has_api_key ? t('llmManagement.configured') : t('llmManagement.notConfigured') }}
              </div>
            </template>
          </el-table-column>

          <el-table-column :label="t('llmManagement.timeoutToken')" width="150" align="center">
            <template #default="{ row }">
              <div>out {{ row.max_tokens > 0 ? row.max_tokens : `${t('llmManagement.auto')} ${row.effective_max_output_tokens}` }}</div>
              <div class="muted-line">ctx {{ row.effective_context_window }}</div>
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

        <div v-loading="loading" class="mobile-config-list">
          <el-empty v-if="!filteredConfigs.length" :description="t('llmManagement.empty')" />
          <article v-for="row in filteredConfigs" :key="row.id" class="mobile-config-card">
            <div class="mobile-config-header">
              <div class="provider-logo" :style="providerBrandStyle(row)">
                <img
                  v-if="providerBrand(row).logo"
                  :src="providerBrand(row).logo"
                  :alt="`${providerBrand(row).name} logo`"
                />
                <el-icon v-else><Cpu /></el-icon>
              </div>
              <div class="config-details">
                <div class="name-line">
                  <span class="config-name">{{ row.name }}</span>
                  <el-tag v-if="row.is_default" type="warning" size="small" effect="light">{{ t('llmManagement.defaultTag') }}</el-tag>
                </div>
                <div class="model-line">{{ row.model }}</div>
                <div class="provider-line">
                  <span class="provider-name">{{ providerBrand(row).name }}</span>
                  <template v-if="endpointHost(row.api_base)">
                    <span class="meta-divider">·</span>
                    <span class="endpoint-host">{{ endpointHost(row.api_base) }}</span>
                  </template>
                </div>
              </div>
            </div>

            <div class="mobile-config-meta">
              <div>
                <span class="mobile-meta-label">{{ t('llmManagement.protocol') }}</span>
                <strong>{{ providerLabel(row.provider) }}</strong>
                <span>{{ protocolLabel(row.protocol) }}</span>
              </div>
              <div>
                <span class="mobile-meta-label">{{ t('llmManagement.timeoutToken') }}</span>
                <strong>out {{ row.max_tokens }}</strong>
                <span>ctx {{ row.effective_context_window }}</span>
              </div>
            </div>

            <div class="mobile-config-footer">
              <div class="mobile-statuses">
                <el-tag :type="row.visibility === 0 ? 'success' : 'info'" size="small">
                  {{ visibilityLabel(row.visibility) }}
                </el-tag>
                <div :class="['connection-state', { 'is-ready': row.has_api_key }]">
                  <span class="connection-dot" />
                  {{ row.has_api_key ? t('llmManagement.configured') : t('llmManagement.notConfigured') }}
                </div>
              </div>
              <div v-if="row.is_admin" class="mobile-actions">
                <el-button
                  v-if="!row.is_default"
                  link
                  type="warning"
                  size="small"
                  @click="handleSetDefault(row)"
                >
                  {{ t('llmManagement.setDefault') }}
                </el-button>
                <el-button link type="primary" size="small" @click="handleEdit(row)">
                  {{ t('llmManagement.edit') }}
                </el-button>
                <el-button link type="danger" size="small" @click="handleDelete(row)">
                  {{ t('common.delete') }}
                </el-button>
              </div>
            </div>
          </article>
        </div>
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

          <el-form-item :label="t('llmManagement.visibility')">
            <el-radio-group v-model="form.visibility">
              <el-radio :value="1">{{ t('llmManagement.private') }}</el-radio>
              <el-radio :value="0">{{ t('llmManagement.public') }}</el-radio>
            </el-radio-group>
            <p class="form-tip form-tip--block">{{ t('llmManagement.visibilityHint') }}</p>
          </el-form-item>

          <el-form-item :label="t('llmManagement.setAsDefault')">
            <el-switch v-model="form.is_default" />
          </el-form-item>

          <el-collapse v-model="advancedPanels" class="advanced-settings">
            <el-collapse-item name="advanced">
              <template #title>
                <div class="advanced-title">
                  <span>{{ t('llmManagement.advancedSettings') }}</span>
                  <span class="advanced-title-desc">{{ t('llmManagement.advancedSettingsHint') }}</span>
                </div>
              </template>

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
                  <el-option v-for="option in authSchemeOptions" :key="option.value" :label="option.label" :value="option.value" />
                </el-select>
              </el-form-item>

              <el-form-item :label="t('llmManagement.timeout')">
                <el-input-number v-model="form.timeout" :min="1" :max="3600" style="width: 180px" />
                <span class="form-suffix">{{ t('llmManagement.seconds') }}</span>
              </el-form-item>

              <el-form-item :label="t('llmManagement.maxToken')">
                <div class="form-control-stack">
                  <el-input-number v-model="form.max_tokens" :min="0" :max="1048576" style="width: 180px" />
                  <p class="form-tip">{{ maxOutputTokenHint }}</p>
                </div>
              </el-form-item>

              <el-form-item :label="t('llmManagement.contextWindow')">
                <div class="form-control-stack">
                  <el-input-number v-model="form.context_window" :min="0" :max="10000000" style="width: 180px" />
                  <p class="form-tip">{{ contextWindowHint }}</p>
                </div>
              </el-form-item>

              <el-form-item :label="t('llmManagement.admin')">
                <el-input v-model="form.admin" :placeholder="t('llmManagement.adminPlaceholder')" />
              </el-form-item>

              <el-form-item :label="t('llmManagement.extraConfig')" prop="extra_config">
                <el-input v-model="form.extra_config" type="textarea" :rows="4" :placeholder="t('llmManagement.extraConfigPlaceholder')" />
              </el-form-item>

              <el-form-item :label="t('llmManagement.headers')" prop="headers">
                <div class="form-control-stack form-control-stack--wide">
                  <el-input v-model="form.headers" type="textarea" :rows="3" :placeholder="t('llmManagement.headersPlaceholder')" />
                  <p class="form-tip form-tip--warning">{{ t('llmManagement.headersSecurityHint') }}</p>
                </div>
              </el-form-item>

              <el-form-item :label="t('llmManagement.capabilities')" prop="capabilities">
                <el-input v-model="form.capabilities" type="textarea" :rows="3" :placeholder="t('llmManagement.capabilitiesPlaceholder')" />
              </el-form-item>
            </el-collapse-item>
          </el-collapse>
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
import { Connection, Cpu, Plus, Refresh, Search } from '@element-plus/icons-vue'
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
import {
  getLLMEndpointHost,
  resolveLLMProviderBrand,
  type LLMProviderBrandSource
} from '@/architecture/shared/assets/llmProviderBrands'

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
  detected_max_output_tokens: number
  detected_max_output_token_source: string
  context_window: number
  detected_context_window: number
  detected_context_window_source: string
  extra_config: string
  capabilities: string
  is_default: boolean
  visibility: number
  admin: string
}

const DEFAULT_TIMEOUT = 300
const DEFAULT_MAX_TOKENS = 0
const DEFAULT_AUTO_MAX_TOKENS = 32768
const DEFAULT_CONTEXT_WINDOW = 128000
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
const advancedPanels = ref<string[]>([])
const submitting = ref(false)
const probing = ref(false)

const configs = ref<LLMInfo[]>([])
const defaultConfig = ref<LLMInfo | null>(null)

const formRef = ref<FormInstance>()
const form = reactive<LLMFormState>(createDefaultForm())

const providerOptions = computed(() => [
  { value: 'openai', label: t('llmManagement.providerOpenAICompatible') },
  { value: 'anthropic', label: t('llmManagement.providerAnthropic') }
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
const contextWindowHint = computed(() => {
  if (form.context_window > 0) {
    return t('llmManagement.contextWindowManualHint', { count: form.context_window })
  }
  if (form.detected_context_window > 0) {
    return t('llmManagement.contextWindowDetectedHint', { count: form.detected_context_window })
  }
  return t('llmManagement.contextWindowDefaultHint', { count: DEFAULT_CONTEXT_WINDOW })
})
const maxOutputTokenHint = computed(() => {
  if (form.max_tokens > 0) {
    return t('llmManagement.maxTokenManualHint', { count: form.max_tokens })
  }
  if (form.detected_max_output_tokens > 0) {
    return t('llmManagement.maxTokenDetectedHint', { count: form.detected_max_output_tokens })
  }
  return t('llmManagement.maxTokenAutoHint', { count: DEFAULT_AUTO_MAX_TOKENS })
})

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
    detected_max_output_tokens: 0,
    detected_max_output_token_source: '',
    context_window: 0,
    detected_context_window: 0,
    detected_context_window_source: '',
    extra_config: '',
    capabilities: '',
    is_default: false,
    visibility: 1,
    admin: ''
  }
}

function resetForm() {
  Object.assign(form, createDefaultForm())
  advancedPanels.value = []
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
  form.max_tokens = info.max_tokens ?? DEFAULT_MAX_TOKENS
  form.detected_max_output_tokens = info.detected_max_output_tokens || 0
  form.detected_max_output_token_source = info.detected_max_output_token_source || ''
  form.context_window = info.context_window || 0
  form.detected_context_window = info.detected_context_window || 0
  form.detected_context_window_source = info.detected_context_window_source || ''
  form.extra_config = info.extra_config || ''
  form.capabilities = info.capabilities || ''
  form.is_default = Boolean(info.is_default)
  form.visibility = typeof info.visibility === 'number' ? info.visibility : 1
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
  return provider === 'anthropic'
    ? t('llmManagement.providerAnthropic')
    : t('llmManagement.providerOpenAICompatible')
}

function providerBrand(config: LLMProviderBrandSource) {
  return resolveLLMProviderBrand(config)
}

function providerBrandStyle(config: LLMProviderBrandSource) {
  const brand = providerBrand(config)
  return {
    '--provider-accent': brand.accent,
    '--provider-surface': brand.surface
  }
}

function endpointHost(apiBase?: string) {
  return getLLMEndpointHost(apiBase)
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
    detected_max_output_tokens: form.detected_max_output_tokens || 0,
    detected_max_output_token_source: form.detected_max_output_token_source,
    context_window: form.context_window || 0,
    detected_context_window: form.detected_context_window || 0,
    detected_context_window_source: form.detected_context_window_source,
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
    if (resp.context_window && resp.context_window_source !== 'default') {
      form.detected_context_window = resp.context_window
      form.detected_context_window_source = resp.context_window_source || ''
    } else {
      form.detected_context_window = 0
      form.detected_context_window_source = ''
    }
    if (resp.max_output_tokens && resp.max_output_token_source !== 'default') {
      form.detected_max_output_tokens = resp.max_output_tokens
      form.detected_max_output_token_source = resp.max_output_token_source || ''
    } else {
      form.detected_max_output_tokens = 0
      form.detected_max_output_token_source = ''
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
    padding: 20px 22px;
    border-radius: 14px;
    background:
      radial-gradient(circle at 88% 12%, rgba(var(--el-color-primary-rgb, 69, 88, 200), 0.12), transparent 32%),
      linear-gradient(135deg, rgba(var(--el-color-primary-rgb, 69, 88, 200), 0.07), rgba(14, 165, 233, 0.03)),
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

  .default-config {
    display: flex;
    align-items: center;
    gap: 14px;
    min-width: 0;
  }

  .default-name {
    font-size: 20px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .default-meta {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 5px;
    color: var(--el-text-color-regular);
  }

  .default-source {
    margin-top: 5px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
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

  :deep(.el-table) {
    --el-table-row-hover-bg-color: rgba(var(--el-color-primary-rgb, 69, 88, 200), 0.045);
    border-radius: 12px;
    overflow: hidden;
  }

  :deep(.el-table th.el-table__cell) {
    height: 46px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.02em;
  }

  :deep(.el-table td.el-table__cell) {
    padding: 13px 0;
  }

  .name-cell {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;
  }

  .provider-logo {
    --provider-accent: #64748b;
    --provider-surface: #f1f5f9;
    display: inline-flex;
    flex: 0 0 42px;
    align-items: center;
    justify-content: center;
    width: 42px;
    height: 42px;
    border: 1px solid color-mix(in srgb, var(--provider-accent) 16%, transparent);
    border-radius: 12px;
    background: var(--provider-surface);
    color: var(--provider-accent);

    img {
      width: 24px;
      height: 24px;
      object-fit: contain;
    }

    .el-icon {
      font-size: 21px;
    }
  }

  .provider-logo--large {
    flex-basis: 50px;
    width: 50px;
    height: 50px;
    border-radius: 14px;

    img {
      width: 28px;
      height: 28px;
    }
  }

  .config-details {
    min-width: 0;
  }

  .name-line {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .config-name {
    overflow: hidden;
    max-width: 240px;
    color: var(--el-text-color-primary);
    font-weight: 700;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .model-line {
    overflow: hidden;
    max-width: 290px;
    margin-top: 4px;
    color: var(--el-text-color-primary);
    font-family: var(--font-family-mono, 'JetBrains Mono', monospace);
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .provider-line {
    display: flex;
    align-items: center;
    gap: 5px;
    min-width: 0;
    margin-top: 4px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .provider-name {
    color: var(--el-text-color-regular);
    font-weight: 650;
  }

  .meta-divider {
    color: var(--el-border-color-darker);
  }

  .endpoint-host {
    overflow: hidden;
    max-width: 180px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .protocol-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
    color: var(--el-text-color-regular);
    font-size: 13px;
  }

  .muted-line {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .connection-state {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    font-weight: 600;
  }

  .connection-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--el-text-color-placeholder);
    box-shadow: 0 0 0 3px var(--el-fill-color-darker);
  }

  .connection-state.is-ready {
    color: var(--el-color-success);

    .connection-dot {
      background: var(--el-color-success);
      box-shadow: 0 0 0 3px var(--el-color-success-light-9);
    }
  }

  .mobile-config-list {
    display: none;
  }

  .form-suffix {
    margin-left: 12px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .form-control-stack {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
  }

  .form-tip {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }

  .form-tip--block {
    width: 100%;
    margin-top: 6px;
  }

  .form-tip--warning {
    color: var(--el-color-warning);
  }

  .form-control-stack--wide {
    width: 100%;
  }

  .advanced-settings {
    margin-top: 8px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .advanced-title {
    display: flex;
    align-items: center;
    gap: 10px;
    color: var(--el-text-color-primary);
    font-weight: 600;
  }

  .advanced-title-desc {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    font-weight: 400;
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

    .default-tags {
      margin-left: 64px;
    }

    .desktop-config-table {
      display: none;
    }

    .mobile-config-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .mobile-config-card {
      overflow: hidden;
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 14px;
      background: var(--el-fill-color-blank);
      box-shadow: 0 4px 16px rgba(15, 23, 42, 0.04);
    }

    .mobile-config-header {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 16px;
    }

    .mobile-config-meta {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 1px;
      border-top: 1px solid var(--el-border-color-lighter);
      border-bottom: 1px solid var(--el-border-color-lighter);
      background: var(--el-border-color-lighter);

      > div {
        display: flex;
        flex-direction: column;
        gap: 3px;
        padding: 11px 16px;
        background: var(--el-fill-color-light);
        color: var(--el-text-color-secondary);
        font-size: 11px;
      }

      strong {
        color: var(--el-text-color-primary);
        font-size: 12px;
        font-weight: 650;
      }
    }

    .mobile-meta-label {
      margin-bottom: 2px;
      color: var(--el-text-color-placeholder);
      font-size: 10px;
      font-weight: 700;
      letter-spacing: 0.06em;
      text-transform: uppercase;
    }

    .mobile-config-footer {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 12px;
      padding: 10px 14px;
    }

    .mobile-statuses,
    .mobile-actions {
      display: flex;
      align-items: center;
      gap: 10px;
    }

    .mobile-actions {
      gap: 2px;
    }

    .mobile-actions .el-button + .el-button {
      margin-left: 4px;
    }
  }
}

@media (max-width: 520px) {
  .llm-management-page {
    .default-tags {
      margin-left: 0;
    }

    .mobile-config-footer {
      align-items: flex-start;
      flex-direction: column;
    }

    .mobile-actions {
      width: 100%;
      justify-content: flex-end;
    }
  }
}
</style>
