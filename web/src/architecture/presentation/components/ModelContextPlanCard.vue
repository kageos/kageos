<template>
  <div :class="['model-context-card', { 'is-expanded': expanded }]">
    <button
      type="button"
      class="model-context-head"
      :aria-expanded="expanded"
      :title="`${t('modelContext.title')} · ${plan.protocol_version} · ${roundLabel}`"
      @click="expanded = !expanded"
    >
      <div class="model-context-title">
        <span>{{ t('modelContext.title') }}</span>
        <span v-if="expanded && roleLabel" class="model-context-role">{{ roleLabel }}</span>
      </div>
      <span class="model-context-meta">
        <span class="model-context-version">{{ expanded ? `${plan.protocol_version} · ${roundLabel}` : roundLabel }}</span>
        <el-icon :size="13" class="model-context-toggle-icon">
          <ArrowUp v-if="expanded" />
          <ArrowDown v-else />
        </el-icon>
      </span>
    </button>

    <div v-if="expanded" class="model-context-grid">
      <section class="model-context-section">
        <div class="model-context-label">{{ t('modelContext.executionDirectory') }}</div>
        <code>{{ plan.execution.full_code_path || t('modelContext.unspecified') }}</code>
        <div v-if="executionSummary" class="model-context-subtle">{{ executionSummary }}</div>
      </section>

      <section class="model-context-section">
        <div class="model-context-label">{{ t('modelContext.historyPolicy') }}</div>
        <div class="model-context-chip-row">
          <span>{{ contextPolicyLabel }}</span>
          <span>{{ messageCountLabel }}</span>
          <span v-if="plan.messages.excluded_by_anchor > 0">{{ t('modelContext.excludedByAnchor', { count: plan.messages.excluded_by_anchor }) }}</span>
          <span v-if="plan.messages.excluded_display_only > 0">{{ t('modelContext.excludedDisplayOnly', { count: plan.messages.excluded_display_only }) }}</span>
          <span v-if="plan.messages.truncated">{{ t('modelContext.truncated') }}</span>
        </div>
        <div class="model-context-subtle">{{ sourceHistoryLabel }}</div>
      </section>

      <section v-if="plan.handoff" class="model-context-section model-context-section--wide">
        <div class="model-context-label">{{ t('modelContext.handoff') }}</div>
        <div class="model-context-chip-row">
          <span v-if="plan.handoff.target_role">{{ t('modelContext.targetRole', { role: formatWorkspaceRoleName(plan.handoff.target_role) }) }}</span>
          <span v-if="plan.handoff.artifact_kind">{{ plan.handoff.artifact_kind }}</span>
          <span v-if="plan.handoff.validation_status">{{ t('modelContext.validationStatus', { status: plan.handoff.validation_status }) }}</span>
        </div>
        <ul v-if="handoffSummary.length" class="model-context-list">
          <li v-for="(item, idx) in handoffSummary" :key="`handoff-${idx}`">{{ item }}</li>
        </ul>
      </section>

      <section class="model-context-section model-context-section--wide">
        <div class="model-context-label">{{ t('modelContext.docsAndTools') }}</div>
        <div class="model-context-chip-row">
          <span>{{ t('modelContext.documentPackage', { count: plan.docs.document_package?.length || 0 }) }}</span>
          <span>{{ t('modelContext.loadedDocs', { count: plan.docs.loaded_docs?.length || 0 }) }}</span>
          <span v-if="plan.docs.missing_docs?.length">{{ t('modelContext.missingDocs', { count: plan.docs.missing_docs.length }) }}</span>
          <span>{{ t('modelContext.tools', { count: plan.tools.llm_tool_count }) }}</span>
        </div>
        <div v-if="docPreview.length" class="model-context-subtle">
          {{ t('modelContext.missingDocsList', { docs: docPreview.join('、') }) }}
        </div>
        <div v-if="toolPreview.length" class="model-context-subtle">
          {{ t('modelContext.toolsList', { tools: toolPreview.join('、') }) }}
        </div>
      </section>

      <section class="model-context-section model-context-section--wide">
        <div class="model-context-label">{{ t('modelContext.cachePlan') }}</div>
        <div class="model-context-chip-row">
          <span>{{ plan.cache_plan.stable_prefix_strategy }}</span>
          <span v-if="plan.llm?.message_count">{{ t('modelContext.requestMessages', { count: plan.llm.message_count }) }}</span>
          <span v-if="plan.llm?.tool_count">{{ t('modelContext.requestTools', { count: plan.llm.tool_count }) }}</span>
          <span :class="`model-context-cache-status--${cacheStatusTone}`">{{ cacheStatusLabel }}</span>
          <span v-if="cacheResult && cacheResult.prompt_tokens > 0">{{ t('modelContext.inputTokens', { count: formatTokenCount(cacheResult.prompt_tokens) }) }}</span>
          <span v-if="cacheResult && cacheResult.cached_tokens_reported">
            {{ t('modelContext.cacheTokens', { tokens: formatTokenCount(cacheResult.cached_tokens), rate: cacheResult.cache_hit_rate_percent }) }}
          </span>
          <span v-else-if="cacheResult">{{ t('modelContext.cacheNotReported') }}</span>
        </div>
        <div v-if="cachePreview.length" class="model-context-subtle">
          {{ t('modelContext.stablePrefix', { items: cachePreview.join('、') }) }}
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
import { useI18n } from 'vue-i18n'
import { ArrowDown, ArrowUp } from '@element-plus/icons-vue'
import type { WorkspaceModelContextPlan } from '@/architecture/presentation/context/api/workspace'
import { formatWorkspaceRoleName } from '@/architecture/presentation/utils/workspaceRoleDisplay'

const props = defineProps<{
  plan: WorkspaceModelContextPlan
  plans?: WorkspaceModelContextPlan[]
}>()

const { t } = useI18n()
const expanded = ref(false)

const roundLabel = computed(() => t('modelContext.round', { round: props.plan.round + 1 }))

const roleLabel = computed(() => {
  const display = formatWorkspaceRoleName(props.plan.role.id, props.plan.role.display_name)
  const source = workspaceRoleSourceLabel(props.plan.role.source)
  return display ? (source ? t('modelContext.roleWithSource', { role: display, source }) : display) : ''
})

function workspaceRoleSourceLabel(source?: string): string {
  switch (String(source || '').trim()) {
    case 'session':
      return t('modelContext.roleSource.session')
    case 'messages':
      return t('modelContext.roleSource.messages')
    case 'default_router':
      return t('modelContext.roleSource.defaultRouter')
    default:
      return source || ''
  }
}

const executionSummary = computed(() => {
  const parts = [
    props.plan.execution.directory_name,
    props.plan.execution.directory_type,
    t('modelContext.childCount', { count: props.plan.execution.children_count }),
    t('modelContext.fileCount', { count: props.plan.execution.files_count })
  ].filter(Boolean)
  return parts.join(' · ')
})

const contextPolicyLabel = computed(() => {
  switch (props.plan.messages.context_policy) {
    case 'artifact_only':
      return t('modelContext.contextPolicy.artifactOnly')
    case 'display_only':
      return t('modelContext.contextPolicy.displayOnly')
    default:
      return t('modelContext.contextPolicy.full')
  }
})

const messageCountLabel = computed(() => {
  const messages = props.plan.messages
  return t('modelContext.includedMessages', {
    included: messages.included_stored_messages,
    total: messages.total_stored_messages,
  })
})

const sourceHistoryLabel = computed(() => {
  switch (props.plan.messages.source_history_policy) {
    case 'same_session_full_with_parent_reference':
      return t('modelContext.sourceHistory.sameSessionWithParentReference')
    default:
      return t('modelContext.sourceHistory.sameSessionFull')
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
      return t('modelContext.cacheStatus.hit')
    case 'miss':
      return t('modelContext.cacheStatus.miss')
    case 'not_reported':
      return t('modelContext.cacheStatus.notReported')
    case 'usage_unavailable':
      return t('modelContext.cacheStatus.usageUnavailable')
    default:
      return t('modelContext.cacheStatus.pending')
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
      if (!result) return t('modelContext.roundCachePending', { round: item.round + 1 })
      if (!result.cached_tokens_reported) return t('modelContext.roundCacheNotReported', { round: item.round + 1 })
      return t('modelContext.roundCacheHit', {
        round: item.round + 1,
        tokens: formatTokenCount(result.cached_tokens),
        rate: result.cache_hit_rate_percent,
      })
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
  display: flex;
  width: fit-content;
  max-width: 100%;
  margin: 0 0 3px;
  padding: 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--text-placeholder);
  overflow: visible;
}

.model-context-card.is-expanded {
  display: block;
  width: 100%;
  margin: 0 0 8px;
  border: 1px solid var(--border-light);
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--text-primary);
  overflow: hidden;
}

.model-context-head {
  width: auto;
  max-width: 100%;
  border: 0;
  border-radius: 6px;
  padding: 1px 4px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 5px;
  font: inherit;
  text-align: left;
  opacity: 0.62;
  transition: background 0.16s ease, color 0.16s ease, opacity 0.16s ease;
}

.model-context-card.is-expanded .model-context-head {
  width: 100%;
  padding: 7px 8px;
  border-radius: 0;
  justify-content: space-between;
  gap: 8px;
  opacity: 1;
}

.model-context-head:hover {
  background: var(--el-fill-color);
  color: var(--text-secondary);
  opacity: 1;
}

.model-context-card.is-expanded .model-context-head:hover {
  color: inherit;
}

.model-context-head:focus-visible {
  outline: 1px solid rgba(var(--color-primary-rgb), 0.42);
  outline-offset: -2px;
}

.model-context-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 10px;
  font-weight: 600;
  color: currentColor;
}

.model-context-card.is-expanded .model-context-title {
  font-size: 11px;
  font-weight: 800;
  color: var(--text-primary);
}

.model-context-role,
.model-context-version {
  min-width: 0;
  overflow: hidden;
  color: currentColor;
  font-size: 10px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-context-card.is-expanded .model-context-role,
.model-context-card.is-expanded .model-context-version {
  color: var(--text-secondary);
  font-weight: 700;
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
  color: currentColor;
  opacity: 0.76;
}

.model-context-card.is-expanded .model-context-toggle-icon {
  color: var(--text-secondary);
  opacity: 1;
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
  color: var(--text-secondary);
  font-size: 10px;
  font-weight: 800;
}

.model-context-section code {
  display: inline-block;
  max-width: 100%;
  overflow-wrap: anywhere;
  color: var(--color-primary);
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
  border: 1px solid var(--border-light);
  border-radius: 999px;
  background: var(--bg-primary);
  color: var(--text-secondary);
  font-size: 10px;
  font-weight: 700;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-context-chip-row .model-context-cache-status--hit {
  border-color: rgba(5, 150, 105, 0.26);
  background: rgba(5, 150, 105, 0.08);
  color: var(--color-success);
}

.model-context-chip-row .model-context-cache-status--miss {
  border-color: rgba(217, 119, 6, 0.26);
  background: rgba(217, 119, 6, 0.08);
  color: var(--color-warning);
}

.model-context-chip-row .model-context-cache-status--unknown,
.model-context-chip-row .model-context-cache-status--pending {
  border-color: var(--border-light);
  background: var(--bg-primary);
  color: var(--text-secondary);
}

.model-context-subtle {
  margin-top: 4px;
  color: var(--text-secondary);
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
  border: 1px solid var(--border-light);
  border-radius: 999px;
  background: var(--bg-primary);
}

.model-context-list {
  margin: 5px 0 0;
  padding-left: 16px;
  color: var(--text-regular);
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
