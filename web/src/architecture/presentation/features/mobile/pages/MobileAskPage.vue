<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  workspaceChatStream,
  workspaceStreamEvents,
  type WorkspaceStreamPayload,
} from '@/architecture/presentation/context/api/workspace'
import { useLazyMarkdownRenderer } from '@/architecture/presentation/composables/useLazyMarkdownRenderer'

const route = useRoute()
const fullCodePath = ref(initialSourcePath())
const question = ref('')
const streaming = ref(false)
const answer = ref('')
const sessionId = ref(initialSessionID())
const error = ref('')
const pageEndRef = ref<HTMLElement | null>(null)

const MOBILE_ASK_DRAFT_STORAGE_KEY = 'kageos_mobile_ask_draft'

const { renderMarkdown, preloadMarkdown } = useLazyMarkdownRenderer()
void preloadMarkdown()

const canSend = computed(() => fullCodePath.value.trim() && question.value.trim() && !streaming.value)
const renderedAnswer = computed(() => renderMarkdown(answer.value || (streaming.value ? '正在处理...' : '')))

function initialSourcePath() {
  const raw = route.query.source_path
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === 'string' ? value : ''
}

function initialSessionID() {
  const raw = route.query.session_id
  const value = Array.isArray(raw) ? raw[0] : raw
  return typeof value === 'string' ? value : ''
}

function loadStoredDraft() {
  const raw = sessionStorage.getItem(MOBILE_ASK_DRAFT_STORAGE_KEY)
  if (!raw) return
  sessionStorage.removeItem(MOBILE_ASK_DRAFT_STORAGE_KEY)
  try {
    const draft = JSON.parse(raw) as {
      full_code_path?: string
      session_id?: string
      message?: string
    }
    if (draft.full_code_path?.trim()) {
      fullCodePath.value = draft.full_code_path.trim()
    }
    if (draft.session_id?.trim()) {
      sessionId.value = draft.session_id.trim()
    }
    if (draft.message?.trim()) {
      question.value = draft.message.trim()
    }
  } catch {
    // Ignore stale or incompatible local draft payloads.
  }
}

function buildMobileAskContent(rawQuestion: string) {
  return [
    '【移动端消息处理上下文】',
    '入口：Kageos Pocket 主动问话',
    '输出格式：最终回复必须使用 Markdown 格式，适合手机阅读。',
    '如需异步处理或后续触达用户，请使用 send_notification；message 使用 Markdown，content_type 使用 markdown 或省略；files 可携带平台文件引用。',
    '不要使用 HTML、富文本，也不要把整段回复包进代码块；不要输出工具日志。',
    '',
    '用户问题：',
    rawQuestion.trim(),
  ].join('\n')
}

async function askKageos() {
  if (!canSend.value) {
    ElMessage.warning('请填写目录路径和问题')
    return
  }
  streaming.value = true
  answer.value = ''
  error.value = ''
  try {
    await workspaceChatStream(
      {
        full_code_path: fullCodePath.value.trim(),
        session_id: sessionId.value || undefined,
        message: {
          content: buildMobileAskContent(question.value),
          display_content: question.value.trim(),
        },
      },
      (event, data: WorkspaceStreamPayload) => {
        if (event === workspaceStreamEvents.session && 'session_id' in data) {
          sessionId.value = data.session_id
        }
        if (event === workspaceStreamEvents.content && 'content' in data) {
          answer.value += data.content || ''
        }
        if (event === workspaceStreamEvents.error && 'message' in data) {
          error.value = data.message || '工作台执行失败'
        }
      }
    )
    if (!answer.value.trim() && !error.value) {
      answer.value = '工作台已收到请求，后续结果可在 Kageos 会话中查看。'
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '发送失败'
  } finally {
    streaming.value = false
  }
}

async function scrollToPageBottom(behavior: ScrollBehavior = 'smooth') {
  await nextTick()
  pageEndRef.value?.scrollIntoView({ behavior, block: 'end' })
}

onMounted(() => {
  loadStoredDraft()
  void scrollToPageBottom('auto')
})

watch([renderedAnswer, error, streaming], () => {
  void scrollToPageBottom()
}, { flush: 'post' })
</script>

<template>
  <main class="mobile-ask-page">
    <section class="mobile-ask-shell">
      <div class="mobile-ask-header">
        <div class="mobile-ask-brand">Kageos Pocket</div>
        <h1>主动问 Kageos</h1>
        <p>适合在手机上快速查询状态、补充信息，或让工作台继续处理。</p>
      </div>

      <section class="ask-panel">
        <label>
          <span>目录路径</span>
          <el-input
            v-model="fullCodePath"
            size="large"
            placeholder="/user/app/order_list.table"
            :disabled="streaming"
          />
        </label>

        <label>
          <span>问题</span>
          <el-input
            v-model="question"
            type="textarea"
            :rows="5"
            resize="none"
            placeholder="请用 Markdown 描述问题，例如：&#10;- 帮我看订单 A123 当前状态&#10;- 如果异常，给我一个处理建议"
            :disabled="streaming"
          />
        </label>

        <el-button
          type="primary"
          size="large"
          :loading="streaming"
          :disabled="!canSend"
          @click="askKageos"
        >
          发送到工作台
        </el-button>
      </section>

      <section v-if="answer || error || streaming" class="answer-panel">
        <div class="answer-title">
          <h2>工作台回复</h2>
          <span v-if="sessionId">会话 {{ sessionId }}</span>
        </div>
        <el-alert v-if="error" type="error" :title="error" show-icon :closable="false" />
        <div v-else class="answer-markdown" v-html="renderedAnswer" />
      </section>

      <div ref="pageEndRef" class="mobile-page-end" aria-hidden="true" />
    </section>
  </main>
</template>

<style scoped>
.mobile-ask-page {
  min-height: 100dvh;
  padding: 14px 14px calc(20px + env(safe-area-inset-bottom));
  background: #eef3f7;
  color: #172033;
}

.mobile-ask-shell {
  width: min(100%, 760px);
  margin: 0 auto;
  display: grid;
  gap: 12px;
}

.mobile-ask-header {
  position: sticky;
  top: 0;
  z-index: 2;
  padding: 8px 0 6px;
  background: rgba(238, 243, 247, 0.94);
  backdrop-filter: blur(10px);
}

.mobile-ask-brand {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  min-height: 24px;
  padding: 0 8px;
  border: 1px solid #cddcf8;
  border-radius: 6px;
  background: #eaf1ff;
  color: #1f5fbf;
  font-size: 13px;
  font-weight: 700;
}

h1,
h2,
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
  font-size: 16px;
}

.mobile-ask-header p {
  margin-top: 8px;
  color: #657089;
  font-size: 14px;
  line-height: 1.55;
}

.ask-panel,
.answer-panel {
  background: #ffffff;
  border: 1px solid #e2e7f1;
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 10px 24px rgba(30, 41, 59, 0.06);
}

.ask-panel {
  display: grid;
  gap: 14px;
}

label {
  display: grid;
  gap: 7px;
}

label span {
  color: #40506b;
  font-size: 13px;
  font-weight: 700;
}

.ask-panel :deep(.el-input__wrapper),
.ask-panel :deep(.el-textarea__inner) {
  border-radius: 8px;
  border-color: #cad3e1;
  background: #fbfcff;
  box-shadow: none;
}

.ask-panel :deep(.el-textarea__inner) {
  min-height: 142px;
  color: #172033;
  font-size: 15px;
  line-height: 1.55;
}

.ask-panel :deep(.el-button) {
  min-height: 44px;
  border-radius: 8px;
  background: #2563eb;
  border-color: #2563eb;
}

.answer-title {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  margin-bottom: 12px;
}

.answer-title span {
  color: #657089;
  font-size: 12px;
}

.answer-markdown {
  margin: 0;
  word-break: break-word;
  font-family: inherit;
  font-size: 15px;
  line-height: 1.65;
}

.answer-markdown :deep(p),
.answer-markdown :deep(ul),
.answer-markdown :deep(ol) {
  margin: 0 0 10px;
}

.answer-markdown :deep(ul),
.answer-markdown :deep(ol) {
  padding-left: 20px;
}

.answer-markdown :deep(code) {
  padding: 2px 5px;
  border-radius: 5px;
  background: #eef2f7;
  color: #1f2937;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.92em;
}

.answer-markdown :deep(pre) {
  overflow-x: auto;
  margin: 10px 0;
  padding: 12px;
  border: 1px solid #d9e1ee;
  border-radius: 8px;
  background: #f8fafc;
}

.answer-markdown :deep(pre code) {
  padding: 0;
  background: transparent;
}

.answer-markdown :deep(blockquote) {
  margin: 10px 0;
  padding-left: 10px;
  border-left: 3px solid #9bb9ef;
  color: #40506b;
}

.answer-markdown :deep(p:last-child),
.answer-markdown :deep(ul:last-child),
.answer-markdown :deep(ol:last-child) {
  margin-bottom: 0;
}

.mobile-page-end {
  height: 1px;
}

@media (max-width: 520px) {
  .mobile-ask-page {
    padding: 10px 10px calc(18px + env(safe-area-inset-bottom));
  }

  .mobile-ask-header {
    padding-top: 6px;
  }

  h1 {
    font-size: 22px;
  }

  .ask-panel,
  .answer-panel {
    padding: 14px;
  }
}
</style>
