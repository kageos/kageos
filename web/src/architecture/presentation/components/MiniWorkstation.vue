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
        <span class="mini-ws-dir-name" :title="fullCodePath">{{ dirName || displayPath }}</span>
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
                    <div v-for="(f, i) in uploadedFiles" :key="'u' + i" class="mini-file-card" @click="previewFile(f)">
                      <div v-if="isImageFile(f)" class="mini-file-thumb">
                        <img :src="f.url" :alt="f.name" loading="lazy" />
                      </div>
                      <div v-else class="mini-file-icon">
                        <el-icon :size="18"><DocumentIcon /></el-icon>
                        <span v-if="fileExt(f)" class="mini-file-ext">{{ fileExt(f) }}</span>
                      </div>
                      <div class="mini-file-info">
                        <span class="mini-file-name" :title="f.name">{{ f.name }}</span>
                        <div class="mini-file-actions">
                          <el-button link size="small" type="primary" @click.stop="previewFile(f)">预览</el-button>
                          <el-button link size="small" type="primary" @click.stop="downloadFile(f)">下载</el-button>
                        </div>
                      </div>
                    </div>
                  </template>
                  <template v-if="outputFiles.length > 0">
                    <div class="mini-file-section-title">
                      <el-icon :size="12"><FolderOpened /></el-icon>
                      输出文件 ({{ outputFiles.length }})
                    </div>
                    <div v-for="(f, i) in outputFiles" :key="'o' + i" class="mini-file-card" @click="previewFile(f)">
                      <div v-if="isImageFile(f)" class="mini-file-thumb">
                        <img :src="f.url" :alt="f.name" loading="lazy" />
                      </div>
                      <div v-else class="mini-file-icon">
                        <el-icon :size="18"><DocumentIcon /></el-icon>
                        <span v-if="fileExt(f)" class="mini-file-ext">{{ fileExt(f) }}</span>
                      </div>
                      <div class="mini-file-info">
                        <span class="mini-file-name" :title="f.name">{{ f.name }}</span>
                        <div class="mini-file-actions">
                          <el-button link size="small" type="primary" @click.stop="previewFile(f)">预览</el-button>
                          <el-button link size="small" type="primary" @click.stop="downloadFile(f)">下载</el-button>
                        </div>
                      </div>
                    </div>
                  </template>
                  <template v-if="allPanelDisplayFields.length > 0">
                    <div class="mini-file-section-title">
                      <el-icon :size="12"><Memo /></el-icon>
                      输出数据 ({{ allPanelDisplayFields.length }})
                    </div>
                    <div v-for="(df, i) in allPanelDisplayFields" :key="'df' + i" class="mini-display-field-card">
                      <div class="mini-df-header">
                        <span class="mini-df-label">{{ df.label }}</span>
                        <div class="mini-df-actions">
                          <el-button link size="small" type="primary" @click.stop="openDfPreview(df)">
                            <el-icon :size="11"><View /></el-icon> 预览
                          </el-button>
                          <el-button link size="small" type="primary" @click.stop="copyDisplayFieldValue(df)">
                            <el-icon :size="11"><CopyDocument /></el-icon> 复制
                          </el-button>
                        </div>
                      </div>
                      <div class="mini-df-value">{{ df.value.length > 150 ? df.value.slice(0, 150) + '…' : df.value }}</div>
                    </div>
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
      <div v-if="maximized" class="mini-session-sidebar">
        <div class="mini-session-header">
          <span class="mini-session-title">会话列表</span>
          <el-button text :icon="Plus" size="small" @click="handleNewSession" title="新建会话" />
        </div>
        <div class="mini-session-list" v-loading="loadingSessions">
          <div
            :class="['mini-session-card', 'mini-session-new', { active: !sessionId }]"
            @click="handleNewSession"
          >
            <el-icon class="mini-session-new-icon"><Plus /></el-icon>
            <span>新建会话</span>
          </div>
          <div
            v-for="s in miniSessionList"
            :key="s.session_id"
            :class="['mini-session-card', { active: s.session_id === sessionId }, { generating: s.status === 'generating' }]"
            @click="handleSelectSession(s.session_id)"
          >
            <div class="mini-session-card-head">
              <el-icon v-if="s.status === 'generating'" class="is-loading" :size="12" color="var(--el-color-primary)"><Loading /></el-icon>
              <span class="mini-session-card-title">{{ s.title || '未命名会话' }}</span>
            </div>
            <div v-if="s.user" class="mini-session-card-user">
              <UserDisplay :username="s.user" mode="simple" size="small" />
            </div>
            <div class="mini-session-card-time">
              <span v-if="s.status === 'generating'" class="mini-session-status">执行中</span>
              <span>{{ formatRelativeTime(s.updated_at) }}</span>
            </div>
          </div>
          <div v-if="miniSessionList.length === 0 && !loadingSessions" class="mini-session-empty">
            <span>暂无会话</span>
          </div>
        </div>
      </div>

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
            <div v-for="(f, i) in uploadedFiles" :key="'u' + i" class="mini-file-card">
              <div v-if="isImageFile(f)" class="mini-file-thumb" @click="previewFile(f)">
                <img :src="f.url" :alt="f.name" loading="lazy" />
              </div>
              <div v-else class="mini-file-icon" @click="previewFile(f)">
                <el-icon :size="20"><DocumentIcon /></el-icon>
                <span v-if="fileExt(f)" class="mini-file-ext">{{ fileExt(f) }}</span>
              </div>
              <div class="mini-file-info">
                <span class="mini-file-name" :title="f.name">{{ f.name }}</span>
                <div class="mini-file-actions">
                  <el-button link size="small" type="primary" @click="previewFile(f)"><el-icon :size="12"><View /></el-icon> 预览</el-button>
                  <el-button link size="small" type="primary" @click="downloadFile(f)"><el-icon :size="12"><Download /></el-icon> 下载</el-button>
                </div>
              </div>
            </div>
          </template>

          <!-- 输出文件 -->
          <template v-if="outputFiles.length > 0">
            <div class="mini-file-section-title">
              <el-icon :size="13"><FolderOpened /></el-icon>
              输出文件 ({{ outputFiles.length }})
            </div>
            <div v-for="(f, i) in outputFiles" :key="'o' + i" class="mini-file-card">
              <div v-if="isImageFile(f)" class="mini-file-thumb" @click="previewFile(f)">
                <img :src="f.url" :alt="f.name" loading="lazy" />
              </div>
              <div v-else class="mini-file-icon" @click="previewFile(f)">
                <el-icon :size="20"><DocumentIcon /></el-icon>
                <span v-if="fileExt(f)" class="mini-file-ext">{{ fileExt(f) }}</span>
              </div>
              <div class="mini-file-info">
                <span class="mini-file-name" :title="f.name">{{ f.name }}</span>
                <div class="mini-file-actions">
                  <el-button link size="small" type="primary" @click="previewFile(f)"><el-icon :size="12"><View /></el-icon> 预览</el-button>
                  <el-button link size="small" type="primary" @click="downloadFile(f)"><el-icon :size="12"><Download /></el-icon> 下载</el-button>
                </div>
              </div>
            </div>
          </template>

          <!-- 输出数据 -->
          <template v-if="allPanelDisplayFields.length > 0">
            <div class="mini-file-section-title">
              <el-icon :size="13"><Memo /></el-icon>
              输出数据 ({{ allPanelDisplayFields.length }})
            </div>
            <div v-for="(df, i) in allPanelDisplayFields" :key="'sdf' + i" class="mini-sidebar-df-card">
              <div class="mini-sidebar-df-header">
                <span class="mini-sidebar-df-label">{{ df.label }}</span>
                <div class="mini-sidebar-df-actions">
                  <el-button link size="small" type="primary" @click="openDfPreview(df)">
                    <el-icon :size="12"><View /></el-icon> 预览
                  </el-button>
                  <el-button link size="small" type="primary" @click="copyDisplayFieldValue(df)">
                    <el-icon :size="12"><CopyDocument /></el-icon> 复制
                  </el-button>
                </div>
              </div>
              <div class="mini-sidebar-df-value">
                <pre class="mini-sidebar-df-pre">{{ df.value }}</pre>
              </div>
            </div>
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

  <!-- 输出数据预览弹窗：自定义实现，Teleport 到 body + z-index 99999，彻底避免被遮挡和事件冒泡干扰 -->
  <Teleport to="body">
    <transition name="df-preview-fade">
      <div
        v-if="dfPreviewVisible"
        class="df-preview-overlay"
        @click.self="closeDfPreview"
        @mousedown.stop
        @mouseup.stop
        @pointerdown.stop
        @pointerup.stop
      >
        <div class="df-preview-modal" @click.stop @mousedown.stop @mouseup.stop @pointerdown.stop @pointerup.stop>
          <div class="df-preview-header">
            <span class="df-preview-title">{{ dfPreviewLabel }}</span>
            <button class="df-preview-close" @click="closeDfPreview" title="关闭">
              <el-icon :size="16"><Close /></el-icon>
            </button>
          </div>
          <div class="df-preview-body">
            <textarea
              v-model="dfPreviewContent"
              class="df-preview-textarea"
              spellcheck="false"
            />
          </div>
          <div class="df-preview-footer">
            <span class="df-preview-stats">{{ dfPreviewContent.length }} 字符 · {{ dfPreviewContent.split('\n').length }} 行</span>
            <div class="df-preview-actions">
              <button class="df-preview-btn" @click="closeDfPreview">关闭</button>
              <button class="df-preview-btn df-preview-btn--primary" @click="copyDfPreviewContent">
                <el-icon :size="14"><CopyDocument /></el-icon>
                复制全部
              </button>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onUnmounted, computed, toRaw } from 'vue'
import { Loading, Close, Minus, FullScreen, CopyDocument, Paperclip, CircleCheck, CircleClose, FolderOpened, UploadFilled, Plus, VideoPause, Download, View, Document as DocumentIcon, Memo } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import { workspaceChatStream, getWorkspaceMessages, getWorkspaceSessions, cancelWorkspaceChat, type WorkspaceChatReq, type WorkspaceChatMessageFile, type WorkspaceSessionItem } from '@/api/workspace'
import { useWorkspaceChatStream } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import type { ChatMessage } from '@/architecture/presentation/composables/useWorkspaceChatStream'
import { uploadFile, notifyUploadComplete } from '@/utils/upload'
import type { UploadProgress } from '@/utils/upload/types'
import OutputFilesDisplay from './OutputFilesDisplay.vue'
import MessageToolCalls from './MessageToolCalls.vue'
import UserDisplay from '../widgets/UserDisplay.vue'
import { extractFileGroupsFromResult, type OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'
import { extractAllDisplayFields, type OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'
import OutputDisplayFields from './OutputDisplayFields.vue'
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
  initialPosition?: 'center'
  initialMaximized?: boolean
}>()

const emit = defineEmits<{
  (e: 'minimize'): void
  (e: 'close'): void
  (e: 'task-started', sessionId: string): void
  (e: 'tool-call-ok', payload: { name: string }): void
  (e: 'maximize-change', payload: { maximized: boolean; sessionId?: string }): void
}>()

const { messages, sending, sessionId, send: sendMessage, handleEvent, setMessages } = useWorkspaceChatStream()

const inputText = ref('')
const inputRef = ref<HTMLTextAreaElement>()
const outputRef = ref<HTMLElement>()
const rootRef = ref<HTMLElement>()
const attachedFiles = ref<WorkspaceChatMessageFile[]>([])
const uploading = ref(false)

const UPLOAD_ROUTER = 'workspace/chat'

// ─── 会话列表（最大化时使用） ───
const miniSessionList = ref<WorkspaceSessionItem[]>([])
const loadingSessions = ref(false)

async function loadMiniSessions() {
  if (!props.fullCodePath) { miniSessionList.value = []; return }
  loadingSessions.value = true
  try {
    const res = await getWorkspaceSessions({ full_code_path: props.fullCodePath })
    miniSessionList.value = res.sessions || []
  } catch {
    miniSessionList.value = []
  } finally {
    loadingSessions.value = false
  }
}

function handleNewSession() {
  stopMiniPoll()
  stopMiniStreamListening()
  sending.value = false
  sessionId.value = undefined
  setMessages([])
}

const stopping = ref(false)
async function handleStopSession() {
  if (!sessionId.value || stopping.value) return
  stopping.value = true
  try {
    await cancelWorkspaceChat(sessionId.value)
    sending.value = false
    stopMiniPoll()
    stopMiniStreamListening()
    ElMessage.success('已停止')
    if (maximized.value) loadMiniSessions()
  } catch (e: any) {
    ElMessage.error(e?.message || '停止失败')
  } finally {
    stopping.value = false
  }
}

async function handleSelectSession(targetSid: string) {
  if (targetSid === sessionId.value) return
  stopMiniPoll()
  stopMiniStreamListening()
  sending.value = false
  sessionId.value = targetSid
  setMessages([])
  if (maximized.value) emit('maximize-change', { maximized: true, sessionId: targetSid })
  await loadMiniSessionMessages(targetSid)
  if (sessionId.value !== targetSid) return
  const found = miniSessionList.value.find(s => s.session_id === targetSid)
  if (found?.status === 'generating') {
    startMiniStreamListening(targetSid)
    startMiniPoll(targetSid)
  }
}

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

const displayPath = computed(() => {
  if (!props.fullCodePath) return '未选择目录'
  const parts = props.fullCodePath.split('/').filter(Boolean)
  return parts[parts.length - 1] || props.fullCodePath
})

// ─── 最大化 / 还原 ───
const maximized = ref(!!props.initialMaximized)
const preMaxRect = ref<{ x: number; y: number; w: number; h: number } | null>(null)

// 最大化时加载会话列表
watch(maximized, (val) => {
  if (val && props.fullCodePath) loadMiniSessions()
})

function toggleMaximize() {
  if (maximized.value) {
    maximized.value = false
    if (preMaxRect.value) {
      posX.value = preMaxRect.value.x
      posY.value = preMaxRect.value.y
      winW.value = preMaxRect.value.w
      winH.value = preMaxRect.value.h
    }
    emit('maximize-change', { maximized: false })
  } else {
    const el = rootRef.value
    if (el) {
      const rect = el.getBoundingClientRect()
      preMaxRect.value = { x: rect.left, y: rect.top, w: rect.width, h: rect.height }
    } else {
      preMaxRect.value = { x: posX.value ?? 0, y: posY.value ?? 0, w: winW.value, h: winH.value }
    }
    maximized.value = true
    emit('maximize-change', { maximized: true, sessionId: sessionId.value })
  }
}

// ─── 拖拽定位 ───
const posX = ref<number | null>(null)
const posY = ref<number | null>(null)
let dragStartX = 0
let dragStartY = 0
let dragOriginX = 0
let dragOriginY = 0
let dragging = false

// ─── 用户指定宽高（拖拽调整或初始值） ───
const MIN_W = 320
const MIN_H = 260
const DEFAULT_W = 380
const DEFAULT_H = 480
const winW = ref(DEFAULT_W)
const winH = ref(DEFAULT_H)

const windowStyle = computed(() => {
  if (maximized.value) {
    return {
      left: '0', top: '0', right: '0', bottom: '0',
      width: '100vw', height: '100vh',
      borderRadius: '0',
      transform: 'none',
    }
  }
  const base: Record<string, string> = {
    width: `${winW.value}px`,
    height: `${winH.value}px`,
  }
  if (posX.value !== null && posY.value !== null) {
    return { ...base, left: `${posX.value}px`, top: `${posY.value}px`, right: 'auto', bottom: 'auto' }
  }
  if (props.initialPosition === 'center') {
    return { ...base, left: '50%', top: '50%', transform: 'translate(-50%, -50%)', right: 'auto', bottom: 'auto' }
  }
  const off = props.initialOffset || 0
  if (off > 0) {
    return { ...base, right: `${24 + off}px`, bottom: `${80 + off}px` }
  }
  return base
})

function startDrag(e: MouseEvent) {
  if (maximized.value) return
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

// ─── 拖拽调整窗口大小 ───
// 四边（n/s/e/w）：等比缩放；四角（ne/nw/se/sw）：自由调整宽高
type ResizeDir = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw'
let resizeDir: ResizeDir = 's'
let resizeStartX = 0
let resizeStartY = 0
let resizeOriginX = 0
let resizeOriginY = 0
let resizeOriginW = 0
let resizeOriginH = 0
let resizeAspect = 1
let resizing = false

function startResize(e: MouseEvent, dir: ResizeDir) {
  e.preventDefault()
  resizing = true
  resizeDir = dir
  resizeStartX = e.clientX
  resizeStartY = e.clientY
  const el = (e.target as HTMLElement).closest('.mini-ws') as HTMLElement
  if (el) {
    const rect = el.getBoundingClientRect()
    resizeOriginX = rect.left
    resizeOriginY = rect.top
    resizeOriginW = rect.width
    resizeOriginH = rect.height
    if (posX.value === null) { posX.value = rect.left; posY.value = rect.top }
  } else {
    resizeOriginW = winW.value
    resizeOriginH = winH.value
  }
  resizeAspect = resizeOriginW / resizeOriginH
  document.addEventListener('mousemove', onResize)
  document.addEventListener('mouseup', stopResize)
}

function onResize(e: MouseEvent) {
  if (!resizing) return
  const dx = e.clientX - resizeStartX
  const dy = e.clientY - resizeStartY
  const d = resizeDir
  const isEdge = d.length === 1

  if (isEdge) {
    // 四边：等比缩放
    if (d === 'e' || d === 'w') {
      const rawW = d === 'e' ? resizeOriginW + dx : resizeOriginW - dx
      const newW = Math.max(MIN_W, rawW)
      const newH = Math.max(MIN_H, Math.round(newW / resizeAspect))
      winW.value = newW
      winH.value = newH
      if (d === 'w') posX.value = resizeOriginX + (resizeOriginW - newW)
    } else {
      const rawH = d === 's' ? resizeOriginH + dy : resizeOriginH - dy
      const newH = Math.max(MIN_H, rawH)
      const newW = Math.max(MIN_W, Math.round(newH * resizeAspect))
      winW.value = newW
      winH.value = newH
      if (d === 'n') posY.value = resizeOriginY + (resizeOriginH - newH)
    }
  } else {
    // 四角：自由调整
    if (d.includes('e')) winW.value = Math.max(MIN_W, resizeOriginW + dx)
    if (d.includes('w')) {
      const newW = Math.max(MIN_W, resizeOriginW - dx)
      posX.value = resizeOriginX + (resizeOriginW - newW)
      winW.value = newW
    }
    if (d.includes('s')) winH.value = Math.max(MIN_H, resizeOriginH + dy)
    if (d.includes('n')) {
      const newH = Math.max(MIN_H, resizeOriginH - dy)
      posY.value = resizeOriginY + (resizeOriginH - newH)
      winH.value = newH
    }
  }
}

function stopResize() {
  resizing = false
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', stopResize)
}

// ─── 自动滚底 ───
function scrollToBottom() {
  nextTick(() => {
    const el = outputRef.value
    if (el) el.scrollTop = el.scrollHeight + 100
  })
}
watch(() => messages.value.length, scrollToBottom)
watch(() => {
  const last = messages.value[messages.value.length - 1]
  return (last?.content?.length ?? 0) + (last?.blocks?.length ?? 0) + (last?.tool_calls?.length ?? 0)
}, scrollToBottom)

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

// ─── 输出数据展示辅助 ───
function getDisplayFieldsFromCalls(calls: Array<{ arguments?: string; result?: string }>): OutputDisplayField[] {
  return extractAllDisplayFields(calls)
}

// ─── 最大化时右侧文件面板：收集所有上传文件 + 输出文件 ───
interface FilePanelItem {
  name: string
  url: string
  source: 'upload' | 'output'
}
const allPanelFiles = computed<FilePanelItem[]>(() => {
  const list: FilePanelItem[] = []
  for (const msg of messages.value) {
    if (msg.role === 'user' && msg.files?.length) {
      for (const f of msg.files) {
        const url = f.url || ''
        list.push({ name: f.source_name || f.name || '未命名文件', url, source: 'upload' })
      }
    }
    if (msg.role === 'assistant' && msg.tool_calls?.length) {
      const groups = getFileGroupsFromCalls(msg.tool_calls)
      for (const g of groups) {
        for (const f of g.files) {
          list.push({ name: f.source_name || f.name || '输出文件', url: f.url, source: 'output' })
        }
      }
    }
  }
  return list
})
const uploadedFiles = computed(() => allPanelFiles.value.filter(f => f.source === 'upload'))
const outputFiles = computed(() => allPanelFiles.value.filter(f => f.source === 'output'))

// ─── 面板：收集所有消息中的 output_display 字段 ───
const allPanelDisplayFields = computed<OutputDisplayField[]>(() => {
  const list: OutputDisplayField[] = []
  for (const msg of messages.value) {
    if (msg.role === 'assistant' && msg.tool_calls?.length) {
      list.push(...extractAllDisplayFields(msg.tool_calls))
    }
  }
  return list
})

const panelHasContent = computed(() => allPanelFiles.value.length > 0 || allPanelDisplayFields.value.length > 0)
const panelItemCount = computed(() => allPanelFiles.value.length + allPanelDisplayFields.value.length)

async function copyDisplayFieldValue(field: OutputDisplayField) {
  try {
    await navigator.clipboard.writeText(field.value)
    ElMessage.success(`已复制「${field.label}」`)
  } catch {
    ElMessage.error('复制失败')
  }
}

// ─── 关键信息 dropdown ref ───
const keyInfoDropdownRef = ref<any>(null)

function onKeyInfoDropdownVisibleChange(visible: boolean) {
  if (!visible && dfPreviewVisible.value) {
    setTimeout(() => { keyInfoDropdownRef.value?.handleOpen?.() }, 50)
  }
}

// ─── 输出数据预览弹窗 ───
const dfPreviewVisible = ref(false)
const dfPreviewLabel = ref('')
const dfPreviewContent = ref('')

function openDfPreview(field: OutputDisplayField) {
  dfPreviewLabel.value = field.label
  dfPreviewContent.value = field.value
  dfPreviewVisible.value = true
}

function closeDfPreview() {
  dfPreviewVisible.value = false
}

async function copyDfPreviewContent() {
  try {
    await navigator.clipboard.writeText(dfPreviewContent.value)
    ElMessage.success(`已复制「${dfPreviewLabel.value}」`)
  } catch {
    ElMessage.error('复制失败')
  }
}

const IMAGE_EXTS = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.bmp', '.svg'])
function isImageFile(f: FilePanelItem): boolean {
  const ext = (f.name || '').toLowerCase().match(/\.\w+$/)?.[0] || ''
  return IMAGE_EXTS.has(ext)
}
function fileExt(f: FilePanelItem): string {
  return ((f.name || '').match(/\.(\w+)$/)?.[1] || '').toUpperCase()
}
function previewFile(file: FilePanelItem) {
  window.open(file.url, '_blank', 'noopener,noreferrer')
}
function downloadFile(file: FilePanelItem) {
  const a = document.createElement('a')
  a.href = file.url
  a.download = file.name
  a.target = '_blank'
  a.rel = 'noopener noreferrer'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
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
  const dt = e.dataTransfer
  if (!dt) return

  // 优先处理：从服务目录拖入的函数/目录节点
  if (dt.types.includes('application/x-workspace-node')) {
    try {
      const raw = dt.getData('application/x-workspace-node')
      const payload = raw ? JSON.parse(raw) as { type?: string; full_code_path?: string; name?: string } : null
      if (payload?.full_code_path) {
        const label = payload.type === 'package' ? '目录' : '函数'
        const name = payload.name || payload.full_code_path.split('/').pop() || payload.full_code_path
        const text = `请处理以下${label}：${name}（${payload.full_code_path}）`
        inputText.value = text
        nextTick(() => inputRef.value?.focus())
      }
    } catch (_) {
      /* ignore parse error */
    }
    return
  }

  const files = dt.files
  if (!files?.length || !props.fullCodePath) return
  for (const file of Array.from(files)) {
    await onFileChange({ raw: file })
  }
}

/** 双击标题栏切换最大化 */
function onHeaderDblClick() {
  toggleMaximize()
}

// ─── 发送 ───
function onInputEnter(e: KeyboardEvent) {
  if ((e as KeyboardEvent).shiftKey) return // Shift+Enter 换行，不拦截
  e.preventDefault()
  handleSend()
}

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
        if (maximized.value) {
          loadMiniSessions()
          emit('maximize-change', { maximized: true, sessionId: data.session_id as string })
        }
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
    sending.value = false
    sessionId.value = newSid
    setMessages([])
    await loadMiniSessionMessages(newSid)
    if (sessionId.value !== newSid) return
    startMiniStreamListening(newSid)
    startMiniPoll(newSid)
  },
  { immediate: true }
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
  if (prev && !cur && maximized.value) loadMiniSessions()
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
.mini-files-dropdown-body .mini-file-card {
  padding: 6px 8px;
  margin-bottom: 4px;
  cursor: pointer;
}
.mini-files-dropdown-body .mini-file-thumb {
  width: 40px;
  height: 40px;
}
.mini-files-dropdown-body .mini-file-icon {
  width: 40px;
  height: 40px;
}
.mini-files-dropdown-body .mini-file-icon .el-icon {
  font-size: 18px;
}
.mini-files-dropdown-body .mini-file-name {
  font-size: 12px;
}
.mini-files-dropdown-body .mini-file-actions {
  margin-top: 2px;
}
.mini-files-dropdown-body .mini-file-actions .el-button {
  font-size: 11px;
}

/* ── 标题栏下拉：输出数据卡片 ── */
.mini-display-field-card {
  padding: 6px 8px;
  margin-bottom: 4px;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: var(--el-border-radius-small);
  background: var(--el-bg-color);
}
.mini-df-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}
.mini-df-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.mini-df-actions {
  display: flex;
  gap: 6px;
  align-items: center;
}
.mini-df-value {
  font-size: 11px;
  line-height: 1.5;
  color: var(--el-text-color-regular);
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 4.5em;
  overflow: hidden;
}

/* ── 主体区域（sidebar + output） ── */
.mini-ws-body {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
}

/* ── 最大化会话侧边栏 ── */
.mini-session-sidebar {
  width: 220px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);
}
.mini-session-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  flex-shrink: 0;
}
.mini-session-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.mini-session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}
.mini-session-card {
  padding: 10px;
  margin-bottom: 6px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  background: var(--el-bg-color);
  cursor: pointer;
  transition: all 0.15s;
}
.mini-session-card:hover {
  border-color: var(--el-color-primary);
  background: var(--el-fill-color-lighter);
}
.mini-session-card.active {
  border-color: var(--el-color-primary);
  border-width: 2px;
}
.mini-session-card.generating {
  border-left: 2px solid var(--el-color-primary);
}
.mini-session-new {
  border-style: dashed;
  background: var(--el-fill-color-lighter);
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.mini-session-new:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary);
}
.mini-session-new-icon {
  color: var(--el-color-primary);
}
.mini-session-card-head {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 4px;
}
.mini-session-card-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}
.mini-session-card-user {
  margin-top: 4px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.mini-session-card-user :deep(.user-display-wrapper) {
  display: inline-flex;
}
.mini-session-card-time {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  display: flex;
  align-items: center;
  gap: 6px;
}
.mini-session-status {
  color: var(--el-color-primary);
  font-size: 11px;
  font-weight: 500;
}
.mini-session-empty {
  padding: 24px;
  text-align: center;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
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
.mini-file-card {
  display: flex;
  gap: 8px;
  padding: 8px;
  margin-bottom: 6px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  transition: all 0.15s;
  &:hover {
    border-color: var(--el-color-primary-light-5);
    background: var(--el-fill-color-lighter);
  }
}
.mini-file-thumb {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  border-radius: 4px;
  overflow: hidden;
  cursor: pointer;
  border: 1px solid var(--el-border-color-extra-light);
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
  &:hover { opacity: 0.8; }
}
.mini-file-icon {
  width: 48px;
  height: 48px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  cursor: pointer;
  &:hover { background: var(--el-fill-color); }
}
.mini-file-ext {
  font-size: 9px;
  font-weight: 600;
  color: var(--el-text-color-placeholder);
  margin-top: 2px;
  text-transform: uppercase;
}
.mini-file-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
}
.mini-file-name {
  font-size: 12px;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.4;
}
.mini-file-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

/* ── 最大化右侧面板：输出数据卡片 ── */
.mini-sidebar-df-card {
  margin-bottom: 6px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  overflow: hidden;
}
.mini-sidebar-df-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color-extra-light);
}
.mini-sidebar-df-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.mini-sidebar-df-actions {
  display: flex;
  gap: 6px;
  align-items: center;
}
.mini-sidebar-df-value {
  padding: 6px 10px;
  max-height: 200px;
  overflow-y: auto;
}
.mini-sidebar-df-pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 11px;
  line-height: 1.5;
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  color: var(--el-text-color-regular);
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

/* ── 输出数据预览弹窗（自定义实现，z-index 99999） ── */
.df-preview-overlay {
  position: fixed;
  inset: 0;
  z-index: 99999;
  background: rgba(0, 0, 0, 0.45);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}
.df-preview-modal {
  width: 860px;
  max-width: 92vw;
  max-height: 88vh;
  background: var(--el-bg-color, #fff);
  border-radius: 8px;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.2);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.df-preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--el-border-color-lighter, #eee);
  flex-shrink: 0;
}
.df-preview-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary, #303133);
}
.df-preview-close {
  border: none;
  background: none;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  color: var(--el-text-color-secondary, #909399);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
  &:hover {
    background: var(--el-fill-color-light, #f5f7fa);
    color: var(--el-text-color-primary, #303133);
  }
}
.df-preview-body {
  flex: 1;
  min-height: 0;
  padding: 16px 20px;
  overflow: hidden;
}
.df-preview-textarea {
  width: 100%;
  min-height: 360px;
  max-height: calc(88vh - 140px);
  padding: 12px 14px;
  border: 1px solid var(--el-border-color, #dcdfe6);
  border-radius: 6px;
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.7;
  color: var(--el-text-color-primary, #303133);
  background: var(--el-fill-color-blank, #fff);
  resize: vertical;
  outline: none;
  box-sizing: border-box;
  &:focus {
    border-color: var(--el-color-primary, #409eff);
  }
}
.df-preview-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  border-top: 1px solid var(--el-border-color-lighter, #eee);
  flex-shrink: 0;
}
.df-preview-stats {
  font-size: 12px;
  color: var(--el-text-color-secondary, #909399);
}
.df-preview-actions {
  display: flex;
  gap: 8px;
}
.df-preview-btn {
  padding: 8px 16px;
  border: 1px solid var(--el-border-color, #dcdfe6);
  border-radius: 4px;
  background: var(--el-bg-color, #fff);
  color: var(--el-text-color-regular, #606266);
  font-size: 13px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  transition: all 0.15s;
  &:hover {
    border-color: var(--el-color-primary-light-3, #79bbff);
    color: var(--el-color-primary, #409eff);
  }
}
.df-preview-btn--primary {
  background: var(--el-color-primary, #409eff);
  border-color: var(--el-color-primary, #409eff);
  color: #fff;
  &:hover {
    background: var(--el-color-primary-light-3, #79bbff);
    border-color: var(--el-color-primary-light-3, #79bbff);
    color: #fff;
  }
}
.df-preview-fade-enter-active {
  transition: opacity 0.2s ease;
}
.df-preview-fade-leave-active {
  transition: opacity 0.15s ease;
}
.df-preview-fade-enter-from,
.df-preview-fade-leave-to {
  opacity: 0;
}
</style>
