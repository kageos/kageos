<!--
  MiniWorkstation - 迷你浮动工作台
  右下角弹出的小窗口，支持输入命令、上传文件、SSE 实时输出、最小化。
-->
<template>
  <transition name="mini-ws-pop">
    <div
      v-if="visible"
      class="mini-ws"
      :style="posStyle"
      @dragover.prevent="onDragOver"
      @dragleave.prevent="onDragLeave"
      @drop.prevent="onDrop"
    >
      <!-- 标题栏：左标题 + 居中目录名 + 右按钮，可拖拽 -->
      <div class="mini-ws-header" @mousedown="startDrag">
        <span class="mini-ws-title">
          <el-icon v-if="sending" class="is-loading" :size="14"><Loading /></el-icon>
          <el-icon v-else :size="14"><FolderOpened /></el-icon>
        </span>
        <span class="mini-ws-dir-name" :title="fullCodePath">{{ dirName || displayPath }}</span>
        <div class="mini-ws-header-actions">
          <el-button link size="small" @click="$emit('minimize')" title="最小化">
            <el-icon :size="14"><Minus /></el-icon>
          </el-button>
          <el-button link size="small" @click="$emit('maximize', sessionId)" title="最大化">
            <el-icon :size="14"><FullScreen /></el-icon>
          </el-button>
          <el-button link size="small" @click="$emit('close')" title="关闭">
            <el-icon :size="14"><Close /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- SSE 输出区（精简版） -->
      <div class="mini-ws-output" ref="outputRef">
        <template v-if="messages.length > 0">
          <div
            v-for="(msg, i) in messages"
            :key="i"
            :class="['mini-msg', msg.role]"
          >
            <div v-if="msg.role === 'user'" class="mini-msg-user">
              <span class="mini-msg-badge">你</span>
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
              <div v-if="msg.blocks?.length" class="mini-msg-assistant">
                <template v-for="(block, bi) in msg.blocks" :key="bi">
                  <div v-if="block.type === 'content'" class="mini-content-block mini-md-content" v-html="renderMarkdown(block.text)"></div>
                  <template v-else-if="block.type === 'tool_calls'">
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
                  </template>
                </template>
              </div>
              <template v-else>
                <div v-if="msg.content" class="mini-msg-assistant mini-content-block mini-md-content" v-html="renderMarkdown(msg.content)"></div>
                <OutputFilesDisplay
                  v-if="msg.tool_calls?.length && getFileGroupsFromCalls(msg.tool_calls).length"
                  :file-groups="getFileGroupsFromCalls(msg.tool_calls)"
                  class="mini-msg-files"
                />
              </template>
            </template>
          </div>
        </template>
        <div v-else class="mini-ws-empty">
          <span>输入命令开始工作</span>
        </div>
      </div>

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

      <!-- 输入区 -->
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
        <input
          ref="inputRef"
          v-model="inputText"
          class="mini-input"
          placeholder="输入命令..."
          :disabled="sending"
          @keydown.enter.exact="handleSend"
        />
        <el-button
          type="primary"
          size="small"
          :loading="sending"
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
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onUnmounted, computed, toRaw } from 'vue'
import { Loading, Close, Minus, FullScreen, Paperclip, CircleCheck, CircleClose, FolderOpened, UploadFilled } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { workspaceChatStream, getWorkspaceMessages, type WorkspaceChatReq, type WorkspaceChatMessageFile } from '@/api/workspace'
import { useWorkspaceChatStream } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import type { ChatMessage } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import { uploadFile, notifyUploadComplete } from '@/utils/upload'
import type { UploadProgress } from '@/utils/upload/types'
import OutputFilesDisplay from './OutputFilesDisplay.vue'
import { extractFileGroupsFromResult, type OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'
import { eventBus } from '@/architecture/infrastructure/eventBus'
import { marked } from 'marked'

marked.setOptions({ breaks: true, gfm: true })

function renderMarkdown(content: string): string {
  if (!content) return ''
  try {
    return marked.parse(content) as string
  } catch {
    return content.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\n/g, '<br>')
  }
}

const props = defineProps<{
  visible: boolean
  fullCodePath: string
  dirName?: string
  initialSessionId?: string
  initialOffset?: number
}>()

const emit = defineEmits<{
  (e: 'minimize'): void
  (e: 'maximize', sessionId?: string): void
  (e: 'close'): void
  (e: 'task-started', sessionId: string): void
  (e: 'tool-call-ok', payload: { name: string }): void
}>()

const { messages, sending, sessionId, send: sendMessage, handleEvent, setMessages } = useWorkspaceChatStream()

const inputText = ref('')
const inputRef = ref<HTMLInputElement>()
const outputRef = ref<HTMLElement>()
const attachedFiles = ref<WorkspaceChatMessageFile[]>([])
const uploading = ref(false)

const UPLOAD_ROUTER = 'workspace/chat'

const displayPath = computed(() => {
  if (!props.fullCodePath) return '未选择目录'
  const parts = props.fullCodePath.split('/').filter(Boolean)
  return parts[parts.length - 1] || props.fullCodePath
})

// ─── 拖拽定位 ───
const posX = ref<number | null>(null)
const posY = ref<number | null>(null)
let dragStartX = 0
let dragStartY = 0
let dragOriginX = 0
let dragOriginY = 0
let dragging = false

const posStyle = computed(() => {
  if (posX.value !== null && posY.value !== null) {
    return { left: `${posX.value}px`, top: `${posY.value}px`, right: 'auto', bottom: 'auto' }
  }
  const off = props.initialOffset || 0
  if (off > 0) {
    return { right: `${24 + off}px`, bottom: `${80 + off}px` }
  }
  return {}
})

function startDrag(e: MouseEvent) {
  dragging = true
  dragStartX = e.clientX
  dragStartY = e.clientY
  const el = (e.currentTarget as HTMLElement).parentElement!
  const rect = el.getBoundingClientRect()
  dragOriginX = rect.left
  dragOriginY = rect.top
  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
}
function onDrag(e: MouseEvent) {
  if (!dragging) return
  posX.value = dragOriginX + (e.clientX - dragStartX)
  posY.value = dragOriginY + (e.clientY - dragStartY)
}
function stopDrag() {
  dragging = false
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
}

// ─── 自动滚底 ───
watch(() => messages.value.length, () => {
  nextTick(() => {
    if (outputRef.value) outputRef.value.scrollTop = outputRef.value.scrollHeight
  })
})
watch(() => {
  const last = messages.value[messages.value.length - 1]
  return last?.content?.length ?? 0
}, () => {
  nextTick(() => {
    if (outputRef.value) outputRef.value.scrollTop = outputRef.value.scrollHeight
  })
})

// 打开时聚焦输入框
watch(() => props.visible, (v) => {
  if (v) nextTick(() => inputRef.value?.focus())
})

// ─── 文件预览辅助 ───
function getFileGroupsFromCalls(calls: Array<{ result?: string }>): OutputFileGroup[] {
  const groups: OutputFileGroup[] = []
  for (const tc of calls) {
    groups.push(...extractFileGroupsFromResult(tc.result))
  }
  return groups
}

// ─── 文件上传 ───
function toPathOnlyUrl(url: string): string {
  if (!url) return url
  try {
    if (url.startsWith('http://') || url.startsWith('https://')) {
      const u = new URL(url)
      return u.pathname + u.search + u.hash
    }
  } catch { /* noop */ }
  return url
}

async function onFileChange(uploadFileObj: { raw?: File }) {
  const file = uploadFileObj?.raw
  if (!file || !props.fullCodePath) return
  uploading.value = true
  try {
    const uploadResult = await uploadFile(UPLOAD_ROUTER, file, (_p: UploadProgress) => {})
    if (!uploadResult.fileInfo) throw new Error('上传失败')
    const completeResult = await notifyUploadComplete({
      key: uploadResult.fileInfo.key,
      success: true,
      router: uploadResult.fileInfo.router,
      file_name: uploadResult.fileInfo.file_name,
      file_size: uploadResult.fileInfo.file_size,
      content_type: uploadResult.fileInfo.content_type,
      hash: uploadResult.fileInfo.hash,
      upload_user: useAuthStore().userName || undefined,
    })
    if (!completeResult?.download_url) throw new Error('获取下载地址失败')
    attachedFiles.value.push({
      name: completeResult.file_name,
      source_name: file.name,
      storage: completeResult.storage || uploadResult.storage,
      hash: completeResult.hash || uploadResult.fileInfo.hash || '',
      size: completeResult.file_size,
      upload_ts: Math.floor(Date.now() / 1000),
      is_uploaded: true,
      url: toPathOnlyUrl(completeResult.download_url),
      server_url: completeResult.server_download_url,
      upload_user: useAuthStore().userName || undefined,
    })
    ElMessage.success(`已添加：${file.name}`)
  } catch (e: any) {
    ElMessage.error(e?.message || '上传失败')
  } finally {
    uploading.value = false
  }
}

function removeFile(idx: number) {
  attachedFiles.value.splice(idx, 1)
}

// ─── 拖拽上传 ───
const dragOver = ref(false)
let dragLeaveTimer: ReturnType<typeof setTimeout> | null = null

function onDragOver(_e: DragEvent) {
  if (dragLeaveTimer) { clearTimeout(dragLeaveTimer); dragLeaveTimer = null }
  dragOver.value = true
}

function onDragLeave(_e: DragEvent) {
  if (dragLeaveTimer) clearTimeout(dragLeaveTimer)
  dragLeaveTimer = setTimeout(() => { dragOver.value = false }, 80)
}

async function onDrop(e: DragEvent) {
  dragOver.value = false
  const files = e.dataTransfer?.files
  if (!files?.length || !props.fullCodePath) return
  for (const file of Array.from(files)) {
    await onFileChange({ raw: file })
  }
}

// ─── 发送 ───
async function handleSend() {
  const text = inputText.value.trim()
  const files = attachedFiles.value.length > 0 ? [...attachedFiles.value] : null
  if (!props.fullCodePath || (!text && !files?.length)) return

  inputText.value = ''
  attachedFiles.value = []

  const payload: WorkspaceChatReq = {
    full_code_path: props.fullCodePath,
    message: {
      content: text || '',
      ...(files?.length ? { files: { files, widget_type: 'files', data_type: 'struct' } } : {}),
    },
    session_id: sessionId.value,
  }

  const streamFn = async (onEvent: (event: string, data: Record<string, unknown>) => void) => {
    await workspaceChatStream(payload, (event, data) => {
      onEvent(event, data as Record<string, unknown>)
      if (event === 'session' && typeof data.session_id === 'string') {
        emit('task-started', data.session_id as string)
      }
      if (event === 'tool_call' && (data as { status?: string })?.status === 'ok' && typeof (data as { name?: string })?.name === 'string') {
        emit('tool-call-ok', { name: (data as { name: string }).name })
      }
    })
  }

  try {
    await sendMessage(text || (files?.length ? '已上传文件' : ''), streamFn, files?.length ? files : undefined)
  } catch {
    ElMessage.error('发送失败')
  }
}

// ─── 从全屏最小化回来：接收已有会话 ───
let miniStreamCleanup: (() => void) | null = null
let miniPollTimer: ReturnType<typeof setInterval> | null = null

function stopMiniStreamListening() {
  if (miniStreamCleanup) { miniStreamCleanup(); miniStreamCleanup = null }
}
function stopMiniPoll() {
  if (miniPollTimer) { clearInterval(miniPollTimer); miniPollTimer = null }
}

async function loadMiniSessionMessages(sid: string) {
  try {
    const res = await getWorkspaceMessages({ session_id: sid })
    const msgs = (res?.messages || [])
      .filter((m: any) => m.role === 'user' || m.role === 'assistant')
      .map((m: any) => ({
        role: m.role as 'user' | 'assistant',
        content: m.content || '',
        files: m.files ? (() => { try { const o = JSON.parse(m.files); return o?.files || [] } catch { return [] } })() : [],
        tool_calls: m.tool_calls || [],
        blocks: (() => {
          const content = m.content || ''
          const tc = m.tool_calls || []
          if (m.role !== 'assistant') return undefined
          if (content && tc.length) return [{ type: 'content' as const, text: content }, { type: 'tool_calls' as const, calls: tc }]
          if (content) return [{ type: 'content' as const, text: content }]
          if (tc.length) return [{ type: 'tool_calls' as const, calls: tc }]
          return undefined
        })(),
      }))
    setMessages(msgs as ChatMessage[])
  } catch (e) {
    console.error('[MiniWs] loadMessages error:', e)
  }
}

function startMiniStreamListening(sid: string) {
  stopMiniStreamListening()
  const handleUpdate = (payload: { session_id: string; messages: ChatMessage[] }) => {
    if (payload.session_id === sid && sessionId.value === sid) {
      stopMiniPoll()
      setMessages(payload.messages)
    }
  }
  const handleDone = (payload: { session_id: string }) => {
    if (payload.session_id === sid) {
      stopMiniStreamListening()
      loadMiniSessionMessages(sid)
    }
  }
  const offUpdate = eventBus.on('workspace:stream-update', handleUpdate)
  const offDone = eventBus.on('workspace:stream-done', handleDone)
  miniStreamCleanup = () => { offUpdate(); offDone() }
}

function startMiniPoll(sid: string) {
  stopMiniPoll()
  miniPollTimer = setInterval(async () => {
    if (sessionId.value !== sid) { stopMiniPoll(); return }
    await loadMiniSessionMessages(sid)
  }, 3000)
}

watch(
  () => props.initialSessionId,
  async (newSid) => {
    if (!newSid || !props.fullCodePath) return
    stopMiniPoll()
    stopMiniStreamListening()
    sessionId.value = newSid
    await loadMiniSessionMessages(newSid)
    startMiniStreamListening(newSid)
    startMiniPoll(newSid)
  }
)

onMounted(() => {
  if (props.visible) nextTick(() => inputRef.value?.focus())
})

onUnmounted(() => {
  stopMiniPoll()
  stopMiniStreamListening()
})

// ─── SSE 转发：隐藏（最大化）时把消息实时转给全屏工作台 ───
watch(messages, (newMsgs) => {
  if (!props.visible && sending.value && sessionId.value) {
    eventBus.emit('workspace:stream-update', {
      session_id: sessionId.value,
      messages: JSON.parse(JSON.stringify(toRaw(newMsgs))),
    })
  }
}, { deep: true })

watch(sending, (cur, prev) => {
  if (!props.visible && prev && !cur && sessionId.value) {
    eventBus.emit('workspace:stream-done', { session_id: sessionId.value })
  }
})
</script>

<style scoped>
.mini-ws {
  position: fixed;
  right: 24px;
  bottom: 80px;
  width: 380px;
  max-height: 480px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  z-index: 2500;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

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
}

/* ── SSE 输出区 ── */
.mini-ws-output {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
  min-height: 120px;
  max-height: 300px;
  font-size: 12px;
  line-height: 1.6;
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

/* ── 附件 ── */
.mini-ws-files {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 4px 12px;
  border-top: 1px solid var(--el-border-color-extra-light);
}

/* ── 输入区 ── */
.mini-ws-input {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);
}
.mini-upload-btn {
  flex-shrink: 0;
}
.mini-input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 13px;
  background: transparent;
  color: var(--el-text-color-primary);
  min-width: 0;
}
.mini-input::placeholder {
  color: var(--el-text-color-placeholder);
}
.mini-send-btn {
  flex-shrink: 0;
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
