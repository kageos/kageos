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
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'
import OutputFilesDisplay from '@/architecture/presentation/components/OutputFilesDisplay.vue'
import type { OutputFileGroup } from '@/architecture/presentation/composables/useOutputFileGroups'

type TaskFactTone = 'blue' | 'green' | 'amber' | 'gray'

interface TaskFact {
  label: string
  value: string
  tone: TaskFactTone
}

interface SummaryItem {
  label: string
  value: string
}

interface QuickReply {
  label: string
  content: string
}

interface TimelineStep {
  label: string
  done: boolean
  active: boolean
}

const MOBILE_ASK_DRAFT_STORAGE_KEY = 'kageos_mobile_ask_draft'

const route = useRoute()
const loading = ref(false)
const submitting = ref(false)
const error = ref('')
const view = ref<MessageActionViewResp | null>(null)
const replyContent = ref('')
const replyResult = ref<MessageActionReplyResp | null>(null)
const lastSubmittedContent = ref('')

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

const sourceName = computed(() => {
  const message = view.value?.message
  return firstNonEmpty(
    message?.source_display?.name,
    message?.source_title,
    message?.source_path,
    'Kageos 消息'
  )
})

const sourcePath = computed(() => {
  const message = view.value?.message
  return firstNonEmpty(message?.source_path, message?.full_code_path, message?.source_parent_path)
})

const expiresText = computed(() => {
  const expiresAt = view.value?.expires_at
  return expiresAt ? dayjs(expiresAt).format('YYYY-MM-DD HH:mm') : ''
})

const submittedAtText = computed(() => {
  const submittedAt = view.value?.submitted_at
  return submittedAt ? dayjs(submittedAt).format('YYYY-MM-DD HH:mm') : ''
})

const statusText = computed(() => {
  const status = view.value?.token_status
  if (status === 'open') return '等待回复'
  if (status === 'submitted') return '已提交'
  if (status === 'expired') return '已过期'
  if (status === 'revoked') return '已撤销'
  return status || '加载中'
})

const overviewCopy = computed(() => {
  if (view.value?.can_reply) {
    return '这条通知可以在手机上直接回复。提交后，Kageos 会读取原消息和来源目录继续处理，最终结果会再通过通知发给你。'
  }
  if (view.value?.token_status === 'submitted') {
    return '这条消息已经提交给 Kageos。你可以等待后续通知，也可以打开来源目录查看工作台会话。'
  }
  return '当前链接不可继续回复，请查看消息状态或回到 Kageos 工作台处理。'
})

const messageSummary = computed(() => {
  return truncateText(stripMarkup(view.value?.message.content || view.value?.message.title || ''), 120)
})

const summaryItems = computed<SummaryItem[]>(() => {
  const message = view.value?.message
  return [
    { label: '来源', value: sourceName.value },
    { label: '发送人', value: firstNonEmpty(message?.from, 'system') },
    { label: '收到时间', value: formatTime(message?.created_at, 'YYYY-MM-DD HH:mm') },
    { label: '处理有效期', value: expiresText.value },
    { label: '来源路径', value: sourcePath.value },
    { label: '会话', value: firstNonEmpty(view.value?.workspace_session_id, message?.workspace_session_id) },
  ].filter(item => Boolean(item.value))
})

const taskFacts = computed<TaskFact[]>(() => {
  const text = `${view.value?.message.title || ''}\n${stripMarkup(view.value?.message.content || '')}`
  const facts: TaskFact[] = []
  pushFact(facts, '任务ID', matchFirst(text, [/任务\s*ID\s*[:：]?\s*([0-9]+)/i, /task_id\s*[:：=]?\s*([0-9]+)/i]), 'blue')
  pushFact(facts, '物流单', matchFirst(text, [/物流单(?:号)?\s*[:：]?\s*([A-Za-z0-9_-]+)/, /运单(?:号)?\s*[:：]?\s*([A-Za-z0-9_-]+)/]), 'green')
  pushFact(facts, '产品', matchFirst(text, [/产品\s*[:：]?\s*([^，,。\n\s]+)/, /产品编码\s*[:：]?\s*([^，,。\n\s]+)/]), 'gray')
  pushFact(facts, '当前阶段', matchFirst(text, [/当前(?:进入)?【([^】]+)】阶段/, /【([^】]+)】阶段/, /阶段\s*[:：]?\s*([^，,。\n\s]+)/]), 'amber')
  pushFact(facts, '确认要求', matchFirst(text, [/请确认[:：]?\s*([^。\n]+)/, /确认要求[:：]?\s*([^。\n]+)/]), 'blue')
  return facts
})

const quickReplies = computed<QuickReply[]>(() => {
  const stage = taskFacts.value.find(fact => fact.label === '当前阶段')?.value || ''
  const isLogistics = Boolean(taskFacts.value.find(fact => fact.label === '物流单')) || /物流|运单|出库|海运|签收/.test(view.value?.message.content || '')
  if (isLogistics) {
    return [
      {
        label: '确认完成',
        content: stage ? `确认${stage}完成，相关资料已齐全。` : '确认完成，相关资料已齐全。',
      },
      {
        label: '延迟处理',
        content: '暂时无法确认，预计延迟到今天 18:00。原因：',
      },
      {
        label: '资料缺失',
        content: '当前资料不齐，暂不能确认。缺失内容：',
      },
      {
        label: '需要升级',
        content: '当前节点存在异常，需要升级给负责人协助处理。原因：',
      },
    ]
  }
  return [
    { label: '确认完成', content: '确认完成，请继续推进。' },
    { label: '稍后处理', content: '我会稍后处理，预计完成时间：' },
    { label: '需要补充', content: '需要补充信息后再处理，缺少：' },
    { label: '转人工', content: '这条消息需要人工进一步确认，原因：' },
  ]
})

const timelineSteps = computed<TimelineStep[]>(() => {
  const submitted = view.value?.token_status === 'submitted' || Boolean(replyResult.value)
  const agentAccepted = Boolean(replyResult.value?.agent_submitted)
  return [
    { label: '收到通知', done: true, active: false },
    { label: submitted ? '回复已提交' : '等待回复', done: submitted, active: Boolean(view.value?.can_reply) },
    { label: agentAccepted ? '工作台已接收' : 'Kageos 处理', done: false, active: agentAccepted },
    { label: '结果再推送', done: false, active: false },
  ]
})

const receiptTitle = computed(() => {
  if (replyResult.value?.agent_submitted) {
    return replyResult.value.workspace_session_id
      ? `已交给工作台会话 ${replyResult.value.workspace_session_id}`
      : '已交给 Kageos 工作台'
  }
  if (replyResult.value?.agent_submit_error) {
    return '消息已记录，但工作台接收失败'
  }
  if (view.value?.token_status === 'submitted') {
    return submittedAtText.value ? `已于 ${submittedAtText.value} 提交` : '这条消息已提交'
  }
  return ''
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
    lastSubmittedContent.value = content
    replyContent.value = ''
    if (result.agent_submitted) {
      ElMessage.success('已交给 Kageos，处理完成后会收到通知')
    } else if (result.agent_submit_error) {
      ElMessage.warning('消息已记录，但工作台接收失败')
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

function applyQuickReply(content: string) {
  const current = replyContent.value.trim()
  replyContent.value = current ? `${current}\n${content}` : content
}

function openAskPage() {
  if (!view.value?.mobile_ask_url) return
  const sessionID = firstNonEmpty(replyResult.value?.workspace_session_id, view.value.workspace_session_id, view.value.message.workspace_session_id)
  const draft = replyContent.value.trim()
  const payload = {
    full_code_path: sourcePath.value,
    session_id: sessionID,
    message: draft ? `围绕这条消息继续处理：\n${draft}` : '',
  }
  sessionStorage.setItem(MOBILE_ASK_DRAFT_STORAGE_KEY, JSON.stringify(payload))
  const targetURL = new URL(view.value.mobile_ask_url, window.location.origin)
  if (sourcePath.value && !targetURL.searchParams.has('source_path')) {
    targetURL.searchParams.set('source_path', sourcePath.value)
  }
  if (sessionID && !targetURL.searchParams.has('session_id')) {
    targetURL.searchParams.set('session_id', sessionID)
  }
  window.location.assign(targetURL.toString())
}

function formatTime(value?: string | null, pattern = 'MM-DD HH:mm') {
  return value ? dayjs(value).format(pattern) : ''
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
      name: ref.split('/').pop() || '文件',
    })),
  }]
}

function firstNonEmpty(...values: Array<string | undefined | null>) {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) {
      return value.trim()
    }
  }
  return ''
}

function matchFirst(text: string, patterns: RegExp[]) {
  for (const pattern of patterns) {
    const match = pattern.exec(text)
    const value = match?.[1] || match?.[2]
    if (value?.trim()) {
      return truncateText(cleanFactValue(value), 48)
    }
  }
  return ''
}

function pushFact(facts: TaskFact[], label: string, value: string, tone: TaskFactTone) {
  if (!value) return
  if (facts.some(fact => fact.label === label && fact.value === value)) return
  facts.push({ label, value, tone })
}

function cleanFactValue(value: string) {
  return stripMarkup(value)
    .replace(/^[:：,，。;；\s]+|[:：,，。;；\s]+$/g, '')
    .trim()
}

function stripMarkup(value: string) {
  return value
    .replace(/<[^>]+>/g, ' ')
    .replace(/[#>*_`[\]()-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function truncateText(value: string, maxLength: number) {
  const text = value.trim()
  if (text.length <= maxLength) return text
  return `${text.slice(0, maxLength)}...`
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
          <h1>移动消息处理</h1>
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
        <section class="overview-panel">
          <div class="overview-kicker">{{ sourceName }}</div>
          <h2>{{ view.message.title || '未命名消息' }}</h2>
          <p>{{ overviewCopy }}</p>
          <div class="timeline" aria-label="处理进度">
            <span
              v-for="step in timelineSteps"
              :key="step.label"
              class="timeline-step"
              :class="{ 'is-done': step.done, 'is-active': step.active }"
            >
              {{ step.label }}
            </span>
          </div>
        </section>

        <section v-if="taskFacts.length > 0" class="task-summary-panel">
          <div class="panel-heading">
            <h3>处理摘要</h3>
            <span>手机优先</span>
          </div>
          <div class="fact-grid">
            <article
              v-for="fact in taskFacts"
              :key="`${fact.label}-${fact.value}`"
              class="fact-item"
              :class="`fact-${fact.tone}`"
            >
              <span>{{ fact.label }}</span>
              <strong>{{ fact.value }}</strong>
            </article>
          </div>
        </section>

        <section class="summary-panel">
          <div class="summary-list">
            <div v-for="item in summaryItems" :key="`${item.label}-${item.value}`" class="summary-row">
              <span>{{ item.label }}</span>
              <strong>{{ item.value }}</strong>
            </div>
          </div>
        </section>

        <section class="message-content">
          <details class="message-details" :open="taskFacts.length === 0">
            <summary>
              <span>原消息详情</span>
              <small v-if="messageSummary">{{ messageSummary }}</small>
            </summary>
            <div class="message-markdown" v-html="renderedMessageContent" />
            <OutputFilesDisplay
              v-if="messageFileGroups(view.message).length > 0"
              class="mobile-message-files"
              :file-groups="messageFileGroups(view.message)"
              section-title="附件"
            />
          </details>
        </section>

        <section v-if="displayThread.length > 1" class="thread-section">
          <details class="thread-details">
            <summary>
              <span>最近上下文</span>
              <small>{{ displayThread.length }} 条消息</small>
            </summary>
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
          </details>
        </section>

        <section class="reply-section">
          <div class="panel-heading">
            <h3>{{ view.can_reply ? '回复并处理' : '处理结果' }}</h3>
            <span v-if="expiresText">有效期至 {{ expiresText }}</span>
          </div>

          <div v-if="receiptTitle" class="receipt-panel" :class="{ 'is-warning': replyResult?.agent_submit_error }">
            <strong>{{ receiptTitle }}</strong>
            <p v-if="replyResult?.agent_submitted">Kageos 会继续处理这条消息；完成后会通过通知给你简短结论。</p>
            <p v-else-if="replyResult?.agent_submit_error">原因：{{ replyResult.agent_submit_error }}</p>
            <p v-else>后续状态请在 Kageos 工作台或通知里查看。</p>
            <blockquote v-if="lastSubmittedContent">{{ lastSubmittedContent }}</blockquote>
          </div>

          <template v-if="view.can_reply">
            <div class="reply-next">
              <strong>提交后会发生什么</strong>
              <p>Kageos 会带着原消息、你的回复和来源目录启动工作台处理。业务完成后，它必须再发一条简短通知给你。</p>
            </div>

            <div class="quick-replies">
              <button
                v-for="reply in quickReplies"
                :key="reply.label"
                type="button"
                class="quick-reply"
                @click="applyQuickReply(reply.content)"
              >
                {{ reply.label }}
              </button>
            </div>

            <el-input
              v-model="replyContent"
              type="textarea"
              :rows="5"
              maxlength="8000"
              show-word-limit
              resize="none"
              placeholder="写清处理意见，例如：确认完成、延迟原因、缺失资料或需要升级的人。"
              :disabled="submitting"
            />
          </template>

          <div class="reply-actions">
            <el-button
              v-if="view.can_reply"
              type="primary"
              size="large"
              :loading="submitting"
              @click="submitReply"
            >
              回复并交给 Kageos 处理
            </el-button>
            <el-button
              v-if="view.mobile_ask_url"
              size="large"
              @click="openAskPage"
            >
              主动问 Kageos
            </el-button>
          </div>
          <p v-if="!view.can_reply" class="reply-hint">
            当前链接不能继续提交。你仍然可以主动问 Kageos，或回到工作台查看最新状态。
          </p>
        </section>
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
h3,
p {
  margin: 0;
  letter-spacing: 0;
}

h1 {
  margin-top: 6px;
  font-size: 24px;
  line-height: 1.2;
}

h2 {
  font-size: 21px;
  line-height: 1.28;
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

.overview-panel,
.task-summary-panel,
.summary-panel,
.message-content,
.thread-section,
.reply-section {
  background: #ffffff;
  border: 1px solid #e2e7f1;
  border-radius: 8px;
  box-shadow: 0 10px 24px rgba(30, 41, 59, 0.06);
}

.overview-panel {
  display: grid;
  gap: 12px;
  padding: 18px;
  border-left: 4px solid #2563eb;
}

.overview-kicker {
  color: #1f5fbf;
  font-size: 13px;
  font-weight: 800;
  line-height: 1.35;
}

.overview-panel p {
  color: #53627c;
  font-size: 14px;
  line-height: 1.6;
}

.timeline {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}

.timeline-step {
  min-height: 34px;
  padding: 7px 8px;
  border: 1px solid #dce4ef;
  border-radius: 7px;
  background: #f8fafc;
  color: #657089;
  font-size: 12px;
  font-weight: 700;
  text-align: center;
}

.timeline-step.is-done {
  border-color: #bfe7ce;
  background: #e8f7ee;
  color: #17663b;
}

.timeline-step.is-active {
  border-color: #bcd7ff;
  background: #e9f3ff;
  color: #155da8;
}

.task-summary-panel,
.summary-panel,
.reply-section {
  padding: 16px;
}

.panel-heading {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  margin-bottom: 12px;
}

.panel-heading span {
  color: #657089;
  font-size: 12px;
  font-weight: 700;
}

.fact-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.fact-item {
  display: grid;
  gap: 5px;
  min-width: 0;
  padding: 12px;
  border: 1px solid #e1e8f2;
  border-left-width: 4px;
  border-radius: 8px;
  background: #fbfcff;
}

.fact-item span {
  color: #64748b;
  font-size: 12px;
  font-weight: 700;
}

.fact-item strong {
  min-width: 0;
  color: #172033;
  font-size: 15px;
  line-height: 1.35;
  word-break: break-word;
}

.fact-blue {
  border-left-color: #2563eb;
}

.fact-green {
  border-left-color: #0f766e;
}

.fact-amber {
  border-left-color: #d97706;
}

.fact-gray {
  border-left-color: #64748b;
}

.summary-list {
  display: grid;
  gap: 10px;
}

.summary-row {
  display: grid;
  grid-template-columns: 86px minmax(0, 1fr);
  gap: 10px;
  align-items: start;
}

.summary-row span {
  color: #64748b;
  font-size: 13px;
  font-weight: 700;
}

.summary-row strong {
  min-width: 0;
  color: #263349;
  font-size: 13px;
  line-height: 1.45;
  word-break: break-word;
}

.message-content {
  overflow: hidden;
}

.message-details,
.thread-details {
  display: block;
}

.message-details summary,
.thread-details summary {
  display: grid;
  gap: 4px;
  padding: 14px 16px;
  cursor: pointer;
  list-style: none;
}

.message-details summary::-webkit-details-marker,
.thread-details summary::-webkit-details-marker {
  display: none;
}

.message-details summary span,
.thread-details summary span {
  color: #172033;
  font-size: 15px;
  font-weight: 800;
}

.message-details summary small,
.thread-details summary small {
  color: #657089;
  font-size: 12px;
  line-height: 1.45;
}

.message-markdown,
.thread-body {
  margin: 0;
  padding: 16px;
  border-top: 1px solid #eef2f7;
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

.thread-list {
  display: grid;
  gap: 10px;
  padding: 0 12px 12px;
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
  color: #657089;
  font-size: 13px;
}

.thread-title {
  margin-top: 6px;
  font-weight: 700;
  color: #172033;
}

.thread-body {
  padding: 0;
  border-top: 0;
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

.reply-section {
  display: grid;
  gap: 12px;
}

.receipt-panel {
  display: grid;
  gap: 8px;
  padding: 12px;
  border: 1px solid #bfe7ce;
  border-radius: 8px;
  background: #f0fbf4;
}

.receipt-panel.is-warning {
  border-color: #f1d29a;
  background: #fff8e8;
}

.receipt-panel strong {
  color: #17663b;
  font-size: 14px;
}

.receipt-panel.is-warning strong {
  color: #92400e;
}

.receipt-panel p {
  color: #526070;
  font-size: 13px;
  line-height: 1.5;
}

.receipt-panel blockquote {
  margin: 2px 0 0;
  padding: 9px 10px;
  border-left: 3px solid #9ccfb1;
  background: rgba(255, 255, 255, 0.7);
  color: #263349;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
}

.reply-next {
  display: grid;
  gap: 5px;
  padding: 12px;
  border: 1px solid #d9e4f5;
  border-radius: 8px;
  background: #f7fbff;
}

.reply-next strong {
  color: #155da8;
  font-size: 13px;
}

.reply-next p,
.reply-hint {
  color: #657089;
  font-size: 13px;
  line-height: 1.5;
}

.quick-replies {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.quick-reply {
  min-height: 34px;
  padding: 0 11px;
  border: 1px solid #cfd9e8;
  border-radius: 999px;
  background: #ffffff;
  color: #34445d;
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}

.quick-reply:active {
  transform: translateY(1px);
}

.reply-section :deep(.el-textarea__inner) {
  min-height: 150px;
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
    font-size: 19px;
  }

  .overview-panel,
  .task-summary-panel,
  .summary-panel,
  .reply-section {
    padding: 14px;
  }

  .timeline,
  .fact-grid {
    grid-template-columns: 1fr 1fr;
  }

  .summary-row {
    grid-template-columns: 74px minmax(0, 1fr);
  }

  .reply-actions {
    grid-template-columns: 1fr;
  }

  .reply-actions :deep(.el-button) {
    width: 100%;
    margin-left: 0;
  }
}
</style>
