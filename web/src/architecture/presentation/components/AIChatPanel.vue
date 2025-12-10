<template>
  <div class="ai-chat-panel">
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
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Close, User, Loading, ChatRound, Upload, Document } from '@element-plus/icons-vue'
import * as agentApi from '@/api/agent'
import type { AgentInfo } from '@/api/agent'
import { uploadFile, notifyUploadComplete } from '@/utils/upload'
import type { UploadFile } from 'element-plus'

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

// 会话ID（首次为空，后端自动生成）
const sessionId = ref<string>('')

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

// 智能体变化处理
function handleAgentChange() {
  messages.value = []
  sessionId.value = '' // 切换智能体时重置会话ID
  uploadedFiles.value = []
  if (props.currentNodeName) {
    addMessage('assistant', `你好！我是 AI 助手，可以帮助你处理「${props.currentNodeName}」相关的问题。有什么可以帮助你的吗？`)
  } else {
    addMessage('assistant', '你好！我是 AI 助手，有什么可以帮助你的吗？')
  }
}

// 初始化欢迎消息
onMounted(async () => {
  await loadAgents()
  
  if (props.currentNodeName) {
    addMessage('assistant', `你好！我是 AI 助手，可以帮助你处理「${props.currentNodeName}」相关的问题。有什么可以帮助你的吗？`)
  } else {
    addMessage('assistant', '你好！我是 AI 助手，有什么可以帮助你的吗？')
  }
})

// 🔥 监听目录切换，重置会话状态
watch(
  () => [props.treeId, props.package, props.currentNodeName],
  ([newTreeId, newPackage, newNodeName], [oldTreeId, oldPackage, oldNodeName]) => {
    // 如果 treeId 或 package 变化，说明切换了目录，需要重置会话
    if (newTreeId !== oldTreeId || newPackage !== oldPackage) {
      // 清空消息
      messages.value = []
      // 重置会话ID
      sessionId.value = ''
      // 清空上传的文件
      uploadedFiles.value = []
      
      // 更新欢迎消息以反映新的目录名称
      if (newNodeName) {
        addMessage('assistant', `你好！我是 AI 助手，可以帮助你处理「${newNodeName}」相关的问题。有什么可以帮助你的吗？`)
      } else {
        addMessage('assistant', '你好！我是 AI 助手，有什么可以帮助你的吗？')
      }
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
    }

    // 添加 AI 回复
    addMessage('assistant', res.content || '抱歉，AI 没有返回内容')
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

// 格式化消息内容（支持 Markdown）
function formatMessage(content: string): string {
  return content
    .replace(/\n/g, '<br>')
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.*?)\*/g, '<em>$1</em>')
    .replace(/`(.*?)`/g, '<code>$1</code>')
}

// 格式化时间
function formatTime(timestamp: number): string {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)

  if (seconds < 60) {
    return '刚刚'
  } else if (minutes < 60) {
    return `${minutes}分钟前`
  } else if (hours < 24) {
    return `${hours}小时前`
  } else {
    return date.toLocaleString('zh-CN', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }
}

// 滚动到底部
function scrollToBottom() {
  if (messagesContainerRef.value) {
    messagesContainerRef.value.scrollTop = messagesContainerRef.value.scrollHeight
  }
}
</script>

<style scoped>
.ai-chat-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--el-bg-color);
  border-left: 1px solid var(--el-border-color);
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

.message-text :deep(code) {
  background: rgba(0, 0, 0, 0.1);
  padding: 2px 4px;
  border-radius: 3px;
  font-family: 'Courier New', monospace;
}

.message-item.user .message-text :deep(code) {
  background: rgba(255, 255, 255, 0.2);
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
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
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
