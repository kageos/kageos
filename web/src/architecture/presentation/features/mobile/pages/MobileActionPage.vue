<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
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
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'
import OutputFilesDisplay from '@/architecture/presentation/components/OutputFilesDisplay.vue'
import type { OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'

const route = useRoute()
const loading = ref(false)
const submitting = ref(false)
const error = ref('')
const view = ref<MessageActionViewResp | null>(null)
const replyContent = ref('')
const replyResult = ref<MessageActionReplyResp | null>(null)
const pageEndRef = ref<HTMLElement | null>(null)

const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()

const token = computed(() => {
  const raw = route.query.t
  return Array.isArray(raw) ? String(raw[0] || '') : String(raw || '')
})

const displayThread = computed<MessageInboxItem[]>(() => {
  const list = view.value?.thread || []
  return [...list].reverse()
})

const renderedMessageContent = computed(() => renderMarkdown(view.value?.message.content || ''))

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
    await scrollToPageBottom('auto')
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

function renderThreadContent(item: MessageInboxItem) {
  return renderMarkdown(item.content || '')
}

function parseMessageFileRefs(files?: string): string[] {
  return Array.from(new Set((files || '')
    .split(',')
    .map(ref => ref.trim().replace(/^\/+/, ''))
    .filter(Boolean)))
}

function messageFileGroups(item?: MessageInboxItem | null): OutputFileGroup[] {
  const refs = parseMessageFileRefs(item?.files)
  if (refs.length === 0) return []
  return [{
    label: '附件',
    files: refs.map(ref => ({
      ref,
      name: ref.split('/').pop() || '文件'
    }))
  }]
}

async function scrollToPageBottom(behavior: ScrollBehavior = 'smooth') {
  await nextTick()
  pageEndRef.value?.scrollIntoView({ behavior, block: 'end' })
}

onMounted(loadAction)
watch(token, loadAction)
watch([renderedMessageContent, displayThread], () => {
  if (view.value) {
    void scrollToPageBottom('auto')
  }
}, { flush: 'post' })
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
          <div class="message-markdown" v-html="renderedMessageContent" />
          <OutputFilesDisplay
            v-if="messageFileGroups(view.message).length > 0"
            class="mobile-message-files"
            :file-groups="messageFileGroups(view.message)"
            section-title="附件"
          />
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
              <div class="thread-body" v-html="renderThreadContent(item)" />
              <OutputFilesDisplay
                v-if="messageFileGroups(item).length > 0"
                class="mobile-thread-files"
                :file-groups="messageFileGroups(item)"
                section-title="附件"
              />
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
        <div ref="pageEndRef" class="mobile-page-end" aria-hidden="true" />
      </template>
    </section>
  </main>
</template>

<style scoped>
.mobile-action-page {
  min-height: 100dvh;
  background: #eef3f7;
  color: #172033;
  padding: 14px 14px calc(20px + env(safe-area-inset-bottom));
}

.mobile-action-shell {
  width: min(100%, 760px);
  margin: 0 auto;
  display: grid;
  gap: 12px;
}

.mobile-action-topbar {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
  padding: 8px 0 6px;
  background: rgba(238, 243, 247, 0.94);
  backdrop-filter: blur(10px);
}

.mobile-action-brand {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  min-height: 24px;
  padding: 0 8px;
  border: 1px solid #cddcf8;
  border-radius: 6px;
  background: #eaf1ff;
  font-size: 13px;
  font-weight: 700;
  color: #1f5fbf;
}

h1,
h2,
h3 {
  margin: 0;
  letter-spacing: 0;
}

h1 {
  margin-top: 6px;
  font-size: 24px;
  line-height: 1.2;
}

h2 {
  font-size: 20px;
  line-height: 1.28;
  margin-top: 5px;
}

h3 {
  font-size: 16px;
}

.status-pill {
  flex: 0 0 auto;
  min-height: 30px;
  padding: 0 10px;
  border-radius: 999px;
  border: 1px solid #d8deeb;
  background: #f8fafc;
  color: #40506b;
  font-size: 13px;
  font-weight: 700;
  line-height: 30px;
}

.status-open {
  border-color: #bcd7ff;
  background: #e9f3ff;
  color: #155da8;
}

.status-submitted {
  border-color: #bfe7ce;
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
.reply-section {
  background: #ffffff;
  border: 1px solid #e2e7f1;
  border-radius: 8px;
  box-shadow: 0 10px 24px rgba(30, 41, 59, 0.06);
}

.message-head,
.reply-section {
  padding: 16px;
}

.message-head {
  border-left: 4px solid #3c6df0;
}

.message-content {
  overflow: hidden;
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
  line-height: 1.35;
}

.message-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
  margin-top: 10px;
}

.message-markdown,
.thread-body {
  margin: 0;
  padding: 16px;
  word-break: break-word;
  font-family: inherit;
  font-size: 15px;
  line-height: 1.65;
}

.mobile-message-files {
  margin: 0 16px 16px;
  border: 1px solid #dbe3ef;
  border-radius: 8px;
  background: #f8fafc;
}

.mobile-thread-files {
  margin-top: 10px;
  border: 1px solid #dbe3ef;
  border-radius: 8px;
  background: #f8fafc;
}

.mobile-message-files :deep(.output-files-head),
.mobile-thread-files :deep(.output-files-head) {
  padding: 10px 10px 0;
}

.mobile-message-files :deep(.output-files-wrap),
.mobile-thread-files :deep(.output-files-wrap) {
  padding: 10px;
}

.mobile-message-files :deep(.output-files-item),
.mobile-thread-files :deep(.output-files-item) {
  min-width: 0;
}

.thread-body {
  padding: 0;
  margin-top: 8px;
  color: #2f3a4e;
  font-size: 14px;
  line-height: 1.55;
}

.message-markdown :deep(p),
.message-markdown :deep(ul),
.message-markdown :deep(ol),
.thread-body :deep(p),
.thread-body :deep(ul),
.thread-body :deep(ol) {
  margin: 0 0 10px;
}

.message-markdown :deep(ul),
.message-markdown :deep(ol),
.thread-body :deep(ul),
.thread-body :deep(ol) {
  padding-left: 20px;
}

.message-markdown :deep(code),
.thread-body :deep(code) {
  padding: 2px 5px;
  border-radius: 5px;
  background: #eef2f7;
  color: #1f2937;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.92em;
}

.message-markdown :deep(pre),
.thread-body :deep(pre) {
  overflow-x: auto;
  margin: 10px 0;
  padding: 12px;
  border: 1px solid #d9e1ee;
  border-radius: 8px;
  background: #f8fafc;
}

.message-markdown :deep(pre code),
.thread-body :deep(pre code) {
  padding: 0;
  background: transparent;
}

.message-markdown :deep(blockquote),
.thread-body :deep(blockquote) {
  margin: 10px 0;
  padding-left: 10px;
  border-left: 3px solid #9bb9ef;
  color: #40506b;
}

.message-markdown :deep(p:last-child),
.message-markdown :deep(ul:last-child),
.message-markdown :deep(ol:last-child),
.thread-body :deep(p:last-child),
.thread-body :deep(ul:last-child),
.thread-body :deep(ol:last-child) {
  margin-bottom: 0;
}

.thread-section {
  display: grid;
  gap: 10px;
}

.thread-list {
  display: grid;
  gap: 10px;
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
  color: #172033;
}

.reply-section {
  display: grid;
  gap: 12px;
}

.reply-section :deep(.el-textarea__inner) {
  min-height: 142px;
  border-radius: 8px;
  border-color: #cad3e1;
  background: #fbfcff;
  color: #172033;
  font-size: 15px;
  line-height: 1.55;
  box-shadow: none;
}

.reply-actions {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
}

.reply-actions :deep(.el-button) {
  min-height: 44px;
  border-radius: 8px;
}

.reply-actions :deep(.el-button--primary) {
  background: #2563eb;
  border-color: #2563eb;
}

.mobile-page-end {
  height: 1px;
}

@media (max-width: 520px) {
  .mobile-action-page {
    padding: 10px 10px calc(18px + env(safe-area-inset-bottom));
  }

  .mobile-action-topbar {
    align-items: stretch;
    padding-top: 6px;
  }

  .status-pill {
    align-self: flex-start;
    width: fit-content;
  }

  h1 {
    font-size: 22px;
  }

  h2 {
    font-size: 18px;
  }

  .message-head,
  .reply-section {
    padding: 14px;
  }

  .reply-actions :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }

  .reply-actions {
    grid-template-columns: 1fr;
  }
}
</style>
