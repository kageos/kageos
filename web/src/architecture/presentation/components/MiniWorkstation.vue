<!--
  MiniWorkstation - 迷你浮动工作台
  右下角弹出的小窗口，支持输入命令、上传文件、SSE 实时输出、最小化。
-->
<template>
  <transition name="mini-ws-pop">
    <div
      v-if="visible"
      ref="rootRef"
      :class="['mini-ws', { 'mini-ws--maximized': maximized }]"
      :style="windowStyle"
      @dragover.prevent="onDragOver"
      @dragleave.prevent="onDragLeave"
      @drop.prevent="onDrop"
    >
      <!-- 四边 + 四角 resize 手柄 -->
      <template v-if="!maximized">
        <div class="mini-resize-handle mini-resize-n" @mousedown.stop="startResize($event, 'n')"></div>
        <div class="mini-resize-handle mini-resize-s" @mousedown.stop="startResize($event, 's')"></div>
        <div class="mini-resize-handle mini-resize-e" @mousedown.stop="startResize($event, 'e')"></div>
        <div class="mini-resize-handle mini-resize-w" @mousedown.stop="startResize($event, 'w')"></div>
        <div class="mini-resize-handle mini-resize-ne" @mousedown.stop="startResize($event, 'ne')"></div>
        <div class="mini-resize-handle mini-resize-nw" @mousedown.stop="startResize($event, 'nw')"></div>
        <div class="mini-resize-handle mini-resize-se" @mousedown.stop="startResize($event, 'se')"></div>
        <div class="mini-resize-handle mini-resize-sw" @mousedown.stop="startResize($event, 'sw')"></div>
      </template>

      <!-- 标题栏：左标题 + 居中目录名 + 右按钮，可拖拽 -->
      <div class="mini-ws-header" @mousedown="startDrag" @dblclick.prevent="onHeaderDblClick">
        <span class="mini-ws-title">
          <el-icon v-if="sending" class="is-loading" :size="14"><Loading /></el-icon>
          <el-icon v-else :size="14"><FolderOpened /></el-icon>
        </span>
        <span class="mini-ws-dir-name" :title="fullCodePath + (firstUserMessageFull ? '\n\n' + firstUserMessageFull : '')">
          {{ dirName || displayPath }}{{ firstUserMessagePreview ? ' · ' + firstUserMessagePreview : '' }}
        </span>
        <div class="mini-ws-header-actions" @mousedown.stop>
          <el-dropdown
            ref="keyInfoDropdownRef"
            v-if="panelHasContent"
            trigger="click"
            placement="left-start"
            popper-class="mini-files-dropdown-popper"
            :hide-on-click="false"
            @visible-change="onKeyInfoDropdownVisibleChange"
          >
            <el-button link size="small" class="mini-header-files-btn" title="查看关键信息">
              <el-icon :size="14"><DocumentIcon /></el-icon>
              <span class="mini-header-files-count">关键信息 ({{ panelItemCount }})</span>
            </el-button>
            <template #dropdown>
              <div class="mini-files-dropdown-panel">
                <div class="mini-files-dropdown-title">关键信息</div>
                <div class="mini-files-dropdown-body">
                  <template v-if="uploadedFiles.length > 0">
                    <div class="mini-file-section-title">
                      <el-icon :size="12"><UploadFilled /></el-icon>
                      上传文件 ({{ uploadedFiles.length }})
                    </div>
                    <MiniWorkstationFileCard
                      v-for="(f, i) in uploadedFiles"
                      :key="'u' + i"
                      :file="f"
                      compact
                      @preview="previewFile"
                      @download="downloadFile"
                    />
                  </template>
                  <template v-if="outputFiles.length > 0">
                    <div class="mini-file-section-title">
                      <el-icon :size="12"><FolderOpened /></el-icon>
                      输出文件 ({{ outputFiles.length }})
                    </div>
                    <MiniWorkstationFileCard
                      v-for="(f, i) in outputFiles"
                      :key="'o' + i"
                      :file="f"
                      compact
                      @preview="previewFile"
                      @download="downloadFile"
                    />
                  </template>
                  <template v-if="allPanelDisplayFields.length > 0">
                    <div class="mini-file-section-title">
                      <el-icon :size="12"><Memo /></el-icon>
                      输出数据 ({{ allPanelDisplayFields.length }})
                    </div>
                    <MiniWorkstationDisplayFieldCard
                      v-for="(df, i) in allPanelDisplayFields"
                      :key="'df' + i"
                      :field="df"
                      compact
                      @preview="openDfPreview"
                      @copy="copyDisplayFieldValue"
                    />
                  </template>
                </div>
              </div>
            </template>
          </el-dropdown>
          <el-button link size="small" @click="$emit('minimize')" title="最小化">
            <el-icon :size="14"><Minus /></el-icon>
          </el-button>
          <el-button link size="small" @click="toggleMaximize" :title="maximized ? '还原' : '最大化'">
            <el-icon :size="14"><component :is="maximized ? CopyDocument : FullScreen" /></el-icon>
          </el-button>
          <el-button link size="small" @click="$emit('close')" title="关闭">
            <el-icon :size="14"><Close /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- 最大化时的主体区域：左侧会话列表 + 右侧消息 -->
      <div class="mini-ws-body">

      <!-- 最大化时：会话列表侧边栏 -->
      <MiniWorkstationSessionList
        v-if="maximized"
        :sessions="miniSessionList"
        :loading="loadingSessions"
        :active-session-id="sessionId"
        :format-relative-time="formatRelativeTime"
        @new="handleNewSession"
        @select="handleSelectSession"
      />

      <!-- SSE 输出区 -->
      <div class="mini-ws-output" ref="outputRef">
        <template v-if="messages.length > 0">
          <div
            v-for="(msg, i) in messages"
            :key="i"
            :class="['mini-msg', msg.role]"
          >
            <div v-if="msg.role === 'user'" class="mini-msg-user">
              <span class="mini-msg-badge">你</span>
              <span class="mini-msg-time">{{ msg.created_at ? formatMessageTime(msg.created_at) : '—' }}</span>
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
              <!-- 助手消息：按 block 渲染 -->
              <div class="mini-msg-assistant-header">
                <span class="mini-msg-badge">工作台</span>
                <span class="mini-msg-time">{{ msg.created_at ? formatMessageTime(msg.created_at) : '—' }}</span>
              </div>
              <div v-if="msg.blocks?.length" class="mini-msg-assistant">
                <template v-for="(block, bi) in msg.blocks" :key="bi">
                  <div v-if="block.type === 'content'" class="mini-content-block mini-md-content" v-html="renderMarkdown((sending && i === messages.length - 1 && bi === msg.blocks!.length - 1) ? block.text.slice(0, streamingDisplayLength) : block.text)"></div>
                  <template v-else-if="block.type === 'tool_calls'">
                    <!-- 最大化：用 MessageToolCalls 组件渲染工具详情 -->
                    <MessageToolCalls
                      v-if="maximized"
                      :tool-calls="block.calls"
                      :file-groups="getFileGroupsFromCalls(block.calls)"
                    />
                    <!-- 正常大小：仅显示工具名标签 -->
                    <template v-else>
                      <div class="mini-tools-block">
                        <div v-for="tc in block.calls" :key="tc.name" class="mini-tool-tag">
                          <el-icon v-if="tc.status === 'streaming' || tc.status === 'running'" class="is-loading" :size="12"><Loading /></el-icon>
                          <el-icon v-else-if="tc.status === 'ok'" :size="12" color="#67c23a"><CircleCheck /></el-icon>
                          <el-icon v-else-if="tc.status === 'error'" :size="12" color="#f56c6c"><CircleClose /></el-icon>
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
                <div v-if="msg.content" class="mini-msg-assistant mini-content-block mini-md-content" v-html="renderMarkdown(msg.content)"></div>
                <!-- 最大化：用 MessageToolCalls 组件渲染工具详情 -->
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
      </div>

      <!-- 最大化时：右侧关键信息面板 -->
      <div v-if="maximized && panelHasContent" class="mini-file-sidebar">
        <div class="mini-file-sidebar-header">关键信息</div>

        <div class="mini-file-sidebar-body">
          <!-- 上传文件 -->
          <template v-if="uploadedFiles.length > 0">
            <div class="mini-file-section-title">
              <el-icon :size="13"><UploadFilled /></el-icon>
              上传文件 ({{ uploadedFiles.length }})
            </div>
            <MiniWorkstationFileCard
              v-for="(f, i) in uploadedFiles"
              :key="'u' + i"
              :file="f"
              @preview="previewFile"
              @download="downloadFile"
            />
          </template>

          <!-- 输出文件 -->
          <template v-if="outputFiles.length > 0">
            <div class="mini-file-section-title">
              <el-icon :size="13"><FolderOpened /></el-icon>
              输出文件 ({{ outputFiles.length }})
            </div>
            <MiniWorkstationFileCard
              v-for="(f, i) in outputFiles"
              :key="'o' + i"
              :file="f"
              @preview="previewFile"
              @download="downloadFile"
            />
          </template>

          <!-- 输出数据 -->
          <template v-if="allPanelDisplayFields.length > 0">
            <div class="mini-file-section-title">
              <el-icon :size="13"><Memo /></el-icon>
              输出数据 ({{ allPanelDisplayFields.length }})
            </div>
            <MiniWorkstationDisplayFieldCard
              v-for="(df, i) in allPanelDisplayFields"
              :key="'sdf' + i"
              :field="df"
              @preview="openDfPreview"
              @copy="copyDisplayFieldValue"
            />
          </template>
        </div>
      </div>

      </div><!-- /.mini-ws-body -->

      <!-- 附件展示 -->
      <div v-if="attachedFiles.length > 0" class="mini-ws-files">
        <el-tag
          v-for="(f, idx) in attachedFiles"
          :key="idx"
          size="small"
          closable
          @close="removeFile(idx)"
        >
          {{ f.source_name || f.name }}
        </el-tag>
      </div>

      <!-- 模型选择 + 输入区 -->
      <div class="mini-ws-model-row">
        <div class="mini-ws-control">
          <span class="mini-ws-model-label">模型</span>
          <el-select
            v-model="selectedLLMConfigId"
            placeholder="默认模型"
            filterable
            :loading="llmLoading"
            teleported
            popper-class="mini-ws-model-select-popper"
            class="mini-ws-model-select"
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
        <div class="mini-ws-control">
          <span class="mini-ws-model-label">模式</span>
          <el-select
            :model-value="selectedModeCode"
            placeholder="dev"
            :disabled="!fullCodePath"
            :loading="modeLoading"
            teleported
            popper-class="mini-ws-model-select-popper"
            class="mini-ws-model-select"
            @update:model-value="setSelectedModeCode"
          >
            <el-option
              v-for="mode in modeOptions"
              :key="mode.code"
              :label="formatModeOptionLabel(mode)"
              :value="mode.code"
            />
          </el-select>
        </div>
      </div>
      <div class="mini-ws-input">
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
          ref="inputRef"
          v-model="inputText"
          class="mini-input"
          placeholder="输入命令...（Enter 发送，Shift+Enter 换行）"
          rows="3"
          @keydown.enter="onInputEnter"
        />
        <el-button
          v-if="sending"
          type="danger"
          size="small"
          :loading="stopping"
          @click="handleStopSession"
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
          @click="handleSend"
          class="mini-send-btn"
        >
          发送
        </el-button>
      </div>

      <!-- 拖拽上传遮罩 -->
      <transition name="el-fade-in-linear">
        <div v-if="dragOver" class="mini-ws-drop-overlay">
          <el-icon :size="28"><UploadFilled /></el-icon>
          <span>松开上传文件</span>
        </div>
      </transition>

    </div>
  </transition>

  <MiniWorkstationDisplayFieldPreviewDialog
    :visible="dfPreviewVisible"
    :label="dfPreviewLabel"
    :content="dfPreviewContent"
    @close="closeDfPreview"
    @copy="copyDfPreviewContent"
    @update:content="dfPreviewContent = $event"
  />
</template>

<script setup lang="ts">
import { ref, onUnmounted, computed, watch } from 'vue'
import { Loading, Close, Minus, FullScreen, CopyDocument, Paperclip, CircleCheck, CircleClose, FolderOpened, UploadFilled, VideoPause, Document as DocumentIcon, Memo } from '@element-plus/icons-vue'
import { useWorkspaceChatStream } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import type { WorkspaceModeItem } from '@/api/workspace'
import OutputFilesDisplay from './OutputFilesDisplay.vue'
import MessageToolCalls from './MessageToolCalls.vue'
import OutputDisplayFields from './OutputDisplayFields.vue'
import MiniWorkstationFileCard from './MiniWorkstationFileCard.vue'
import MiniWorkstationDisplayFieldCard from './MiniWorkstationDisplayFieldCard.vue'
import MiniWorkstationDisplayFieldPreviewDialog from './MiniWorkstationDisplayFieldPreviewDialog.vue'
import MiniWorkstationSessionList from './MiniWorkstationSessionList.vue'
import { useLazyMarkdownRenderer } from '@/composables/useLazyMarkdownRenderer'
import { useMiniWorkstationPanel } from '../composables/useMiniWorkstationPanel'
import { useMiniWorkstationWindow } from '../composables/useMiniWorkstationWindow'
import { useMiniWorkstationSessions } from '../composables/useMiniWorkstationSessions'
import { useMiniWorkstationUploads } from '../composables/useMiniWorkstationUploads'
import { useMiniWorkstationComposer } from '../composables/useMiniWorkstationComposer'
import { useMiniWorkstationEffects } from '../composables/useMiniWorkstationEffects'
import { useWorkspaceModeSelection } from '../composables/useWorkspaceModeSelection'

const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()

const props = defineProps<{
  visible: boolean
  fullCodePath: string
  dirName?: string
  initialSessionId?: string
  initialOffset?: number
  initialPosition?: 'center'
  initialMaximized?: boolean
}>()

const fullCodePathRef = computed(() => props.fullCodePath)
const initialSessionIdRef = computed(() => props.initialSessionId)

const emit = defineEmits<{
  (e: 'minimize'): void
  (e: 'close'): void
  (e: 'task-started', sessionId: string): void
  (e: 'tool-call-ok', payload: { name: string }): void
  (e: 'maximize-change', payload: { maximized: boolean; sessionId?: string }): void
}>()

const { messages, sending, sessionId, streamingDisplayLength, send: sendMessage, setMessages } = useWorkspaceChatStream()
const outputRef = ref<HTMLElement>()
const inputText = ref('')
const inputRef = ref<HTMLTextAreaElement>()

const displayPath = computed(() => {
  if (!props.fullCodePath) return '未选择目录'
  const parts = props.fullCodePath.split('/').filter(Boolean)
  return parts[parts.length - 1] || props.fullCodePath
})

// 首条用户消息摘要，用于同目录多 Mini 时区分（如「分析数据」「帮我把xxx改成」）
const firstUserMessagePreview = computed(() => {
  const first = messages.value?.find(m => m.role === 'user')
  const content = typeof first?.content === 'string' ? first.content.trim() : ''
  if (!content) return ''
  const maxLen = 12
  return content.length > maxLen ? content.slice(0, maxLen) + '…' : content
})
const firstUserMessageFull = computed(() => {
  const first = messages.value?.find(m => m.role === 'user')
  return typeof first?.content === 'string' ? first.content.trim() : ''
})

// ─── 最大化 / 还原 ───
const maximized = ref(!!props.initialMaximized)
const preMaxRect = ref<{ x: number; y: number; w: number; h: number } | null>(null)

const {
  miniSessionList,
  loadingSessions,
  stopping,
  loadMiniSessions,
  handleNewSession,
  handleStopSession,
  handleSelectSession,
  formatRelativeTime,
  formatMessageTime,
  startMiniStreamListening,
  startMiniPoll,
  stopMiniStreamListening,
  stopMiniPoll
} = useMiniWorkstationSessions({
  fullCodePath: fullCodePathRef,
  initialSessionId: initialSessionIdRef,
  maximized,
  sending,
  sessionId,
  setMessages,
  onSelectMaximizedSession: (targetSessionId) => {
    emit('maximize-change', { maximized: true, sessionId: targetSessionId })
  }
})

const {
  rootRef,
  windowStyle,
  startDrag,
  startResize,
  captureWindowRect,
  restoreWindowRect,
  dispose: disposeWindowState
} = useMiniWorkstationWindow({
  maximized,
  initialOffset: props.initialOffset,
  initialPosition: props.initialPosition
})

function toggleMaximize() {
  if (maximized.value) {
    maximized.value = false
    // 从最大化恢复：若当前会话仍在执行中，重新开轮询兜底（可能连接已断）
    const cur = miniSessionList.value.find(s => s.session_id === sessionId.value)
    if (sessionId.value && cur?.status === 'generating') {
      startMiniStreamListening(sessionId.value)
      startMiniPoll(sessionId.value)
    }
    restoreWindowRect(preMaxRect.value)
    emit('maximize-change', { maximized: false })
  } else {
    preMaxRect.value = captureWindowRect()
    maximized.value = true
    stopMiniPoll()
    emit('maximize-change', { maximized: true, sessionId: sessionId.value })
  }
}

function formatModeOptionLabel(mode: WorkspaceModeItem): string {
  return mode.name && mode.name !== mode.code ? `${mode.name} (${mode.code})` : mode.code
}

// ─── 文件预览辅助 ───
const {
  getFileGroupsFromCalls,
  getDisplayFieldsFromCalls,
  keyInfoDropdownRef,
  allPanelDisplayFields,
  uploadedFiles,
  outputFiles,
  panelHasContent,
  panelItemCount,
  copyDisplayFieldValue,
  onKeyInfoDropdownVisibleChange,
  dfPreviewVisible,
  dfPreviewLabel,
  dfPreviewContent,
  openDfPreview,
  closeDfPreview,
  copyDfPreviewContent,
  previewFile,
  downloadFile
} = useMiniWorkstationPanel(messages)

const {
  attachedFiles,
  uploading,
  dragOver,
  onFileChange,
  removeFile,
  onDragOver,
  onDragLeave,
  onDrop
} = useMiniWorkstationUploads({
  fullCodePath: fullCodePathRef,
  inputText,
  inputRef
})

const {
  modeOptions,
  modeLoading,
  selectedModeCode,
  setSelectedModeCode,
  applySessionMode
} = useWorkspaceModeSelection(fullCodePathRef)

const {
  llmList,
  llmLoading,
  selectedLLMConfigId,
  onLLMSelectVisibleChange,
  onInputEnter,
  handleSend
} = useMiniWorkstationComposer({
  fullCodePath: fullCodePathRef,
  sessionId,
  selectedModeCode,
  maximized,
  inputText,
  inputRef,
  attachedFiles,
  sending,
  sendMessage,
  onTaskStarted: (startedSessionId) => {
    emit('task-started', startedSessionId)
  },
  onToolCallOk: (payload) => {
    emit('tool-call-ok', payload)
  },
  onMaximizedSessionStarted: (startedSessionId) => {
    void loadMiniSessions()
    emit('maximize-change', { maximized: true, sessionId: startedSessionId })
  }
})

watch(
  () => [props.visible, props.fullCodePath] as const,
  ([visible, fullCodePath]) => {
    if (visible && fullCodePath) {
      void loadMiniSessions()
    }
  },
  { immediate: true }
)

watch(
  () => [sessionId.value, miniSessionList.value] as const,
  ([currentSessionId, sessions]) => {
    if (!currentSessionId) return
    const found = sessions.find((session) => session.session_id === currentSessionId)
    if (found) {
      applySessionMode(found)
    }
  },
  { immediate: true }
)

/** 双击标题栏切换最大化 */
function onHeaderDblClick() {
  toggleMaximize()
}

useMiniWorkstationEffects({
  visible: computed(() => props.visible),
  maximized,
  messages,
  sending,
  sessionId,
  inputRef,
  outputRef,
  stopMiniPoll,
  loadMiniSessions
})

onUnmounted(() => {
  disposeWindowState()
})
</script>

<style scoped>
.mini-ws {
  position: fixed;
  right: 24px;
  bottom: 80px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  z-index: 2500;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  transition: left 0.3s ease, top 0.3s ease, width 0.3s ease, height 0.3s ease, max-height 0.3s ease, border-radius 0.3s ease;
}
.mini-ws--maximized {
  box-shadow: none;
  border: none;
}

/* ── Resize 手柄 ── */
.mini-resize-handle { position: absolute; z-index: 5; }
.mini-resize-n  { top: -3px; left: 6px; right: 6px; height: 6px; cursor: n-resize; }
.mini-resize-s  { bottom: -3px; left: 6px; right: 6px; height: 6px; cursor: s-resize; }
.mini-resize-e  { right: -3px; top: 6px; bottom: 6px; width: 6px; cursor: e-resize; }
.mini-resize-w  { left: -3px; top: 6px; bottom: 6px; width: 6px; cursor: w-resize; }
.mini-resize-ne { top: -3px; right: -3px; width: 10px; height: 10px; cursor: ne-resize; }
.mini-resize-nw { top: -3px; left: -3px; width: 10px; height: 10px; cursor: nw-resize; }
.mini-resize-se { bottom: -3px; right: -3px; width: 10px; height: 10px; cursor: se-resize; }
.mini-resize-sw { bottom: -3px; left: -3px; width: 10px; height: 10px; cursor: sw-resize; }

/* ── 标题栏 ── */
.mini-ws-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  cursor: move;
  user-select: none;
  background: var(--el-fill-color-blank);
  flex-shrink: 0;
}
.mini-ws--maximized .mini-ws-header {
  cursor: default;
}
.mini-ws-title {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  color: var(--el-color-primary);
}
.mini-ws-dir-name {
  flex: 1;
  text-align: center;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0 4px;
}
.mini-ws-header-actions {
  flex-shrink: 0;
  display: flex;
  gap: 2px;
  align-items: center;
}
.mini-header-files-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.mini-header-files-count {
  font-size: 12px;
  color: var(--el-color-primary);
}

/* ── 标题栏文件下拉（不遮挡内容区） ── */
.mini-files-dropdown-panel {
  min-width: 260px;
  max-width: 320px;
  background: var(--el-bg-color);
  border-radius: var(--el-border-radius-base);
  box-shadow: var(--el-box-shadow-light);
}
.mini-files-dropdown-title {
  padding: 10px 12px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.mini-files-dropdown-body {
  max-height: 360px;
  overflow-y: auto;
  padding: 8px;
}
.mini-files-dropdown-body .mini-file-section-title {
  margin-top: 6px;
}
.mini-files-dropdown-body .mini-file-section-title:first-child {
  margin-top: 0;
}

/* ── 主体区域（sidebar + output） ── */
.mini-ws-body {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
}

/* ── 最大化右侧文件面板 ── */
.mini-file-sidebar {
  width: 220px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-left: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);
}
.mini-file-sidebar-header {
  padding: 10px 12px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  border-bottom: 1px solid var(--el-border-color-extra-light);
  flex-shrink: 0;
}
.mini-file-sidebar-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}
.mini-file-section-title {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  padding: 6px 4px 4px;
  margin-top: 4px;
  &:first-child { margin-top: 0; }
}

/* ── SSE 输出区 ── */
.mini-ws-output {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px 24px;
  min-height: 0;
  font-size: 12px;
  line-height: 1.6;
}
.mini-ws--maximized .mini-ws-output {
  padding: 16px 24px;
  font-size: 13px;
}
.mini-ws-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 80px;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

.mini-msg { margin-bottom: 8px; }
.mini-msg-user {
  display: flex;
  gap: 6px;
  align-items: flex-start;
}
.mini-msg-badge {
  flex-shrink: 0;
  background: var(--el-color-primary);
  color: #fff;
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 4px;
  margin-top: 1px;
}
.mini-msg-time {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
}
.mini-msg-assistant-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}
.mini-msg-assistant-header .mini-msg-badge {
  background: var(--el-color-info-light-5, #909399);
  color: var(--el-text-color-primary);
}
.mini-msg-assistant {
  padding-left: 2px;
}

.mini-content-block {
  margin: 0;
  font-size: 12px;
  line-height: 1.6;
  font-family: inherit;
  color: var(--el-text-color-primary);
  word-break: break-word;
}
/* Markdown 渲染样式 */
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
  background: var(--el-fill-color-light);
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 11px;
  font-family: 'SF Mono', 'Fira Code', monospace;
}
.mini-md-content :deep(pre) {
  background: var(--el-fill-color-darker, #1e1e1e);
  color: #d4d4d4;
  padding: 8px 10px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 6px 0;
  font-size: 11px;
  line-height: 1.5;
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
  border-left: 3px solid var(--el-border-color);
  color: var(--el-text-color-secondary);
}
.mini-md-content :deep(table) {
  border-collapse: collapse;
  margin: 6px 0;
  font-size: 11px;
  width: 100%;
}
.mini-md-content :deep(th),
.mini-md-content :deep(td) {
  border: 1px solid var(--el-border-color-lighter);
  padding: 3px 6px;
}
.mini-md-content :deep(th) {
  background: var(--el-fill-color-light);
  font-weight: 600;
}
.mini-md-content :deep(a) {
  color: var(--el-color-primary);
  text-decoration: none;
}
.mini-md-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--el-border-color-lighter);
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
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  padding: 2px 6px;
  border-radius: 4px;
}

/* ── mini 消息内文件预览 ── */
.mini-msg-user-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
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
}
.mini-msg-files :deep(.output-files-item) {
  padding: 6px;
  min-width: 120px;
  max-width: 200px;
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

/* ── mini 消息内输出数据展示 ── */
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

/* ── 附件 ── */
.mini-ws-files {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 4px 12px;
  border-top: 1px solid var(--el-border-color-extra-light);
}

/* ── 输入区 ── */
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

/* ── 最大化时输入/附件区 ── */
.mini-ws--maximized .mini-ws-input {
  padding: 12px 24px;
}
.mini-ws--maximized .mini-ws-files {
  padding: 6px 24px;
}

/* ── 拖拽上传遮罩 ── */
.mini-ws-drop-overlay {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: rgba(var(--el-color-primary-rgb, 64, 158, 255), 0.12);
  border: 2px dashed var(--el-color-primary);
  border-radius: 12px;
  color: var(--el-color-primary);
  font-size: 14px;
  font-weight: 500;
  backdrop-filter: blur(2px);
  pointer-events: none;
}

/* ── 弹出动画 ── */
.mini-ws-pop-enter-active {
  transition: all 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.mini-ws-pop-leave-active {
  transition: all 0.2s ease-in;
}
.mini-ws-pop-enter-from {
  transform: translateY(20px) scale(0.95);
  opacity: 0;
}
.mini-ws-pop-leave-to {
  transform: translateY(10px) scale(0.97);
  opacity: 0;
}
</style>

<style lang="scss">
/* 文件下拉 popper：去掉默认内边距（z-index 已放在 main.css 全局，确保高于迷你窗） */
.mini-files-dropdown-popper.el-dropdown__popper {
  padding: 0;
}
</style>
