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
          class="session-item new-session-item"
        >
          <div class="session-content">
            <div class="session-title">
              <el-icon><Plus /></el-icon>
              <span>新会话</span>
            </div>
            <div class="session-meta">
              <div class="session-times">
                <div class="session-time-item">
                  <span class="time-label">创建:</span>
                  <span class="time-value">{{ formatFullTime(new Date().toISOString()) }}</span>
                </div>
                <div class="session-time-item">
                  <span class="time-label">更新:</span>
                  <span class="time-value">{{ formatFullTime(new Date().toISOString()) }}</span>
                </div>
              </div>
              <div v-if="currentAgent" class="session-agent-info">
                <el-avatar
                  :size="16"
                  :src="getAgentLogo(currentAgent)"
                  class="session-agent-mini-logo"
                >
                  <span class="agent-logo-text-mini">{{ getAgentLogoText(currentAgent) }}</span>
                </el-avatar>
                <span class="session-agent-name">{{ currentAgent.name }}</span>
              </div>
            </div>
          </div>
        </div>
        
        <div
          v-for="session in sessionList"
          :key="session.session_id"
          :class="['session-item', { active: session.session_id === sessionId }]"
          @click="handleSelectSession(session.session_id)"
        >
          <div class="session-content">
            <div class="session-title">
              {{ session.title || '未命名会话' }}
            </div>
            <div class="session-meta">
              <div class="session-times">
                <div class="session-time-item">
                  <span class="time-label">创建:</span>
                  <span class="time-value">{{ formatFullTime(session.created_at) }}</span>
                </div>
                <div class="session-time-item">
                  <span class="time-label">更新:</span>
                  <span class="time-value">{{ formatFullTime(session.updated_at) }}</span>
                </div>
              </div>
              <div v-if="session.agent" class="session-agent-info">
                <el-avatar
                  :size="16"
                  :src="getAgentLogo(session.agent)"
                  class="session-agent-mini-logo"
                >
                  <span class="agent-logo-text-mini">{{ getAgentLogoText(session.agent) }}</span>
                </el-avatar>
                <span class="session-agent-name">{{ session.agent.name }}</span>
              </div>
            </div>
          </div>
        </div>
        <div v-if="sessionList.length === 0 && !loadingSessions && sessionId" class="empty-sessions">
          暂无会话，点击"新建"创建会话
        </div>
      </div>
    </div>

    <!-- 主聊天区域 -->
    <div class="chat-main">
      <div class="chat-header">
        <h3>AI 助手</h3>
        <div class="header-actions">
          <el-select
            v-model="selectedAgentId"
            placeholder="选择智能体"
            filterable
            :loading="agentLoading"
            style="width: 200px; margin-right: 8px;"
            @change="handleAgentChange"
          >
            <el-option
              v-for="agent in agentOptions"
              :key="agent.id"
              :label="agent.name"
              :value="agent.id"
            >
              <div style="display: flex; justify-content: space-between; align-items: center;">
                <span>{{ agent.name }}</span>
                <el-tag size="small" :type="agent.agent_type === 'plugin' ? 'warning' : 'success'" style="margin-left: 8px;">
                  {{ agent.agent_type === 'plugin' ? '插件' : '知识库' }}
                </el-tag>
              </div>
            </el-option>
          </el-select>
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
          <div class="message-text" v-html="formatMessage(message.content)"></div>
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
        :disabled="loading"
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
          :disabled="loading"
          accept="*"
          multiple
        >
          <el-button :icon="Upload" :disabled="loading">上传文件</el-button>
        </el-upload>
        <el-button
          type="primary"
          :loading="loading"
          :disabled="!inputMessage.trim() && uploadedFiles.length === 0"
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
      :tree-id="treeId"
      :package="package"
      :current-node-name="currentNodeName"
      @confirm="handleAgentSelect"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { Close, User, Loading, ChatRound, Upload, Document, Plus } from '@element-plus/icons-vue'
import * as agentApi from '@/api/agent'
import type { AgentInfo, ChatSessionInfo } from '@/api/agent'
import { uploadFile, notifyUploadComplete } from '@/utils/upload'
import type { UploadFile } from 'element-plus'
import { marked } from 'marked'
import AgentSelectDialog from '@/components/Agent/AgentSelectDialog.vue'

interface Props {
  agentId: number | null
  treeId: number | null // 服务目录ID（TreeID）
  package?: string // Package 名称
  currentNodeName?: string
  existingFiles?: string[] // 当前 package 下已存在的文件名（不含 .go 后缀）
}

const props = withDefaults(defineProps<Props>(), {
  agentId: null,
  treeId: null,
  package: '',
  currentNodeName: '',
  existingFiles: () => []
})

const emit = defineEmits<{
  close: []
}>()

interface ChatFile {
  url: string
  remark: string
}

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  files?: ChatFile[]
  timestamp: number
}

const messages = ref<ChatMessage[]>([])
const inputMessage = ref('')
const loading = ref(false)
const messagesContainerRef = ref<HTMLElement>()
const inputRef = ref<InstanceType<typeof HTMLTextAreaElement>>()

// 文件上传相关
const uploadedFiles = ref<ChatFile[]>([])
const isDragOver = ref(false)

// 智能体选择相关
const selectedAgentId = ref<number | null>(props.agentId)
const agentOptions = ref<AgentInfo[]>([])
const agentLoading = ref(false)

// 当前选中的智能体信息
const currentAgent = computed(() => {
  if (!selectedAgentId.value) return null
  return agentOptions.value.find(agent => agent.id === selectedAgentId.value) || null
})

// 会话ID（首次为空，后端自动生成）
const sessionId = ref<string>('')
const loadingSession = ref(false)
// 正在加载的会话ID（用于防止竞态条件）
const pendingSessionId = ref<string | null>(null)
// 防抖定时器（用于防止过于频繁的切换）
let switchDebounceTimer: ReturnType<typeof setTimeout> | null = null

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
  if (!props.treeId) {
    sessionList.value = []
    return
  }

  loadingSessions.value = true
  try {
    const res = await agentApi.getChatSessionList({
      tree_id: props.treeId,
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
async function loadSessionMessages(targetSessionId: string) {
  // 检查是否已经被其他请求覆盖（通过 pendingSessionId 判断）
  // 注意：这里只检查 pendingSessionId，不检查 sessionId，因为 sessionId 在 handleSelectSession 中已经被立即设置了
  if (pendingSessionId.value !== targetSessionId) {
    console.log('[AIChatPanel] 加载请求已被新的请求覆盖，放弃加载:', targetSessionId)
    return
  }
  
  try {
    const messageRes = await agentApi.getChatMessageList({
      session_id: targetSessionId
    })

    // 再次检查是否仍然是要加载的会话（只检查 pendingSessionId，因为这是唯一能判断请求是否被覆盖的标识）
    // 注意：不检查 sessionId，因为 sessionId 在 handleSelectSession 中已经被立即设置为最新的会话ID
    if (pendingSessionId.value !== targetSessionId) {
      console.log('[AIChatPanel] 加载消息过程中会话已切换，放弃加载结果:', targetSessionId)
      return
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
          timestamp: parseDateTime(msg.created_at)
        }
      })

      // 滚动到底部
      nextTick(() => {
        scrollToBottom()
      })
    } else {
      // 如果没有消息，显示欢迎消息（但保持 sessionId）
      messages.value = []
      if (props.currentNodeName) {
        addMessage('assistant', `你好！我是 AI 助手，可以帮助你处理「${props.currentNodeName}」相关的问题。有什么可以帮助你的吗？`)
      } else {
        addMessage('assistant', '你好！我是 AI 助手，有什么可以帮助你的吗？')
      }
    }
  } catch (error: any) {
    console.error('[AIChatPanel] 加载会话消息失败:', error)
    // 检查是否仍然是要加载的会话（只检查 pendingSessionId）
    if (pendingSessionId.value !== targetSessionId) {
      return
    }
    ElMessage.error(error.message || '加载会话消息失败')
    // 加载失败时显示欢迎消息（但保持 sessionId）
    messages.value = []
    if (props.currentNodeName) {
      addMessage('assistant', `你好！我是 AI 助手，可以帮助你处理「${props.currentNodeName}」相关的问题。有什么可以帮助你的吗？`)
    } else {
      addMessage('assistant', '你好！我是 AI 助手，有什么可以帮助你的吗？')
    }
  }
}

// 从后端加载会话和消息
async function loadSessionFromBackend() {
  if (!props.treeId) {
    // 如果没有 treeId，显示欢迎消息
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

  // 如果有会话列表，加载最新的会话
  if (sessionList.value.length > 0) {
    const latestSession = sessionList.value[0]
    sessionId.value = latestSession.session_id
    await loadSessionMessages(latestSession.session_id)
  } else {
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
  console.log('[AIChatPanel] 新建会话被点击, treeId:', props.treeId)
  
  if (!props.treeId) {
    ElMessage.warning('缺少服务目录ID，无法创建会话')
    return
  }
  
  // 弹出智能体选择对话框
  agentSelectDialogVisible.value = true
}

// 处理智能体选择（从外部选择智能体时调用）
function handleAgentSelect(agent: AgentInfo) {
  console.log('[AIChatPanel] 选择智能体:', agent)
  
  // 设置选中的智能体
  selectedAgentId.value = agent.id
  
  // 清空当前会话ID，表示新建会话
  sessionId.value = ''
  messages.value = []
  uploadedFiles.value = []
  
  // 刷新会话列表（确保显示最新的会话）
  loadSessionList()
  
  // 显示欢迎消息
  if (props.currentNodeName) {
    addMessage('assistant', `你好！我是 ${agent.name}，可以帮助你处理「${props.currentNodeName}」相关的问题。有什么可以帮助你的吗？`)
  } else {
    addMessage('assistant', `你好！我是 ${agent.name}，有什么可以帮助你的吗？`)
  }
  
  // 滚动到底部
  nextTick(() => {
    scrollToBottom()
  })
  
  ElMessage.success('已创建新会话，发送第一条消息后将自动保存')
}

// 选择会话
async function handleSelectSession(targetSessionId: string) {
  // 如果点击的是当前会话，直接返回（不重新加载）
  if (targetSessionId === sessionId.value && !loadingSession.value) {
    console.log('[AIChatPanel] 已经是当前会话，无需切换')
    return
  }
  
  // 清除之前的防抖定时器（如果有）
  if (switchDebounceTimer) {
    clearTimeout(switchDebounceTimer)
    switchDebounceTimer = null
  }
  
  console.log('[AIChatPanel] 切换会话:', targetSessionId, '当前会话:', sessionId.value)
  
  // 立即更新 UI 状态（不等待防抖）
  // 查找会话信息，设置对应的智能体
  const session = sessionList.value.find(s => s.session_id === targetSessionId)
  if (session && session.agent_id) {
    selectedAgentId.value = session.agent_id
  }
  
  // 先设置会话ID（立即更新，确保UI状态正确）
  sessionId.value = targetSessionId
  // 清空当前消息，准备加载新会话的消息
  messages.value = []
  uploadedFiles.value = []
  
  // 使用防抖：如果用户在短时间内多次点击，只执行最后一次加载
  // 但是 UI 状态（sessionId、messages）会立即更新，确保用户体验流畅
  switchDebounceTimer = setTimeout(async () => {
    const currentTargetSessionId = targetSessionId
    switchDebounceTimer = null
    
    // 检查是否仍然是要加载的会话（防止在防抖期间被新的点击覆盖）
    if (sessionId.value !== currentTargetSessionId) {
      console.log('[AIChatPanel] 防抖期间会话已切换，放弃加载:', currentTargetSessionId)
      return
    }
    
    // 设置加载状态和待加载的会话ID（防止并发请求）
    loadingSession.value = true
    pendingSessionId.value = currentTargetSessionId
    
    // 加载会话消息
    try {
      await loadSessionMessages(currentTargetSessionId)
    } catch (error) {
      console.error('[AIChatPanel] 加载会话消息失败:', currentTargetSessionId, error)
      // 加载失败时，保持当前会话ID不变
    } finally {
      // 只有当前待加载的会话ID仍然是 currentTargetSessionId 时，才清除加载状态
      // 这样可以防止旧的请求覆盖新的状态
      if (pendingSessionId.value === currentTargetSessionId) {
        loadingSession.value = false
        pendingSessionId.value = null
      }
    }
  }, 150) // 150ms 防抖，如果用户在 150ms 内多次点击，只执行最后一次加载
}

// 智能体变化处理
async function handleAgentChange() {
  messages.value = []
  sessionId.value = '' // 切换智能体时重置会话ID
  uploadedFiles.value = []
  // 从后端加载新智能体的会话记录
  await loadSessionFromBackend()
}

// 初始化欢迎消息
onMounted(async () => {
  await loadAgents()
  
  // 从后端加载会话记录（如果有）
  await loadSessionFromBackend()
})

// 监听目录切换，恢复会话记录
watch(
  () => [props.treeId, props.package, props.currentNodeName, selectedAgentId.value],
  async ([newTreeId, newPackage, newNodeName, newAgentId], [oldTreeId, oldPackage, oldNodeName, oldAgentId]) => {
    // 如果 treeId、package 或 agentId 变化，说明切换了目录或智能体
    if (newTreeId !== oldTreeId || newPackage !== oldPackage || newAgentId !== oldAgentId) {
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

// 🔥 监听 agentId prop 变化，更新选中的智能体
watch(
  () => props.agentId,
  (newAgentId) => {
    if (newAgentId && newAgentId !== selectedAgentId.value) {
      selectedAgentId.value = newAgentId
      // 切换智能体时重置会话
      handleAgentChange()
    }
  }
)

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
function addMessage(role: 'user' | 'assistant', content: string, files?: ChatFile[]) {
  messages.value.push({
    role,
    content,
    files,
    timestamp: Date.now()
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
      const downloadUrl = await notifyUploadComplete({
        key: uploadResult.fileInfo.key,
        success: true,
        router: uploadResult.fileInfo.router,
        file_name: uploadResult.fileInfo.file_name,
        file_size: uploadResult.fileInfo.file_size,
        content_type: uploadResult.fileInfo.content_type,
        hash: uploadResult.fileInfo.hash,
      })
      
      if (downloadUrl) {
        uploadedFiles.value.push({
          url: downloadUrl,
          remark: rawFile.name
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

  if (!props.treeId) {
    ElMessage.warning('缺少服务目录ID')
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
      tree_id: props.treeId,
      package: props.package || '', // 传递 Package 名称
      session_id: sessionId.value || '', // 首次为空，后端自动生成
      existing_files: props.existingFiles || [], // 传递已存在的文件名
      message: {
        content: userMessage || '',
        files: files.map(f => ({
          url: f.url,
          remark: f.remark
        }))
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

// 解析时间字符串（支持多种格式：DateTime、RFC3339等）
function parseDateTime(timeStr: string): number {
  if (!timeStr) return Date.now()
  
  // 尝试解析多种格式
  // 格式1: "2006-01-02 15:04:05" (time.DateTime)
  // 格式2: "2006-01-02T15:04:05Z" (RFC3339 UTC)
  // 格式3: "2006-01-02T15:04:05+08:00" (RFC3339 with timezone)
  
  let date: Date
  
  // 如果包含 T 和 Z，是 RFC3339 格式
  if (timeStr.includes('T') && (timeStr.includes('Z') || timeStr.match(/[+-]\d{2}:\d{2}$/))) {
    date = new Date(timeStr)
  } else if (timeStr.includes(' ')) {
    // 如果是 "2006-01-02 15:04:05" 格式，需要转换为 ISO 格式
    // 将空格替换为 T，并添加 Z（假设是 UTC，或者使用本地时区）
    // 注意：如果后端返回的是本地时间，这里可能需要调整
    const isoStr = timeStr.replace(' ', 'T') + 'Z'
    date = new Date(isoStr)
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

// 格式化会话时间
function formatSessionTime(timeStr: string): string {
  if (!timeStr) return '-'
  
  // 解析时间字符串（支持多种格式）
  const timestamp = parseDateTime(timeStr)
  const date = new Date(timestamp)
  
  // 检查日期是否有效
  if (isNaN(date.getTime())) {
    console.error('[formatSessionTime] 无效的时间字符串:', timeStr)
    return '-'
  }
  
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (days === 0) {
    // 今天：显示时间
    return date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit'
    })
  } else if (days === 1) {
    return '昨天'
  } else if (days < 7 && days > 0) {
    return `${days}天前`
  } else {
    // 超过7天或负数（未来时间）：显示日期
    return date.toLocaleDateString('zh-CN', {
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
  width: 240px;
  border-right: 1px solid var(--el-border-color);
  display: flex;
  flex-direction: column;
  background: var(--el-fill-color-lighter);
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color);
}

.sidebar-header h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}

.session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.session-item {
  padding: 12px;
  margin-bottom: 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: background-color 0.2s;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
}

.session-item:hover {
  background: var(--el-fill-color-light);
}

.session-item.active {
  background: var(--el-fill-color-lighter);
  border-color: var(--el-color-primary);
  border-left-width: 3px;
  border-left-color: var(--el-color-primary);
  
  .session-title {
    color: var(--el-text-color-primary);
    font-weight: 600;
  }
  
  .session-meta {
    color: var(--el-text-color-regular);
    
    .session-agent-info {
      background: var(--el-color-primary-light-8);
      border-color: var(--el-color-primary-light-6);
    }
    
    .session-agent-name {
      color: var(--el-color-primary);
      font-weight: 600;
    }
    
    /* .session-time 已移除，使用 .session-times 替代 */
  }
}

.session-item.new-session-item {
  background: var(--el-bg-color);
  border-color: var(--el-color-primary);
  border-style: dashed;
  border-width: 2px;
  
  .session-title {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--el-text-color-primary);
    font-weight: 600;
  }
  
  .session-meta {
    /* .session-time 已移除，使用 .session-times 替代 */
  }
}

.session-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.session-agent-logo {
  flex-shrink: 0;
  border: 2px solid var(--el-border-color-lighter);
  
  .agent-logo-text {
    font-size: 14px;
    font-weight: bold;
    color: white;
  }
}

.session-content {
  width: 100%;
  min-width: 0;
}

.session-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
  width: 100%;
}

.session-times {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 100%;
}

.session-time-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  line-height: 1.4;
}

.time-label {
  color: var(--el-text-color-placeholder);
  font-weight: 500;
  flex-shrink: 0;
}

.time-value {
  color: var(--el-text-color-secondary);
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 11px;
}

.session-agent-info {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  background: var(--el-color-primary-light-9);
  border-radius: 4px;
  border: 1px solid var(--el-color-primary-light-7);
  flex-shrink: 0;
  margin-top: 4px;
}

.session-agent-mini-logo {
  flex-shrink: 0;
  border: 1px solid var(--el-border-color-lighter);
  
  .agent-logo-text-mini {
    font-size: 10px;
    font-weight: bold;
    color: white;
  }
}

.session-agent-name {
  color: var(--el-color-primary);
  font-weight: 600;
  white-space: nowrap;
  font-size: 11px;
}

.session-time {
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  font-weight: 500;
  align-self: flex-end;
  flex-shrink: 0;
}

.empty-sessions {
  padding: 20px;
  text-align: center;
  color: var(--el-text-color-secondary);
  font-size: 12px;
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
  padding: 16px;
  border-bottom: 1px solid var(--el-border-color);
}

.chat-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
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
