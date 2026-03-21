<template>
  <div class="scheduled-task-list" v-loading="loading">
    <div class="list-toolbar">
      <span class="list-tip">仅展示当前函数的定时任务（按路径过滤）</span>
      <el-button type="primary" link @click="loadList">刷新</el-button>
    </div>
    <el-empty v-if="!loading && list.length === 0" description="暂无定时任务" />
    <el-table v-else :data="list" stripe style="width: 100%">
      <el-table-column prop="name" label="任务名称" min-width="160" show-overflow-tooltip />
      <el-table-column prop="action" label="动作" width="110">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ actionLabel(row.action) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="schedule_type" label="类型" width="100">
        <template #default="{ row }">
          <el-tag size="small">{{ scheduleTypeLabel(row.schedule_type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="run_at" label="首次执行" width="165">
        <template #default="{ row }">{{ formatDateTime(row.run_at) }}</template>
      </el-table-column>
      <el-table-column prop="next_run_at" label="下次执行" width="165">
        <template #default="{ row }">
          {{ row.next_run_at ? formatDateTime(row.next_run_at) : '-' }}
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="run_count" label="已执行" width="80" />
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="row.status === 'pending'"
            type="danger"
            link
            size="small"
            @click="handleCancel(row)"
          >
            取消
          </el-button>
          <el-button type="primary" link size="small" @click="openExecutions(row)">
            执行记录
          </el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-pagination
      v-if="total > 0"
      v-model:current-page="page"
      v-model:page-size="pageSize"
      :total="total"
      :page-sizes="[10, 20, 50]"
      layout="total, sizes, prev, pager, next"
      class="list-pagination"
      @current-change="loadList"
      @size-change="loadList"
    />

    <el-dialog
      v-model="executionsVisible"
      :title="`执行记录：${currentTaskName}`"
      width="80%"
      destroy-on-close
    >
      <el-table :data="executions" stripe v-loading="executionsLoading">
        <el-table-column prop="executed_at" label="执行时间" width="175">
          <template #default="{ row }">{{ formatDateTime(row.executed_at) }}</template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="error_message" label="错误信息" min-width="200" show-overflow-tooltip />
        <el-table-column label="请求/响应" width="120">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="showPayload(row.request_payload, '请求')">
              请求
            </el-button>
            <el-button link type="primary" size="small" @click="showPayload(row.response_payload, '响应')">
              响应
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        v-if="executionsTotal > 0"
        v-model:current-page="executionsPage"
        v-model:page-size="executionsPageSize"
        :total="executionsTotal"
        layout="total, prev, pager, next"
        class="executions-pagination"
        @current-change="loadExecutions"
        @size-change="loadExecutions"
      />
    </el-dialog>

    <el-dialog v-model="payloadVisible" :title="payloadDialogTitle" width="600px" destroy-on-close>
      <pre class="payload-pre">{{ payloadPreview }}</pre>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listScheduledTasks,
  cancelScheduledTask,
  listScheduledTaskExecutions,
  type ScheduledTaskItem,
  type ScheduledTaskExecutionItem
} from '@/api/scheduledTask'
import { eventBus, WorkspaceEvent } from '@/architecture/infrastructure/eventBus'

const props = withDefaults(
  defineProps<{
    resourcePath?: string
    autoLoad?: boolean
  }>(),
  { autoLoad: false }
)
const emit = defineEmits<{ (e: 'total-change', total: number): void }>()

const loading = ref(false)
const list = ref<ScheduledTaskItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

function scheduleTypeLabel(t: string) {
  const m: Record<string, string> = { atime: '指定时间', cron: 'Cron', every: '每 N 秒' }
  return m[t] ?? t
}

function actionLabel(a?: string) {
  const m: Record<string, string> = {
    execute: '普通执行',
    table_create: '表格新增',
    table_update: '表格更新',
    table_delete: '表格删除'
  }
  return a ? (m[a] ?? a) : '普通执行'
}

function statusTagType(s: string) {
  const m: Record<string, string> = {
    pending: 'warning',
    done: 'success',
    failed: 'danger',
    cancelled: 'info'
  }
  return m[s] ?? 'info'
}

/** 状态中文展示 */
function statusLabel(s: string) {
  const m: Record<string, string> = {
    pending: '待执行',
    done: '已完成',
    failed: '失败',
    cancelled: '已取消',
    success: '成功'
  }
  return m[s] ?? s
}

function formatDateTime(s: string) {
  if (!s) return ''
  const d = new Date(s)
  return d.toLocaleString('zh-CN')
}

async function loadList() {
  if (!props.resourcePath) {
    list.value = []
    total.value = 0
    return
  }
  loading.value = true
  try {
    const res = await listScheduledTasks({
      full_code_path: props.resourcePath,
      page: page.value,
      page_size: pageSize.value
    })
    list.value = res.list ?? []
    total.value = res.total ?? 0
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
    emit('total-change', total.value)
  }
}

watch(
  () => [props.resourcePath, props.autoLoad] as const,
  ([path, auto]) => {
    if (path && auto) loadList()
    else if (!path) {
      list.value = []
      total.value = 0
    }
  },
  { immediate: true }
)

eventBus.on(WorkspaceEvent.scheduledTaskCreated, () => {
  if (props.resourcePath) loadList()
})

function handleCancel(row: ScheduledTaskItem) {
  ElMessageBox.confirm(`确定取消定时任务「${row.name}」？`, '取消任务', {
    type: 'warning'
  }).then(async () => {
    try {
      await cancelScheduledTask(row.id)
      ElMessage.success('已取消')
      loadList()
    } catch (e: any) {
      ElMessage.error(e?.message || '取消失败')
    }
  }).catch(() => {})
}

const executionsVisible = ref(false)
const executions = ref<ScheduledTaskExecutionItem[]>([])
const executionsTotal = ref(0)
const executionsPage = ref(1)
const executionsPageSize = ref(20)
const executionsLoading = ref(false)
const currentTaskId = ref(0)
const currentTaskName = ref('')

function openExecutions(row: ScheduledTaskItem) {
  currentTaskId.value = row.id
  currentTaskName.value = row.name
  executionsPage.value = 1
  executionsVisible.value = true
  loadExecutions()
}

async function loadExecutions() {
  if (!currentTaskId.value) return
  executionsLoading.value = true
  try {
    const res = await listScheduledTaskExecutions(currentTaskId.value, {
      page: executionsPage.value,
      page_size: executionsPageSize.value
    })
    executions.value = res.list ?? []
    executionsTotal.value = res.total ?? 0
  } catch {
    executions.value = []
    executionsTotal.value = 0
  } finally {
    executionsLoading.value = false
  }
}

const payloadVisible = ref(false)
const payloadDialogTitle = ref('')
const payloadPreview = ref('')

function showPayload(raw: string, title: string) {
  try {
    payloadPreview.value = JSON.stringify(JSON.parse(raw || '{}'), null, 2)
  } catch {
    payloadPreview.value = raw || ''
  }
  payloadDialogTitle.value = title
  payloadVisible.value = true
}
</script>

<style scoped>
.scheduled-task-list {
  padding: 12px;
}
.list-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.list-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.list-pagination,
.executions-pagination {
  margin-top: 16px;
  justify-content: flex-end;
}
.payload-pre {
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 60vh;
  overflow: auto;
  font-size: 12px;
}
</style>
