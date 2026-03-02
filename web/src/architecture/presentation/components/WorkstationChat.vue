<!--
  WorkstationChat - 智能工作台对话组件（可独立页或内嵌到工作空间右侧）
  - embedded=false：独立页用，fullCodePath 来自 route，头部含「返回工作空间」
  - embedded=true：内嵌用，fullCodePath 来自 prop，头部含「返回详情」并 emit('back')
-->
<template>
  <div :class="['workstation-chat', { 'workstation-chat--embedded': embedded }]">
    <!-- 会话列表侧边栏 -->
    <div v-if="fullCodePath" class="session-sidebar" :class="{ collapsed: !sessionSidebarExpanded }">
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
      <div v-show="sessionSidebarExpanded" class="session-list" v-loading="loadingSessions">
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
        <!-- 会话列表 -->
        <div
          v-for="session in sessionList"
          :key="session.session_id"
          :class="['session-card', { active: session.session_id === sessionId }]"
          @click="handleSelectSession(session.session_id)"
        >
          <div class="session-card-header">
            <span class="session-card-title">{{ session.title || '未命名会话' }}</span>
          </div>
          <div class="session-card-time">
            <span>{{ formatRelativeTime(session.updated_at) }}</span>
          </div>
        </div>
        <div v-if="sessionList.length === 0 && !loadingSessions" class="empty-sessions">
          <el-empty description="暂无会话" :image-size="60" />
        </div>
      </div>
    </div>

    <div class="workstation-chat-content">
    <header class="workstation-chat-header">
      <template v-if="embedded">
        <span v-if="fullCodePath" class="path">当前目录：{{ fullCodePath }}</span>
        <span v-else class="path empty">暂无目录</span>
        <el-select
          v-model="selectedLLMConfigId"
          placeholder="选择 LLM（不选则用默认）"
          clearable
          class="llm-select"
          :loading="llmLoading"
          :disabled="!fullCodePath"
        >
          <el-option v-for="l in llmList" :key="l.id" :value="l.id" :label="l.name" />
        </el-select>
        <el-link type="primary" :underline="false" @click="handleBack" class="back">返回详情</el-link>
      </template>
      <template v-else>
        <span class="title">工作台对话</span>
        <span v-if="fullCodePath" class="path">当前目录：{{ fullCodePath }}</span>
        <span v-else class="path empty">请先「返回工作空间」，在左侧服务目录对任意目录节点悬停点 ⋮ → 打开工作台</span>
        <el-select
          v-model="selectedLLMConfigId"
          placeholder="选择 LLM（不选则用默认）"
          clearable
          class="llm-select"
          :loading="llmLoading"
          :disabled="!fullCodePath"
        >
          <el-option v-for="l in llmList" :key="l.id" :value="l.id" :label="l.name" />
        </el-select>
        <el-link type="primary" :underline="false" @click="handleBack" class="back">返回工作空间</el-link>
      </template>
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
            <span v-if="m.created_at" class="message-time">{{ formatMessageTime(m.created_at) }}</span>
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
                  <div class="message-text" v-html="renderMarkdown(block.text)"></div>
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
import { ref, onMounted, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, ArrowRight, Plus, Paperclip } from '@element-plus/icons-vue'
import { marked } from 'marked'
import { workspaceChatStream, getWorkspaceSessions, getWorkspaceMessages, type WorkspaceSessionItem, type WorkspaceChatReq, type WorkspaceChatMessageFile } from '@/api/workspace'
import { getLLMList, type LLMInfo } from '@/api/agent'
import MessageToolCalls from './MessageToolCalls.vue'
import OutputFilesDisplay from './OutputFilesDisplay.vue'
import { extractFileGroupsFromResult, type OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'
import { ElMessage } from 'element-plus'
import { useWorkspaceChatStream, type ChatMessage } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import { uploadFile, notifyUploadComplete } from '@/utils/upload'
import type { UploadProgress } from '@/utils/upload/types'
import { useAuthStore } from '@/stores/auth'

// 配置 marked：支持换行、GFM
marked.setOptions({
  breaks: true,
  gfm: true,
  headerIds: false,
  mangle: false,
})

const props = withDefaults(
  defineProps<{
    fullCodePath: string
    embedded?: boolean
  }>(),
  { embedded: false }
)
const emit = defineEmits<{ (e: 'back'): void; (e: 'tool-call-ok', payload: { name: string }): void; (e: 'update:sending', value: boolean): void }>()

const router = useRouter()

const { messages, sending, sessionId, send: sendMessage, handleEvent, setMessages } = useWorkspaceChatStream()
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

/** 工作台上传文件 router，与存储路径一致 */
const WORKSPACE_CHAT_UPLOAD_ROUTER = 'workspace/chat'

/** 本条消息附带的文件（上传后加入，发送时随 message.files 提交，发送成功后清空） */
const attachedFiles = ref<WorkspaceChatMessageFile[]>([])
const uploading = ref(false)
/** 拖拽悬停时高亮输入区 */
const isDraggingOver = ref(false)

/** 将可能带 host 的 URL 转为仅 path（工作台发送给后端时 url 不要 host） */
function toPathOnlyUrl(url: string): string {
  if (!url) return url
  try {
    if (url.startsWith('http://') || url.startsWith('https://')) {
      const u = new URL(url)
      return u.pathname + u.search + u.hash
    }
  } catch {
    // 解析失败则原样返回
  }
  return url
}

/** 上传单个文件并加入附件列表（按钮选择与拖拽共用） */
async function addFileAsAttachment(file: File): Promise<void> {
  if (!file || !props.fullCodePath) return
  const uploadResult = await uploadFile(
    WORKSPACE_CHAT_UPLOAD_ROUTER,
    file,
    (_progress: UploadProgress) => {}
  )
  if (!uploadResult.fileInfo) {
    throw new Error('上传失败')
  }
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
  if (!completeResult?.download_url) {
    throw new Error('获取下载地址失败')
  }
  // 工作台发送时 url 只要 path，不要 host（后端可能返回带 cdnDomain 的完整 URL）
  const urlPathOnly = toPathOnlyUrl(completeResult.download_url)
  const item: WorkspaceChatMessageFile = {
    name: completeResult.file_name,
    source_name: file.name,
    storage: completeResult.storage || uploadResult.storage,
    hash: completeResult.hash || uploadResult.fileInfo.hash || '',
    size: completeResult.file_size,
    upload_ts: Math.floor(Date.now() / 1000),
    is_uploaded: true,
    url: urlPathOnly,
    server_url: completeResult.server_download_url,
    upload_user: useAuthStore().userName || undefined,
  }
  attachedFiles.value = [...attachedFiles.value, item]
  ElMessage.success(`已添加：${file.name}`)
}

async function onAttachFileChange(uploadFileObj: { raw?: File; name?: string }) {
  const file = uploadFileObj?.raw
  if (!file || !props.fullCodePath) return
  uploading.value = true
  try {
    await addFileAsAttachment(file)
  } catch (e: unknown) {
    console.error('[WorkstationChat] 上传失败:', e)
    ElMessage.error(e instanceof Error ? e.message : '上传失败')
  } finally {
    uploading.value = false
  }
}

/** 拖拽放下：将文件加入附件并上传（多个文件逐个上传，单个失败不影响其余） */
async function onDropFiles(e: DragEvent) {
  isDraggingOver.value = false
  if (!props.fullCodePath || uploading.value) return
  const files = e.dataTransfer?.files
  if (!files?.length) return
  const fileList = Array.from(files)
  uploading.value = true
  try {
    for (const file of fileList) {
      if (!file.name) continue
      try {
        await addFileAsAttachment(file)
      } catch (err: unknown) {
        console.error('[WorkstationChat] 拖拽上传失败:', file.name, err)
        ElMessage.error(`${file.name} 上传失败：${err instanceof Error ? err.message : '未知错误'}`)
      }
    }
  } finally {
    uploading.value = false
  }
}

function removeAttachedFile(index: number) {
  attachedFiles.value = attachedFiles.value.filter((_, i) => i !== index)
}

// 向父组件上报执行中状态（用于抽屉关闭时显示浮动按钮）
watch(sending, (v) => {
  emit('update:sending', v)
}, { immediate: true })

const llmList = ref<LLMInfo[]>([])
const llmLoading = ref(false)
const selectedLLMConfigId = ref<number | null>(null)

// 会话列表相关
const sessionList = ref<WorkspaceSessionItem[]>([])
const loadingSessions = ref(false)
const sessionSidebarExpanded = ref(true) // 默认展开

function handleBack() {
  if (props.embedded) {
    emit('back')
  } else {
    router.push({ name: 'workspace' })
  }
}

async function loadLLMs() {
  llmLoading.value = true
  try {
    const res = await getLLMList({ scope: 'market', page: 1, page_size: 200 }) as { configs?: LLMInfo[]; total?: number }
    llmList.value = res?.configs ?? []
  } catch (e: unknown) {
    console.error('加载 LLM 列表失败:', e)
    llmList.value = []
  } finally {
    llmLoading.value = false
  }
}

// 加载会话列表
async function loadSessions() {
  if (!props.fullCodePath) {
    sessionList.value = []
    return
  }
  loadingSessions.value = true
  try {
    const res = await getWorkspaceSessions({
      full_code_path: props.fullCodePath,
      page: 1,
      page_size: 50,
    })
    sessionList.value = res.sessions || []
  } catch (e: any) {
    console.error('加载会话列表失败:', e)
    ElMessage.error('加载会话列表失败')
    sessionList.value = []
  } finally {
    loadingSessions.value = false
  }
}

/** 从接口返回的 files JSON 解析出文件列表（用于展示用户消息附件） */
function parseMessageFiles(filesStr: string | null | undefined): WorkspaceChatMessageFile[] {
  if (!filesStr) return []
  try {
    const o = JSON.parse(filesStr) as { files?: WorkspaceChatMessageFile[] }
    return Array.isArray(o?.files) ? o.files : []
  } catch {
    return []
  }
}

/** 历史消息中可能含 <files>...</files> 与说明文，展示时去掉，只留用户文字 */
function stripFilesBlockForDisplay(content: string): string {
  if (!content) return ''
  const stripped = content
    .replace(/<files>[\s\S]*?<\/files>/i, '')
    .replace(/\s*以上\s*<files>\s*标签中的 JSON[^。]*。\s*/g, '')
    .trim()
  return stripped || content
}

/** 用于展示的消息正文：用户消息去掉 <files> 块，避免出现整段 JSON */
function getMessageDisplayContent(m: { role: string; content: string }): string {
  return m.role === 'user' ? stripFilesBlockForDisplay(m.content) : m.content
}

// 加载会话消息
async function loadSessionMessages(targetSessionId: string) {
  try {
    const res = await getWorkspaceMessages({ session_id: targetSessionId })
    const msgs = res.messages
      .filter((msg) => msg.role === 'user' || msg.role === 'assistant')
      .map((msg) => {
        const role = msg.role as 'user' | 'assistant'
        const content = msg.content || ''
        const tool_calls = msg.tool_calls || []
        let blocks: ChatMessage['blocks'] | undefined
        if (role === 'assistant' && (content || tool_calls.length)) {
          if (content && tool_calls.length) {
            blocks = [{ type: 'content', text: content }, { type: 'tool_calls', calls: tool_calls }]
          } else if (content) {
            blocks = [{ type: 'content', text: content }]
          } else {
            blocks = [{ type: 'tool_calls', calls: tool_calls }]
          }
        }
        return {
          role,
          content,
          files: parseMessageFiles(msg.files),
          tool_calls,
          blocks,
          created_at: msg.created_at,
        }
      })
    setMessages(msgs as ChatMessage[])
    setTimeout(() => {
      if (messagesRef.value) {
        messagesRef.value.scrollTop = messagesRef.value.scrollHeight
      }
    }, 100)
  } catch (e: unknown) {
    console.error('加载会话消息失败:', e)
    ElMessage.error('加载会话消息失败')
    setMessages([])
  }
}

// 新建会话
function handleNewSession() {
  sessionId.value = undefined
  setMessages([])
  ElMessage.success('已创建新会话，发送第一条消息后将自动保存')
}

// 选择会话
async function handleSelectSession(targetSessionId: string) {
  if (targetSessionId === sessionId.value) return
  sessionId.value = targetSessionId
  await loadSessionMessages(targetSessionId)
}

// 格式化相对时间
function formatRelativeTime(timeStr: string): string {
  const time = new Date(timeStr)
  const now = new Date()
  const diff = now.getTime() - time.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`
  return time.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

onMounted(() => {
  loadLLMs()
  if (props.fullCodePath) {
    loadSessions()
  }
})

// 监听 fullCodePath 变化，自动加载会话列表
watch(
  () => props.fullCodePath,
  (newPath) => {
    if (newPath) {
      loadSessions()
      sessionId.value = undefined
      setMessages([])
    } else {
      sessionList.value = []
      sessionId.value = undefined
      setMessages([])
    }
  }
)

function renderMarkdown(content: string): string {
  if (!content) return ''
  try {
    return marked.parse(content) as string
  } catch {
    return content.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\n/g, '<br>')
  }
}

/** 从一组工具调用的 result 中提取输出文件（用于按块展示时的每个 tool_calls 块） */
function getFileGroupsFromCalls(calls: Array<{ result?: string }>): OutputFileGroup[] {
  const groups: OutputFileGroup[] = []
  for (const tc of calls) {
    groups.push(...extractFileGroupsFromResult(tc.result))
  }
  return groups
}

/** 聚合本消息所有工具调用的 result 中的输出文件，供下方独立展示（不展开工具详情也能看到） */
function getMessageFileGroups(m: ChatMessage): OutputFileGroup[] {
  const list = m.tool_calls ?? []
  return getFileGroupsFromCalls(list)
}

function formatMessageTime(isoString: string): string {
  if (!isoString) return ''
  const date = new Date(isoString)
  const y = date.getFullYear()
  const M = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  const h = String(date.getHours()).padStart(2, '0')
  const m = String(date.getMinutes()).padStart(2, '0')
  const s = String(date.getSeconds()).padStart(2, '0')
  return `${y}-${M}-${d} ${h}:${m}:${s}`
}

async function send() {
  const text = inputText.value.trim()
  const files = attachedFiles.value.length > 0 ? attachedFiles.value : null
  if (!props.fullCodePath || (!text && !files?.length)) return
  inputText.value = ''
  // 发送后清空附件，便于下一条消息重新选文件
  attachedFiles.value = []
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
  }
  if (selectedLLMConfigId.value != null && selectedLLMConfigId.value > 0) {
    payload.llm_config_id = selectedLLMConfigId.value
  }

  const streamFn = async (onEvent: (event: string, data: Record<string, unknown>) => void) => {
    await workspaceChatStream(payload, (event, data) => {
      onEvent(event, data as Record<string, unknown>)
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
.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
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
.session-card-time {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
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
.workstation-chat-header .title { font-weight: 600; color: var(--el-text-color-primary); }
.workstation-chat-header .path { color: var(--el-text-color-regular); font-size: 13px; }
.workstation-chat-header .path.empty { color: var(--el-text-color-placeholder); }
.workstation-chat-header .mode-select { min-width: 120px; margin-right: 8px; }
.workstation-chat-header .agent-select { min-width: 180px; }
.workstation-chat-header .back { margin-left: auto; }
.workstation-chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 16px;
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
.input-area-attach {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.attached-files { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
.attached-tag { max-width: 160px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
</style>
