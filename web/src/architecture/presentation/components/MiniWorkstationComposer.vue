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
  >
    <div class="mini-composer-left-actions">
      <slot name="left-actions" />
      <el-tooltip
        v-if="expandable"
        :content="t('miniWorkstation.expandComposer')"
        placement="top"
        effect="light"
        popper-class="mini-workstation-tooltip-popper"
      >
        <el-button
          :icon="ArrowUp"
          link
          size="small"
          :disabled="blocked"
          class="mini-expand-editor-btn"
          :title="t('miniWorkstation.expandComposer')"
          @mousedown.stop
          @click.stop="openExpandedEditor"
        />
      </el-tooltip>
      <el-upload
        :auto-upload="false"
        :show-file-list="false"
        :on-change="onFileChange"
        :disabled="uploading || blocked"
        class="mini-upload-btn"
      >
        <el-button :icon="Paperclip" link :loading="uploading" size="small" :title="t('miniWorkstation.uploadFile')" />
      </el-upload>
    </div>

    <div class="mini-input-wrap" :class="{ 'is-blocked': blocked }">
      <span v-if="variant !== 'schedule'" class="mini-path-pill" :title="fullCodePath">{{ displayPath }}</span>
      <span v-if="blocked" class="mini-blocked-pill" :title="blockedLabel || t('miniWorkstation.blockingGeneric')">
        {{ blockedLabel || t('miniWorkstation.blockingGeneric') }}
      </span>
      <StructuredPromptComposer
        ref="structuredInputRef"
        class="mini-structured-input"
        :model-value="inputText"
        :placeholder="composerPlaceholder"
        :disabled="blocked"
        :submit-on-enter="true"
        :show-toolbar="variant === 'schedule'"
        :compact="variant !== 'schedule'"
        :min-rows="variant === 'schedule' ? 3 : 1"
        :max-rows="variant === 'schedule' ? 10 : 4"
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
          :label="`${llm.name} (${llm.model})`"
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
          type="primary"
          size="small"
          :disabled="blocked || sessionRunning || !fullCodePath || (!inputText.trim() && attachedFiles.length === 0)"
          data-testid="mini-workstation-send"
          class="mini-send-btn"
          @click="$emit('send')"
        >
          {{ sending ? (queuedCount > 0 ? t('miniWorkstation.queuedShort', { count: queuedCount }) : t('miniWorkstation.queued')) : t('miniWorkstation.send') }}
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
              mention-panel-placement="below"
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
import { ArrowUp, Check, Close, Paperclip, VideoPause } from '@element-plus/icons-vue'
import type { LLMInfo } from '@/architecture/presentation/context/api/agent'
import type { WorkspaceChatMessageFile } from '@/architecture/presentation/context/api/workspace'
import StructuredPromptComposer from './StructuredPromptComposer.vue'

interface FocusableInput {
  focus: () => void
}

const props = withDefaults(defineProps<{
  fullCodePath: string
  dirName?: string
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
  border-top: 1px solid rgba(96, 231, 255, 0.12);
  background: rgba(4, 12, 24, 0.46);
}

.mini-ws-files :deep(.el-tag) {
  border-color: rgba(34, 211, 238, 0.32);
  background: rgba(34, 211, 238, 0.1);
  color: var(--mini-cyber-text, #d8f8ff);
}

.mini-ws-input {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  box-sizing: border-box;
  min-height: 50px;
  padding: 4px 10px;
  position: relative;
  border: 1px solid rgba(130, 153, 190, 0.26);
  border-radius: 14px;
  background:
    linear-gradient(180deg, rgba(12, 18, 32, 0.84), rgba(8, 12, 22, 0.68)),
    rgba(8, 12, 22, 0.72);
  box-shadow:
    0 24px 70px rgba(0, 0, 0, 0.42),
    inset 0 1px 0 rgba(255, 255, 255, 0.06);
  backdrop-filter: blur(24px) saturate(140%);
}

.mini-ws-input--schedule {
  grid-template-columns: auto minmax(0, 1fr);
  min-height: 88px;
  align-items: stretch;
  background: var(--bg-secondary);
  border: 1px solid var(--border-base);
  box-shadow: none;
  backdrop-filter: none;
  border-radius: var(--border-radius-base);
}

html.dark .mini-ws-input--schedule {
  background: var(--bg-tertiary);
  border-color: var(--border-light);
}

.mini-ws-input--schedule .mini-structured-input :deep(.spc-editor),
.mini-ws-input--schedule .mini-structured-input :deep(.spc-preview) {
  color: var(--text-primary);
  -webkit-text-fill-color: var(--text-primary);
}

.mini-ws-input--schedule .mini-upload-btn :deep(.el-button) {
  background: transparent;
  border-color: var(--border-base);
  color: var(--text-regular);
}

.mini-ws-input--schedule .mini-upload-btn :deep(.el-button):hover {
  background: var(--bg-tertiary);
  color: var(--color-primary);
  border-color: var(--color-primary);
  box-shadow: none;
}

.mini-composer-left-actions {
  flex-shrink: 0;
  min-width: 142px;
  display: flex;
  flex-direction: row;
  align-items: center;
  align-self: center;
  gap: 8px;
}

.mini-ws-input--schedule .mini-composer-left-actions {
  min-width: auto;
}

.mini-upload-btn {
  flex-shrink: 0;
}

.mini-expand-editor-btn.el-button {
  width: 42px;
  height: 42px;
  border: 1px solid rgba(128, 151, 198, 0.22);
  border-radius: 8px;
  color: #8ed0ff;
  background: rgba(30, 42, 68, 0.72);
}

.mini-expand-editor-btn.el-button:hover {
  color: #ffffff;
  background: rgba(55, 163, 255, 0.16);
  box-shadow: 0 0 18px rgba(55, 163, 255, 0.14);
}

.mini-upload-btn :deep(.el-button) {
  width: 42px;
  height: 42px;
  border: 1px solid rgba(128, 151, 198, 0.22);
  border-radius: 8px;
  color: #8ed0ff;
  background: rgba(30, 42, 68, 0.72);
}

.mini-upload-btn :deep(.el-button:hover) {
  color: #ffffff;
  background: rgba(55, 163, 255, 0.16);
  box-shadow: 0 0 18px rgba(55, 163, 255, 0.14);
}

.mini-input-wrap {
  position: relative;
  min-width: 0;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  box-sizing: border-box;
  min-height: 36px;
  padding: 0 10px;
  border: 1px solid rgba(124, 146, 189, 0.16);
  border-radius: 10px;
  background: rgba(10, 16, 29, 0.32);
}

.mini-ws-input--schedule .mini-input-wrap {
  min-height: 76px;
  grid-template-columns: minmax(0, 1fr);
  align-items: stretch;
}

.mini-path-pill {
  max-width: 148px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  padding: 0 10px;
  overflow: hidden;
  border: 1px solid rgba(55, 163, 255, 0.25);
  border-radius: 8px;
  background: rgba(55, 163, 255, 0.13);
  color: #8ed0ff;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  border: 1px solid rgba(245, 158, 11, 0.28);
  border-radius: 8px;
  background: rgba(245, 158, 11, 0.1);
  color: #f6c76b;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  color: #e6f0ff;
}

.mini-input-wrap.is-blocked .mini-structured-input :deep(.spc-editor) {
  padding-right: min(190px, 42%);
  cursor: not-allowed;
  color: rgba(230, 240, 255, 0.44);
  -webkit-text-fill-color: rgba(230, 240, 255, 0.44);
}

.mini-action-stack {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.mini-ws-model-select {
  width: 112px;
  min-width: 0;
}

.mini-ws-model-select :deep(.el-select__wrapper) {
  min-height: 42px;
  border: 1px solid rgba(128, 151, 198, 0.22);
  border-radius: 8px;
  background: rgba(30, 42, 68, 0.78);
  box-shadow: none;
}

.mini-ws-model-select :deep(.el-select__placeholder),
.mini-ws-model-select :deep(.el-select__selected-item) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--mini-cyber-text, #d8f8ff);
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
  min-height: 42px;
  border-radius: 8px;
  font-weight: 700;
  letter-spacing: 0;
  box-shadow: 0 0 20px rgba(104, 119, 255, 0.16);
}

.mini-send-btn.el-button--primary {
  border: 0;
  background: linear-gradient(135deg, #6d70ff, #8b5cf6);
  color: #ffffff;
}

.mini-hide-btn {
  width: 42px;
  height: 42px;
  min-width: 42px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(124, 146, 189, 0.22);
  border-radius: 8px;
  background: rgba(30, 42, 68, 0.78);
  color: #c8d8ef;
}

.mini-hide-btn:hover {
  border-color: rgba(83, 174, 255, 0.42);
  background: rgba(55, 163, 255, 0.16);
  color: #ffffff;
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
  background: rgba(15, 23, 42, 0.42);
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
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--app-shell-panel-bg-strong, var(--el-bg-color)) 96%, var(--el-color-primary) 4%), var(--app-shell-panel-bg, var(--el-bg-color))),
    var(--app-shell-panel-bg, var(--el-bg-color));
  box-shadow:
    0 28px 80px rgba(15, 23, 42, 0.28),
    inset 0 1px 0 var(--app-shell-panel-highlight, rgba(255, 255, 255, 0.74));
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
  border: 1px solid rgba(96, 231, 255, 0.18);
  background:
    radial-gradient(circle at 20% 0%, rgba(34, 211, 238, 0.12), transparent 34%),
    linear-gradient(150deg, rgba(6, 18, 33, 0.98), rgba(11, 30, 49, 0.96));
  box-shadow: 0 18px 46px rgba(0, 0, 0, 0.36), 0 0 24px rgba(34, 211, 238, 0.1);
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
  color: rgba(216, 248, 255, 0.78);
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
  background: rgba(34, 211, 238, 0.1);
  color: #d8f8ff;
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__item.is-selected {
  color: #22d3ee;
  background: rgba(34, 211, 238, 0.14);
}

.mini-ws-model-select-popper.el-select__popper .el-popper__arrow::before {
  background: rgba(8, 22, 38, 0.98);
}
</style>
