<template>
  <div class="role-handoff-card">
    <div class="role-handoff-head">
      <div class="role-handoff-title">
        <span>角色交接</span>
        <span v-if="roleLabel" class="role-handoff-role">{{ roleLabel }}</span>
      </div>
      <el-tag size="small" :type="statusType">{{ statusLabel }}</el-tag>
    </div>

    <div class="role-handoff-section role-handoff-directory">
      <div class="role-handoff-label">执行目录</div>
      <code>{{ executeDirectory || '未指定' }}</code>
    </div>

    <div v-if="metaBadges.length" class="role-handoff-meta">
      <span v-for="badge in metaBadges" :key="badge">{{ badge }}</span>
    </div>

    <div class="role-handoff-grid">
      <section class="role-handoff-section">
        <div class="role-handoff-label">任务上下文</div>
        <ul v-if="taskContext.length">
          <li v-for="(item, idx) in taskContext" :key="`ctx-${idx}`">{{ item }}</li>
        </ul>
        <span v-else class="role-handoff-empty">未提供</span>
      </section>

      <section class="role-handoff-section">
        <div class="role-handoff-label">关键信息</div>
        <ul v-if="keyInformation.length">
          <li v-for="(item, idx) in keyInformation" :key="`key-${idx}`">{{ item }}</li>
        </ul>
        <span v-else class="role-handoff-empty">未提供</span>
      </section>

      <section class="role-handoff-section role-handoff-section--wide">
        <div class="role-handoff-label">参考资料</div>
        <ul v-if="references.length" class="role-handoff-references">
          <li v-for="(item, idx) in references" :key="`ref-${idx}`">
            <code>{{ item }}</code>
          </li>
        </ul>
        <span v-else class="role-handoff-empty">未提供</span>
      </section>
    </div>

    <div v-if="hasRuntimeContract" class="role-handoff-grid role-handoff-runtime">
      <section class="role-handoff-section">
        <div class="role-handoff-label">角色 SOP</div>
        <ul v-if="runtimeSop.length">
          <li v-for="(item, idx) in runtimeSop" :key="`sop-${idx}`">{{ item }}</li>
        </ul>
        <span v-else class="role-handoff-empty">未提供</span>
      </section>

      <section class="role-handoff-section">
        <div class="role-handoff-label">完成标准</div>
        <ul v-if="runtimeDoneWhen.length">
          <li v-for="(item, idx) in runtimeDoneWhen" :key="`done-${idx}`">{{ item }}</li>
        </ul>
        <span v-else class="role-handoff-empty">未提供</span>
      </section>

      <section v-if="runtimeHooks.length" class="role-handoff-section role-handoff-section--wide">
        <div class="role-handoff-label">生命周期 Hook</div>
        <ul>
          <li v-for="(item, idx) in runtimeHooks" :key="`hook-${idx}`">{{ item }}</li>
        </ul>
      </section>
    </div>

    <div v-if="hasAppCapabilities" class="role-handoff-grid role-handoff-capabilities">
      <section class="role-handoff-section role-handoff-section--wide">
        <div class="role-handoff-label">当前应用能力</div>
        <div v-if="appCapabilitySummary" class="role-handoff-capability-summary">{{ appCapabilitySummary }}</div>
        <ul v-if="appCapabilityGuidance.length" class="role-handoff-capability-guidance">
          <li v-for="(item, idx) in appCapabilityGuidance" :key="`cap-guide-${idx}`">{{ item }}</li>
        </ul>
        <ul v-if="appCapabilityFunctions.length" class="role-handoff-capability-functions">
          <li v-for="fn in appCapabilityFunctions" :key="fn.fullCodePath || fn.name">
            <div class="role-handoff-capability-name">
              <span>{{ fn.title }}</span>
              <code v-if="fn.fullCodePath">{{ fn.fullCodePath }}</code>
            </div>
            <div class="role-handoff-capability-meta">{{ fn.meta }}</div>
            <div v-if="fn.schemaSummary.length" class="role-handoff-capability-schema">
              {{ fn.schemaSummary.join(' / ') }}
            </div>
          </li>
        </ul>
        <span v-if="!appCapabilityFunctions.length && !appCapabilityGuidance.length" class="role-handoff-empty">未提供</span>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WorkspaceChatToolCallSummary } from '@/architecture/presentation/context/api/workspace'

type UnknownRecord = Record<string, unknown>

interface HandoffBlock {
  execute_directory?: string
  task_context?: unknown
  key_information?: unknown
  references?: unknown
}

interface RuntimeHook {
  id?: unknown
  stage?: unknown
  purpose?: unknown
  produces?: unknown
}

interface CapabilityFunctionView {
  title: string
  name: string
  fullCodePath: string
  meta: string
  schemaSummary: string[]
}

const props = defineProps<{
  toolCall: WorkspaceChatToolCallSummary
}>()

const args = computed<UnknownRecord>(() => parseJSONRecord(props.toolCall.arguments))
const resultData = computed<UnknownRecord>(() => asRecord(props.toolCall.result_data))
const resultHandoff = computed<HandoffBlock>(() => asRecord(resultData.value.handoff) as HandoffBlock)
const runtimeContract = computed<UnknownRecord>(() => asRecord(resultData.value.runtime_contract))
const appCapabilities = computed<UnknownRecord>(() => asRecord(resultData.value.app_capabilities))

const executeDirectory = computed(() =>
  firstString(
    resultHandoff.value.execute_directory,
    resultData.value.execute_directory,
    resultData.value.directory,
    args.value.execute_directory,
    args.value.directory
  )
)

const taskContext = computed(() =>
  firstList(
    resultHandoff.value.task_context,
    args.value.task_context,
    args.value.task_summary,
    args.value.user_input
  )
)

const keyInformation = computed(() =>
  firstList(
    resultHandoff.value.key_information,
    args.value.key_information
  )
)

const references = computed(() => uniqueStrings([
  ...firstList(resultHandoff.value.references, args.value.references),
  ...firstList(resultData.value.reference_docs, args.value.reference_docs),
  ...firstList(resultData.value.reference_files, args.value.reference_files),
  ...firstList(resultData.value.required_docs),
]))

const runtimeSop = computed(() => firstList(runtimeContract.value.sop))
const runtimeDoneWhen = computed(() => firstList(runtimeContract.value.done_when))
const runtimeHooks = computed(() => normalizeHooks(runtimeContract.value.hooks))
const hasRuntimeContract = computed(() =>
  runtimeSop.value.length > 0 || runtimeDoneWhen.value.length > 0 || runtimeHooks.value.length > 0
)
const appCapabilityGuidance = computed(() => firstList(appCapabilities.value.guidance))
const appCapabilityFunctions = computed(() => normalizeCapabilityFunctions(appCapabilities.value.functions))
const appCapabilitySummary = computed(() => formatAppCapabilitySummary(appCapabilities.value))
const hasAppCapabilities = computed(() =>
  Boolean(appCapabilitySummary.value || appCapabilityGuidance.value.length > 0 || appCapabilityFunctions.value.length > 0)
)

const metaBadges = computed(() => {
  const badges: string[] = []
  const requiredCount = arrayLength(resultData.value.required_docs)
  const loadedCount = arrayLength(resultData.value.loaded_docs)
  if (requiredCount > 0) badges.push(`文档包 ${requiredCount} 项`)
  if (loadedCount > 0) badges.push(`已加载 ${loadedCount} 项`)
  if (references.value.some((item) => /agent_app_(prd|build)|完整\s*(PRD|产物)|artifact/i.test(item))) {
    badges.push('完整产物已引用')
  }
  if (hasRuntimeContract.value) badges.push('角色契约已加载')
  const capabilityTotal = numberValue(appCapabilities.value.total_functions)
  if (hasAppCapabilities.value && capabilityTotal >= 0) badges.push(`能力快照 ${capabilityTotal} 个函数`)
  return badges
})

const roleLabel = computed(() => {
  const previous = firstString(resultData.value.previous_role_name, resultData.value.previous_role)
  const current = firstString(resultData.value.display_name, resultData.value.current_role, args.value.target_role)
  if (previous && current && previous !== current) return `${previous} -> ${current}`
  return current
})

const statusLabel = computed(() => {
  if (props.toolCall.status === 'streaming') return '解析中'
  if (props.toolCall.status === 'running') return '执行中'
  if (props.toolCall.status === 'ok') return '已交接'
  if (props.toolCall.status === 'error') return '失败'
  return props.toolCall.status
})

const statusType = computed(() => {
  if (props.toolCall.status === 'ok') return 'success'
  if (props.toolCall.status === 'error') return 'danger'
  return 'info'
})

function parseJSONRecord(raw: string | undefined): UnknownRecord {
  if (!raw || !raw.trim()) return {}
  try {
    return asRecord(JSON.parse(raw))
  } catch {
    return {}
  }
}

function asRecord(value: unknown): UnknownRecord {
  return value && typeof value === 'object' && !Array.isArray(value) ? value as UnknownRecord : {}
}

function firstString(...values: unknown[]): string {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) return value.trim()
  }
  return ''
}

function firstList(...values: unknown[]): string[] {
  for (const value of values) {
    const list = normalizeList(value)
    if (list.length) return list
  }
  return []
}

function normalizeList(value: unknown): string[] {
  if (Array.isArray(value)) {
    return uniqueStrings(value.map((item) => String(item ?? '').trim()).filter(Boolean))
  }
  if (typeof value === 'string' && value.trim()) {
    return [value.trim()]
  }
  return []
}

function normalizeHooks(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  const out: string[] = []
  for (const item of value) {
    const hook = asRecord(item) as RuntimeHook
    const stage = firstString(hook.stage)
    const id = firstString(hook.id)
    const purpose = firstString(hook.purpose)
    const produces = firstList(hook.produces)
    const label = [stage, id].filter(Boolean).join(' · ')
    const tail = produces.length ? `产出：${produces.join('、')}` : ''
    const text = [label, purpose, tail].filter(Boolean).join('；')
    if (text) out.push(text)
  }
  return uniqueStrings(out)
}

function normalizeCapabilityFunctions(value: unknown): CapabilityFunctionView[] {
  if (!Array.isArray(value)) return []
  const out: CapabilityFunctionView[] = []
  for (const item of value) {
    const record = asRecord(item)
    const type = firstString(record.type)
    const name = firstString(record.name, record.code)
    const fullCodePath = firstString(record.full_code_path)
    const capabilities = firstString(record.capabilities)
    const runTools = firstList(record.run_tools)
    const schemaSummary = firstList(record.schema_summary).slice(0, 4)
    const title = [type, name].filter(Boolean).join(' · ')
    const meta = [
      capabilities ? `能力：${capabilities}` : '',
      runTools.length ? `工具：${runTools.join('、')}` : '',
    ].filter(Boolean).join('；')
    out.push({
      title: title || fullCodePath || '未命名函数',
      name,
      fullCodePath,
      meta,
      schemaSummary,
    })
  }
  return out
}

function formatAppCapabilitySummary(value: UnknownRecord): string {
  const status = firstString(value.status)
  if (!status) return ''
  const directory = firstString(value.execute_directory)
  const total = numberValue(value.total_functions)
  const displayed = numberValue(value.displayed_functions)
  const counts = asRecord(value.counts)
  const tables = numberValue(counts.tables)
  const forms = numberValue(counts.forms)
  const charts = numberValue(counts.charts)
  const error = firstString(value.error)
  if (status === 'error') {
    return `获取失败${directory ? ` · ${directory}` : ''}${error ? ` · ${error}` : ''}`
  }
  if (status === 'empty') {
    return `未发现已注册函数${directory ? ` · ${directory}` : ''}`
  }
  if (status === 'skipped') {
    return `未生成能力快照${directory ? ` · ${directory}` : ''}`
  }
  const countText = total >= 0 ? `${total} 个函数` : '函数数量未知'
  const displayText = displayed >= 0 ? `展示 ${displayed} 个` : ''
  const typeText = [tables >= 0 ? `Table ${tables}` : '', forms >= 0 ? `Form ${forms}` : '', charts >= 0 ? `Chart ${charts}` : '']
    .filter(Boolean)
    .join(' / ')
  return [directory, countText, typeText, displayText].filter(Boolean).join(' · ')
}

function arrayLength(value: unknown): number {
  return Array.isArray(value) ? value.length : 0
}

function numberValue(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string' && value.trim() && Number.isFinite(Number(value))) return Number(value)
  return -1
}

function uniqueStrings(items: string[]): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const item of items) {
    const text = item.trim()
    if (!text || seen.has(text)) continue
    seen.add(text)
    out.push(text)
  }
  return out
}
</script>

<style scoped lang="scss">
.role-handoff-card {
  padding: 10px 12px;
  background: var(--el-fill-color-blank);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.role-handoff-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 10px;
}

.role-handoff-title {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 600;
}

.role-handoff-role {
  min-width: 0;
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.role-handoff-directory {
  margin-bottom: 10px;
}

.role-handoff-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: -2px 0 10px;
}

.role-handoff-meta span {
  padding: 2px 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 4px;
}

.role-handoff-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 10px;
}

.role-handoff-runtime {
  margin-top: 10px;
}

.role-handoff-capabilities {
  margin-top: 10px;
}

.role-handoff-section {
  min-width: 0;
  padding: 8px;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
}

.role-handoff-section--wide {
  grid-column: 1 / -1;
}

.role-handoff-label {
  margin-bottom: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1;
}

ul {
  display: grid;
  gap: 4px;
  margin: 0;
  padding-left: 16px;
}

li {
  color: var(--el-text-color-regular);
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
}

code {
  color: var(--el-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 12px;
  white-space: normal;
  word-break: break-all;
}

.role-handoff-references {
  padding-left: 0;
  list-style: none;
}

.role-handoff-empty {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.role-handoff-capability-summary {
  margin-bottom: 6px;
  color: var(--el-text-color-primary);
  font-size: 12px;
  line-height: 1.45;
}

.role-handoff-capability-guidance {
  margin-bottom: 8px;
}

.role-handoff-capability-functions {
  padding-left: 0;
  list-style: none;
}

.role-handoff-capability-functions > li {
  padding: 6px 0;
  border-top: 1px solid var(--el-border-color-extra-light);
}

.role-handoff-capability-functions > li:first-child {
  border-top: 0;
}

.role-handoff-capability-name {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
  align-items: baseline;
  font-weight: 600;
}

.role-handoff-capability-meta,
.role-handoff-capability-schema {
  margin-top: 3px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
}

@media (max-width: 720px) {
  .role-handoff-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
