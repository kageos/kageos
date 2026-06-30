<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import dayjs from 'dayjs'
import { ElMessage } from 'element-plus'
import {
  getPublicMessageAction,
  submitPublicMessageActionReply,
  type MessageActionReplyResp,
  type MessageActionViewResp,
  type MessageInboxItem,
} from '@/architecture/presentation/context/api/message'

const route = useRoute()
const loading = ref(false)
const submitting = ref(false)
const error = ref('')
const view = ref<MessageActionViewResp | null>(null)
const replyContent = ref('')
const replyResult = ref<MessageActionReplyResp | null>(null)

const token = computed(() => {
  const raw = route.query.t
  return Array.isArray(raw) ? String(raw[0] || '') : String(raw || '')
})

const displayThread = computed<MessageInboxItem[]>(() => {
  const list = view.value?.thread || []
  return [...list].reverse()
})

const expiresText = computed(() => {
  const expiresAt = view.value?.expires_at
  return expiresAt ? dayjs(expiresAt).format('YYYY-MM-DD HH:mm') : ''
})

const statusText = computed(() => {
  const status = view.value?.token_status
  if (status === 'open') return '等待处理'
  if (status === 'submitted') return '已提交'
  if (status === 'expired') return '已过期'
  if (status === 'revoked') return '已撤销'
  return status || '未知状态'
})

async function loadAction() {
  if (!token.value.trim()) {
    error.value = '处理链接缺少 token'
    return
  }
  loading.value = true
  error.value = ''
  try {
    view.value = await getPublicMessageAction(token.value)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载消息失败'
  } finally {
    loading.value = false
  }
}

async function submitReply() {
  if (!view.value?.can_reply || submitting.value) return
  const content = replyContent.value.trim()
  if (!content) {
    ElMessage.warning('请输入回复内容')
    return
  }
  submitting.value = true
  try {
    const result = await submitPublicMessageActionReply(token.value, { content, action: 'reply' })
    replyResult.value = result
    replyContent.value = ''
    if (result.agent_submitted) {
      ElMessage.success('已提交给工作台，处理完成后会收到简短结论')
    } else if (result.agent_submit_error) {
      ElMessage.warning('消息已处理，但工作台提交失败')
    } else {
      ElMessage.success('已提交')
    }
    await loadAction()
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : '提交失败')
  } finally {
    submitting.value = false
  }
}

function formatTime(value?: string | null) {
  return value ? dayjs(value).format('MM-DD HH:mm') : ''
}

onMounted(loadAction)
watch(token, loadAction)
</script>

<template>
  <main class="mobile-action-page">
    <section class="mobile-action-shell">
      <div class="mobile-action-topbar">
        <div>
          <div class="mobile-action-brand">Kageos Pocket</div>
          <h1>处理消息</h1>
        </div>
        <span class="status-pill" :class="`status-${view?.token_status || 'loading'}`">{{ statusText }}</span>
      </div>

      <el-skeleton v-if="loading" :rows="8" animated />

      <el-alert
        v-else-if="error"
        type="error"
        :title="error"
        show-icon
        :closable="false"
      />

      <template v-else-if="view">
        <section class="message-head">
          <div class="message-source">
            {{ view.message.source_display?.name || view.message.source_title || view.message.source_path || 'Kageos 消息' }}
          </div>
          <h2>{{ view.message.title || '未命名消息' }}</h2>
          <div class="message-meta">
            <span>{{ view.message.from || 'system' }}</span>
            <span>{{ formatTime(view.message.created_at) }}</span>
            <span v-if="expiresText">有效期至 {{ expiresText }}</span>
          </div>
        </section>

        <section class="message-content">
          <pre>{{ view.message.content }}</pre>
        </section>

        <section v-if="displayThread.length > 1" class="thread-section">
          <h3>最近上下文</h3>
          <div class="thread-list">
            <article v-for="item in displayThread" :key="item.id" class="thread-item">
              <div class="thread-meta">
                <strong>{{ item.from || 'system' }}</strong>
                <span>{{ formatTime(item.created_at) }}</span>
              </div>
              <div class="thread-title">{{ item.title || '消息' }}</div>
              <p>{{ item.content }}</p>
            </article>
          </div>
        </section>

        <section class="reply-section">
          <h3>提交给工作台</h3>
          <el-alert
            v-if="replyResult?.agent_submitted"
            type="success"
            :title="replyResult.workspace_session_id ? `已提交给工作台会话 ${replyResult.workspace_session_id}，处理完成后会收到简短结论。` : '已提交给工作台处理，处理完成后会收到简短结论。'"
            show-icon
            :closable="false"
          />
          <el-alert
            v-else-if="replyResult?.agent_submit_error"
            type="warning"
            :title="`消息已处理，但工作台提交失败：${replyResult.agent_submit_error}`"
            show-icon
            :closable="false"
          />
          <el-input
            v-model="replyContent"
            type="textarea"
            :rows="5"
            maxlength="8000"
            show-word-limit
            resize="none"
            placeholder="请用 Markdown 写清处理意见，例如：&#10;- 延迟到下午 5 点&#10;- 通知相关人"
            :disabled="!view.can_reply || submitting"
          />
          <div class="reply-actions">
            <el-button
              type="primary"
              size="large"
              :loading="submitting"
              :disabled="!view.can_reply"
              @click="submitReply"
            >
              提交给工作台
            </el-button>
            <el-button
              v-if="view.mobile_ask_url"
              tag="a"
              :href="view.mobile_ask_url"
              size="large"
            >
              主动问 Kageos
            </el-button>
          </div>
          <p v-if="!view.can_reply" class="reply-hint">
            当前链接不能继续提交，请回到 Kageos 工作台查看最新状态。
          </p>
        </section>
      </template>
    </section>
  </main>
</template>

<style scoped>
.mobile-action-page {
  min-height: 100vh;
  background: #f5f7fb;
  color: #172033;
  padding: 16px;
}

.mobile-action-shell {
  width: min(100%, 760px);
  margin: 0 auto;
  display: grid;
  gap: 14px;
}

.mobile-action-topbar {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
  padding: 8px 0 2px;
}

.mobile-action-brand {
  font-size: 13px;
  font-weight: 700;
  color: #3c6df0;
}

h1,
h2,
h3 {
  margin: 0;
  letter-spacing: 0;
}

h1 {
  font-size: 26px;
  line-height: 1.2;
}

h2 {
  font-size: 22px;
  line-height: 1.25;
  margin-top: 5px;
}

h3 {
  font-size: 16px;
}

.status-pill {
  flex: 0 0 auto;
  padding: 7px 10px;
  border-radius: 999px;
  background: #e7ecf8;
  color: #40506b;
  font-size: 13px;
  font-weight: 700;
}

.status-open {
  background: #e8f4ff;
  color: #1465c0;
}

.status-submitted {
  background: #e8f7ee;
  color: #17663b;
}

.status-expired,
.status-revoked {
  background: #f1f2f5;
  color: #727b8d;
}

.message-head,
.message-content,
.thread-section,
.reply-section {
  background: #ffffff;
  border: 1px solid #e2e7f1;
  border-radius: 8px;
  padding: 16px;
}

.message-source,
.message-meta,
.thread-meta,
.reply-hint {
  color: #657089;
  font-size: 13px;
}

.message-source {
  font-weight: 700;
}

.message-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
  margin-top: 10px;
}

.message-content pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: inherit;
  font-size: 15px;
  line-height: 1.65;
}

.thread-list {
  display: grid;
  gap: 10px;
  margin-top: 12px;
}

.thread-item {
  border: 1px solid #e7ebf3;
  border-radius: 8px;
  padding: 12px;
  background: #fbfcff;
}

.thread-meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.thread-title {
  margin-top: 6px;
  font-weight: 700;
}

.thread-item p {
  margin: 6px 0 0;
  color: #2f3a4e;
  font-size: 14px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}

.reply-section {
  display: grid;
  gap: 12px;
}

.reply-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

@media (max-width: 520px) {
  .mobile-action-page {
    padding: 12px;
  }

  .mobile-action-topbar {
    align-items: stretch;
    flex-direction: column;
  }

  .status-pill {
    width: fit-content;
  }

  .reply-actions :deep(.el-button) {
    flex: 1 1 150px;
  }
}
</style>
