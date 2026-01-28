<template>
  <div class="ai-chat-panel">
    <!-- 会话列表侧边栏 -->
    <div class="session-sidebar">
      <div class="sidebar-header">
        <h4>会话列表</h4>
        <el-button
          type="primary"
          :icon="Plus"
          size="small"
          @click.stop="handleNewSession"
          title="新建会话"
        >
          新建
        </el-button>
      </div>
      <div class="session-list" v-loading="loadingSessions">
        <!-- 新建会话提示项（当没有选中会话时显示） -->
        <div
          v-if="!sessionId"
          class="session-card new-session-card"
        >
          <div class="session-card-header">
            <el-icon class="new-icon"><Plus /></el-icon>
            <span class="session-card-title">新会话</span>
          </div>
          <div v-if="currentAgent" class="session-card-agent">
            <el-avatar
              :size="20"
              :src="getAgentLogo(currentAgent)"
            >
              <span class="agent-logo-text">{{ getAgentLogoText(currentAgent) }}</span>
            </el-avatar>
            <span class="agent-name">{{ currentAgent.name }}</span>
          </div>
          <div class="session-card-time">
            <span>{{ formatRelativeTime(new Date()) }}</span>
          </div>
        </div>
        
        <!-- 会话列表项 -->
        <div
          v-for="session in sessionList"
          :key="session.session_id"
          :class="['session-card', { 
            active: session.session_id === sessionId,
            loading: loadingSession && pendingSessionId === session.session_id
          }]"
          @click="handleSelectSession(session.session_id)"
        >
          <div class="session-card-header">
            <div class="session-card-title-wrapper">
              <span class="session-card-title">{{ session.title || '未命名会话' }}</span>
              <el-icon v-if="loadingSession && pendingSessionId === session.session_id" class="loading-icon">
                <Loading />
              </el-icon>
            </div>
          </div>
          
          <div v-if="session.agent" class="session-card-agent">
            <el-avatar
              :size="20"
              :src="getAgentLogo(session.agent)"
            >
              <span class="agent-logo-text">{{ getAgentLogoText(session.agent) }}</span>
            </el-avatar>
            <span class="agent-name">{{ session.agent.name }}</span>
          </div>
          
          <div class="session-card-time">
            <span>{{ formatRelativeTime(session.updated_at) }}</span>
          </div>
        </div>
        
        <!-- 空状态 -->
        <div v-if="sessionList.length === 0 && !loadingSessions && sessionId" class="empty-sessions">
          <el-empty description="暂无会话" :image-size="80">
            <el-button type="primary" size="small" @click="handleNewSession">创建新会话</el-button>
          </el-empty>
        </div>
      </div>
    </div>

    <!-- 主聊天区域 -->
    <div class="chat-main">
      <div class="chat-header">
        <div class="header-left"></div>
        <div class="header-center">
          <div v-if="currentSessionAgent" class="header-agent-info">
            <el-avatar
              :size="28"
              :src="getAgentLogo(currentSessionAgent)"
              class="header-agent-avatar"
            >
              <span class="header-agent-logo-text">{{ getAgentLogoText(currentSessionAgent) }}</span>
            </el-avatar>
            <div class="header-agent-details">
              <div class="header-agent-name-row">
                <span class="header-agent-name">{{ currentSessionAgent.name }}</span>
                <el-tag 
                  size="small" 
                  :type="currentSessionAgent.agent_type === 'plugin' ? 'warning' : 'success'"
                  class="header-agent-tag"
                >
                  {{ currentSessionAgent.agent_type === 'plugin' ? '插件' : '知识库' }}
                </el-tag>
              </div>
              <div v-if="currentSessionAgent.description" class="header-agent-description">
                {{ currentSessionAgent.description }}
              </div>
            </div>
          </div>
          <div v-else class="header-agent-info">
            <span class="header-agent-name-placeholder">请选择智能体开始对话</span>
          </div>
        </div>
        <div class="header-right">
          <el-button
            link
            :icon="Close"
            @click="$emit('close')"
            class="close-button"
            title="关闭"
          />
        </div>
      </div>

    <div class="chat-messages" ref="messagesContainerRef">
      <div
        v-for="(message, index) in messages"
        :key="index"
        :class="['message-item', message.role]"
      >
        <div class="message-avatar">
          <el-avatar v-if="message.role === 'user'" :size="32">
            <el-icon><User /></el-icon>
          </el-avatar>
          <el-avatar v-else :size="32" style="background-color: #409eff">
            <el-icon><ChatRound /></el-icon>
          </el-avatar>
        </div>
        <div class="message-content">
          <div 
            :class="['message-text', { 
              'is-greeting': message.isGreeting,
              'is-collapsed': message.isGreeting && !message.isExpanded && needsExpand(message),
              'needs-expand': message.isGreeting && needsExpand(message)
            }]"
            v-html="message.isHtml ? message.content : formatMessage(message.content)"
            @click="handleMessageLinkClick"
          ></div>
          <!-- 开场白展开/收起按钮 -->
          <div v-if="message.isGreeting && needsExpand(message)" class="greeting-expand">
            <el-button
              text
              type="primary"
              size="small"
              @click="toggleGreetingExpand(index)"
            >
              {{ message.isExpanded ? '收起' : '展开' }}
              <el-icon>
                <ArrowDown v-if="!message.isExpanded" />
                <ArrowUp v-else />
              </el-icon>
            </el-button>
          </div>
          <!-- 显示文件列表 -->
          <div v-if="message.files && message.files.length > 0" class="message-files">
            <div v-for="(file, fileIndex) in message.files" :key="fileIndex" class="file-item">
              <el-icon><Document /></el-icon>
              <span class="file-name">{{ file.remark || '文件' }}</span>
              <el-link :href="file.url" target="_blank" type="primary" :underline="false">
                查看
              </el-link>
            </div>
          </div>
          <div class="message-time">{{ formatTime(message.timestamp) }}</div>
        </div>
      </div>

      <div v-if="loading" class="message-item assistant">
        <div class="message-avatar">
          <el-avatar :size="32" style="background-color: #409eff">
            <el-icon><ChatRound /></el-icon>
          </el-avatar>
        </div>
        <div class="message-content">
          <div class="message-text">
            <el-icon class="is-loading"><Loading /></el-icon>
            <span>AI 正在思考...</span>
          </div>
        </div>
      </div>
    </div>

    <div 
      class="chat-input"
      :class="{ 'drag-over': isDragOver }"
      @drop.prevent="handleDrop"
      @dragover.prevent="handleDragOver"
      @dragleave.prevent="handleDragLeave"
    >
      <!-- 文件列表显示 -->
      <div v-if="uploadedFiles.length > 0" class="uploaded-files">
        <div v-for="(file, index) in uploadedFiles" :key="index" class="file-tag">
          <el-icon><Document /></el-icon>
          <span>{{ file.remark || file.url.split('/').pop() }}</span>
          <el-button
            text
            type="danger"
            :icon="Close"
            size="small"
            @click="removeFile(index)"
          />
        </div>
      </div>
      
      <el-input
        v-model="inputMessage"
        type="textarea"
        :rows="3"
        placeholder="输入消息，按 Ctrl+Enter 发送（支持拖拽文件上传）"
        :disabled="loading || !canContinue"
        @keydown.ctrl.enter="handleSend"
        @keydown.meta.enter="handleSend"
        ref="inputRef"
      />
      <div class="input-actions">
        <el-upload
          :action="''"
          :auto-upload="false"
          :show-file-list="false"
          :on-change="handleFileSelect"
          :disabled="loading || !canContinue"
          accept="*"
          multiple
        >
          <el-button :icon="Upload" :disabled="loading || !canContinue">上传文件</el-button>
        </el-upload>
        <el-button
          type="primary"
          :loading="loading"
          :disabled="(!inputMessage.trim() && uploadedFiles.length === 0) || !canContinue"
          @click="handleSend"
        >
          发送
        </el-button>
        <el-button @click="handleClear">清空</el-button>
      </div>
    </div>
    </div>

    <!-- 智能体选择对话框 -->
    <AgentSelectDialog
      v-model="agentSelectDialogVisible"
      :full-code-path="fullCodePath"
      :package="package"
      :current-node-name="currentNodeName"
      @confirm="handleAgentSelect"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElNotification } from 'element-plus'
import { Close, User, Loading, ChatRound, Upload, Document, Plus, ArrowDown, ArrowUp } from '@element-plus/icons-vue'
import * as agentApi from '@/api/agent'
import type { AgentInfo, ChatSessionInfo } from '@/api/agent'
import { uploadFile, notifyUploadComplete, type UploadCompleteResult } from '@/utils/upload'
import { formatDuration } from '@/utils/date'
import type { UploadFile } from 'element-plus'
import { marked } from 'marked'
import AgentSelectDialog from '@/components/Agent/AgentSelectDialog.vue'
import { useAuthStore } from '@/stores/auth'
import { WidgetType, DataType } from '@/core/constants/widget'

interface Props {
  agentId: number | null
  fullCodePath: string | null // 服务目录完整路径
  package?: string // Package 名称
  currentNodeName?: string
  existingFiles?: string[] // 当前 package 下已存在的文件名（不含 .go 后缀）
}

const props = withDefaults(defineProps<Props>(), {
  agentId: null,
  fullCodePath: null,
  package: '',
  currentNodeName: '',
  existingFiles: () => []
})

const emit = defineEmits<{
  close: []
}>()

const router = useRouter()

// ⭐ ChatFile 与 types.File 保持一致（智能体插件场景）
interface ChatFile {
  name: string              // 文件名
  source_name: string       // 源文件名称
  storage: string           // 存储引擎类型（minio/qiniu/xxxxx）
  description: string      // 文件描述/备注
  hash: string              // 文件hash
  size: number              // 文件大小（字节）
  upload_ts: number         // 上传时间戳（毫秒）
  local_path: string        // 本地路径（前端不需要，设为空）
  is_uploaded: boolean    // 是否已上传到云端
  url: string               // 外部访问地址（前端下载使用）
  server_url: string        // 内部访问地址（服务端下载使用）
  downloaded: boolean       // 是否已下载到本地（前端不需要，设为false）
  upload_user: string       // 上传用户
}

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  files?: ChatFile[]
  timestamp: number
  isHtml?: boolean // 标记内容是否为 HTML 格式（用于开场白等）
  isGreeting?: boolean // 标记是否为开场白
  isExpanded?: boolean // 标记是否已展开（用于开场白）
}

const messages = ref<ChatMessage[]>([])
const inputMessage = ref('')
const loading = ref(false)
const messagesContainerRef = ref<HTMLElement>()
const inputRef = ref<InstanceType<typeof HTMLTextAreaElement>>()
const canContinue = ref(true) // 是否可以继续输入

// 轮询相关
const pollingTimers = ref<Map<number, NodeJS.Timeout>>(new Map()) // 每个 record_id 对应的定时器
const pollingRecordIds = ref<Set<number>>(new Set()) // 正在轮询的 record_id 集合
// 轮询状态：记录每个 recordId 的轮询次数和开始时间
const pollingStates = ref<Map<number, { count: number; startTime: number }>>(new Map())

// 文件上传相关
const uploadedFiles = ref<ChatFile[]>([])
const isDragOver = ref(false)

// 获取当前用户信息（用于文件上传）
const authStore = useAuthStore()

// 智能体选择相关
const selectedAgentId = ref<number | null>(props.agentId)
const agentOptions = ref<AgentInfo[]>([])
const agentLoading = ref(false)

// 当前选中的智能体信息（用于新建会话时显示）
const currentAgent = computed(() => {
  if (!selectedAgentId.value) return null
  return agentOptions.value.find(agent => agent.id === selectedAgentId.value) || null
})

// 当前会话的智能体信息（用于header显示）
const currentSessionAgent = computed(() => {
  if (!sessionId.value) {
    // 如果没有会话，显示当前选中的智能体（新建会话时）
    return currentAgent.value
  }
  // 如果有会话，从会话列表中查找对应的智能体
  const session = sessionList.value.find(s => s.session_id === sessionId.value)
  return session?.agent || null
})

// 会话ID（首次为空，后端自动生成）
const sessionId = ref<string>('')
const loadingSession = ref(false)
// 正在加载的会话ID（用于显示加载状态）
const pendingSessionId = ref<string | null>(null)
// 请求取消控制器（用于取消正在进行的请求）
let currentAbortController: AbortController | null = null

// 会话列表相关
const sessionList = ref<ChatSessionInfo[]>([])
const loadingSessions = ref(false)

// 智能体选择对话框
const agentSelectDialogVisible = ref(false)

// 获取智能体 Logo（如果有则使用，否则使用默认生成的）
function getAgentLogo(agent: AgentInfo): string {
  if (agent.logo) {
    return agent.logo
  }
  // 生成默认 Logo（使用智能体 ID 生成唯一颜色）
  return generateDefaultLogo(agent.id, agent.name)
}

// 生成默认 Logo URL（使用智能体 ID 生成唯一颜色）
function generateDefaultLogo(agentId: number, agentName: string): string {
  // 使用智能体 ID 生成一个稳定的颜色
  const colors = [
    '#409EFF', '#67C23A', '#E6A23C', '#F56C6C', '#909399',
    '#606266', '#303133', '#409EFF', '#67C23A', '#E6A23C'
  ]
  const colorIndex = agentId % colors.length
  const color = colors[colorIndex]
  
  // 生成 SVG data URL
  const svg = `
    <svg width="48" height="48" xmlns="http://www.w3.org/2000/svg">
      <rect width="48" height="48" fill="${color}" rx="8"/>
      <text x="24" y="32" font-family="Arial, sans-serif" font-size="20" font-weight="bold" fill="white" text-anchor="middle">${getAgentLogoText({ id: agentId, name: agentName } as AgentInfo)}</text>
    </svg>
  `.trim()
  
  return `data:image/svg+xml;base64,${btoa(unescape(encodeURIComponent(svg)))}`
}

// 获取智能体 Logo 文本（取名称首字符）
function getAgentLogoText(agent: AgentInfo): string {
  if (!agent.name) return 'A'
  // 取第一个字符（支持中文）
  const firstChar = agent.name.charAt(0)
  return firstChar.toUpperCase()
}

// 加载智能体列表
async function loadAgents() {
  agentLoading.value = true
  try {
    const res = await agentApi.getAgentList({
      enabled: true,
      page: 1,
      page_size: 1000
    })
    agentOptions.value = res.agents || []
    
    if (!selectedAgentId.value && props.agentId) {
      selectedAgentId.value = props.agentId
    }
    
    if (!selectedAgentId.value && agentOptions.value.length > 0) {
      selectedAgentId.value = agentOptions.value[0].id
    }
  } catch (error: any) {
    console.error('加载智能体列表失败:', error)
    ElMessage.error(error.message || '加载智能体列表失败')
  } finally {
    agentLoading.value = false
  }
}

// 加载会话列表
async function loadSessionList() {
  if (!props.fullCodePath) {
    sessionList.value = []
    return
  }

  loadingSessions.value = true
  try {
    const res = await agentApi.getChatSessionList({
      full_code_path: props.fullCodePath,
      page: 1,
      page_size: 50 // 加载最近50个会话
    })
    sessionList.value = res.sessions || []
  } catch (error: any) {
    console.error('[AIChatPanel] 加载会话列表失败:', error)
    ElMessage.error(error.message || '加载会话列表失败')
  } finally {
    loadingSessions.value = false
  }
}

// 加载指定会话的消息
async function loadSessionMessages(targetSessionId: string, signal?: AbortSignal) {
  try {
    const messageRes = await agentApi.getChatMessageList({
      session_id: targetSessionId
    })

    // 检查请求是否已被取消
    if (signal?.aborted) {
      return
    }

    // 检查是否仍然是要加载的会话
    if (sessionId.value !== targetSessionId) {
      return
    }

    // 根据会话状态设置 canContinue
    const session = sessionList.value.find(s => s.session_id === targetSessionId)
    if (session) {
      // 只有 active 状态才能继续输入
      canContinue.value = session.status === 'active'
    } else {
      // 如果找不到会话信息，默认可以继续输入
      canContinue.value = true
    }
    
    if (messageRes.messages && messageRes.messages.length > 0) {
      // 转换消息格式
      messages.value = messageRes.messages.map(msg => {
        let files: ChatFile[] | undefined
        if (msg.files) {
          try {
            const filesData = JSON.parse(msg.files)
            files = Array.isArray(filesData) ? filesData.map((f: any) => ({
              url: f.url || '',
              remark: f.remark || ''
            })) : undefined
          } catch (e) {
            console.error('[AIChatPanel] 解析文件列表失败:', e)
          }
        }
        return {
          role: msg.role as 'user' | 'assistant',
          content: msg.content,
          files,
          timestamp: parseDateTime(msg.created_at),
          isHtml: false,
          isGreeting: false,
          isExpanded: false
        }
      })

      // 滚动到底部
      nextTick(() => {
        scrollToBottom()
      })
    } else {
      // 如果没有消息，显示欢迎消息（优先使用智能体的开场白）
      messages.value = []
      const agent = sessionList.value.find(s => s.session_id === targetSessionId)?.agent
      if (agent?.greeting) {
        // 如果有开场白，根据格式类型渲染
        const greetingHtml = renderGreeting(agent.greeting, agent.greeting_type)
        addMessage('assistant', greetingHtml, undefined, agent.greeting_type === 'html', true)
      } else {
        // 如果没有开场白，使用默认欢迎消息
        const agentName = agent?.name || 'AI 助手'
        if (props.currentNodeName) {
          addMessage('assistant', `你好！我是 ${agentName}，可以帮助你处理「${props.currentNodeName}」相关的问题。有什么可以帮助你的吗？`)
        } else {
          addMessage('assistant', `你好！我是 ${agentName}，有什么可以帮助你的吗？`)
        }
      }
    }
  } catch (error: any) {
    // 如果请求被取消，不显示错误
    if (signal?.aborted) {
      return
    }
    
    // 检查是否仍然是要加载的会话
    if (sessionId.value !== targetSessionId) {
      return
    }
    
    console.error('[AIChatPanel] 加载会话消息失败:', error)
    ElMessage.error(error.message || '加载会话消息失败')
    
    // 加载失败时显示欢迎消息（优先使用智能体的开场白）
    messages.value = []
    const agent = sessionList.value.find(s => s.session_id === targetSessionId)?.agent
    if (agent?.greeting) {
      // 如果有开场白，根据格式类型渲染
      const greetingHtml = renderGreeting(agent.greeting, agent.greeting_type)
      addMessage('assistant', greetingHtml, undefined, agent.greeting_type === 'html', true)
    } else {
      // 如果没有开场白，使用默认欢迎消息
      const agentName = agent?.name || 'AI 助手'
      if (props.currentNodeName) {
        addMessage('assistant', `你好！我是 ${agentName}，可以帮助你处理「${props.currentNodeName}」相关的问题。有什么可以帮助你的吗？`)
      } else {
        addMessage('assistant', `你好！我是 ${agentName}，有什么可以帮助你的吗？`)
      }
    }
  }
}

// 从后端加载会话和消息
async function loadSessionFromBackend() {
  // 如果正在创建新会话，不加载旧会话
  if (isCreatingNewSession.value) {
    console.log('[AIChatPanel] 正在创建新会话，跳过加载旧会话')
    return
  }
  
  // 如果 sessionId 为空且消息列表不为空，说明正在创建新会话，不加载旧会话
  if (!sessionId.value && messages.value.length > 0) {
    console.log('[AIChatPanel] 检测到新会话状态，跳过加载旧会话')
    return
  }
  
  if (!props.fullCodePath) {
    // 如果没有 fullCodePath，显示欢迎消息
    if (messages.value.length === 0) {
      if (props.currentNodeName) {
        addMessage('assistant', `你好！我是 AI 助手，可以帮助你处理「${props.currentNodeName}」相关的问题。有什么可以帮助你的吗？`)
      } else {
        addMessage('assistant', '你好！我是 AI 助手，有什么可以帮助你的吗？')
      }
    }
    return
  }

  // 先加载会话列表
  await loadSessionList()

  // 如果正在创建新会话，不加载旧会话
  if (isCreatingNewSession.value) {
    console.log('[AIChatPanel] 加载会话列表后检测到正在创建新会话，跳过加载旧会话')
    return
  }

  // 如果有会话列表且 sessionId 为空，加载最新的会话
  if (sessionList.value.length > 0 && !sessionId.value) {
    const latestSession = sessionList.value[0]
    sessionId.value = latestSession.session_id
    loadingSession.value = true
    pendingSessionId.value = latestSession.session_id
    try {
      await loadSessionMessages(latestSession.session_id)
    } finally {
      loadingSession.value = false
      pendingSessionId.value = null
    }
  } else if (sessionList.value.length === 0) {
    // 如果没有会话，显示欢迎消息
    sessionId.value = ''
    messages.value = []
    if (props.currentNodeName) {
      addMessage('assistant', `你好！我是 AI 助手，可以帮助你处理「${props.currentNodeName}」相关的问题。有什么可以帮助你的吗？`)
    } else {
      addMessage('assistant', '你好！我是 AI 助手，有什么可以帮助你的吗？')
    }
  }
}

// 新建会话
function handleNewSession() {
  console.log('[AIChatPanel] 新建会话被点击, fullCodePath:', props.fullCodePath)
  
  if (!props.fullCodePath) {
    ElMessage.warning('缺少服务目录路径，无法创建会话')
    return
  }
  
  // 弹出智能体选择对话框
  agentSelectDialogVisible.value = true
}

// 处理智能体选择（从外部选择智能体时调用）
function handleAgentSelect(agent: AgentInfo) {
  console.log('[AIChatPanel] 选择智能体，创建新会话:', agent)
  
  // 设置创建新会话标志，防止 watch 监听器加载旧会话
  isCreatingNewSession.value = true
  
  // 设置选中的智能体
  selectedAgentId.value = agent.id
  
  // 清空当前会话ID，表示新建会话
  sessionId.value = ''
  messages.value = []
  uploadedFiles.value = []
  canContinue.value = true // 新建会话时可以继续输入
  
  // 刷新会话列表（确保显示最新的会话）
  loadSessionList()
  
  // 显示欢迎消息（优先使用智能体的开场白）
  if (agent.greeting) {
    // 如果有开场白，根据格式类型渲染
    const greetingHtml = renderGreeting(agent.greeting, agent.greeting_type)
    addMessage('assistant', greetingHtml, undefined, agent.greeting_type === 'html')
  } else {
    // 如果没有开场白，使用默认欢迎消息
    if (props.currentNodeName) {
      addMessage('assistant', `你好！我是 ${agent.name}，可以帮助你处理「${props.currentNodeName}」相关的问题。有什么可以帮助你的吗？`)
    } else {
      addMessage('assistant', `你好！我是 ${agent.name}，有什么可以帮助你的吗？`)
    }
  }
  
  // 滚动到底部
  nextTick(() => {
    scrollToBottom()
  })
  
  // 延迟清除创建新会话标志，确保不会被 loadSessionFromBackend 覆盖
  setTimeout(() => {
    isCreatingNewSession.value = false
  }, 500)
  
  ElMessage.success('已创建新会话，发送第一条消息后将自动保存')
}

// 是否正在手动切换会话（用于防止 watch 监听器触发）
const isManualSwitching = ref(false)
// 是否正在创建新会话（用于防止 watch 监听器加载旧会话）
const isCreatingNewSession = ref(false)

// 选择会话
async function handleSelectSession(targetSessionId: string) {
  // 如果点击的是当前会话，直接返回（不重新加载）
  if (targetSessionId === sessionId.value && !loadingSession.value) {
    return
  }
  
  // 取消之前的请求（如果有）
  if (currentAbortController) {
    currentAbortController.abort()
    currentAbortController = null
  }
  
  // 设置手动切换标志，防止 watch 监听器触发
  isManualSwitching.value = true
  
  // 查找会话信息，设置对应的智能体
  const session = sessionList.value.find(s => s.session_id === targetSessionId)
  if (session && session.agent_id) {
    selectedAgentId.value = session.agent_id
  }
  
  // 立即更新会话ID和UI状态
  sessionId.value = targetSessionId
  messages.value = []
  uploadedFiles.value = []
  
  // 根据会话状态设置 canContinue
  if (session) {
    // 只有 active 状态才能继续输入
    canContinue.value = session.status === 'active'
  } else {
    // 如果找不到会话信息，默认可以继续输入
    canContinue.value = true
  }
  
  // 创建新的 AbortController
  const abortController = new AbortController()
  currentAbortController = abortController
  
  // 设置加载状态
  loadingSession.value = true
  pendingSessionId.value = targetSessionId
  
  try {
    // 加载会话消息
    await loadSessionMessages(targetSessionId, abortController.signal)
  } catch (error: any) {
    // 如果请求被取消，不显示错误
    if (abortController.signal.aborted) {
      return
    }
    console.error('[AIChatPanel] 加载会话消息失败:', error)
  } finally {
    // 只有当前请求没有被取消时，才清除加载状态
    if (!abortController.signal.aborted && sessionId.value === targetSessionId) {
      loadingSession.value = false
      pendingSessionId.value = null
    }
    // 如果这是当前请求，清除引用
    if (currentAbortController === abortController) {
      currentAbortController = null
    }
    // 清除手动切换标志
    isManualSwitching.value = false
  }
}

// 智能体变化处理（已移除，智能体选择通过新建会话实现）
// async function handleAgentChange() {
//   messages.value = []
//   sessionId.value = '' // 切换智能体时重置会话ID
//   uploadedFiles.value = []
//   // 从后端加载新智能体的会话记录
//   await loadSessionFromBackend()
// }

// 组件卸载时清理轮询
onUnmounted(() => {
  stopAllPolling()
})

// 初始化欢迎消息
onMounted(async () => {
  await loadAgents()
  
  // 从后端加载会话记录（如果有）
  await loadSessionFromBackend()
})

// 监听目录切换，恢复会话记录
watch(
  () => [props.fullCodePath, props.package, props.currentNodeName, selectedAgentId.value],
  async ([newFullCodePath, newPackage, newNodeName, newAgentId], [oldFullCodePath, oldPackage, oldNodeName, oldAgentId]) => {
    // 如果正在手动切换会话，不触发自动加载
    if (isManualSwitching.value) {
      return
    }
    
    // 如果正在创建新会话，不触发自动加载（避免加载旧会话）
    if (isCreatingNewSession.value) {
      return
    }
    
    // 如果 fullCodePath、package 或 agentId 变化，说明切换了目录或智能体
    if (newFullCodePath !== oldFullCodePath || newPackage !== oldPackage || newAgentId !== oldAgentId) {
      // 清空当前状态
      messages.value = []
      sessionId.value = ''
      uploadedFiles.value = []
      
      // 从后端加载新目录/智能体的会话记录（会同时加载会话列表和消息）
      await loadSessionFromBackend()
    } else if (newNodeName !== oldNodeName) {
      // 如果只是目录名称变化（但 treeId 和 package 没变），更新欢迎消息
      // 这种情况比较少见，但为了完整性还是处理一下
      if (messages.value.length > 0 && messages.value[0].role === 'assistant') {
        messages.value[0].content = newNodeName
          ? `你好！我是 AI 助手，可以帮助你处理「${newNodeName}」相关的问题。有什么可以帮助你的吗？`
          : '你好！我是 AI 助手，有什么可以帮助你的吗？'
      }
    }
  }
)

// 🔥 监听 agentId prop 变化，更新选中的智能体（已移除，智能体选择通过新建会话实现）
// watch(
//   () => props.agentId,
//   (newAgentId) => {
//     if (newAgentId && newAgentId !== selectedAgentId.value) {
//       selectedAgentId.value = newAgentId
//       // 切换智能体时重置会话
//       handleAgentChange()
//     }
//   }
// )

// 监听消息变化，自动滚动到底部
watch(
  () => messages.value.length,
  () => {
    nextTick(() => {
      scrollToBottom()
    })
  }
)

// 添加消息
function addMessage(role: 'user' | 'assistant', content: string, files?: ChatFile[], isHtml: boolean = false, isGreeting: boolean = false) {
  messages.value.push({
    role,
    content,
    files,
    timestamp: Date.now(),
    isHtml,
    isGreeting,
    isExpanded: false // 开场白默认收起
  })
  // 注意：消息已由后端保存，不需要前端保存
}

// 处理文件选择（el-upload 组件）
async function handleFileSelect(file: UploadFile) {
  await processFile(file.raw)
}

// 处理拖拽上传
async function handleDrop(event: DragEvent) {
  isDragOver.value = false
  const files = event.dataTransfer?.files
  if (!files || files.length === 0) return

  for (let i = 0; i < files.length; i++) {
    await processFile(files[i])
  }
}

// 处理拖拽悬停
function handleDragOver(event: DragEvent) {
  event.preventDefault()
  isDragOver.value = true
}

// 处理拖拽离开
function handleDragLeave() {
  isDragOver.value = false
}

// 处理单个文件（上传逻辑）
async function processFile(rawFile: File | null) {
  if (!rawFile) return

  try {
    ElMessage.info(`正在上传 ${rawFile.name}...`)
    
    // 上传文件
    const uploadResult = await uploadFile(
      'agent/chat/files', // 上传路由
      rawFile,
      () => {} // 不显示进度
    )
    
    // 通知后端上传完成
    if (uploadResult.fileInfo) {
      // ⭐ 获取当前用户信息
      const currentUser = authStore.user?.username || ''
      
      // ⭐ 获取完整文件信息（包括 server_url）
      const completeResult = await notifyUploadComplete({
        key: uploadResult.fileInfo.key,
        success: true,
        router: uploadResult.fileInfo.router,
        file_name: uploadResult.fileInfo.file_name,
        file_size: uploadResult.fileInfo.file_size,
        content_type: uploadResult.fileInfo.content_type,
        hash: uploadResult.fileInfo.hash,
        storage: uploadResult.storage, // ⭐ 传递存储引擎类型
        upload_user: currentUser, // ⭐ 传递上传用户
      })
      
      if (completeResult) {
        // ⭐ 保存完整文件信息（与 types.File 保持一致）
        // ⭐ 使用原始文件名作为 name 和 source_name，不要使用后端返回的 file_name（可能是 UUID）
        uploadedFiles.value.push({
          name: rawFile.name, // ⭐ 使用原始文件名
          source_name: rawFile.name, // ⭐ 使用原始文件名
          storage: completeResult.storage || 'minio',
          description: rawFile.name, // 使用原始文件名作为描述
          hash: completeResult.hash || '',
          size: completeResult.file_size,
          upload_ts: Date.now(),
          local_path: '', // 前端不需要
          is_uploaded: true,
          url: completeResult.download_url,
          server_url: completeResult.server_download_url || completeResult.download_url, // 如果没有 server_url，使用 download_url
          downloaded: false, // 前端不需要
          upload_user: currentUser,
        })
        ElMessage.success(`${rawFile.name} 上传成功`)
      } else {
        throw new Error('获取下载地址失败')
      }
    }
  } catch (error: any) {
    console.error('[AIChatPanel] 文件上传失败:', error)
    ElMessage.error(error.message || '文件上传失败')
  }
}

// 移除文件
function removeFile(index: number) {
  uploadedFiles.value.splice(index, 1)
}

// 发送消息
async function handleSend() {
  if ((!inputMessage.value.trim() && uploadedFiles.value.length === 0) || loading.value) {
    return
  }

  if (!selectedAgentId.value) {
    ElMessage.warning('请先选择一个智能体')
    return
  }

  if (!props.fullCodePath) {
    ElMessage.warning('缺少服务目录路径')
    return
  }

  const userMessage = inputMessage.value.trim()
  const files = [...uploadedFiles.value]
  
  // 清空输入和文件列表
  inputMessage.value = ''
  uploadedFiles.value = []

  // 添加用户消息
  addMessage('user', userMessage || '(无文本消息)', files.length > 0 ? files : undefined)

  // 发送请求
  loading.value = true
  try {
    const res = await agentApi.functionGenChat({
      agent_id: selectedAgentId.value,
      full_code_path: props.fullCodePath,
      package: props.package || '', // 传递 Package 名称
      session_id: sessionId.value || '', // 首次为空，后端自动生成
      existing_files: props.existingFiles || [], // 传递已存在的文件名
      message: {
        content: userMessage || '',
        // ⭐ 直接传递 types.Files 格式
        files: files.length > 0 ? {
          files: files.map(f => ({
            name: f.name,
            source_name: f.source_name,
            storage: f.storage,
            description: f.description,
            hash: f.hash,
            size: f.size,
            upload_ts: f.upload_ts,
            local_path: f.local_path,
            is_uploaded: f.is_uploaded,
            url: f.url,
            server_url: f.server_url,
            downloaded: f.downloaded,
            upload_user: f.upload_user,
          })),
          widget_type: WidgetType.FILES,
          data_type: DataType.STRUCT,
          remark: '',
          metadata: {},
        } : undefined,
      }
    })

    // 更新会话ID（首次创建时返回）
    if (res.session_id) {
      sessionId.value = res.session_id
      // 如果创建了新会话，刷新会话列表
      await loadSessionList()
    }

    // 添加 AI 回复
    addMessage('assistant', res.content || '抱歉，AI 没有返回内容')
    // 注意：消息已由后端保存，不需要前端保存

    // 更新是否可以继续输入的状态
    canContinue.value = res.can_continue ?? false

    // 如果返回了 record_id，开始轮询状态
    if (res.record_id) {
      startPolling(res.record_id)
    }
  } catch (error: any) {
    ElMessage.error(error.message || '发送消息失败')
    // 移除用户消息（因为发送失败）
    messages.value.pop()
    // 恢复文件列表
    uploadedFiles.value = files
  } finally {
    loading.value = false
    nextTick(() => {
      inputRef.value?.focus()
    })
  }
}

// 清空消息
function handleClear() {
  messages.value = []
  sessionId.value = ''
  uploadedFiles.value = []
  if (props.currentNodeName) {
    addMessage('assistant', `你好！我是 AI 助手，可以帮助你处理「${props.currentNodeName}」相关的问题。有什么可以帮助你的吗？`)
  } else {
    addMessage('assistant', '你好！我是 AI 助手，有什么可以帮助你的吗？')
  }
}

// 配置 marked 选项
marked.setOptions({
  breaks: true, // 支持换行
  gfm: true, // 支持 GitHub Flavored Markdown
  headerIds: false, // 不生成 header IDs
  mangle: false // 不混淆邮箱地址
})

// 格式化消息内容（支持 Markdown）
function formatMessage(content: string): string {
  if (!content) return ''
  
  try {
    // 使用 marked 渲染 Markdown
    const html = marked.parse(content) as string
    return html
  } catch (error) {
    console.error('[AIChatPanel] Markdown 渲染失败:', error)
    // 如果渲染失败，返回转义后的原始内容
    return content
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/\n/g, '<br>')
  }
}

// 处理消息中的链接点击（用于跳转到函数组）
function handleMessageLinkClick(event: MouseEvent) {
  const target = event.target as HTMLElement
  // 检查是否点击的是链接
  if (target.tagName === 'A' && target.getAttribute('href')) {
    const href = target.getAttribute('href')
    // 检查是否是 full_group_code 格式（以 / 开头，包含至少 3 段路径）
    if (href && href.startsWith('/') && href.split('/').filter(Boolean).length >= 3) {
      event.preventDefault()
      // 更新路由，添加 full_group_code 查询参数
      router.push({
        path: router.currentRoute.value.path,
        query: {
          ...router.currentRoute.value.query,
          full_group_code: href
        }
      })
    }
  }
}

// ==================== 轮询相关 ====================

/**
 * 智能轮询策略：根据已用时间和轮询次数动态调整间隔
 * 
 * 策略说明：
 * - 模型升级后，20秒内有可能就成功，所以需要更快的响应
 * - 第一次：8秒后轮询（快速开始检查）
 * - 后续每次：3秒间隔（保持快速响应）
 * - 超时后（超过2分钟）：降低频率（10秒），因为可能出问题了
 * 
 * 具体策略：
 * - 第1次：8秒后轮询（快速开始）
 * - 第2次及以后：3秒后轮询（保持快速响应）
 * - 超过2分钟：10秒间隔（可能出问题，降低频率）
 */
function getPollInterval(count: number, elapsed: number): number {
  // 超时阈值：2分钟（120秒）
  const TIMEOUT_THRESHOLD = 120 * 1000
  
  // 如果超过超时阈值，降低频率（10秒）
  if (elapsed > TIMEOUT_THRESHOLD) {
    return 10 * 1000
  }
  
  // 根据轮询次数决定间隔
  if (count === 1) {
    // 第1次：8秒后轮询（快速开始检查）
    return 8 * 1000
  } else {
    // 第2次及以后：3秒后轮询（保持快速响应）
    return 3 * 1000
  }
}

// 开始轮询代码生成状态
function startPolling(recordId: number) {
  // 如果已经在轮询这个 record_id，不重复启动
  if (pollingRecordIds.value.has(recordId)) {
    return
  }
  
  pollingRecordIds.value.add(recordId)
  
  // 初始化轮询状态（count 从 1 开始，因为第一次轮询是第 1 次）
  const startTime = Date.now()
  pollingStates.value.set(recordId, { count: 1, startTime })
  
  // 轮询函数
  const poll = async () => {
    // 检查是否还在轮询列表中
    if (!pollingRecordIds.value.has(recordId)) {
      return
    }
    
    // 获取当前轮询状态
    const state = pollingStates.value.get(recordId)
    if (!state) {
      return
    }
    
    // 当前轮询次数
    const currentCount = state.count
    
    try {
      const res = await agentApi.getFunctionGenStatus({ record_id: recordId })
      
      if (res.status === 'completed') {
        // 生成完成，停止轮询并发送通知
        stopPolling(recordId)
        
        // 刷新消息列表（获取系统消息）
        if (sessionId.value) {
          await loadSessionMessages(sessionId.value)
          // 生成完成后，会话状态变为 done，不能再输入
          canContinue.value = false
        }
        
        // 发送成功通知
        const durationText = res.duration ? `（耗时：${formatDuration(res.duration)}）` : ''
        
        // 构建通知消息，包含函数完整代码路径按钮和耗时
        let notificationMessage = `代码生成已完成${durationText}`
        if (res.full_code_paths && res.full_code_paths.length > 0) {
          const buttons = res.full_code_paths.map((code: string, index: number) => {
            // 构建函数详情页面 URL：域名 + /workspace + 函数路径 + ?_node_type=function
            const fullCodePath = code.startsWith('/') ? code : `/${code}`
            const url = `${window.location.origin}/workspace${fullCodePath}?_node_type=function`
            // 按钮只显示4个字
            const buttonText = '查看详情'
            // 使用按钮样式的链接，点击在新窗口打开
            return `<a href="${url}" target="_blank" onclick="event.preventDefault(); window.open('${url}', '_blank'); return false;" style="display: inline-block; padding: 6px 12px; margin: 4px 8px 4px 0; background-color: #67C23A; color: white; text-decoration: none; border-radius: 4px; cursor: pointer; font-size: 12px; transition: background-color 0.3s;" onmouseover="this.style.backgroundColor='#5daf34'" onmouseout="this.style.backgroundColor='#67C23A'">${buttonText}</a>`
          }).join('')
          notificationMessage = `已生成 ${res.full_code_paths.length} 个函数${durationText}：<br><div style="margin-top: 8px;">${buttons}</div>`
        }
        
        ElNotification({
          title: '代码生成完成',
          dangerouslyUseHTMLString: true,
          message: notificationMessage,
          type: 'success',
          duration: 0, // 不自动关闭，需要手动点击关闭或点击跳转
          onClick: () => {
            // 点击通知时，如果有函数路径，跳转到第一个
            if (res.full_code_paths && res.full_code_paths.length > 0) {
              const firstCode = res.full_code_paths[0]
              const fullCodePath = firstCode.startsWith('/') ? firstCode : `/${firstCode}`
              router.push({
                path: `/workspace${fullCodePath}`,
                query: {
                  _node_type: 'function'
                }
              })
            }
          }
        })
      } else if (res.status === 'failed') {
        // 生成失败，停止轮询并发送通知
        stopPolling(recordId)
        
        // 刷新消息列表
        if (sessionId.value) {
          await loadSessionMessages(sessionId.value)
          // 生成失败后，会话状态恢复为 active，可以继续输入
          canContinue.value = true
        }
        
        // 发送失败通知
        ElNotification({
          title: '代码生成失败',
          message: res.error_msg || '代码生成过程中出现错误',
          type: 'error',
          duration: 5000
        })
      } else {
        // generating 状态：继续轮询，使用智能间隔
        // 更新轮询次数（为下一次轮询准备）
        state.count++
        const elapsed = Date.now() - state.startTime
        const interval = getPollInterval(state.count, elapsed)
        
        // 使用 setTimeout 而不是 setInterval，因为间隔是动态的
        const timer = setTimeout(() => {
          poll()
        }, interval)
        
        // 更新定时器引用
        pollingTimers.value.set(recordId, timer)
      }
    } catch (error: any) {
      console.error('[AIChatPanel] 轮询状态失败:', error)
      // 轮询失败不中断，继续尝试（使用默认间隔）
      if (state) {
        state.count++
        const elapsed = Date.now() - state.startTime
        const interval = getPollInterval(state.count, elapsed)
        
        const timer = setTimeout(() => {
          poll()
        }, interval)
        
        pollingTimers.value.set(recordId, timer)
      }
    }
  }
  
  // 第一次轮询：等待 30 秒后执行（count = 1）
  const firstInterval = getPollInterval(1, 0)
  const timer = setTimeout(() => {
    poll()
  }, firstInterval)
  
  // 保存定时器引用
  pollingTimers.value.set(recordId, timer)
}

// 停止轮询
function stopPolling(recordId: number) {
  pollingRecordIds.value.delete(recordId)
  // 清理对应的定时器
  const timer = pollingTimers.value.get(recordId)
  if (timer) {
    clearTimeout(timer)
    pollingTimers.value.delete(recordId)
  }
  // 清理轮询状态
  pollingStates.value.delete(recordId)
}

// 停止所有轮询
function stopAllPolling() {
  pollingRecordIds.value.clear()
  // 清理所有定时器
  pollingTimers.value.forEach((timer) => {
    clearTimeout(timer)
  })
  pollingTimers.value.clear()
  // 清理所有轮询状态
  pollingStates.value.clear()
}

// 根据格式类型渲染开场白
function renderGreeting(greeting: string, greetingType?: string): string {
  if (!greeting) return ''
  
  const type = greetingType || 'text'
  
  switch (type) {
    case 'md':
      try {
        return marked.parse(greeting) as string
      } catch (error) {
        console.error('[AIChatPanel] Markdown 渲染失败:', error)
        return greeting.replace(/\n/g, '<br>')
      }
    case 'html':
      return greeting
    case 'text':
    default:
      // 普通文本，转义 HTML 并保留换行
      return greeting
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/\n/g, '<br>')
  }
}

// 获取智能体的开场白（如果有）
function getAgentGreeting(agent: AgentInfo | null): string {
  if (!agent || !agent.greeting) {
    return ''
  }
  return renderGreeting(agent.greeting, agent.greeting_type)
}

// 判断开场白是否需要展开按钮（内容超过一定高度）
function needsExpand(message: ChatMessage): boolean {
  if (!message.isGreeting) return false
  // 简单判断：如果内容长度超过 500 字符，或者包含多个段落，可能需要展开
  return message.content.length > 500 || (message.content.match(/<p>|<\/p>|<div>|<\/div>/g)?.length || 0) > 3
}

// 切换开场白展开/收起状态
function toggleGreetingExpand(index: number) {
  if (messages.value[index]) {
    messages.value[index].isExpanded = !messages.value[index].isExpanded
  }
}

// 解析时间字符串（支持多种格式：DateTime、RFC3339等）
function parseDateTime(timeStr: string): number {
  if (!timeStr) return Date.now()
  
  // 尝试解析多种格式
  // 格式1: "2006-01-02 15:04:05" (time.DateTime，本地时间格式)
  // 格式2: "2006-01-02T15:04:05Z" (RFC3339 UTC)
  // 格式3: "2006-01-02T15:04:05+08:00" (RFC3339 with timezone)
  
  let date: Date
  
  // 如果包含 T 和 Z 或时区信息，是 RFC3339 格式
  if (timeStr.includes('T') && (timeStr.includes('Z') || timeStr.match(/[+-]\d{2}:\d{2}$/))) {
    date = new Date(timeStr)
  } else if (timeStr.includes(' ')) {
    // 如果是 "2006-01-02 15:04:05" 格式，后端返回的是本地时间（没有时区信息）
    // 需要手动解析为本地时间，而不是当作 UTC 时间
    // 格式：YYYY-MM-DD HH:mm:ss
    const parts = timeStr.split(' ')
    if (parts.length === 2) {
      const datePart = parts[0]?.split('-') || []
      const timePart = parts[1]?.split(':') || []
      if (datePart.length === 3 && timePart.length >= 2) {
        const year = parseInt(datePart[0] || '0', 10)
        const month = parseInt(datePart[1] || '1', 10) - 1 // 月份从 0 开始
        const day = parseInt(datePart[2] || '1', 10)
        const hours = parseInt(timePart[0] || '0', 10)
        const minutes = parseInt(timePart[1] || '0', 10)
        const seconds = timePart.length > 2 ? parseInt(timePart[2] || '0', 10) : 0
        // 使用本地时间创建 Date 对象
        date = new Date(year, month, day, hours, minutes, seconds)
      } else {
        date = new Date(timeStr)
      }
    } else {
      date = new Date(timeStr)
    }
  } else {
    date = new Date(timeStr)
  }
  
  if (isNaN(date.getTime())) {
    console.error('[parseDateTime] 无效的时间字符串:', timeStr)
    return Date.now()
  }
  
  return date.getTime()
}

// 格式化完整时间（显示到秒）
function formatFullTime(timeStr: string): string {
  if (!timeStr) return '-'
  
  const timestamp = parseDateTime(timeStr)
  const date = new Date(timestamp)
  
  if (isNaN(date.getTime())) {
    console.error('[formatFullTime] 无效的时间字符串:', timeStr)
    return '-'
  }
  
  // 格式：YYYY-MM-DD HH:mm:ss
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

// 格式化时间（用于消息显示，显示到秒）
function formatTime(timestamp: number): string {
  const date = new Date(timestamp)
  
  // 格式：YYYY-MM-DD HH:mm:ss
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

// 格式化相对时间（用于会话列表显示）
function formatRelativeTime(timeStr: string | Date): string {
  let date: Date
  if (timeStr instanceof Date) {
    date = timeStr
  } else {
    if (!timeStr) return '-'
    const timestamp = parseDateTime(timeStr)
    date = new Date(timestamp)
    if (isNaN(date.getTime())) {
      return '-'
    }
  }
  
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (seconds < 60) {
    return '刚刚'
  } else if (minutes < 60) {
    return `${minutes}分钟前`
  } else if (hours < 24) {
    return `${hours}小时前`
  } else if (days === 1) {
    return '昨天'
  } else if (days < 7) {
    return `${days}天前`
  } else if (days < 30) {
    const weeks = Math.floor(days / 7)
    return `${weeks}周前`
  } else if (days < 365) {
    const months = Math.floor(days / 30)
    return `${months}个月前`
  } else {
    return date.toLocaleDateString('zh-CN', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    })
  }
}

// 滚动到底部
function scrollToBottom() {
  if (messagesContainerRef.value) {
    messagesContainerRef.value.scrollTop = messagesContainerRef.value.scrollHeight
  }
}

// 暴露方法给父组件调用
defineExpose({
  handleAgentSelect
})
</script>

<style scoped>
.ai-chat-panel {
  display: flex;
  height: 100%;
  background: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color);
}

.session-sidebar {
  width: 280px;
  border-right: 1px solid var(--el-border-color);
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color-page);
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}

.sidebar-header h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.session-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* 会话卡片样式 */
.session-card {
  padding: 14px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  position: relative;
}

.session-card::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 0;
  background: var(--el-color-primary);
  transition: width 0.2s ease;
  border-radius: 8px 0 0 8px;
}

.session-card:hover {
  background: var(--el-fill-color-light);
  border-color: var(--el-border-color);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.session-card.active {
  background: var(--el-bg-color);
  border-color: var(--el-color-primary-light-7);
  box-shadow: none;
}

.session-card.active::before {
  width: 0;
}

.session-card.loading {
  opacity: 0.7;
}

.session-card.new-session-card {
  border-style: solid;
  border-width: 1px;
  border-color: var(--el-color-primary-light-7);
  background: var(--el-bg-color);
}

.session-card.new-session-card:hover {
  border-color: var(--el-color-primary);
  background: var(--el-fill-color-light);
}

.session-card.new-session-card .session-card-title {
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.session-card.new-session-card .session-card-agent {
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
}

.session-card.new-session-card .agent-name {
  color: var(--el-text-color-regular);
}

.session-card.new-session-card .session-card-time {
  color: var(--el-text-color-placeholder);
}

/* 会话卡片头部 */
.session-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.session-card-title-wrapper {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}

.session-card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.session-card.active .session-card-title {
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.new-icon {
  color: var(--el-color-primary);
  font-size: 16px;
}

.loading-icon {
  color: var(--el-color-primary);
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* 智能体信息 */
.session-card-agent {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  padding: 6px 8px;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
}

.session-card.active .session-card-agent {
  background: var(--el-fill-color-lighter);
  border: none;
}

.agent-logo-text {
  font-size: 12px;
  font-weight: bold;
  color: white;
}

.agent-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.session-card.active .agent-name {
  color: var(--el-text-color-regular);
  font-weight: 500;
}

/* 时间显示 */
.session-card-time {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  margin-top: 4px;
}

.session-card.active .session-card-time {
  color: var(--el-text-color-placeholder);
}

.empty-sessions {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
}

.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--el-border-color);
  background: var(--el-bg-color);
}

.header-left {
  flex: 1;
  min-width: 0;
}

.header-center {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: center;
}

.header-agent-info {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  background: var(--el-fill-color-lighter);
  border-radius: 20px;
  border: 1px solid var(--el-border-color-lighter);
}

.header-agent-avatar {
  flex-shrink: 0;
  border: 2px solid var(--el-color-primary-light-7);
}

.header-agent-logo-text {
  font-size: 14px;
  font-weight: bold;
  color: white;
}

.header-agent-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-left: 10px;
  flex: 1;
  min-width: 0;
}

.header-agent-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.header-agent-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.header-agent-description {
  font-size: 12px;
  color: var(--el-text-color-regular);
  line-height: 1.5;
  margin-top: 2px;
  /* 限制最多显示2行，超出部分用省略号 */
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 500px;
}

.header-agent-tag {
  flex-shrink: 0;
}

.header-agent-name-placeholder {
  font-size: 14px;
  color: var(--el-text-color-placeholder);
  font-style: italic;
}

.header-right {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  min-width: 0;
}

.close-button {
  padding: 0;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.message-item {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.message-item.user {
  flex-direction: row-reverse;
}

.message-avatar {
  flex-shrink: 0;
}

.message-content {
  flex: 1;
  min-width: 0;
}

.message-item.user .message-content {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.message-text {
  padding: 10px 14px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  word-wrap: break-word;
  line-height: 1.5;
  max-width: 100%;
}

.message-item.user .message-text {
  background: var(--el-color-primary);
  color: white;
}

.message-item.assistant .message-text {
  background: var(--el-fill-color-lighter);
}

/* Markdown 样式 */
.message-text :deep(code) {
  background: rgba(0, 0, 0, 0.1);
  padding: 2px 6px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
  font-size: 0.9em;
}

.message-item.user .message-text :deep(code) {
  background: rgba(255, 255, 255, 0.2);
}

.message-text :deep(pre) {
  background: rgba(0, 0, 0, 0.05);
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 8px 0;
  border: 1px solid var(--el-border-color);
}

.message-item.user .message-text :deep(pre) {
  background: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.2);
}

.message-text :deep(pre code) {
  background: transparent;
  padding: 0;
  border-radius: 0;
  font-size: 0.9em;
  line-height: 1.5;
}

.message-text :deep(h1),
.message-text :deep(h2),
.message-text :deep(h3),
.message-text :deep(h4),
.message-text :deep(h5),
.message-text :deep(h6) {
  margin: 16px 0 8px 0;
  font-weight: 600;
  line-height: 1.4;
}

.message-text :deep(h1) {
  font-size: 1.5em;
  border-bottom: 2px solid var(--el-border-color);
  padding-bottom: 8px;
}

.message-text :deep(h2) {
  font-size: 1.3em;
  border-bottom: 1px solid var(--el-border-color);
  padding-bottom: 6px;
}

.message-text :deep(h3) {
  font-size: 1.1em;
}

.message-text :deep(p) {
  margin: 8px 0;
  line-height: 1.6;
}

.message-text :deep(ul),
.message-text :deep(ol) {
  margin: 8px 0;
  padding-left: 24px;
}

.message-text :deep(li) {
  margin: 4px 0;
  line-height: 1.6;
}

.message-text :deep(blockquote) {
  margin: 8px 0;
  padding: 8px 16px;
  border-left: 4px solid var(--el-color-primary);
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
}

.message-item.user .message-text :deep(blockquote) {
  background: rgba(255, 255, 255, 0.1);
  border-left-color: rgba(255, 255, 255, 0.5);
}

.message-text :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
  font-size: 0.9em;
}

.message-text :deep(th),
.message-text :deep(td) {
  border: 1px solid var(--el-border-color);
  padding: 8px 12px;
  text-align: left;
}

.message-text :deep(th) {
  background: var(--el-fill-color-light);
  font-weight: 600;
}

.message-text :deep(a) {
  color: var(--el-color-primary);
  text-decoration: none;
}

.message-text :deep(a:hover) {
  text-decoration: underline;
}

.message-item.user .message-text :deep(a) {
  color: rgba(255, 255, 255, 0.9);
}

.message-text :deep(hr) {
  border: none;
  border-top: 1px solid var(--el-border-color);
  margin: 16px 0;
}

.message-text :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  margin: 8px 0;
}

/* 开场白样式 */
.message-text.is-greeting {
  max-width: 600px; /* 限制开场白宽度 */
}

.message-text.is-greeting.is-collapsed {
  max-height: 200px; /* 默认最大高度 */
  overflow: hidden;
  position: relative;
}

/* 如果开场白内容很短，不需要限制高度 */
.message-text.is-greeting:not(.needs-expand).is-collapsed {
  max-height: none;
  overflow: visible;
}

.message-text.is-greeting.is-collapsed::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 60px;
  background: linear-gradient(to bottom, transparent, var(--el-fill-color-light));
  pointer-events: none;
}

.greeting-expand {
  margin-top: 8px;
  text-align: center;
}

.message-files {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
  font-size: 12px;
}

.message-item.user .file-item {
  background: rgba(255, 255, 255, 0.2);
  color: white;
}

.file-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.message-time {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  opacity: 0.8;
}

.chat-input {
  padding: 16px;
  border-top: 1px solid var(--el-border-color);
  position: relative;
  transition: background-color 0.2s;
}

.chat-input.drag-over {
  background-color: var(--el-color-primary-light-9);
  border: 2px dashed var(--el-color-primary);
}

.uploaded-files {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
}

.file-tag {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
  font-size: 12px;
}

.input-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 8px;
}
</style>
