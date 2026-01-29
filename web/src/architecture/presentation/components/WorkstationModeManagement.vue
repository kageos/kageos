<!--
  WorkstationModeManagement - 智能工作台模式列表与配置（CRUD），右侧展示工具列表
-->
<template>
  <div class="workstation-mode-management">
    <header class="page-header">
      <div class="header-text">
        <h1 class="page-title">工作台模式</h1>
        <p class="page-desc">管理开发 / 修改 / 执行等模式，配置各模式下的工具与系统提示片段。</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="handleCreate">
        新增模式
      </el-button>
    </header>

    <div class="layout-with-tools">
      <!-- 左侧：模式列表 -->
      <el-card class="mode-card" shadow="hover">
        <template #header>
          <span class="card-title">模式列表</span>
          <span class="card-meta">共 {{ modeList.length }} 个</span>
        </template>
        <el-table
          v-loading="loading"
          :data="modeList"
          stripe
          class="mode-table"
          :header-cell-style="{ background: 'var(--el-fill-color-light)' }"
        >
          <el-table-column prop="code" label="编码" width="96" />
          <el-table-column prop="name" label="名称" width="110" />
          <el-table-column prop="description" label="描述" min-width="140" show-overflow-tooltip />
          <el-table-column label="工具数" width="88" align="center">
            <template #default="{ row }">
              <el-tag type="primary" size="small" effect="plain">
                {{ (row.tool_names && row.tool_names.length) || 0 }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="sort_order" label="排序" width="72" align="center" />
          <el-table-column label="内置" width="72" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.is_builtin" type="info" size="small" effect="plain">是</el-tag>
              <span v-else class="text-muted">—</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120" align="center">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="handleEdit(row)">编辑</el-button>
              <el-button
                link
                type="danger"
                size="small"
                :disabled="row.is_builtin"
                @click="handleDelete(row)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="!loading && modeList.length === 0" description="暂无模式，点击「新增模式」创建" :image-size="80" />
      </el-card>

      <!-- 右侧：工具列表（粘性） -->
      <el-card class="tools-card" shadow="hover">
        <template #header>
          <span class="card-title">工具列表</span>
          <span class="card-meta">共 {{ toolList.length }} 个</span>
        </template>
        <div v-loading="toolsLoading" class="tools-wrap">
          <div class="tools-list">
            <div
              v-for="(t, idx) in toolList"
              :key="t.name"
              class="tool-item"
              role="button"
              tabindex="0"
              @click="openToolDetail(t)"
              @keydown.enter="openToolDetail(t)"
            >
              <span class="tool-index">{{ idx + 1 }}</span>
              <div class="tool-body">
                <div class="tool-name">{{ t.name }}</div>
                <el-tooltip v-if="t.description" :content="t.description" placement="top" :show-after="300">
                  <div class="tool-desc">{{ t.description }}</div>
                </el-tooltip>
              </div>
            </div>
          </div>
          <el-empty v-if="!toolsLoading && toolList.length === 0" description="暂无工具" :image-size="64" />
        </div>
      </el-card>
    </div>

    <!-- 工具详情弹窗（函数定义形式 · 科幻动漫风） -->
    <el-dialog
      v-model="toolDetailVisible"
      :title="selectedTool ? selectedTool.name : '工具详情'"
      width="1200px"
      destroy-on-close
      class="tool-detail-dialog sci-fi-dialog"
      align-center
      @close="selectedTool = null"
    >
      <template v-if="selectedTool">
        <div class="tool-detail-inner">
          <div class="tool-detail-signature">
            <code class="sig-line">
              <template v-for="(seg, i) in toolSignatureSegments" :key="i">
                <span :class="['sig-token', 'sig-' + seg.kind]">{{ seg.text }}</span>
              </template>
            </code>
          </div>
          <p v-if="selectedTool.description" class="tool-detail-desc">{{ selectedTool.description }}</p>
          <div v-if="toolDetailParams.length > 0" class="tool-detail-params">
            <div class="tool-detail-params-title">入参</div>
            <el-table :data="toolDetailParams" class="tool-params-table" max-height="420">
              <el-table-column prop="name" label="参数名" width="220">
                <template #default="{ row }">
                  <code class="param-name">{{ row.name }}</code>
                  <el-tag v-if="row.required" type="danger" size="small" effect="plain" class="required-tag">必填</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="type" label="类型" width="120">
                <template #default="{ row }">
                  <span class="param-type">{{ row.type }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="description" label="说明" min-width="360" show-overflow-tooltip />
            </el-table>
          </div>
          <div class="tool-detail-params tool-detail-output">
            <div class="tool-detail-params-title">返回值</div>
            <el-table v-if="toolDetailOutputParams.length > 0" :data="toolDetailOutputParams" class="tool-params-table" max-height="280">
              <el-table-column prop="name" label="字段" width="220">
                <template #default="{ row }">
                  <code class="param-name">{{ row.name }}</code>
                </template>
              </el-table-column>
              <el-table-column prop="type" label="类型" width="120">
                <template #default="{ row }">
                  <span class="param-type">{{ row.type }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="description" label="说明" min-width="360" show-overflow-tooltip />
            </el-table>
            <p v-else class="tool-detail-no-output">当前未定义输出结构（返回值由工具实现决定）。LLM 标准工具通常只约定入参，返回内容多为自由文本。</p>
          </div>
        </div>
      </template>
      <template #footer>
        <el-button type="primary" size="large" class="sci-fi-btn" @click="toolDetailVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 新增/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="editingId ? '编辑模式' : '新增模式'"
      width="580px"
      destroy-on-close
      class="mode-dialog"
      @close="resetForm"
    >
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="120px" label-position="top">
        <div class="form-section">
          <div class="form-section-title">基本信息</div>
          <el-form-item label="编码" prop="code">
            <el-input v-model="form.code" placeholder="如 dev / modify / execute" :disabled="!!editingId" />
          </el-form-item>
          <el-form-item label="名称" prop="name">
            <el-input v-model="form.name" placeholder="模式显示名称" />
          </el-form-item>
          <el-form-item label="描述" prop="description">
            <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选" />
          </el-form-item>
        </div>
        <el-divider />
        <div class="form-section">
          <div class="form-section-title">系统提示</div>
          <el-form-item label="系统提示片段" prop="system_prompt_fragment">
            <el-input v-model="form.system_prompt_fragment" type="textarea" :rows="3" placeholder="追加到系统提示的片段，可选" />
          </el-form-item>
        </div>
        <el-divider />
        <div class="form-section">
          <div class="form-section-title">工具与智能体</div>
          <el-form-item label="启用工具" prop="tool_names">
            <el-select
              v-model="form.tool_names"
              multiple
              filterable
              placeholder="选择该模式启用的工具"
              style="width: 100%"
              :loading="toolNamesLoading"
            >
              <el-option v-for="name in allToolNames" :key="name" :value="name" :label="name" />
            </el-select>
          </el-form-item>
          <el-form-item label="绑定智能体" prop="agent_id">
            <div class="agent-selector-trigger">
              <el-input
                :model-value="selectedAgentDisplay"
                placeholder="请选择智能体（留空则使用默认 LLM）"
                readonly
                clearable
                @click="agentSelectVisible = true"
                @clear="handleClearAgent"
              >
                <template #append>
                  <el-button :icon="UserFilled" @click="agentSelectVisible = true">
                    选择智能体
                  </el-button>
                </template>
              </el-input>
              <div v-if="form.agent_id && selectedAgentDisplay" class="agent-selector-tag">
                <el-tag closable type="primary" effect="plain" @close="handleClearAgent">
                  {{ selectedAgentDisplay }}
                </el-tag>
              </div>
            </div>
            <AgentSelectDialog
              v-model="agentSelectVisible"
              @confirm="handleAgentSelectConfirm"
            />
          </el-form-item>
          <el-form-item label="排序" prop="sort_order">
            <el-input-number v-model="form.sort_order" :min="0" style="width: 120px" />
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ editingId ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus, UserFilled } from '@element-plus/icons-vue'
import AgentSelectDialog from '@/components/Agent/AgentSelectDialog.vue'
import { getAgent, type AgentInfo } from '@/api/agent'
import type { WorkspaceModeItem, CreateWorkspaceModeReq, UpdateWorkspaceModeReq, WorkspaceToolDef } from '@/api/workspace'
import {
  getWorkspaceModes,
  createWorkspaceMode,
  updateWorkspaceMode,
  deleteWorkspaceMode,
  listWorkspaceToolNames,
  getWorkspaceTools
} from '@/api/workspace'

/** 从 input_schema 解析出的参数项 */
interface ToolParamRow {
  name: string
  type: string
  required: boolean
  description: string
}

function parseSchemaProperties(schema: Record<string, unknown> | undefined): ToolParamRow[] {
  if (!schema || typeof schema !== 'object') return []
  const props = schema.properties as Record<string, Record<string, unknown>> | undefined
  const required = (schema.required as string[] | undefined) ?? []
  if (!props || typeof props !== 'object') return []
  return Object.entries(props).map(([name, p]) => {
    const t = (p?.type as string) ?? 'any'
    const typeLabel = { string: 'string', integer: 'number', number: 'number', boolean: 'boolean', array: 'array', object: 'object' }[t] ?? t
    return {
      name,
      type: typeLabel,
      required: required.includes(name),
      description: (p?.description as string) ?? ''
    }
  })
}

function parseInputSchema(schema: Record<string, unknown> | undefined): ToolParamRow[] {
  return parseSchemaProperties(schema)
}

function formatToolSignature(name: string, params: ToolParamRow[]): string {
  const args = params.map((p) => (p.required ? `${p.name}: ${p.type}` : `${p.name}?: ${p.type}`))
  return `${name}(${args.join(', ')})`
}

/** 签名高亮片段：fn=函数名 param=参数名 type=类型 punct=标点 */
interface SigSegment {
  kind: 'fn' | 'param' | 'type' | 'punct'
  text: string
}

function getSignatureSegments(name: string, params: ToolParamRow[]): SigSegment[] {
  const segs: SigSegment[] = []
  segs.push({ kind: 'fn', text: name })
  segs.push({ kind: 'punct', text: '(' })
  params.forEach((p, i) => {
    if (i > 0) segs.push({ kind: 'punct', text: ', ' })
    segs.push({ kind: 'param', text: p.name })
    if (!p.required) segs.push({ kind: 'punct', text: '?' })
    segs.push({ kind: 'punct', text: ': ' })
    segs.push({ kind: 'type', text: p.type })
  })
  segs.push({ kind: 'punct', text: ')' })
  return segs
}

const loading = ref(false)
const modeList = ref<WorkspaceModeItem[]>([])
const toolList = ref<WorkspaceToolDef[]>([])
const toolsLoading = ref(false)
const toolDetailVisible = ref(false)
const selectedTool = ref<WorkspaceToolDef | null>(null)

const toolDetailParams = computed(() =>
  selectedTool.value ? parseInputSchema(selectedTool.value.input_schema) : []
)
const toolDetailOutputParams = computed(() =>
  selectedTool.value ? parseSchemaProperties(selectedTool.value.output_schema) : []
)
const toolSignature = computed(() =>
  selectedTool.value ? formatToolSignature(selectedTool.value.name, toolDetailParams.value) : ''
)
const toolSignatureSegments = computed(() =>
  selectedTool.value ? getSignatureSegments(selectedTool.value.name, toolDetailParams.value) : []
)

function openToolDetail(tool: WorkspaceToolDef) {
  selectedTool.value = tool
  toolDetailVisible.value = true
}
const allToolNames = ref<string[]>([])
const toolNamesLoading = ref(false)
const dialogVisible = ref(false)
const submitLoading = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()
const agentSelectVisible = ref(false)
const selectedAgentName = ref('')

const selectedAgentDisplay = computed(() => {
  if (selectedAgentName.value) return selectedAgentName.value
  if (form.value.agent_id) return `ID: ${form.value.agent_id}`
  return ''
})

const form = ref<CreateWorkspaceModeReq & { id?: number }>({
  code: '',
  name: '',
  description: '',
  system_prompt_fragment: '',
  tool_names: [],
  agent_id: undefined,
  sort_order: 0
})

const formRules: FormRules = {
  code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }]
}

async function loadModes() {
  loading.value = true
  try {
    const res = await getWorkspaceModes({ page: 1, page_size: 200 })
    modeList.value = res.list ?? []
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : '加载模式列表失败'
    ElMessage.error(msg)
  } finally {
    loading.value = false
  }
}

async function loadToolNames() {
  toolNamesLoading.value = true
  try {
    allToolNames.value = await listWorkspaceToolNames()
  } catch {
    allToolNames.value = []
  } finally {
    toolNamesLoading.value = false
  }
}

async function loadTools() {
  toolsLoading.value = true
  try {
    toolList.value = await getWorkspaceTools()
  } catch {
    toolList.value = []
  } finally {
    toolsLoading.value = false
  }
}

function handleCreate() {
  editingId.value = null
  resetForm()
  loadToolNames()
  dialogVisible.value = true
}

async function handleEdit(row: WorkspaceModeItem) {
  editingId.value = row.id
  form.value = {
    code: row.code,
    name: row.name,
    description: row.description ?? '',
    system_prompt_fragment: row.system_prompt_fragment ?? '',
    tool_names: row.tool_names ? [...row.tool_names] : [],
    agent_id: row.agent_id ?? undefined,
    sort_order: row.sort_order ?? 0
  }
  selectedAgentName.value = ''
  if (row.agent_id) {
    try {
      const a = await getAgent({ id: row.agent_id }) as unknown as AgentInfo
      selectedAgentName.value = a?.name ?? ''
    } catch {
      selectedAgentName.value = ''
    }
  }
  loadToolNames()
  dialogVisible.value = true
}

function resetForm() {
  form.value = {
    code: '',
    name: '',
    description: '',
    system_prompt_fragment: '',
    tool_names: [],
    agent_id: undefined,
    sort_order: 0
  }
  selectedAgentName.value = ''
  formRef.value?.clearValidate()
}

function handleAgentSelectConfirm(agent: AgentInfo) {
  form.value.agent_id = agent.id
  selectedAgentName.value = agent.name
  agentSelectVisible.value = false
}

function handleClearAgent() {
  form.value.agent_id = undefined
  selectedAgentName.value = ''
}

async function handleSubmit() {
  await formRef.value?.validate().catch(() => {})
  submitLoading.value = true
  try {
    if (editingId.value) {
      const req: UpdateWorkspaceModeReq = {
        name: form.value.name,
        description: form.value.description ?? '',
        system_prompt_fragment: form.value.system_prompt_fragment ?? '',
        tool_names: form.value.tool_names?.length ? form.value.tool_names : undefined,
        agent_id: form.value.agent_id ?? null,
        sort_order: form.value.sort_order
      }
      await updateWorkspaceMode(editingId.value, req)
      ElMessage.success('保存成功')
    } else {
      await createWorkspaceMode({
        code: form.value.code,
        name: form.value.name,
        description: form.value.description,
        system_prompt_fragment: form.value.system_prompt_fragment,
        tool_names: form.value.tool_names,
        agent_id: form.value.agent_id ?? undefined,
        sort_order: form.value.sort_order ?? 0
      })
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await loadModes()
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : '操作失败'
    ElMessage.error(msg)
  } finally {
    submitLoading.value = false
  }
}

async function handleDelete(row: WorkspaceModeItem) {
  if (row.is_builtin) return
  try {
    await ElMessageBox.confirm(`确定删除模式「${row.name}」吗？`, '确认删除', {
      type: 'warning'
    })
  } catch {
    return
  }
  try {
    await deleteWorkspaceMode(row.id)
    ElMessage.success('已删除')
    await loadModes()
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : '删除失败'
    ElMessage.error(msg)
  }
}

onMounted(() => {
  loadModes()
  loadTools()
})
</script>

<style scoped>
/* 科幻动漫风变量 */
.workstation-mode-management {
  --ws-cyan: #22d3ee;
  --ws-cyan-dim: rgba(34, 211, 238, 0.35);
  --ws-violet: #a78bfa;
  --ws-violet-dim: rgba(167, 139, 250, 0.25);
  --ws-bg-dark: rgba(15, 23, 42, 0.92);
  --ws-glass: rgba(30, 41, 59, 0.6);
  --ws-glow: 0 0 20px var(--ws-cyan-dim), 0 0 40px var(--ws-violet-dim);
  padding: 24px 28px 32px;
  max-width: 1280px;
  margin: 0 auto;
  min-height: 100%;
}
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}
.header-text {
  flex: 1;
  min-width: 0;
}
.page-title {
  margin: 0 0 6px 0;
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
}
.page-desc {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
.layout-with-tools {
  display: flex;
  gap: 20px;
  align-items: flex-start;
}
.mode-card {
  flex: 1;
  min-width: 0;
}
.mode-card :deep(.el-card__header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  font-weight: 600;
  background: var(--el-fill-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.mode-card :deep(.el-card__body) {
  padding: 12px 18px 18px;
}
.mode-table {
  margin: 0;
}
.mode-table :deep(.el-table__row) {
  cursor: default;
}
.text-muted {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}
.tools-card {
  width: 400px;
  flex-shrink: 0;
  position: sticky;
  top: 24px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  overflow: hidden;
}
.tools-card :deep(.el-card__header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 20px;
  font-weight: 600;
  background: linear-gradient(135deg, var(--el-fill-color-lighter) 0%, var(--el-fill-color-light) 100%);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.tools-card :deep(.el-card__body) {
  padding: 16px 20px 20px;
}
.card-title {
  font-size: 16px;
  letter-spacing: 0.02em;
}
.card-meta {
  font-size: 13px;
  font-weight: 400;
  color: var(--el-text-color-secondary);
}
.tools-wrap {
  min-height: 140px;
}
.tools-list {
  max-height: calc(100vh - 280px);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.tools-list::-webkit-scrollbar {
  width: 8px;
}
.tools-list::-webkit-scrollbar-thumb {
  background: linear-gradient(180deg, var(--ws-cyan-dim), var(--ws-violet-dim));
  border-radius: 4px;
}
.tool-item {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 16px 18px;
  background: var(--el-fill-color-light);
  border-radius: 10px;
  border: 1px solid var(--el-border-color-lighter);
  transition: border-color 0.25s ease, box-shadow 0.25s ease, transform 0.2s ease;
  cursor: pointer;
}
.tool-item:hover {
  border-color: var(--ws-cyan-dim);
  box-shadow: 0 0 16px var(--ws-cyan-dim), 0 2px 12px rgba(0, 0, 0, 0.06);
  transform: translateY(-1px);
}
.tool-index {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  line-height: 28px;
  text-align: center;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  background: linear-gradient(145deg, var(--el-fill-color) 0%, var(--el-fill-color-dark) 100%);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}
.tool-body {
  flex: 1;
  min-width: 0;
}
.tool-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 6px;
  letter-spacing: 0.01em;
}
.tool-desc {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.55;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 表单对话框 */
.mode-dialog :deep(.el-dialog__body) {
  padding-top: 8px;
  max-height: 70vh;
  overflow-y: auto;
}
.form-section {
  margin-bottom: 4px;
}
.form-section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 12px;
}
.mode-dialog :deep(.el-divider) {
  margin: 16px 0;
}
.agent-selector-trigger {
  width: 100%;
}
.agent-selector-trigger .agent-selector-tag {
  margin-top: 8px;
}

/* 工具详情弹窗（科幻动漫风 · 函数定义） */
.sci-fi-dialog :deep(.el-overlay),
.sci-fi-dialog :deep(.el-overlay-dialog) {
  backdrop-filter: blur(8px);
}
.sci-fi-dialog :deep(.el-dialog) {
  border-radius: 16px;
  overflow: hidden;
  border: 1px solid var(--ws-cyan-dim);
  box-shadow: var(--ws-glow), 0 24px 48px rgba(0, 0, 0, 0.2);
  background: linear-gradient(180deg, var(--ws-bg-dark) 0%, rgba(15, 23, 42, 0.98) 100%);
}
.sci-fi-dialog :deep(.el-dialog__header) {
  padding: 22px 28px;
  border-bottom: 1px solid var(--ws-cyan-dim);
  background: var(--ws-glass);
}
.sci-fi-dialog :deep(.el-dialog__title) {
  font-family: ui-monospace, 'JetBrains Mono', 'SF Mono', Monaco, monospace;
  font-size: 20px;
  font-weight: 600;
  color: var(--ws-cyan);
  letter-spacing: 0.03em;
  text-shadow: 0 0 12px var(--ws-cyan-dim);
}
.sci-fi-dialog :deep(.el-dialog__headerbtn .el-dialog__close) {
  color: var(--el-text-color-secondary);
  font-size: 18px;
}
.sci-fi-dialog :deep(.el-dialog__body) {
  padding: 0;
  background: transparent;
}
.tool-detail-inner {
  padding: 28px 28px 24px;
  min-height: 200px;
}
.tool-detail-signature {
  padding: 22px 24px;
  background: rgba(0, 0, 0, 0.35);
  border-radius: 12px;
  border: 1px solid var(--ws-violet-dim);
  margin-bottom: 24px;
  box-shadow: inset 0 0 24px rgba(34, 211, 238, 0.06);
}
.tool-detail-signature .sig-line {
  font-family: ui-monospace, 'JetBrains Mono', 'SF Mono', Monaco, monospace;
  font-size: 18px;
  line-height: 1.7;
  word-break: break-all;
  letter-spacing: 0.02em;
  display: block;
}
/* 代码高亮：函数名 / 参数名 / 类型 / 标点 */
.tool-detail-signature .sig-token.sig-fn {
  color: #22d3ee;
  text-shadow: 0 0 10px rgba(34, 211, 238, 0.4);
  font-weight: 600;
}
.tool-detail-signature .sig-token.sig-param {
  color: #fbbf24;
  font-weight: 500;
}
.tool-detail-signature .sig-token.sig-type {
  color: #a78bfa;
  font-weight: 500;
  text-shadow: 0 0 8px rgba(167, 139, 250, 0.3);
}
.tool-detail-signature .sig-token.sig-punct {
  color: rgba(148, 163, 184, 0.9);
}
.tool-detail-desc {
  margin: 0 0 24px 0;
  font-size: 15px;
  color: var(--el-text-color-regular);
  line-height: 1.75;
  white-space: pre-wrap;
  padding: 0 2px;
}
.tool-detail-params {
  margin-top: 24px;
}
.tool-detail-params-title {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 14px;
  color: var(--el-text-color-primary);
  letter-spacing: 0.02em;
}
.tool-detail-output {
  margin-top: 24px;
}
.tool-detail-no-output {
  margin: 0;
  font-size: 14px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
  padding: 14px 16px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 8px;
  border: 1px dashed var(--el-border-color-lighter);
}
.tool-params-table :deep(.el-table) {
  --el-table-border-color: var(--el-border-color-lighter);
  --el-table-header-bg-color: var(--ws-glass);
  font-size: 14px;
}
.tool-params-table :deep(.el-table th.el-table__cell) {
  font-size: 14px;
  font-weight: 600;
  padding: 14px 12px;
}
.tool-params-table :deep(.el-table td.el-table__cell) {
  padding: 14px 12px;
}
.tool-detail-dialog .param-name {
  font-family: ui-monospace, 'JetBrains Mono', monospace;
  font-size: 14px;
  font-weight: 500;
  color: var(--ws-cyan);
  margin-right: 8px;
}
.tool-detail-dialog .required-tag {
  margin-left: 6px;
  font-size: 11px;
}
.tool-detail-dialog .param-type {
  font-size: 14px;
  color: var(--ws-violet);
  font-weight: 500;
}
.sci-fi-dialog :deep(.el-dialog__footer) {
  padding: 18px 28px 22px;
  border-top: 1px solid var(--ws-violet-dim);
  background: var(--ws-glass);
}
.sci-fi-btn {
  min-width: 100px;
  font-size: 14px;
  padding: 10px 24px;
}
</style>
