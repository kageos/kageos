<template>
  <div class="llm-management-page">
    <el-card shadow="hover" class="page-card">
      <template #header>
        <div class="card-header">
          <div>
            <h2>LLM 管理</h2>
            <p class="header-desc">管理模型配置、默认模型与公开可见性。</p>
          </div>
          <div class="header-actions">
            <el-button :icon="Refresh" @click="handleRefresh">刷新</el-button>
            <el-button
              v-if="activeScope === 'mine'"
              type="primary"
              :icon="Plus"
              @click="handleCreate"
            >
              新建配置
            </el-button>
          </div>
        </div>
      </template>

      <div class="page-body">
        <div class="default-panel" v-loading="defaultLoading">
          <div class="default-panel-label">当前默认配置</div>
          <template v-if="defaultConfig">
            <div class="default-panel-main">
              <div>
                <div class="default-name">{{ defaultConfig.name }}</div>
                <div class="default-meta">
                  {{ defaultConfig.model }}
                </div>
              </div>
              <div class="default-tags">
                <el-tag type="warning">默认</el-tag>
                <el-tag :type="defaultConfig.visibility === 0 ? 'success' : 'info'">
                  {{ visibilityLabel(defaultConfig.visibility) }}
                </el-tag>
              </div>
            </div>
          </template>
          <template v-else>
            <div class="default-empty">当前没有可见的默认 LLM 配置。</div>
          </template>
        </div>

        <el-tabs v-model="activeScope" class="scope-tabs" @tab-change="handleScopeChange">
          <el-tab-pane label="我的配置" name="mine" />
          <el-tab-pane label="公开市场" name="market" />
        </el-tabs>

        <div class="toolbar">
          <el-input
            v-model="keyword"
            clearable
            placeholder="按名称、模型、API Base 或管理员过滤当前列表"
            class="toolbar-search"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <div class="toolbar-summary">
            共 {{ filteredConfigs.length }} 条
            <span v-if="keyword">（已过滤）</span>
          </div>
        </div>

        <el-table
          v-loading="loading"
          :data="filteredConfigs"
          stripe
          style="width: 100%"
          empty-text="暂无 LLM 配置"
        >
          <el-table-column label="配置" min-width="240">
            <template #default="{ row }">
              <div class="name-cell">
                <div class="name-line">
                  <span class="config-name">{{ row.name }}</span>
                  <el-tag v-if="row.is_default" type="warning" size="small">默认</el-tag>
                  <el-tag v-if="row.is_admin" type="primary" size="small">可管理</el-tag>
                </div>
                <div class="meta-line">{{ row.model }}</div>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="可见性" width="120" align="center">
            <template #default="{ row }">
              <el-tag :type="row.visibility === 0 ? 'success' : 'info'">
                {{ visibilityLabel(row.visibility) }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column label="API Key" width="110" align="center">
            <template #default="{ row }">
              <el-tag :type="row.has_api_key ? 'success' : 'info'">
                {{ row.has_api_key ? '已配置' : '未配置' }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column prop="api_base" label="API Base" min-width="220" show-overflow-tooltip />

          <el-table-column label="超时/Token" width="140" align="center">
            <template #default="{ row }">
              <div>{{ row.timeout }}s</div>
              <div class="muted-line">{{ row.max_tokens }} tokens</div>
            </template>
          </el-table-column>

          <el-table-column prop="admin" label="管理员" min-width="180" show-overflow-tooltip />

          <el-table-column label="更新时间" width="180">
            <template #default="{ row }">
              {{ formatDateTime(row.updated_at) }}
            </template>
          </el-table-column>

          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }">
              <el-button
                v-if="row.is_admin && !row.is_default"
                link
                type="warning"
                size="small"
                @click="handleSetDefault(row)"
              >
                设为默认
              </el-button>
              <el-button
                v-if="row.is_admin"
                link
                type="primary"
                size="small"
                @click="handleEdit(row)"
              >
                编辑
              </el-button>
              <el-button
                v-if="row.is_admin"
                link
                type="danger"
                size="small"
                @click="handleDelete(row)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="dialogMode === 'create' ? '新建 LLM 配置' : '编辑 LLM 配置'"
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
          <el-form-item label="配置名称" prop="name">
            <el-input v-model="form.name" placeholder="例如：OpenAI GPT-4.1" />
          </el-form-item>

          <el-form-item label="模型" prop="model">
            <el-input v-model="form.model" placeholder="例如：gpt-4.1、gpt-4o-mini" />
          </el-form-item>

          <el-form-item label="API Key">
            <el-input
              v-model="form.api_key"
              type="password"
              show-password
              placeholder="编辑时会加载当前值；清空后保存表示删除当前 Key"
            />
          </el-form-item>

          <el-form-item label="API Base">
            <el-input v-model="form.api_base" placeholder="例如：https://api.openai.com/v1" />
          </el-form-item>

          <el-form-item label="超时时间">
            <el-input-number v-model="form.timeout" :min="1" :max="3600" style="width: 180px" />
            <span class="form-suffix">秒</span>
          </el-form-item>

          <el-form-item label="最大 Token">
            <el-input-number v-model="form.max_tokens" :min="1" :max="1048576" style="width: 180px" />
          </el-form-item>

          <el-form-item label="可见性">
            <el-radio-group v-model="form.visibility">
              <el-radio :value="0">公开</el-radio>
              <el-radio :value="1">私有</el-radio>
            </el-radio-group>
          </el-form-item>

          <el-form-item label="管理员">
            <el-input
              v-model="form.admin"
              placeholder="多个用户名用英文逗号分隔；留空默认当前用户"
            />
          </el-form-item>

          <el-form-item label="设为默认">
            <el-switch v-model="form.is_default" />
          </el-form-item>

          <el-form-item label="额外配置" prop="extra_config">
            <el-input
              v-model="form.extra_config"
              type="textarea"
              :rows="6"
              placeholder='请输入 JSON，例如：{"temperature":0.2}'
            />
          </el-form-item>
        </el-form>
      </div>

      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ dialogMode === 'create' ? '创建' : '保存' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, Refresh, Search } from '@element-plus/icons-vue'
import dayjs from 'dayjs'
import {
  createLLM,
  deleteLLM,
  getDefaultLLM,
  getLLM,
  getLLMList,
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
  model: string
  api_key: string
  api_base: string
  timeout: number
  max_tokens: number
  extra_config: string
  is_default: boolean
  visibility: number
  admin: string
}

const DEFAULT_TIMEOUT = 300
const DEFAULT_MAX_TOKENS = 8196

const activeScope = ref<Scope>('mine')
const keyword = ref('')
const loading = ref(false)
const defaultLoading = ref(false)
const dialogLoading = ref(false)
const dialogVisible = ref(false)
const dialogMode = ref<DialogMode>('create')
const submitting = ref(false)

const configs = ref<LLMInfo[]>([])
const defaultConfig = ref<LLMInfo | null>(null)

const formRef = ref<FormInstance>()
const form = reactive<LLMFormState>(createDefaultForm())

const filteredConfigs = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return configs.value
  return configs.value.filter((item) => {
    return [
      item.name,
      item.model,
      item.api_base,
      item.admin
    ].some((field) => (field || '').toLowerCase().includes(q))
  })
})

const rules: FormRules<LLMFormState> = {
  name: [
    { required: true, message: '请输入配置名称', trigger: 'blur' }
  ],
  model: [
    { required: true, message: '请输入模型名称', trigger: 'blur' }
  ],
  extra_config: [
    {
      validator: (_rule, value: string, callback) => {
        const text = (value || '').trim()
        if (!text) {
          callback()
          return
        }
        try {
          JSON.parse(text)
          callback()
        } catch {
          callback(new Error('额外配置必须是合法 JSON'))
        }
      },
      trigger: 'blur'
    }
  ]
}

function createDefaultForm(): LLMFormState {
  return {
    id: null,
    name: '',
    model: '',
    api_key: '',
    api_base: '',
    timeout: DEFAULT_TIMEOUT,
    max_tokens: DEFAULT_MAX_TOKENS,
    extra_config: '',
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
  form.model = info.model || ''
  form.api_key = info.api_key || ''
  form.api_base = info.api_base || ''
  form.timeout = info.timeout || DEFAULT_TIMEOUT
  form.max_tokens = info.max_tokens || DEFAULT_MAX_TOKENS
  form.extra_config = info.extra_config || ''
  form.is_default = Boolean(info.is_default)
  form.visibility = typeof info.visibility === 'number' ? info.visibility : 0
  form.admin = info.admin || ''
}

function visibilityLabel(visibility: number) {
  return visibility === 1 ? '私有' : '公开'
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
    ElMessage.error(error?.message || '加载 LLM 配置失败')
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
    ElMessage.error(error?.message || '加载 LLM 配置详情失败')
    dialogVisible.value = false
  } finally {
    dialogLoading.value = false
  }
}

function buildCreatePayload(): LLMCreateReq {
  return {
    name: form.name.trim(),
    model: form.model.trim(),
    api_key: form.api_key.trim(),
    api_base: form.api_base.trim(),
    timeout: form.timeout,
    max_tokens: form.max_tokens,
    extra_config: form.extra_config.trim(),
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

async function handleSubmit() {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
    submitting.value = true

    if (dialogMode.value === 'create') {
      await createLLM(buildCreatePayload())
      ElMessage.success('LLM 配置创建成功')
    } else {
      await updateLLM(buildUpdatePayload())
      ElMessage.success('LLM 配置保存成功')
    }

    dialogVisible.value = false
    await handleRefresh()
  } catch (error: any) {
    if (error?.message && !String(error.message).includes('validate')) {
      console.error('提交 LLM 配置失败:', error)
      ElMessage.error(error.message || '提交失败')
    }
  } finally {
    submitting.value = false
  }
}

async function handleDelete(row: LLMInfo) {
  try {
    await ElMessageBox.confirm(
      `确定删除 LLM 配置「${row.name}」吗？`,
      '删除确认',
      {
        type: 'warning',
        confirmButtonText: '删除',
        cancelButtonText: '取消'
      }
    )

    await deleteLLM({ id: row.id })
    ElMessage.success('LLM 配置已删除')
    await handleRefresh()
  } catch (error: any) {
    if (error === 'cancel' || error === 'close') return
    console.error('删除 LLM 配置失败:', error)
    ElMessage.error(error?.message || '删除失败')
  }
}

async function handleSetDefault(row: LLMInfo) {
  try {
    await setDefaultLLM({ id: row.id })
    ElMessage.success(`已将「${row.name}」设为默认 LLM`)
    await handleRefresh()
  } catch (error: any) {
    console.error('设置默认 LLM 失败:', error)
    ElMessage.error(error?.message || '设置默认 LLM 失败')
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
