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

.mini-ws-model-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 8px 12px;
  border-top: 1px solid rgba(96, 231, 255, 0.12);
  background:
    linear-gradient(90deg, rgba(9, 28, 48, 0.72), rgba(4, 12, 24, 0.56)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.04), transparent);
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
  color: var(--mini-cyber-muted, rgba(184, 225, 235, 0.68));
  flex-shrink: 0;
  font-weight: 700;
  letter-spacing: 0.1em;
}
.mini-ws-model-select {
  flex: 1;
  min-width: 0;
}
.mini-ws-model-select :deep(.el-select__wrapper) {
  min-height: 30px;
  border: 1px solid rgba(96, 231, 255, 0.18);
  border-radius: 10px;
  background: rgba(3, 10, 22, 0.66);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.03), 0 0 18px rgba(34, 211, 238, 0.06);
}
.mini-ws-model-select :deep(.el-select__placeholder),
.mini-ws-model-select :deep(.el-select__selected-item) {
  color: var(--mini-cyber-text, #d8f8ff);
}
.mini-ws-input {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  padding: 10px 12px;
  border-top: 1px solid rgba(96, 231, 255, 0.16);
  background:
    radial-gradient(circle at 10% 0%, rgba(34, 211, 238, 0.1), transparent 32%),
    linear-gradient(180deg, rgba(9, 28, 48, 0.82), rgba(4, 12, 24, 0.9));
}
.mini-upload-btn {
  flex-shrink: 0;
  align-self: center;
}
.mini-upload-btn :deep(.el-button) {
  width: 32px;
  height: 32px;
  border: 1px solid rgba(96, 231, 255, 0.2);
  border-radius: 10px;
  color: var(--mini-cyber-accent, #22d3ee);
  background: rgba(34, 211, 238, 0.08);
}
.mini-upload-btn :deep(.el-button:hover) {
  color: #ffffff;
  background: rgba(34, 211, 238, 0.16);
  box-shadow: 0 0 18px rgba(34, 211, 238, 0.16);
}
.mini-input {
  flex: 1;
  min-width: 0;
  min-height: 56px;
  max-height: 120px;
  padding: 10px 12px;
  border: 1px solid rgba(96, 231, 255, 0.16);
  border-radius: 12px;
  outline: none;
  font-size: 13px;
  line-height: 1.5;
  font-family: inherit;
  background:
    linear-gradient(180deg, rgba(3, 10, 22, 0.76), rgba(3, 10, 22, 0.46)),
    repeating-linear-gradient(90deg, rgba(96, 231, 255, 0.035) 0 1px, transparent 1px 18px);
  color: var(--mini-cyber-text, #d8f8ff);
  resize: none;
  overflow-y: auto;
  box-shadow: inset 0 0 24px rgba(0, 0, 0, 0.24);
  transition: border-color 0.18s ease, box-shadow 0.18s ease, background 0.18s ease;
}
.mini-input:focus {
  border-color: rgba(34, 211, 238, 0.56);
  box-shadow: inset 0 0 24px rgba(0, 0, 0, 0.24), 0 0 0 3px rgba(34, 211, 238, 0.1);
}
.mini-input::placeholder {
  color: var(--mini-cyber-dim, rgba(143, 187, 204, 0.48));
}
.mini-send-btn {
  flex-shrink: 0;
  align-self: flex-end;
  min-height: 32px;
  border-radius: 10px;
  font-weight: 700;
  letter-spacing: 0.04em;
  box-shadow: 0 0 20px rgba(34, 211, 238, 0.16);
}
.mini-send-btn.el-button--primary {
  border-color: rgba(34, 211, 238, 0.68);
  background: linear-gradient(135deg, #0891b2, #22d3ee);
  color: #03111d;
}

:deep(.mini-ws--maximized) .mini-ws-input {
  padding: 12px 24px;
}
:deep(.mini-ws--maximized) .mini-ws-files {
  padding: 6px 24px;
}
</style>

<style lang="scss">
.mini-ws-model-select-popper.el-select__popper {
  border: 1px solid rgba(96, 231, 255, 0.18);
  background:
    radial-gradient(circle at 20% 0%, rgba(34, 211, 238, 0.12), transparent 34%),
    linear-gradient(150deg, rgba(6, 18, 33, 0.98), rgba(11, 30, 49, 0.96));
  box-shadow: 0 18px 46px rgba(0, 0, 0, 0.36), 0 0 24px rgba(34, 211, 238, 0.1);
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown {
  background: transparent;
}

.mini-ws-model-select-popper.el-select__popper .el-select-dropdown__item {
  color: rgba(216, 248, 255, 0.78);
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
  border-color: rgba(96, 231, 255, 0.18);
}
</style>
