<!--
  WorkstationChat - 智能工作台对话组件（可独立页或内嵌到工作空间右侧）
  - embedded=false：独立页用，fullCodePath 来自 route，头部含「返回工作空间」
  - embedded=true：内嵌用，fullCodePath 来自 prop，头部含「返回详情」并 emit('back')
-->
<template>
  <div :class="['workstation-chat', { 'workstation-chat--embedded': embedded }]">
    <!-- 会话列表侧边栏 -->
    <div v-if="fullCodePath" class="session-sidebar" :class="{ collapsed: !sessionSidebarExpanded }">
      <div v-if="dirName" class="sidebar-dir-name" :title="fullCodePath">
        <el-icon :size="14"><FolderOpened /></el-icon>
        <span>{{ dirName }}</span>
      </div>
      <div class="sidebar-header">
        <h4>会话列表</h4>
        <div class="header-actions">
          <el-button
            text
            :icon="Plus"
            size="small"
            @click.stop="handleNewSession"
            title="新建会话"
          />
          <el-button
            text
            :icon="sessionSidebarExpanded ? ArrowLeft : ArrowRight"
            size="small"
            @click="sessionSidebarExpanded = !sessionSidebarExpanded"
            title="展开/收起"
          />
        </div>
      </div>
      <div v-show="sessionSidebarExpanded" class="session-list-wrap">
        <el-input
          v-model="sessionSearchKeyword"
          class="session-search-input"
          placeholder="搜索会话…"
          clearable
          :prefix-icon="Search"
        />
      <div class="session-list" v-loading="loadingSessions">
        <!-- 新建会话卡片 -->
        <div
          :class="['session-card', 'new-session-card', { active: !sessionId }]"
          @click="handleNewSession"
        >
          <div class="session-card-header">
            <el-icon class="new-icon"><Plus /></el-icon>
            <span class="session-card-title">新建会话</span>
          </div>
          <div class="session-card-time">
            <span>开始新的对话</span>
          </div>
        </div>
        <!-- 会话列表（按关键词过滤） -->
        <div
          v-for="session in filteredSessionList"
          :key="session.session_id"
          :class="['session-card', { active: session.session_id === sessionId }, { generating: session.status === 'generating' }]"
          @click="handleSelectSession(session.session_id)"
        >
          <div class="session-card-header">
            <el-icon v-if="session.status === 'generating'" class="is-loading session-generating-icon" :size="12"><Loading /></el-icon>
            <span class="session-card-title">{{ session.title || '未命名会话' }}</span>
            <el-tag size="small" effect="plain" class="session-mode-tag">
              {{ session.mode_code || 'dev' }}
            </el-tag>
          </div>
          <div v-if="session.user" class="session-card-user">
            <UserDisplay :username="session.user" mode="simple" size="small" />
          </div>
          <div class="session-card-time">
            <span v-if="session.status === 'generating'" class="session-status-text">执行中</span>
            <span>{{ formatRelativeTime(session.updated_at) }}</span>
          </div>
        </div>
        <div v-if="filteredSessionList.length === 0 && !loadingSessions" class="empty-sessions">
          <el-empty :description="sessionSearchKeyword ? '无匹配会话' : '暂无会话'" :image-size="60" />
        </div>
      </div>
      </div>
    </div>

    <div class="workstation-chat-content">
    <header v-if="!embedded" class="workstation-chat-header workstation-chat-header--slim">
      <el-link type="primary" :underline="false" @click="handleBack" class="back">← 返回工作空间</el-link>
    </header>

    <main class="workstation-chat-main">
      <div class="messages" ref="messagesRef">
        <div v-if="messages.length === 0" class="messages-placeholder">
          <p v-if="fullCodePath">在下方输入需求，按需调用工具、多轮对话。</p>
          <p v-else>请先「返回工作空间」，在左侧服务目录对任意目录节点悬停点 ⋮ → 打开工作台。</p>
        </div>
        <div
          v-for="(m, i) in messages"
          :key="i"
          :class="['message', m.role]"
        >
          <div class="message-header">
            <span class="role">{{ m.role === 'user' ? '我' : '工作台' }}</span>
            <span class="message-time">{{ m.created_at ? formatMessageTime(m.created_at) : '—' }}</span>
          </div>
          <!-- 用户消息 -->
          <template v-if="m.role === 'user'">
            <div class="content">
              <OutputFilesDisplay
                v-if="m.files?.length"
                :file-groups="[{ label: '', files: m.files }]"
                section-title="上传的文件"
                class="message-files"
              />
              <div class="message-text" v-html="renderMarkdown(m.content)"></div>
            </div>
          </template>
          <!-- assistant：按块顺序渲染（文本 → 工具调用 → 文本 → …），层次清晰 -->
          <template v-else-if="m.role === 'assistant' && m.blocks?.length">
            <div class="content content--blocks">
              <template v-for="(block, bi) in m.blocks" :key="bi">
                <template v-if="block.type === 'content'">
                  <div class="message-text" v-html="renderMarkdown((sending && i === messages.length - 1 && bi === m.blocks!.length - 1) ? block.text.slice(0, streamingDisplayLength) : block.text)"></div>
                  <span v-if="sending && i === messages.length - 1 && bi === m.blocks!.length - 1" class="streaming-cursor">▌</span>
                </template>
                <MessageToolCalls
                  v-else-if="block.type === 'tool_calls' && block.calls.length"
                  :key="`msg-${i}-block-${bi}`"
                  :tool-calls="block.calls"
                  :file-groups="getFileGroupsFromCalls(block.calls)"
                />
              </template>
            </div>
          </template>
          <!-- assistant 无 blocks 时退化为：整段文本 + 整段工具调用 -->
          <template v-else-if="m.role === 'assistant'">
            <div class="content">
              <div class="message-text" v-html="renderMarkdown(getMessageDisplayContent(m))"></div>
              <span v-if="sending && i === messages.length - 1" class="streaming-cursor">▌</span>
            </div>
            <MessageToolCalls
              v-if="m.tool_calls?.length"
              :key="`msg-${i}-tool_calls`"
              :tool-calls="m.tool_calls"
              :file-groups="getMessageFileGroups(m)"
            />
          </template>
        </div>
      </div>

      <div
        class="input-area-drop-zone"
        :class="{ 'is-dragging': isDraggingOver }"
        @dragover.prevent="isDraggingOver = true"
        @dragleave.prevent="isDraggingOver = false"
        @drop.prevent="onDropFiles"
      >
        <div class="input-area">
          <div class="input-area-controls">
            <div class="input-area-control">
              <span class="input-area-model-label">模型</span>
              <LLMSelector
                :model-value="selectedLLMConfigId ?? 0"
                scope="market"
                @update:model-value="onLLMSelect"
              />
            </div>
            <div class="input-area-control">
              <span class="input-area-model-label">模式</span>
              <el-select
                :model-value="selectedModeCode"
                class="input-area-mode-select"
                :disabled="!fullCodePath"
                :loading="modeLoading"
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
          <div class="input-area-attach">
            <el-upload
              :auto-upload="false"
              :show-file-list="false"
              :disabled="!fullCodePath || uploading"
              accept="*"
              multiple
              @change="onAttachFileChange"
            >
              <el-button type="default" size="small" :loading="uploading" :disabled="!fullCodePath">
                <el-icon><Paperclip /></el-icon>
                上传文件
              </el-button>
            </el-upload>
            <div v-if="attachedFiles.length > 0" class="attached-files">
              <el-tag
                v-for="(f, idx) in attachedFiles"
                :key="idx"
                size="small"
                closable
                class="attached-tag"
                @close="removeAttachedFile(idx)"
              >
                {{ f.name }}
              </el-tag>
            </div>
          </div>
          <el-input
            v-model="inputText"
            type="textarea"
            :rows="3"
            placeholder="描述你的需求…（可上传文件后在此说明，如：需要帮我转换成 png 格式；支持拖拽文件到此处）"
            :disabled="!fullCodePath"
            @keydown.ctrl.enter="send"
          />
          <el-button
            type="primary"
            :loading="sending"
            :disabled="!fullCodePath || (!inputText.trim() && attachedFiles.length === 0)"
            @click="send"
          >
            发送
          </el-button>
        </div>
      </div>
    </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, toRef } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, ArrowRight, Plus, Paperclip, FolderOpened, Loading, Search } from '@element-plus/icons-vue'
import { workspaceChatStream, type WorkspaceChatReq, type WorkspaceChatMessageFile, type WorkspaceModeItem } from '@/api/workspace'
import MessageToolCalls from './MessageToolCalls.vue'
import OutputFilesDisplay from './OutputFilesDisplay.vue'
import UserDisplay from '@/shared/components/UserDisplay.vue'
import LLMSelector from '@/shared/components/LLMSelector.vue'
import { ElMessage } from 'element-plus'
import { useWorkspaceChatStream, type ChatMessage } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import { useLazyMarkdownRenderer } from '@/composables/useLazyMarkdownRenderer'
import { useWorkstationChatAttachments } from '@/architecture/presentation/composables/useWorkstationChatAttachments'
import { useWorkstationChatSessions } from '@/architecture/presentation/composables/useWorkstationChatSessions'
import { useWorkspaceModeSelection } from '@/architecture/presentation/composables/useWorkspaceModeSelection'

const props = withDefaults(
  defineProps<{
    fullCodePath: string
    embedded?: boolean
    dirName?: string
    initialSessionId?: string
    visible?: boolean
    /** 从 mini 最大化时带过来的输入框文案 */
    initialInputText?: string
    /** 从 mini 最大化时带过来的附件 */
    initialAttachedFiles?: WorkspaceChatMessageFile[]
  }>(),
  { embedded: false, dirName: '', initialSessionId: '', visible: true, initialInputText: '', initialAttachedFiles: () => [] }
)
const emit = defineEmits<{
  (e: 'back'): void
  (e: 'tool-call-ok', payload: { name: string }): void
  (e: 'update:sending', value: boolean): void
  (e: 'update:sessionId', value: string | undefined): void
  (e: 'clear-initial-input'): void
}>()

const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()

const router = useRouter()

const { messages, sending, sessionId, streamingDisplayLength, send: sendMessage, setMessages } = useWorkspaceChatStream()
const inputText = ref('')
const messagesRef = ref<HTMLElement | null>(null)

/** 消息内容变化时自动滚到底部，用 rAF 节流避免抖动 */
let _scrollRafId = 0
function scrollMessagesToBottom() {
  if (_scrollRafId) return
  _scrollRafId = requestAnimationFrame(() => {
    _scrollRafId = 0
    const el = messagesRef.value
    if (!el) return
    // 仅当用户在底部附近时才自动滚（避免用户往上翻阅时被强制拉回）
    const threshold = 150
    const isNearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < threshold
    if (isNearBottom) {
      el.scrollTop = el.scrollHeight
    }
  })
}
watch(messages, () => scrollMessagesToBottom(), { deep: true })
// 发送消息时强制滚到底
watch(sending, (v) => {
  if (v) {
    nextTick(() => {
      const el = messagesRef.value
      if (el) el.scrollTop = el.scrollHeight
    })
  }
})

const {
  attachedFiles,
  uploading,
  isDraggingOver,
  onAttachFileChange,
  onDropFiles,
  removeAttachedFile,
  setAttachedFiles,
  clearAttachedFiles
} = useWorkstationChatAttachments(toRef(props, 'fullCodePath'))

// 向父组件上报执行中状态（用于抽屉关闭时显示浮动按钮）
watch(sending, (v) => {
  emit('update:sending', v)
}, { immediate: true })

const selectedLLMConfigId = ref<number | null>(null)

function handleBack() {
  if (props.embedded) {
    emit('back')
  } else {
    router.push({ name: 'workspace' })
  }
}

function onLLMSelect(value: number) {
  selectedLLMConfigId.value = value === 0 ? null : value
}

function formatModeOptionLabel(mode: WorkspaceModeItem): string {
  return mode.name && mode.name !== mode.code ? `${mode.name} (${mode.code})` : mode.code
}

const {
  sessionList,
  loadingSessions,
  sessionSidebarExpanded,
  sessionSearchKeyword,
  filteredSessionList,
  loadSessions,
  handleNewSession,
  handleSelectSession,
  formatRelativeTime,
  formatMessageTime,
  getMessageDisplayContent,
  getFileGroupsFromCalls,
  getMessageFileGroups
} = useWorkstationChatSessions({
  fullCodePath: toRef(props, 'fullCodePath'),
  initialSessionId: toRef(props, 'initialSessionId'),
  initialInputText: toRef(props, 'initialInputText'),
  initialAttachedFiles: toRef(props, 'initialAttachedFiles'),
  visible: toRef(props, 'visible'),
  messages,
  sending,
  sessionId,
  messagesRef,
  setMessages,
  setInputText: (value) => {
    inputText.value = value
  },
  setAttachedFiles,
  clearInitialInput: () => emit('clear-initial-input'),
  updateSessionId: (value) => emit('update:sessionId', value)
})

const {
  modeOptions,
  modeLoading,
  selectedModeCode,
  setSelectedModeCode,
  applySessionMode
} = useWorkspaceModeSelection(toRef(props, 'fullCodePath'))

watch(
  () => [sessionId.value, sessionList.value] as const,
  ([currentSessionId, sessions]) => {
    if (!currentSessionId) return
    const found = sessions.find((session) => session.session_id === currentSessionId)
    if (found) {
      applySessionMode(found)
    }
  },
  { immediate: true }
)

async function send() {
  const text = inputText.value.trim()
  const files = attachedFiles.value.length > 0 ? attachedFiles.value : null
  if (!props.fullCodePath || (!text && !files?.length)) return
  inputText.value = ''
  // 发送后清空附件，便于下一条消息重新选文件
  clearAttachedFiles()
  const content = text || ''
  const payload: WorkspaceChatReq = {
    full_code_path: props.fullCodePath,
    message: {
      content,
      ...(files?.length
        ? {
            files: {
              files,
              widget_type: 'files',
              data_type: 'struct',
            },
          }
        : {}),
    },
    session_id: sessionId.value,
    mode_code: selectedModeCode.value,
  }
  if (selectedLLMConfigId.value != null && selectedLLMConfigId.value > 0) {
    payload.llm_config_id = selectedLLMConfigId.value
  }

  const streamFn = async (onEvent: (event: string, data: Record<string, unknown>) => void) => {
    await workspaceChatStream(payload, (event, data) => {
      onEvent(event, data as Record<string, unknown>)
      if (event === 'session' && typeof data.session_id === 'string') {
        emit('update:sessionId', data.session_id as string)
      }
      if (event === 'done') loadSessions()
      if (event === 'error') ElMessage.error(String((data as { message?: string })?.message || '发送失败'))
      // 工具执行成功时通知父组件（用于刷新服务树：create_directory / write_doc / build_workspace / write_go_file）
      if (event === 'tool_call' && (data as { status?: string })?.status === 'ok' && typeof (data as { name?: string })?.name === 'string') {
        emit('tool-call-ok', { name: (data as { name: string }).name })
      }
    })
  }
  try {
    await sendMessage(content || (files?.length ? '已上传文件' : ''), streamFn, files?.length ? files : undefined)
  } catch {
    ElMessage.error('发送失败')
  }
}
</script>

<style scoped>
.workstation-chat {
  display: flex;
  flex-direction: row;
  height: 100%;
  min-height: 100vh;
  background: var(--el-bg-color-page);
}
.workstation-chat--embedded {
  min-height: 0;
}
.workstation-chat-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
}
.session-sidebar {
  width: 280px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-lighter);
  transition: width 0.3s;
}
.session-sidebar.collapsed {
  width: 60px;
}
.sidebar-dir-name {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-bottom: 1px solid var(--el-border-color-extra-light);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.session-sidebar.collapsed .sidebar-dir-name {
  display: none;
}
.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.sidebar-header h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.session-sidebar.collapsed .sidebar-header h4 {
  display: none;
}
.header-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}
.session-sidebar.collapsed .header-actions {
  flex-direction: column;
}
.session-list-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}
.session-search-input {
  flex-shrink: 0;
  padding: 6px 8px 4px;
}
.session-search-input :deep(.el-input__wrapper) {
  border-radius: 6px;
}
.session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}
.session-card {
  padding: 12px;
  margin-bottom: 8px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  background: var(--el-bg-color);
  cursor: pointer;
  transition: all 0.2s;
}
.session-card:hover {
  border-color: var(--el-color-primary);
  background: var(--el-fill-color-lighter);
}
.session-card.active {
  border-color: var(--el-color-primary);
  border-width: 2px;
  background: var(--el-bg-color);
}
.session-card.new-session-card {
  border-style: dashed;
  background: var(--el-fill-color-lighter);
}
.session-card.new-session-card:hover {
  border-color: var(--el-color-primary);
  background: var(--el-fill-color-light);
}
.session-card.new-session-card.active {
  border-style: solid;
  border-color: var(--el-color-primary);
  background: var(--el-bg-color);
}
.session-card.new-session-card .new-icon {
  margin-right: 6px;
  color: var(--el-color-primary);
}
.session-card-header {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}
.session-card-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}
.session-card-user {
  margin-top: 4px;
  margin-bottom: 4px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.session-card-user :deep(.user-display-wrapper) {
  display: inline-flex;
}
.session-card-time {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  display: flex;
  align-items: center;
  gap: 6px;
}
.session-card.generating {
  border-left: 2px solid var(--el-color-primary);
}
.session-generating-icon {
  color: var(--el-color-primary);
  margin-right: 4px;
  flex-shrink: 0;
}
.session-status-text {
  color: var(--el-color-primary);
  font-size: 11px;
  font-weight: 500;
}
.empty-sessions {
  padding: 24px;
  text-align: center;
}
.workstation-chat-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
  flex-shrink: 0;
}
.workstation-chat-header--slim {
  padding: 6px 16px;
}
.workstation-chat-header .title { font-weight: 600; color: var(--el-text-color-primary); }
.workstation-chat-header .path { color: var(--el-text-color-regular); font-size: 13px; }
.workstation-chat-header .path.empty { color: var(--el-text-color-placeholder); }
.workstation-chat-header .back { margin-left: auto; }
.workstation-chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 8px 16px 12px;
}
.messages {
  flex: 1;
  overflow-y: auto;
  margin-bottom: 12px;
  padding: 12px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
}
.messages-placeholder { color: var(--el-text-color-placeholder); font-size: 14px; padding: 24px; }
.message {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  margin-bottom: 16px;
  padding: 10px 12px;
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
}
.message.user {
  background: var(--el-fill-color-lighter);
  border-color: var(--el-border-color-lighter);
}
.message-header {
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
}
.message .role { font-weight: 600; font-size: 13px; color: var(--el-text-color-primary); }
.message.user .role { color: var(--el-color-primary); }
.message-time {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-weight: normal;
  opacity: 0.85;
}
.message .content {
  width: 100%;
  font-size: 14px;
  line-height: 1.5;
  color: var(--el-text-color-regular);
}
.message-files {
  margin-bottom: 8px;
}
.message .message-text {
  word-break: break-word;
}
.message .message-text :deep(code) {
  background: rgba(0, 0, 0, 0.08);
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
  font-size: 0.9em;
}
.message .message-text :deep(pre) {
  background: rgba(0, 0, 0, 0.05);
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 8px 0;
  border: 1px solid var(--el-border-color);
}
.message .message-text :deep(pre code) {
  background: transparent;
  padding: 0;
  font-size: 0.9em;
  line-height: 1.5;
}
.message .message-text :deep(h1), .message .message-text :deep(h2), .message .message-text :deep(h3),
.message .message-text :deep(h4), .message .message-text :deep(h5), .message .message-text :deep(h6) {
  margin: 14px 0 6px 0;
  font-weight: 600;
  line-height: 1.4;
}
.message .message-text :deep(h1) { font-size: 1.4em; border-bottom: 2px solid var(--el-border-color); padding-bottom: 6px; }
.message .message-text :deep(h2) { font-size: 1.25em; border-bottom: 1px solid var(--el-border-color); padding-bottom: 4px; }
.message .message-text :deep(h3) { font-size: 1.1em; }
.message .message-text :deep(p) { margin: 8px 0; line-height: 1.6; }
.message .message-text :deep(ul), .message .message-text :deep(ol) { margin: 8px 0; padding-left: 24px; }
.message .message-text :deep(li) { margin: 4px 0; line-height: 1.6; }
.message .message-text :deep(blockquote) {
  margin: 8px 0;
  padding: 8px 14px;
  border-left: 4px solid var(--el-color-primary);
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
}
.message .message-text :deep(table) { width: 100%; border-collapse: collapse; margin: 10px 0; font-size: 0.9em; }
.message .message-text :deep(th), .message .message-text :deep(td) {
  border: 1px solid var(--el-border-color);
  padding: 6px 10px;
  text-align: left;
}
.message .message-text :deep(th) { background: var(--el-fill-color-light); font-weight: 600; }
.message .message-text :deep(a) { color: var(--el-color-primary); text-decoration: none; }
.message .message-text :deep(a:hover) { text-decoration: underline; }
.message .message-text :deep(hr) { border: none; border-top: 1px solid var(--el-border-color); margin: 12px 0; }
.message .message-text :deep(img) { max-width: 100%; height: auto; border-radius: 4px; margin: 6px 0; }
.streaming-cursor { animation: blink 0.8s step-end infinite; color: var(--el-color-primary); display: inline-block; margin-left: 2px; }
@keyframes blink { 50% { opacity: 0; } }

.input-area-drop-zone {
  border-radius: var(--el-border-radius-base);
  border: 2px dashed transparent;
  transition: border-color 0.2s, background-color 0.2s;
}
.input-area-drop-zone.is-dragging {
  border-color: var(--el-color-primary);
  background-color: var(--el-color-primary-light-9);
}
.input-area { display: flex; flex-direction: column; gap: 8px; }
.input-area .el-button { align-self: flex-end; }
.input-area-controls {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.input-area-control {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 240px;
}
.input-area-model-label {
  font-size: 13px;
  color: var(--el-text-color-regular);
  flex-shrink: 0;
}
.input-area-control :deep(.llm-selector) { flex: 1; min-width: 0; }
.input-area-mode-select {
  flex: 1;
  min-width: 0;
}
.input-area-attach {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.attached-files { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
.attached-tag { max-width: 160px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.session-mode-tag {
  flex-shrink: 0;
  text-transform: lowercase;
}
</style>
