<template>
  <template v-if="messages.length > 0">
    <div
      v-for="(msg, i) in messages"
      :key="i"
      :class="['mini-msg', msg.role, { 'mini-msg--maximized': maximized }]"
    >
      <div v-if="msg.role === 'user'" class="mini-msg-user">
        <div class="mini-msg-user-header">
          <UserDisplay
            :username="msg.user || currentUsername || null"
            mode="simple"
            size="small"
            class="mini-msg-user-display"
          />
          <span class="mini-msg-time">{{ msg.created_at ? formatMessageTime(msg.created_at) : '—' }}</span>
        </div>
        <div class="mini-msg-user-body">
          <OutputFilesDisplay
            v-if="msg.files?.length"
            :file-groups="[{ label: '', files: msg.files }]"
            :section-title="t('miniWorkstation.uploadedFiles')"
            :archive-download="false"
            class="mini-msg-files"
          />
          <span>{{ msg.content }}</span>
        </div>
      </div>
      <template v-else>
        <div class="mini-msg-assistant-header">
          <MiniWorkstationResourceIdentity
            variant="message"
            :name="counterpartName"
            :full-code-path="fullCodePath"
            :resource-type="resourceType"
            :resource-template-type="resourceTemplateType"
          />
          <span v-if="getAssistantModelLabel(msg)" class="mini-msg-model">{{ getAssistantModelLabel(msg) }}</span>
          <span
            v-if="getAssistantCacheLabel(msg)"
            class="mini-msg-cache"
            :title="getAssistantCacheTitle(msg)"
          >
            {{ getAssistantCacheLabel(msg) }}
          </span>
          <span class="mini-msg-time">{{ msg.created_at ? formatMessageTime(msg.created_at) : '—' }}</span>
          <span
            v-if="getAssistantDurationLabel(msg, i)"
            :class="['mini-msg-output-duration', { 'mini-msg-output-duration--running': isAssistantTimerRunning(msg, i) }]"
          >
            <span v-if="isAssistantTimerRunning(msg, i)" class="mini-msg-output-duration-dot"></span>
            {{ t('miniWorkstation.outputDuration', { duration: getAssistantDurationLabel(msg, i) }) }}
          </span>
        </div>
        <ModelContextPlanCard
          v-if="msg.model_context_plan"
          :plan="msg.model_context_plan"
          :plans="msg.model_context_plans"
          class="mini-msg-model-context"
        />
        <div v-if="msg.blocks?.length" class="mini-msg-assistant">
          <template v-for="(block, bi) in msg.blocks" :key="bi">
            <div
              v-if="block.type === 'content'"
              class="mini-content-block mini-md-content"
              v-html="renderContentBlock(block.text, i, bi, msg.blocks.length)"
            ></div>
            <template v-else-if="block.type === 'tool_calls'">
              <template v-if="maximized">
                <MessageToolCalls
                  :tool-calls="block.calls"
                  :file-groups="getFileGroupsFromCalls(block.calls)"
                  :confirm-disabled="sending"
                  @confirm-prd="emit('confirm-prd', $event)"
                />
                <MiniWorkstationPendingActionBar
                  v-for="(interaction, ii) in getInteractionCardsFromCalls(block.calls)"
                  :key="`interaction-max-${interaction.id || ii}`"
                  :interaction="interaction"
                  :sending="sending"
                  :readonly="!isActiveInteraction(interaction)"
                  @view="emit('view', interaction)"
                  @revise="emit('revise', { interaction, text: $event.text })"
                  @cancel="emit('cancel', interaction)"
                  @confirm="emit('confirm', interaction)"
                />
              </template>
              <template v-else>
                <div v-if="getVisibleToolCallsFromCalls(block.calls).length" class="mini-tools-block">
                  <div
                    v-for="(tc, ti) in getVisibleToolCallsFromCalls(block.calls)"
                    :key="`${tc.name}-${ti}`"
                    class="mini-tool-tag"
                  >
                    <el-icon v-if="tc.status === 'streaming' || tc.status === 'running'" class="is-loading" :size="12">
                      <Loading />
                    </el-icon>
                    <el-icon v-else-if="tc.status === 'ok'" :size="12" color="#67c23a">
                      <CircleCheck />
                    </el-icon>
                    <el-icon v-else-if="tc.status === 'error'" :size="12" color="#f56c6c">
                      <CircleClose />
                    </el-icon>
                    <span>{{ getToolDisplayName(tc) }}</span>
                  </div>
                </div>
                <PrdPreview
                  v-for="(tc, pi) in getPrdCallsFromCalls(block.calls)"
                  :key="`prd-${tc.name}-${pi}`"
                  :data="tc.result_data"
                  :confirm-disabled="sending"
                  @confirm="emit('confirm-prd', $event)"
                  class="mini-msg-prd-preview"
                />
                <BuildWorkspaceDiagnosticsCard
                  v-for="(tc, bi) in getBuildWorkspaceFailureCallsFromCalls(block.calls)"
                  :key="`build-failure-${tc.name}-${bi}`"
                  :tool-call="tc"
                  class="mini-msg-build-diagnostics"
                />
                <MiniWorkstationPendingActionBar
                  v-for="(interaction, ii) in getInteractionCardsFromCalls(block.calls)"
                  :key="`interaction-${interaction.id || ii}`"
                  :interaction="interaction"
                  :sending="sending"
                  :readonly="!isActiveInteraction(interaction)"
                  @view="emit('view', interaction)"
                  @revise="emit('revise', { interaction, text: $event.text })"
                  @cancel="emit('cancel', interaction)"
                  @confirm="emit('confirm', interaction)"
                />
                <OutputFilesDisplay
                  v-if="getFileGroupsFromCalls(block.calls).length"
                  :file-groups="getFileGroupsFromCalls(block.calls)"
                  class="mini-msg-files"
                />
                <OutputDisplayFields
                  v-if="getDisplayFieldsFromCalls(block.calls).length"
                  :fields="getDisplayFieldsFromCalls(block.calls)"
                  class="mini-msg-display-fields"
                />
              </template>
            </template>
          </template>
        </div>
        <template v-else>
          <div
            v-if="msg.content"
            class="mini-msg-assistant mini-content-block mini-md-content"
            v-html="renderMarkdown(msg.content)"
          ></div>
          <template v-if="maximized && msg.tool_calls?.length">
            <MessageToolCalls
              :tool-calls="msg.tool_calls"
              :file-groups="getFileGroupsFromCalls(msg.tool_calls)"
              :confirm-disabled="sending"
              @confirm-prd="emit('confirm-prd', $event)"
            />
            <MiniWorkstationPendingActionBar
              v-for="(interaction, ii) in getInteractionCardsFromCalls(msg.tool_calls)"
              :key="`msg-interaction-max-${interaction.id || ii}`"
              :interaction="interaction"
              :sending="sending"
              :readonly="!isActiveInteraction(interaction)"
              @view="emit('view', interaction)"
              @revise="emit('revise', { interaction, text: $event.text })"
              @cancel="emit('cancel', interaction)"
              @confirm="emit('confirm', interaction)"
            />
          </template>
          <template v-else-if="msg.tool_calls?.length">
            <PrdPreview
              v-for="(tc, pi) in getPrdCallsFromCalls(msg.tool_calls)"
              :key="`msg-prd-${tc.name}-${pi}`"
              :data="tc.result_data"
              :confirm-disabled="sending"
              @confirm="emit('confirm-prd', $event)"
              class="mini-msg-prd-preview"
            />
            <BuildWorkspaceDiagnosticsCard
              v-for="(tc, bi) in getBuildWorkspaceFailureCallsFromCalls(msg.tool_calls)"
              :key="`msg-build-failure-${tc.name}-${bi}`"
              :tool-call="tc"
              class="mini-msg-build-diagnostics"
            />
            <MiniWorkstationPendingActionBar
              v-for="(interaction, ii) in getInteractionCardsFromCalls(msg.tool_calls)"
              :key="`msg-interaction-${interaction.id || ii}`"
              :interaction="interaction"
              :sending="sending"
              :readonly="!isActiveInteraction(interaction)"
              @view="emit('view', interaction)"
              @revise="emit('revise', { interaction, text: $event.text })"
              @cancel="emit('cancel', interaction)"
              @confirm="emit('confirm', interaction)"
            />
            <OutputFilesDisplay
              v-if="getFileGroupsFromCalls(msg.tool_calls).length"
              :file-groups="getFileGroupsFromCalls(msg.tool_calls)"
              class="mini-msg-files"
            />
            <OutputDisplayFields
              v-if="getDisplayFieldsFromCalls(msg.tool_calls).length"
              :fields="getDisplayFieldsFromCalls(msg.tool_calls)"
              class="mini-msg-display-fields"
            />
          </template>
        </template>
      </template>
    </div>
  </template>
  <div v-else class="mini-ws-empty">
    <span>{{ t('miniWorkstation.startByCommand') }}</span>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { CircleCheck, CircleClose, Loading } from '@element-plus/icons-vue'
import type { OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'
import type { OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'
import type { ChatMessage, ChatMessageToolCall } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import MessageToolCalls from './MessageToolCalls.vue'
import ModelContextPlanCard from './ModelContextPlanCard.vue'
import OutputDisplayFields from './OutputDisplayFields.vue'
import OutputFilesDisplay from './OutputFilesDisplay.vue'
import PrdPreview from './PrdPreview.vue'
import BuildWorkspaceDiagnosticsCard from './BuildWorkspaceDiagnosticsCard.vue'
import MiniWorkstationPendingActionBar from './MiniWorkstationPendingActionBar.vue'
import MiniWorkstationResourceIdentity from './MiniWorkstationResourceIdentity.vue'
import UserDisplay from '@/architecture/presentation/shared/components/UserDisplay.vue'
import { useAuthStore } from '@/architecture/presentation/context/appStoresContext'
import type { WorkspaceInteraction } from '@/architecture/presentation/context/api/workspace'
import {
  getVisibleWorkspaceToolCalls,
  getWorkspaceToolCallDisplayName
} from '@/architecture/presentation/utils/workspaceRoleDisplay'

const { t } = useI18n()

const authStore = useAuthStore()
const currentUsername = authStore.user?.username || authStore.userName || ''

const props = withDefaults(defineProps<{
  messages: ChatMessage[]
  maximized: boolean
  sending: boolean
  counterpartName?: string
  fullCodePath?: string
  resourceType?: string
  resourceTemplateType?: string
  streamingDisplayLength: number
  renderMarkdown: (text: string) => string
  formatMessageTime: (value: string) => string
  getFileGroupsFromCalls: (calls: ChatMessageToolCall[]) => OutputFileGroup[]
  getDisplayFieldsFromCalls: (calls: ChatMessageToolCall[]) => OutputDisplayField[]
  pendingInteraction?: WorkspaceInteraction | null
}>(), {
  counterpartName: '',
  fullCodePath: '',
  resourceType: '',
  resourceTemplateType: '',
  pendingInteraction: null,
})

const emit = defineEmits<{
  (e: 'confirm-prd', payload: { remark: string; prd: unknown }): void
  (e: 'view', interaction: WorkspaceInteraction): void
  (e: 'revise', payload: { interaction: WorkspaceInteraction; text: string }): void
  (e: 'cancel', interaction: WorkspaceInteraction): void
  (e: 'confirm', interaction: WorkspaceInteraction): void
}>()

interface RuntimeTimer {
  messageIndex: number
  createdAt: string
  startedAt: number
  completedAt?: number
}

const assistantTimer = ref<RuntimeTimer | null>(null)
const assistantTimerNow = ref(Date.now())
let assistantTimerInterval: ReturnType<typeof setInterval> | null = null

function getAssistantOutputSize(message: ChatMessage): number {
  let size = message.content?.length ?? 0
  if (message.blocks?.length) {
    for (const block of message.blocks) {
      if (block.type === 'content') {
        size += block.text.length
      } else {
        const visibleCalls = getVisibleToolCallsFromCalls(block.calls)
        size += visibleCalls.length
        for (const call of visibleCalls) {
          size += (call.arguments?.length ?? 0) + (call.result?.length ?? 0) + (call.error?.length ?? 0)
        }
      }
    }
  }
  if (message.tool_calls?.length) {
    const visibleCalls = getVisibleToolCallsFromCalls(message.tool_calls)
    size += visibleCalls.length
    for (const call of visibleCalls) {
      size += (call.arguments?.length ?? 0) + (call.result?.length ?? 0) + (call.error?.length ?? 0)
    }
  }
  if (message.model_context_plans?.length || message.model_context_plan) {
    size += message.model_context_plans?.length || 1
  }
  return size
}

function hasAssistantVisibleOutput(message: ChatMessage): boolean {
  return message.role === 'assistant' && getAssistantOutputSize(message) > 0
}

function isAssistantTimerTarget(message: ChatMessage, msgIndex: number): boolean {
  const timer = assistantTimer.value
  return !!timer && message.role === 'assistant' && timer.messageIndex === msgIndex && timer.createdAt === (message.created_at || '')
}

function syncAssistantTimerInterval() {
  const timer = assistantTimer.value
  if (timer && timer.completedAt == null) {
    if (assistantTimerInterval == null) {
      assistantTimerInterval = setInterval(() => {
        assistantTimerNow.value = Date.now()
      }, 250)
    }
    return
  }
  if (assistantTimerInterval != null) {
    clearInterval(assistantTimerInterval)
    assistantTimerInterval = null
  }
}

function syncAssistantTimer() {
  const now = Date.now()
  const lastIndex = props.messages.length - 1
  const lastMessage = props.messages[lastIndex]

  if (props.sending && lastMessage?.role === 'assistant' && hasAssistantVisibleOutput(lastMessage)) {
    const createdAt = lastMessage.created_at || ''
    const timer = assistantTimer.value
    if (!timer || timer.messageIndex !== lastIndex || timer.createdAt !== createdAt || timer.completedAt != null) {
      assistantTimer.value = { messageIndex: lastIndex, createdAt, startedAt: now }
      assistantTimerNow.value = now
    }
  } else if (!props.sending && assistantTimer.value && assistantTimer.value.completedAt == null) {
    assistantTimer.value = { ...assistantTimer.value, completedAt: now }
    assistantTimerNow.value = now
  }

  const timer = assistantTimer.value
  if (timer) {
    const targetMessage = props.messages[timer.messageIndex]
    if (!targetMessage || targetMessage.role !== 'assistant' || (targetMessage.created_at || '') !== timer.createdAt) {
      assistantTimer.value = null
    }
  }

  syncAssistantTimerInterval()
}

function formatDuration(ms: number): string {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  if (hours > 0) return t('miniWorkstation.durationHours', { hours, minutes, seconds })
  if (minutes > 0) return t('miniWorkstation.durationMinutes', { minutes, seconds })
  return t('miniWorkstation.durationSeconds', { seconds })
}

function getAssistantDurationLabel(message: ChatMessage, msgIndex: number): string {
  if (!isAssistantTimerTarget(message, msgIndex)) return ''
  const timer = assistantTimer.value
  if (!timer) return ''
  const end = timer.completedAt ?? assistantTimerNow.value
  return formatDuration(end - timer.startedAt)
}

function isAssistantTimerRunning(message: ChatMessage, msgIndex: number): boolean {
  const timer = assistantTimer.value
  return isAssistantTimerTarget(message, msgIndex) && !!timer && timer.completedAt == null
}

function renderContentBlock(text: string, msgIndex: number, blockIndex: number, blockCount: number): string {
  const isStreamingTail =
    props.sending &&
    msgIndex === props.messages.length - 1 &&
    blockIndex === blockCount - 1

  return props.renderMarkdown(isStreamingTail ? text.slice(0, props.streamingDisplayLength) : text)
}

function getPrdCallsFromCalls(calls: ChatMessageToolCall[]): ChatMessageToolCall[] {
  return calls.filter((call) => call.name === 'write_prd' && call.status === 'ok' && call.result_data != null)
}

function getVisibleToolCallsFromCalls(calls: ChatMessageToolCall[]): ChatMessageToolCall[] {
  return getVisibleWorkspaceToolCalls(calls)
}

function getToolDisplayName(call: ChatMessageToolCall): string {
  return getWorkspaceToolCallDisplayName(call)
}

function getBuildWorkspaceFailureCallsFromCalls(calls: ChatMessageToolCall[]): ChatMessageToolCall[] {
  return calls.filter((call) =>
    call.name === 'build_workspace' &&
    call.result_data != null &&
    typeof call.result_data === 'object' &&
    (call.result_data as { kind?: string }).kind === 'agent_app_build_failure'
  )
}

type StageInteractionArtifact = Record<string, unknown> & {
  kind?: string
  interaction?: Partial<WorkspaceInteraction>
}

function getInteractionCardsFromCalls(calls: ChatMessageToolCall[]): WorkspaceInteraction[] {
  const interactions: WorkspaceInteraction[] = []
  for (const call of calls) {
    const interaction = buildWorkspaceInteractionFromArtifact(call.result_data)
    if (interaction) interactions.push(interaction)
  }
  return interactions
}

function buildWorkspaceInteractionFromArtifact(value: unknown): WorkspaceInteraction | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  const artifact = value as StageInteractionArtifact
  const rawInteraction = artifact.interaction
  if (!rawInteraction || typeof rawInteraction !== 'object') return null
  const status = typeof rawInteraction.status === 'string' ? rawInteraction.status.trim() : ''
  if (!status.startsWith('pending_')) return null
  const cardType = typeof rawInteraction.card_type === 'string' ? rawInteraction.card_type : fallbackCardType(artifact.kind, status)
  return {
    id: typeof rawInteraction.id === 'string' ? rawInteraction.id : getStageArtifactKey(artifact),
    card_type: cardType,
    artifact_kind: typeof rawInteraction.artifact_kind === 'string' ? rawInteraction.artifact_kind : artifact.kind,
    status,
    blocking: typeof rawInteraction.blocking === 'boolean' ? rawInteraction.blocking : true,
    title: typeof rawInteraction.title === 'string' ? rawInteraction.title : fallbackInteractionTitle(cardType),
    description: typeof rawInteraction.description === 'string' ? rawInteraction.description : undefined,
    help_text: typeof rawInteraction.help_text === 'string' ? rawInteraction.help_text : undefined,
    view_text: typeof rawInteraction.view_text === 'string' ? rawInteraction.view_text : undefined,
    confirm_text: typeof rawInteraction.confirm_text === 'string' ? rawInteraction.confirm_text : undefined,
    revise_text: typeof rawInteraction.revise_text === 'string' ? rawInteraction.revise_text : undefined,
    cancel_text: typeof rawInteraction.cancel_text === 'string' ? rawInteraction.cancel_text : undefined,
    target_role_on_confirm: typeof rawInteraction.target_role_on_confirm === 'string' ? rawInteraction.target_role_on_confirm : undefined,
    allowed_actions: Array.isArray(rawInteraction.allowed_actions) ? rawInteraction.allowed_actions.map(String) : undefined,
    artifact
  }
}

function isActiveInteraction(interaction: WorkspaceInteraction): boolean {
  if (!props.pendingInteraction) return false
  return getInteractionKey(interaction) === getInteractionKey(props.pendingInteraction)
}

function getInteractionKey(interaction: WorkspaceInteraction): string {
  return interaction.id || getStageArtifactKey(interaction.artifact) || `${interaction.status}:${interaction.card_type}`
}

function getStageArtifactKey(artifact: unknown): string {
  try {
    return JSON.stringify(artifact)
  } catch {
    return String(artifact)
  }
}

function fallbackCardType(kind: unknown, status: string): string {
  if (kind === 'agent_app_build_failure' || status === 'pending_build_repair') return 'build_repair'
  if (kind === 'agent_app_prd' || status === 'pending_confirmation') return 'prd_confirmation'
  return 'stage_confirmation'
}

function fallbackInteractionTitle(cardType: string): string {
  if (cardType === 'build_repair') return t('miniWorkstation.buildRepairTitle')
  if (cardType === 'prd_confirmation') return t('miniWorkstation.interactionPrdTitle')
  return t('miniWorkstation.interactionWaitingTitle')
}

function getAssistantModelLabel(message: ChatMessage): string {
  if (message.llm_config_name) return message.llm_config_name
  const provider = (message.llm_provider || '').trim()
  const model = (message.llm_model || '').trim()
  if (provider && model) return `${provider}/${model}`
  return model || provider
}

function getAssistantCacheLabel(message: ChatMessage): string {
  const usage = message.llm_usage
  if (!usage || usage.total_tokens <= 0) return ''
  if (usage.cached_tokens_reported === false) return t('miniWorkstation.cacheNotReported')
  const cached = Math.max(0, usage.cached_tokens || 0)
  const prompt = Math.max(0, usage.prompt_tokens || 0)
  const rate = prompt > 0 ? Math.round((cached / prompt) * 100) : 0
  return t('miniWorkstation.cacheLabel', { cached: formatTokenCount(cached), rate })
}

function getAssistantCacheTitle(message: ChatMessage): string {
  const usage = message.llm_usage
  if (!usage) return ''
  if (usage.cached_tokens_reported === false) {
    return [
      t('miniWorkstation.cacheMissingTitle'),
      t('miniWorkstation.inputTokens', { tokens: formatTokenCount(usage.prompt_tokens) }),
      t('miniWorkstation.outputTokens', { tokens: formatTokenCount(usage.completion_tokens) }),
      t('miniWorkstation.totalTokens', { tokens: formatTokenCount(usage.total_tokens) })
    ].join(' · ')
  }
  return [
    t('miniWorkstation.inputTokens', { tokens: formatTokenCount(usage.prompt_tokens) }),
    t('miniWorkstation.cachedTokens', { tokens: formatTokenCount(usage.cached_tokens) }),
    t('miniWorkstation.outputTokens', { tokens: formatTokenCount(usage.completion_tokens) }),
    t('miniWorkstation.totalTokens', { tokens: formatTokenCount(usage.total_tokens) })
  ].join(' · ')
}

function formatTokenCount(value: number): string {
  const n = Math.max(0, Math.round(value || 0))
  if (n >= 1_000_000) return `${trimTrailingZeros(n / 1_000_000)}m`
  if (n >= 1000) return `${trimTrailingZeros(n / 1000)}k`
  return String(n)
}

function trimTrailingZeros(value: number): string {
  return value.toFixed(value >= 10 ? 0 : 1).replace(/\.0$/, '')
}

watch(
  () => [
    props.sending ? 'sending' : 'idle',
    props.messages.length,
    props.messages.map((message, index) => `${index}:${message.role}:${message.created_at || ''}:${getAssistantOutputSize(message)}`).join('|')
  ].join(':'),
  syncAssistantTimer,
  { immediate: true }
)

onBeforeUnmount(() => {
  if (assistantTimerInterval != null) clearInterval(assistantTimerInterval)
})
</script>

<style scoped>
.mini-ws-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 80px;
  color: var(--text-placeholder);
  font-size: 13px;
  letter-spacing: 0;
  text-transform: none;
}

.mini-msg {
  margin-bottom: 12px;
  animation: miniMsgEnter 0.22s ease-out;
}
.mini-msg-user {
  display: flex;
  flex-direction: column;
  gap: 7px;
  align-items: flex-end;
}
.mini-msg-user-header {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 7px;
  max-width: 100%;
}
.mini-msg-user-display {
  flex-shrink: 1;
  min-width: 0;
}
.mini-msg-user-display :deep(.user-display-wrapper) {
  display: inline-flex;
  color: var(--text-primary);
}
.mini-msg-time {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-placeholder);
  margin-top: 2px;
}
.mini-msg-assistant-header {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  margin-bottom: 4px;
}
.mini-msg-model {
  flex-shrink: 1;
  min-width: 0;
  max-width: 220px;
  overflow: hidden;
  padding: 2px 6px;
  border: 1px solid rgba(var(--color-primary-rgb), 0.18);
  border-radius: 999px;
  background: rgba(var(--color-primary-rgb), 0.08);
  color: var(--color-primary);
  font-size: 10px;
  font-weight: 700;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mini-msg-cache {
  flex-shrink: 0;
  padding: 2px 6px;
  border: 1px solid rgba(var(--color-success-rgb), 0.18);
  border-radius: 999px;
  background: rgba(var(--color-success-rgb), 0.08);
  color: var(--color-success);
  font-size: 10px;
  font-weight: 700;
  line-height: 1.2;
  white-space: nowrap;
}
.mini-msg-model-context {
  margin-bottom: 6px;
}
.mini-msg-output-duration {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  padding: 2px 6px;
  border: 1px solid var(--border-light);
  border-radius: 999px;
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  font-size: 10px;
  font-weight: 700;
  line-height: 1.2;
}
.mini-msg-output-duration--running {
  border-color: rgba(var(--color-primary-rgb), 0.2);
  color: var(--color-primary);
}
.mini-msg-output-duration-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
  box-shadow: none;
}
.mini-msg-assistant {
  padding: 8px 10px;
  border: 1px solid var(--border-light);
  border-radius: 10px;
  background: var(--bg-primary);
  box-shadow: none;
}

.mini-content-block {
  margin: 0;
  font-size: 12px;
  line-height: 1.6;
  font-family: inherit;
  color: var(--text-primary);
  word-break: break-word;
}
.mini-md-content :deep(p) {
  margin: 0 0 6px;
}
.mini-md-content :deep(p:last-child) {
  margin-bottom: 0;
}
.mini-md-content :deep(ul),
.mini-md-content :deep(ol) {
  margin: 4px 0;
  padding-left: 18px;
}
.mini-md-content :deep(li) {
  margin: 2px 0;
}
.mini-md-content :deep(code) {
  background: var(--bg-tertiary);
  border: 1px solid var(--border-light);
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 11px;
  font-family: 'SF Mono', 'Fira Code', monospace;
  color: var(--text-primary);
}
.mini-md-content :deep(pre) {
  background: var(--bg-tertiary);
  color: var(--text-primary);
  border: 1px solid var(--border-light);
  padding: 8px 10px;
  border-radius: 10px;
  overflow-x: auto;
  margin: 6px 0;
  font-size: 11px;
  line-height: 1.5;
  box-shadow: none;
}
.mini-md-content :deep(pre code) {
  background: none;
  padding: 0;
  font-size: inherit;
  color: inherit;
}
.mini-md-content :deep(h1),
.mini-md-content :deep(h2),
.mini-md-content :deep(h3),
.mini-md-content :deep(h4) {
  margin: 8px 0 4px;
  font-size: 13px;
  font-weight: 600;
}
.mini-md-content :deep(h1) { font-size: 15px; }
.mini-md-content :deep(h2) { font-size: 14px; }
.mini-md-content :deep(blockquote) {
  margin: 4px 0;
  padding: 2px 8px;
  border-left: 3px solid rgba(var(--color-primary-rgb), 0.24);
  color: var(--text-secondary);
  background: var(--bg-tertiary);
}
.mini-md-content :deep(table) {
  border-collapse: collapse;
  margin: 6px 0;
  font-size: 11px;
  width: 100%;
}
.mini-md-content :deep(th),
.mini-md-content :deep(td) {
  border: 1px solid var(--border-light);
  padding: 3px 6px;
}
.mini-md-content :deep(th) {
  background: var(--bg-tertiary);
  font-weight: 600;
}
.mini-md-content :deep(a) {
  color: var(--color-primary);
  text-decoration: none;
}
.mini-md-content :deep(img) {
  display: block;
  width: auto;
  max-width: min(100%, 260px);
  max-height: 180px;
  margin: 6px 0;
  object-fit: contain;
  border: 1px solid var(--border-light);
  border-radius: 8px;
  background: var(--bg-tertiary);
  box-shadow: none;
}
.mini-msg--maximized .mini-md-content :deep(img) {
  max-width: min(100%, 460px);
  max-height: 300px;
}
.mini-md-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--border-light);
  margin: 8px 0;
}
.mini-tools-block {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin: 4px 0;
}
.mini-tool-tag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 11px;
  color: var(--text-secondary);
  border: 1px solid var(--border-light);
  background: var(--bg-tertiary);
  padding: 2px 6px;
  border-radius: 999px;
}

.mini-msg-user-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 10px;
  border: 1px solid rgba(var(--color-primary-rgb), 0.18);
  border-radius: 10px;
  background: rgba(var(--color-primary-rgb), 0.08);
  color: var(--text-primary);
}
.mini-msg-files {
  margin: 4px 0;
}

.mini-msg-prd-preview {
  margin: 6px 0;
}
.mini-msg-build-diagnostics {
  --el-text-color-primary: var(--text-primary);
  --el-text-color-regular: var(--text-primary);
  --el-text-color-secondary: var(--text-secondary);
  --el-border-color-lighter: rgba(248, 113, 113, 0.28);
  --el-border-color-extra-light: rgba(248, 113, 113, 0.14);
  --el-fill-color-blank: var(--bg-primary);
  --el-fill-color-lighter: var(--bg-tertiary);

  margin: 6px 0;
  border-radius: 8px;
}
.mini-msg-prd-preview :deep(.prd-preview) {
  --prd-bg: var(--bg-primary);
  --prd-surface: var(--bg-secondary);
  --prd-surface-strong: var(--bg-secondary);
  --prd-control-bg: var(--bg-tertiary);
  --prd-muted-bg: var(--bg-tertiary);
  --prd-primary-bg: rgba(var(--color-primary-rgb), 0.08);
  --prd-danger-bg: rgba(248, 113, 113, 0.14);
  --prd-warning-bg: rgba(251, 191, 36, 0.13);
  --prd-border: var(--border-light);
  --prd-border-soft: var(--border-light);
  --prd-shadow: none;
  --el-text-color-primary: var(--text-primary);
  --el-text-color-regular: var(--text-primary);
  --el-text-color-secondary: var(--text-secondary);
  --el-text-color-placeholder: var(--text-placeholder);
  --el-border-color: var(--border-light);
  --el-border-color-light: var(--border-light);
  --el-border-color-lighter: var(--border-light);
  --el-border-color-extra-light: var(--border-light);
  --el-fill-color-blank: var(--bg-secondary);
  --el-fill-color-extra-light: var(--bg-primary);
  --el-fill-color-light: var(--bg-tertiary);

  margin-top: 0;
  border-color: var(--border-light);
  background: var(--bg-primary);
  color: var(--text-primary);
}
.mini-msg-prd-preview :deep(.prd-preview-head),
.mini-msg-prd-preview :deep(.prd-section) {
  padding: 9px 10px;
}
.mini-msg-prd-preview :deep(.prd-table) {
  font-size: 11px;
}

.mini-msg-files :deep(.output-files-head) {
  font-size: 11px;
  margin-bottom: 4px;
}
.mini-msg-files :deep(.output-files-wrap) {
  padding: 6px;
  border-color: var(--border-light);
  background: var(--bg-primary);
}
.mini-msg-files :deep(.output-files-item) {
  padding: 6px;
  min-width: 120px;
  min-height: 0;
  border-color: var(--border-light);
  background: var(--bg-secondary);
}
.mini-msg-files :deep(.output-files-main) {
  grid-template-columns: 40px minmax(0, 1fr);
  gap: 8px;
}
.mini-msg-files :deep(.output-files-preview) {
  width: 40px;
  height: 40px;
}
.mini-msg-files :deep(.output-files-icon) {
  width: 40px;
  height: 40px;
  font-size: 18px;
}
.mini-msg-files :deep(.output-files-name) {
  font-size: 11px;
}
.mini-msg-files :deep(.output-files-meta) {
  font-size: 10px;
}
.mini-msg-files :deep(.output-files-footer) {
  align-items: flex-start;
  flex-direction: column;
  gap: 4px;
  padding-top: 6px;
}
.mini-msg-files :deep(.output-files-actions) {
  font-size: 11px;
  gap: 8px;
}

.mini-msg-display-fields {
  margin: 4px 0;
}
.mini-msg-display-fields :deep(.odf-head) {
  font-size: 11px;
  margin-bottom: 4px;
}
.mini-msg-display-fields :deep(.odf-card-header) {
  padding: 4px 8px;
}
.mini-msg-display-fields :deep(.odf-label) {
  font-size: 11px;
}
.mini-msg-display-fields :deep(.odf-value) {
  padding: 4px 8px;
}
.mini-msg-display-fields :deep(.odf-pre) {
  font-size: 11px;
}

@keyframes miniMsgEnter {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
