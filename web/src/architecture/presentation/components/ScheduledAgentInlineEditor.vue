<template>
  <el-form class="detail-inline-form" label-position="top">
    <el-form-item :label="t('scheduledTask.employeeName')" required>
      <el-input
        :model-value="title"
        maxlength="100"
        show-word-limit
        :placeholder="t('scheduledTask.agentTaskNamePlaceholder')"
        @update:model-value="emit('update:title', String($event || ''))"
      />
    </el-form-item>

    <el-form-item :label="t('scheduledTask.agentDescription')">
      <el-input
        :model-value="description"
        type="textarea"
        :autosize="{ minRows: 3, maxRows: 8 }"
        maxlength="500"
        show-word-limit
        :placeholder="t('scheduledTask.agentDescriptionPlaceholder')"
        @update:model-value="emit('update:description', String($event || ''))"
      />
    </el-form-item>

    <el-form-item :label="t('scheduledTask.agentModel')">
      <el-select
        :model-value="llmConfigId"
        filterable
        :placeholder="t('scheduledTask.defaultModel')"
        :loading="llmLoading"
        style="width: 100%"
        @update:model-value="updateLLMConfigID"
        @visible-change="emit('llm-visible-change', $event)"
      >
        <el-option :label="t('scheduledTask.defaultModel')" :value="0" />
        <el-option
          v-for="llm in llmList"
          :key="llm.id"
          :label="llmOptionLabel(llm)"
          :value="llm.id"
        />
      </el-select>
      <div class="detail-inline-hint">{{ t('scheduledTask.agentModelHint') }}</div>
    </el-form-item>

    <el-form-item :label="t('scheduledTask.agentMessage')" required>
      <div
        class="detail-inline-composer"
        :class="{ 'is-dragging': dragOver }"
        @paste="onPaste"
        @dragover.prevent="onDragOver"
        @dragleave.prevent="onDragLeave"
        @drop.prevent="onDrop"
      >
        <MiniWorkstationComposer
          variant="schedule"
          :full-code-path="fullCodePath"
          :attached-files="inlineComposerFiles"
          :uploading="uploading"
          :input-text="message"
          :sending="false"
          :session-running="false"
          :stopping="false"
          :selected-l-l-m-config-id="0"
          :llm-list="[]"
          :llm-loading="false"
          :queued-count="0"
          :register-input-ref="registerInlineMessageInputRef"
          :on-l-l-m-select-visible-change="noop"
          :on-file-change="onFileChange"
          :remove-file="removeInlineComposerFile"
          :on-input-enter="noopInputEnter"
          :placeholder="t('scheduledTask.agentMessagePlaceholder')"
          :expanded-title="title || t('scheduledTask.editAgentDialogTitle')"
          :expanded-subtitle="fullCodePath"
          :expanded-save-label="t('common.save')"
          mention-panel-placement="above"
          @update:input-text="emit('update:message', $event)"
          @expanded-save="handleExpandedSave"
        />
        <div class="detail-inline-editor-help">{{ t('scheduledTask.agentMessageHelp') }}</div>
        <div v-if="dragOver" class="detail-inline-drop-hint">
          {{ t('scheduledTask.dropUpload') }}
        </div>
      </div>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LLMInfo } from '@/architecture/presentation/context/api/agent'
import type { WorkspaceChatMessageFile } from '@/architecture/presentation/context/api/workspace'
import { useMiniWorkstationUploads } from '@/architecture/presentation/composables/useMiniWorkstationUploads'
import { fileNameFromRef, parseFileRefs, stringifyFileRefs } from '@/architecture/presentation/widgets/filesWidgetTypes'
import MiniWorkstationComposer from './MiniWorkstationComposer.vue'

const props = defineProps<{
  title: string
  description: string
  message: string
  files: string
  llmConfigId: number
  llmList: LLMInfo[]
  llmLoading: boolean
  fullCodePath: string
  llmOptionLabel: (llm: LLMInfo) => string
}>()

const emit = defineEmits<{
  (e: 'update:title', value: string): void
  (e: 'update:description', value: string): void
  (e: 'update:message', value: string): void
  (e: 'update:files', value: string): void
  (e: 'update:llmConfigId', value: number): void
  (e: 'llm-visible-change', value: boolean): void
  (e: 'save'): void
}>()

const { t } = useI18n()
const inlineMessageInputRef = ref<{ focus: () => void }>()
const existingFileRefs = ref<string[]>([])
const fullCodePathRef = computed(() => props.fullCodePath)
const inputText = computed({
  get: () => props.message,
  set: (value: string) => emit('update:message', value),
})

const {
  attachedFiles,
  uploading,
  dragOver,
  onFileChange,
  removeFile,
  onDragOver,
  onDragLeave,
  onDrop,
  onPaste,
} = useMiniWorkstationUploads({
  fullCodePath: fullCodePathRef,
  inputText,
  inputRef: inlineMessageInputRef,
})

const currentFileRefs = computed(() => {
  return stringifyFileRefs([
    ...existingFileRefs.value,
    ...attachedFiles.value.map((file) => file.ref).filter((ref): ref is string => !!ref),
  ])
})

const inlineComposerFiles = computed<WorkspaceChatMessageFile[]>(() => {
  return [
    ...existingFileRefs.value.map(fileRefToMessageFile),
    ...attachedFiles.value,
  ]
})

watch(
  () => props.files,
  (files) => {
    const normalized = stringifyFileRefs(parseFileRefs(files))
    if (normalized === currentFileRefs.value) {
      return
    }
    existingFileRefs.value = parseFileRefs(files)
    attachedFiles.value = []
  },
  { immediate: true }
)

watch(currentFileRefs, (files) => {
  if (files !== props.files) {
    emit('update:files', files)
  }
})

function fileRefToMessageFile(ref: string): WorkspaceChatMessageFile {
  const name = fileNameFromRef(ref)
  return {
    ref,
    name,
    source_name: name,
    is_uploaded: true,
  }
}

function removeInlineComposerFile(index: number) {
  if (index < existingFileRefs.value.length) {
    existingFileRefs.value.splice(index, 1)
    return
  }
  removeFile(index - existingFileRefs.value.length)
}

function registerInlineMessageInputRef(element: { focus: () => void } | null) {
  inlineMessageInputRef.value = element || undefined
}

function updateLLMConfigID(value: string | number | boolean | null | undefined) {
  const id = Number(value || 0)
  emit('update:llmConfigId', Number.isFinite(id) && id > 0 ? id : 0)
}

function handleExpandedSave(value: string) {
  emit('update:message', value)
  emit('save')
}

function noop() {}

function noopInputEnter() {}
</script>

<style scoped lang="scss">
.detail-inline-form {
  max-width: 980px;
  padding-top: 18px;
}

.detail-inline-form :deep(.el-form-item__label) {
  color: var(--scheduled-session-ink);
  font-weight: 650;
}

.detail-inline-hint {
  margin-top: 6px;
  color: var(--scheduled-session-muted);
  font-size: 12px;
  line-height: 1.45;
}

.detail-inline-editor-help {
  padding: 8px 2px 0;
  color: var(--scheduled-session-muted);
  font-size: 12px;
  line-height: 1.45;
}

.detail-inline-composer {
  position: relative;
  width: 100%;
}

.detail-inline-composer.is-dragging {
  outline: 2px dashed rgba(var(--el-color-primary-rgb), 0.55);
  outline-offset: 4px;
  border-radius: 14px;
}

.detail-inline-composer :deep(.mini-ws-input) {
  min-height: 240px;
}

.detail-inline-composer :deep(.mini-input-wrap) {
  min-height: 220px;
}

.detail-inline-composer :deep(.mini-structured-input .spc-editor),
.detail-inline-composer :deep(.mini-structured-input .spc-preview) {
  min-height: 190px !important;
  max-height: 520px !important;
}

.detail-inline-drop-hint {
  position: absolute;
  inset: 0;
  z-index: 3;
  display: grid;
  place-items: center;
  border-radius: 14px;
  background: rgba(15, 23, 42, 0.54);
  color: #fff;
  font-weight: 700;
  pointer-events: none;
}
</style>
