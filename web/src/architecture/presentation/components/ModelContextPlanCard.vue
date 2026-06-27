<template>
  <div class="model-context-card">
    <button
      type="button"
      class="model-context-head"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      <div class="model-context-title">
        <span>模型上下文</span>
        <span v-if="roleLabel" class="model-context-role">{{ roleLabel }}</span>
      </div>
      <span class="model-context-meta">
        <span class="model-context-version">{{ plan.protocol_version }} · 第 {{ plan.round + 1 }} 轮</span>
        <el-icon :size="13" class="model-context-toggle-icon">
          <ArrowUp v-if="expanded" />
          <ArrowDown v-else />
        </el-icon>
      </span>
    </button>

    <div v-if="expanded" class="model-context-grid">
      <section class="model-context-section">
        <div class="model-context-label">执行目录</div>
        <code>{{ plan.execution.full_code_path || '未指定' }}</code>
        <div v-if="executionSummary" class="model-context-subtle">{{ executionSummary }}</div>
      </section>

      <section class="model-context-section">
        <div class="model-context-label">历史策略</div>
        <div class="model-context-chip-row">
          <span>{{ contextPolicyLabel }}</span>
          <span>{{ messageCountLabel }}</span>
          <span v-if="plan.messages.excluded_by_anchor > 0">历史锚点已忽略 {{ plan.messages.excluded_by_anchor }}</span>
          <span v-if="plan.messages.excluded_display_only > 0">展示标签已保留 {{ plan.messages.excluded_display_only }}</span>
          <span v-if="plan.messages.truncated">列表已截断</span>
        </div>
        <div class="model-context-subtle">{{ sourceHistoryLabel }}</div>
      </section>

      <section v-if="plan.handoff" class="model-context-section model-context-section--wide">
        <div class="model-context-label">交接包</div>
        <div class="model-context-chip-row">
          <span v-if="plan.handoff.target_role">目标 {{ formatWorkspaceRoleName(plan.handoff.target_role) }}</span>
          <span v-if="plan.handoff.artifact_kind">{{ plan.handoff.artifact_kind }}</span>
          <span v-if="plan.handoff.validation_status">校验 {{ plan.handoff.validation_status }}</span>
        </div>
        <ul v-if="handoffSummary.length" class="model-context-list">
          <li v-for="(item, idx) in handoffSummary" :key="`handoff-${idx}`">{{ item }}</li>
        </ul>
      </section>

      <section class="model-context-section model-context-section--wide">
        <div class="model-context-label">文档与工具</div>
        <div class="model-context-chip-row">
          <span>文档包 {{ plan.docs.document_package?.length || 0 }}</span>
          <span>已读 {{ plan.docs.loaded_docs?.length || 0 }}</span>
          <span v-if="plan.docs.missing_docs?.length">待读 {{ plan.docs.missing_docs.length }}</span>
          <span>工具 {{ plan.tools.llm_tool_count }}</span>
        </div>
        <div v-if="docPreview.length" class="model-context-subtle">
          待读：{{ docPreview.join('、') }}
        </div>
        <div v-if="toolPreview.length" class="model-context-subtle">
          工具：{{ toolPreview.join('、') }}
        </div>
      </section>

      <section class="model-context-section model-context-section--wide">
        <div class="model-context-label">Cache 计划</div>
        <div class="model-context-chip-row">
          <span>{{ plan.cache_plan.stable_prefix_strategy }}</span>
          <span v-if="plan.llm?.message_count">请求消息 {{ plan.llm.message_count }}</span>
          <span v-if="plan.llm?.tool_count">请求工具 {{ plan.llm.tool_count }}</span>
          <span :class="`model-context-cache-status--${cacheStatusTone}`">{{ cacheStatusLabel }}</span>
          <span v-if="cacheResult && cacheResult.prompt_tokens > 0">输入 {{ formatTokenCount(cacheResult.prompt_tokens) }}</span>
          <span v-if="cacheResult && cacheResult.cached_tokens_reported">
            缓存 {{ formatTokenCount(cacheResult.cached_tokens) }} / {{ cacheResult.cache_hit_rate_percent }}%
          </span>
          <span v-else-if="cacheResult">缓存未上报</span>
        </div>
        <div v-if="cachePreview.length" class="model-context-subtle">
          稳定前缀：{{ cachePreview.join('、') }}
        </div>
        <div v-if="roundCacheBadges.length > 1" class="model-context-subtle model-context-rounds">
          <span v-for="badge in roundCacheBadges" :key="badge">{{ badge }}</span>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { ArrowDown, ArrowUp } from '@element-plus/icons-vue'
import type { WorkspaceModelContextPlan } from '@/architecture/presentation/context/api/workspace'
import { formatWorkspaceRoleName } from '@/architecture/presentation/utils/workspaceRoleDisplay'

const props = defineProps<{
  plan: WorkspaceModelContextPlan
  plans?: WorkspaceModelContextPlan[]
}>()

const expanded = ref(false)

const roleLabel = computed(() => {
  const display = formatWorkspaceRoleName(props.plan.role.id, props.plan.role.display_name)
  const source = workspaceRoleSourceLabel(props.plan.role.source)
  return display ? `${display}${source}` : ''
})

function workspaceRoleSourceLabel(source?: string): string {
  switch (String(source || '').trim()) {
    case 'session':
      return ' · 当前会话'
    case 'messages':
      return ' · 历史消息'
    case 'default_router':
      return ' · 默认'
    default:
      return source ? ` · ${source}` : ''
  }
}

const executionSummary = computed(() => {
  const parts = [
    props.plan.execution.directory_name,
    props.plan.execution.directory_type,
    `子节点 ${props.plan.execution.children_count}`,
    `文件 ${props.plan.execution.files_count}`
  ].filter(Boolean)
  return parts.join(' · ')
})

const contextPolicyLabel = computed(() => {
  switch (props.plan.messages.context_policy) {
    case 'artifact_only':
      return '完整上下文 · 产物重点'
    case 'display_only':
      return '完整上下文 · 展示标签'
    default:
      return '完整上下文'
  }
})

const messageCountLabel = computed(() => {
  const messages = props.plan.messages
  return `入模 ${messages.included_stored_messages}/${messages.total_stored_messages}`
})

const sourceHistoryLabel = computed(() => {
  switch (props.plan.messages.source_history_policy) {
    case 'same_session_full_with_parent_reference':
      return '同一会话保留完整历史；父会话信息仅作来源引用。'
    default:
      return '同一会话保留完整历史进入模型。'
  }
})

const handoffSummary = computed(() => [
  ...(props.plan.handoff?.task_context || []),
  ...(props.plan.handoff?.key_information || [])
].slice(0, 4))

const docPreview = computed(() => (props.plan.docs.missing_docs || []).slice(0, 3))
const toolPreview = computed(() => (props.plan.tools.llm_tools || []).slice(0, 8))
const cachePreview = computed(() => (props.plan.cache_plan.stable_prefix_items || []).slice(0, 4))
const cacheResult = computed(() => props.plan.cache_plan.result)

const cacheStatusLabel = computed(() => {
  switch (cacheResult.value?.status) {
    case 'hit':
      return 'cache 命中'
    case 'miss':
      return 'cache 未命中'
    case 'not_reported':
      return '上游未上报'
    case 'usage_unavailable':
      return 'usage 缺失'
    default:
      return '等待 usage'
  }
})

const cacheStatusTone = computed(() => {
  switch (cacheResult.value?.status) {
    case 'hit':
      return 'hit'
    case 'miss':
      return 'miss'
    case 'not_reported':
      return 'unknown'
    case 'usage_unavailable':
      return 'unknown'
    default:
      return 'pending'
  }
})

const roundCacheBadges = computed(() => {
  const plans = props.plans?.length ? props.plans : [props.plan]
  return [...plans]
    .sort((left, right) => left.round - right.round)
    .map(item => {
      const result = item.cache_plan.result
      if (!result) return `第 ${item.round + 1} 轮 等待 usage`
      if (!result.cached_tokens_reported) return `第 ${item.round + 1} 轮 未上报`
      return `第 ${item.round + 1} 轮 ${formatTokenCount(result.cached_tokens)}/${result.cache_hit_rate_percent}%`
    })
})

function formatTokenCount(value: number): string {
  const n = Math.max(0, Math.round(value || 0))
  if (n >= 1_000_000) return `${trimTrailingZeros(n / 1_000_000)}m`
  if (n >= 1000) return `${trimTrailingZeros(n / 1000)}k`
  return String(n)
}

function trimTrailingZeros(value: number): string {
  return value.toFixed(value >= 10 ? 0 : 1).replace(/\.0$/, '')
}
</script>

<style scoped>
.model-context-card {
  margin: 0 0 8px;
  padding: 0;
  border: 1px solid rgba(96, 231, 255, 0.16);
  border-radius: 8px;
  background: rgba(6, 18, 30, 0.54);
  color: var(--mini-cyber-text, #d8f8ff);
  overflow: hidden;
}

.model-context-head {
  width: 100%;
  border: 0;
  padding: 7px 8px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font: inherit;
  text-align: left;
}

.model-context-head:hover {
  background: rgba(34, 211, 238, 0.06);
}

.model-context-head:focus-visible {
  outline: 1px solid rgba(96, 231, 255, 0.52);
  outline-offset: -2px;
}

.model-context-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 11px;
  font-weight: 800;
  color: #dffbff;
}

.model-context-role,
.model-context-version {
  min-width: 0;
  overflow: hidden;
  color: rgba(184, 225, 235, 0.68);
  font-size: 10px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-context-meta {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
}

.model-context-version {
  flex-shrink: 1;
}

.model-context-toggle-icon {
  flex-shrink: 0;
  color: rgba(184, 225, 235, 0.72);
}

.model-context-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 7px;
  padding: 0 8px 8px;
}

.model-context-section {
  min-width: 0;
}

.model-context-label {
  margin-bottom: 4px;
  color: rgba(143, 187, 204, 0.68);
  font-size: 10px;
  font-weight: 800;
}

.model-context-section code {
  display: inline-block;
  max-width: 100%;
  overflow-wrap: anywhere;
  color: #bff8ff;
  font-size: 11px;
}

.model-context-chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.model-context-chip-row span {
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  padding: 2px 6px;
  border: 1px solid rgba(96, 231, 255, 0.14);
  border-radius: 999px;
  background: rgba(34, 211, 238, 0.08);
  color: rgba(215, 249, 255, 0.86);
  font-size: 10px;
  font-weight: 700;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-context-chip-row .model-context-cache-status--hit {
  border-color: rgba(74, 222, 128, 0.25);
  background: rgba(34, 197, 94, 0.1);
  color: rgba(187, 247, 208, 0.92);
}

.model-context-chip-row .model-context-cache-status--miss {
  border-color: rgba(250, 204, 21, 0.24);
  background: rgba(234, 179, 8, 0.09);
  color: rgba(254, 240, 138, 0.92);
}

.model-context-chip-row .model-context-cache-status--unknown,
.model-context-chip-row .model-context-cache-status--pending {
  border-color: rgba(148, 163, 184, 0.2);
  background: rgba(148, 163, 184, 0.08);
  color: rgba(226, 232, 240, 0.78);
}

.model-context-subtle {
  margin-top: 4px;
  color: rgba(184, 225, 235, 0.62);
  font-size: 10px;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.model-context-rounds {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.model-context-rounds span {
  padding: 1px 5px;
  border: 1px solid rgba(96, 231, 255, 0.12);
  border-radius: 999px;
  background: rgba(34, 211, 238, 0.06);
}

.model-context-list {
  margin: 5px 0 0;
  padding-left: 16px;
  color: rgba(216, 248, 255, 0.82);
  font-size: 11px;
  line-height: 1.45;
}

.model-context-list li {
  margin: 2px 0;
}

@media (min-width: 720px) {
  .model-context-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .model-context-section--wide {
    grid-column: 1 / -1;
  }
}
</style>
