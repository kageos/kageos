<template>
  <div class="role-handoff-card">
    <!-- 折叠态：极简 chip，只透出角色切换与状态 -->
    <div
      v-if="collapsed"
      class="role-handoff-chip"
      role="button"
      tabindex="0"
      @click="collapsed = false"
      @keydown.enter.prevent="collapsed = false"
      @keydown.space.prevent="collapsed = false"
    >
      <span class="role-handoff-chip-icon" aria-hidden="true">⇄</span>
      <span class="role-handoff-chip-text">
        <span v-if="roleLabel" class="role-handoff-chip-role">{{ roleLabel }}</span>
        <span v-else>角色交接</span>
      </span>
      <el-tag size="small" :type="statusType" class="role-handoff-chip-status">{{ statusLabel }}</el-tag>
      <span class="role-handoff-chip-hint">展开详情</span>
    </div>

    <!-- 展开态：完整 debug 信息 -->
    <template v-else>
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

    <section v-if="hasPacketValidation" class="role-handoff-section role-handoff-validation">
      <div class="role-handoff-label">交接协议校验</div>
      <div class="role-handoff-chip-row role-handoff-chip-row--validation">
        <span :class="`role-handoff-validation-chip--${packetValidationStatus}`">{{ packetValidationLabel }}</span>
      </div>
      <ul v-if="packetValidationDetails.length">
        <li v-for="(item, idx) in packetValidationDetails" :key="`packet-validation-${idx}`">{{ item }}</li>
      </ul>
    </section>

    <section v-if="hasContextPolicy" class="role-handoff-section role-handoff-context-policy">
      <div class="role-handoff-label">模型上下文策略</div>
      <div v-if="contextPolicyBadges.length" class="role-handoff-chip-row role-handoff-chip-row--context">
        <span v-for="badge in contextPolicyBadges" :key="badge">{{ badge }}</span>
      </div>
      <div v-if="contextPolicy" class="role-handoff-policy-text">{{ contextPolicy }}</div>
    </section>

    <section v-if="hasRoleDefinition" class="role-handoff-section role-handoff-definition">
      <div class="role-handoff-label">角色协议</div>
      <div v-if="roleResponsibility" class="role-handoff-policy-text">{{ roleResponsibility }}</div>
      <div v-if="roleAllowedTools.length" class="role-handoff-diagnostic-block">
        <div class="role-handoff-mini-label">允许工具</div>
        <div class="role-handoff-chip-row role-handoff-chip-row--tools">
          <span v-for="tool in roleAllowedTools" :key="`allowed-${tool}`">{{ tool }}</span>
        </div>
      </div>
      <div v-if="roleForbiddenTools.length" class="role-handoff-diagnostic-block">
        <div class="role-handoff-mini-label">禁止工具</div>
        <div class="role-handoff-chip-row role-handoff-chip-row--forbidden">
          <span v-for="tool in roleForbiddenTools" :key="`forbidden-${tool}`">{{ tool }}</span>
        </div>
      </div>
    </section>

    <div v-if="executedHooks.length || hasBuildDiagnostics" class="role-handoff-grid role-handoff-observability">
      <section v-if="executedHooks.length" class="role-handoff-section">
        <div class="role-handoff-label">已执行 Hook</div>
        <ul class="role-handoff-hook-list">
          <li v-for="hook in executedHooks" :key="`${hook.id}-${hook.stage}`">
            <div class="role-handoff-hook-head">
              <code>{{ hook.id }}</code>
              <el-tag size="small" :type="hook.statusType">{{ hook.status || 'unknown' }}</el-tag>
            </div>
            <div v-if="hook.meta" class="role-handoff-subtle">{{ hook.meta }}</div>
            <div v-if="hook.produced.length" class="role-handoff-subtle">产出：{{ hook.produced.join('、') }}</div>
            <div v-if="hook.note" class="role-handoff-subtle">{{ hook.note }}</div>
          </li>
        </ul>
      </section>

      <section v-if="hasBuildDiagnostics" class="role-handoff-section role-handoff-section--wide">
        <div class="role-handoff-label">构建诊断</div>
        <div v-if="buildDiagnosticSummary" class="role-handoff-diagnostic-summary">{{ buildDiagnosticSummary }}</div>
        <div v-if="buildDiagnosticCategories.length" class="role-handoff-chip-row">
          <span v-for="item in buildDiagnosticCategories" :key="`diag-cat-${item}`">{{ item }}</span>
        </div>
        <div v-if="buildDiagnosticRouters.length" class="role-handoff-diagnostic-block">
          <div class="role-handoff-mini-label">Router</div>
          <ul class="role-handoff-references">
            <li v-for="item in buildDiagnosticRouters" :key="`diag-router-${item}`">
              <code>{{ item }}</code>
            </li>
          </ul>
        </div>
        <div v-if="buildDiagnosticFieldIssues.length" class="role-handoff-diagnostic-block">
          <div class="role-handoff-mini-label">字段问题</div>
          <ul>
            <li v-for="item in buildDiagnosticFieldIssues" :key="`${item.field}-${item.jsonName}-${item.message}`">
              <span>{{ item.title }}</span>
              <span v-if="item.message">：{{ item.message }}</span>
            </li>
          </ul>
        </div>
        <div v-if="buildDiagnosticDocs.length" class="role-handoff-diagnostic-block">
          <div class="role-handoff-mini-label">必读资料</div>
          <ul class="role-handoff-references">
            <li v-for="item in buildDiagnosticDocs" :key="`diag-doc-${item}`">
              <code>{{ item }}</code>
            </li>
          </ul>
        </div>
        <div v-if="buildDiagnosticPolicy.length" class="role-handoff-diagnostic-block">
          <div class="role-handoff-mini-label">修复策略</div>
          <ul>
            <li v-for="(item, idx) in buildDiagnosticPolicy" :key="`diag-policy-${idx}`">{{ item }}</li>
          </ul>
        </div>
        <div v-if="buildDiagnosticRetryPolicy" class="role-handoff-diagnostic-retry">
          {{ buildDiagnosticRetryPolicy }}
        </div>
      </section>
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

    <div class="role-handoff-collapse-footer">
      <button type="button" class="role-handoff-collapse-btn" @click="collapsed = true">
        收起详情
      </button>
    </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
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
  implementation_status?: unknown
}

interface ExecutedHookView {
  id: string
  stage: string
  status: string
  statusType: 'success' | 'warning' | 'danger' | 'info'
  meta: string
  produced: string[]
  note: string
}

interface CapabilityFunctionView {
  title: string
  name: string
  fullCodePath: string
  meta: string
  schemaSummary: string[]
}

interface BuildDiagnosticFieldIssueView {
  field: string
  jsonName: string
  title: string
  message: string
}

const props = withDefaults(
  defineProps<{
    toolCall: WorkspaceChatToolCallSummary
    defaultCollapsed?: boolean
  }>(),
  {
    defaultCollapsed: true,
  }
)

const collapsed = ref(props.defaultCollapsed)

const args = computed<UnknownRecord>(() => parseJSONRecord(props.toolCall.arguments))
const resultData = computed<UnknownRecord>(() => asRecord(props.toolCall.result_data))
const handoffPacket = computed<UnknownRecord>(() => asRecord(resultData.value.handoff_packet))
const resultHandoff = computed<HandoffBlock>(() => mergeHandoffBlock(handoffPacket.value, asRecord(resultData.value.handoff)))
const roleDefinition = computed<UnknownRecord>(() => asRecord(resultData.value.role_definition))
const runtimeContract = computed<UnknownRecord>(() => firstRecord(resultData.value.runtime_contract, roleDefinition.value.runtime_contract))
const appCapabilities = computed<UnknownRecord>(() => asRecord(resultData.value.app_capabilities))
const buildDiagnostics = computed<UnknownRecord>(() => firstRecord(handoffPacket.value.build_diagnostics, resultData.value.build_diagnostics))
const executedHooks = computed(() => uniqueExecutedHooks([
  ...normalizeExecutedHooks(handoffPacket.value.executed_hooks),
  ...normalizeExecutedHooks(resultData.value.executed_hooks),
]))
const contextPolicy = computed(() => firstString(handoffPacket.value.context_policy, resultData.value.context_policy))
const packetValidation = computed<UnknownRecord>(() => asRecord(handoffPacket.value.validation))
const packetValidationStatus = computed(() => normalizePacketValidationStatus(firstString(packetValidation.value.status)))
const packetValidationLabel = computed(() => {
  if (packetValidationStatus.value === 'error') return '字段异常'
  if (packetValidationStatus.value === 'warning') return '字段已修正'
  return '字段完整'
})
const packetValidationDetails = computed(() => [
  ...firstList(packetValidation.value.errors).map((item) => `错误：${item}`),
  ...firstList(packetValidation.value.warnings).map((item) => `警告：${item}`),
  ...firstList(packetValidation.value.repaired).map((item) => `已修正：${item}`),
])
const hasPacketValidation = computed(() =>
  Object.keys(handoffPacket.value).length > 0 && Object.keys(packetValidation.value).length > 0
)

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
  ...firstList(handoffPacket.value.references),
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
const buildDiagnosticSummary = computed(() => formatBuildDiagnosticSummary(buildDiagnostics.value))
const buildDiagnosticCategories = computed(() => firstList(buildDiagnostics.value.categories).slice(0, 8))
const buildDiagnosticRouters = computed(() => firstList(buildDiagnostics.value.routers).slice(0, 8))
const buildDiagnosticFieldIssues = computed(() => normalizeDiagnosticFieldIssues(buildDiagnostics.value.field_issues).slice(0, 6))
const buildDiagnosticDocs = computed(() => firstList(buildDiagnostics.value.required_docs).slice(0, 6))
const buildDiagnosticPolicy = computed(() => firstList(buildDiagnostics.value.repair_policy).slice(0, 6))
const buildDiagnosticRetryPolicy = computed(() => firstString(buildDiagnostics.value.retry_policy))
const hasBuildDiagnostics = computed(() =>
  Boolean(
    buildDiagnosticSummary.value ||
    buildDiagnosticCategories.value.length ||
    buildDiagnosticRouters.value.length ||
    buildDiagnosticFieldIssues.value.length ||
    buildDiagnosticDocs.value.length ||
    buildDiagnosticPolicy.value.length ||
    buildDiagnosticRetryPolicy.value
  )
)
const contextPolicyBadges = computed(() => {
  const badges: string[] = []
  if (booleanValue(resultData.value.switched)) badges.push('角色已切换')
  if (booleanValue(args.value.reset_context) || contextPolicy.value.includes('丢弃旧细节')) badges.push('旧细节已裁剪')
  if (contextPolicy.value.includes('旧上下文只作背景')) badges.push('旧上下文仅作背景')
  if (contextPolicy.value.includes('标准四块交接')) badges.push('四块交接生效')
  if (executeDirectory.value) badges.push('目录已固定')
  return uniqueStrings(badges)
})
const hasContextPolicy = computed(() => Boolean(contextPolicy.value || contextPolicyBadges.value.length))
const roleResponsibility = computed(() => firstString(roleDefinition.value.responsibility, roleDefinition.value.default_next_action))
const roleAllowedTools = computed(() => firstList(roleDefinition.value.allowed_tools).slice(0, 16))
const roleForbiddenTools = computed(() => firstList(roleDefinition.value.forbidden_tools).slice(0, 12))
const hasRoleDefinition = computed(() =>
  Boolean(roleResponsibility.value || roleAllowedTools.value.length > 0 || roleForbiddenTools.value.length > 0)
)

const metaBadges = computed(() => {
  const badges: string[] = []
  const packetVersion = firstString(handoffPacket.value.version)
  const requiredCount = arrayLength(resultData.value.required_docs)
  const loadedCount = arrayLength(resultData.value.loaded_docs)
  if (Object.keys(handoffPacket.value).length > 0) badges.push(packetVersion ? `交接协议 ${packetVersion}` : '交接协议')
  if (hasPacketValidation.value) badges.push(packetValidationLabel.value)
  if (requiredCount > 0) badges.push(`文档包 ${requiredCount} 项`)
  if (loadedCount > 0) badges.push(`已加载 ${loadedCount} 项`)
  if (references.value.some((item) => /agent_app_(prd|build)|完整\s*(PRD|产物)|artifact/i.test(item))) {
    badges.push('完整产物已引用')
  }
  if (hasRoleDefinition.value) badges.push('角色协议')
  if (hasRuntimeContract.value) badges.push('角色契约已加载')
  if (hasContextPolicy.value) badges.push('上下文策略')
  const capabilityTotal = numberValue(appCapabilities.value.total_functions)
  if (hasAppCapabilities.value && capabilityTotal >= 0) badges.push(`能力快照 ${capabilityTotal} 个函数`)
  if (executedHooks.value.length) badges.push(`Hook ${executedHooks.value.length} 个`)
  if (hasBuildDiagnostics.value) badges.push('构建诊断')
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

function firstRecord(...values: unknown[]): UnknownRecord {
  for (const value of values) {
    const record = asRecord(value)
    if (Object.keys(record).length > 0) return record
  }
  return {}
}

function mergeHandoffBlock(packet: UnknownRecord, legacy: UnknownRecord): HandoffBlock {
  return {
    execute_directory: firstString(packet.execute_directory, legacy.execute_directory),
    task_context: firstList(packet.task_context, legacy.task_context),
    key_information: firstList(packet.key_information, legacy.key_information),
    references: firstList(packet.references, legacy.references),
  }
}

function firstString(...values: unknown[]): string {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) return value.trim()
  }
  return ''
}

function normalizePacketValidationStatus(status: string): 'ok' | 'warning' | 'error' {
  if (status === 'error') return 'error'
  if (status === 'warning') return 'warning'
  return 'ok'
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
    const status = formatHookImplementationStatus(firstString(hook.implementation_status))
    const label = [stage, id, status].filter(Boolean).join(' · ')
    const tail = produces.length ? `产出：${produces.join('、')}` : ''
    const text = [label, purpose, tail].filter(Boolean).join('；')
    if (text) out.push(text)
  }
  return uniqueStrings(out)
}

function formatHookImplementationStatus(status: string): string {
  if (status === 'implemented') return '已实现'
  if (status === 'planned') return '计划中'
  if (status === 'unknown') return '未登记'
  return ''
}

function normalizeExecutedHooks(value: unknown): ExecutedHookView[] {
  if (!Array.isArray(value)) return []
  const out: ExecutedHookView[] = []
  for (const item of value) {
    const record = asRecord(item)
    const id = firstString(record.id)
    const stage = firstString(record.stage)
    const status = firstString(record.status)
    const sourceRole = firstString(record.source_role)
    const targetRole = firstString(record.target_role)
    const produced = firstList(record.produced)
    const note = firstString(record.note)
    const roleText = [sourceRole, targetRole].filter(Boolean).join(' -> ')
    const meta = [stage, roleText].filter(Boolean).join(' · ')
    if (!id && !stage && !status && !note) continue
    out.push({
      id: id || 'unknown_hook',
      stage,
      status,
      statusType: hookStatusType(status),
      meta,
      produced,
      note,
    })
  }
  return out
}

function uniqueExecutedHooks(items: ExecutedHookView[]): ExecutedHookView[] {
  const seen = new Set<string>()
  const out: ExecutedHookView[] = []
  for (const item of items) {
    const key = [item.id, item.stage, item.status, item.note].join('|')
    if (seen.has(key)) continue
    seen.add(key)
    out.push(item)
  }
  return out
}

function hookStatusType(status: string): 'success' | 'warning' | 'danger' | 'info' {
  if (status === 'ok') return 'success'
  if (status === 'error') return 'danger'
  if (status === 'empty' || status === 'skipped') return 'warning'
  return 'info'
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

function normalizeDiagnosticFieldIssues(value: unknown): BuildDiagnosticFieldIssueView[] {
  if (!Array.isArray(value)) return []
  const out: BuildDiagnosticFieldIssueView[] = []
  for (const item of value) {
    const record = asRecord(item)
    const field = firstString(record.field)
    const jsonName = firstString(record.json_name)
    const message = firstString(record.message)
    const title = field && jsonName ? `${field} (${jsonName})` : firstString(field, jsonName, '未知字段')
    if (!field && !jsonName && !message) continue
    out.push({
      field,
      jsonName,
      title,
      message,
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

function formatBuildDiagnosticSummary(value: UnknownRecord): string {
  const status = firstString(value.status)
  if (!status) return ''
  const directory = firstString(value.workspace_path)
  const errorSummary = firstString(value.error_summary)
  if (status === 'empty') {
    return [directory, '未拿到完整构建错误，先读取 build_workspace 失败输出'].filter(Boolean).join(' · ')
  }
  const summary = errorSummary ? truncateText(errorSummary, 220) : '已生成构建失败诊断'
  return [directory, summary].filter(Boolean).join(' · ')
}

function truncateText(value: string, maxLength: number): string {
  const text = value.trim()
  if (!maxLength || text.length <= maxLength) return text
  return `${text.slice(0, maxLength)}...`
}

function arrayLength(value: unknown): number {
  return Array.isArray(value) ? value.length : 0
}

function numberValue(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) return value
  if (typeof value === 'string' && value.trim() && Number.isFinite(Number(value))) return Number(value)
  return -1
}

function booleanValue(value: unknown): boolean {
  if (typeof value === 'boolean') return value
  if (typeof value === 'string') return value.trim().toLowerCase() === 'true'
  return false
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

  // 折叠态：去掉卡片感，变成内敛的 inline chip
  &:has(.role-handoff-chip) {
    display: inline-flex;
    padding: 0;
    background: transparent;
    border: none;
    border-radius: 0;
  }
}

.role-handoff-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 5px 10px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 999px;
  cursor: pointer;
  user-select: none;
  transition: all 0.15s ease;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-5);

    .role-handoff-chip-hint {
      opacity: 1;
      color: var(--el-color-primary);
    }
  }

  &:focus-visible {
    outline: none;
    box-shadow: 0 0 0 2px var(--el-color-primary-light-7);
  }
}

.role-handoff-chip-icon {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  opacity: 0.7;

  .role-handoff-chip:hover & {
    color: var(--el-color-primary);
    opacity: 1;
  }
}

.role-handoff-chip-text {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  max-width: 280px;
}

.role-handoff-chip-role {
  overflow: hidden;
  color: var(--el-text-color-regular);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;

  .role-handoff-chip:hover & {
    color: var(--el-color-primary);
  }
}

.role-handoff-chip-status {
  flex-shrink: 0;
}

.role-handoff-chip-hint {
  color: var(--el-text-color-placeholder);
  font-size: 11px;
  opacity: 0;
  transition: opacity 0.15s ease;
}

.role-handoff-collapse-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px dashed var(--el-border-color-lighter);
}

.role-handoff-collapse-btn {
  padding: 4px 10px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
  background: transparent;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-5);
  }

  &:focus-visible {
    outline: none;
    box-shadow: 0 0 0 2px var(--el-color-primary-light-7);
  }
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

.role-handoff-observability {
  margin-bottom: 10px;
}

.role-handoff-context-policy {
  margin-bottom: 10px;
}

.role-handoff-validation {
  margin-bottom: 10px;
}

.role-handoff-definition {
  margin-bottom: 10px;
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

.role-handoff-hook-list {
  padding-left: 0;
  list-style: none;
}

.role-handoff-hook-list > li {
  padding: 6px 0;
  border-top: 1px solid var(--el-border-color-extra-light);
}

.role-handoff-hook-list > li:first-child {
  padding-top: 0;
  border-top: 0;
}

.role-handoff-hook-head {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
}

.role-handoff-subtle {
  margin-top: 3px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
}

.role-handoff-diagnostic-summary {
  margin-bottom: 6px;
  color: var(--el-text-color-primary);
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
}

.role-handoff-chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}

.role-handoff-chip-row span {
  padding: 2px 6px;
  color: var(--el-color-warning-dark-2);
  font-size: 12px;
  line-height: 18px;
  background: var(--el-color-warning-light-9);
  border: 1px solid var(--el-color-warning-light-7);
  border-radius: 4px;
}

.role-handoff-chip-row--context {
  margin-bottom: 6px;
}

.role-handoff-chip-row--context span {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-7);
}

.role-handoff-chip-row--validation {
  margin-bottom: 6px;
}

.role-handoff-chip-row--validation .role-handoff-validation-chip--ok {
  color: var(--el-color-success);
  background: var(--el-color-success-light-9);
  border-color: var(--el-color-success-light-7);
}

.role-handoff-chip-row--validation .role-handoff-validation-chip--warning {
  color: var(--el-color-warning-dark-2);
  background: var(--el-color-warning-light-9);
  border-color: var(--el-color-warning-light-7);
}

.role-handoff-chip-row--validation .role-handoff-validation-chip--error {
  color: var(--el-color-danger);
  background: var(--el-color-danger-light-9);
  border-color: var(--el-color-danger-light-7);
}

.role-handoff-policy-text {
  color: var(--el-text-color-regular);
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
}

.role-handoff-chip-row--tools span {
  color: var(--el-color-success);
  background: var(--el-color-success-light-9);
  border-color: var(--el-color-success-light-7);
}

.role-handoff-chip-row--forbidden span {
  color: var(--el-color-danger);
  background: var(--el-color-danger-light-9);
  border-color: var(--el-color-danger-light-7);
}

.role-handoff-diagnostic-block {
  margin-top: 8px;
}

.role-handoff-mini-label {
  margin-bottom: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1;
}

.role-handoff-diagnostic-retry {
  margin-top: 8px;
  padding: 6px 8px;
  color: var(--el-color-warning-dark-2);
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
  background: var(--el-color-warning-light-9);
  border: 1px solid var(--el-color-warning-light-7);
  border-radius: 6px;
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
