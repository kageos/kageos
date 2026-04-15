<template>
  <div v-if="attachedFiles.length > 0" class="mini-ws-files">
    <el-tag
      v-for="(file, idx) in attachedFiles"
      :key="`${file.url || file.name}-${idx}`"
      size="small"
      closable
      @close="removeFile(idx)"
    >
      {{ file.source_name || file.name }}
    </el-tag>
  </div>

  <div class="mini-ws-model-row">
    <div class="mini-ws-control">
      <span class="mini-ws-model-label">模型</span>
      <el-select
        :model-value="selectedLLMConfigId"
        placeholder="默认模型"
        filterable
        :loading="llmLoading"
        teleported
        popper-class="mini-ws-model-select-popper"
        class="mini-ws-model-select"
        @update:model-value="emit('update:selectedLLMConfigId', Number($event))"
        @visible-change="onLLMSelectVisibleChange"
      >
        <el-option label="默认" :value="0" />
        <el-option
          v-for="llm in llmList"
          :key="llm.id"
          :label="`${llm.name} (${llm.provider}/${llm.model})`"
          :value="llm.id"
        />
      </el-select>
    </div>
  </div>

  <div class="mini-ws-input" data-testid="mini-workstation-composer">
    <el-upload
      :auto-upload="false"
      :show-file-list="false"
      :on-change="onFileChange"
      :disabled="uploading"
      class="mini-upload-btn"
    >
      <el-button :icon="Paperclip" link :loading="uploading" size="small" title="上传文件" />
    </el-upload>
    <textarea
      :ref="bindInputRef"
      :value="inputText"
      class="mini-input"
      data-testid="mini-workstation-input"
      placeholder="输入命令...（Enter 发送，Shift+Enter 换行）"
      rows="3"
      @input="emitInput"
      @keydown.enter="onInputEnter"
    />
    <el-button
      v-if="sending"
      type="danger"
      size="small"
      :loading="stopping"
      data-testid="mini-workstation-stop"
      @click="$emit('stop')"
      class="mini-send-btn"
    >
      <el-icon><VideoPause /></el-icon>
      停止
    </el-button>
    <el-button
      v-else
      type="primary"
      size="small"
      :disabled="!fullCodePath || (!inputText.trim() && attachedFiles.length === 0)"
      data-testid="mini-workstation-send"
      @click="$emit('send')"
      class="mini-send-btn"
    >
      发送
    </el-button>
  </div>
</template>

<script setup lang="ts">
import { Paperclip, VideoPause } from '@element-plus/icons-vue'
import type { LLMInfo } from '@/api/agent'
import type { WorkspaceChatMessageFile } from '@/api/workspace'

const props = defineProps<{
  fullCodePath: string
  attachedFiles: WorkspaceChatMessageFile[]
  uploading: boolean
  inputText: string
  sending: boolean
  stopping: boolean
  selectedLLMConfigId: number
  llmList: LLMInfo[]
  llmLoading: boolean
  registerInputRef: (el: HTMLTextAreaElement | null) => void
  onLLMSelectVisibleChange: (visible: boolean) => void
  onFileChange: (uploadFileObj: { raw?: File }) => void | Promise<void>
  removeFile: (index: number) => void
  onInputEnter: (event: KeyboardEvent) => void
}>()

const emit = defineEmits<{
  (e: 'update:inputText', value: string): void
  (e: 'update:selectedLLMConfigId', value: number): void
  (e: 'send'): void
  (e: 'stop'): void
}>()

function emitInput(event: Event) {
  emit('update:inputText', (event.target as HTMLTextAreaElement).value)
}

function bindInputRef(element: unknown) {
  props.registerInputRef(element instanceof HTMLTextAreaElement ? element : null)
}
</script>

<style scoped>
.mini-ws-files {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 4px 12px;
  border-top: 1px solid var(--el-border-color-extra-light);
}

.mini-ws-model-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 6px 10px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-light);
}
.mini-ws-control {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 180px;
}
.mini-ws-model-label {
  font-size: 12px;
  color: var(--el-text-color-regular);
  flex-shrink: 0;
}
.mini-ws-model-select {
  flex: 1;
  min-width: 0;
}
.mini-ws-input {
  display: flex;
  align-items: flex-end;
  gap: 6px;
  padding: 8px 10px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);
}
.mini-upload-btn {
  flex-shrink: 0;
  align-self: center;
}
.mini-input {
  flex: 1;
  min-width: 0;
  min-height: 56px;
  max-height: 120px;
  padding: 8px 10px;
  border: none;
  outline: none;
  font-size: 13px;
  line-height: 1.5;
  font-family: inherit;
  background: transparent;
  color: var(--el-text-color-primary);
  resize: none;
  overflow-y: auto;
}
.mini-input::placeholder {
  color: var(--el-text-color-placeholder);
}
.mini-send-btn {
  flex-shrink: 0;
  align-self: flex-end;
}

:deep(.mini-ws--maximized) .mini-ws-input {
  padding: 12px 24px;
}
:deep(.mini-ws--maximized) .mini-ws-files {
  padding: 6px 24px;
}
</style>
