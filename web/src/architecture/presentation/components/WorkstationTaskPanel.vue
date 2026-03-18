<!--
  WorkstationTaskPanel - 全局浮动工作台任务面板
  右上角常驻 badge，点击展开面板查看执行中/已结束的任务。
  支持预览弹窗查看工作台 SSE 输出过程。
-->
<template>
  <!-- 触发按钮：图标 + 气泡 badge -->
  <div class="task-panel-trigger" @click="panelVisible = !panelVisible">
    <el-badge :value="runningCount" :hidden="runningCount === 0" :max="99" :offset="[-2, 2]">
      <el-icon :size="18" :class="{ 'is-loading': runningCount > 0 }" class="trigger-icon"><Monitor /></el-icon>
    </el-badge>
  </div>

  <!-- 浮动面板 -->
  <transition name="slide-right">
    <div v-if="panelVisible" class="task-panel">
      <div class="task-panel-header">
        <span class="task-panel-title">工作台任务</span>
        <el-button link @click="panelVisible = false" class="close-btn">
          <el-icon><Close /></el-icon>
        </el-button>
      </div>

      <!-- Tab 切换 -->
      <div class="task-panel-tabs">
        <div class="tabs-left">
          <div
            :class="['tab-item', { active: activeTab === 'running' }]"
            @click="activeTab = 'running'"
          >
            执行中
            <span v-if="runningCount > 0" class="tab-count">{{ runningCount }}</span>
          </div>
          <div
            :class="['tab-item', { active: activeTab === 'finished' }]"
            @click="switchToFinished"
          >
            已结束
          </div>
        </div>
        <div class="tabs-right">
          <el-segmented v-model="scopeFilter" :options="scopeOptions" size="small" />
        </div>
      </div>

      <div class="task-panel-body">
        <!-- 执行中 tab -->
        <template v-if="activeTab === 'running'">
          <div v-if="runningTasks.length === 0 && !loadingRunning" class="empty-state">
            <el-empty description="暂无执行中的任务" :image-size="60" />
          </div>
          <div v-else class="task-list">
            <div v-for="task in runningTasks" :key="task.session_id" class="task-card">
              <div class="task-card-main">
                <div class="task-card-left">
                  <span class="task-card-title">{{ task.title || '未命名任务' }}</span>
                  <span class="task-card-path">{{ shortenPath(task.full_code_path) }}</span>
                </div>
                <div class="task-card-right">
                  <span class="task-status task-status--primary">
                    <span class="status-dot status-dot--pulse" />
                    执行中
                  </span>
                  <span class="task-card-time">{{ formatRelativeTime(task.updated_at) }}</span>
                </div>
              </div>
              <div class="task-card-actions">
                <el-button size="small" link type="primary" @click="handleOpenFullScreen(task)">查看</el-button>
                <el-button size="small" link type="danger" @click="handleCancel(task)" :loading="cancellingId === task.session_id">停止</el-button>
              </div>
            </div>
          </div>
        </template>

        <!-- 已结束 tab -->
        <template v-else>
          <div v-if="finishedTasks.length === 0 && !loadingFinished" class="empty-state">
            <el-empty description="暂无已结束的任务" :image-size="60" />
          </div>
          <div v-else class="task-list">
            <div v-for="task in finishedTasks" :key="task.session_id" class="task-card">
              <div class="task-card-main">
                <div class="task-card-left">
                  <span class="task-card-title">{{ task.title || '未命名任务' }}</span>
                  <span class="task-card-path">{{ shortenPath(task.full_code_path) }}</span>
                </div>
                <div class="task-card-right">
                  <span :class="['task-status', task.status === 'cancelled' ? 'task-status--info' : 'task-status--success']">
                    <span class="status-dot" />
                    {{ task.status === 'cancelled' ? '已取消' : '已完成' }}
                  </span>
                  <span class="task-card-time">{{ formatRelativeTime(task.updated_at) }}</span>
                </div>
              </div>
              <div class="task-card-actions">
                <el-button size="small" link type="primary" @click="handleOpenFullScreen(task)">查看</el-button>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </transition>

</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { Monitor, Close } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { eventBus } from '@/architecture/infrastructure/eventBus'
import {
  getRunningSessions,
  getFinishedSessions,
  cancelWorkspaceChat,
  type WorkspaceSessionItem,
} from '@/api/workspace'

const props = withDefaults(defineProps<{
  currentFullCodePath?: string
}>(), { currentFullCodePath: '' })


// ─── 目录筛选 ───
const scopeFilter = ref<'all' | 'current'>('all')
const scopeOptions = [
  { label: '全部', value: 'all' },
  { label: '当前目录', value: 'current' },
]

function normalizePath(p?: string): string {
  return (p || '').replace(/^\/+|\/+$/g, '')
}

function filterByScope(tasks: WorkspaceSessionItem[]): WorkspaceSessionItem[] {
  if (scopeFilter.value === 'current' && props.currentFullCodePath) {
    const current = normalizePath(props.currentFullCodePath)
    return tasks.filter(t => normalizePath(t.full_code_path) === current)
  }
  return tasks
}

// ─── 主面板状态 ───
const panelVisible = ref(false)
const activeTab = ref<'running' | 'finished'>('running')

const allRunningTasks = ref<WorkspaceSessionItem[]>([])
const allFinishedTasks = ref<WorkspaceSessionItem[]>([])
const loadingRunning = ref(false)
const loadingFinished = ref(false)
const cancellingId = ref<string | null>(null)

let pollTimer: ReturnType<typeof setInterval> | null = null

const runningTasks = computed(() => filterByScope(allRunningTasks.value))
const finishedTasks = computed(() => filterByScope(allFinishedTasks.value))
const runningCount = computed(() => runningTasks.value.length)


// ─── 工具函数 ───
function shortenPath(path?: string): string {
  if (!path) return ''
  const parts = path.split('/').filter(Boolean)
  return parts.length > 2 ? parts.slice(-3).join(' / ') : path
}

function formatRelativeTime(dateStr: string): string {
  if (!dateStr) return ''
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins}分钟前`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}小时前`
  return `${Math.floor(hours / 24)}天前`
}


// ─── 数据加载 ───
async function loadRunningTasks() {
  loadingRunning.value = true
  try {
    const res = await getRunningSessions()
    allRunningTasks.value = res?.sessions || []
  } catch (e: any) {
    console.error('[TaskPanel] loadRunning error:', e)
  } finally {
    loadingRunning.value = false
  }
}

async function loadFinishedTasks() {
  loadingFinished.value = true
  try {
    const res = await getFinishedSessions(20)
    allFinishedTasks.value = res?.sessions || []
  } catch (e: any) {
    console.error('[TaskPanel] loadFinished error:', e)
  } finally {
    loadingFinished.value = false
  }
}

function switchToFinished() {
  activeTab.value = 'finished'
  loadFinishedTasks()
}

// ─── 打开全屏工作台 ───
function handleOpenFullScreen(task: WorkspaceSessionItem) {
  panelVisible.value = false
  if (task.full_code_path) {
    eventBus.emit('workspace:open-workstation', {
      full_code_path: task.full_code_path,
      session_id: task.session_id,
      open_as_mini: true,
    })
  }
}

async function handleCancel(task: WorkspaceSessionItem) {
  cancellingId.value = task.session_id
  try {
    await cancelWorkspaceChat(task.session_id)
    task.status = 'cancelled'
    ElMessage.success('已停止该任务')
    await loadRunningTasks()
  } catch (e: any) {
    ElMessage.error(e?.message || '停止失败')
  } finally {
    cancellingId.value = null
  }
}

// ─── 定时轮询 ───
function startPoll() {
  stopPoll()
  pollTimer = setInterval(() => {
    loadRunningTasks()
  }, 5000)
}

function stopPoll() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// 面板展开时刷新
watch(panelVisible, (v) => {
  if (v) {
    loadRunningTasks()
    if (activeTab.value === 'finished') loadFinishedTasks()
  }
})

onMounted(() => {
  loadRunningTasks()
  startPoll()
})

onUnmounted(() => {
  stopPoll()
})
</script>

<style scoped>
.task-panel-trigger {
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 6px;
  border-radius: 6px;
  transition: background 0.2s;
  position: relative;
}
.task-panel-trigger:hover {
  background: var(--el-fill-color-light);
}
.trigger-icon {
  color: var(--el-text-color-regular);
}
.task-panel-trigger:hover .trigger-icon {
  color: var(--el-color-primary);
}

/* ── 浮动面板 ── */
.task-panel {
  position: fixed;
  top: 48px;
  right: 12px;
  width: 440px;
  max-height: calc(100vh - 72px);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
  z-index: 2000;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.task-panel-header {
  display: flex;
  align-items: center;
  padding: 14px 16px 0;
}
.task-panel-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  flex: 1;
}
.close-btn { margin-left: auto; }

/* ── Tabs ── */
.task-panel-tabs {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  padding: 8px 12px 0 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.tabs-left {
  display: flex;
  gap: 0;
}
.tabs-right {
  padding-bottom: 6px;
}
.tab-item {
  padding: 8px 16px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 4px;
}
.tab-item:hover {
  color: var(--el-text-color-primary);
}
.tab-item.active {
  color: var(--el-color-primary);
  border-bottom-color: var(--el-color-primary);
  font-weight: 600;
}
.tab-count {
  background: var(--el-color-danger);
  color: #fff;
  font-size: 11px;
  min-width: 18px;
  height: 18px;
  line-height: 18px;
  text-align: center;
  border-radius: 9px;
  padding: 0 5px;
  font-weight: 500;
}

/* ── Body ── */
.task-panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.empty-state { padding: 32px 0; }

.task-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

/* ── 任务卡片 ── */
.task-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 10px 12px;
  transition: border-color 0.2s;
}
.task-card:hover {
  border-color: var(--el-color-primary-light-5);
}
.task-card-main {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
}
.task-card-left {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.task-card-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.task-card-path {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.task-card-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  flex-shrink: 0;
}
.task-card-time {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}
.task-card-actions {
  display: flex;
  gap: 6px;
  margin-top: 6px;
  justify-content: flex-end;
}

/* ── 状态标签 ── */
.task-status {
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}
.task-status--primary { color: var(--el-color-primary); }
.task-status--success { color: var(--el-color-success); }
.task-status--info { color: var(--el-text-color-secondary); }

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  display: inline-block;
  background: currentColor;
}
.status-dot--pulse {
  animation: dot-pulse 1.5s ease-in-out infinite;
}
@keyframes dot-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

/* ── 动画 ── */
.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}
.slide-right-enter-from,
.slide-right-leave-to {
  transform: translateX(20px);
  opacity: 0;
}
</style>
