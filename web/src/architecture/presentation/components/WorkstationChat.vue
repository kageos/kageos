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
        <span class="title">智能工作台</span>
        <span v-if="fullCodePath" class="path">当前目录：{{ fullCodePath }}</span>
        <span v-else class="path empty">请先「返回工作空间」，在左侧服务目录对任意目录节点悬停点 ⋮ → 打开工作台</span>
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
          <span class="role">{{ m.role === 'user' ? '我' : '工作台' }}</span>
          <div v-if="m.role === 'assistant' && m.tool_calls?.length" class="tool-calls">
            <ToolCallCard
              v-for="(tc, j) in m.tool_calls"
              :key="tc.id || j"
              :tool-call="tc"
            />
          </div>
          <div class="content">{{ m.content }}<span v-if="sending && i === messages.length - 1 && m.role === 'assistant' && !m.content" class="streaming-cursor">▌</span></div>
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
import { workspaceChatStream, getWorkspaceSessions, getWorkspaceMessages, type WorkspaceSessionItem } from '@/api/workspace'
import { getAgentList } from '@/api/agent'
import ToolCallCard from './ToolCallCard.vue'
import type { AgentInfo } from '@/api/agent'
import { ElMessage } from 'element-plus'

const props = withDefaults(
  defineProps<{
    fullCodePath: string
    embedded?: boolean
  }>(),
  { embedded: false }
)
const emit = defineEmits<{ (e: 'back'): void }>()

const router = useRouter()

const inputText = ref('')
const sending = ref(false)
const messages = ref<
  Array<{ role: 'user' | 'assistant'; content: string; tool_calls?: Array<{ name: string; status: string }> }>
>([])
const sessionId = ref<string | undefined>(undefined)
const messagesRef = ref<HTMLElement | null>(null)

const agentList = ref<AgentInfo[]>([])
const agentLoading = ref(false)
const selectedAgentId = ref<number | null>(null)

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
    // 转换消息格式，过滤掉 tool 角色的消息（不需要在界面显示）
    messages.value = res.messages
      .filter(msg => msg.role === 'user' || msg.role === 'assistant')
      .map(msg => ({
        role: msg.role as 'user' | 'assistant',
        content: msg.content,
        tool_calls: msg.tool_calls || [],
      }))
    // 滚动到底部
    setTimeout(() => {
      if (messagesRef.value) {
        messagesRef.value.scrollTop = messagesRef.value.scrollHeight
      }
    }, 100)
  } catch (e: any) {
    console.error('加载会话消息失败:', e)
    ElMessage.error('加载会话消息失败')
    messages.value = []
  }
}

// 新建会话
function handleNewSession() {
  // 清空当前会话ID和消息
  sessionId.value = undefined
  messages.value = []
  // 保持当前选中的智能体（如果已选择）
  // 不重置 selectedAgentId，让用户继续使用之前选择的智能体
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
  loadAgents()
  if (props.fullCodePath) {
    loadSessions()
  }
})

// 监听 fullCodePath 变化，自动加载会话列表
watch(() => props.fullCodePath, (newPath) => {
  if (newPath) {
    loadSessions()
    // 切换目录时重置会话和消息
    sessionId.value = undefined
    messages.value = []
  } else {
    sessionList.value = []
    sessionId.value = undefined
    messages.value = []
  }
})

async function send() {
  const text = inputText.value.trim()
  if (!text || !props.fullCodePath) return
  inputText.value = ''
  messages.value.push({ role: 'user', content: text })
  messages.value.push({ role: 'assistant', content: '', tool_calls: [] })
  sending.value = true
  const idx = messages.value.length - 1
  try {
    const payload = {
      full_code_path: props.fullCodePath,
      message: { content: text },
      session_id: sessionId.value,
    } as { full_code_path: string; message: { content: string }; session_id?: string; agent_id?: number }
    if (selectedAgentId.value != null) payload.agent_id = selectedAgentId.value

    await workspaceChatStream(payload, (event, data) => {
      const m = messages.value[idx]
      if (!m || m.role !== 'assistant') return
      if (event === 'session' && typeof data.session_id === 'string') sessionId.value = data.session_id
      if (event === 'agent_id' && data.agent_id != null && Number(data.agent_id) > 0)
        selectedAgentId.value = Number(data.agent_id)
      if (event === 'tool_call' && typeof data.name === 'string') {
        const list = [...(m.tool_calls || []), { name: data.name, status: String(data.status || 'ok') }]
        messages.value[idx] = { ...m, tool_calls: list }
      }
      if (event === 'content' && typeof data.content === 'string') {
        messages.value[idx] = { ...m, content: m.content + data.content }
      }
      if (event === 'done') {
        sending.value = false
        if (Array.isArray(data.tool_calls)) {
          messages.value[idx] = { ...m, tool_calls: data.tool_calls as Array<{ name: string; status: string }> }
        }
        // 对话完成后刷新会话列表
        loadSessions()
      }
      if (event === 'error') {
        messages.value[idx] = { ...m, content: m.content || String(data.message || '请求失败') }
        sending.value = false
        ElMessage.error(String(data.message || '发送失败'))
      }
    })
  } catch (e: any) {
    const m = messages.value[idx]
    if (m && m.role === 'assistant')
      messages.value[idx] = { ...m, content: m.content || '请求失败：' + (e?.message || String(e)) }
    ElMessage.error('发送失败')
  } finally {
    sending.value = false
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
.message .role { font-weight: 600; font-size: 13px; color: var(--el-text-color-primary); }
.message.user .role { color: var(--el-color-primary); }
.message .content { white-space: pre-wrap; word-break: break-word; width: 100%; font-size: 14px; line-height: 1.5; color: var(--el-text-color-regular); }
.streaming-cursor { animation: blink 0.8s step-end infinite; color: var(--el-color-primary); }
@keyframes blink { 50% { opacity: 0; } }
.tool-calls { 
  display: flex; 
  flex-direction: column; 
  gap: 8px; 
  width: 100%;
  margin-top: 8px;
}
.input-area { display: flex; flex-direction: column; gap: 8px; }
.input-area .el-button { align-self: flex-end; }
</style>
