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
        <div class="workflow-graph-panel">
          <div class="section-bar">
            <div>
              <h2>流程图</h2>
              <span>{{ workflowNodes.length }} 个节点，{{ workflowEdges.length }} 条边</span>
            </div>
            <el-button
              size="small"
              :icon="showDefinitionDebug ? Hide : View"
              @click="showDefinitionDebug = !showDefinitionDebug"
            >
              {{ showDefinitionDebug ? '隐藏 JSON' : '查看 JSON' }}
            </el-button>
          </div>

          <el-alert
            v-if="definitionParseError"
            type="error"
            :title="`定义 JSON 格式错误：${definitionParseError}`"
            :closable="false"
            show-icon
          />

          <div v-else-if="orderedWorkflowNodes.length === 0" class="empty-graph">
            <div class="empty-title">暂无节点</div>
            <div class="empty-copy">创建 workflow 后，这里会按节点顺序展示输入、输出和连线。</div>
          </div>

          <div v-else class="workflow-canvas-wrap">
            <div class="workflow-canvas" :style="graphCanvasStyle">
              <svg
                class="workflow-canvas-lines"
                :viewBox="`0 0 ${graphCanvasSize.width} ${graphCanvasSize.height}`"
                aria-hidden="true"
              >
                <defs>
                  <marker
                    id="workflow-arrow"
                    markerWidth="10"
                    markerHeight="10"
                    refX="8"
                    refY="5"
                    orient="auto"
                    markerUnits="strokeWidth"
                  >
                    <path d="M 0 0 L 10 5 L 0 10 z" />
                  </marker>
                </defs>
                <path
                  v-for="connection in graphConnections"
                  :key="connection.id"
                  :d="connection.path"
                />
              </svg>

              <article
                v-for="item in graphCanvasItems"
                :key="item.id"
                class="canvas-node"
                :class="`canvas-node--${item.kind}`"
                :style="graphNodeStyle(item)"
              >
                <div class="canvas-node-head">
                  <span class="canvas-node-dot"></span>
                  <div class="canvas-node-title">
                    <strong>{{ item.title }}</strong>
                    <code>{{ item.subtitle }}</code>
                  </div>
                  <el-tag size="small" effect="plain">{{ item.badge }}</el-tag>
                </div>

                <template v-if="item.kind === 'start'">
                  <div class="canvas-node-body">
                    <div
                      v-for="input in workflowInputs"
                      :key="input.code"
                      class="canvas-list-row"
                    >
                      <span>{{ input.title || input.code }}</span>
                      <el-tag size="small" :type="input.required ? 'warning' : 'info'" effect="plain">
                        {{ input.required ? '必填' : input.type || '输入' }}
                      </el-tag>
                    </div>
                    <div v-if="workflowInputs.length === 0" class="canvas-muted">无输入参数</div>
                  </div>
                </template>

                <template v-else-if="item.kind === 'step' && item.node">
                  <div v-if="item.node.ref" class="canvas-node-ref">{{ item.node.ref }}</div>
                  <div class="canvas-node-body">
                    <div
                      v-for="mapping in inputMappingsForNode(item.node)"
                      :key="mapping.field"
                      class="canvas-mapping-row"
                    >
                      <span>{{ mapping.field }}</span>
                      <el-tag size="small" :type="mapping.kind === 'ref' ? 'primary' : 'info'" effect="plain">
                        {{ mapping.kind === 'ref' ? '引用' : mapping.kind === 'const' ? '固定' : '表达式' }}
                      </el-tag>
                      <code>{{ mapping.value }}</code>
                    </div>
                    <div v-if="inputMappingsForNode(item.node).length === 0" class="canvas-muted">无显式输入映射</div>
                  </div>
                </template>

                <template v-else>
                  <div class="canvas-node-body">
                    <div
                      v-for="output in workflowOutputs"
                      :key="output.code"
                      class="canvas-output-row"
                    >
                      <span>{{ output.title || output.code }}</span>
                      <code>{{ output.value }}</code>
                    </div>
                    <div v-if="workflowOutputs.length === 0" class="canvas-muted">未声明最终输出</div>
                  </div>
                </template>
              </article>
            </div>
          </div>
        </div>

        <div v-if="showDefinitionDebug" class="workflow-json-panel">
          <div class="section-bar">
            <div>
              <h2>定义 JSON（调试）</h2>
              <span>草稿定义</span>
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
        </div>
      </section>

      <aside class="workflow-runner">
        <div class="section-bar">
          <div>
            <h2>运行输入</h2>
            <span>{{ workflowInputs.length ? 'Start Schema' : 'JSON Object' }}</span>
          </div>
        </div>
        <div v-if="workflowInputs.length" class="workflow-input-form">
          <div
            v-for="field in workflowInputs"
            :key="field.code"
            class="workflow-input-field"
          >
            <label>
              <span>{{ field.title || field.code }}</span>
              <el-tag size="small" :type="field.required ? 'warning' : 'info'" effect="plain">
                {{ field.required ? '必填' : field.type || field.widgetType || '可选' }}
              </el-tag>
            </label>

            <el-select
              v-if="isSelectField(field)"
              v-model="workflowInputValues[field.code]"
              clearable
              filterable
              class="workflow-input-control"
            >
              <el-option
                v-for="option in selectOptions(field)"
                :key="String(option.value)"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
            <el-switch
              v-else-if="isSwitchField(field)"
              v-model="workflowInputValues[field.code]"
            />
            <el-input-number
              v-else-if="isNumberField(field)"
              v-model="workflowInputValues[field.code]"
              class="workflow-input-control"
              controls-position="right"
            />
            <el-input
              v-else-if="isTextareaField(field)"
              v-model="workflowInputValues[field.code]"
              class="workflow-input-control"
              type="textarea"
              :autosize="{ minRows: 3, maxRows: 7 }"
            />
            <el-input
              v-else
              v-model="workflowInputValues[field.code]"
              class="workflow-input-control"
            />
          </div>
        </div>
        <el-input
          v-else
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
          <div v-if="lastRunOutputRows.length" class="workflow-output-form">
            <div
              v-for="row in lastRunOutputRows"
              :key="row.code"
              class="workflow-output-field"
            >
              <span>{{ row.title || row.code }}</span>
              <code>{{ compactValue(row.value) }}</code>
            </div>
          </div>
          <pre v-else-if="lastRun.run.output_json" class="json-output">{{ formatJson(lastRun.run.output_json) }}</pre>
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
import { Connection, DocumentChecked, Hide, Refresh, Upload, VideoPlay, View } from '@element-plus/icons-vue'
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

type WorkflowNode = {
  id: string
  name?: string
  type: string
  ref?: string
  schema?: Record<string, any>
  input?: Record<string, unknown>
}

type WorkflowEdge = {
  from: string
  to: string
}

type MappingItem = {
  field: string
  kind: 'ref' | 'const' | 'expr'
  value: string
}

type WorkflowInputItem = {
  code: string
  name?: string
  title?: string
  type?: string
  widgetType?: string
  widgetConfig?: Record<string, any>
  required: boolean
}

type WorkflowOutputItem = WorkflowInputItem & {
  value: string
}

type GraphCanvasItem = {
  id: string
  kind: 'start' | 'step' | 'output'
  title: string
  subtitle: string
  badge: string
  x: number
  y: number
  width: number
  height: number
  node?: WorkflowNode
}

const EMPTY_DEFINITION = {
  schema_version: 'workflow.v1',
  mode: 'graph',
  nodes: [
    {
      id: 'start',
      name: '开始',
      type: 'workflow.start',
      schema: {
        version: 1,
        type: 'form',
        form: {
          request: []
        }
      }
    },
    {
      id: 'output',
      name: '输出',
      type: 'workflow.output',
      schema: {
        version: 1,
        type: 'form',
        form: {
          response: []
        }
      },
      input: {}
    }
  ],
  edges: [{ from: 'start', to: 'output' }]
}

const loading = ref(false)
const saving = ref(false)
const publishing = ref(false)
const running = ref(false)
const showDefinitionDebug = ref(false)
const workflow = ref<WorkflowItem | null>(null)
const definitionText = ref(formatJson(EMPTY_DEFINITION))
const inputText = ref('{\n  \n}')
const workflowInputValues = ref<Record<string, any>>({})
const lastRun = ref<WorkflowRunDetail | null>(null)

const GRAPH_PADDING = 44
const START_NODE_WIDTH = 220
const STEP_NODE_WIDTH = 320
const OUTPUT_NODE_WIDTH = 250
const GRAPH_NODE_HEIGHT = 178
const GRAPH_GAP = 110
const GRAPH_TOP = 54

const parsedDefinition = computed<Record<string, any> | null>(() => {
  try {
    return JSON.parse(definitionText.value || '{}')
  } catch {
    return null
  }
})

const definitionParseError = computed(() => {
  try {
    JSON.parse(definitionText.value || '{}')
    return ''
  } catch (error: any) {
    return error?.message || '无法解析'
  }
})

const workflowNodes = computed<WorkflowNode[]>(() => {
  const nodes = parsedDefinition.value?.nodes
  return Array.isArray(nodes) ? nodes.map(toWorkflowNode) : []
})

const workflowEdges = computed<WorkflowEdge[]>(() => {
  const edges = parsedDefinition.value?.edges
  if (!Array.isArray(edges)) return []
  return edges
    .map((edge) => ({
      from: String(edge?.from || ''),
      to: String(edge?.to || '')
    }))
    .filter((edge) => edge.from && edge.to)
})

const orderedWorkflowNodes = computed(() => orderWorkflowNodes(workflowNodes.value, workflowEdges.value))

const startNode = computed(() => workflowNodes.value.find((node) => node.type === 'workflow.start'))

const outputNode = computed(() => workflowNodes.value.find((node) => node.type === 'workflow.output'))

const workflowOutputs = computed<WorkflowOutputItem[]>(() => {
  const node = outputNode.value
  if (!node) return []
  const responseFields = extractFormFields(node.schema, 'response')
  const mappings = node.input || {}
  if (responseFields.length === 0) {
    return Object.entries(mappings).map(([code, expr]) => ({
      code,
      title: code,
      required: false,
      value: expressionLabel(expr)
    }))
  }
  return responseFields.map((field) => ({
    ...field,
    value: expressionLabel(mappings[field.code])
  }))
})

const workflowInputs = computed<WorkflowInputItem[]>(() => {
  return extractFormFields(startNode.value?.schema, 'request')
})

const graphCanvasItems = computed<GraphCanvasItem[]>(() => {
  if (orderedWorkflowNodes.value.length === 0) return []

  const items: GraphCanvasItem[] = []
  let x = GRAPH_PADDING
  for (const node of orderedWorkflowNodes.value) {
    const kind = node.type === 'workflow.start' ? 'start' : node.type === 'workflow.output' ? 'output' : 'step'
    const width = kind === 'start' ? START_NODE_WIDTH : kind === 'output' ? OUTPUT_NODE_WIDTH : STEP_NODE_WIDTH
    const height = kind === 'step' ? GRAPH_NODE_HEIGHT : GRAPH_NODE_HEIGHT - 18
    items.push({
      id: node.id,
      kind,
      title: node.name || (kind === 'start' ? 'Start' : kind === 'output' ? 'Output' : node.id),
      subtitle: node.id,
      badge: kind === 'start' ? `${workflowInputs.value.length} inputs` : kind === 'output' ? `${workflowOutputs.value.length} outputs` : node.type,
      x,
      y: kind === 'step' ? GRAPH_TOP : GRAPH_TOP + 9,
      width,
      height,
      node
    })
    x += width + GRAPH_GAP
  }

  return items
})

const graphCanvasSize = computed(() => {
  const maxRight = Math.max(...graphCanvasItems.value.map((item) => item.x + item.width), 720)
  const maxBottom = Math.max(...graphCanvasItems.value.map((item) => item.y + item.height), GRAPH_TOP + GRAPH_NODE_HEIGHT)
  return {
    width: maxRight + GRAPH_PADDING,
    height: maxBottom + 70
  }
})

const graphCanvasStyle = computed(() => ({
  width: `${graphCanvasSize.value.width}px`,
  height: `${graphCanvasSize.value.height}px`
}))

const graphConnections = computed(() => {
  const itemByID = new Map(graphCanvasItems.value.map((item) => [item.id, item]))
  const connections: Array<{ id: string; path: string }> = []
  for (const edge of workflowEdges.value) {
    const from = itemByID.get(edge.from)
    const to = itemByID.get(edge.to)
    if (!from || !to) continue
    const fromX = from.x + from.width
    const fromY = from.y + from.height / 2
    const toX = to.x
    const toY = to.y + to.height / 2
    const curve = Math.max(48, Math.min(120, (toX - fromX) / 2))
    connections.push({
      id: `${from.id}-${to.id}`,
      path: `M ${fromX} ${fromY} C ${fromX + curve} ${fromY}, ${toX - curve} ${toY}, ${toX} ${toY}`
    })
  }
  return connections
})

const lastRunOutputRows = computed(() => {
  const output = parseOutputJSON(lastRun.value?.run?.output_json)
  if (!output || workflowOutputs.value.length === 0) return []
  return workflowOutputs.value.map((field) => ({
    code: field.code,
    title: field.title,
    value: output[field.code]
  }))
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

watch(
  workflowInputs,
  () => {
    syncWorkflowInputDefaults()
  },
  { immediate: true }
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
  const input = collectWorkflowInput()
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

function toWorkflowNode(value: any, index: number): WorkflowNode {
  const fallbackID = `node${index + 1}`
  const input = isRecord(value?.input) ? value.input : {}
  const schema = isRecord(value?.schema) ? value.schema : undefined
  return {
    id: String(value?.id || fallbackID),
    name: typeof value?.name === 'string' ? value.name : undefined,
    type: String(value?.type || 'unknown'),
    ref: typeof value?.ref === 'string' ? value.ref : undefined,
    schema,
    input
  }
}

function extractFormFields(schema: unknown, section: 'request' | 'response'): WorkflowInputItem[] {
  if (!isRecord(schema) || schema.type !== 'form' || !isRecord(schema.form)) return []
  const fields = schema.form[section]
  if (!Array.isArray(fields)) return []
  return fields
    .filter(isRecord)
    .map((field) => {
      const data = isRecord(field.data) ? field.data : {}
      const widget = isRecord(field.widget) ? field.widget : {}
      const widgetConfig = isRecord(widget.config) ? widget.config : {}
      const code = String(field.code || '').trim()
      const validation = typeof field.validation === 'string' ? field.validation : ''
      return {
        code,
        name: typeof field.name === 'string' ? field.name : undefined,
        title: typeof field.name === 'string' ? field.name : code,
        type: typeof data.type === 'string' ? data.type : undefined,
        widgetType: typeof widget.type === 'string' ? widget.type : undefined,
        widgetConfig,
        required: validation.split(',').some((rule) => rule.trim() === 'required')
      }
    })
    .filter((field) => field.code)
}

function orderWorkflowNodes(nodes: WorkflowNode[], edges: WorkflowEdge[]): WorkflowNode[] {
  if (nodes.length <= 1) return nodes

  const nodeByID = new Map(nodes.map((node) => [node.id, node]))
  const incoming = new Set(edges.map((edge) => edge.to))
  const outgoing = new Map<string, string[]>()
  for (const edge of edges) {
    if (!nodeByID.has(edge.from) || !nodeByID.has(edge.to)) continue
    const list = outgoing.get(edge.from) || []
    list.push(edge.to)
    outgoing.set(edge.from, list)
  }

  const ordered: WorkflowNode[] = []
  const seen = new Set<string>()
  let current: WorkflowNode | undefined = nodes.find((node) => !incoming.has(node.id)) || nodes[0]

  while (current && !seen.has(current.id)) {
    ordered.push(current)
    seen.add(current.id)
    const nextID = (outgoing.get(current.id) || []).find((id) => !seen.has(id))
    current = nextID ? nodeByID.get(nextID) : undefined
  }

  for (const node of nodes) {
    if (!seen.has(node.id)) ordered.push(node)
  }
  return ordered
}

function inputMappingsForNode(node: WorkflowNode): MappingItem[] {
  return Object.entries(node.input || {}).map(([field, expr]) => ({
    field,
    kind: expressionKind(expr),
    value: expressionLabel(expr)
  }))
}

function collectWorkflowInput(): Record<string, unknown> | null {
  if (workflowInputs.value.length === 0) {
    return parseJson(inputText.value, '运行输入')
  }

  const input: Record<string, unknown> = {}
  for (const field of workflowInputs.value) {
    const value = workflowInputValues.value[field.code]
    if (field.required && isEmptyInputValue(value)) {
      ElMessage.warning(`请填写 ${field.title || field.code}`)
      return null
    }
    if (!isEmptyInputValue(value) || typeof value === 'boolean') {
      input[field.code] = value
    }
  }
  return input
}

function syncWorkflowInputDefaults() {
  const next: Record<string, any> = {}
  for (const field of workflowInputs.value) {
    if (Object.prototype.hasOwnProperty.call(workflowInputValues.value, field.code)) {
      next[field.code] = workflowInputValues.value[field.code]
      continue
    }
    next[field.code] = defaultInputValue(field)
  }
  workflowInputValues.value = next
}

function defaultInputValue(field: WorkflowInputItem): any {
  const config = field.widgetConfig || {}
  if (Object.prototype.hasOwnProperty.call(config, 'render_default')) return config.render_default
  if (Object.prototype.hasOwnProperty.call(config, 'default')) return config.default
  if (isSwitchField(field)) return false
  return ''
}

function isEmptyInputValue(value: unknown) {
  return value === undefined || value === null || value === '' || (Array.isArray(value) && value.length === 0)
}

function isSelectField(field: WorkflowInputItem) {
  return field.widgetType === 'select'
}

function isSwitchField(field: WorkflowInputItem) {
  return field.widgetType === 'switch' || field.type === 'bool' || field.type === 'boolean'
}

function isNumberField(field: WorkflowInputItem) {
  return field.widgetType === 'number' || field.widgetType === 'float' || field.type === 'int' || field.type === 'float' || field.type === 'number'
}

function isTextareaField(field: WorkflowInputItem) {
  return field.widgetType === 'text_area'
}

function selectOptions(field: WorkflowInputItem): Array<{ label: string; value: string }> {
  const options = field.widgetConfig?.options
  if (!Array.isArray(options)) return []
  return options.map((option) => ({
    label: String(option),
    value: String(option)
  }))
}

function parseOutputJSON(raw: unknown): Record<string, any> | null {
  if (!raw) return null
  if (isRecord(raw)) return raw
  if (typeof raw !== 'string') return null
  try {
    const parsed = JSON.parse(raw)
    return isRecord(parsed) ? parsed : null
  } catch {
    return null
  }
}

function graphNodeStyle(item: GraphCanvasItem) {
  return {
    left: `${item.x}px`,
    top: `${item.y}px`,
    width: `${item.width}px`,
    minHeight: `${item.height}px`
  }
}

function expressionKind(expr: unknown): MappingItem['kind'] {
  if (isRecord(expr) && typeof expr.$ref === 'string') return 'ref'
  if (isRecord(expr) && Object.prototype.hasOwnProperty.call(expr, '$const')) return 'const'
  return 'expr'
}

function expressionLabel(expr: unknown): string {
  if (isRecord(expr) && typeof expr.$ref === 'string') return expr.$ref
  if (isRecord(expr) && Object.prototype.hasOwnProperty.call(expr, '$const')) {
    return compactValue(expr.$const)
  }
  return compactValue(expr)
}

function compactValue(value: unknown): string {
  if (typeof value === 'string') return value
  try {
    const text = JSON.stringify(value)
    return typeof text === 'string' ? text : String(value)
  } catch {
    return String(value)
  }
}

function isRecord(value: unknown): value is Record<string, any> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
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
  min-height: calc(100vh - 98px);
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

.workflow-graph-panel,
.workflow-json-panel {
  min-width: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
  padding: 14px;
}

.workflow-graph-panel {
  flex: 1;
  min-height: 640px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.workflow-json-panel {
  flex: 1;
  min-height: 360px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.workflow-canvas-wrap {
  flex: 1;
  min-height: 560px;
  overflow: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background:
    linear-gradient(var(--el-fill-color-light) 1px, transparent 1px),
    linear-gradient(90deg, var(--el-fill-color-light) 1px, transparent 1px),
    var(--el-bg-color-page);
  background-size: 22px 22px;
}

.workflow-canvas {
  position: relative;
}

.workflow-canvas-lines {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  overflow: visible;
  pointer-events: none;
}

.workflow-canvas-lines path {
  fill: none;
  stroke: #94a3b8;
  stroke-width: 2;
  marker-end: url("#workflow-arrow");
}

.workflow-canvas-lines marker path {
  fill: #94a3b8;
}

.canvas-node {
  position: absolute;
  z-index: 1;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  background: var(--el-bg-color);
  padding: 12px;
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.08);
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.canvas-node--start {
  border-color: rgba(14, 165, 233, 0.38);
}

.canvas-node--output {
  border-color: rgba(34, 197, 94, 0.36);
}

.canvas-node-head {
  min-width: 0;
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
}

.canvas-node-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #0f766e;
  box-shadow: 0 0 0 4px rgba(15, 118, 110, 0.12);
}

.canvas-node--start .canvas-node-dot {
  background: #0284c7;
  box-shadow: 0 0 0 4px rgba(2, 132, 199, 0.14);
}

.canvas-node--output .canvas-node-dot {
  background: #16a34a;
  box-shadow: 0 0 0 4px rgba(22, 163, 74, 0.14);
}

.canvas-node-title {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.canvas-node-title strong {
  color: var(--el-text-color-primary);
  font-size: 14px;
  line-height: 1.35;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.canvas-node-title code,
.canvas-node-ref,
.canvas-mapping-row code,
.canvas-output-row code {
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
}

.canvas-node-title code {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.canvas-node-ref {
  padding: 7px 8px;
  border-radius: 6px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.45;
  word-break: break-all;
}

.canvas-node-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.canvas-list-row,
.canvas-output-row,
.canvas-mapping-row {
  min-width: 0;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.canvas-list-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
}

.canvas-mapping-row {
  display: grid;
  grid-template-columns: minmax(64px, 0.7fr) auto minmax(0, 1.2fr);
}

.canvas-output-row {
  display: grid;
  grid-template-columns: minmax(76px, 0.7fr) minmax(0, 1fr);
}

.canvas-list-row span,
.canvas-output-row span,
.canvas-mapping-row span {
  min-width: 0;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.canvas-output-row code,
.canvas-mapping-row code {
  min-width: 0;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.canvas-muted {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.empty-graph {
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  padding: 28px 16px;
  text-align: center;
  background: var(--el-bg-color);
}

.empty-title {
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
}

.empty-copy {
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
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

.run-summary {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px;
  background: var(--el-fill-color-blank);
}

.workflow-input-form,
.workflow-output-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.workflow-input-form {
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}

.workflow-input-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.workflow-input-field label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--el-text-color-primary);
  font-size: 12px;
  font-weight: 600;
}

.workflow-input-control {
  width: 100%;
}

.workflow-output-field {
  display: grid;
  grid-template-columns: minmax(92px, 0.45fr) minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  font-size: 12px;
}

.workflow-output-field span {
  color: var(--el-text-color-secondary);
}

.workflow-output-field code {
  min-width: 0;
  color: var(--el-text-color-primary);
  white-space: pre-wrap;
  word-break: break-word;
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
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
