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
          <div v-if="session.agent_name" class="session-card-agent">
            <span class="agent-name">{{ session.agent_name }}</span>
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
          v-model="selectedModeCode"
          placeholder="模式"
          clearable
          class="mode-select"
          :loading="modeLoading"
          :disabled="!fullCodePath"
        >
          <el-option v-for="m in modeList" :key="m.code" :value="m.code" :label="m.name" />
        </el-select>
        <el-select
          v-model="selectedAgentId"
          placeholder="选择智能体（不选则用默认 LLM）"
          clearable
          class="agent-select"
          :loading="agentLoading"
          :disabled="!fullCodePath"
        >
          <el-option v-for="a in agentList" :key="a.id" :value="a.id" :label="a.name" />
        </el-select>
        <el-link type="primary" :underline="false" @click="handleBack" class="back">返回详情</el-link>
      </template>
      <template v-else>
        <span class="title">工作台对话</span>
        <span v-if="fullCodePath" class="path">当前目录：{{ fullCodePath }}</span>
        <span v-else class="path empty">请先「返回工作空间」，在左侧服务目录对任意目录节点悬停点 ⋮ → 打开工作台</span>
        <el-select
          v-model="selectedModeCode"
          placeholder="模式"
          clearable
          class="mode-select"
          :loading="modeLoading"
          :disabled="!fullCodePath"
        >
          <el-option v-for="m in modeList" :key="m.code" :value="m.code" :label="m.name" />
        </el-select>
        <el-select
          v-model="selectedAgentId"
          placeholder="选择智能体（不选则用默认 LLM）"
          clearable
          class="agent-select"
          :loading="agentLoading"
          :disabled="!fullCodePath"
        >
          <el-option v-for="a in agentList" :key="a.id" :value="a.id" :label="a.name" />
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
          <div class="content">
            <div class="message-text" v-html="renderMarkdown(m.content)"></div>
            <span v-if="sending && i === messages.length - 1 && m.role === 'assistant'" class="streaming-cursor">▌</span>
          </div>
          <!-- 本消息发起的工具调用：放在消息下方，可折叠，时间线更清晰 -->
          <div v-if="m.role === 'assistant' && m.tool_calls?.length" class="message-tool-calls-wrap">
            <details class="message-tool-calls-details" open>
              <summary class="message-tool-calls-summary">本消息发起的调用（{{ m.tool_calls.length }} 个）</summary>
              <div class="tool-calls">
                <ToolCallCard
                  v-for="(tc, j) in m.tool_calls"
                  :key="tc.id || j"
                  :tool-call="tc"
                />
              </div>
            </details>
          </div>
        </div>
      </div>

      <div class="input-area">
        <el-input
          v-model="inputText"
          type="textarea"
          :rows="3"
          placeholder="描述你的需求…"
          :disabled="!fullCodePath"
          @keydown.ctrl.enter="send"
        />
        <el-button
          type="primary"
          :loading="sending"
          :disabled="!fullCodePath || !inputText.trim()"
          @click="send"
        >
          发送
        </el-button>
      </div>
    </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, ArrowRight, Plus } from '@element-plus/icons-vue'
import { marked } from 'marked'
import { workspaceChatStream, getWorkspaceSessions, getWorkspaceMessages, getWorkspaceModes, type WorkspaceSessionItem, type WorkspaceModeItem } from '@/api/workspace'
import { getAgentList } from '@/api/agent'
import ToolCallCard from './ToolCallCard.vue'
import type { AgentInfo } from '@/api/agent'
import { ElMessage } from 'element-plus'
import { useWorkspaceChatStream } from '@/architecture/presentation/composables/useWorkspaceChatStream'

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

const { messages, sending, sessionId, agentId, send: sendMessage, handleEvent, setMessages } = useWorkspaceChatStream()
const inputText = ref('')
const messagesRef = ref<HTMLElement | null>(null)

// 后端下发 agent_id 时同步到选择器
watch(agentId, (v) => {
  if (v != null) selectedAgentId.value = v
})

// 向父组件上报执行中状态（用于抽屉关闭时显示浮动按钮）
watch(sending, (v) => {
  emit('update:sending', v)
}, { immediate: true })

const agentList = ref<AgentInfo[]>([])
const agentLoading = ref(false)
const selectedAgentId = ref<number | null>(null)
const modeList = ref<WorkspaceModeItem[]>([])
const modeLoading = ref(false)
const selectedModeCode = ref<string>('dev')

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

async function loadModes() {
  modeLoading.value = true
  try {
    const res = await getWorkspaceModes({ page: 1, page_size: 50 })
    modeList.value = res.list || []
    if (modeList.value.length > 0) {
      const hasDev = modeList.value.some((m) => m.code === 'dev')
      if (hasDev) selectedModeCode.value = 'dev'
      else if (!selectedModeCode.value) selectedModeCode.value = modeList.value[0].code
    }
  } catch {
    modeList.value = []
  } finally {
    modeLoading.value = false
  }
}

async function loadAgents() {
  agentLoading.value = true
  try {
    const res = (await getAgentList({ enabled: true, page: 1, page_size: 200 })) as any
    agentList.value = res?.data?.agents ?? res?.agents ?? []
  } catch (e: any) {
    console.error('加载智能体列表失败:', e)
    agentList.value = []
  } finally {
    agentLoading.value = false
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

// 加载会话消息
async function loadSessionMessages(targetSessionId: string) {
  try {
    const res = await getWorkspaceMessages({ session_id: targetSessionId })
    const msgs = res.messages
      .filter((msg) => msg.role === 'user' || msg.role === 'assistant')
      .map((msg) => ({
        role: msg.role as 'user' | 'assistant',
        content: msg.content,
        tool_calls: msg.tool_calls || [],
        created_at: msg.created_at,
      }))
    setMessages(msgs)
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
  // 如果会话有关联的智能体，更新选中的智能体
  const session = sessionList.value.find(s => s.session_id === targetSessionId)
  if (session?.agent_id) {
    selectedAgentId.value = session.agent_id
  }
  // 加载会话消息
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
  loadModes()
  loadAgents()
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
  if (!text || !props.fullCodePath) return
  inputText.value = ''
  const payload = {
    full_code_path: props.fullCodePath,
    message: { content: text },
    session_id: sessionId.value,
  } as { full_code_path: string; message: { content: string }; session_id?: string; agent_id?: number; mode?: string }
  if (selectedAgentId.value != null) payload.agent_id = selectedAgentId.value
  if (selectedModeCode.value) payload.mode = selectedModeCode.value

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
    await sendMessage(text, streamFn)
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
.session-card-agent {
  margin-bottom: 6px;
  font-size: 12px;
  color: var(--el-text-color-regular);
}
.agent-name {
  color: var(--el-text-color-regular);
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
/* 本消息发起的调用：放在消息下方、可折叠 */
.message-tool-calls-wrap {
  width: 100%;
  margin-top: 8px;
}
.message-tool-calls-details {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-lighter);
  overflow: hidden;
}
.message-tool-calls-summary {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  list-style: none;
  user-select: none;
}
.message-tool-calls-summary::-webkit-details-marker { display: none; }
.message-tool-calls-summary::before {
  content: '▶';
  font-size: 10px;
  transition: transform 0.2s;
  color: var(--el-text-color-placeholder);
}
.message-tool-calls-details[open] .message-tool-calls-summary::before {
  transform: rotate(90deg);
}
.message-tool-calls-summary:hover {
  background: var(--el-fill-color);
  color: var(--el-text-color-regular);
}
.message-tool-calls-details .tool-calls {
  padding: 8px 12px 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.tool-calls {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}
.input-area { display: flex; flex-direction: column; gap: 8px; }
.input-area .el-button { align-self: flex-end; }
</style>
