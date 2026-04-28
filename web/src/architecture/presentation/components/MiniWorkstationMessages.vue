<template>
  <template v-if="messages.length > 0">
    <div
      v-for="(msg, i) in messages"
      :key="i"
      :class="['mini-msg', msg.role]"
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
            section-title="上传的文件"
            class="mini-msg-files"
          />
          <span>{{ msg.content }}</span>
        </div>
      </div>
      <template v-else>
        <div class="mini-msg-assistant-header">
          <span class="mini-msg-badge">工作台</span>
          <span class="mini-msg-time">{{ msg.created_at ? formatMessageTime(msg.created_at) : '—' }}</span>
        </div>
        <div v-if="msg.blocks?.length" class="mini-msg-assistant">
          <template v-for="(block, bi) in msg.blocks" :key="bi">
            <div
              v-if="block.type === 'content'"
              class="mini-content-block mini-md-content"
              v-html="renderContentBlock(block.text, i, bi, msg.blocks.length)"
            ></div>
            <template v-else-if="block.type === 'tool_calls'">
              <MessageToolCalls
                v-if="maximized"
                :tool-calls="block.calls"
                :file-groups="getFileGroupsFromCalls(block.calls)"
              />
              <template v-else>
                <div class="mini-tools-block">
                  <div
                    v-for="(tc, ti) in block.calls"
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
                    <span>{{ tc.name }}</span>
                  </div>
                </div>
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
          <MessageToolCalls
            v-if="maximized && msg.tool_calls?.length"
            :tool-calls="msg.tool_calls"
            :file-groups="getFileGroupsFromCalls(msg.tool_calls)"
          />
          <template v-else-if="msg.tool_calls?.length">
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
    <span>输入命令开始工作</span>
  </div>
</template>

<script setup lang="ts">
import { CircleCheck, CircleClose, Loading } from '@element-plus/icons-vue'
import type { OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'
import type { OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'
import type { ChatMessage, ChatMessageToolCall } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import MessageToolCalls from './MessageToolCalls.vue'
import OutputDisplayFields from './OutputDisplayFields.vue'
import OutputFilesDisplay from './OutputFilesDisplay.vue'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const currentUsername = authStore.user?.username || authStore.userName || ''

const props = defineProps<{
  messages: ChatMessage[]
  maximized: boolean
  sending: boolean
  streamingDisplayLength: number
  renderMarkdown: (text: string) => string
  formatMessageTime: (value: string) => string
  getFileGroupsFromCalls: (calls: ChatMessageToolCall[]) => OutputFileGroup[]
  getDisplayFieldsFromCalls: (calls: ChatMessageToolCall[]) => OutputDisplayField[]
}>()

function renderContentBlock(text: string, msgIndex: number, blockIndex: number, blockCount: number): string {
  const isStreamingTail =
    props.sending &&
    msgIndex === props.messages.length - 1 &&
    blockIndex === blockCount - 1

  return props.renderMarkdown(isStreamingTail ? text.slice(0, props.streamingDisplayLength) : text)
}
</script>

<style scoped>
.mini-ws-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 80px;
  color: var(--mini-cyber-dim, rgba(143, 187, 204, 0.48));
  font-size: 13px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
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
  color: var(--mini-cyber-text, #d8f8ff);
}
.mini-msg-badge {
  flex-shrink: 0;
  border: 1px solid rgba(96, 231, 255, 0.28);
  background: rgba(34, 211, 238, 0.16);
  color: var(--mini-cyber-text, #d8f8ff);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.08em;
  padding: 2px 6px;
  border-radius: 999px;
  margin-top: 1px;
  box-shadow: 0 0 14px rgba(34, 211, 238, 0.12);
}
.mini-msg-time {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--mini-cyber-dim, rgba(143, 187, 204, 0.48));
  margin-top: 2px;
}
.mini-msg-assistant-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}
.mini-msg-assistant-header .mini-msg-badge {
  border-color: rgba(246, 199, 107, 0.3);
  background: rgba(246, 199, 107, 0.12);
  color: #ffe7ad;
}
.mini-msg-assistant {
  padding: 8px 10px;
  border: 1px solid rgba(96, 231, 255, 0.14);
  border-radius: 13px;
  background:
    linear-gradient(145deg, rgba(9, 28, 48, 0.72), rgba(4, 12, 24, 0.48)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.035), transparent);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

.mini-content-block {
  margin: 0;
  font-size: 12px;
  line-height: 1.6;
  font-family: inherit;
  color: var(--mini-cyber-text, #d8f8ff);
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
  background: rgba(34, 211, 238, 0.1);
  border: 1px solid rgba(96, 231, 255, 0.13);
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 11px;
  font-family: 'SF Mono', 'Fira Code', monospace;
  color: #bff8ff;
}
.mini-md-content :deep(pre) {
  background:
    linear-gradient(180deg, rgba(2, 8, 18, 0.94), rgba(4, 13, 24, 0.9)),
    repeating-linear-gradient(90deg, rgba(96, 231, 255, 0.035) 0 1px, transparent 1px 20px);
  color: #d8f8ff;
  border: 1px solid rgba(96, 231, 255, 0.16);
  padding: 8px 10px;
  border-radius: 10px;
  overflow-x: auto;
  margin: 6px 0;
  font-size: 11px;
  line-height: 1.5;
  box-shadow: inset 0 0 20px rgba(0, 0, 0, 0.26);
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
  border-left: 3px solid rgba(34, 211, 238, 0.42);
  color: var(--mini-cyber-muted, rgba(184, 225, 235, 0.68));
  background: rgba(34, 211, 238, 0.06);
}
.mini-md-content :deep(table) {
  border-collapse: collapse;
  margin: 6px 0;
  font-size: 11px;
  width: 100%;
}
.mini-md-content :deep(th),
.mini-md-content :deep(td) {
  border: 1px solid rgba(96, 231, 255, 0.16);
  padding: 3px 6px;
}
.mini-md-content :deep(th) {
  background: rgba(34, 211, 238, 0.1);
  font-weight: 600;
}
.mini-md-content :deep(a) {
  color: var(--mini-cyber-accent, #22d3ee);
  text-decoration: none;
}
.mini-md-content :deep(hr) {
  border: none;
  border-top: 1px solid rgba(96, 231, 255, 0.16);
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
  color: var(--mini-cyber-muted, rgba(184, 225, 235, 0.68));
  border: 1px solid rgba(96, 231, 255, 0.16);
  background: rgba(34, 211, 238, 0.08);
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
  border: 1px solid rgba(34, 211, 238, 0.22);
  border-radius: 13px;
  background:
    linear-gradient(135deg, rgba(34, 211, 238, 0.14), rgba(8, 22, 38, 0.48)),
    linear-gradient(180deg, rgba(255, 255, 255, 0.04), transparent);
  color: var(--mini-cyber-text, #d8f8ff);
}
.mini-msg-files {
  margin: 4px 0;
}
.mini-msg-files :deep(.output-files-head) {
  font-size: 11px;
  margin-bottom: 4px;
}
.mini-msg-files :deep(.output-files-wrap) {
  padding: 6px;
  border-color: rgba(96, 231, 255, 0.14);
  background: rgba(2, 8, 18, 0.34);
}
.mini-msg-files :deep(.output-files-item) {
  padding: 6px;
  min-width: 120px;
  max-width: 200px;
  border-color: rgba(96, 231, 255, 0.14);
  background: rgba(8, 22, 38, 0.62);
}
.mini-msg-files :deep(.output-files-preview) {
  width: 40px;
  height: 40px;
}
.mini-msg-files :deep(.output-files-icon) {
  width: 32px;
  height: 32px;
  font-size: 18px;
}
.mini-msg-files :deep(.output-files-name) {
  font-size: 11px;
}
.mini-msg-files :deep(.output-files-meta) {
  font-size: 10px;
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
