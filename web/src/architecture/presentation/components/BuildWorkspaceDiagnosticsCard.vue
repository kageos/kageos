<template>
  <div class="build-diagnostics-card">
    <div class="build-diagnostics-head">
      <div class="build-diagnostics-title">
        <span>构建诊断</span>
        <code v-if="workspacePath">{{ workspacePath }}</code>
      </div>
      <el-tag size="small" type="danger">构建失败</el-tag>
    </div>

    <p v-if="summary" class="build-diagnostics-summary">{{ summary }}</p>

    <div v-if="categories.length" class="build-diagnostics-chips">
      <span v-for="item in categories" :key="`cat-${item}`">{{ item }}</span>
    </div>

    <div class="build-diagnostics-grid">
      <section v-if="routers.length" class="build-diagnostics-section">
        <div class="build-diagnostics-label">Router</div>
        <ul class="build-diagnostics-code-list">
          <li v-for="item in routers" :key="`router-${item}`"><code>{{ item }}</code></li>
        </ul>
      </section>

      <section v-if="fieldIssues.length" class="build-diagnostics-section">
        <div class="build-diagnostics-label">字段问题</div>
        <ul>
          <li v-for="item in fieldIssues" :key="`${item.title}-${item.message}`">
            <span>{{ item.title }}</span>
            <span v-if="item.message">：{{ item.message }}</span>
          </li>
        </ul>
      </section>

      <section v-if="sdkSymbols.length" class="build-diagnostics-section">
        <div class="build-diagnostics-label">SDK/API 符号</div>
        <ul class="build-diagnostics-code-list">
          <li v-for="item in sdkSymbols" :key="`sdk-${item}`"><code>{{ item }}</code></li>
        </ul>
      </section>

      <section v-if="requiredDocs.length" class="build-diagnostics-section">
        <div class="build-diagnostics-label">必读资料</div>
        <ul class="build-diagnostics-code-list">
          <li v-for="item in requiredDocs" :key="`doc-${item}`"><code>{{ item }}</code></li>
        </ul>
      </section>

      <section v-if="repairPolicy.length" class="build-diagnostics-section build-diagnostics-section--wide">
        <div class="build-diagnostics-label">修复策略</div>
        <ul>
          <li v-for="(item, idx) in repairPolicy" :key="`policy-${idx}`">{{ item }}</li>
        </ul>
      </section>
    </div>

    <div v-if="retryPolicy" class="build-diagnostics-retry">{{ retryPolicy }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WorkspaceChatToolCallSummary } from '@/architecture/presentation/context/api/workspace'

type UnknownRecord = Record<string, unknown>

interface FieldIssueView {
  title: string
  message: string
}

const props = defineProps<{
  toolCall: WorkspaceChatToolCallSummary
}>()

const resultData = computed(() => asRecord(props.toolCall.result_data))
const diagnostics = computed(() => asRecord(resultData.value.build_diagnostics))
const workspacePath = computed(() => firstString(diagnostics.value.workspace_path, resultData.value.workspace_path))
const summary = computed(() => truncateText(firstString(diagnostics.value.error_summary, resultData.value.error), 260))
const categories = computed(() => firstList(diagnostics.value.categories).slice(0, 8))
const routers = computed(() => firstList(diagnostics.value.routers).slice(0, 8))
const sdkSymbols = computed(() => firstList(diagnostics.value.sdk_symbols).slice(0, 8))
const requiredDocs = computed(() => firstList(diagnostics.value.required_docs).slice(0, 6))
const repairPolicy = computed(() => firstList(diagnostics.value.repair_policy).slice(0, 6))
const retryPolicy = computed(() => firstString(diagnostics.value.retry_policy))
const fieldIssues = computed(() => normalizeFieldIssues(diagnostics.value.field_issues).slice(0, 6))

function asRecord(value: unknown): UnknownRecord {
  return value && typeof value === 'object' && !Array.isArray(value) ? value as UnknownRecord : {}
}

function firstString(...values: unknown[]): string {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) return value.trim()
  }
  return ''
}

function firstList(value: unknown): string[] {
  if (Array.isArray(value)) {
    return uniqueStrings(value.map((item) => String(item ?? '').trim()).filter(Boolean))
  }
  if (typeof value === 'string' && value.trim()) return [value.trim()]
  return []
}

function normalizeFieldIssues(value: unknown): FieldIssueView[] {
  if (!Array.isArray(value)) return []
  const out: FieldIssueView[] = []
  for (const item of value) {
    const record = asRecord(item)
    const field = firstString(record.field)
    const jsonName = firstString(record.json_name)
    const message = firstString(record.message)
    const title = field && jsonName ? `${field} (${jsonName})` : firstString(field, jsonName, '未知字段')
    if (!field && !jsonName && !message) continue
    out.push({ title, message })
  }
  return out
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

function truncateText(value: string, maxLength: number): string {
  const text = value.trim()
  if (!maxLength || text.length <= maxLength) return text
  return `${text.slice(0, maxLength)}...`
}
</script>

<style scoped lang="scss">
.build-diagnostics-card {
  padding: 10px 12px;
  background: var(--el-fill-color-blank);
  border: 1px solid var(--el-color-error-light-7);
  border-radius: 8px;
}

.build-diagnostics-head {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.build-diagnostics-title {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 10px;
  align-items: baseline;
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 600;
}

.build-diagnostics-summary {
  margin: 0 0 8px;
  color: var(--el-text-color-regular);
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
}

.build-diagnostics-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}

.build-diagnostics-chips span {
  padding: 2px 6px;
  color: var(--el-color-warning-dark-2);
  font-size: 12px;
  line-height: 18px;
  background: var(--el-color-warning-light-9);
  border: 1px solid var(--el-color-warning-light-7);
  border-radius: 4px;
}

.build-diagnostics-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 8px;
}

.build-diagnostics-section {
  min-width: 0;
  padding: 8px;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
}

.build-diagnostics-section--wide {
  grid-column: 1 / -1;
}

.build-diagnostics-label {
  margin-bottom: 5px;
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

.build-diagnostics-code-list {
  padding-left: 0;
  list-style: none;
}

code {
  color: var(--el-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 12px;
  white-space: normal;
  word-break: break-all;
}

.build-diagnostics-retry {
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

@media (max-width: 720px) {
  .build-diagnostics-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
