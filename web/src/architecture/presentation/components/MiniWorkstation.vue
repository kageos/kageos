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
      data-testid="mini-workstation"
      :data-full-code-path="fullCodePath"
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
      <div class="mini-ws-header" data-testid="mini-workstation-header" @mousedown="startDrag" @dblclick.prevent="onHeaderDblClick">
        <span class="mini-ws-title" data-testid="mini-workstation-title">
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
                <MiniWorkstationKeyInfoSection
                  compact
                  :uploaded-files="uploadedFiles"
                  :output-files="outputFiles"
                  :display-fields="allPanelDisplayFields"
                  @preview-file="previewFile"
                  @download-file="downloadFile"
                  @preview-field="openDfPreview"
                  @copy-field="copyDisplayFieldValue"
                />
              </div>
            </template>
          </el-dropdown>
          <el-button link size="small" @click="$emit('minimize')" title="最小化">
            <el-icon :size="14"><Minus /></el-icon>
          </el-button>
          <el-button link size="small" @click="toggleMaximize" :title="maximized ? '还原' : '最大化'">
            <el-icon :size="14"><component :is="maximized ? CopyDocument : FullScreen" /></el-icon>
          </el-button>
          <el-button link size="small" data-testid="mini-workstation-close" @click="$emit('close')" title="关闭">
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
        <MiniWorkstationMessages
          :messages="messages"
          :maximized="maximized"
          :sending="sending"
          :streaming-display-length="streamingDisplayLength"
          :render-markdown="renderMarkdown"
          :format-message-time="formatMessageTime"
          :get-file-groups-from-calls="getFileGroupsFromCalls"
          :get-display-fields-from-calls="getDisplayFieldsFromCalls"
        />
      </div>

      <!-- 最大化时：右侧关键信息面板 -->
      <div v-if="maximized && panelHasContent" class="mini-file-sidebar">
        <div class="mini-file-sidebar-header">关键信息</div>
        <MiniWorkstationKeyInfoSection
          :uploaded-files="uploadedFiles"
          :output-files="outputFiles"
          :display-fields="allPanelDisplayFields"
          @preview-file="previewFile"
          @download-file="downloadFile"
          @preview-field="openDfPreview"
          @copy-field="copyDisplayFieldValue"
        />
      </div>

      </div><!-- /.mini-ws-body -->

      <MiniWorkstationComposer
        :full-code-path="fullCodePath"
        :attached-files="attachedFiles"
        :uploading="uploading"
        :input-text="inputText"
        :sending="sending"
        :stopping="stopping"
        :selected-l-l-m-config-id="selectedLLMConfigId"
        :llm-list="llmList"
        :llm-loading="llmLoading"
        :register-input-ref="registerInputRef"
        :on-l-l-m-select-visible-change="onLLMSelectVisibleChange"
        :on-file-change="onFileChange"
        :remove-file="removeFile"
        :on-input-enter="onInputEnter"
        @update:input-text="inputText = $event"
        @update:selected-l-l-m-config-id="selectedLLMConfigId = $event"
        @stop="handleStopSession"
        @send="handleSend"
      />

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
import { Loading, Close, Minus, FullScreen, CopyDocument, FolderOpened, UploadFilled, Document as DocumentIcon } from '@element-plus/icons-vue'
import { useWorkspaceChatStream } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import MiniWorkstationDisplayFieldPreviewDialog from './MiniWorkstationDisplayFieldPreviewDialog.vue'
import MiniWorkstationComposer from './MiniWorkstationComposer.vue'
import MiniWorkstationKeyInfoSection from './MiniWorkstationKeyInfoSection.vue'
import MiniWorkstationMessages from './MiniWorkstationMessages.vue'
import MiniWorkstationSessionList from './MiniWorkstationSessionList.vue'
import { useLazyMarkdownRenderer } from '@/composables/useLazyMarkdownRenderer'
import { useMiniWorkstationPanel } from '../composables/useMiniWorkstationPanel'
import { useMiniWorkstationWindow } from '../composables/useMiniWorkstationWindow'
import { useMiniWorkstationSessions } from '../composables/useMiniWorkstationSessions'
import { useMiniWorkstationUploads } from '../composables/useMiniWorkstationUploads'
import { useMiniWorkstationComposer } from '../composables/useMiniWorkstationComposer'
import { useMiniWorkstationEffects } from '../composables/useMiniWorkstationEffects'

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

function registerInputRef(element: HTMLTextAreaElement | null) {
  inputRef.value = element || undefined
}

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
  llmList,
  llmLoading,
  selectedLLMConfigId,
  onLLMSelectVisibleChange,
  onInputEnter,
  handleSend
} = useMiniWorkstationComposer({
  fullCodePath: fullCodePathRef,
  sessionId,
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
/* ── 主体区域（sidebar + output） ── */
.mini-ws-body {
  position: relative;
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
