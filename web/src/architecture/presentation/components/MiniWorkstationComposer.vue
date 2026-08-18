<template>
  <div v-if="attachedFiles.length > 0" class="mini-ws-files">
    <el-tag
      v-for="(file, idx) in attachedFiles"
      :key="`${file.ref || file.name}-${idx}`"
      size="small"
      closable
      @close="removeFile(idx)"
    >
      {{ file.source_name || file.name }}
    </el-tag>
  </div>

  <div
    class="mini-ws-input"
    :class="{ 'mini-ws-input--schedule': variant === 'schedule' }"
    data-testid="mini-workstation-composer"
    @click="handleContainerClick"
  >
    <div class="mini-composer-left-actions">
      <el-upload
        :auto-upload="false"
        :show-file-list="false"
        :on-change="onFileChange"
        :disabled="uploading || blocked"
        class="mini-upload-btn"
      >
        <el-button :icon="Paperclip" link :loading="uploading" size="small" :title="t('miniWorkstation.uploadFile')" />
      </el-upload>
      <el-button
        v-if="variant === 'schedule' && expandable"
        :icon="FullScreen"
        link
        size="small"
        class="mini-expand-editor-btn"
        :title="t('miniWorkstation.expandEditor')"
        @click="openExpandedEditor"
      />
      <slot name="left-actions" />
    </div>

    <div class="mini-input-wrap" :class="{ 'is-blocked': blocked }">
      <MiniWorkstationResourceIdentity
        v-if="variant !== 'schedule'"
        class="mini-path-pill"
        :name="displayPath"
        :full-code-path="fullCodePath"
        :resource-type="resourceType"
        :resource-template-type="resourceTemplateType"
      />
      <span v-if="blocked" class="mini-blocked-pill" :title="blockedLabel || t('miniWorkstation.blockingGeneric')">
        {{ blockedLabel || t('miniWorkstation.blockingGeneric') }}
      </span>
      <StructuredPromptComposer
        ref="structuredInputRef"
        class="mini-structured-input"
        :model-value="inputText"
        :placeholder="composerPlaceholder"
        :disabled="blocked"
        :submit-on-enter="composerSubmitOnEnter"
        :show-toolbar="variant === 'schedule'"
        :compact="variant !== 'schedule'"
        :min-rows="variant === 'schedule' ? 6 : 1"
        :max-rows="variant === 'schedule' ? 14 : 4"
        :full-code-path="fullCodePath"
        :mention-panel-placement="mentionPanelPlacement"
        editor-test-id="mini-workstation-input"
        @update:model-value="emitInput"
        @enter="handleComposerEnter"
      />
    </div>

    <div v-if="variant !== 'schedule'" class="mini-action-stack">
      <el-select
        :model-value="selectedLLMConfigId"
        :placeholder="t('miniWorkstation.defaultModel')"
        filterable
        :loading="llmLoading"
        teleported
        fit-input-width
        :persistent="false"
        placement="top-start"
        :popper-options="modelSelectPopperOptions"
        popper-class="mini-ws-model-select-popper"
        class="mini-ws-model-select"
        @update:model-value="emit('update:selectedLLMConfigId', Number($event))"
        @visible-change="onLLMSelectVisibleChange"
      >
        <el-option :label="t('miniWorkstation.defaultModel')" :value="0" />
        <el-option
          v-for="llm in llmList"
          :key="llm.id"
          :label="llmOptionLabel(llm)"
          :value="llm.id"
        />
      </el-select>
      <div class="mini-action-row">
        <el-button
          v-if="sending || sessionRunning"
          type="danger"
          size="small"
          :loading="stopping"
          data-testid="mini-workstation-stop"
          class="mini-stop-btn"
          @click="$emit('stop')"
        >
          <el-icon><VideoPause /></el-icon>
          {{ t('miniWorkstation.stop') }}
        </el-button>
        <el-button
          v-else
          type="primary"
          size="small"
          :disabled="blocked || !fullCodePath || (!inputText.trim() && attachedFiles.length === 0)"
          data-testid="mini-workstation-send"
          class="mini-send-btn"
          @click="$emit('send')"
        >
          {{ t('miniWorkstation.send') }}
        </el-button>
        <el-tooltip
          :content="miniHideShortcutHint"
          placement="top"
          effect="light"
          popper-class="mini-workstation-tooltip-popper"
        >
          <button
            type="button"
            class="mini-hide-btn"
            :title="miniHideShortcutHint"
            data-testid="mini-workstation-collapse"
            @click="$emit('collapse')"
          >
            <el-icon><Close /></el-icon>
          </button>
        </el-tooltip>
      </div>
    </div>
  </div>

  <Teleport to="body">
    <Transition name="mini-composer-expanded">
      <div
        v-if="expandedEditorVisible"
        class="mini-composer-expanded-backdrop"
        @click.self="cancelExpandedEditor"
      >
        <section class="mini-composer-expanded-panel" role="dialog" aria-modal="true">
          <header class="mini-composer-expanded-header">
            <div class="mini-composer-expanded-title-block">
              <div class="mini-composer-expanded-kicker">{{ t('miniWorkstation.expandedComposerKicker') }}</div>
              <h2 class="mini-composer-expanded-title">{{ expandedTitle || t('miniWorkstation.expandedComposerTitle') }}</h2>
              <div class="mini-composer-expanded-subtitle">
                {{ expandedSubtitle || displayPath }}
              </div>
            </div>
            <div class="mini-composer-expanded-actions">
              <el-button @click="cancelExpandedEditor">{{ t('common.cancel') }}</el-button>
              <el-button type="primary" :icon="Check" @click="saveExpandedEditor">
                {{ expandedSaveLabel || t('common.save') }}
              </el-button>
              <button
                type="button"
                class="mini-composer-expanded-close"
                :aria-label="t('miniWorkstation.closeExpandedComposer')"
                @click="cancelExpandedEditor"
              >
                <el-icon><Close /></el-icon>
              </button>
            </div>
          </header>

          <div class="mini-composer-expanded-body">
            <StructuredPromptComposer
              ref="expandedInputRef"
              class="mini-expanded-structured-input"
              :model-value="expandedDraft"
              :placeholder="composerPlaceholder"
              :disabled="blocked"
              :submit-on-enter="false"
              :show-toolbar="true"
              :compact="false"
              :min-rows="18"
              :max-rows="36"
              :full-code-path="fullCodePath"
              mention-panel-placement="above"
              editor-test-id="mini-workstation-expanded-input"
              @update:model-value="expandedDraft = $event"
            />
          </div>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Close, FullScreen, Paperclip, VideoPause } from '@element-plus/icons-vue'
import type { LLMInfo } from '@/architecture/presentation/context/api/agent'
import type { WorkspaceChatMessageFile } from '@/architecture/presentation/context/api/workspace'
import StructuredPromptComposer from './StructuredPromptComposer.vue'
import MiniWorkstationResourceIdentity from './MiniWorkstationResourceIdentity.vue'

interface FocusableInput {
  focus: () => void
}

const props = withDefaults(defineProps<{
  fullCodePath: string
  dirName?: string
  resourceType?: string
  resourceTemplateType?: string
  attachedFiles: WorkspaceChatMessageFile[]
  uploading: boolean
  inputText: string
  sending: boolean
  sessionRunning?: boolean
  stopping: boolean
  selectedLLMConfigId: number
  llmList: LLMInfo[]
  llmLoading: boolean
  queuedCount: number
  blocked?: boolean
  blockedLabel?: string
  blockedPlaceholder?: string
  registerInputRef: (el: FocusableInput | null) => void
  onLLMSelectVisibleChange: (visible: boolean) => void
  onFileChange: (uploadFileObj: { raw?: File }) => void | Promise<void>
  removeFile: (index: number) => void
  onInputEnter: (event: KeyboardEvent) => void
  toggleShortcutLabel?: string
  variant?: 'chat' | 'schedule'
  placeholder?: string
  expandable?: boolean
  expandedTitle?: string
  expandedSubtitle?: string
  expandedSaveLabel?: string
  mentionPanelPlacement?: 'above' | 'below'
}>(), {
  variant: 'chat',
  resourceType: '',
  resourceTemplateType: '',
  placeholder: '',
  expandable: true,
  expandedTitle: '',
  expandedSubtitle: '',
  expandedSaveLabel: '',
  mentionPanelPlacement: 'above',
})

const emit = defineEmits<{
  (e: 'update:inputText', value: string): void
  (e: 'update:selectedLLMConfigId', value: number): void
  (e: 'send'): void
  (e: 'stop'): void
  (e: 'collapse'): void
  (e: 'expanded-save', value: string): void
}>()

const { t } = useI18n()
const structuredInputRef = ref<InstanceType<typeof StructuredPromptComposer> | null>(null)
const expandedInputRef = ref<InstanceType<typeof StructuredPromptComposer> | null>(null)
const expandedEditorVisible = ref(false)
const expandedDraft = ref('')

const modelSelectPopperOptions = {
  strategy: 'fixed' as const,
}

function llmProtocolLabel(protocol: string) {
  switch ((protocol || '').trim()) {
    case 'openai_responses':
      return 'Responses'
    case 'anthropic_messages':
      return 'Messages'
    default:
      return 'Chat'
  }
}

function llmEndpointLabel(llm: LLMInfo) {
  const endpoint = (llm.endpoint_path || '').trim()
  const base = (llm.api_base || '').trim().replace(/\/+$/, '')
  if (!base) {
    return endpoint
  }
  try {
    const url = new URL(base)
    const basePath = url.pathname.replace(/\/+$/, '')
    const endpointPath = endpoint ? (endpoint.startsWith('/') ? endpoint : `/${endpoint}`) : ''
    return `${url.host}${basePath}${endpointPath}`
  } catch {
    if (endpoint) {
      return `${base}${endpoint.startsWith('/') ? endpoint : `/${endpoint}`}`.replace(/^https?:\/\//, '')
    }
    return base.replace(/^https?:\/\//, '')
  }
}

function llmOptionLabel(llm: LLMInfo) {
  const endpoint = llmEndpointLabel(llm)
  const suffix = endpoint ? ` · ${endpoint}` : ''
  return `#${llm.id} ${llm.name} (${llm.model} · ${llmProtocolLabel(llm.protocol)}${suffix})`
}

const miniHideShortcutHint = computed(() => {
  const toggleShortcut = (props.toggleShortcutLabel || '').trim()
  return toggleShortcut
    ? t('miniWorkstation.closeBackgroundHintWithShortcut', { shortcut: toggleShortcut })
    : t('miniWorkstation.closeBackgroundHint')
})

const composerPlaceholder = computed(() => {
  if (props.blocked) {
    return props.blockedPlaceholder || t('miniWorkstation.blockedPlaceholder')
  }
  const placeholder = props.placeholder.trim()
  if (placeholder) {
    return placeholder
  }
  if (props.variant === 'schedule') {
    return t('miniWorkstation.schedulePlaceholder')
  }
  return t('miniWorkstation.chatPlaceholder')
})

const composerSubmitOnEnter = computed(() => props.variant !== 'schedule')

const displayPath = computed(() => {
  const label = (props.dirName || '').trim()
  if (label) return label
  if (!props.fullCodePath) return t('miniWorkstation.noDirectorySelected')
  const parts = props.fullCodePath.split('/').filter(Boolean)
  return parts[parts.length - 1] || props.fullCodePath
})

watch(structuredInputRef, () => {
  registerStructuredInputRef()
}, { flush: 'post' })

onMounted(() => {
  void nextTick(registerStructuredInputRef)
})

onBeforeUnmount(() => {
  props.registerInputRef(null)
})

function registerStructuredInputRef() {
  props.registerInputRef(structuredInputRef.value)
}

function emitInput(value: string) {
  if (props.blocked) return
  emit('update:inputText', value)
}

function handleComposerEnter(event: KeyboardEvent) {
  if (props.blocked) return
  props.onInputEnter(event)
}

function handleContainerClick(event: MouseEvent) {
  if (props.blocked) return
  const target = event.target as HTMLElement
  if (target.closest('button') || target.closest('a') || target.closest('.mini-path-pill') || target.closest('.el-select') || target.closest('.el-upload')) {
    return
  }
  if (target.closest('.spc-editor')) {
    return
  }
  structuredInputRef.value?.focus()
}

function openExpandedEditor() {
  if (props.blocked) return
  expandedDraft.value = props.inputText
  expandedEditorVisible.value = true
  void nextTick(() => expandedInputRef.value?.focus())
}

function saveExpandedEditor() {
  emit('update:inputText', expandedDraft.value)
  emit('expanded-save', expandedDraft.value)
  expandedEditorVisible.value = false
  void nextTick(() => structuredInputRef.value?.focus())
}

function cancelExpandedEditor() {
  expandedEditorVisible.value = false
  void nextTick(() => structuredInputRef.value?.focus())
}
</script>

<style scoped>
.mini-ws-files {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 7px 12px;
  border-top: 1px solid var(--border-light);
  background: transparent;
}

.mini-ws-files :deep(.el-tag) {
  border-color: var(--border-light);
  background: var(--bg-tertiary);
  color: var(--text-primary);
}

.mini-ws-input {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: start;
  gap: 12px;
  box-sizing: border-box;
  min-height: 50px;
  padding: 4px 10px;
  position: relative;
  border: 1px solid transparent;
  border-radius: 14px;
  background: var(--el-fill-color-light);
  box-shadow: none;
  backdrop-filter: none;
  transition: background-color 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
}

.mini-ws-input:hover {
  border-color: var(--border-light);
  background: var(--el-fill-color);
}

.mini-ws-input:focus-within {
  border-color: transparent;
  background: var(--el-fill-color-light);
  box-shadow: none;
}

.mini-ws-input--schedule {
  grid-template-columns: auto minmax(0, 1fr);
  min-height: 180px;
  align-items: stretch;
  background: var(--el-fill-color-light);
  border: 1px solid var(--border-light);
  box-shadow: none;
  backdrop-filter: none;
  border-radius: var(--border-radius-lg);
  transition: all 0.25s cubic-bezier(0.25, 0.8, 0.25, 1);

  &:hover {
    border-color: var(--border-base);
    background: var(--el-fill-color-light);
  }
  
  &:focus-within {
    background: var(--el-fill-color-light);
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px rgba(var(--color-primary-rgb), 0.12);
  }
}

html.dark .mini-ws-input--schedule {
  background: var(--el-fill-color-light);
  border-color: var(--border-light);
}

.mini-ws-input--schedule .mini-structured-input :deep(.spc-editor),
.mini-ws-input--schedule .mini-structured-input :deep(.spc-preview) {
  color: var(--text-primary);
  -webkit-text-fill-color: var(--text-primary);
}

.mini-ws-input--schedule .mini-upload-btn :deep(.el-button) {
  background: transparent;
  border-color: transparent;
  color: var(--text-secondary);
}

.mini-ws-input--schedule .mini-upload-btn :deep(.el-button):hover {
  background: rgba(var(--color-primary-rgb), 0.08);
  color: var(--color-primary);
  border-color: transparent;
  box-shadow: none;
}

.mini-composer-left-actions {
  flex-shrink: 0;
  min-width: 142px;
  display: flex;
  flex-direction: row;
  align-items: center;
  align-self: start;
  padding-top: 1px;
  gap: 8px;
}

.mini-ws-input--schedule .mini-composer-left-actions {
  min-width: auto;
}

.mini-upload-btn {
  flex-shrink: 0;
}

.mini-expand-editor-btn.el-button {
  width: 40px;
  height: 40px;
  border: 1px solid transparent;
  border-radius: 8px;
  color: var(--color-primary);
  background: var(--el-fill-color-light);
  box-shadow: none;
  transition: background-color 0.2s ease, border-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.mini-expand-editor-btn.el-button:hover {
  border-color: var(--border-light);
  color: var(--color-primary-light-1);
  background: var(--el-fill-color);
}

.mini-upload-btn :deep(.el-button) {
  width: 40px;
  height: 40px;
  border: 1px solid transparent;
  border-radius: 8px;
  color: var(--color-primary);
  background: var(--el-fill-color-light);
  box-shadow: none;
  transition: background-color 0.2s ease, border-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.mini-upload-btn :deep(.el-button:hover) {
  border-color: var(--border-light);
  color: var(--color-primary-light-1);
  background: var(--el-fill-color);
}

.mini-input-wrap {
  position: relative;
  min-width: 0;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: start;
  gap: 10px;
  box-sizing: border-box;
  min-height: 36px;
  padding: 0 10px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: var(--el-fill-color-light);
  cursor: text;
  transition: background-color 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
}

.mini-input-wrap:hover {
  border-color: var(--border-light);
  background: var(--el-fill-color);
}

.mini-input-wrap:focus-within {
  border-color: transparent;
  background: var(--el-fill-color-light);
  box-shadow: none;
}

.mini-input-wrap.is-blocked {
  cursor: default;
}

.mini-ws-input--schedule .mini-input-wrap {
  min-height: 160px;
  grid-template-columns: minmax(0, 1fr);
  align-items: stretch;
  background: var(--bg-primary);
}

.mini-path-pill {
  max-width: 148px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  padding: 0 10px;
  overflow: hidden;
  border: 1px solid rgba(var(--color-primary-rgb), 0.18);
  border-radius: 8px;
  background: rgba(var(--color-primary-rgb), 0.08);
  color: var(--color-primary);
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: default;
  margin-top: 4px;
}

.mini-blocked-pill {
  position: absolute;
  right: 10px;
  top: 5px;
  z-index: 2;
  max-width: min(180px, calc(100% - 116px));
  height: 24px;
  display: inline-flex;
  align-items: center;
  padding: 0 8px;
  overflow: hidden;
  border: 1px solid var(--border-light);
  border-radius: 8px;
  background: var(--bg-tertiary);
  color: #f6c76b;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: default;
}

.mini-structured-input {
  min-width: 0;
  width: 100%;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.mini-structured-input.is-focused {
  border-color: transparent;
  box-shadow: none;
}

.mini-structured-input :deep(.spc-editor),
.mini-structured-input :deep(.spc-preview) {
  color: var(--text-primary);
}

.mini-input-wrap.is-blocked .mini-structured-input :deep(.spc-editor) {
  padding-right: min(190px, 42%);
  cursor: not-allowed;
  color: var(--text-placeholder);
  -webkit-text-fill-color: var(--text-placeholder);
}

.mini-action-stack {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 1px;
}

.mini-ws-model-select {
  width: 112px;
  min-width: 0;
}

.mini-ws-model-select :deep(.el-select__wrapper) {
  min-height: 40px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  box-shadow: none;
  transition: background-color 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
}

.mini-ws-model-select :deep(.el-select__wrapper:hover) {
  border-color: var(--border-light);
  background: var(--el-fill-color);
}

.mini-ws-model-select :deep(.el-select__wrapper.is-focused) {
  border-color: var(--color-primary);
  background: var(--el-bg-color);
  box-shadow: none;
}

.mini-ws-model-select :deep(.el-select__placeholder),
.mini-ws-model-select :deep(.el-select__selected-item) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-primary);
}

.mini-action-row {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.mini-action-row :deep(.el-button + .el-button) {
  margin-left: 0;
}

.mini-send-btn,
.mini-stop-btn {
  flex-shrink: 0;
  min-width: 108px;
  min-height: 40px;
  border-radius: 8px;
  font-weight: 700;
  letter-spacing: 0;
  box-shadow: none;
}

.mini-send-btn.el-button--primary {
  border: 0;
  background: var(--color-primary);
  color: #ffffff;
}

.mini-hide-btn {
  width: 40px;
  height: 40px;
  min-width: 40px;
  display: grid;
  place-items: center;
  border: 1px solid transparent;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  color: var(--color-primary);
  transition: background-color 0.2s ease, border-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.mini-hide-btn:hover {
  border-color: var(--border-light);
  background: var(--el-fill-color);
  color: var(--color-primary-light-1);
}

:deep(.mini-ws--maximized) .mini-ws-input {
  padding: 4px 10px;
}

:deep(.mini-ws--maximized) .mini-ws-files {
  padding: 6px 24px;
}

.mini-composer-expanded-backdrop {
  position: fixed;
  inset: 0;
  z-index: 3600;
  display: flex;
  align-items: stretch;
  justify-content: center;
  padding: 28px;
  box-sizing: border-box;
  background: var(--bg-tertiary);
  backdrop-filter: blur(14px) saturate(118%);
}

.mini-composer-expanded-panel {
  width: min(1180px, 100%);
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
  border-radius: 12px;
  background: var(--bg-secondary);
  box-shadow:
    0 12px 32px rgba(15, 23, 42, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.4);
}

.mini-composer-expanded-header {
  flex-shrink: 0;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding: 22px 24px 18px;
  border-bottom: 1px solid var(--app-shell-panel-border, var(--el-border-color-lighter));
}

.mini-composer-expanded-title-block {
  min-width: 0;
}

.mini-composer-expanded-kicker {
  color: var(--el-color-primary);
  font-size: 12px;
  font-weight: 700;
  line-height: 1.4;
}

.mini-composer-expanded-title {
  margin: 5px 0 0;
  color: var(--el-text-color-primary);
  font-size: 24px;
  font-weight: 760;
  line-height: 1.25;
  letter-spacing: 0;
  word-break: break-word;
}

.mini-composer-expanded-subtitle {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.55;
  word-break: break-all;
}

.mini-composer-expanded-actions {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.mini-composer-expanded-close {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
  color: var(--el-text-color-secondary);
  cursor: pointer;
}

.mini-composer-expanded-close:hover {
  color: var(--el-text-color-primary);
  border-color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 8%, var(--el-fill-color-blank));
}

.mini-composer-expanded-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 22px 24px 26px;
}

.mini-expanded-structured-input {
  min-height: 100%;
  border-radius: 10px;
  background: var(--app-shell-panel-bg, var(--el-bg-color));
}

.mini-expanded-structured-input :deep(.spc-editor),
.mini-expanded-structured-input :deep(.spc-preview) {
  min-height: min(680px, calc(100vh - 220px)) !important;
  max-height: none !important;
  color: var(--el-text-color-regular);
}

.mini-composer-expanded-enter-active,
.mini-composer-expanded-leave-active {
  transition: opacity 0.18s ease;
}

.mini-composer-expanded-enter-active .mini-composer-expanded-panel,
.mini-composer-expanded-leave-active .mini-composer-expanded-panel {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.mini-composer-expanded-enter-from,
.mini-composer-expanded-leave-to {
  opacity: 0;
}

.mini-composer-expanded-enter-from .mini-composer-expanded-panel,
.mini-composer-expanded-leave-to .mini-composer-expanded-panel {
  opacity: 0;
  transform: translateY(28px);
}

@media (max-width: 720px) {
  .mini-ws-input {
    grid-template-columns: minmax(0, 1fr);
    align-items: stretch;
  }

  .mini-composer-left-actions,
  .mini-action-stack {
    min-width: 0;
  }

  .mini-action-stack {
    justify-content: space-between;
  }

  .mini-composer-expanded-backdrop {
    padding: 0;
  }

  .mini-composer-expanded-panel {
    min-height: 100%;
    border-radius: 0;
  }

  .mini-composer-expanded-header {
    flex-direction: column;
    padding: 18px 16px 14px;
  }

  .mini-composer-expanded-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .mini-composer-expanded-body {
    padding: 16px;
  }
}
</style>

<style lang="scss">
.mini-ws-model-select-popper.el-select__popper {
  max-width: min(360px, calc(100vw - 24px));
  overflow: hidden;
  border: 1px solid var(--border-light);
  background: var(--bg-secondary);
  box-shadow: 0 18px 46px rgba(0, 0, 0, 0.1);
}

.mini-ws-model-select-popper.el-select__popper:not([data-popper-placement]) {
  visibility: hidden;
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown {
  width: 100% !important;
  min-width: 0 !important;
  max-width: 100%;
  background: transparent;
}

.mini-ws-model-select-popper.el-select__popper .el-scrollbar,
.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__wrap,
.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__list {
  max-width: 100%;
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__item {
  max-width: 100%;
  padding-right: 28px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-secondary);
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__item span {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__item.is-hovering,
.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__item:hover {
  background: rgba(var(--color-primary-rgb), 0.08);
  color: var(--text-primary);
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__item.is-selected {
  color: var(--color-primary);
  background: rgba(var(--color-primary-rgb), 0.12);
}

.mini-ws-model-select-popper.el-select__popper .el-popper__arrow::before {
  background: var(--bg-tertiary);
}
</style>
