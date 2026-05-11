<template>
  <div class="workflow-view" v-loading="loading">
    <header class="workflow-header">
      <div class="workflow-title-group">
        <div class="workflow-kicker">
          <el-icon><Connection /></el-icon>
          <span>WORKFLOW</span>
        </div>
        <h1>{{ node.name || node.code || '未命名工作流' }}</h1>
        <div class="workflow-path">{{ node.full_code_path }}</div>
      </div>
      <div class="workflow-actions">
        <el-tag :type="statusTagType" effect="plain">{{ statusText }}</el-tag>
        <el-button :icon="Refresh" @click="loadWorkflow">刷新</el-button>
        <el-button :icon="DocumentChecked" :loading="saving" @click="handleSave">保存草稿</el-button>
        <el-button type="primary" :icon="Upload" :loading="publishing" @click="handlePublish">发布</el-button>
        <el-button type="success" :icon="VideoPlay" :loading="running" @click="handleRun">运行</el-button>
      </div>
    </header>

    <main class="workflow-main">
      <section class="workflow-editor">
        <div class="section-bar">
          <div>
            <h2>定义 JSON</h2>
            <span>{{ workflowNodes.length }} 个节点，{{ workflowEdges.length }} 条边</span>
          </div>
        </div>
        <el-input
          v-model="definitionText"
          class="definition-editor"
          type="textarea"
          resize="none"
          spellcheck="false"
          :autosize="false"
        />
        <div class="node-chain" v-if="workflowNodes.length > 0">
          <div
            v-for="item in workflowNodes"
            :key="item.id"
            class="node-pill"
          >
            <span class="node-id">{{ item.id }}</span>
            <span class="node-name">{{ item.name || item.type }}</span>
            <span class="node-type">{{ item.type }}</span>
          </div>
        </div>
      </section>

      <aside class="workflow-runner">
        <div class="section-bar">
          <div>
            <h2>运行输入</h2>
            <span>JSON Object</span>
          </div>
        </div>
        <el-input
          v-model="inputText"
          class="input-editor"
          type="textarea"
          resize="none"
          spellcheck="false"
        />

        <div v-if="lastRun?.run" class="run-summary">
          <div class="run-summary-head">
            <span>Run #{{ lastRun.run.id }}</span>
            <el-tag size="small" :type="runStatusTagType(lastRun.run.status)">
              {{ lastRun.run.status }}
            </el-tag>
          </div>
          <pre v-if="lastRun.run.output_json" class="json-output">{{ formatJson(lastRun.run.output_json) }}</pre>
          <el-alert
            v-if="lastRun.run.error_message"
            type="error"
            :title="lastRun.run.error_message"
            :closable="false"
          />
        </div>

        <el-table
          v-if="lastRun?.steps?.length"
          :data="lastRun.steps"
          size="small"
          class="step-table"
        >
          <el-table-column prop="step_name" label="步骤" min-width="120" />
          <el-table-column prop="node_type" label="类型" width="112" />
          <el-table-column prop="status" label="状态" width="96">
            <template #default="{ row }">
              <el-tag size="small" :type="runStatusTagType(row.status)">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="duration_millis" label="耗时" width="90">
            <template #default="{ row }">{{ row.duration_millis }}ms</template>
          </el-table-column>
        </el-table>
      </aside>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection, DocumentChecked, Refresh, Upload, VideoPlay } from '@element-plus/icons-vue'
import {
  createWorkflow,
  getWorkflowByPath,
  publishWorkflow,
  runWorkflow,
  updateWorkflow,
  type WorkflowItem,
  type WorkflowRunDetail,
  type WorkflowRunStatus
} from '@/api/workflow'
import type { ServiceTree } from '@/types'

const props = defineProps<{
  node: ServiceTree
}>()

const EMPTY_DEFINITION = {
  schema_version: 'workflow.v1',
  mode: 'sequence',
  inputs: {},
  triggers: [{ type: 'manual' }],
  nodes: [],
  edges: [],
  outputs: {}
}

const loading = ref(false)
const saving = ref(false)
const publishing = ref(false)
const running = ref(false)
const workflow = ref<WorkflowItem | null>(null)
const definitionText = ref(formatJson(EMPTY_DEFINITION))
const inputText = ref('{\n  \n}')
const lastRun = ref<WorkflowRunDetail | null>(null)

const parsedDefinition = computed<Record<string, any> | null>(() => {
  try {
    return JSON.parse(definitionText.value || '{}')
  } catch {
    return null
  }
})

const workflowNodes = computed<Array<{ id: string; name?: string; type: string }>>(() => {
  const nodes = parsedDefinition.value?.nodes
  return Array.isArray(nodes) ? nodes : []
})

const workflowEdges = computed<Array<{ from: string; to: string }>>(() => {
  const edges = parsedDefinition.value?.edges
  return Array.isArray(edges) ? edges : []
})

const statusText = computed(() => {
  if (!workflow.value) return '未初始化'
  if (workflow.value.status === 'enabled') return '已发布'
  if (workflow.value.status === 'disabled') return '已停用'
  return '草稿'
})

const statusTagType = computed(() => {
  if (workflow.value?.status === 'enabled') return 'success'
  if (workflow.value?.status === 'disabled') return 'info'
  return 'warning'
})

watch(
  () => props.node.full_code_path,
  () => {
    void loadWorkflow()
  }
)

onMounted(() => {
  void loadWorkflow()
})

async function loadWorkflow() {
  if (!props.node.full_code_path) return

  loading.value = true
  lastRun.value = null
  try {
    let item = await getWorkflowByPath(props.node.full_code_path)
    if (!item) {
      item = await createWorkflow({
        name: props.node.name || props.node.code || '未命名工作流',
        description: props.node.description || '',
        app_id: props.node.app_id,
        full_code_path: props.node.full_code_path
      })
    }
    workflow.value = item
    definitionText.value = formatJson(item.draft_definition_json || EMPTY_DEFINITION)
  } catch (error: any) {
    ElMessage.error(error?.message || '加载工作流失败')
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  const item = workflow.value
  if (!item) {
    ElMessage.warning('工作流尚未初始化')
    return
  }
  const definition = parseJson(definitionText.value, '定义 JSON')
  if (!definition) return

  saving.value = true
  try {
    workflow.value = await updateWorkflow(item.id, {
      name: props.node.name || item.name,
      description: props.node.description || item.description,
      app_id: props.node.app_id || item.app_id,
      full_code_path: props.node.full_code_path || item.full_code_path,
      definition
    })
    definitionText.value = formatJson(workflow.value.draft_definition_json || definition)
    ElMessage.success('草稿已保存')
  } catch (error: any) {
    ElMessage.error(error?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handlePublish() {
  await publishCurrentDefinition(false)
}

async function publishCurrentDefinition(silent: boolean) {
  const item = workflow.value
  if (!item) {
    ElMessage.warning('工作流尚未初始化')
    return false
  }
  const definition = parseJson(definitionText.value, '定义 JSON')
  if (!definition) return false

  publishing.value = true
  try {
    await publishWorkflow(item.id, { definition })
    await loadWorkflow()
    if (!silent) {
      ElMessage.success('工作流已发布')
    }
    return true
  } catch (error: any) {
    ElMessage.error(error?.message || '发布失败')
    return false
  } finally {
    publishing.value = false
  }
}

async function handleRun() {
  const item = workflow.value
  if (!item) {
    ElMessage.warning('工作流尚未初始化')
    return
  }
  const input = parseJson(inputText.value, '运行输入')
  if (!input) return

  running.value = true
  try {
    let runnable: WorkflowItem = item
    if (runnable.status !== 'enabled') {
      const published = await publishCurrentDefinition(true)
      if (!published || !workflow.value) return
      runnable = workflow.value
    }
    lastRun.value = await runWorkflow(runnable.id, { input })
    ElMessage.success(lastRun.value.run.status === 'success' ? '运行成功' : '运行完成')
  } catch (error: any) {
    ElMessage.error(error?.message || '运行失败')
  } finally {
    running.value = false
  }
}

function parseJson(text: string, label: string): Record<string, unknown> | null {
  try {
    const parsed = JSON.parse(text || '{}')
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      ElMessage.warning(`${label} 必须是 JSON Object`)
      return null
    }
    return parsed
  } catch (error: any) {
    ElMessage.error(`${label} 格式错误: ${error?.message || '无法解析'}`)
    return null
  }
}

function formatJson(value: unknown): string {
  return JSON.stringify(value || {}, null, 2)
}

function runStatusTagType(status: WorkflowRunStatus | string) {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'canceled') return 'info'
  return 'warning'
}
</script>

<style scoped>
.workflow-view {
  min-height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
}

.workflow-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  padding: 24px 28px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}

.workflow-title-group {
  min-width: 0;
}

.workflow-kicker {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: #0f766e;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
  margin-bottom: 8px;
}

.workflow-title-group h1 {
  margin: 0;
  font-size: 24px;
  line-height: 1.25;
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.workflow-path {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  word-break: break-all;
}

.workflow-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
}

.workflow-main {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(360px, 420px);
  gap: 1px;
  background: var(--el-border-color-lighter);
}

.workflow-editor,
.workflow-runner {
  min-width: 0;
  min-height: 0;
  background: var(--el-bg-color);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.section-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.section-bar h2 {
  margin: 0;
  font-size: 15px;
  line-height: 1.4;
  color: var(--el-text-color-primary);
}

.section-bar span {
  display: block;
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.definition-editor,
.input-editor {
  flex: 1;
  min-height: 260px;
}

.definition-editor :deep(.el-textarea__inner),
.input-editor :deep(.el-textarea__inner) {
  height: 100%;
  min-height: 260px;
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
  font-size: 13px;
  line-height: 1.55;
  border-radius: 8px;
}

.node-chain {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.node-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  border: 1px solid rgba(15, 118, 110, 0.24);
  border-radius: 8px;
  background: rgba(15, 118, 110, 0.06);
  max-width: 100%;
}

.node-id,
.node-type {
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
  font-size: 12px;
}

.node-id {
  color: #0f766e;
  font-weight: 700;
}

.node-name {
  min-width: 0;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-type {
  color: var(--el-text-color-secondary);
}

.run-summary {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px;
  background: var(--el-fill-color-blank);
}

.run-summary-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
  font-weight: 600;
}

.json-output {
  margin: 0;
  max-height: 220px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-regular);
}

.step-table {
  width: 100%;
}

@media (max-width: 980px) {
  .workflow-header {
    flex-direction: column;
  }

  .workflow-actions {
    justify-content: flex-start;
  }

  .workflow-main {
    grid-template-columns: 1fr;
  }

  .workflow-runner {
    min-height: 520px;
  }
}
</style>
